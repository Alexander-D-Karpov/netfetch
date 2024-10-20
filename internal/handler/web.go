package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"netfetch/internal/model"
	"strings"
)

func parseColors(colors string) []string {
	colors = strings.ReplaceAll(colors, " ", "")
	colorList := strings.Split(colors, ",")
	parsedColors := make([]string, len(colorList))
	for i, color := range colorList {
		if !strings.HasPrefix(color, "#") {
			parsedColors[i] = "#" + color
		} else {
			parsedColors[i] = color
		}
	}
	return parsedColors
}

func (h *Handler) handleWeb(w http.ResponseWriter) {
	info := h.collector.GetInfo()
	logoData := h.logos[strings.ToLower(info.OS.Distro)]
	if logoData == nil {
		logoData = h.logos[h.config.DefaultLogo]
	}

	colors := parseColors(logoData.Colors)

	processedAsciiArt := make([]string, len(logoData.AsciiArt))
	for i, line := range logoData.AsciiArt {
		for j, color := range colors {
			line = strings.ReplaceAll(line, fmt.Sprintf("${c%d}", j+1), fmt.Sprintf("<span style=\"color: %s\">", color))
		}
		line = strings.ReplaceAll(line, "${c}", "</span>")
		processedAsciiArt[i] = line
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Neofetch-Net</title>
    <style>
        body {
            font-family: monospace;
            background-color: #000;
            color: #fff;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
        }
        .container {
            display: flex;
        }
        .logo {
            white-space: pre;
            margin-right: 20px;
            line-height: 1.2;
        }
        .info {
            display: flex;
            flex-direction: column;
            justify-content: center;
        }
        .info-line {
            display: flex;
        }
        .info-key {
            color: {{index .Colors 0}};
            margin-right: 10px;
        }
        .info-value {
            color: {{index .Colors 1}};
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">
{{range .Logo}}{{.}}
{{end}}
        </div>
        <div class="info">
            <div class="info-line"><span class="info-key">OS:</span><span class="info-value">{{.Info.OS.Distro}} {{.Info.OS.Arch}}</span></div>
            <div class="info-line"><span class="info-key">Host:</span><span class="info-value">{{.Info.Host}}</span></div>
            <div class="info-line"><span class="info-key">Kernel:</span><span class="info-value">{{.Info.Kernel}}</span></div>
            <div class="info-line"><span class="info-key">Uptime:</span><span class="info-value">{{.Info.Uptime}}</span></div>
            <div class="info-line"><span class="info-key">Packages:</span><span class="info-value">{{.Info.Packages}}</span></div>
            <div class="info-line"><span class="info-key">Shell:</span><span class="info-value">{{.Info.Shell}}</span></div>
            <div class="info-line"><span class="info-key">Resolution:</span><span class="info-value">{{.Info.Resolution}}</span></div>
            <div class="info-line"><span class="info-key">DE:</span><span class="info-value">{{.Info.DE}}</span></div>
            <div class="info-line"><span class="info-key">WM:</span><span class="info-value">{{.Info.WM}}</span></div>
            <div class="info-line"><span class="info-key">WM Theme:</span><span class="info-value">{{.Info.WMTheme}}</span></div>
            <div class="info-line"><span class="info-key">Theme:</span><span class="info-value">{{.Info.Theme}}</span></div>
            <div class="info-line"><span class="info-key">Icons:</span><span class="info-value">{{.Info.Icons}}</span></div>
            <div class="info-line"><span class="info-key">Terminal:</span><span class="info-value">{{.Info.Terminal}}</span></div>
            <div class="info-line"><span class="info-key">CPU:</span><span class="info-value">{{.Info.CPU.Model}} ({{.Info.CPU.Cores}} cores)</span></div>
            <div class="info-line"><span class="info-key">GPU:</span><span class="info-value">{{.Info.GPU}}</span></div>
            <div class="info-line"><span class="info-key">Memory:</span><span class="info-value">{{.Info.Memory.Used | formatMiB}} / {{.Info.Memory.Total | formatMiB}}</span></div>
            <div class="info-line"><span class="info-key">Disk (/):</span><span class="info-value">{{.Info.Disk.Used | formatGB}} / {{.Info.Disk.Total | formatGB}} ({{.Info.Disk.UsedPercent | printf "%.0f"}}%)</span></div>
        </div>
    </div>
</body>
</html>
`

	funcMap := template.FuncMap{
		"formatMiB": func(b uint64) string {
			return fmt.Sprintf("%.0fMiB", float64(b)/1024/1024)
		},
		"formatGB": func(b uint64) string {
			return fmt.Sprintf("%.0fG", float64(b)/1024/1024/1024)
		},
	}

	t, err := template.New("neofetch").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Info   *model.SystemInfo
		Logo   []string
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
