package collector

import (
	"github.com/shirou/gopsutil/v3/mem"
	"netfetch/internal/model"
)

func (c *Collector) collectMemory() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	memInfo, _ := mem.VirtualMemory()
	c.info.Memory = &model.MemoryInfo{
		Total: memInfo.Total,
		Used:  memInfo.Used,
		Free:  memInfo.Free,
	}

	return c.info.Memory
}
