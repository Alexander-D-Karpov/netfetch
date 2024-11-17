package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"netfetch/internal/logo"
	"netfetch/internal/model"
	"path/filepath"
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
		return "#FFFFFF" // Default foreground color
	}
	if color == "bg" {
		return "#000000" // Default background color
	}
	ansiColorNum, err := strconv.Atoi(color)
	if err == nil && ansiColorNum >= 0 && ansiColorNum < len(ansiColors) {
		return ansiColors[ansiColorNum]
	}
	// Default to white if parsing fails
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
		// Replace color placeholders with HTML spans
		for j := range colors {
			placeholder := fmt.Sprintf("${c%d}", j+1)
			line = strings.ReplaceAll(line, placeholder, fmt.Sprintf("<span style=\"color: %s\">", colors[j]))
		}
		line = strings.ReplaceAll(line, "${c}", "</span>")
		// Close any unclosed spans
		if strings.Count(line, "<span") > strings.Count(line, "</span>") {
			line += "</span>"
		}
		processedAsciiArt[i] = template.HTML(line)
	}

	funcMap := template.FuncMap{
		"formatMiB": func(b uint64) string {
			return fmt.Sprintf("%.0fMiB", float64(b)/1024/1024)
		},
		"formatGB": func(b uint64) string {
			return fmt.Sprintf("%.0fG", float64(b)/1024/1024/1024)
		},
		"join": strings.Join,
		"formatFreq": func(freq uint32) string {
			return fmt.Sprintf("%.2f GHz", float64(freq)/1000)
		},
	}

	tmplPath := filepath.Join("templates", "neofetch.html")
	t, err := template.New("neofetch.html").Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Info   *model.SystemInfo
		Logo   []template.HTML
		Colors []string
	}{
		Info:   info,
		Logo:   processedAsciiArt,
		Colors: colors,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
