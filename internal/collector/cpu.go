package collector

import (
	"bufio"
	"netfetch/internal/model"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func (c *Collector) collectCPU() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.info.CPU = &model.CPUInfo{}

	detectCPU(c.info.CPU)
}

func detectCPU(cpu *model.CPUInfo) {
	// Initialize fields
	cpu.Name = ""
	cpu.Vendor = ""
	cpu.CoresPhysical = 0
	cpu.CoresLogical = 0
	cpu.CoresOnline = 0
	cpu.FrequencyBase = 0
	cpu.FrequencyMax = 0
	cpu.Temperature = 0.0

	// Detect CPU information based on OS
	switch runtime.GOOS {
	case "linux":
		detectCPULinux(cpu)
	case "darwin":
		detectCPUDarwin(cpu)
	case "windows":
		detectCPUWindows(cpu)
	default:
		// Unsupported OS
	}
}

func detectCPULinux(cpu *model.CPUInfo) {
	// Open /proc/cpuinfo
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)

	scanner := bufio.NewScanner(file)

	var (
		cpuModel  string
		cpuVendor string
	)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			if cpuModel == "" {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					cpuModel = strings.TrimSpace(parts[1])
				}
			}
		} else if strings.HasPrefix(line, "vendor_id") {
			if cpuVendor == "" {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					cpuVendor = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	if cpuModel == "" {
		// For ARM architectures, try "Hardware" field
		_, err := file.Seek(0, 0)
		if err != nil {
			return
		} // Reset file pointer
		scanner = bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Hardware") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					cpuModel = strings.TrimSpace(parts[1])
				}
				break
			}
		}
	}

	cpu.Name = cpuModel
	cpu.Vendor = cpuVendor

	cpu.CoresLogical = uint16(runtime.NumCPU())
	cpu.CoresOnline = cpu.CoresLogical

	// Get physical cores
	cpu.CoresPhysical = getPhysicalCoresLinux()

	// Get frequencies
	cpu.FrequencyBase, cpu.FrequencyMax = getCPUFrequenciesLinux()

	// Clean up CPU name
	cleanCPUName(cpu)
}

func getPhysicalCoresLinux() uint16 {
	coreIDSet := make(map[string]bool)
	coreIDFiles, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/topology/core_id")
	if err != nil || len(coreIDFiles) == 0 {
		// Fallback to using logical cores count
		return uint16(runtime.NumCPU())
	}
	for _, coreIDFile := range coreIDFiles {
		data, err := os.ReadFile(coreIDFile)
		if err == nil {
			coreID := strings.TrimSpace(string(data))
			coreIDSet[coreID] = true
		}
	}
	return uint16(len(coreIDSet))
}

func getCPUFrequenciesLinux() (uint32, uint32) {
	var freqBase float64
	var freqMax float64

	// Attempt to read max frequency from /sys/devices/system/cpu/cpu*/cpufreq/cpuinfo_max_freq
	cpuFreqMaxFiles, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/cpufreq/cpuinfo_max_freq")
	if err == nil && len(cpuFreqMaxFiles) > 0 {
		for _, freqFile := range cpuFreqMaxFiles {
			data, err := os.ReadFile(freqFile)
			if err == nil {
				freqStr := strings.TrimSpace(string(data))
				freq, err := strconv.ParseFloat(freqStr, 64)
				if err == nil {
					// Frequency is in kHz, convert to MHz
					freqMHz := freq / 1000.0
					if freqMHz > freqMax {
						freqMax = freqMHz
					}
				}
			}
		}
	}

	// Read base frequency from /proc/cpuinfo
	file, err := os.Open("/proc/cpuinfo")
	if err == nil {
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				return
			}
		}(file)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "cpu MHz") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					freqStr := strings.TrimSpace(parts[1])
					freq, err := strconv.ParseFloat(freqStr, 64)
					if err == nil && freq > freqBase {
						freqBase = freq
					}
				}
			}
		}
	}

	// If max frequency couldn't be read, assume it equals base frequency
	if freqMax == 0 {
		freqMax = freqBase
	}

	return uint32(freqBase), uint32(freqMax)
}

func cleanCPUName(cpu *model.CPUInfo) {
	removeStrings := []string{
		" CPU", " FPU", " APU", " Processor",
		" Dual-Core", " Quad-Core", " Six-Core", " Eight-Core", " Ten-Core",
		" 2-Core", " 4-Core", " 6-Core", " 8-Core", " 10-Core", " 12-Core", " 14-Core", " 16-Core",
		" with Radeon Graphics",
	}
	name := cpu.Name
	for _, s := range removeStrings {
		name = strings.ReplaceAll(name, s, "")
	}
	// Remove content after '@' (e.g., frequency info)
	if idx := strings.Index(name, "@"); idx != -1 {
		name = strings.TrimSpace(name[:idx])
	}
	cpu.Name = strings.TrimSpace(name)
}

func detectCPUDarwin(cpu *model.CPUInfo) {
	// TODO: Implement CPU detection for macOS
}

func detectCPUWindows(cpu *model.CPUInfo) {
	// TODO: Implement CPU detection for Windows
}
