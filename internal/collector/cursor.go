package collector

import (
	"os"
	"path/filepath"
	"strings"
)

func (c *Collector) collectCursor() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	configPaths := []string{
		filepath.Join(homeDir, ".icons", "default", "index.theme"),
		filepath.Join(homeDir, ".config", "gtk-3.0", "settings.ini"),
		filepath.Join(homeDir, ".gtkrc-2.0"),
	}
	for _, configPath := range configPaths {
		if data, err := os.ReadFile(configPath); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "Inherits=") || strings.HasPrefix(line, "gtk-cursor-theme-name=") {
					cursorTheme := strings.Trim(strings.Split(line, "=")[1], `"`)
					c.info.Cursor = cursorTheme
					return
				}
			}
		}
	}
	c.info.Cursor = "Unknown"
}
