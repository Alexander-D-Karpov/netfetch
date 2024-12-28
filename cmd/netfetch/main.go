package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/daemon"
	"netfetch/internal/display"
	"netfetch/internal/logo"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Parse command line arguments
	args := os.Args[1:]
	if len(args) == 0 {
		// Default behavior: display in console
		showLocalInfo()
		return
	}

	// Handle remote host connection (no flag needed)
	if !strings.HasPrefix(args[0], "-") {
		if err := connectToRemote(args[0]); err != nil {
			log.Fatalf("Error: %v", err)
		}
		return
	}

	// Parse daemon commands
	if args[0] == "-d" {
		if len(args) < 2 {
			log.Fatal("Missing daemon command. Usage: netfetch -d [start|stop|status]")
		}

		cmd := flag.NewFlagSet("daemon", flag.ExitOnError)
		port := cmd.Int("port", 22828, "Port for daemon communication")
		webPort := cmd.Int("web-port", 22828, "Web server port")
		configFile := cmd.String("config", "config.yaml", "Path to config file")

		// Parse remaining arguments after the daemon command
		if err := cmd.Parse(args[2:]); err != nil {
			log.Fatal(err)
		}

		// Get executable path for daemon
		exePath, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		absExePath, err := filepath.Abs(exePath)
		if err != nil {
			log.Fatal(err)
		}

		// Initialize and run daemon command
		d := daemon.New(absExePath, *configFile, *port, *webPort)

		switch args[1] {
		case "start":
			if err := d.Start(); err != nil {
				log.Fatalf("Error starting daemon: %v", err)
			}
		case "stop":
			if err := d.Stop(); err != nil {
				log.Fatalf("Error stopping daemon: %v", err)
			}
		case "status":
			if err := d.Status(); err != nil {
				log.Fatalf("Error checking daemon status: %v", err)
			}
		case "install":
			if err := d.InstallSystemd(); err != nil {
				log.Fatalf("Error installing systemd service: %v", err)
			}
		case "uninstall":
			if err := d.UninstallSystemd(); err != nil {
				log.Fatalf("Error uninstalling systemd service: %v", err)
			}
		default:
			log.Fatalf("Unknown daemon command: %s", args[1])
		}
		return
	}

	// If we got here, show help
	showHelp()
}

func showLocalInfo() {
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

	// Initialize collector
	c := collector.New(cfg.ActiveModules)

	// Display info
	if err := display.ShowColorized(c, logos, cfg); err != nil {
		log.Fatalf("Error displaying info: %v", err)
	}
}

func connectToRemote(host string) error {
	// Add default port if not specified
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:22828", host)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make request
	resp, err := client.Get(fmt.Sprintf("http://%s", host))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", host, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("Error closing response body: %v", err)
		}
	}(resp.Body)

	// Copy response to stdout
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

func showHelp() {
	fmt.Printf(`Usage: netfetch [OPTIONS] [HOST]

Display system information in a fancy way.

Options:
  -d start [--port PORT] [--web-port PORT] [--config FILE]  Start daemon
  -d stop                                                   Stop daemon
  -d status                                                Show daemon status
  -d install                                               Install systemd service
  -d uninstall                                             Uninstall systemd service

Examples:
  netfetch                            Show local system info
  netfetch example.com                Connect to remote host
  netfetch -d start --web-port 8080   Start daemon with custom web port
`)
}
