package collector

import (
	"os"
	"path/filepath"
	"strings"
)

func (c *Collector) collectFont() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	configPaths := []string{
		filepath.Join(homeDir, ".config", "fontconfig", "fonts.conf"),
		filepath.Join(homeDir, ".fonts.conf"),
	}
	for _, configPath := range configPaths {
		if data, err := os.ReadFile(configPath); err == nil {
			// Parse XML to get the font (left as an exercise or can use an XML parser)
			// For simplicity, we'll look for a pattern
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.Contains(line, "<family>") {
					fontName := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "<family>"), "</family>"))
					c.info.Font = fontName
					return
				}
			}
		}
	}
	c.info.Font = "Unknown"
}
