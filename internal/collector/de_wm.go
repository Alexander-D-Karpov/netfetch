package collector

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func (c *Collector) collectDE() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.DE = getDE()
}

func (c *Collector) collectWM() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.WM = getWM()
	c.info.WMTheme = getWMTheme(c.info.WM)
}

func getDE() string {
	if runtime.GOOS == "darwin" {
		return "Aqua"
	}

	if runtime.GOOS == "windows" {
		return "Windows Explorer"
	}

	if de := os.Getenv("XDG_CURRENT_DESKTOP"); de != "" {
		parts := strings.Split(de, ":")
		return parts[0]
	}

	if de := os.Getenv("DESKTOP_SESSION"); de != "" {
		return de
	}

	if de := os.Getenv("GDMSESSION"); de != "" {
		return de
	}

	if de := os.Getenv("XDG_SESSION_DESKTOP"); de != "" {
		return de
	}

	deProcesses := map[string][]string{
		"GNOME":    {"gnome-session", "gnome-shell"},
		"KDE":      {"plasmashell", "ksmserver"},
		"XFCE":     {"xfce4-session"},
		"Cinnamon": {"cinnamon-session", "cinnamon"},
		"MATE":     {"mate-session"},
		"Unity":    {"unity-panel-service"},
		"LXDE":     {"lxsession"},
		"LXQt":     {"lxqt-session"},
		"Deepin":   {"dde-desktop", "dde-session"},
		"Pantheon": {"pantheon-session"},
		"Budgie":   {"budgie-desktop", "budgie-wm"},
		"Trinity":  {"trinity-session"},
	}

	return detectProcess(deProcesses)
}

func getWM() string {
	if runtime.GOOS == "darwin" {
		return detectMacOSWM()
	}

	if runtime.GOOS == "windows" {
		return "DWM"
	}

	if wm := os.Getenv("WAYLAND_DISPLAY"); wm != "" {
		if detected := detectWaylandCompositor(); detected != "" {
			return detected
		}
	}

	wmProcesses := map[string][]string{
		"i3":            {"i3", "i3-gaps"},
		"Sway":          {"sway"},
		"Hyprland":      {"Hyprland"},
		"bspwm":         {"bspwm"},
		"awesome":       {"awesome"},
		"dwm":           {"dwm"},
		"Openbox":       {"openbox"},
		"Fluxbox":       {"fluxbox"},
		"IceWM":         {"icewm"},
		"JWM":           {"jwm"},
		"herbstluftwm":  {"herbstluftwm"},
		"qtile":         {"qtile"},
		"xmonad":        {"xmonad"},
		"Mutter":        {"mutter"},
		"KWin":          {"kwin_x11", "kwin_wayland"},
		"Xfwm4":         {"xfwm4"},
		"Marco":         {"marco"},
		"Metacity":      {"metacity"},
		"Compiz":        {"compiz"},
		"Enlightenment": {"enlightenment"},
		"fvwm":          {"fvwm", "fvwm3"},
		"ctwm":          {"ctwm"},
		"ratpoison":     {"ratpoison"},
		"Wayfire":       {"wayfire"},
		"River":         {"river"},
		"Labwc":         {"labwc"},
	}

	return detectProcess(wmProcesses)
}

func detectMacOSWM() string {
	wmProcesses := map[string][]string{
		"yabai":     {"yabai"},
		"Aerospace": {"aerospace"},
		"Rectangle": {"Rectangle"},
		"Amethyst":  {"Amethyst"},
		"Magnet":    {"Magnet"},
	}

	if wm := detectProcess(wmProcesses); wm != "" {
		return wm
	}

	return "Quartz Compositor"
}

func detectWaylandCompositor() string {
	compositorProcesses := map[string][]string{
		"Hyprland": {"Hyprland"},
		"Sway":     {"sway"},
		"Wayfire":  {"wayfire"},
		"River":    {"river"},
		"Labwc":    {"labwc"},
		"KWin":     {"kwin_wayland"},
		"Mutter":   {"mutter"},
		"wlroots":  {"wlroots"},
	}

	return detectProcess(compositorProcesses)
}

func detectProcess(processMap map[string][]string) string {
	if runtime.GOOS != "linux" && runtime.GOOS != "freebsd" && runtime.GOOS != "openbsd" && runtime.GOOS != "netbsd" && runtime.GOOS != "darwin" {
		return "Unknown"
	}

	procDir := "/proc"
	if runtime.GOOS == "darwin" {
		return detectProcessDarwin(processMap)
	}

	entries, err := os.ReadDir(procDir)
	if err != nil {
		return "Unknown"
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid := entry.Name()
		if pid[0] < '0' || pid[0] > '9' {
			continue
		}

		cmdlinePath := filepath.Join(procDir, pid, "cmdline")
		cmdline, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}

		cmd := strings.Split(string(cmdline), "\x00")[0]
		exe := filepath.Base(cmd)

		for name, executables := range processMap {
			for _, executable := range executables {
				if exe == executable || strings.HasPrefix(exe, executable) {
					return name
				}
			}
		}
	}

	return "Unknown"
}

func detectProcessDarwin(processMap map[string][]string) string {
	for name, executables := range processMap {
		for _, executable := range executables {
			if isProcessRunningDarwin(executable) {
				return name
			}
		}
	}
	return ""
}

func isProcessRunningDarwin(process string) bool {
	entries, err := os.ReadDir("/proc")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			cmdlinePath := filepath.Join("/proc", entry.Name(), "cmdline")
			cmdline, err := os.ReadFile(cmdlinePath)
			if err != nil {
				continue
			}

			if strings.Contains(string(cmdline), process) {
				return true
			}
		}
	}

	return false
}

func getWMTheme(wm string) string {
	if wm == "KWin" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "Unknown"
		}

		kwinrc := filepath.Join(homeDir, ".config", "kwinrc")
		content, err := os.ReadFile(kwinrc)
		if err != nil {
			return "Unknown"
		}

		lines := strings.Split(string(content), "\n")
		inSection := false

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if line == "[WindowDecoration]" {
				inSection = true
				continue
			}

			if inSection {
				if strings.HasPrefix(line, "[") {
					break
				}

				if strings.HasPrefix(line, "theme=") {
					return strings.TrimSpace(strings.TrimPrefix(line, "theme="))
				}
			}
		}
	}

	return "Unknown"
}
