//go:build windows

package collector

import "netfetch/internal/model"

func collectDiskLinux(info *model.SystemInfo) {
	info.Disks = nil
	info.Disk = &model.DiskInfo{}
}

func collectDiskDarwin(info *model.SystemInfo) {
	info.Disks = nil
	info.Disk = &model.DiskInfo{}
}

func collectDiskBSD(info *model.SystemInfo) {
	info.Disks = nil
	info.Disk = &model.DiskInfo{}
}
