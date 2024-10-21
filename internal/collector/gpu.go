package collector

import (
	"os/exec"
	"regexp"
	"strings"
)

func (c *Collector) collectGPU() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.GPU = getGPU()
	return c.info.GPU
}

func getGPU() string {
	out, err := exec.Command("lspci").Output()
	if err != nil {
		return "Unknown"
	}
	re := regexp.MustCompile(`VGA compatible controller: (.*)`)
	matches := re.FindAllStringSubmatch(string(out), -1)
	gpus := []string{}
	for _, match := range matches {
		gpus = append(gpus, match[1])
	}
	return strings.Join(gpus, ", ")
}
