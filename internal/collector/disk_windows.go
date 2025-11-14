//go:build windows

package collector

import (
	"encoding/csv"
	"netfetch/internal/model"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

func collectDiskWindows(info *model.SystemInfo) {
	out, err := exec.Command("wmic", "logicaldisk", "get", "DeviceID,FileSystem,FreeSpace,Size", "/format:csv").Output()
	if err != nil {
		info.Disks = nil
		info.Disk = &model.DiskInfo{}
		return
	}
	r := csv.NewReader(strings.NewReader(string(out)))
	records, _ := r.ReadAll()
	var vols []model.DiskInfo
	for _, rec := range records {
		if len(rec) < 5 || rec[0] == "Node" {
			continue
		}
		deviceID := strings.TrimSpace(rec[1])
		fs := strings.TrimSpace(rec[2])
		freeStr := strings.TrimSpace(rec[3])
		sizeStr := strings.TrimSpace(rec[4])

		if deviceID == "" || sizeStr == "" {
			continue
		}
		total, _ := strconv.ParseUint(sizeStr, 10, 64)
		free, _ := strconv.ParseUint(freeStr, 10, 64)
		used := uint64(0)
		if total > free {
			used = total - free
		}
		usedPct := 0.0
		if total > 0 {
			usedPct = (float64(used) / float64(total)) * 100.0
		}
		vols = append(vols, model.DiskInfo{
			Total:       total,
			Used:        used,
			Free:        free,
			UsedPercent: usedPct,
			Mountpoint:  deviceID,
			FSType:      fs,
			Device:      deviceID,
		})
	}
	sort.SliceStable(vols, func(i, j int) bool { return vols[i].Mountpoint < vols[j].Mountpoint })
	info.Disks = vols
	if len(vols) > 0 {
		info.Disk = &model.DiskInfo{
			Total:       vols[0].Total,
			Used:        vols[0].Used,
			Free:        vols[0].Free,
			UsedPercent: vols[0].UsedPercent,
			Mountpoint:  vols[0].Mountpoint,
			FSType:      vols[0].FSType,
			Device:      vols[0].Device,
		}
	} else {
		info.Disk = &model.DiskInfo{}
	}

	pout, err := exec.Command("wmic", "diskdrive", "get", "Model,Size,Index", "/format:csv").Output()
	if err == nil {
		r := csv.NewReader(strings.NewReader(string(pout)))
		records, _ := r.ReadAll()
		for _, rec := range records {
			if len(rec) < 4 || rec[0] == "Node" {
				continue
			}
			modelStr := strings.TrimSpace(rec[2])
			sizeStr := strings.TrimSpace(rec[3])
			size, _ := strconv.ParseUint(sizeStr, 10, 64)
			name := "PhysicalDrive" + strings.TrimSpace(rec[1])
			info.PhysicalDisks = append(info.PhysicalDisks, model.PhysicalDisk{
				Name:       name,
				Model:      modelStr,
				Size:       size,
				Type:       "Unknown",
				Rotational: false,
			})
		}
	}
}
