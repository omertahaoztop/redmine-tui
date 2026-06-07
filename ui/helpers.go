package ui

import (
	"fmt"
	"redmine-tui/i18n"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// priorityKind classifies a priority name (locale-independent) into a tier.
type priorityKind int

const (
	prioUrgent priorityKind = iota
	prioHigh
	prioNormal
	prioLow
)

func classifyPriority(name string) priorityKind {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(n, "kritik") || strings.Contains(n, "acil") || strings.Contains(n, "urgent") || strings.Contains(n, "immediate"):
		return prioUrgent
	case strings.Contains(n, "yüksek") || strings.Contains(n, "high"):
		return prioHigh
	case n == "normal" || strings.Contains(n, "orta") || strings.Contains(n, "medium"):
		return prioNormal
	default:
		return prioLow
	}
}

func priorityStyle(name string) lipgloss.Style {
	switch classifyPriority(name) {
	case prioUrgent:
		return stylePriorityUrgent
	case prioHigh:
		return stylePriorityHigh
	case prioNormal:
		return stylePriorityNormal
	default:
		return stylePriorityLow
	}
}

func priorityIcon(name string) string {
	switch classifyPriority(name) {
	case prioUrgent:
		return "▲▲"
	case prioHigh:
		return "▲"
	case prioNormal:
		return "●"
	default:
		return "▽"
	}
}

func trackerIcon(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bug", "hata":
		return "● "
	case "feature", "özellik", "yeni özellik":
		return "✦ "
	case "support", "destek":
		return "◆ "
	case "task", "görev":
		return "▪ "
	default:
		return ""
	}
}

func dueDateStyle(due string) (string, lipgloss.Style) {
	if due == "" {
		return "", stylePriorityLow
	}
	t, err := time.Parse("2006-01-02", due)
	if err != nil {
		return due, stylePriorityLow
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(t.Sub(today).Hours() / 24)
	switch {
	case days < 0:
		return fmt.Sprintf("⚠ %s (%dd)", due, days), styleOverdue
	case days == 0:
		return "⏰ " + i18n.T("msg.due.today"), styleDueToday
	case days <= 3:
		return fmt.Sprintf("→ %s (%dd)", due, days), styleDueSoon
	default:
		return fmt.Sprintf("→ %s", due), stylePriorityLow
	}
}

func progressBar(ratio int, width int) string {
	if width < 4 {
		return ""
	}
	filled := (ratio * width) / 100
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	color := colorSuccess
	if ratio < 30 {
		color = colorMuted
	} else if ratio < 70 {
		color = colorInfo
	}
	return lipgloss.NewStyle().Foreground(color).Render(bar)
}

// truncate shortens s to at most max display cells, appending an ellipsis.
// It is rune-aware so multibyte characters (ş, ğ, ü, …) are never split.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
