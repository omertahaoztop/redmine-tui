package sync

import (
	"fmt"
	"redmine-tui/config"
	"redmine-tui/planka"
	"redmine-tui/redmine"
)

func SyncIssuesToPlanka(redmineClient *redmine.Client, plankaClient *planka.Client, cfg *config.Config) error {
	// 1. Fetch assigned issues from Redmine
	issues, err := redmineClient.GetAssignedIssues()
	if err != nil {
		return fmt.Errorf("failed to fetch redmine issues: %w", err)
	}

	// 2. Fetch current cards from Planka List
	cards, err := plankaClient.GetCards(cfg.Planka.BoardID, cfg.Planka.ListID)
	if err != nil {
		return fmt.Errorf("failed to fetch planka cards: %w", err)
	}

	// 3. Map for quick lookup
	// Map Planka Card Name -> Card ID
	plankaMap := make(map[string]string)
	for _, card := range cards {
		plankaMap[card.Name] = card.ID
	}

	// Map Redmine Issue Subject -> Issue
	redmineMap := make(map[string]redmine.Issue)
	for _, issue := range issues {
		redmineMap[issue.Subject] = issue
	}

	// 4. Identify Additions
	for subject := range redmineMap {
		if _, exists := plankaMap[subject]; !exists {
			fmt.Printf("Adding card to Planka: %s\n", subject)
			if err := plankaClient.CreateCard(cfg.Planka.BoardID, cfg.Planka.ListID, subject); err != nil {
				fmt.Printf("Error creating card '%s': %v\n", subject, err)
			}
		}
	}

	// 5. Identify Removals
	for name, id := range plankaMap {
		if _, exists := redmineMap[name]; !exists {
			fmt.Printf("Removing card from Planka: %s\n", name)
			if err := plankaClient.DeleteCard(id); err != nil {
				fmt.Printf("Error deleting card '%s': %v\n", name, err)
			}
		}
	}

	return nil
}
