package collector

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (c *Collector) collectProcesses() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.Processes = getProcessCountLinux()
	case "darwin":
		c.info.Processes = getProcessCountDarwin()
	case "windows":
		c.info.Processes = getProcessCountWindows()
	default:
		c.info.Processes = 0
	}
}

func (c *Collector) collectCPUUsage() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.CPUUsage = getCPUUsageLinux()
	case "darwin":
		c.info.CPUUsage = getCPUUsageDarwin()
	case "windows":
		c.info.CPUUsage = getCPUUsageWindows()
	default:
		c.info.CPUUsage = 0
	}
}

func getProcessCountLinux() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
			count++
		}
	}

	return count
}

func getCPUUsageLinux() float64 {
	stat1 := readCPUStat()
	if stat1 == nil {
		return 0
	}

	time.Sleep(100 * time.Millisecond)

	stat2 := readCPUStat()
	if stat2 == nil {
		return 0
	}

	total1 := stat1.user + stat1.nice + stat1.system + stat1.idle + stat1.iowait + stat1.irq + stat1.softirq + stat1.steal
	total2 := stat2.user + stat2.nice + stat2.system + stat2.idle + stat2.iowait + stat2.irq + stat2.softirq + stat2.steal

	idle1 := stat1.idle + stat1.iowait
	idle2 := stat2.idle + stat2.iowait

	totalDelta := total2 - total1
	idleDelta := idle2 - idle1

	if totalDelta == 0 {
		return 0
	}

	usage := 100.0 * (1.0 - float64(idleDelta)/float64(totalDelta))
	return usage
}

type cpuStat struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

func readCPUStat() *cpuStat {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 9 {
				return nil
			}

			stat := &cpuStat{}
			stat.user, _ = strconv.ParseUint(fields[1], 10, 64)
			stat.nice, _ = strconv.ParseUint(fields[2], 10, 64)
			stat.system, _ = strconv.ParseUint(fields[3], 10, 64)
			stat.idle, _ = strconv.ParseUint(fields[4], 10, 64)
			stat.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
			stat.irq, _ = strconv.ParseUint(fields[6], 10, 64)
			stat.softirq, _ = strconv.ParseUint(fields[7], 10, 64)
			stat.steal, _ = strconv.ParseUint(fields[8], 10, 64)

			return stat
		}
	}

	return nil
}

func getProcessCountDarwin() int {
	out, err := exec.Command("ps", "-A").Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(out), "\n")
	return len(lines) - 1
}

func getCPUUsageDarwin() float64 {
	out, err := exec.Command("ps", "-A", "-o", "%cpu").Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(out), "\n")
	totalCPU := 0.0

	for i, line := range lines {
		if i == 0 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		cpu, err := strconv.ParseFloat(line, 64)
		if err == nil {
			totalCPU += cpu
		}
	}

	numCPU := float64(runtime.NumCPU())
	if numCPU == 0 {
		return 0
	}

	return totalCPU / numCPU
}

func getProcessCountWindows() int {
	out, err := exec.Command("tasklist").Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(out), "\n")
	count := 0
	for i, line := range lines {
		if i < 3 {
			continue
		}
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	return count
}

func getCPUUsageWindows() float64 {
	out, err := exec.Command("wmic", "cpu", "get", "loadpercentage", "/format:list").Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LoadPercentage=") {
			percentStr := strings.TrimPrefix(line, "LoadPercentage=")
			if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
				return percent
			}
		}
	}

	return 0
}
