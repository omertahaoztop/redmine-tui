package main

import (
	"fmt"
	"os"
	"redmine-tui/config"
	"redmine-tui/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.APIKey == "" || cfg.Host == "" {
		fmt.Println("Error: API Key and Host must be configured via ~/.redmine-tui.yaml or environment variables (REDMINE_API_KEY, REDMINE_HOST)")
		os.Exit(1)
	}

	m := ui.NewModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
