package ui

import (
	"fmt"

	"redmine-tui/sync"
	"redmine-tui/vikunja"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// clipboardWrite is isolated so the model does not depend on the clipboard lib.
func clipboardWrite(s string) error { return clipboard.WriteAll(s) }

// ── Commands ────────────────────────────────────────────────────────────────
// Every async Redmine/Vikunja call lives here; the model only dispatches them.

func (m Model) fetchIssues() tea.Msg {
	issues, err := m.client.GetAssignedIssues()
	if err != nil {
		return err
	}
	return issuesMsg{issues: issues, loaded: true}
}

func (m Model) fetchStatuses() tea.Msg {
	statuses, err := m.client.GetIssueStatuses()
	if err != nil {
		return err
	}
	return statuses
}

func (m Model) fetchIssueDetails(id int) tea.Cmd {
	return func() tea.Msg {
		issue, err := m.client.GetIssueDetails(id)
		if err != nil {
			return err
		}
		return issue
	}
}

func (m Model) updateStatus(issueID, statusID int, statusName string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.UpdateIssueStatus(issueID, statusID); err != nil {
			return err
		}
		return statusUpdatedMsg{status: statusName}
	}
}

func (m Model) logTime(issueID int, hours, comments string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.LogTime(issueID, hours, comments); err != nil {
			return err
		}
		return "time_logged"
	}
}

func (m Model) searchIssues(query string) tea.Cmd {
	return func() tea.Msg {
		issues, err := m.client.SearchIssues(query)
		if err != nil {
			return err
		}
		return issuesMsg{issues: issues, loaded: true}
	}
}

func (m Model) syncVikunja() tea.Cmd {
	return func() tea.Msg {
		vc := vikunja.NewClient(m.cfg.Vikunja.BaseURL, m.cfg.Vikunja.Token, m.cfg.Vikunja.Username, m.cfg.Vikunja.Password)
		if err := vc.Login(); err != nil {
			return fmt.Errorf("vikunja login: %w", err)
		}
		if err := sync.SyncIssuesToVikunja(m.client, vc, m.cfg); err != nil {
			return err
		}
		return "vikunja_synced"
	}
}
