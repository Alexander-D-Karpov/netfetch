package config

import (
	"gopkg.in/yaml.v3"
	"os"
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

	return &cfg, nil
}
