package collector

import (
	"github.com/shirou/gopsutil/v3/disk"
	"netfetch/internal/model"
)

func (c *Collector) collectDisk() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	usage, _ := disk.Usage("/")
	c.info.Disk = &model.DiskInfo{
		Total:       usage.Total,
		Used:        usage.Used,
		Free:        usage.Free,
		UsedPercent: usage.UsedPercent,
	}

	return c.info.Disk
}
