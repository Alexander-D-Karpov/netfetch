package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (c *Collector) collectTerminal() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	terminal := getTerminal()
	c.info.Terminal = terminal
}

func getTerminal() string {
	// Check environment variables
	terminal := os.Getenv("TERM_PROGRAM")
	if terminal != "" {
		return terminal
	}

	// Walk up the process tree to find the terminal emulator
	pid := os.Getppid()
	for pid > 1 {
		exePath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if err == nil {
			exeName := filepath.Base(exePath)
			if isTerminalEmulator(exeName) {
				return exeName
			}
		}
		// Get parent process ID
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			break
		}
		fields := strings.Fields(string(stat))
		if len(fields) >= 4 {
			pid, _ = strconv.Atoi(fields[3]) // Parent PID is the fourth field
		} else {
			break
		}
	}
	return "Unknown"
}

func isTerminalEmulator(name string) bool {
	terminalEmulators := []string{
		"alacritty", "gnome-terminal", "konsole", "xfce4-terminal", "xterm",
		"terminator", "tilix", "st", "urxvt", "kitty", "hyper", "cool-retro-term",
		"lxterminal", "wezterm",
	}
	for _, term := range terminalEmulators {
		if strings.Contains(name, term) {
			return true
		}
	}
	return false
}
