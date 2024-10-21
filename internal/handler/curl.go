package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func (h *Handler) handleCurl(w http.ResponseWriter) {
	info := h.collector.GetInfo()
	logoData := h.getLogo(info.OS.Distro)
	if logoData == nil {
		logoData = h.logos[h.config.DefaultLogo]
	}

	// Parse colors
	colors := strings.Fields(logoData.Colors)
	colorCodes := make(map[string]string)
	for i, color := range colors {
		placeholder := fmt.Sprintf("${c%d}", i+1)
		colorCode := mapColorToANSI(color)
		colorCodes[placeholder] = colorCode
	}
	colorCodes["${c}"] = "\033[0m"

	var response strings.Builder

	asciiArtLines := logoData.AsciiArt

	// Prepare the info lines
	infoLines := []string{
		fmt.Sprintf("%s@%s", info.User, info.Host),
		"-------------",
		fmt.Sprintf("OS: %s %s", info.OS.Distro, info.OS.Arch),
		fmt.Sprintf("Kernel: %s", info.Kernel),
		fmt.Sprintf("Uptime: %s", info.Uptime),
		fmt.Sprintf("Packages: %s", info.Packages),
		fmt.Sprintf("Shell: %s", info.Shell),
		fmt.Sprintf("Resolution: %s", info.Resolution),
		fmt.Sprintf("DE: %s", info.DE),
		fmt.Sprintf("WM: %s", info.WM),
		fmt.Sprintf("WM Theme: %s", info.WMTheme),
		fmt.Sprintf("Theme: %s", info.Theme),
		fmt.Sprintf("Icons: %s", info.Icons),
		fmt.Sprintf("Terminal: %s", info.Terminal),
		fmt.Sprintf("CPU: %s (%d) @ %.2fGHz", info.CPU.Model, info.CPU.Cores, info.CPU.Frequency/1000),
		fmt.Sprintf("GPU: %s", info.GPU),
		fmt.Sprintf("Memory: %dMiB / %dMiB", info.Memory.Used/1024/1024, info.Memory.Total/1024/1024),
		fmt.Sprintf("Disk (/): %dG / %dG (%d%%)", info.Disk.Used/1024/1024/1024, info.Disk.Total/1024/1024/1024, int(info.Disk.UsedPercent)),
	}

	// Calculate the maximum length of the uncolored logo lines
	maxLogoWidth := 0
	plainLogoLines := make([]string, len(asciiArtLines))
	for i, line := range asciiArtLines {
		plainLine := line
		for k := range colorCodes {
			plainLine = strings.ReplaceAll(plainLine, k, "")
		}
		plainLine = stripANSICodes(plainLine)
		lineWidth := len([]rune(plainLine))
		if lineWidth > maxLogoWidth {
			maxLogoWidth = lineWidth
		}
		plainLogoLines[i] = plainLine
	}

	// Ensure we have enough info lines
	maxLines := len(asciiArtLines)
	if len(infoLines) > maxLines {
		maxLines = len(infoLines)
	}

	// Build the output
	for i := 0; i < maxLines; i++ {
		artLine := ""
		if i < len(asciiArtLines) {
			artLine = asciiArtLines[i]
			for k, v := range colorCodes {
				artLine = strings.ReplaceAll(artLine, k, v)
			}
			artLine += "\033[0m" // Reset color
		}

		// Pad art line to max width
		var plainArtLine string
		if i < len(plainLogoLines) {
			plainArtLine = plainLogoLines[i]
		}
		padding := maxLogoWidth - len([]rune(plainArtLine))
		if padding < 0 {
			padding = 0
		}
		artLinePadded := artLine + strings.Repeat(" ", padding)

		infoLine := ""
		if i < len(infoLines) {
			infoLine = infoLines[i]
		}

		response.WriteString(fmt.Sprintf("%s  %s\n", artLinePadded, infoLine))
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, response.String())
}

func mapColorToANSI(color string) string {
	colorMap := map[string]string{
		"fg": "\033[39m", // Default foreground
		"bg": "\033[49m", // Default background
	}
	if ansiCode, ok := colorMap[color]; ok {
		return ansiCode
	}

	ansiColorNum, err := strconv.Atoi(color)
	if err == nil && ansiColorNum >= 0 && ansiColorNum <= 255 {
		return fmt.Sprintf("\033[38;5;%sm", color)
	}

	// Default to reset if parsing fails
	return "\033[0m"
}

func stripANSICodes(str string) string {
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiEscape.ReplaceAllString(str, "")
}
