package collector

import (
	"netfetch/internal/model"
	"sync"
)

type Collector struct {
	activeModules map[string]bool
	info          *model.SystemInfo
	mutex         sync.RWMutex
}

func New(activeModules []string) *Collector {
	c := &Collector{
		activeModules: make(map[string]bool),
		info: &model.SystemInfo{
			Network: &model.NetworkInfo{Interfaces: make([]model.InterfaceInfo, 0)},
			Disk:    &model.DiskInfo{},
		},
	}

	// Initialize active modules map
	for _, moduleName := range activeModules {
		c.activeModules[moduleName] = true
	}

	// Collect static info at startup
	c.collectStaticInfo()

	return c
}

func (c *Collector) collectStaticInfo() {
	if c.activeModules["os"] {
		c.collectOS()
	}
	if c.activeModules["cpu"] {
		c.collectCPU()
	}
	if c.activeModules["gpu"] {
		c.collectGPU()
	}
	if c.activeModules["de"] {
		c.collectDE()
	}
	if c.activeModules["wm"] {
		c.collectWM()
	}
	if c.activeModules["theme"] {
		c.collectTheme()
	}
	if c.activeModules["icons"] {
		c.collectIcons()
	}
	if c.activeModules["terminal"] {
		c.collectTerminal()
	}
	if c.activeModules["font"] {
		c.collectFont()
	}
	if c.activeModules["cursor"] {
		c.collectCursor()
	}
}

func (c *Collector) CollectDynamicInfo() {
	if c.activeModules["uptime"] {
		c.collectUptime()
	}
	if c.activeModules["memory"] {
		c.collectMemory()
	}
	if c.activeModules["disk"] {
		c.collectDisk()
	}
	if c.activeModules["network"] {
		c.collectNetwork()
	}
	if c.activeModules["resolution"] {
		c.collectResolution()
	}
	if c.activeModules["packages"] {
		c.collectPackages()
	}
	if c.activeModules["swap"] {
		c.collectMemory()
	}
	if c.activeModules["localip"] {
		c.collectLocalIP()
	}
	if c.activeModules["battery"] {
		c.collectBattery()
	}
	if c.activeModules["poweradapter"] {
		c.collectPowerAdapter()
	}
	if c.activeModules["locale"] {
		c.collectLocale()
	}
}

func (c *Collector) GetInfo() *model.SystemInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.info
}
