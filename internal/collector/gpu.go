package collector

import (
	"os/exec"
	"regexp"
	"strings"
)

func (c *Collector) collectGPU() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.GPU = getGPU()
}

func getGPU() string {
	out, err := exec.Command("lspci").Output()
	if err != nil {
		return "Unknown"
	}
	re := regexp.MustCompile(`(VGA compatible controller|3D controller|Display controller): (.*)`)
	matches := re.FindAllStringSubmatch(string(out), -1)
	gpus := []string{}
	for _, match := range matches {
		gpus = append(gpus, match[2])
	}
	if len(gpus) == 0 {
		return "Unknown"
	}
	return strings.Join(gpus, ", ")
}
