package display

import (
	"fmt"
	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/logo"
	"netfetch/internal/model"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	colorReset  = "\033[0m"
	colorKey    = "\033[96m"
	colorGood   = "\033[92m"
	colorWarn   = "\033[93m"
	colorError  = "\033[91m"
	colorAccent = "\033[95m"
)

func ShowColorized(c *collector.Collector, logos map[string]*logo.Logo, cfg *config.Config) error {
	info := c.GetInfo()
	if info == nil {
		return fmt.Errorf("failed to get system info")
	}

	var logoData *logo.Logo

	if info.OS != nil && info.OS.Distro != "" {
		distroLower := strings.ToLower(info.OS.Distro)
		if l, ok := logos[distroLower]; ok {
			logoData = l
		}
	}

	if logoData == nil && cfg.DefaultLogo != "" {
		if l, ok := logos[cfg.DefaultLogo]; ok {
			logoData = l
		}
	}

	if logoData == nil {
		for _, l := range logos {
			logoData = l
			break
		}
	}

	if logoData == nil {
		return fmt.Errorf("no logo available")
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
	colorCodes["${c}"] = colorReset

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
			return colorError
		case pct >= 70:
			return colorWarn
		default:
			return colorGood
		}
	}

	getCPUInfo := func() string {
		if info.CPU == nil {
			return "unknown"
		}
		cpuStr := fmt.Sprintf("%s (%d) @ %.2fGHz",
			info.CPU.Name,
			info.CPU.CoresLogical,
			float64(info.CPU.FrequencyMax)/1000)

		if info.CPU.Temperature > 0 && info.CPU.Temperature < 150 {
			temp := info.CPU.Temperature
			tempColor := colorGood
			if temp > 80 {
				tempColor = colorError
			} else if temp > 70 {
				tempColor = colorWarn
			}
			cpuStr += fmt.Sprintf(" - %s%.1f°C%s", tempColor, temp, colorReset)
		}

		return cpuStr
	}

	getMemoryInfo := func() string {
		if info.Memory == nil || info.Memory.Total == 0 {
			return "unknown"
		}

		usedPercent := (float64(info.Memory.Used) / float64(info.Memory.Total)) * 100
		memColor := colorUsage(usedPercent)

		return fmt.Sprintf("%s%s%s / %s %s(%d%%)%s",
			memColor,
			formatSize(info.Memory.Used),
			colorReset,
			formatSize(info.Memory.Total),
			memColor,
			int(usedPercent),
			colorReset)
	}

	getSwapInfo := func() string {
		if info.Swap == nil || info.Swap.Total == 0 {
			return "not configured"
		}

		usedPercent := (float64(info.Swap.Used) / float64(info.Swap.Total)) * 100
		swapColor := colorUsage(usedPercent)

		return fmt.Sprintf("%s%s%s / %s %s(%d%%)%s",
			swapColor,
			formatSize(info.Swap.Used),
			colorReset,
			formatSize(info.Swap.Total),
			swapColor,
			int(usedPercent),
			colorReset)
	}

	getBatteryInfo := func() string {
		if info.Battery == nil {
			return "unknown"
		}

		percent := info.Battery.Percentage
		battColor := colorGood
		if percent < 20 {
			battColor = colorError
		} else if percent < 50 {
			battColor = colorWarn
		}

		status := info.Battery.Status
		if status == "" {
			status = "Unknown"
		}

		return fmt.Sprintf("%s%.0f%%%s (%s)",
			battColor,
			percent,
			colorReset,
			status)
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
					colorKey, mp, colorReset,
					colorUsage(pct),
					formatSize(info.Disk.Used),
					colorReset,
					formatSize(info.Disk.Total),
					colorUsage(pct),
					int(pct),
					colorReset,
					fs)}
			}
			return []string{fmt.Sprintf("%sDisk:%s unknown", colorKey, colorReset)}
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
				colorKey,
				disk.Mountpoint,
				colorReset,
				colorUsage(pct),
				formatSize(disk.Used),
				colorReset,
				formatSize(disk.Total),
				colorUsage(pct),
				int(pct),
				colorReset,
				disk.FSType))
		}
		return lines
	}

	isActive := func(name string) bool {
		for _, m := range cfg.ActiveModules {
			if m == name {
				return true
			}
		}
		return false
	}

	infoLines := []string{
		fmt.Sprintf("%s%s@%s%s", colorKey, user, host, colorReset),
		"-------------",
	}

	if isActive("os") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sOS:%s %s", colorKey, colorReset, osInfo))
	}
	if isActive("kernel") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sKernel:%s %s", colorKey, colorReset, info.Kernel))
	}
	if isActive("uptime") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sUptime:%s %s", colorKey, colorReset, info.Uptime))
	}
	if isActive("packages") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sPackages:%s %s", colorKey, colorReset, info.Packages))
	}
	if isActive("shell") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sShell:%s %s", colorKey, colorReset, info.Shell))
	}
	if isActive("resolution") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sResolution:%s %s", colorKey, colorReset, info.Resolution))
	}
	if isActive("de") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sDE:%s %s", colorKey, colorReset, info.DE))
	}
	if isActive("wm") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sWM:%s %s", colorKey, colorReset, info.WM))
		if info.WMTheme != "Unknown" && info.WMTheme != "" {
			infoLines = append(infoLines,
				fmt.Sprintf("%sWM Theme:%s %s", colorKey, colorReset, info.WMTheme))
		}
	}
	if isActive("theme") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sTheme:%s %s", colorKey, colorReset, info.Theme))
	}
	if isActive("icons") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sIcons:%s %s", colorKey, colorReset, info.Icons))
	}
	if isActive("terminal") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sTerminal:%s %s", colorKey, colorReset, info.Terminal))
	}
	if isActive("cpu") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sCPU:%s %s", colorKey, colorReset, getCPUInfo()))
	}
	if isActive("gpu") {
		gpuStr := info.GPU
		if info.GPUTemp > 0 && info.GPUTemp < 150 {
			tempColor := colorGood
			if info.GPUTemp > 80 {
				tempColor = colorError
			} else if info.GPUTemp > 70 {
				tempColor = colorWarn
			}
			gpuStr += fmt.Sprintf(" - %s%.1f°C%s", tempColor, info.GPUTemp, colorReset)
		}
		infoLines = append(infoLines,
			fmt.Sprintf("%sGPU:%s %s", colorKey, colorReset, gpuStr))
	}
	if isActive("memory") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sMemory:%s %s", colorKey, colorReset, getMemoryInfo()))
	}

	if isActive("disk") {
		infoLines = append(infoLines, getDiskLines()...)
	}

	if isActive("swap") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sSwap:%s %s", colorKey, colorReset, getSwapInfo()))
	}
	if isActive("battery") {
		infoLines = append(infoLines,
			fmt.Sprintf("%sBattery:%s %s", colorKey, colorReset, getBatteryInfo()))
	}
	if isActive("locale") {
		locale := getValueOrDefault(info.Locale, "unknown")
		infoLines = append(infoLines,
			fmt.Sprintf("%sLocale:%s %s", colorKey, colorReset, locale))
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
				fmt.Sprintf("%sHost:%s %s", colorKey, colorReset, hostStr))
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
				fmt.Sprintf("%sBIOS:%s %s", colorKey, colorReset, biosStr))
		}
	}

	if isActive("loginmanager") && info.LoginManager != "" && info.LoginManager != "Unknown" {
		infoLines = append(infoLines,
			fmt.Sprintf("%sLM:%s %s", colorKey, colorReset, info.LoginManager))
	}

	if isActive("processes") && info.Processes > 0 {
		infoLines = append(infoLines,
			fmt.Sprintf("%sProcesses:%s %d", colorKey, colorReset, info.Processes))
	}

	if isActive("cpuusage") && info.CPUUsage > 0 {
		usageColor := colorGood
		if info.CPUUsage >= 90 {
			usageColor = colorError
		} else if info.CPUUsage >= 70 {
			usageColor = colorWarn
		}
		infoLines = append(infoLines,
			fmt.Sprintf("%sCPU Usage:%s %s%.1f%%%s",
				colorKey, colorReset, usageColor, info.CPUUsage, colorReset))
	}

	if isActive("brightness") && info.Brightness != nil {
		infoLines = append(infoLines,
			fmt.Sprintf("%sBrightness:%s %d%%", colorKey, colorReset, info.Brightness.Current))
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
			strengthColor := colorGood
			if info.Wifi.Strength < 40 {
				strengthColor = colorError
			} else if info.Wifi.Strength < 60 {
				strengthColor = colorWarn
			}
			wifiStr = fmt.Sprintf("%s %s(%d%%)%s", wifiStr, strengthColor, info.Wifi.Strength, colorReset)
		}
		infoLines = append(infoLines,
			fmt.Sprintf("%sWiFi:%s %s", colorKey, colorReset, wifiStr))
	}

	if isActive("publicip") && info.PublicIP != "" {
		infoLines = append(infoLines,
			fmt.Sprintf("%sPublic IP:%s %s", colorKey, colorReset, info.PublicIP))
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
					fmt.Sprintf("%sUsers:%s %s", colorKey, colorReset, userStr))
			} else {
				infoLines = append(infoLines,
					fmt.Sprintf("       %s", userStr))
			}
		}
	}

	if isActive("datetime") && info.DateTime != "" {
		infoLines = append(infoLines,
			fmt.Sprintf("%sDate & Time:%s %s", colorKey, colorReset, info.DateTime))
	}

	if len(infoLines) == 2 {
		infoLines = append(infoLines, "No active modules")
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
			processedLine += colorReset
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

		fmt.Printf("%s  %s\n", artLine, infoLine)
	}

	return nil
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
