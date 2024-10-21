package handler

import (
	"fmt"
	"html/template"
	"net/http"
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
	colorMap := map[string]string{
		"fg": "#FFFFFF", // Default foreground color
		"bg": "#000000", // Default background color
	}
	if hex, ok := colorMap[color]; ok {
		return hex
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
	logoData := h.getLogo(info.OS.Distro)
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
		// Close any unclosed <span> tags
		if strings.Count(line, "<span") > strings.Count(line, "</span>") {
			line += "</span>"
		}
		// Mark the line as safe HTML to prevent escaping
		processedAsciiArt[i] = template.HTML(line)
	}

	funcMap := template.FuncMap{
		"formatMiB": func(b uint64) string {
			return fmt.Sprintf("%.0fMiB", float64(b)/1024/1024)
		},
		"formatGB": func(b uint64) string {
			return fmt.Sprintf("%.0fG", float64(b)/1024/1024/1024)
		},
	}

	// Load the template from file
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
