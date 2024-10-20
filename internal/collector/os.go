package collector

import (
	"bufio"
	"fmt"
	"netfetch/internal/model"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func (c *Collector) collectOS() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hostname, _ := os.Hostname()
	c.info.OS = &model.OSInfo{
		Name:   runtime.GOOS,
		Distro: detectDistro(),
		Arch:   runtime.GOARCH,
	}
	c.info.Host = hostname
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
	uptime, _ := time.ParseDuration("11h31m") // Replace with actual uptime detection
	return formatUptime(uptime)
}

func formatUptime(uptime time.Duration) string {
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60

	var parts []string
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

func detectDistro() string {
	if runtime.GOOS != "linux" {
		return runtime.GOOS
	}

	if _, err := os.Stat("/etc/arch-release"); err == nil {
		return "arch"
	}

	// Check /etc/os-release file
	if file, err := os.Open("/etc/os-release"); err == nil {
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				panic(err)
			}
		}(file)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "ID=") {
				return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
		}
	}

	// Check /etc/lsb-release file
	if file, err := os.Open("/etc/lsb-release"); err == nil {
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				panic(err)
			}
		}(file)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "DISTRIB_ID=") {
				return strings.ToLower(strings.TrimPrefix(line, "DISTRIB_ID="))
			}
		}
	}

	return "unknown"
}
