package collector

import (
	"bufio"
	"fmt"
	"netfetch/internal/model"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (c *Collector) collectOS() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hostname, _ := os.Hostname()
	user := os.Getenv("USER")
	c.info.OS = &model.OSInfo{
		Name:   runtime.GOOS,
		Distro: detectDistro(),
		Arch:   runtime.GOARCH,
	}
	c.info.Host = hostname
	c.info.User = user
	c.info.Kernel = getKernelVersion()
	c.info.Uptime = getUptime()
	c.info.Packages = getPackages()
	c.info.Shell = os.Getenv("SHELL")

	return c.info.OS
}

func getKernelVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}

func getUptime() string {
	out, err := exec.Command("cat", "/proc/uptime").Output()
	if err != nil {
		return "Unknown"
	}
	fields := strings.Fields(string(out))
	uptimeSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "Unknown"
	}
	uptimeDuration := time.Duration(uptimeSeconds) * time.Second
	return formatUptime(uptimeDuration)
}

func formatUptime(uptime time.Duration) string {
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d days", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hours", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d mins", minutes))
	}

	return strings.Join(parts, ", ")
}

func getPackages() string {
	managers := []struct {
		cmd  string
		args []string
	}{
		{"pacman", []string{"-Qq"}},
		{"dpkg", []string{"--get-selections"}},
		{"rpm", []string{"-qa"}},
		{"xbps-query", []string{"-l"}},
		{"apk", []string{"info"}},
		{"opkg", []string{"list-installed"}},
		{"brew", []string{"list"}},
		{"flatpak", []string{"list"}},
		{"snap", []string{"list"}},
	}

	var count int
	var usedManagers []string

	for _, manager := range managers {
		if path, err := exec.LookPath(manager.cmd); err == nil {
			cmd := exec.Command(path, manager.args...)
			output, err := cmd.Output()
			if err == nil {
				lines := strings.Count(string(output), "\n")
				count += lines
				usedManagers = append(usedManagers, fmt.Sprintf("%d (%s)", lines, manager.cmd))
			}
		}
	}

	if count > 0 {
		return fmt.Sprintf("%d (%s)", count, strings.Join(usedManagers, ", "))
	}

	return "Unknown"
}

var (
	distro     string
	distroOnce sync.Once
)

func detectDistro() string {
	distroOnce.Do(func() {
		distro = parseOSRelease("/etc/os-release")
		if distro == "" {
			distro = parseOSRelease("/usr/lib/os-release")
		}
		if distro == "" {
			distro = runtime.GOOS
		}
	})
	return distro
}

func parseOSRelease(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(file)
	scanner := bufio.NewScanner(file)
	var name, version string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			name = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			break
		} else if strings.HasPrefix(line, "NAME=") {
			name = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		} else if strings.HasPrefix(line, "VERSION=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION="), "\"")
		}
	}
	if name != "" && version != "" {
		return name + " " + version
	}
	return name
}
