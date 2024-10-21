package collector

import (
	"bufio"
	"netfetch/internal/model"
	"os"
	"strconv"
	"strings"
)

func (c *Collector) collectMemory() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	memInfo := parseMemInfo("/proc/meminfo")
	c.info.Memory = &model.MemoryInfo{
		Total: memInfo["MemTotal"],
		Used:  memInfo["MemTotal"] - memInfo["MemAvailable"],
		Free:  memInfo["MemAvailable"],
	}

	return c.info.Memory
}

func parseMemInfo(filePath string) map[string]uint64 {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
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
		memInfo[key] = value * 1024 // Convert from kB to bytes
	}
	return memInfo
}
