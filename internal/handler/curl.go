package handler

import (
	"fmt"
	"net/http"
	"netfetch/internal/logo"
	"regexp"
	"strconv"
	"strings"
)

const (
	keyColor   = "\033[96m" // Bright cyan
	resetColor = "\033[0m"
)

func (h *Handler) handleCurl(w http.ResponseWriter) {
	info := h.collector.GetInfo()
	if info == nil {
		http.Error(w, "Failed to get system info", http.StatusInternalServerError)
		return
	}

	// Get logo, falling back to default if necessary
	var logoData *logo.Logo
	if info.OS != nil && info.OS.Distro != "" {
		logoData = h.getLogo(info.OS.Distro)
	}
	if logoData == nil {
		logoData = h.getLogo(h.config.DefaultLogo)
	}
	if logoData == nil {
		http.Error(w, "No logo available", http.StatusInternalServerError)
		return
	}

	// Parse colors and setup color codes
	colors := strings.Fields(logoData.Colors)
	colorCodes := make(map[string]string)
	defaultColor := ""

	// Set up color codes map and get default color
	if len(colors) > 0 {
		defaultColor = mapColorToANSI(colors[0]) // First color is default
		for i, color := range colors {
			placeholder := fmt.Sprintf("${c%d}", i+1)
			colorCode := mapColorToANSI(color)
			colorCodes[placeholder] = colorCode
		}
	}
	colorCodes["${c}"] = resetColor

	var response strings.Builder
	asciiArtLines := logoData.AsciiArt

	// Safely get values with nil checks
	user := getValueOrDefault(info.User, "unknown")
	host := getValueOrDefault(info.Host, "unknown")
	osInfo := "unknown"
	if info.OS != nil {
		osInfo = fmt.Sprintf("%s %s", info.OS.Distro, info.OS.Arch)
	}

	// Helper functions for system info
	getCPUInfo := func() string {
		if info.CPU == nil {
			return "unknown"
		}
		return fmt.Sprintf("%s (%d) @ %.2fGHz",
			info.CPU.Name,
			info.CPU.CoresPhysical,
			float64(info.CPU.FrequencyBase)/1000)
	}

	getMemoryInfo := func() string {
		if info.Memory == nil {
			return "unknown"
		}
		return fmt.Sprintf("%s / %s",
			formatSize(info.Memory.Used),
			formatSize(info.Memory.Total))
	}

	getDiskInfo := func() string {
		if info.Disk == nil {
			return "unknown"
		}
		return fmt.Sprintf("%s / %s (%d%%)",
			formatSize(info.Disk.Used),
			formatSize(info.Disk.Total),
			int(info.Disk.UsedPercent))
	}

	// Helper functions for additional modules
	getSwapInfo := func() string {
		if info.Swap == nil {
			return "unknown"
		}
		return fmt.Sprintf("%s / %s",
			formatSize(info.Swap.Used),
			formatSize(info.Swap.Total))
	}

	getBatteryInfo := func() string {
		if info.Battery == nil {
			return "unknown"
		}
		return fmt.Sprintf("%.0f%% (%s)",
			info.Battery.Percentage,
			info.Battery.Status)
	}

	// Prepare info lines with colored keys
	infoLines := []string{
		fmt.Sprintf("%s%s@%s%s", keyColor, user, host, resetColor),
		"-------------",
		fmt.Sprintf("%sOS:%s %s", keyColor, resetColor, osInfo),
		fmt.Sprintf("%sKernel:%s %s", keyColor, resetColor, info.Kernel),
		fmt.Sprintf("%sUptime:%s %s", keyColor, resetColor, info.Uptime),
		fmt.Sprintf("%sPackages:%s %s", keyColor, resetColor, info.Packages),
		fmt.Sprintf("%sShell:%s %s", keyColor, resetColor, info.Shell),
		fmt.Sprintf("%sResolution:%s %s", keyColor, resetColor, info.Resolution),
		fmt.Sprintf("%sDE:%s %s", keyColor, resetColor, info.DE),
		fmt.Sprintf("%sWM:%s %s", keyColor, resetColor, info.WM),
		fmt.Sprintf("%sWM Theme:%s %s", keyColor, resetColor, info.WMTheme),
		fmt.Sprintf("%sTheme:%s %s", keyColor, resetColor, info.Theme),
		fmt.Sprintf("%sIcons:%s %s", keyColor, resetColor, info.Icons),
		fmt.Sprintf("%sTerminal:%s %s", keyColor, resetColor, info.Terminal),
		fmt.Sprintf("%sCPU:%s %s", keyColor, resetColor, getCPUInfo()),
		fmt.Sprintf("%sGPU:%s %s", keyColor, resetColor, info.GPU),
		fmt.Sprintf("%sMemory:%s %s", keyColor, resetColor, getMemoryInfo()),
		fmt.Sprintf("%sDisk (/):%s %s", keyColor, resetColor, getDiskInfo()),
		fmt.Sprintf("%sSwap:%s %s", keyColor, resetColor, getSwapInfo()),
		fmt.Sprintf("%sBattery:%s %s", keyColor, resetColor, getBatteryInfo()),
	}

	// Process logo lines and calculate max width
	maxLogoWidth := 0
	processedLogoLines := make([]string, len(asciiArtLines))
	plainLogoLines := make([]string, len(asciiArtLines))

	for i, line := range asciiArtLines {
		// Store plain version for width calculation
		plainLine := line
		for k := range colorCodes {
			plainLine = strings.ReplaceAll(plainLine, k, "")
		}
		plainLine = stripANSICodes(plainLine)
		plainLogoLines[i] = plainLine

		// Calculate max width from plain version
		lineWidth := len([]rune(plainLine))
		if lineWidth > maxLogoWidth {
			maxLogoWidth = lineWidth
		}

		// Process colored version
		processedLine := line
		hasColorPlaceholder := false
		for k, v := range colorCodes {
			if strings.Contains(processedLine, k) {
				hasColorPlaceholder = true
				processedLine = strings.ReplaceAll(processedLine, k, v)
			}
		}

		// Apply default color if no placeholders present
		if !hasColorPlaceholder && len(processedLine) > 0 {
			processedLine = defaultColor + processedLine
		}

		// Ensure color reset at end of line
		if len(processedLine) > 0 {
			processedLine += resetColor
		}

		processedLogoLines[i] = processedLine
	}

	// Build the output
	maxLines := max(len(processedLogoLines), len(infoLines))
	for i := 0; i < maxLines; i++ {
		var artLine string
		if i < len(processedLogoLines) {
			artLine = processedLogoLines[i]

			// Calculate padding based on plain version
			plainWidth := len([]rune(plainLogoLines[i]))
			padding := maxLogoWidth - plainWidth
			if padding > 0 {
				artLine += strings.Repeat(" ", padding)
			}
		} else {
			artLine = strings.Repeat(" ", maxLogoWidth)
		}

		infoLine := ""
		if i < len(infoLines) {
			infoLine = infoLines[i]
		}

		response.WriteString(fmt.Sprintf("%s  %s\n", artLine, infoLine))
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := fmt.Fprint(w, response.String())
	if err != nil {
		return
	}
}

func mapColorToANSI(color string) string {
	if color == "fg" {
		return "\033[39m" // Default foreground
	}
	if color == "bg" {
		return "\033[49m" // Default background
	}

	ansiColorNum, err := strconv.Atoi(color)
	if err == nil {
		if ansiColorNum >= 0 && ansiColorNum <= 7 {
			return fmt.Sprintf("\033[3%dm", ansiColorNum)
		} else if ansiColorNum >= 8 && ansiColorNum <= 15 {
			return fmt.Sprintf("\033[9%dm", ansiColorNum-8)
		} else if ansiColorNum >= 16 && ansiColorNum <= 255 {
			return fmt.Sprintf("\033[38;5;%dm", ansiColorNum)
		}
	}

	return "\033[0m" // Default to reset if parsing fails
}

func stripANSICodes(str string) string {
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiEscape.ReplaceAllString(str, "")
}

func getValueOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

func formatSize(bytes uint64) string {
	const (
		_         = iota             // ignore first value by assigning to blank identifier
		KB uint64 = 1 << (10 * iota) // 1 << (10 * 1) = 1024
		MB                           // 1 << (10 * 2) = 1,048,576
		GB                           // 1 << (10 * 3) = 1,073,741,824
		TB                           // 1 << (10 * 4) = 1,099,511,627,776
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TiB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
