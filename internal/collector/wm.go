package collector

import (
	"os"
	"path/filepath"
	"strings"
)

func (c *Collector) collectWM() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.WM = getWM()
	c.info.WMTheme = getWMTheme(c.info.WM)
}

func getWM() string {
	wmProcesses := map[string][]string{
		"i3":            {"i3"},
		"Openbox":       {"openbox"},
		"Metacity":      {"metacity"},
		"Compiz":        {"compiz"},
		"KWin":          {"kwin_x11", "kwin_wayland"},
		"Mutter":        {"mutter"},
		"Xfwm4":         {"xfwm4"},
		"Marco":         {"marco"},
		"AwesomeWM":     {"awesome"},
		"Fluxbox":       {"fluxbox"},
		"HerbstluftWM":  {"herbstluftwm"},
		"BSPWM":         {"bspwm"},
		"Sway":          {"sway"},
		"dwm":           {"dwm"},
		"Enlightenment": {"enlightenment"},
	}

	return detectProcess(wmProcesses)
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
