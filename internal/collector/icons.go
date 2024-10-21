package collector

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func (c *Collector) collectIcons() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	icons := getCurrentIcons()
	c.info.Icons = icons
	return c.info.Icons
}

func getCurrentIcons() string {
	// Try to read from ~/.config/gtk-3.0/settings.ini
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "Unknown"
	}
	gtk3Settings := filepath.Join(homeDir, ".config", "gtk-3.0", "settings.ini")
	data, err := ioutil.ReadFile(gtk3Settings)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "gtk-icon-theme-name=") {
				return strings.TrimSpace(strings.TrimPrefix(line, "gtk-icon-theme-name="))
			}
		}
	}
	// Try other methods or DE-specific configs
	return "Unknown"
}
