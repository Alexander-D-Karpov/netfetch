package collector

import (
	"os"
)

func (c *Collector) collectDE() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.DE = getDE()
	return c.info.DE
}

func getDE() string {
	de := os.Getenv("XDG_CURRENT_DESKTOP")
	if de == "" {
		de = os.Getenv("DESKTOP_SESSION")
	}
	if de == "" {
		de = os.Getenv("GDMSESSION")
	}
	if de == "" {
		de = "Unknown"
	}
	return de
}
