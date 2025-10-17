package collector

import (
	"bufio"
	"fmt"
	"netfetch/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func (c *Collector) collectCPU() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.info.CPU = &model.CPUInfo{}

	switch runtime.GOOS {
	case "linux":
		detectCPULinux(c.info.CPU)
	case "darwin":
		detectCPUDarwin(c.info.CPU)
	case "windows":
		detectCPUWindows(c.info.CPU)
	case "freebsd", "openbsd", "netbsd":
		detectCPUBSD(c.info.CPU)
	}
}

func detectCPULinux(cpu *model.CPUInfo) {
	cpuinfo := parseCPUInfo("/proc/cpuinfo")

	cpu.Name = detectCPUName(cpuinfo)
	cpu.Vendor = getOrDefault(cpuinfo, "vendor_id", "")
	cpu.CoresLogical = uint16(runtime.NumCPU())
	cpu.CoresPhysical = getPhysicalCoresLinux()
	cpu.CoresOnline = cpu.CoresLogical

	baseFreq, maxFreq := getCPUFrequenciesLinux(cpuinfo)
	cpu.FrequencyBase = baseFreq
	cpu.FrequencyMax = maxFreq

	cpu.Temperature = getCPUTemperatureLinux()

	cleanCPUName(cpu)
}

func detectCPUName(cpuinfo map[string]string) string {
	if name := cpuinfo["model name"]; name != "" {
		return name
	}

	if name := cpuinfo["Hardware"]; name != "" {
		return name
	}

	arch := runtime.GOARCH
	if arch == "arm64" || arch == "arm" {
		return detectARMCPUName(cpuinfo)
	}

	return "Unknown"
}

func detectARMCPUName(cpuinfo map[string]string) string {
	implementer := cpuinfo["CPU implementer"]
	part := cpuinfo["CPU part"]

	if implementer == "" || part == "" {
		if out, err := exec.Command("lscpu").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "Model name:") {
					return strings.TrimSpace(strings.TrimPrefix(line, "Model name:"))
				}
			}
		}
		return "ARM Processor"
	}

	cpuName := lookupARMCPU(implementer, part)
	if cpuName != "" {
		return cpuName
	}

	return fmt.Sprintf("ARM %s:%s", implementer, part)
}

func lookupARMCPU(implementer, part string) string {
	armCPUs := map[string]map[string]string{
		"0x41": {
			"0xd07": "Cortex-A57",
			"0xd08": "Cortex-A72",
			"0xd09": "Cortex-A73",
			"0xd0a": "Cortex-A75",
			"0xd0b": "Cortex-A76",
			"0xd0c": "Neoverse N1",
			"0xd0d": "Cortex-A77",
			"0xd0e": "Cortex-A76AE",
			"0xd40": "Neoverse V1",
			"0xd41": "Cortex-A78",
			"0xd44": "Cortex-X1",
			"0xd46": "Cortex-A510",
			"0xd47": "Cortex-A710",
			"0xd48": "Cortex-X2",
			"0xd49": "Neoverse N2",
			"0xd4a": "Neoverse E1",
			"0xd4b": "Cortex-A78AE",
			"0xd4c": "Cortex-X1C",
			"0xd4d": "Cortex-A715",
			"0xd4e": "Cortex-X3",
			"0xd4f": "Neoverse V2",
		},
		"0x51": {
			"0x800": "Kryo",
			"0x801": "Kryo Silver",
			"0x802": "Kryo Gold",
			"0x803": "Kryo Silver",
			"0x804": "Kryo Gold",
		},
	}

	if parts, ok := armCPUs[implementer]; ok {
		if name, ok := parts[part]; ok {
			return name
		}
	}

	return ""
}

func parseCPUInfo(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return make(map[string]string)
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, ":"); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			if _, exists := result[key]; !exists {
				result[key] = value
			}
		}
	}

	return result
}

func getPhysicalCoresLinux() uint16 {
	coreIDSet := make(map[string]bool)

	coreIDFiles, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/topology/core_id")
	if err != nil || len(coreIDFiles) == 0 {
		return uint16(runtime.NumCPU())
	}

	for _, coreIDFile := range coreIDFiles {
		data, err := os.ReadFile(coreIDFile)
		if err == nil {
			coreID := strings.TrimSpace(string(data))
			coreIDSet[coreID] = true
		}
	}

	if len(coreIDSet) == 0 {
		return uint16(runtime.NumCPU())
	}

	return uint16(len(coreIDSet))
}

func getCPUFrequenciesLinux(cpuinfo map[string]string) (uint32, uint32) {
	maxFreq := readCPUFreqSysfs()

	baseFreq := maxFreq
	if maxFreq == 0 {
		if freq := extractFrequencyFromName(cpuinfo["model name"]); freq > 0 {
			maxFreq = freq
			baseFreq = freq
		}
	}

	return baseFreq, maxFreq
}

func readCPUFreqSysfs() uint32 {
	freqPaths := []string{
		"/sys/devices/system/cpu/cpu0/cpufreq/bios_limit",
		"/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq",
		"/sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq",
	}

	for _, path := range freqPaths {
		data, err := os.ReadFile(path)
		if err == nil {
			freqStr := strings.TrimSpace(string(data))
			if freq, err := strconv.ParseUint(freqStr, 10, 64); err == nil {
				return uint32(freq / 1000)
			}
		}
	}

	return 0
}

