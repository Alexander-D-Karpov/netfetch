package collector

import (
	"bufio"
	"encoding/csv"
	"netfetch/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	type mountRow struct {
		device string
		mp     string
		fs     string
	}

	rows := []mountRow{}
	f, err := os.Open("/proc/self/mounts")
	if err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			dev, mp, fs := unescapeOctal(parts[0]), unescapeOctal(parts[1]), parts[2]
			if shouldSkipMountLinux(mp, fs) {
				continue
			}
			rows = append(rows, mountRow{device: dev, mp: mp, fs: fs})
		}
	}

	// Deduplicate by mountpoint (keep first)
	seen := map[string]bool{}
	vols := make([]model.DiskInfo, 0, len(rows))
	for _, r := range rows {
		if seen[r.mp] {
			continue
		}
		seen[r.mp] = true

		var st syscall.Statfs_t
		if err := syscall.Statfs(r.mp, &st); err != nil {
			continue
		}
		// Use f_bsize; compute "available to unprivileged" via f_bavail
		total := st.Blocks * uint64(st.Bsize)
		avail := st.Bavail * uint64(st.Bsize)
		used := total - avail
		usedPct := 0.0
		if total > 0 {
			usedPct = (float64(used) / float64(total)) * 100.0
		}
		vols = append(vols, model.DiskInfo{
			Total:       total,
			Used:        used,
			Free:        avail,
			UsedPercent: usedPct,
			Mountpoint:  r.mp,
			FSType:      r.fs,
			Device:      r.device,
		})
	}

	// Sort: root "/" first, then by mountpoint path length asc
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
	if info.Disk == nil {
		info.Disk = &model.DiskInfo{}
	}
	if len(vols) > 0 {
		root := vols[0]
		*info.Disk = model.DiskInfo{
			Total:       root.Total,
			Used:        root.Used,
			Free:        root.Free,
			UsedPercent: root.UsedPercent,
			Mountpoint:  root.Mountpoint,
			FSType:      root.FSType,
			Device:      root.Device,
		}
	} else {
		info.Disk = &model.DiskInfo{}
	}

	// Physical disks (best-effort, Linux-only via /sys/block)
	info.PhysicalDisks = listPhysicalLinux()
}

func shouldSkipMountLinux(mp, fs string) bool {
	skipTypes := map[string]bool{
		"proc": true, "sysfs": true, "devpts": true, "devtmpfs": true, "tmpfs": true,
		"cgroup": true, "cgroup2": true, "pstore": true, "securityfs": true, "debugfs": true,
		"configfs": true, "tracefs": true, "nsfs": true, "mqueue": true, "hugetlbfs": true,
		"ramfs": true, "fusectl": true, "binfmt_misc": true, "overlay": true, "squashfs": true,
		"bpf": true, "autofs": true, "efivarfs": true,
	}
	if skipTypes[fs] {
		return true
	}
	skipPrefixes := []string{"/proc", "/sys", "/dev", "/run", "/var/lib/docker",
		"/var/lib/containers", "/snap", "/var/cache/pacman/pkg", "/var/lib/snapd",
		"/var/lib/kubelet", "/var/lib/flatpak", "/var/log", "/.snapshots", "/boot"}
	for _, p := range skipPrefixes {
		if strings.HasPrefix(mp, p) {
			return true
		}
	}
	return false
}

