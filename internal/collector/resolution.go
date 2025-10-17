package collector

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func (c *Collector) collectResolution() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.Resolution = getResolutionLinux()
	case "darwin":
		c.info.Resolution = getResolutionDarwin()
	case "windows":
		c.info.Resolution = getResolutionWindows()
	case "freebsd", "openbsd", "netbsd":
		c.info.Resolution = getResolutionBSD()
	default:
		c.info.Resolution = "Unknown"
	}
}

func getResolutionLinux() string {
	if res := getResolutionWayland(); res != "" {
		return res
	}

	if res := getResolutionX11(); res != "" {
		return res
	}

	if res := getResolutionDRM(); res != "" {
		return res
	}

	return "Unknown"
}

func getResolutionWayland() string {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return ""
	}

	out, err := exec.Command("wlr-randr").Output()
	if err == nil {
		return parseWlrRandr(string(out))
	}

	return ""
}

func parseWlrRandr(output string) string {
	var resolutions []string
	re := regexp.MustCompile(`(\d+x\d+).*current`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if matches := re.FindStringSubmatch(line); len(matches) >= 2 {
			resolutions = append(resolutions, matches[1])
		}
	}

	if len(resolutions) > 0 {
		return strings.Join(resolutions, ", ")
	}

	return ""
}

func getResolutionX11() string {
	displayEnv := os.Getenv("DISPLAY")
	if displayEnv == "" {
		os.Setenv("DISPLAY", ":0")
	}

	out, err := exec.Command("xrandr", "--current").Output()
	if err != nil {
		return ""
	}

	return parseXrandr(string(out))
}

func parseXrandr(output string) string {
	var resolutions []string
	re := regexp.MustCompile(`(\d+x\d+)\+\d+\+\d+`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, " connected") {
			if matches := re.FindStringSubmatch(line); len(matches) >= 2 {
				resolutions = append(resolutions, matches[1])
			}
		}
	}

	if len(resolutions) > 0 {
		return strings.Join(resolutions, ", ")
	}

	re2 := regexp.MustCompile(`(\d+x\d+).*\*`)
	for _, line := range lines {
		if matches := re2.FindStringSubmatch(line); len(matches) >= 2 {
			resolutions = append(resolutions, matches[1])
		}
	}

	if len(resolutions) > 0 {
		return strings.Join(resolutions, ", ")
	}

	return ""
}

func getResolutionDRM() string {
	var resolutions []string

	drmDir := "/sys/class/drm"
	entries, err := os.ReadDir(drmDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "card") {
			continue
		}

		statusPath := filepath.Join(drmDir, entry.Name(), "status")
		status, err := os.ReadFile(statusPath)
		if err != nil || strings.TrimSpace(string(status)) != "connected" {
			continue
		}

		modesPath := filepath.Join(drmDir, entry.Name(), "modes")
		modes, err := os.ReadFile(modesPath)
		if err != nil {
			continue
		}

		modeLines := strings.Split(strings.TrimSpace(string(modes)), "\n")
		if len(modeLines) > 0 {
			resolutions = append(resolutions, modeLines[0])
		}
	}

	if len(resolutions) > 0 {
		return strings.Join(resolutions, ", ")
	}

	return ""
}

func getResolutionDarwin() string {
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return "Unknown"
	}

	var resolutions []string
	re := regexp.MustCompile(`Resolution:\s*(\d+\s*x\s*\d+)`)

	matches := re.FindAllStringSubmatch(string(out), -1)
	for _, match := range matches {
		if len(match) >= 2 {
			res := strings.ReplaceAll(match[1], " ", "")
			resolutions = append(resolutions, res)
		}
	}

	if len(resolutions) > 0 {
		return strings.Join(resolutions, ", ")
	}

	return "Unknown"
}

func getResolutionWindows() string {
	out, err := exec.Command("wmic", "path", "Win32_VideoController", "get", "CurrentHorizontalResolution,CurrentVerticalResolution", "/format:list").Output()
	if err != nil {
		return "Unknown"
	}

	var width, height string
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CurrentHorizontalResolution=") {
			width = strings.TrimPrefix(line, "CurrentHorizontalResolution=")
		} else if strings.HasPrefix(line, "CurrentVerticalResolution=") {
			height = strings.TrimPrefix(line, "CurrentVerticalResolution=")
		}
	}

	if width != "" && height != "" {
		return width + "x" + height
	}

	return "Unknown"
}

func getResolutionBSD() string {
	out, err := exec.Command("xrandr", "--current").Output()
	if err != nil {
		return "Unknown"
	}

	return parseXrandr(string(out))
}
