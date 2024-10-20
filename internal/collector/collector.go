package collector

import (
	"netfetch/internal/model"
	"sync"
	"time"
)

type Collector struct {
	modules  map[string]func() interface{}
	info     *model.SystemInfo
	mutex    sync.RWMutex
	interval time.Duration
}

func New(activeModules []string) *Collector {
	c := &Collector{
		modules: make(map[string]func() interface{}),
		info: &model.SystemInfo{
			Network: &model.NetworkInfo{Interfaces: make([]model.InterfaceInfo, 0)},
			Disk:    &model.DiskInfo{},
		},
		interval: 5 * time.Minute,
	}

	// Register modules
	c.registerModule("os", c.collectOS)
	c.registerModule("cpu", c.collectCPU)
	c.registerModule("memory", c.collectMemory)
	c.registerModule("disk", c.collectDisk)
	c.registerModule("network", c.collectNetwork)
	c.registerModule("de", c.collectDE)
	c.registerModule("wm", c.collectWM)
	c.registerModule("theme", c.collectTheme)
	c.registerModule("icons", c.collectIcons)
	c.registerModule("terminal", c.collectTerminal)
	c.registerModule("gpu", c.collectGPU)
	c.registerModule("resolution", c.collectResolution)

	// Activate specified modules
	for _, module := range activeModules {
		if fn, ok := c.modules[module]; ok {
			fn()
		}
	}

	// Start periodic collection
	go c.periodicCollection()

	return c
}

func (c *Collector) registerModule(name string, fn func() interface{}) {
	c.modules[name] = fn
}

func (c *Collector) periodicCollection() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for range ticker.C {
		c.collectAll()
	}
}

func (c *Collector) collectAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, collect := range c.modules {
		collect()
	}
}

func (c *Collector) GetInfo() *model.SystemInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.info
}
