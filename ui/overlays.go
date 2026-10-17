package ui

import (
	"fmt"
	"strings"

	"redmine-tui/i18n"

	"github.com/charmbracelet/lipgloss"
)

// overlayHeader renders the shared chrome for a full-screen overlay.
func overlayHeader(title, hint string, accent lipgloss.TerminalColor, w int) string {
	bar := lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(accent).
		Bold(true).
		Width(w).
		Padding(0, 2).
		Render("◈ " + title + "  " + styleHelpDesc.Render(hint))
	divider := lipgloss.NewStyle().Foreground(accent).Render(strings.Repeat("─", w))
	return lipgloss.JoinVertical(lipgloss.Left, bar, divider)
}

func (m Model) overlayWidth(fallback int) int {
	if m.windowWidth == 0 {
		return fallback
	}
	return m.windowWidth
}

// ── Error ───────────────────────────────────────────────────────────────────

func (m Model) viewError() string {
	w := m.windowWidth
	if w == 0 {
		w = 80
	}
	// Leave room for the panel border and padding so a long error wraps
	// inside the box instead of being cut off by the terminal edge.
	panelW := max(24, min(84, w-12))
	body := styleDetailPanel.Width(panelW).Render(
		styleDetailTitle.Render(i18n.T("error")) + "\n\n" +
			lipgloss.NewStyle().Foreground(colorDanger).Render(singleLine(m.err.Error(), 240)) + "\n\n" +
			styleHelpDesc.Render(i18n.T("err.hint")),
	)
	return lipgloss.Place(w, m.windowHeight, lipgloss.Center, lipgloss.Center, body)
}

// ── Status picker ─────────────────────────────────────────────────────────────

func (m Model) viewStatusPicker() string {
	w := m.overlayWidth(100)

	title := ""
	if m.selected != nil {
		title = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2).
			Render(fmt.Sprintf("#%d %s", m.selected.ID, m.selected.Subject))
	}

	cards := make([]string, 0, len(m.statuses))
	for i, st := range m.statuses {
		accent := statusColor(st.Name)
		style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
		if i == m.colIdx {
			style = style.BorderForeground(accent).Foreground(accent).Bold(true)
			cards = append(cards, style.Render("● "+st.Name))
		} else {
			style = style.BorderForeground(colorBorder).Foreground(colorMuted)
			cards = append(cards, style.Render("○ "+st.Name))
		}
	}

	board := lipgloss.NewStyle().PaddingLeft(2).PaddingTop(1).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, cards...))

	return lipgloss.JoinVertical(lipgloss.Left,
		overlayHeader(i18n.T("status.title"), i18n.T("hint.status"), colorPurple, w),
		title, board,
	)
}

// ── Time input ────────────────────────────────────────────────────────────────

func (m Model) viewTimeInput() string {
	w := m.overlayWidth(80)

	step := i18n.T("time.step1")
	if m.inputState == 1 {
		step = i18n.T("time.step2")
	}

	panel := styleInputPanel.
		BorderForeground(colorInfo).
		Render(lipgloss.NewStyle().Foreground(colorInfo).Bold(true).Render(step) +
			"\n\n" + m.textInput.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		overlayHeader(i18n.T("time.title"), i18n.T("hint.time"), colorInfo, w),
		lipgloss.NewStyle().Padding(2, 4).Render(panel),
	)
}

// ── Search ───────────────────────────────────────────────────────────────────

func (m Model) viewSearch() string {
	w := m.overlayWidth(80)

	panel := styleInputPanel.
		BorderForeground(colorWarn).
		Render(lipgloss.NewStyle().Foreground(colorWarn).Bold(true).Render(i18n.T("search.label")) +
			"\n\n" + m.textInput.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		overlayHeader(i18n.T("search.title"), i18n.T("hint.search"), colorWarn, w),
		lipgloss.NewStyle().Padding(2, 4).Render(panel),
	)
}

// ── Help ─────────────────────────────────────────────────────────────────────

func (m Model) viewHelp() string {
	w := m.overlayWidth(100)

	groups := keymapGroups()
	blocks := make([]string, 0, len(groups))
	for _, g := range groups {
		lines := []string{lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(g.title)}
		for _, b := range g.items {
			lines = append(lines, "  "+styleHelpKey.Width(14).Render(b.keys)+"  "+
				styleHelpDesc.Render(i18n.T(b.desc)))
		}
		blocks = append(blocks, lipgloss.JoinVertical(lipgloss.Left, lines...))
	}

	var body string
	if w >= 90 && len(blocks) == 4 {
		left := lipgloss.JoinVertical(lipgloss.Left, blocks[0], "", blocks[1])
		right := lipgloss.JoinVertical(lipgloss.Left, blocks[2], "", blocks[3])
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width((w-8)/2).Render(left),
			right,
		)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, blocks...)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		overlayHeader(i18n.T("help.title"), i18n.T("hint.help"), colorAccent, w),
		lipgloss.NewStyle().Padding(1, 2).Render(body),
	)
}
