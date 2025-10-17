package collector

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

func getUserShellWithVersion() string {
	shellPath := os.Getenv("SHELL")

	if shellPath == "" {
		shellPath = os.Getenv("COMSPEC")
	}

	if shellPath == "" {
		currentUser, err := user.Current()
		if err == nil && runtime.GOOS != "windows" {
			shellPath = getShellFromPasswd(currentUser.Username)
		}
	}

	if shellPath == "" {
		return "Unknown"
	}

	shellName := filepath.Base(shellPath)
	version := getShellVersion(shellPath, shellName)

	if version != "" {
		return shellName + " " + version
	}

	return shellName
}

func getShellFromPasswd(username string) string {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, username+":") {
			fields := strings.Split(line, ":")
			if len(fields) >= 7 {
				return fields[6]
			}
		}
	}

	return ""
}

func getShellVersion(shellPath, shellName string) string {
	versionArgs := map[string][]string{
		"bash":       {"--version"},
		"zsh":        {"--version"},
		"fish":       {"--version"},
		"ksh":        {"--version"},
		"tcsh":       {"--version"},
		"csh":        {"--version"},
		"dash":       {"--version"},
		"sh":         {"--version"},
		"pwsh":       {"--version"},
		"powershell": {"-Version"},
		"cmd":        {"/c", "ver"},
	}

	args, ok := versionArgs[shellName]
	if !ok {
		args = []string{"--version"}
	}

	out, err := exec.Command(shellPath, args...).Output()
	if err != nil {
		return ""
	}

	versionOutput := string(out)
	lines := strings.Split(versionOutput, "\n")

	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		version := extractVersionFromOutput(firstLine, shellName)
		return version
	}

	return ""
}

func extractVersionFromOutput(output, shellName string) string {
	output = strings.ToLower(output)

	fields := strings.Fields(output)
	for i, field := range fields {
		if strings.Contains(field, "version") && i+1 < len(fields) {
			return cleanVersion(fields[i+1])
		}

		if isVersionString(field) {
			return cleanVersion(field)
		}
	}

	return ""
}

func isVersionString(s string) bool {
	if len(s) < 3 {
		return false
	}

	if s[0] >= '0' && s[0] <= '9' {
		return strings.Contains(s, ".")
	}

	if strings.HasPrefix(s, "v") && len(s) > 1 && s[1] >= '0' && s[1] <= '9' {
		return true
	}

	return false
}

func cleanVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")

	version = strings.Trim(version, "(),[];")

	return version
}

func (c *Collector) collectShell() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = os.Getenv("COMSPEC")
	}

	if shellPath == "" {
		c.info.Shell = "Unknown"
		return
	}

	shellName := filepath.Base(shellPath)
	version := getShellVersion(shellPath, shellName)

	if version != "" {
		c.info.Shell = shellName + " " + version
	} else {
		c.info.Shell = shellName
	}
}
