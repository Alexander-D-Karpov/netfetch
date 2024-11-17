package handler

import (
	"log"
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
	h.collector.CollectDynamicInfo()
	if strings.Contains(r.Header.Get("User-Agent"), "curl") {
		h.handleCurl(w)
	} else {
		h.handleWeb(w)
	}
}

func (h *Handler) getLogo(distro string) *logo.Logo {
	// Convert distro name to lowercase for case-insensitive matching
	distroLower := strings.ToLower(distro)

	// Try to find the logo for the specific distro
	if logo, ok := h.logos[distroLower]; ok {
		return logo
	}

	// If not found, try default logo
	if logo, ok := h.logos[h.config.DefaultLogo]; ok {
		return logo
	}

	// If default logo is not found, log a warning and return nil
	log.Printf("Warning: Default logo '%s' not found. Proceeding without a logo.", h.config.DefaultLogo)
	return nil
}
