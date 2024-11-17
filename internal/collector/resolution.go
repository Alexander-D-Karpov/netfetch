package collector

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func (c *Collector) collectResolution() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Resolution = getResolution()
}

func getResolution() string {
	displayEnv := os.Getenv("DISPLAY")
	if displayEnv == "" {
		err := os.Setenv("DISPLAY", ":0")
		if err != nil {
			return ""
		}
	}
	out, err := exec.Command("xrandr").Output()
	if err == nil {
		re := regexp.MustCompile(` connected.*? (\d+x\d+)`)
		matches := re.FindAllStringSubmatch(string(out), -1)
		resolutions := []string{}
		for _, match := range matches {
			resolutions = append(resolutions, match[1])
		}
		if len(resolutions) > 0 {
			return strings.Join(resolutions, ", ")
		}
	}
	// Alternative method
	resolutions := []string{}
	drmDir := "/sys/class/drm"
	entries, err := os.ReadDir(drmDir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), "connected") {
				modesFile := filepath.Join(drmDir, entry.Name(), "modes")
				modesData, err := os.ReadFile(modesFile)
				if err == nil {
					modes := strings.Fields(string(modesData))
					resolutions = append(resolutions, modes...)
				}
			}
		}
	}
	if len(resolutions) > 0 {
		return strings.Join(resolutions, ", ")
	}
	return "Unknown"
}
