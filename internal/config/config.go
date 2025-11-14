package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddress string   `yaml:"listen_address"`
	ActiveModules []string `yaml:"active_modules"`
	DefaultLogo   string   `yaml:"default_logo"`
	LogoDir       string   `yaml:"logo_dir"`
}

func Load(filename string) (*Config, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.DefaultLogo == "" {
		cfg.DefaultLogo = "arch"
	}

	if len(cfg.ActiveModules) == 0 {
		cfg.ActiveModules = GetDefaultModules()
	}

	return &cfg, nil
}

func GetDefaultModules() []string {
	return []string{
		"os", "hostinfo", "bios", "kernel", "uptime", "packages", "shell",
		"resolution", "de", "wm", "theme", "icons", "terminal", "cpu", "gpu",
		"memory", "disk", "swap", "battery", "locale", "processes", "cpuusage",
		"publicip", "wifi", "datetime", "users", "brightness", "loginmanager",
	}
}
