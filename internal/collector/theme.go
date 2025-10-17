package collector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func (c *Collector) collectTheme() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Theme = getCurrentTheme()
}

func (c *Collector) collectIcons() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Icons = getCurrentIcons()
}

func (c *Collector) collectFont() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Font = getCurrentFont()
}

func (c *Collector) collectCursor() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Cursor = getCurrentCursor()
}

func getCurrentTheme() string {
	if runtime.GOOS == "darwin" {
		return getMacOSTheme()
	}

	if runtime.GOOS == "windows" {
		return getWindowsTheme()
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "Unknown"
	}

	gtk2Theme := parseGTKSetting(filepath.Join(homeDir, ".gtkrc-2.0"), "gtk-theme-name")
	gtk3Theme := parseGTKSetting(filepath.Join(homeDir, ".config", "gtk-3.0", "settings.ini"), "gtk-theme-name")

	if gtk3Theme == "" {
		gtk3Theme = getGSettingsValue("org.gnome.desktop.interface", "gtk-theme")
	}

	if gtk2Theme != "" && gtk3Theme != "" {
		if gtk2Theme == gtk3Theme {
			return gtk2Theme
		}
		return fmt.Sprintf("%s [GTK2], %s [GTK3]", gtk2Theme, gtk3Theme)
	}

	if gtk3Theme != "" {
		return gtk3Theme
	}

	if gtk2Theme != "" {
		return gtk2Theme
	}

	kdeglobals := filepath.Join(homeDir, ".config", "kdeglobals")
	if theme := parseINISetting(kdeglobals, "General", "ColorScheme"); theme != "" {
		return theme
	}

	return "Unknown"
}

func getCurrentIcons() string {
	if runtime.GOOS == "darwin" {
		return "macOS"
	}

	if runtime.GOOS == "windows" {
		return "Windows"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "Unknown"
	}

	gtk2Icons := parseGTKSetting(filepath.Join(homeDir, ".gtkrc-2.0"), "gtk-icon-theme-name")
	gtk3Icons := parseGTKSetting(filepath.Join(homeDir, ".config", "gtk-3.0", "settings.ini"), "gtk-icon-theme-name")

	if gtk3Icons == "" {
		gtk3Icons = getGSettingsValue("org.gnome.desktop.interface", "icon-theme")
	}

	if gtk2Icons != "" && gtk3Icons != "" {
		if gtk2Icons == gtk3Icons {
			return gtk2Icons
		}
		return fmt.Sprintf("%s [GTK2], %s [GTK3]", gtk2Icons, gtk3Icons)
	}

	if gtk3Icons != "" {
		return gtk3Icons
	}

	if gtk2Icons != "" {
		return gtk2Icons
	}

	kdeglobals := filepath.Join(homeDir, ".config", "kdeglobals")
	if icons := parseINISetting(kdeglobals, "Icons", "Theme"); icons != "" {
		return icons
	}

	return "Unknown"
}

func getCurrentFont() string {
	if runtime.GOOS == "darwin" {
		return getMacOSFont()
	}

	if runtime.GOOS == "windows" {
		return getWindowsFont()
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "Unknown"
	}

	gtk2Font := parseGTKSetting(filepath.Join(homeDir, ".gtkrc-2.0"), "gtk-font-name")
	gtk3Font := parseGTKSetting(filepath.Join(homeDir, ".config", "gtk-3.0", "settings.ini"), "gtk-font-name")

	if gtk3Font == "" {
		gtk3Font = getGSettingsValue("org.gnome.desktop.interface", "font-name")
	}

	if gtk2Font != "" && gtk3Font != "" {
		if gtk2Font == gtk3Font {
			return gtk2Font
		}
		return fmt.Sprintf("%s [GTK2], %s [GTK3]", gtk2Font, gtk3Font)
	}

	if gtk3Font != "" {
		return gtk3Font
	}

	if gtk2Font != "" {
		return gtk2Font
	}

	kdeglobals := filepath.Join(homeDir, ".config", "kdeglobals")
	if font := parseINISetting(kdeglobals, "General", "font"); font != "" {
		return font
	}

	fontconfigPaths := []string{
		filepath.Join(homeDir, ".config", "fontconfig", "fonts.conf"),
		filepath.Join(homeDir, ".fonts.conf"),
	}

	for _, path := range fontconfigPaths {
		if font := parseFontConfig(path); font != "" {
			return font
		}
	}

	return "Unknown"
}

func getCurrentCursor() string {
	if runtime.GOOS == "darwin" {
		return "macOS"
	}

	if runtime.GOOS == "windows" {
		return "Windows"
	}

	if cursor := os.Getenv("XCURSOR_THEME"); cursor != "" {
		return cursor
	}

	if cursor := getGSettingsValue("org.gnome.desktop.interface", "cursor-theme"); cursor != "" {
		return cursor
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "Unknown"
	}

	gtkPaths := []string{
		filepath.Join(homeDir, ".config", "gtk-4.0", "settings.ini"),
		filepath.Join(homeDir, ".config", "gtk-3.0", "settings.ini"),
		filepath.Join(homeDir, ".gtkrc-2.0"),
	}

	for _, path := range gtkPaths {
		if cursor := parseGTKSetting(path, "gtk-cursor-theme-name"); cursor != "" {
			return cursor
		}
	}

	xresources := filepath.Join(homeDir, ".Xresources")
	if cursor := parseXResourcesSetting(xresources, "Xcursor.theme"); cursor != "" {
		return cursor
	}

	iconsIndex := filepath.Join(homeDir, ".icons", "default", "index.theme")
	if cursor := parseINISetting(iconsIndex, "Icon Theme", "Inherits"); cursor != "" {
		return cursor
	}

	return "Unknown"
}

func parseGTKSetting(path, key string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, key+"=") {
			value := strings.TrimPrefix(line, key+"=")
			value = strings.Trim(value, `"'`)
			return value
		}
	}

	return ""
}

func parseINISetting(path, section, key string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	inSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection := strings.Trim(line, "[]")
			inSection = (currentSection == section)
			continue
		}

		if inSection && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

func parseFontConfig(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "<family>") {
			start := strings.Index(line, "<family>") + 8
			end := strings.Index(line, "</family>")
			if start >= 8 && end > start {
				return line[start:end]
			}
		}
	}

	return ""
}

func parseXResourcesSetting(path, key string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+":") {
			value := strings.TrimSpace(strings.TrimPrefix(line, key+":"))
			return value
		}
	}

	return ""
}

func getGSettingsValue(schema, key string) string {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return ""
	}

	value := strings.TrimSpace(string(out))
	value = strings.Trim(value, "'\"")

	return value
}

func getMacOSTheme() string {
	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err != nil {
		return "Light"
	}

	if strings.Contains(string(out), "Dark") {
		return "Dark"
	}

	return "Light"
}

func getWindowsTheme() string {
	return "Windows"
}

func getMacOSFont() string {
	return "San Francisco"
}

func getWindowsFont() string {
	return "Segoe UI"
}
