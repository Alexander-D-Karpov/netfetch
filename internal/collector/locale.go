package collector

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func (c *Collector) collectLocale() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		c.info.Locale = getLocaleUnix()
	case "windows":
		c.info.Locale = getLocaleWindows()
	default:
		c.info.Locale = "Unknown"
	}
}

func getLocaleUnix() string {
	localeVars := []string{"LANG", "LC_ALL", "LC_MESSAGES", "LANGUAGE"}

	for _, varName := range localeVars {
		if locale := os.Getenv(varName); locale != "" {
			return locale
		}
	}

	out, err := exec.Command("locale").Output()
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "LANG=") {
			locale := strings.TrimPrefix(line, "LANG=")
			locale = strings.Trim(locale, `"'`)
			return locale
		}
	}

	return "Unknown"
}

func getLocaleWindows() string {
	out, err := exec.Command("powershell", "-Command", "Get-WinSystemLocale | Select-Object -ExpandProperty Name").Output()
	if err != nil {
		out, err = exec.Command("cmd", "/c", "echo %LANG%").Output()
		if err != nil {
			return "Unknown"
		}
	}

	locale := strings.TrimSpace(string(out))
	if locale == "" || locale == "%LANG%" {
		return "en-US"
	}

	return locale
}
