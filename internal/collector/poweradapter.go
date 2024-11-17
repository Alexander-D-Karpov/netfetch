package collector

import (
	"netfetch/internal/model"
	"os"
	"path/filepath"
	"strings"
)

func (c *Collector) collectPowerAdapter() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	adapterPath := "/sys/class/power_supply/AC"
	onlineData, err := os.ReadFile(filepath.Join(adapterPath, "online"))
	if err != nil {
		return
	}

	onlineStr := strings.TrimSpace(string(onlineData))
	isConnected := onlineStr == "1"

	c.info.PowerAdapter = &model.PowerAdapterInfo{
		IsConnected: isConnected,
	}
}