func extractFrequencyFromName(name string) uint32 {
	if idx := strings.Index(name, "@"); idx != -1 {
		freqStr := strings.TrimSpace(name[idx+1:])
		freqStr = strings.TrimSuffix(freqStr, "GHz")
		freqStr = strings.TrimSpace(freqStr)

		if freq, err := strconv.ParseFloat(freqStr, 64); err == nil {
			return uint32(freq * 1000)
		}
	}
	return 0
}

func getCPUTemperatureLinux() float64 {
	hwmonFiles, _ := filepath.Glob("/sys/class/hwmon/hwmon*/temp*_input")

	for _, file := range hwmonFiles {
		nameFile := filepath.Join(filepath.Dir(file), "name")
		if nameData, err := os.ReadFile(nameFile); err == nil {
			name := strings.TrimSpace(string(nameData))
			if strings.Contains(name, "coretemp") || strings.Contains(name, "k10temp") || strings.Contains(name, "cpu") {
				if tempData, err := os.ReadFile(file); err == nil {
					if temp, err := strconv.ParseInt(strings.TrimSpace(string(tempData)), 10, 64); err == nil {
						return float64(temp) / 1000.0
					}
				}
			}
		}
	}

	thermalFiles, _ := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	for _, file := range thermalFiles {
		if tempData, err := os.ReadFile(file); err == nil {
			if temp, err := strconv.ParseInt(strings.TrimSpace(string(tempData)), 10, 64); err == nil {
				return float64(temp) / 1000.0
			}
		}
	}

	return 0.0
}

func detectCPUDarwin(cpu *model.CPUInfo) {
	cpu.Name = getSysctlString("machdep.cpu.brand_string")
	cpu.Vendor = getSysctlString("machdep.cpu.vendor")

	if ncpu := getSysctlInt("hw.ncpu"); ncpu > 0 {
		cpu.CoresLogical = uint16(ncpu)
		cpu.CoresOnline = uint16(ncpu)
	}

	if physCPU := getSysctlInt("hw.physicalcpu"); physCPU > 0 {
		cpu.CoresPhysical = uint16(physCPU)
	} else {
		cpu.CoresPhysical = cpu.CoresLogical
	}

	if freq := getSysctlInt("hw.cpufrequency"); freq > 0 {
		cpu.FrequencyBase = uint32(freq / 1000000)
	}
	if maxFreq := getSysctlInt("hw.cpufrequency_max"); maxFreq > 0 {
		cpu.FrequencyMax = uint32(maxFreq / 1000000)
	} else {
		cpu.FrequencyMax = cpu.FrequencyBase
	}

	cleanCPUName(cpu)
}

func getSysctlString(key string) string {
	out, err := exec.Command("sysctl", "-n", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getSysctlInt(key string) int {
	out, err := exec.Command("sysctl", "-n", key).Output()
	if err != nil {
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return val
}

func detectCPUWindows(cpu *model.CPUInfo) {
	out, err := exec.Command("wmic", "cpu", "get", "Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed", "/format:list").Output()
	if err != nil {
		cpu.Name = "Unknown"
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			cpu.Name = strings.TrimPrefix(line, "Name=")
		} else if strings.HasPrefix(line, "NumberOfCores=") {
			if cores, err := strconv.Atoi(strings.TrimPrefix(line, "NumberOfCores=")); err == nil {
				cpu.CoresPhysical = uint16(cores)
			}
		} else if strings.HasPrefix(line, "NumberOfLogicalProcessors=") {
			if logical, err := strconv.Atoi(strings.TrimPrefix(line, "NumberOfLogicalProcessors=")); err == nil {
				cpu.CoresLogical = uint16(logical)
				cpu.CoresOnline = uint16(logical)
			}
		} else if strings.HasPrefix(line, "MaxClockSpeed=") {
			if freq, err := strconv.Atoi(strings.TrimPrefix(line, "MaxClockSpeed=")); err == nil {
				cpu.FrequencyMax = uint32(freq)
				cpu.FrequencyBase = uint32(freq)
			}
		}
	}

	cleanCPUName(cpu)
}

func detectCPUBSD(cpu *model.CPUInfo) {
	cpu.Name = getSysctlString("hw.model")
	cpu.CoresLogical = uint16(runtime.NumCPU())
	cpu.CoresPhysical = cpu.CoresLogical
	cpu.CoresOnline = cpu.CoresLogical

	if freq := getSysctlInt("hw.clockrate"); freq > 0 {
		cpu.FrequencyBase = uint32(freq)
		cpu.FrequencyMax = uint32(freq)
	}

	cleanCPUName(cpu)
}

func cleanCPUName(cpu *model.CPUInfo) {
	removeStrings := []string{
		" CPU", " FPU", " APU", " Processor", " processor",
		" Dual-Core", " Quad-Core", " Six-Core", " Eight-Core", " Ten-Core",
		" 2-Core", " 4-Core", " 6-Core", " 8-Core", " 10-Core", " 12-Core",
		" 14-Core", " 16-Core", " 24-Core", " 32-Core",
		" with Radeon Graphics", " with Radeon Vega Graphics",
	}

	name := cpu.Name
	for _, s := range removeStrings {
		name = strings.ReplaceAll(name, s, "")
	}

	if idx := strings.Index(name, "@"); idx != -1 {
		name = strings.TrimSpace(name[:idx])
	}

	cpu.Name = strings.TrimSpace(name)
}
