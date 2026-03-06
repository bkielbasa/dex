package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type SavedQuery struct {
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
}

type SavedQueries struct {
	// Connection name -> list of saved queries
	Connections map[string][]SavedQuery `yaml:"connections"`
}

func QueriesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dex", "queries.yaml")
}

func LoadQueries(path string) *SavedQueries {
	data, err := os.ReadFile(path)
	if err != nil {
		return &SavedQueries{Connections: make(map[string][]SavedQuery)}
	}
	var sq SavedQueries
	if err := yaml.Unmarshal(data, &sq); err != nil {
		return &SavedQueries{Connections: make(map[string][]SavedQuery)}
	}
	if sq.Connections == nil {
		sq.Connections = make(map[string][]SavedQuery)
	}
	return &sq
}

func SaveQueries(path string, sq *SavedQueries) error {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	data, err := yaml.Marshal(sq)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
