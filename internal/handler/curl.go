package handler

import (
	"fmt"
	"net/http"
	"strings"
)

func (h *Handler) handleCurl(w http.ResponseWriter) {
	info := h.collector.GetInfo()
	logoData := h.logos[strings.ToLower(info.OS.Distro)]
	if logoData == nil {
		logoData = h.logos[h.config.DefaultLogo]
	}

	colorCodes := map[string]string{
		"${c1}": "\033[38;5;6m",
		"${c2}": "\033[38;5;7m",
		"${c}":  "\033[0m",
	}

	var response strings.Builder

	for _, line := range logoData.AsciiArt {
		for k, v := range colorCodes {
			line = strings.ReplaceAll(line, k, v)
		}
		response.WriteString(line + "\n")
	}

	response.WriteString("\033[0m") // Reset color
	response.WriteString(fmt.Sprintf("OS: %s %s\n", info.OS.Distro, info.OS.Arch))
	response.WriteString(fmt.Sprintf("Host: %s\n", info.Host))
	response.WriteString(fmt.Sprintf("Kernel: %s\n", info.Kernel))
	response.WriteString(fmt.Sprintf("Uptime: %s\n", info.Uptime))
	response.WriteString(fmt.Sprintf("Packages: %s\n", info.Packages))
	response.WriteString(fmt.Sprintf("Shell: %s\n", info.Shell))
	response.WriteString(fmt.Sprintf("Resolution: %s\n", info.Resolution))
	response.WriteString(fmt.Sprintf("DE: %s\n", info.DE))
	response.WriteString(fmt.Sprintf("WM: %s\n", info.WM))
	response.WriteString(fmt.Sprintf("WM Theme: %s\n", info.WMTheme))
	response.WriteString(fmt.Sprintf("Theme: %s\n", info.Theme))
	response.WriteString(fmt.Sprintf("Icons: %s\n", info.Icons))
	response.WriteString(fmt.Sprintf("Terminal: %s\n", info.Terminal))
	response.WriteString(fmt.Sprintf("CPU: %s (%d) @ %.2fGHz\n", info.CPU.Model, info.CPU.Cores, info.CPU.Frequency/1000))
	response.WriteString(fmt.Sprintf("GPU: %s\n", info.GPU))
	response.WriteString(fmt.Sprintf("Memory: %dMiB / %dMiB\n", info.Memory.Used/1024/1024, info.Memory.Total/1024/1024))
	response.WriteString(fmt.Sprintf("Disk (/): %dG / %dG (%d%%)\n", info.Disk.Used/1024/1024/1024, info.Disk.Total/1024/1024/1024, int(info.Disk.UsedPercent)))

	w.Header().Set("Content-Type", "text/plain")
	_, err := fmt.Fprint(w, response.String())
	if err != nil {
		return
	}
}
