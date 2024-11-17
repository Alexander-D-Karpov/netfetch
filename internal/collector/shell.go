package collector

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

func (c *Collector) collectShell() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellUser, err := user.Current()
		if err == nil {
			shellPath = shellUser.Uid
		}
	}

	shellName := filepath.Base(shellPath)

	// Get shell version
	version := getShellVersion(shellPath)

	c.info.Shell = fmt.Sprintf("%s %s", shellName, version)
}

func getShellVersion(shellPath string) string {
	out, err := exec.Command(shellPath, "--version").Output()
	if err != nil {
		return "unknown"
	}
	versionOutput := string(out)
	versionLines := strings.Split(versionOutput, "\n")
	if len(versionLines) > 0 {
		versionLine := versionLines[0]
		// Extract version number
		fields := strings.Fields(versionLine)
		for _, field := range fields {
			if strings.ContainsAny(field, "0123456789") {
				return field
			}
		}
		return versionLine
	}
	return "unknown"
}
