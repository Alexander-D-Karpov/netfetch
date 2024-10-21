package collector

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (c *Collector) collectWM() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.WM = getWM()
	c.info.WMTheme = getWMTheme(c.info.WM)
	return c.info.WM
}

func getWM() string {
	out, err := exec.Command("wmctrl", "-m").Output()
	if err != nil {
		return getWindowManager()
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Name:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		}
	}
	return "Unknown"
}

func getWindowManager() string {
	// Try wmctrl -m
	cmd := exec.Command("wmctrl", "-m")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err == nil {
		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Name:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
			}
		}
	}

	// Try xprop
	cmd = exec.Command("xprop", "-root", "_NET_SUPPORTING_WM_CHECK")
	out.Reset()
	cmd.Stdout = &out
	err = cmd.Run()
	if err == nil {
		output := out.String()
		fields := strings.Fields(output)
		if len(fields) >= 5 {
			wmId := fields[4]
			cmd = exec.Command("xprop", "-id", wmId, "-notype", "-len", "100", "-f", "_NET_WM_NAME", "8t")
			out.Reset()
			cmd.Stdout = &out
			err = cmd.Run()
			if err == nil {
				lines := strings.Split(out.String(), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "_NET_WM_NAME") {
						parts := strings.Split(line, "=")
						if len(parts) >= 2 {
							return strings.Trim(strings.Trim(parts[1], "\""), " ")
						}
					}
				}
			}
		}
	}

	return "Unknown"
}

func getWMTheme(wm string) string {
	if wm == "KWin" {
		// Read ~/.config/kwinrc
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "Unknown"
		}
		kwinrc := filepath.Join(homeDir, ".config", "kwinrc")
		data, err := os.ReadFile(kwinrc)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if strings.TrimSpace(line) == "[WindowDecoration]" {
					for j := i + 1; j < len(lines); j++ {
						if strings.HasPrefix(lines[j], "[") {
							break
						}
						if strings.HasPrefix(lines[j], "theme=") {
							return strings.TrimSpace(strings.TrimPrefix(lines[j], "theme="))
						}
					}
				}
			}
		}
	}
	// Other WMs
	return "Unknown"
}
