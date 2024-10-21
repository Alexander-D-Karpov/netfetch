package collector

import (
	"os/exec"
	"regexp"
	"strings"
)

func (c *Collector) collectResolution() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Resolution = getResolution()
	return c.info.Resolution
}

func getResolution() string {
	out, err := exec.Command("xrandr").Output()
	if err != nil {
		return "Unknown"
	}
	re := regexp.MustCompile(` connected.*? (\d+x\d+)`)
	matches := re.FindAllStringSubmatch(string(out), -1)
	resolutions := []string{}
	for _, match := range matches {
		resolutions = append(resolutions, match[1])
	}
	return strings.Join(resolutions, ", ")
}