func unescapeOctal(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			o := s[i+1 : i+4]
			if o[0] >= '0' && o[0] <= '7' && o[1] >= '0' && o[1] <= '7' && o[2] >= '0' && o[2] <= '7' {
				v, _ := strconv.ParseInt(o, 8, 0)
				b.WriteByte(byte(v))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func listPhysicalLinux() []model.PhysicalDisk {
	base := "/sys/block"
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var out []model.PhysicalDisk
	skipNames := []string{"loop", "ram", "zram", "dm-", "md", "sr", "fd"}
	isSkip := func(n string) bool {
		for _, p := range skipNames {
			if strings.HasPrefix(n, p) {
				return true
			}
		}
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if isSkip(name) {
			continue
		}
		devDir := filepath.Join(base, name)
		sizeBytes := func() uint64 {
			b, err := os.ReadFile(filepath.Join(devDir, "size"))
			if err != nil {
				return 0
			}
			// size is 512-byte sectors
			sectors, _ := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
			return sectors * 512
		}()
		modelStr := func() string {
			b, _ := os.ReadFile(filepath.Join(devDir, "device/model"))
			s := strings.TrimSpace(string(b))
			if s == "" {
				b, _ = os.ReadFile(filepath.Join(devDir, "device/name"))
				s = strings.TrimSpace(string(b))
			}
			return s
		}()
		rotational := func() bool {
			b, err := os.ReadFile(filepath.Join(devDir, "queue/rotational"))
			if err != nil {
				return false
			}
			return strings.TrimSpace(string(b)) == "1"
		}()
		dtype := "Unknown"
		if rotational {
			dtype = "HDD"
		} else {
			dtype = "SSD"
		}
		out = append(out, model.PhysicalDisk{
			Name:       name,
			Model:      modelStr,
			Size:       sizeBytes,
			Type:       dtype,
			Rotational: rotational,
		})
	}
	return out
}

func collectDiskDarwin(info *model.SystemInfo) {
	out, err := exec.Command("df", "-kP").Output()
	if err != nil {
		info.Disks = nil
		info.Disk = &model.DiskInfo{}
		return
	}
	fsTypes := map[string]string{}
	mountOut, err := exec.Command("mount").Output()
	if err == nil {
		// Example: "/dev/disk3s1 on / (apfs, local, read-only, ...)"
		sc := bufio.NewScanner(strings.NewReader(string(mountOut)))
		for sc.Scan() {
			line := sc.Text()
			onIdx := strings.Index(line, " on ")
			if onIdx < 0 {
				continue
			}
			rest := line[onIdx+4:]
			paren := strings.Index(rest, "(")
			if paren < 0 {
				continue
			}
			mp := strings.TrimSpace(rest[:paren])
			inside := rest[paren+1:]
			rp := strings.Index(inside, ")")
			if rp < 0 {
				continue
			}
			firstField := strings.Split(strings.TrimSpace(inside[:rp]), ",")[0]
			fsTypes[mp] = strings.TrimSpace(firstField)
		}
	}

	var vols []model.DiskInfo
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue // header
		}
		fields := strings.Fields(sc.Text())
		if len(fields) < 6 {
			continue
		}
		device := fields[0]
		totalK, _ := strconv.ParseUint(fields[1], 10, 64)
		usedK, _ := strconv.ParseUint(fields[2], 10, 64)
		availK, _ := strconv.ParseUint(fields[3], 10, 64)
		mp := fields[5]

		// Skip devfs/autofs or odd non-device lines
		if device == "devfs" || strings.HasPrefix(device, "map") {
			continue
		}

		total := totalK * 1024
		used := usedK * 1024
		avail := availK * 1024
		usedPct := 0.0
		if total > 0 {
			usedPct = (float64(used) / float64(total)) * 100.0
		}
		vols = append(vols, model.DiskInfo{
			Total:       total,
			Used:        used,
			Free:        avail,
			UsedPercent: usedPct,
			Mountpoint:  mp,
			FSType:      fsTypes[mp],
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
	// PhysicalDisks left empty on macOS for now (requires diskutil/system_profiler).
}

func collectDiskWindows(info *model.SystemInfo) {
	// Volumes (logical disks)
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
		// Expected columns: Node,DeviceID,FileSystem,FreeSpace,Size
		if len(rec) < 5 || rec[0] == "Node" {
			continue
		}
		deviceID := strings.TrimSpace(rec[1]) // like "C:"
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

	// Physical disks
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

func collectDiskBSD(info *model.SystemInfo) {
	out, err := exec.Command("df", "-kP").Output()
	if err != nil {
		info.Disks = nil
		info.Disk = &model.DiskInfo{}
		return
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
		vols = append(vols, model.DiskInfo{
			Total:       total,
			Used:        used,
			Free:        avail,
			UsedPercent: usedPct,
			Mountpoint:  mp,
			FSType:      "",
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
