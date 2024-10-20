package main

import (
	"log"
	"net/http"

	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/handler"
	"netfetch/internal/logo"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Load logos
	logos, err := logo.LoadAll(cfg.LogoDir)
	if err != nil {
		log.Fatalf("Failed to load logos: %v", err)
	}
	log.Printf("Loaded %d logos", len(logos))

	// Initialize collector
	c := collector.New(cfg.ActiveModules)

	// Initialize handler
	h := handler.New(c, logos, cfg)

	// Set up HTTP server
	http.HandleFunc("/", h.ServeHTTP)

	log.Printf("Starting server on %s", cfg.ListenAddress)
	log.Fatal(http.ListenAndServe(cfg.ListenAddress, nil))
}
