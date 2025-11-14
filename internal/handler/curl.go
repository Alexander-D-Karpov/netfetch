package handler

import (
	"fmt"
	"net/http"
	"netfetch/internal/logo"
	"netfetch/internal/model"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	keyColor   = "\033[96m"
	resetColor = "\033[0m"
	goodColor  = "\033[92m"
	warnColor  = "\033[93m"
	errorColor = "\033[91m"
)

func (h *Handler) handleCurl(w http.ResponseWriter) {
	info := h.collector.GetInfo()
	if info == nil {
		http.Error(w, "Failed to get system info", http.StatusInternalServerError)
		return
	}

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

	colors := strings.Fields(logoData.Colors)
	colorCodes := make(map[string]string)
	defaultColor := ""

	if len(colors) > 0 {
		defaultColor = mapColorToANSI(colors[0])
		for i, color := range colors {
			placeholder := fmt.Sprintf("${c%d}", i+1)
			colorCode := mapColorToANSI(color)
			colorCodes[placeholder] = colorCode
		}
	}
	colorCodes["${c}"] = resetColor

	var response strings.Builder
	asciiArtLines := logoData.AsciiArt

	user := getValueOrDefault(info.User, "unknown")
	host := getValueOrDefault(info.Host, "unknown")
	osInfo := "unknown"
	if info.OS != nil {
		osInfo = fmt.Sprintf("%s %s", info.OS.Distro, info.OS.Arch)
	}

	colorUsage := func(pct float64) string {
		switch {
		case pct >= 90:
			return errorColor
		case pct >= 70:
			return warnColor
		default:
			return goodColor
		}
	}

	getCPUInfo := func() string {
		if info.CPU == nil {
			return "unknown"
		}
		return fmt.Sprintf("%s (%d) @ %.2fGHz",
			info.CPU.Name,
			info.CPU.CoresLogical,
			float64(info.CPU.FrequencyMax)/1000)
	}

	getMemoryInfo := func() string {
		if info.Memory == nil || info.Memory.Total == 0 {
			return "unknown"
		}
		pct := (float64(info.Memory.Used) / float64(info.Memory.Total)) * 100
		return fmt.Sprintf("%s%s%s / %s %s(%d%%)%s",
			colorUsage(pct),
			formatSize(info.Memory.Used),
			resetColor,
			formatSize(info.Memory.Total),
			colorUsage(pct),
			int(pct),
			resetColor)
	}

	getSwapInfo := func() string {
		if info.Swap == nil || info.Swap.Total == 0 {
			return "not configured"
		}
		pct := (float64(info.Swap.Used) / float64(info.Swap.Total)) * 100
		return fmt.Sprintf("%s%s%s / %s %s(%d%%)%s",
			colorUsage(pct),
			formatSize(info.Swap.Used),
			resetColor,
			formatSize(info.Swap.Total),
			colorUsage(pct),
			int(pct),
			resetColor)
	}

	getBatteryInfo := func() string {
		if info.Battery == nil {
			return "unknown"
		}
		col := goodColor
		if info.Battery.Percentage <= 20 {
			col = errorColor
		} else if info.Battery.Percentage <= 50 {
			col = warnColor
		}
		return fmt.Sprintf("%s%.0f%%%s (%s)",
			col,
			info.Battery.Percentage,
			resetColor,
			info.Battery.Status)
	}

	getDiskLines := func() []string {
		if len(info.Disks) == 0 {
			if info.Disk != nil && info.Disk.Total > 0 {
				pct := info.Disk.UsedPercent
				fs := info.Disk.FSType
				mp := info.Disk.Mountpoint
				if mp == "" {
					mp = "/"
				}
				return []string{fmt.Sprintf("%sDisk (%s):%s %s%s%s / %s %s(%d%%)%s - %s",
					keyColor, mp, resetColor,
					colorUsage(pct),
					formatSize(info.Disk.Used),
					resetColor,
					formatSize(info.Disk.Total),
					colorUsage(pct),
					int(pct),
					resetColor,
					fs)}
			}
			return []string{fmt.Sprintf("%sDisk:%s unknown", keyColor, resetColor)}
		}

		sortedDisks := make([]model.DiskInfo, len(info.Disks))
		copy(sortedDisks, info.Disks)
		sort.SliceStable(sortedDisks, func(i, j int) bool {
			if sortedDisks[i].Mountpoint == "/" {
				return true
			}
			if sortedDisks[j].Mountpoint == "/" {
				return false
			}
			return sortedDisks[i].Mountpoint < sortedDisks[j].Mountpoint
		})

		var lines []string
		for _, disk := range sortedDisks {
			pct := disk.UsedPercent
			lines = append(lines, fmt.Sprintf("%sDisk (%s):%s %s%s%s / %s %s(%d%%)%s - %s",
				keyColor,
				disk.Mountpoint,
				resetColor,
				colorUsage(pct),
				formatSize(disk.Used),
				resetColor,
				formatSize(disk.Total),
				colorUsage(pct),
				int(pct),
				resetColor,
				disk.FSType))
		}
		return lines
	}

	isActive := func(name string) bool {
		for _, m := range h.config.ActiveModules {
			if m == name {
				return true
			}
		}
		return false
	}

	infoLines := []string{
		fmt.Sprintf("%s%s@%s%s", keyColor, user, host, resetColor),
		"-------------",
	}

	if isActive("os") {
		infoLines = append(infoLines, fmt.Sprintf("%sOS:%s %s", keyColor, resetColor, osInfo))
	}
	if isActive("kernel") {
		infoLines = append(infoLines, fmt.Sprintf("%sKernel:%s %s", keyColor, resetColor, info.Kernel))
	}
	if isActive("uptime") {
		infoLines = append(infoLines, fmt.Sprintf("%sUptime:%s %s", keyColor, resetColor, info.Uptime))
	}
	if isActive("packages") {
		infoLines = append(infoLines, fmt.Sprintf("%sPackages:%s %s", keyColor, resetColor, info.Packages))
	}
	if isActive("shell") {
		infoLines = append(infoLines, fmt.Sprintf("%sShell:%s %s", keyColor, resetColor, info.Shell))
	}
	if isActive("resolution") {
		infoLines = append(infoLines, fmt.Sprintf("%sResolution:%s %s", keyColor, resetColor, info.Resolution))
	}
	if isActive("de") {
		infoLines = append(infoLines, fmt.Sprintf("%sDE:%s %s", keyColor, resetColor, info.DE))
	}
	if isActive("wm") {
		infoLines = append(infoLines, fmt.Sprintf("%sWM:%s %s", keyColor, resetColor, info.WM))
		if info.WMTheme != "Unknown" && info.WMTheme != "" {
			infoLines = append(infoLines, fmt.Sprintf("%sWM Theme:%s %s", keyColor, resetColor, info.WMTheme))
		}
	}
	if isActive("theme") {
		infoLines = append(infoLines, fmt.Sprintf("%sTheme:%s %s", keyColor, resetColor, info.Theme))
	}
	if isActive("icons") {
		infoLines = append(infoLines, fmt.Sprintf("%sIcons:%s %s", keyColor, resetColor, info.Icons))
	}
	if isActive("terminal") {
		infoLines = append(infoLines, fmt.Sprintf("%sTerminal:%s %s", keyColor, resetColor, info.Terminal))
	}
	if isActive("cpu") {
		infoLines = append(infoLines, fmt.Sprintf("%sCPU:%s %s", keyColor, resetColor, getCPUInfo()))
	}
	if isActive("gpu") {
		infoLines = append(infoLines, fmt.Sprintf("%sGPU:%s %s", keyColor, resetColor, info.GPU))
	}
	if isActive("memory") {
		infoLines = append(infoLines, fmt.Sprintf("%sMemory:%s %s", keyColor, resetColor, getMemoryInfo()))
	}

	if isActive("disk") {
		infoLines = append(infoLines, getDiskLines()...)
	}

	if isActive("swap") {
		infoLines = append(infoLines, fmt.Sprintf("%sSwap:%s %s", keyColor, resetColor, getSwapInfo()))
	}
	if isActive("battery") {
		infoLines = append(infoLines, fmt.Sprintf("%sBattery:%s %s", keyColor, resetColor, getBatteryInfo()))
	}
	if isActive("locale") {
		locale := getValueOrDefault(info.Locale, "unknown")
		infoLines = append(infoLines, fmt.Sprintf("%sLocale:%s %s", keyColor, resetColor, locale))
	}
	if isActive("hostinfo") && info.HostInfo != nil {
		hostStr := ""
		if info.HostInfo.Model != "" {
			hostStr = info.HostInfo.Model
			if info.HostInfo.Vendor != "" && info.HostInfo.Vendor != info.HostInfo.Model {
				hostStr = fmt.Sprintf("%s %s", info.HostInfo.Vendor, info.HostInfo.Model)
			}
			if info.HostInfo.Type != "" && info.HostInfo.Type != "Unknown" {
				hostStr = fmt.Sprintf("%s (%s)", hostStr, info.HostInfo.Type)
			}
		}
		if hostStr != "" {
			infoLines = append(infoLines,
				fmt.Sprintf("%sHost:%s %s", keyColor, resetColor, hostStr))
		}
	}

	if isActive("bios") && info.BIOS != nil {
		biosStr := ""
		if info.BIOS.Version != "" {
			biosStr = info.BIOS.Version
			if info.BIOS.Type != "" {
				biosStr = fmt.Sprintf("%s (%s)", biosStr, info.BIOS.Type)
			}
		}
		if biosStr != "" {
			infoLines = append(infoLines,
				fmt.Sprintf("%sBIOS:%s %s", keyColor, resetColor, biosStr))
		}
	}

	if isActive("loginmanager") && info.LoginManager != "" && info.LoginManager != "Unknown" {
		infoLines = append(infoLines,
			fmt.Sprintf("%sLM:%s %s", keyColor, resetColor, info.LoginManager))
	}

	if isActive("processes") && info.Processes > 0 {
		infoLines = append(infoLines,
			fmt.Sprintf("%sProcesses:%s %d", keyColor, resetColor, info.Processes))
	}

	if isActive("cpuusage") && info.CPUUsage > 0 {
		usageColor := goodColor
		if info.CPUUsage >= 90 {
			usageColor = errorColor
		} else if info.CPUUsage >= 70 {
			usageColor = warnColor
		}
		infoLines = append(infoLines,
			fmt.Sprintf("%sCPU Usage:%s %s%.1f%%%s",
				keyColor, resetColor, usageColor, info.CPUUsage, resetColor))
	}

	if isActive("brightness") && info.Brightness != nil {
		infoLines = append(infoLines,
			fmt.Sprintf("%sBrightness:%s %d%%", keyColor, resetColor, info.Brightness.Current))
	}

	if isActive("wifi") && info.Wifi != nil {
		wifiStr := info.Wifi.SSID
		if info.Wifi.Protocol != "" && info.Wifi.Protocol != "Unknown" {
			wifiStr = fmt.Sprintf("%s - %s", wifiStr, info.Wifi.Protocol)
		}
		if info.Wifi.Frequency != "" {
			wifiStr = fmt.Sprintf("%s - %s", wifiStr, info.Wifi.Frequency)
		}
		if info.Wifi.Security != "" {
			wifiStr = fmt.Sprintf("%s - %s", wifiStr, info.Wifi.Security)
		}
		if info.Wifi.Strength > 0 {
			strengthColor := goodColor
			if info.Wifi.Strength < 40 {
				strengthColor = errorColor
			} else if info.Wifi.Strength < 60 {
				strengthColor = warnColor
			}
			wifiStr = fmt.Sprintf("%s %s(%d%%)%s", wifiStr, strengthColor, info.Wifi.Strength, resetColor)
		}
		infoLines = append(infoLines,
			fmt.Sprintf("%sWiFi:%s %s", keyColor, resetColor, wifiStr))
	}

	if isActive("publicip") && info.PublicIP != "" {
		infoLines = append(infoLines,
			fmt.Sprintf("%sPublic IP:%s %s", keyColor, resetColor, info.PublicIP))
	}

	if isActive("users") && len(info.Users) > 0 {
		for i, user := range info.Users {
			userStr := user.Name
			if user.Terminal != "" {
				userStr = fmt.Sprintf("%s@%s", userStr, user.Terminal)
			}
			if user.LoginTime != "" {
				userStr = fmt.Sprintf("%s - %s", userStr, user.LoginTime)
			}
			if i == 0 {
				infoLines = append(infoLines,
					fmt.Sprintf("%sUsers:%s %s", keyColor, resetColor, userStr))
			} else {
				infoLines = append(infoLines,
					fmt.Sprintf("       %s", userStr))
			}
		}
	}

	if isActive("datetime") && info.DateTime != "" {
		infoLines = append(infoLines,
			fmt.Sprintf("%sDate & Time:%s %s", keyColor, resetColor, info.DateTime))
	}

	maxLogoWidth := 0
	processedLogoLines := make([]string, len(asciiArtLines))
	plainLogoLines := make([]string, len(asciiArtLines))

	for i, line := range asciiArtLines {
		plainLine := line
		for k := range colorCodes {
			plainLine = strings.ReplaceAll(plainLine, k, "")
		}
		plainLine = stripANSICodes(plainLine)
		plainLogoLines[i] = plainLine

		lineWidth := len([]rune(plainLine))
		if lineWidth > maxLogoWidth {
			maxLogoWidth = lineWidth
		}

		processedLine := line
		hasColorPlaceholder := false
		for k, v := range colorCodes {
			if strings.Contains(processedLine, k) {
				hasColorPlaceholder = true
				processedLine = strings.ReplaceAll(processedLine, k, v)
			}
		}

		if !hasColorPlaceholder && len(processedLine) > 0 {
			processedLine = defaultColor + processedLine
		}

		if len(processedLine) > 0 {
			processedLine += resetColor
		}

		processedLogoLines[i] = processedLine
	}

	maxLines := max(len(processedLogoLines), len(infoLines))
	for i := 0; i < maxLines; i++ {
		var artLine string
		if i < len(processedLogoLines) {
			artLine = processedLogoLines[i]

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
		return "\033[39m"
	}
	if color == "bg" {
		return "\033[49m"
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

	return "\033[0m"
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
		_         = iota
		KB uint64 = 1 << (10 * iota)
		MB
		GB
		TB
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
