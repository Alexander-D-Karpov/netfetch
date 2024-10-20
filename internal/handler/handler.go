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
	distro    string
}

func New(c *collector.Collector, l map[string]*logo.Logo, cfg *config.Config) *Handler {
	info := c.GetInfo()
	distro := strings.ToLower(info.OS.Distro)

	log.Printf("Detected distro: %s", distro)

	return &Handler{
		collector: c,
		logos:     l,
		config:    cfg,
		distro:    distro,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("User-Agent"), "curl") {
		h.handleCurl(w)
	} else {
		h.handleWeb(w)
	}
}
