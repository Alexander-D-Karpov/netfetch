package collector

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func detectProcess(processMap map[string][]string) string {
	processes, err := os.ReadDir("/proc")
	if err != nil {
		return "Unknown"
	}

	for _, proc := range processes {
		if !proc.IsDir() {
			continue
		}
		pid := proc.Name()
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		cmdlinePath := filepath.Join("/proc", pid, "cmdline")
		cmdline, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}
		cmd := strings.Split(string(cmdline), "\x00")[0]
		exe := filepath.Base(cmd)
		for name, executables := range processMap {
			for _, executable := range executables {
				if exe == executable {
					return name
				}
			}
		}
	}
	return "Unknown"
}
