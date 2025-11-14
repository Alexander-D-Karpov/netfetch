package collector

import (
	"bufio"
	"netfetch/internal/model"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func (c *Collector) collectMemory() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		collectMemoryLinux(c.info)
	case "darwin":
		collectMemoryDarwin(c.info)
	case "windows":
		collectMemoryWindows(c.info)
	case "freebsd", "openbsd", "netbsd":
		collectMemoryBSD(c.info)
	}

	return c.info.Memory
}

func collectMemoryLinux(info *model.SystemInfo) {
	memInfo := parseMemInfo("/proc/meminfo")

	total := memInfo["MemTotal"]
	available := memInfo["MemAvailable"]

	if available == 0 {
		free := memInfo["MemFree"]
		buffers := memInfo["Buffers"]
		cached := memInfo["Cached"]
		sReclaimable := memInfo["SReclaimable"]

		available = free + buffers + cached + sReclaimable
	}

	used := total - available
	if used < 0 {
		used = 0
	}

	info.Memory = &model.MemoryInfo{
		Total: total,
		Used:  used,
		Free:  available,
	}

	swaps := parseSwapDevices()
	if len(swaps) > 0 {
		// Calculate total from devices
		var totalSwap, usedSwap uint64
		for _, swap := range swaps {
			totalSwap += swap.Total
			usedSwap += swap.Used
		}
		info.Swap = &model.SwapInfo{
			Total: totalSwap,
			Used:  usedSwap,
			Free:  totalSwap - usedSwap,
		}
	} else {
		// Fallback to meminfo
		swapTotal := memInfo["SwapTotal"]
		swapFree := memInfo["SwapFree"]
		swapUsed := swapTotal - swapFree

		if swapTotal > 0 {
			info.Swap = &model.SwapInfo{
				Total: swapTotal,
				Used:  swapUsed,
				Free:  swapFree,
			}
		}
	}
}

func parseSwapDevices() []model.SwapInfo {
	data, err := os.ReadFile("/proc/swaps")
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var swaps []model.SwapInfo

	// Skip header line
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		device := fields[0]
		totalKB, _ := strconv.ParseUint(fields[2], 10, 64)
		usedKB, _ := strconv.ParseUint(fields[3], 10, 64)

		swaps = append(swaps, model.SwapInfo{
			Device: device,
			Total:  totalKB * 1024,
			Used:   usedKB * 1024,
		})
	}

	return swaps
}

func parseMemInfo(filePath string) map[string]uint64 {
	file, err := os.Open(filePath)
	if err != nil {
		return make(map[string]uint64)
	}
	defer file.Close()

	memInfo := make(map[string]uint64)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		memInfo[key] = value * 1024
	}

	return memInfo
}

func collectMemoryDarwin(info *model.SystemInfo) {
	total := getSysctlUint64("hw.memsize")

	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		info.Memory = &model.MemoryInfo{
			Total: total,
			Used:  0,
			Free:  0,
		}
		return
	}

	pageSize := uint64(4096)
	var freePages, inactivePages, speculativePages uint64

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		valueStr := strings.TrimSuffix(fields[len(fields)-1], ".")
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			continue
		}

		if strings.HasPrefix(line, "Pages free:") {
			freePages = value
		} else if strings.HasPrefix(line, "Pages inactive:") {
			inactivePages = value
		} else if strings.HasPrefix(line, "Pages speculative:") {
			speculativePages = value
		}
	}

	free := (freePages + inactivePages + speculativePages) * pageSize
	used := total - free

	info.Memory = &model.MemoryInfo{
		Total: total,
		Used:  used,
		Free:  free,
	}

	info.Swap = &model.SwapInfo{
		Total: 0,
		Used:  0,
		Free:  0,
	}
}

func getSysctlUint64(key string) uint64 {
	out, err := exec.Command("sysctl", "-n", key).Output()
	if err != nil {
		return 0
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return val
}

func collectMemoryWindows(info *model.SystemInfo) {
	out, err := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory", "/format:list").Output()
	if err != nil {
		info.Memory = &model.MemoryInfo{
			Total: 0,
			Used:  0,
			Free:  0,
		}
		return
	}

	var total, free uint64
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
			valStr := strings.TrimPrefix(line, "TotalVisibleMemorySize=")
			if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
				total = val * 1024
			}
		} else if strings.HasPrefix(line, "FreePhysicalMemory=") {
			valStr := strings.TrimPrefix(line, "FreePhysicalMemory=")
			if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
				free = val * 1024
			}
		}
	}

	used := total - free

	info.Memory = &model.MemoryInfo{
		Total: total,
		Used:  used,
		Free:  free,
	}

	out, err = exec.Command("wmic", "PAGEFILE", "get", "AllocatedBaseSize,CurrentUsage", "/format:list").Output()
	if err != nil {
		info.Swap = &model.SwapInfo{Total: 0, Used: 0, Free: 0}
		return
	}

	var swapTotal, swapUsed uint64
	lines = strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "AllocatedBaseSize=") {
			valStr := strings.TrimPrefix(line, "AllocatedBaseSize=")
			if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
				swapTotal = val * 1024 * 1024
			}
		} else if strings.HasPrefix(line, "CurrentUsage=") {
			valStr := strings.TrimPrefix(line, "CurrentUsage=")
			if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
				swapUsed = val * 1024 * 1024
			}
		}
	}

	swapFree := swapTotal - swapUsed
	if swapFree < 0 {
		swapFree = 0
	}

	info.Swap = &model.SwapInfo{
		Total: swapTotal,
		Used:  swapUsed,
		Free:  swapFree,
	}
}

func collectMemoryBSD(info *model.SystemInfo) {
	total := getSysctlUint64("hw.physmem")

	freePages := getSysctlUint64("vm.stats.vm.v_free_count")
	inactivePages := getSysctlUint64("vm.stats.vm.v_inactive_count")

	pageSize := getSysctlUint64("hw.pagesize")
	if pageSize == 0 {
		pageSize = 4096
	}

	free := (freePages + inactivePages) * pageSize
	used := total - free
	if used < 0 {
		used = 0
	}

	info.Memory = &model.MemoryInfo{
		Total: total,
		Used:  used,
		Free:  free,
	}

	out, err := exec.Command("swapctl", "-sk").Output()
	if err != nil {
		out, err = exec.Command("swapinfo", "-k").Output()
	}

	if err != nil {
		info.Swap = &model.SwapInfo{Total: 0, Used: 0, Free: 0}
		return
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		info.Swap = &model.SwapInfo{Total: 0, Used: 0, Free: 0}
		return
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		info.Swap = &model.SwapInfo{Total: 0, Used: 0, Free: 0}
		return
	}

	var swapTotal, swapUsed uint64
	if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
		swapTotal = val * 1024
	}
	if val, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
		swapUsed = val * 1024
	}

	swapFree := swapTotal - swapUsed

	info.Swap = &model.SwapInfo{
		Total: swapTotal,
		Used:  swapUsed,
		Free:  swapFree,
	}
}
