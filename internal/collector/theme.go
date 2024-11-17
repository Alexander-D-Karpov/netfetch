package collector

import (
	"os"
	"path/filepath"
	"strings"
)

func (c *Collector) collectTheme() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Theme = getCurrentTheme()
}

func getCurrentTheme() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "Unknown"
	}
	gtkSettingsPaths := []string{
		filepath.Join(homeDir, ".config", "gtk-3.0", "settings.ini"),
		filepath.Join(homeDir, ".gtkrc-2.0"),
	}
	for _, gtkSettings := range gtkSettingsPaths {
		if data, err := os.ReadFile(gtkSettings); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "gtk-theme-name=") {
					return strings.Trim(strings.TrimPrefix(line, "gtk-theme-name="), `"`)
				}
			}
		}
	}
	return "Unknown"
}
