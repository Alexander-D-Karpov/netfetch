package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/display"
	"netfetch/internal/handler"
	"netfetch/internal/logo"
)

const (
	defaultPort      = 22828
	defaultConfigDir = "."
	defaultLogoDir   = "logos"
)

type Mode int

const (
	ModeServe Mode = iota
	ModeShow
	ModeConnect
	ModeHelp
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help") {
		printHelp()
		return
	}

	var (
		port       int
		configFile string
		logoDir    string
		timeout    int
	)

	flagSet := flag.NewFlagSet("netfetch", flag.ExitOnError)
	flagSet.IntVar(&port, "port", 0, "Port for server/client")
	flagSet.StringVar(&configFile, "config", "", "Path to config file")
	flagSet.StringVar(&logoDir, "logo-dir", "", "Directory containing logo files")
	flagSet.IntVar(&timeout, "timeout", 5, "Connection timeout in seconds")

	mode, host, args := parseArgs(os.Args[1:])

	flagSet.Parse(args)

	switch mode {
	case ModeServe:
		runServe(port, configFile, logoDir)
	case ModeShow:
		runShow(configFile, logoDir)
	case ModeConnect:
		runConnect(host, port, timeout)
	case ModeHelp:
		printHelp()
	}
}

func parseArgs(args []string) (Mode, string, []string) {
	if len(args) == 0 {
		return ModeServe, "", args
	}

	firstArg := args[0]

	if firstArg == "serve" {
		return ModeServe, "", args[1:]
	}

	if firstArg == "show" {
		return ModeShow, "", args[1:]
	}

	if firstArg == "connect" {
		if len(args) > 1 {
			return ModeConnect, args[1], args[2:]
		}
		log.Fatal("connect mode requires a host argument")
	}

	if !isFlag(firstArg) {
		return ModeConnect, firstArg, args[1:]
	}

	return ModeServe, "", args
}

func isFlag(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}

func runShow(configFile, logoDir string) {
	cfg := loadConfig(configFile, logoDir, 0)

	logos, err := logo.LoadAll(cfg.LogoDir)
	if err != nil {
		log.Fatalf("Failed to load logos: %v", err)
	}

	c := collector.New(cfg.ActiveModules)
	c.CollectDynamicInfo()

	if err := display.ShowColorized(c, logos, cfg); err != nil {
		log.Fatalf("Error displaying info: %v", err)
	}
}

func runServe(port int, configFile, logoDir string) {
	cfg := loadConfig(configFile, logoDir, port)

	logos, err := logo.LoadAll(cfg.LogoDir)
	if err != nil {
		log.Fatalf("Failed to load logos: %v", err)
	}
	log.Printf("Loaded %d logos", len(logos))

	c := collector.New(cfg.ActiveModules)
	h := handler.New(c, logos, cfg)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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

	<-sigChan
	log.Println("Shutting down server...")
	if err := server.Close(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

func runConnect(host string, port, timeout int) {
	if host == "" {
		log.Fatal("No host specified for connect mode")
	}

	if port == 0 {
		port = defaultPort
	}

	fullHost := host
	if !containsPort(host) {
		fullHost = fmt.Sprintf("%s:%d", host, port)
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("http://%s", fullHost))
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", fullHost, err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}
}

func loadConfig(configFile, logoDir string, port int) *config.Config {
	if configFile == "" {
		configFile = "config.yaml"
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		log.Printf("Warning: Could not load config file '%s', using defaults: %v", configFile, err)
		cfg = &config.Config{
			ListenAddress: fmt.Sprintf(":%d", defaultPort),
			ActiveModules: []string{
				"os", "kernel", "uptime", "packages", "shell", "resolution",
				"de", "wm", "theme", "icons", "terminal", "cpu", "gpu",
				"memory", "disk", "swap", "battery", "locale",
			},
			DefaultLogo: "arch",
			LogoDir:     defaultLogoDir,
		}
	}

	if logoDir != "" {
		cfg.LogoDir = logoDir
	} else if cfg.LogoDir == "" {
		cfg.LogoDir = defaultLogoDir
	}

	if port > 0 {
		cfg.ListenAddress = fmt.Sprintf(":%d", port)
	} else if cfg.ListenAddress == "" {
		cfg.ListenAddress = fmt.Sprintf(":%d", defaultPort)
	}

	return cfg
}

func containsPort(host string) bool {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return true
		}
		if host[i] == ']' {
			return false
		}
	}
	return false
}

func printHelp() {
	fmt.Println(`netfetch - Display system information

USAGE:
    netfetch [MODE] [OPTIONS] [HOST]

MODES:
    (default)
        Start HTTP server (default mode when no arguments)

    serve
        Start HTTP server to serve system information
        netfetch serve [OPTIONS]

    show
        Display local system information (dry run)
        netfetch show [OPTIONS]

    connect <host>
        Connect to a remote netfetch server
        netfetch connect <host> [OPTIONS]
        netfetch <host> [OPTIONS]

OPTIONS:
    -port int
        Port number for server/client (default: 22828)

    -config string
        Path to config file (default: config.yaml)

    -logo-dir string
        Directory containing logo files (default: logos)

    -timeout int
        Connection timeout in seconds (default: 5)

    -h, -help, help
        Show this help message

EXAMPLES:
    Start server (default):
        netfetch
        netfetch serve

    Start server on custom port:
        netfetch -port 8080
        netfetch serve -port 8080

    Show local system info:
        netfetch show

    Connect to remote server:
        netfetch example.com
        netfetch connect example.com
        netfetch example.com:8080 -timeout 10

    Use custom config:
        netfetch -config /path/to/config.yaml
        netfetch show -config custom.yaml -logo-dir /path/to/logos`)
}
