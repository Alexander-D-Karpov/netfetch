package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func (c *Collector) collectTerminal() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Terminal = getTerminal()
}

func getTerminal() string {
	if runtime.GOOS == "windows" {
		return getTerminalWindows()
	}

	if runtime.GOOS == "darwin" {
		return getTerminalDarwin()
	}

	if term := os.Getenv("TERM_PROGRAM"); term != "" {
		return term
	}

	if term := os.Getenv("TERMINAL_EMULATOR"); term != "" {
		return term
	}

	if os.Getenv("KITTY_PID") != "" {
		return "kitty"
	}

	if os.Getenv("ALACRITTY_SOCKET") != "" {
		return "Alacritty"
	}

	if os.Getenv("WEZTERM_EXECUTABLE") != "" {
		return "WezTerm"
	}

	if os.Getenv("WT_SESSION") != "" {
		return "Windows Terminal"
	}

	if os.Getenv("KONSOLE_VERSION") != "" {
		return "Konsole"
	}

	if os.Getenv("GNOME_TERMINAL_SERVICE") != "" {
		return "GNOME Terminal"
	}

	if os.Getenv("TERMINATOR_UUID") != "" {
		return "Terminator"
	}

	detected := detectTerminalFromProcessTree()
	if detected != "Unknown" {
		return detected
	}

	tty := getTTY()
	if tty != "" {
		return tty
	}

	return "Unknown"
}

func getTTY() string {
	tty, err := os.Readlink("/proc/self/fd/0")
	if err == nil && strings.HasPrefix(tty, "/dev/") {
		return tty
	}

	return ""
}

func detectTerminalFromProcessTree() string {
	pid := os.Getppid()
	skipProcesses := map[string]bool{
		"login":   true,
		"init":    true,
		"systemd": true,
		"sshd":    true,
		"ssh":     true,
		"tmux":    true,
		"screen":  true,
		"zellij":  true,
		"sh":      true,
		"bash":    true,
		"zsh":     true,
		"fish":    true,
		"dash":    true,
		"ksh":     true,
		"tcsh":    true,
		"csh":     true,
		"su":      true,
		"sudo":    true,
		"doas":    true,
	}

	terminalProcesses := map[string]string{
		"alacritty":       "Alacritty",
		"kitty":           "kitty",
		"wezterm":         "WezTerm",
		"wezterm-gui":     "WezTerm",
		"gnome-terminal":  "GNOME Terminal",
		"konsole":         "Konsole",
		"xfce4-terminal":  "XFCE Terminal",
		"xterm":           "xterm",
		"urxvt":           "urxvt",
		"rxvt":            "rxvt",
		"terminator":      "Terminator",
		"tilix":           "Tilix",
		"st":              "st",
		"cool-retro-term": "cool-retro-term",
		"lxterminal":      "LXTerminal",
		"mate-terminal":   "MATE Terminal",
		"terminology":     "Terminology",
		"hyper":           "Hyper",
		"Hyper":           "Hyper",
		"foot":            "foot",
		"ghostty":         "Ghostty",
		"goland":          "goland",
		"idea":            "IntelliJ IDEA",
		"pycharm":         "PyCharm",
		"webstorm":        "WebStorm",
		"code":            "VS Code",
		"code-oss":        "VS Code",
		"cursor":          "Cursor",
	}

	ideProcesses := map[string]string{
		"goland":   "goland",
		"idea":     "IntelliJ IDEA",
		"pycharm":  "PyCharm",
		"webstorm": "WebStorm",
		"code":     "VS Code",
		"code-oss": "VS Code",
		"cursor":   "Cursor",
	}

	foundIDE := ""

	for i := 0; i < 20 && pid > 1; i++ {
		cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
		cmdlineData, err := os.ReadFile(cmdlinePath)
		if err == nil {
			cmdline := string(cmdlineData)
			parts := strings.Split(cmdline, "\x00")
			if len(parts) > 0 {
				exePath := parts[0]
				exeName := filepath.Base(exePath)

				if ideName, isIDE := ideProcesses[exeName]; isIDE {
					foundIDE = ideName
				}

				if !skipProcesses[exeName] {
					if terminal, ok := terminalProcesses[exeName]; ok {
						if foundIDE != "" {
							return foundIDE
						}
						return terminal
					}
				}
			}
		}

		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		statData, err := os.ReadFile(statPath)
		if err != nil {
			break
		}

		statStr := string(statData)
		closeParen := strings.LastIndex(statStr, ")")
		if closeParen == -1 {
			break
		}

		fields := strings.Fields(statStr[closeParen+1:])
		if len(fields) < 2 {
			break
		}

		ppid, err := strconv.Atoi(fields[1])
		if err != nil || ppid <= 1 {
			break
		}

		pid = ppid
	}

	if foundIDE != "" {
		return foundIDE
	}

	return "Unknown"
}

func getTerminalWindows() string {
	if os.Getenv("WT_SESSION") != "" {
		return "Windows Terminal"
	}

	if os.Getenv("ConEmuPID") != "" {
		return "ConEmu"
	}

	if os.Getenv("ALACRITTY_SOCKET") != "" {
		return "Alacritty"
	}

	return "cmd"
}

func getTerminalDarwin() string {
	if term := os.Getenv("TERM_PROGRAM"); term != "" {
		termMap := map[string]string{
			"Apple_Terminal": "Terminal.app",
			"iTerm.app":      "iTerm2",
			"WezTerm":        "WezTerm",
			"Alacritty":      "Alacritty",
			"kitty":          "kitty",
		}

		if mapped, ok := termMap[term]; ok {
			return mapped
		}
		return term
	}

	return "Terminal.app"
}
