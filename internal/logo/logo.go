package logo

import (
	"encoding/json"
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

func LoadAll(dir string) (map[string]*Logo, error) {
	logos := make(map[string]*Logo)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Error reading logo directory: %v", err)
		return nil, err
	}

	log.Printf("Found %d entries in logo directory", len(entries))

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			logo, err := loadLogo(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, err
			}
			logoKey := strings.ToLower(logo.DistroName)
			logos[logoKey] = logo
		}
	}

	return logos, nil
}

func loadLogo(path string) (*Logo, error) {
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
