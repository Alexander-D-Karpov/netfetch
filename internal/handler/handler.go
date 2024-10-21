package handler

import (
	"net/http"
	"strings"

	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/logo"
)

type Handler struct {
	collector *collector.Collector
	logos     map[string]*logo.Logo
	config    *config.Config
}

func New(c *collector.Collector, l map[string]*logo.Logo, cfg *config.Config) *Handler {
	return &Handler{
		collector: c,
		logos:     l,
		config:    cfg,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("User-Agent"), "curl") {
		h.handleCurl(w)
	} else {
		h.handleWeb(w)
	}
}

func (h *Handler) getLogo(distro string) *logo.Logo {
	distro = strings.ToLower(distro)
	if logoData, ok := h.logos[distro]; ok {
		return logoData
	}
	// Try to match by substring
	for key, logoData := range h.logos {
		if strings.Contains(distro, strings.ToLower(key)) {
			return logoData
		}
	}
	// Use default logo
	return h.logos[h.config.DefaultLogo]
}
