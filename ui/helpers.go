package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"redmine-tui/i18n"

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
	case hasAny(n, "kritik", "acil", "urgent", "immediate"):
		return prioUrgent
	case hasAny(n, "yüksek", "high"):
		return prioHigh
	case n == "normal" || hasAny(n, "orta", "medium"):
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

func priorityColor(name string) lipgloss.TerminalColor {
	switch classifyPriority(name) {
	case prioUrgent:
		return colorDanger
	case prioHigh:
		return colorWarn
	case prioNormal:
		return colorInfo
	default:
		return colorMuted
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

// humanDue renders a due date as a short human phrase plus its urgency style.
// It returns an empty label when there is no parsable date.
func humanDue(due string) (string, lipgloss.Style) {
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
		return "⚠ " + fmt.Sprintf(i18n.T("due.over"), -days), styleOverdue
	case days == 0:
		return "⏰ " + i18n.T("due.today"), styleDueToday
	case days <= 3:
		return "→ " + fmt.Sprintf(i18n.T("due.left"), days), styleDueSoon
	default:
		return "→ " + fmt.Sprintf(i18n.T("due.left"), days), stylePriorityLow
	}
}

// relativeTime renders how long ago t happened, in the active language.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return i18n.T("rel.now")
	case d < time.Hour:
		return fmt.Sprintf(i18n.T("rel.minutes"), int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf(i18n.T("rel.hours"), int(d.Hours()))
	case d < 48*time.Hour:
		return i18n.T("rel.yesterday")
	default:
		return fmt.Sprintf(i18n.T("rel.days"), int(d.Hours()/24))
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

// ── Markdown-lite ────────────────────────────────────────────────────────────

var (
	reBold = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reCode = regexp.MustCompile("`([^`]+)`")
	reLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

var (
	styleMdHeading = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleMdBold    = lipgloss.NewStyle().Bold(true).Foreground(colorText)
	styleMdCode    = lipgloss.NewStyle().Foreground(colorPurple)
	styleMdLink    = lipgloss.NewStyle().Foreground(colorInfo).Underline(true)
)

// renderMarkdownLite renders the small subset of markdown that shows up in
// Redmine descriptions and notes: headings, bullets, bold, inline code, links.
func renderMarkdownLite(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		heading := ""
		switch {
		case strings.HasPrefix(trimmed, "### "):
			heading, trimmed = strings.TrimPrefix(trimmed, "### "), ""
		case strings.HasPrefix(trimmed, "## "):
			heading, trimmed = strings.TrimPrefix(trimmed, "## "), ""
		case strings.HasPrefix(trimmed, "# "):
			heading, trimmed = strings.TrimPrefix(trimmed, "# "), ""
		case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "):
			trimmed = "  • " + trimmed[2:]
		}

		trimmed = reLink.ReplaceAllStringFunc(trimmed, func(m string) string {
			g := reLink.FindStringSubmatch(m)
			return styleMdLink.Render(g[1])
		})
		trimmed = reCode.ReplaceAllStringFunc(trimmed, func(m string) string {
			return styleMdCode.Render(reCode.FindStringSubmatch(m)[1])
		})
		trimmed = reBold.ReplaceAllStringFunc(trimmed, func(m string) string {
			return styleMdBold.Render(reBold.FindStringSubmatch(m)[1])
		})

		if heading != "" {
			trimmed = styleMdHeading.Render(heading)
		}
		lines[i] = trimmed
	}
	return strings.Join(lines, "\n")
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
