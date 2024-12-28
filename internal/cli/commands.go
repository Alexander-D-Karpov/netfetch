package cli

import (
	"fmt"
	"io"
	"net"
	"netfetch/internal/daemon"
	"os"
	"strings"
	"time"

	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/display"
	"netfetch/internal/logo"
)

type ConsoleCommand struct {
	opts *Options
}

func (c *ConsoleCommand) Execute() error {
	cfg, err := config.Load(c.opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	logos, err := logo.LoadAll(cfg.LogoDir)
	if err != nil {
		return fmt.Errorf("failed to load logos: %v", err)
	}

	col := collector.New(cfg.ActiveModules)
	return display.ShowColorized(col, logos, cfg)
}

type RemoteCommand struct {
	host    string
	timeout int
	opts    *Options
}

func (c *RemoteCommand) Execute() error {
	host := c.host
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, c.opts.DefaultPort)
	}

	// Set connection timeout
	dialer := net.Dialer{Timeout: time.Duration(c.timeout) * time.Second}
	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "Error closing connection: %v\n", err)
			if err != nil {
				return
			}
		}
	}(conn)

	_, err = io.Copy(os.Stdout, conn)
	return err
}

type DaemonCommand struct {
	action     string
	port       int
	webPort    int
	configPath string
	opts       *Options
}

func (c *DaemonCommand) Execute() error {
	d := daemon.New(c.opts.ExePath, c.configPath, c.port, c.webPort)

	switch c.action {
	case "start":
		return d.Start()
	case "stop":
		return d.Stop()
	case "status":
		return d.Status()
	case "install":
		return d.InstallSystemd()
	case "uninstall":
		return d.UninstallSystemd()
	default:
		return fmt.Errorf("unknown daemon command: %s", c.action)
	}
}

type HelpCommand struct {
	opts *Options
}

func (c *HelpCommand) Execute() error {
	fmt.Printf(`Usage: netfetch [OPTIONS] [HOST]

Display system information in a fancy way.

Options:
  -d start [--port PORT] [--web-port PORT] [--config FILE]  Start daemon
  -d stop                                                   Stop daemon
  -d status                                                 Show daemon status
  -d install                                               Install systemd service
  -d uninstall                                             Uninstall systemd service

Examples:
  netfetch                            Show local system info
  netfetch example.com                Connect to remote host
  netfetch -d start --web-port 8080   Start daemon with custom web port
`)
	return nil
}
