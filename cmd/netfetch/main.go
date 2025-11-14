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

	"netfetch/assets"
	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/display"
	"netfetch/internal/handler"
	"netfetch/internal/logo"
)

const (
	defaultPort    = 22828
	defaultLogoDir = "logos"
)

type Mode int

const (
	ModeServe Mode = iota
	ModeShow
	ModeConnect
	ModeHelp
)

func init() {
	logo.EmbeddedLogos = assets.LogosFS
}

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
		showAll    bool
	)

	flagSet := flag.NewFlagSet("netfetch", flag.ExitOnError)
	flagSet.IntVar(&port, "port", 0, "Port for server/client")
	flagSet.StringVar(&configFile, "config", "", "Path to config file")
	flagSet.StringVar(&logoDir, "logo-dir", "", "Directory containing logo files")
	flagSet.IntVar(&timeout, "timeout", 5, "Connection timeout in seconds")
	flagSet.BoolVar(&showAll, "all", false, "Show all modules")

	mode, host, args := parseArgs(os.Args[1:])

	flagSet.Parse(args)

	var modules []string
	if mode == ModeShow {
		modules = flagSet.Args()
	}

	switch mode {
	case ModeServe:
		runServe(port, configFile, logoDir)
	case ModeShow:
		runShow(configFile, logoDir, showAll, modules)
	case ModeConnect:
		runConnect(host, port, timeout)
	case ModeHelp:
		printHelp()
	}
}

func withBaseModules(mods []string) []string {
	base := []string{"os"}
	out := make([]string, 0, len(mods)+len(base))
	out = append(out, mods...)

	for _, b := range base {
		found := false
		for _, m := range mods {
			if m == b {
				found = true
				break
			}
		}
		if !found {
			out = append(out, b)
		}
	}

	return out
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

func runShow(configFile, logoDir string, showAll bool, modules []string) {
	cfg := loadConfig(configFile, logoDir, 0)

	if showAll {
		cfg.ActiveModules = config.GetDefaultModules()
	} else if len(modules) > 0 {
		cfg.ActiveModules = modules
	}

	logos, err := logo.LoadAll(cfg.LogoDir)
	if err != nil {
		log.Fatalf("Failed to load logos: %v", err)
	}

	collectorModules := withBaseModules(cfg.ActiveModules)

	c := collector.New(collectorModules)
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

	collectorModules := withBaseModules(cfg.ActiveModules)

	c := collector.New(collectorModules)
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
	var cfg *config.Config

	if configFile == "" {
		configFile = "config.yaml"
	}

	loadedCfg, err := config.Load(configFile)
	if err != nil {
		log.Printf("Config file not found, using defaults")
		cfg = getDefaultConfig()
	} else {
		cfg = loadedCfg
	}

	if logoDir != "" {
		cfg.LogoDir = logoDir
	}

	if port > 0 {
		cfg.ListenAddress = fmt.Sprintf(":%d", port)
	} else if cfg.ListenAddress == "" {
		cfg.ListenAddress = fmt.Sprintf(":%d", defaultPort)
	}

	if cfg.LogoDir == "" {
		cfg.LogoDir = ""
	}

	return cfg
}

func getDefaultConfig() *config.Config {
	return &config.Config{
		ListenAddress: fmt.Sprintf(":%d", defaultPort),
		ActiveModules: []string{
			"os", "kernel", "uptime", "packages", "shell", "resolution",
			"de", "wm", "theme", "icons", "terminal", "cpu", "gpu",
			"memory", "disk", "swap", "battery", "locale",
		},
		DefaultLogo: "linux",
		LogoDir:     "",
	}
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
        netfetch show [OPTIONS] [MODULE ...]

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
        Directory containing logo files (uses embedded logos by default)

    -timeout int
        Connection timeout in seconds (default: 5)

    -all
        Show all modules (ignore active_modules from config)

    -h, -help, help
        Show this help message

EXAMPLES:
    Start server (default):
        netfetch
        netfetch serve

    Start server on custom port:
        netfetch -port 8080
        netfetch serve -port 8080

    Show local system info (active modules from config):
        netfetch show

    Show only specific modules:
        netfetch show cpu gpu

    Show all modules:
        netfetch show -all

    Connect to remote server:
        netfetch example.com
        netfetch connect example.com
        netfetch example.com:8080 -timeout 10

    Use custom config:
        netfetch -config /path/to/config.yaml
        netfetch show -config custom.yaml -logo-dir /path/to/logos`)
}
