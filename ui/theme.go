package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Palette ──────────────────────────────────────────────────────────────────
// A modern, soft-contrast palette with adaptive light/dark variants.

var (
	colorBg         = lipgloss.AdaptiveColor{Light: "#f4f4fb", Dark: "#13131f"}
	colorSurface    = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#1c1c2e"}
	colorSurfaceAlt = lipgloss.AdaptiveColor{Light: "#f0f0f8", Dark: "#23233a"}
	colorBorder     = lipgloss.AdaptiveColor{Light: "#dcdce8", Dark: "#2e2e48"}
	colorAccent     = lipgloss.AdaptiveColor{Light: "#6c5ce7", Dark: "#a29bfe"}
	colorAccentDim  = lipgloss.AdaptiveColor{Light: "#a29bfe", Dark: "#6c5ce7"}
	colorText       = lipgloss.AdaptiveColor{Light: "#1a1a24", Dark: "#e8e8f4"}
	colorMuted      = lipgloss.AdaptiveColor{Light: "#8a8a9e", Dark: "#6c6c8a"}
	colorSuccess    = lipgloss.AdaptiveColor{Light: "#16a34a", Dark: "#4ade80"}
	colorWarn       = lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#fbbf24"}
	colorDanger     = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#f87171"}
	colorInfo       = lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#60a5fa"}
	colorPurple     = lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#c4b5fd"}
	colorSelBg      = lipgloss.AdaptiveColor{Light: "#ece9ff", Dark: "#252046"}
)

// ── Shared styles ────────────────────────────────────────────────────────────

var (
	styleColumnHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorMuted).
				PaddingLeft(1).
				PaddingRight(1).
				PaddingBottom(1)

	styleColumn = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			PaddingLeft(1).
			PaddingRight(1)

	styleColumnActive = styleColumn.
				BorderForeground(colorAccent)

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			PaddingLeft(1).
			PaddingRight(1).
			MarginBottom(1)

	styleCardSelected = styleCard.
				BorderForeground(colorAccent).
				Background(colorSelBg)

	styleDetailTitle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				PaddingBottom(1)

	styleDetailLabel = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true)

	styleDetailPanel = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(1, 2)

	styleInputPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple).
			Padding(1, 2)

	styleHelpKey  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleHelpDesc = lipgloss.NewStyle().Foreground(colorMuted)

	styleProjectTag = lipgloss.NewStyle().
			Foreground(colorSurface).
			Background(colorInfo).
			PaddingLeft(1).
			PaddingRight(1)

	styleCountBadge = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSurfaceAlt).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	stylePriorityUrgent = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	stylePriorityHigh   = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	stylePriorityNormal = lipgloss.NewStyle().Foreground(colorInfo)
	stylePriorityLow    = lipgloss.NewStyle().Foreground(colorMuted)

	styleOverdue  = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	styleDueToday = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	styleDueSoon  = lipgloss.NewStyle().Foreground(colorInfo)

	// Overlay chrome
	styleOverlayItem = lipgloss.NewStyle().Foreground(colorText)

	// Loading skeleton
	styleSkeleton = lipgloss.NewStyle().Foreground(colorBorder)

	// Command palette
	stylePaletteCursor = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)
)

// ── Status semantics ─────────────────────────────────────────────────────────

// statusColor maps a Redmine status name (English or Turkish) onto the
// palette, so a column reads as a state at a glance.
func statusColor(name string) lipgloss.TerminalColor {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case hasAny(n, "reopen", "yeniden aç"):
		return colorDanger
	case hasAny(n, "progress", "devam", "üzerinde çalış", "inceleniyor"):
		return colorWarn
	case hasAny(n, "feedback", "geri bildirim"):
		return colorPurple
	case hasAny(n, "resolved", "çözül", "çözüm"):
		return colorSuccess
	case hasAny(n, "closed", "kapalı", "kapatıl", "rejected", "reddedil", "iptal"):
		return colorMuted
	case hasAny(n, "new", "yeni", "açık", "open", "backlog"):
		return colorInfo
	default:
		return colorAccentDim
	}
}

func statusDot(name string) string {
	return lipgloss.NewStyle().Foreground(statusColor(name)).Render("●")
}

func hasAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
