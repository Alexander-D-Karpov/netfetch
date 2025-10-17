package collector

import (
	"fmt"
	"netfetch/internal/model"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
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

func collectDiskLinux(info *model.SystemInfo) {
	mountpoint := "/"

	var stat syscall.Statfs_t
	if err := syscall.Statfs(mountpoint, &stat); err != nil {
		info.Disk = &model.DiskInfo{}
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	avail := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	usedPercent := 0.0
	if total > 0 {
		usedPercent = (float64(used) / float64(total)) * 100
	}

	info.Disk = &model.DiskInfo{
		Total:       total,
		Used:        used,
		Free:        avail,
		UsedPercent: usedPercent,
	}
}

func collectDiskDarwin(info *model.SystemInfo) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		info.Disk = &model.DiskInfo{}
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	avail := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	usedPercent := 0.0
	if total > 0 {
		usedPercent = (float64(used) / float64(total)) * 100
	}

	info.Disk = &model.DiskInfo{
		Total:       total,
		Used:        used,
		Free:        avail,
		UsedPercent: usedPercent,
	}
}

func collectDiskWindows(info *model.SystemInfo) {
	out, err := exec.Command("wmic", "logicaldisk", "where", "DeviceID='C:'", "get", "Size,FreeSpace", "/format:list").Output()
	if err != nil {
		info.Disk = &model.DiskInfo{}
		return
	}

	var total, free uint64
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Size=") {
			fmt.Sscanf(line, "Size=%d", &total)
		} else if strings.HasPrefix(line, "FreeSpace=") {
			fmt.Sscanf(line, "FreeSpace=%d", &free)
		}
	}

	used := total - free
	usedPercent := 0.0
	if total > 0 {
		usedPercent = (float64(used) / float64(total)) * 100
	}

	info.Disk = &model.DiskInfo{
		Total:       total,
		Used:        used,
		Free:        free,
		UsedPercent: usedPercent,
	}
}

func collectDiskBSD(info *model.SystemInfo) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		info.Disk = &model.DiskInfo{}
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	avail := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	usedPercent := 0.0
	if total > 0 {
		usedPercent = (float64(used) / float64(total)) * 100
	}

	info.Disk = &model.DiskInfo{
		Total:       total,
		Used:        used,
		Free:        avail,
		UsedPercent: usedPercent,
	}
}
