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
	"time"
)

func (c *Collector) collectOS() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hostname, _ := os.Hostname()
	user := os.Getenv("USER")
	osInfo := parseOSRelease("/etc/os-release")

	c.info.OS = &model.OSInfo{
		Name:       osInfo["NAME"],
		PrettyName: osInfo["PRETTY_NAME"],
		Distro:     osInfo["ID"],
		IDLike:     osInfo["ID_LIKE"],
		Version:    osInfo["VERSION"],
		VersionID:  osInfo["VERSION_ID"],
		Codename:   osInfo["VERSION_CODENAME"],
		BuildID:    osInfo["BUILD_ID"],
		Variant:    osInfo["VARIANT"],
		VariantID:  osInfo["VARIANT_ID"],
		Arch:       getArchitecture(),
	}
	c.info.Host = hostname
	c.info.User = user
	c.info.Shell = getUserShell()
	c.info.Kernel = getKernelVersion()
}

func parseOSRelease(filePath string) map[string]string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	osInfo := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "="); idx != -1 {
			key := line[:idx]
			value := strings.Trim(line[idx+1:], `"`)
			osInfo[key] = value
		}
	}
	return osInfo
}

func getKernelVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}

func (c *Collector) collectUptime() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Uptime = getUptime()
}

func getUptime() string {
	out, err := os.ReadFile("/proc/uptime")
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

func getArchitecture() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		return "x86_64"
	}
	return arch
}

func getUserShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "Unknown"
	}
	return shell
}
