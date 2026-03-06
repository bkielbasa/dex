package main

import (
	"fmt"
	"os"

	"github.com/bklimczak/dex/internal/app"
	"github.com/bklimczak/dex/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfgPath := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app.New(cfg, cfgPath), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
