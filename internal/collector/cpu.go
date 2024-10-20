package collector

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"netfetch/internal/model"
)

func (c *Collector) collectCPU() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cpuInfo, _ := cpu.Info()
	if len(cpuInfo) > 0 {
		c.info.CPU = &model.CPUInfo{
			Model:     cpuInfo[0].ModelName,
			Cores:     int(cpuInfo[0].Cores),
			Frequency: cpuInfo[0].Mhz,
		}
	}

	return c.info.CPU
}
