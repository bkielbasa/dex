package config

import (
	"os"
	"path/filepath"
	"strings"
)

const maxHistory = 500

func HistoryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dex", "history")
}

func LoadHistory(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var result []string
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

func SaveHistory(path string, history []string) error {
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	return os.WriteFile(path, []byte(strings.Join(history, "\n")+"\n"), 0644)
}
