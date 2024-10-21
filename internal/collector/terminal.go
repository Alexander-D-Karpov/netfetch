package collector

import (
	"os"
	"strconv"
	"strings"
)

func (c *Collector) collectTerminal() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Terminal = getTerminal()
	return c.info.Terminal
}

func getTerminal() string {
	pid := os.Getppid()
	for pid > 1 {
		exePath, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		if err == nil {
			exeName := strings.TrimSpace(exePath)
			if strings.Contains(exeName, "alacritty") || strings.Contains(exeName, "gnome-terminal") || strings.Contains(exeName, "konsole") {
				return exeName
			}
		}
		// Get parent process ID
		stat, err := os.Stat("/proc/" + strconv.Itoa(pid) + "/stat")
		if err != nil {
			break
		}
		content, err := os.ReadFile(stat.Name())
		if err != nil {
			break
		}
		fields := strings.Fields(string(content))
		if len(fields) > 3 {
			pid, _ = strconv.Atoi(fields[3]) // PPID is the fourth field
		} else {
			break
		}
	}
	return "Unknown"
}
