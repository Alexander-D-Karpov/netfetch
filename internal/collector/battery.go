package collector

import (
	"netfetch/internal/model"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (c *Collector) collectBattery() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	batteryPath := "/sys/class/power_supply/BAT0"
	capacityData, err := os.ReadFile(filepath.Join(batteryPath, "capacity"))
	if err != nil {
		return
	}
	statusData, err := os.ReadFile(filepath.Join(batteryPath, "status"))
	if err != nil {
		return
	}

	capacityStr := strings.TrimSpace(string(capacityData))
	capacity, err := strconv.ParseFloat(capacityStr, 64)
	if err != nil {
		return
	}

	status := strings.TrimSpace(string(statusData))

	c.info.Battery = &model.BatteryInfo{
		Percentage: capacity,
		Status:     status,
	}
}
