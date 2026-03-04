package sync

import (
	"fmt"
	"redmine-tui/config"
	"redmine-tui/planka"
	"redmine-tui/redmine"
)

func SyncIssuesToPlanka(redmineClient *redmine.Client, plankaClient *planka.Client, cfg *config.Config) error {
	issues, err := redmineClient.GetAssignedIssues()
	if err != nil {
		return fmt.Errorf("failed to fetch redmine issues: %w", err)
	}

	cards, err := plankaClient.GetCards(cfg.Planka.BoardID, cfg.Planka.ListID)
	if err != nil {
		return fmt.Errorf("failed to fetch planka cards: %w", err)
	}

	// Also fetch closed list cards so we don't re-add already-moved cards.
	var closedCards []planka.Card
	if cfg.Planka.ClosedListID != "" {
		closedCards, _ = plankaClient.GetCards(cfg.Planka.BoardID, cfg.Planka.ClosedListID)
	}

	plankaMap := make(map[string]string)
	for _, card := range cards {
		plankaMap[card.Name] = card.ID
	}

	closedMap := make(map[string]string)
	for _, card := range closedCards {
		closedMap[card.Name] = card.ID
	}

	redmineMap := make(map[string]redmine.Issue)
	for _, issue := range issues {
		redmineMap[issue.Subject] = issue
	}

	for subject := range redmineMap {
		if _, exists := plankaMap[subject]; !exists {
			if id, inClosed := closedMap[subject]; inClosed && cfg.Planka.ClosedListID != "" {
				// Issue reopened — move back to active list.
				if err := plankaClient.MoveCard(id, cfg.Planka.ListID); err != nil {
					return fmt.Errorf("failed to move card '%s' to active list: %w", subject, err)
				}
			} else {
				if err := plankaClient.CreateCard(cfg.Planka.BoardID, cfg.Planka.ListID, subject); err != nil {
					return fmt.Errorf("failed to create card '%s': %w", subject, err)
				}
			}
		}
	}

	for name, id := range plankaMap {
		if _, exists := redmineMap[name]; !exists {
			if cfg.Planka.ClosedListID != "" {
				if err := plankaClient.MoveCard(id, cfg.Planka.ClosedListID); err != nil {
					return fmt.Errorf("failed to move card '%s' to closed list: %w", name, err)
				}
			} else {
				if err := plankaClient.DeleteCard(id); err != nil {
					return fmt.Errorf("failed to delete card '%s': %w", name, err)
				}
			}
		}
	}

	return nil
}
