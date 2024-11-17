package collector

import (
	"os"
)

func (c *Collector) collectLocale() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	locale := os.Getenv("LANG")
	c.info.Locale = locale
}
