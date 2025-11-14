package collector

import (
	"runtime"
)

func (c *Collector) collectDisk() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		collectDiskLinux(c.info)
	case "darwin":
		collectDiskDarwin(c.info)
	case "windows":
		collectDiskWindows(c.info)
	case "freebsd", "openbsd", "netbsd":
		collectDiskBSD(c.info)
	}

	return c.info.Disk
}
