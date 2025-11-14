package logo

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Logo struct {
	DistroName string   `json:"distro_name"`
	Colors     string   `json:"colors"`
	AsciiArt   []string `json:"ascii_art"`
}

var EmbeddedLogos embed.FS

func LoadAll(dir string) (map[string]*Logo, error) {
	if dir != "" && dirExistsOnDisk(dir) {
		return loadFromDisk(dir)
	}

	return loadFromEmbedded()
}

func dirExistsOnDisk(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func loadFromDisk(dir string) (map[string]*Logo, error) {
	logos := make(map[string]*Logo)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			logo, err := loadLogoFromDisk(filepath.Join(dir, entry.Name()))
			if err != nil {
				log.Printf("Error loading logo %s: %v", entry.Name(), err)
				continue
			}
			logoKey := strings.ToLower(logo.DistroName)
			logos[logoKey] = logo
		}
	}

	return logos, nil
}

func loadFromEmbedded() (map[string]*Logo, error) {
	logos := make(map[string]*Logo)

	entries, err := EmbeddedLogos.ReadDir("logos")
	if err != nil {
		log.Printf("Error reading embedded logos: %v", err)
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			logo, err := loadEmbeddedLogo(filepath.Join("logos", entry.Name()))
			if err != nil {
				log.Printf("Error loading embedded logo %s: %v", entry.Name(), err)
				continue
			}
			logoKey := strings.ToLower(logo.DistroName)
			logos[logoKey] = logo
		}
	}

	return logos, nil
}

func loadLogoFromDisk(path string) (*Logo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var logo Logo
	if err := json.Unmarshal(data, &logo); err != nil {
		return nil, err
	}

	return &logo, nil
}

func loadEmbeddedLogo(path string) (*Logo, error) {
	data, err := fs.ReadFile(EmbeddedLogos, path)
	if err != nil {
		return nil, err
	}

	var logo Logo
	if err := json.Unmarshal(data, &logo); err != nil {
		return nil, err
	}

	return &logo, nil
}
