package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/handler"
	"netfetch/internal/logo"
)

func main() {
	// Parse command line flags
	webPort := flag.Int("web-port", 22828, "Web server port")
	configFile := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override config with command line flags
	if *webPort != 22828 {
		cfg.ListenAddress = fmt.Sprintf(":%d", *webPort)
	}

	// Load logos
	logos, err := logo.LoadAll(cfg.LogoDir)
	if err != nil {
		log.Fatalf("Failed to load logos: %v", err)
	}
	log.Printf("Loaded %d logos", len(logos))

	// Initialize collector and handler
	c := collector.New(cfg.ActiveModules)
	h := handler.New(c, logos, cfg)

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server
	server := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: h,
	}

	go func() {
		log.Printf("Starting server on %s", cfg.ListenAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down server...")
	if err := server.Close(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
