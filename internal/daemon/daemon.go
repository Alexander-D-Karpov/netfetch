package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type Daemon struct {
	exePath     string
	configPath  string
	port        int
	webPort     int
	pidFile     string
	systemdUnit string
}

func New(exePath, configPath string, port, webPort int) *Daemon {
	return &Daemon{
		exePath:     exePath,
		configPath:  configPath,
		port:        port,
		webPort:     webPort,
		pidFile:     filepath.Join(os.TempDir(), "netfetch.pid"),
		systemdUnit: "netfetch.service",
	}
}

func (d *Daemon) Start() error {
	if d.IsRunning() {
		return fmt.Errorf("daemon is already running")
	}

	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	// Create .netfetch directory if it doesn't exist
	logDir := filepath.Join(homeDir, ".netfetch")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Open log file in user's directory
	logFile, err := os.OpenFile(
		filepath.Join(logDir, "netfetch.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// Get server binary path (in the same directory as the CLI binary)
	serverBinary := filepath.Join(filepath.Dir(d.exePath), "netfetchd")
	if _, err := os.Stat(serverBinary); os.IsNotExist(err) {
		return fmt.Errorf("server binary 'netfetchd' not found in %s", filepath.Dir(d.exePath))
	}

	// Start the daemon process
	cmd := exec.Command(serverBinary,
		"--web-port", fmt.Sprintf("%d", d.webPort),
		"--config", d.configPath,
	)

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %v", err)
	}

	// Write PID file
	if err := os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %v", err)
	}

	fmt.Printf("Daemon started (PID: %d)\n", cmd.Process.Pid)
	return nil
}

func (d *Daemon) Stop() error {
	pid, err := d.getPID()
	if err != nil {
		return fmt.Errorf("daemon is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %v", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop daemon: %v", err)
	}

	err = os.Remove(d.pidFile)
	if err != nil {
		return err
	}
	fmt.Println("Daemon stopped")
	return nil
}

func (d *Daemon) Status() error {
	if !d.IsRunning() {
		fmt.Println("Daemon is not running")
		return nil
	}

	pid, _ := d.getPID()
	fmt.Printf("Daemon is running (PID: %d)\n", pid)
	return nil
}

func (d *Daemon) IsRunning() bool {
	pid, err := d.getPID()
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

func (d *Daemon) getPID() (int, error) {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, err
	}

	return pid, nil
}

func (d *Daemon) InstallSystemd() error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=NetFetch System Information Service
After=network.target

[Service]
Type=forking
ExecStart=%s -d start --port %d --web-port %d --config %s
ExecStop=%s -d stop
PIDFile=%s
Restart=on-failure

[Install]
WantedBy=multi-user.target
`, d.exePath, d.port, d.webPort, d.configPath, d.exePath, d.pidFile)

	unitPath := filepath.Join("/etc/systemd/system", d.systemdUnit)
	if err := os.WriteFile(unitPath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write systemd unit file: %v", err)
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %v", err)
	}

	fmt.Println("Systemd service installed successfully")
	fmt.Println("To start the service:")
	fmt.Println("  sudo systemctl start netfetch")
	fmt.Println("To enable on boot:")
	fmt.Println("  sudo systemctl enable netfetch")
	return nil
}

func (d *Daemon) UninstallSystemd() error {
	unitPath := filepath.Join("/etc/systemd/system", d.systemdUnit)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove systemd unit file: %v", err)
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %v", err)
	}

	fmt.Println("Systemd service uninstalled successfully")
	return nil
}
