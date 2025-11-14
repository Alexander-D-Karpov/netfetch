package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"netfetch/assets"
	"netfetch/internal/logo"
	"netfetch/internal/model"
	"sort"
	"strconv"
	"strings"
)

var ansiColors = []string{
	"#000000", "#800000", "#008000", "#808000", "#000080", "#800080", "#008080", "#c0c0c0",
	"#808080", "#ff0000", "#00ff00", "#ffff00", "#0000ff", "#ff00ff", "#00ffff", "#ffffff",
}

func parseColors(colors string) []string {
	colorList := strings.Fields(colors)
	parsedColors := make([]string, len(colorList))
	for i, color := range colorList {
		parsedColors[i] = mapColorToHex(color)
	}
	return parsedColors
}

func mapColorToHex(color string) string {
	if color == "fg" {
		return "#FFFFFF"
	}
	if color == "bg" {
		return "#000000"
	}
	ansiColorNum, err := strconv.Atoi(color)
	if err == nil && ansiColorNum >= 0 && ansiColorNum < len(ansiColors) {
		return ansiColors[ansiColorNum]
	}
	return "#FFFFFF"
}

func (h *Handler) handleWeb(w http.ResponseWriter) {
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
		http.Error(w, "Logo not found", http.StatusInternalServerError)
		return
	}

	colors := parseColors(logoData.Colors)

	processedAsciiArt := make([]template.HTML, len(logoData.AsciiArt))
	for i, line := range logoData.AsciiArt {
		for j := range colors {
			placeholder := fmt.Sprintf("${c%d}", j+1)
			line = strings.ReplaceAll(line, placeholder, fmt.Sprintf("<span style=\"color: %s\">", colors[j]))
		}
		line = strings.ReplaceAll(line, "${c}", "</span>")
		if strings.Count(line, "<span") > strings.Count(line, "</span>") {
			line += "</span>"
		}
		processedAsciiArt[i] = template.HTML(line)
	}

	funcMap := template.FuncMap{
		"join": strings.Join,
		"formatFreq": func(freq uint32) string {
			return fmt.Sprintf("%.2f GHz", float64(freq)/1000)
		},
		"isActive": func(name string) bool {
			for _, m := range h.config.ActiveModules {
				if m == name {
					return true
				}
			}
			return false
		},
		"formatDiskSize": func(b uint64) string {
			const (
				_  = iota
				KB = 1 << (10 * iota)
				MB
				GB
				TB
			)
			switch {
			case b >= TB:
				return fmt.Sprintf("%.2f TiB", float64(b)/float64(TB))
			case b >= GB:
				return fmt.Sprintf("%.2f GiB", float64(b)/float64(GB))
			case b >= MB:
				return fmt.Sprintf("%.2f MiB", float64(b)/float64(MB))
			case b >= KB:
				return fmt.Sprintf("%.2f KiB", float64(b)/float64(KB))
			default:
				return fmt.Sprintf("%d B", b)
			}
		},
		"sortDisks": func(disks []model.DiskInfo) []model.DiskInfo {
			sorted := make([]model.DiskInfo, len(disks))
			copy(sorted, disks)
			sort.SliceStable(sorted, func(i, j int) bool {
				if sorted[i].Mountpoint == "/" {
					return true
				}
				if sorted[j].Mountpoint == "/" {
					return false
				}
				return sorted[i].Mountpoint < sorted[j].Mountpoint
			})
			return sorted
		},
		"diskColorClass": func(disk *model.DiskInfo) string {
			if disk == nil {
				return ""
			}
			pct := disk.UsedPercent
			if pct >= 90 {
				return "color-bad"
			} else if pct >= 70 {
				return "color-warn"
			}
			return "color-good"
		},
		"memoryColorClass": func(mem *model.MemoryInfo) string {
			if mem == nil || mem.Total == 0 {
				return ""
			}
			pct := (float64(mem.Used) / float64(mem.Total)) * 100
			if pct >= 90 {
				return "color-bad"
			} else if pct >= 70 {
				return "color-warn"
			}
			return "color-good"
		},
		"memoryPercent": func(mem *model.MemoryInfo) int {
			if mem == nil || mem.Total == 0 {
				return 0
			}
			return int((float64(mem.Used) / float64(mem.Total)) * 100)
		},
		"swapColorClass": func(swap *model.SwapInfo) string {
			if swap == nil || swap.Total == 0 {
				return ""
			}
			pct := (float64(swap.Used) / float64(swap.Total)) * 100
			if pct >= 90 {
				return "color-bad"
			} else if pct >= 70 {
				return "color-warn"
			}
			return "color-good"
		},
		"swapPercent": func(swap *model.SwapInfo) int {
			if swap == nil || swap.Total == 0 {
				return 0
			}
			return int((float64(swap.Used) / float64(swap.Total)) * 100)
		},
		"batteryColorClass": func(battery *model.BatteryInfo) string {
			if battery == nil {
				return ""
			}
			pct := battery.Percentage
			if pct <= 20 {
				return "color-bad"
			} else if pct <= 50 {
				return "color-warn"
			}
			return "color-good"
		},
		"printf": fmt.Sprintf,
		"hostInfoStr": func(host *model.HostInfo) string {
			if host == nil || host.Model == "" {
				return ""
			}
			hostStr := host.Model
			if host.Vendor != "" && host.Vendor != host.Model {
				hostStr = fmt.Sprintf("%s %s", host.Vendor, host.Model)
			}
			if host.Type != "" && host.Type != "Unknown" {
				hostStr = fmt.Sprintf("%s (%s)", hostStr, host.Type)
			}
			return hostStr
		},
		"biosStr": func(bios *model.BIOSInfo) string {
			if bios == nil || bios.Version == "" {
				return ""
			}
			biosStr := bios.Version
			if bios.Type != "" {
				biosStr = fmt.Sprintf("%s (%s)", biosStr, bios.Type)
			}
			return biosStr
		},
		"wifiStr": func(wifi *model.WifiInfo) string {
			if wifi == nil || wifi.SSID == "" {
				return ""
			}
			wifiStr := wifi.SSID
			if wifi.Protocol != "" && wifi.Protocol != "Unknown" {
				wifiStr = fmt.Sprintf("%s - %s", wifiStr, wifi.Protocol)
			}
			if wifi.Frequency != "" {
				wifiStr = fmt.Sprintf("%s - %s", wifiStr, wifi.Frequency)
			}
			if wifi.Security != "" {
				wifiStr = fmt.Sprintf("%s - %s", wifiStr, wifi.Security)
			}
			return wifiStr
		},
		"wifiStrengthClass": func(wifi *model.WifiInfo) string {
			if wifi == nil {
				return ""
			}
			if wifi.Strength < 40 {
				return "color-bad"
			} else if wifi.Strength < 60 {
				return "color-warn"
			}
			return "color-good"
		},
		"cpuUsageClass": func(usage float64) string {
			if usage >= 90 {
				return "color-bad"
			} else if usage >= 70 {
				return "color-warn"
			}
			return "color-good"
		},
	}

	tmplContent, err := assets.TemplatesFS.ReadFile("templates/neofetch.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read template: %v", err), http.StatusInternalServerError)
		return
	}

	t, err := template.New("neofetch.html").Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Info   *model.SystemInfo
		Logo   []template.HTML
		Colors []string
		Config interface{}
	}{
		Info:   info,
		Logo:   processedAsciiArt,
		Colors: colors,
		Config: h.config,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
