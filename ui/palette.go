package ui

import (
	"strings"

	"redmine-tui/i18n"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// paletteCommand is one entry of the command palette. hint shows the key that
// does the same thing, so the palette teaches the keyboard.
type paletteCommand struct {
	title string
	hint  string
	run   func(*Model) tea.Cmd
}

func (m *Model) openPalette() tea.Cmd {
	m.state = paletteView
	m.paletteIdx = 0
	m.paletteInput.Reset()
	m.paletteInput.Focus()
	return textinput.Blink
}

func (m Model) paletteCommands() []paletteCommand {
	return []paletteCommand{
		{i18n.T("cmd.refresh"), "r", func(m *Model) tea.Cmd {
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.fetchIssues, m.fetchStatuses)
		}},
		{i18n.T("cmd.search"), "^f", func(m *Model) tea.Cmd { return m.openSearch() }},
		{i18n.T("cmd.detail"), "↵", func(m *Model) tea.Cmd { return m.updateKanban("enter") }},
		{i18n.T("cmd.status"), "s", func(m *Model) tea.Cmd { cmd, _ := m.issueAction("s"); return cmd }},
		{i18n.T("cmd.time"), "t", func(m *Model) tea.Cmd { cmd, _ := m.issueAction("t"); return cmd }},
		{i18n.T("cmd.copy"), "y", func(m *Model) tea.Cmd { cmd, _ := m.issueAction("y"); return cmd }},
		{i18n.T("cmd.open"), "o", func(m *Model) tea.Cmd { cmd, _ := m.issueAction("o"); return cmd }},
		{i18n.T("cmd.filter"), "f", func(m *Model) tea.Cmd { return m.updateKanban("f") }},
		{i18n.T("cmd.sort"), "S", func(m *Model) tea.Cmd { return m.updateKanban("S") }},
		{i18n.T("cmd.export"), "e", func(m *Model) tea.Cmd { return m.updateKanban("e") }},
		{i18n.T("cmd.sync"), "v", func(m *Model) tea.Cmd { return m.updateKanban("v") }},
		{i18n.T("cmd.help"), "?", func(m *Model) tea.Cmd { m.state = helpView; return nil }},
		{i18n.T("cmd.quit"), "q", func(*Model) tea.Cmd { return tea.Quit }},
	}
}

// paletteMatches filters the commands with a fuzzy match on their titles.
func (m Model) paletteMatches() []paletteCommand {
	all := m.paletteCommands()
	query := strings.TrimSpace(m.paletteInput.Value())
	if query == "" {
		return all
	}

	titles := make([]string, len(all))
	for i, c := range all {
		titles[i] = c.title
	}

	matches := fuzzy.Find(query, titles)
	out := make([]paletteCommand, 0, len(matches))
	for _, mt := range matches {
		out = append(out, all[mt.Index])
	}
	return out
}

func (m Model) viewPalette() string {
	w := m.overlayWidth(80)

	cmds := m.paletteMatches()
	h := m.windowHeight
	if h == 0 {
		h = 30
	}
	maxRows := max(3, h-8)

	start := 0
	if m.paletteIdx >= maxRows {
		start = m.paletteIdx - maxRows + 1
	}
	end := min(start+maxRows, len(cmds))

	lines := []string{}
	if len(cmds) == 0 {
		lines = append(lines, styleHelpDesc.Render("  "+i18n.T("palette.noresult")))
	}
	for i := start; i < end; i++ {
		c := cmds[i]
		if i == m.paletteIdx {
			lines = append(lines, stylePaletteCursor.Render("  ❯ "+c.title)+"  "+styleHelpDesc.Render(c.hint))
		} else {
			lines = append(lines, "    "+styleOverlayItem.Render(c.title)+"  "+styleHelpDesc.Render(c.hint))
		}
	}

	panel := styleInputPanel.
		BorderForeground(colorAccent).
		Render(m.paletteInput.View() + "\n\n" + strings.Join(lines, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left,
		overlayHeader(i18n.T("palette.title"), i18n.T("hint.palette"), colorAccent, w),
		lipgloss.NewStyle().Padding(1, 2).Render(panel),
	)
}
