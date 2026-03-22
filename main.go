package main

import (
	"flag"
	"fmt"
	"os"
	"redmine-tui/config"
	"redmine-tui/redmine"
	"redmine-tui/sync"
	"redmine-tui/ui"
	"redmine-tui/vikunja"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	syncMode := flag.Bool("sync", false, "Run in sync mode (headless) to sync Redmine issues to Vikunja")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.APIKey == "" || cfg.Host == "" {
		fmt.Println("Error: API Key and Host must be configured via ~/.redmine-tui.yaml or environment variables (REDMINE_API_KEY, REDMINE_HOST)")
		os.Exit(1)
	}

	if *syncMode {
		runSync(cfg)
		return
	}

	m := ui.NewModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func runSync(cfg *config.Config) {
	fmt.Println("Starting Redmine -> Vikunja Sync...")

	redmineClient := redmine.NewClient(cfg.APIKey, cfg.Host)
	vikunjaClient := vikunja.NewClient(cfg.Vikunja.BaseURL, cfg.Vikunja.Token, cfg.Vikunja.Username, cfg.Vikunja.Password)

	fmt.Println("Logging into Vikunja...")
	if err := vikunjaClient.Login(); err != nil {
		fmt.Printf("Error logging into Vikunja: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Syncing issues...")
	if err := sync.SyncIssuesToVikunja(redmineClient, vikunjaClient, cfg); err != nil {
		fmt.Printf("Error syncing: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sync completed successfully!")
}
