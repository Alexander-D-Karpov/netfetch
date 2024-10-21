package collector

import (
	"bufio"
	"netfetch/internal/model"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func (c *Collector) collectCPU() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cpuModel := getCPUModel()
	cpuCores := runtime.NumCPU()
	cpuFreq := getCPUFrequency()

	c.info.CPU = &model.CPUInfo{
		Model:     cpuModel,
		Cores:     cpuCores,
		Frequency: cpuFreq,
	}

	return c.info.CPU
}

func getCPUModel() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return runtime.GOARCH
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return runtime.GOARCH
}

func getCPUFrequency() float64 {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0.0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu MHz") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				freqStr := strings.TrimSpace(parts[1])
				freq, err := strconv.ParseFloat(freqStr, 64)
				if err == nil {
					return freq
				}
			}
		}
	}
	return 0.0
}
