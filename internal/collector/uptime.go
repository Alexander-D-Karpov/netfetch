package collector

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (c *Collector) collectUptime() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.Uptime = getUptimeLinux()
	case "darwin":
		c.info.Uptime = getUptimeDarwin()
	case "windows":
		c.info.Uptime = getUptimeWindows()
	case "freebsd", "openbsd", "netbsd":
		c.info.Uptime = getUptimeBSD()
	default:
		c.info.Uptime = "Unknown"
	}
}

func getUptimeLinux() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "Unknown"
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "Unknown"
	}

	uptimeSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "Unknown"
	}

	duration := time.Duration(uptimeSeconds) * time.Second
	return formatUptime(duration)
}

func getUptimeDarwin() string {
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return "Unknown"
	}

	bootTimeStr := strings.TrimSpace(string(out))
	bootTimeStr = strings.TrimPrefix(bootTimeStr, "{ sec = ")
	bootTimeStr = strings.Split(bootTimeStr, ",")[0]

	bootTime, err := strconv.ParseInt(bootTimeStr, 10, 64)
	if err != nil {
		return "Unknown"
	}

	uptime := time.Since(time.Unix(bootTime, 0))
	return formatUptime(uptime)
}

func getUptimeWindows() string {
	out, err := exec.Command("wmic", "os", "get", "LastBootUpTime", "/format:list").Output()
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LastBootUpTime=") {
			bootTimeStr := strings.TrimPrefix(line, "LastBootUpTime=")
			bootTimeStr = strings.TrimSpace(bootTimeStr)

			if len(bootTimeStr) >= 14 {
				year := bootTimeStr[0:4]
				month := bootTimeStr[4:6]
				day := bootTimeStr[6:8]
				hour := bootTimeStr[8:10]
				minute := bootTimeStr[10:12]
				second := bootTimeStr[12:14]

				bootTimeFormatted := fmt.Sprintf("%s-%s-%sT%s:%s:%sZ", year, month, day, hour, minute, second)
				bootTime, err := time.Parse(time.RFC3339, bootTimeFormatted)
				if err == nil {
					uptime := time.Since(bootTime)
					return formatUptime(uptime)
				}
			}
		}
	}

	return "Unknown"
}

func getUptimeBSD() string {
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return "Unknown"
	}

	bootTimeStr := strings.TrimSpace(string(out))

	if strings.Contains(bootTimeStr, "sec =") {
		bootTimeStr = strings.TrimPrefix(bootTimeStr, "{ sec = ")
		bootTimeStr = strings.Split(bootTimeStr, ",")[0]
	}

	bootTime, err := strconv.ParseInt(strings.TrimSpace(bootTimeStr), 10, 64)
	if err != nil {
		return "Unknown"
	}

	uptime := time.Since(time.Unix(bootTime, 0))
	return formatUptime(uptime)
}

func formatUptime(uptime time.Duration) string {
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60

	var parts []string

	if days > 0 {
		if days == 1 {
			parts = append(parts, "1 day")
		} else {
			parts = append(parts, fmt.Sprintf("%d days", days))
		}
	}

	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hour")
		} else {
			parts = append(parts, fmt.Sprintf("%d hours", hours))
		}
	}

	if minutes > 0 || len(parts) == 0 {
		if minutes == 1 {
			parts = append(parts, "1 min")
		} else {
			parts = append(parts, fmt.Sprintf("%d mins", minutes))
		}
	}

	return strings.Join(parts, ", ")
}
