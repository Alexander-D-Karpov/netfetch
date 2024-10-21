package collector

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func (c *Collector) collectTheme() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	theme := getCurrentTheme()
	c.info.Theme = theme
	return c.info.Theme
}

func getCurrentTheme() string {
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
			if strings.HasPrefix(line, "gtk-theme-name=") {
				return strings.TrimSpace(strings.TrimPrefix(line, "gtk-theme-name="))
			}
		}
	}
	// Try other methods or DE-specific configs
	return "Unknown"
}
