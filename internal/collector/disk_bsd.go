//go:build freebsd || openbsd || netbsd

package collector

import (
	"bufio"
	"netfetch/internal/model"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

func collectDiskBSD(info *model.SystemInfo) {
	out, err := exec.Command("df", "-kP").Output()
	if err != nil {
		info.Disks = nil
		info.Disk = &model.DiskInfo{}
		return
	}

	mountOut, err := exec.Command("mount").Output()
	fsTypes := map[string]string{}
	if err == nil {
		sc := bufio.NewScanner(strings.NewReader(string(mountOut)))
		for sc.Scan() {
			line := sc.Text()
			parts := strings.Fields(line)
			if len(parts) < 4 {
				continue
			}
			mp := parts[2]
			fsType := strings.Trim(parts[3], "()")
			fsTypes[mp] = fsType
		}
	}

	var vols []model.DiskInfo
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		f := strings.Fields(sc.Text())
		if len(f) < 6 {
			continue
		}
		device := f[0]
		totalK, _ := strconv.ParseUint(f[1], 10, 64)
		usedK, _ := strconv.ParseUint(f[2], 10, 64)
		availK, _ := strconv.ParseUint(f[3], 10, 64)
		mp := f[5]

		if device == "devfs" || device == "autofs" {
			continue
		}

		total := totalK * 1024
		used := usedK * 1024
		avail := availK * 1024
		usedPct := 0.0
		if total > 0 {
			usedPct = (float64(used) / float64(total)) * 100.0
		}

		var st syscall.Statfs_t
		fsType := fsTypes[mp]
		if fsType == "" && syscall.Statfs(mp, &st) == nil {
			fsType = string(st.Fstypename[:])
			fsType = strings.TrimRight(fsType, "\x00")
		}

		vols = append(vols, model.DiskInfo{
			Total:       total,
			Used:        used,
			Free:        avail,
			UsedPercent: usedPct,
			Mountpoint:  mp,
			FSType:      fsType,
			Device:      device,
		})
	}
	sort.SliceStable(vols, func(i, j int) bool {
		if vols[i].Mountpoint == "/" {
			return true
		}
		if vols[j].Mountpoint == "/" {
			return false
		}
		return len(vols[i].Mountpoint) < len(vols[j].Mountpoint)
	})
	info.Disks = vols
	if len(vols) > 0 {
		r := vols[0]
		info.Disk = &model.DiskInfo{
			Total:       r.Total,
			Used:        r.Used,
			Free:        r.Free,
			UsedPercent: r.UsedPercent,
			Mountpoint:  r.Mountpoint,
			FSType:      r.FSType,
			Device:      r.Device,
		}
	} else {
		info.Disk = &model.DiskInfo{}
	}
}
