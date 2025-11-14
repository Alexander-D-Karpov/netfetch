//go:build linux

package collector

import (
	"bufio"
	"netfetch/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

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
