package ui

import (
	"fmt"
	"strings"

	"redmine-tui/i18n"
	"redmine-tui/redmine"

	"github.com/charmbracelet/lipgloss"
)

// narrowBreakpoint is the width below which the board collapses to a single
// column with a tab strip instead of squeezing every column.
const narrowBreakpoint = 80

func (m Model) narrow() bool {
	return m.windowWidth > 0 && m.windowWidth < narrowBreakpoint
}

// ── Kanban view ──────────────────────────────────────────────────────────────

func (m Model) viewKanban() string {
	w := m.windowWidth
	h := m.windowHeight
	if w == 0 {
		w = 120
	}
	if h == 0 {
		h = 40
	}

	headerH := 2
	footerH := 2
	if m.statusMsg != "" {
		footerH = 3
	}
	boardH := h - headerH - footerH
	if boardH < 4 {
		boardH = 4
	}

	header := m.renderHeader(w)
	footer := m.renderFooter(w)

	if !m.loaded {
		return lipgloss.JoinVertical(lipgloss.Left, header, m.renderSkeleton(w, boardH), footer)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, m.renderBoard(w, boardH), footer)
}

func (m Model) renderHeader(w int) string {
	left := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("◈ " + i18n.T("app.title"))
	if m.activeFilter != "" {
		left += "  " + styleProjectTag.Render(truncate(m.activeFilter, 24))
	}
	if m.sortMode != sortServer {
		left += "  " + styleCountBadge.Render("⇅ "+m.sortMode.label())
	}

	count := fmt.Sprintf("%d %s", len(m.allIssues), i18n.T("app.issues"))
	right := lipgloss.NewStyle().Foreground(colorMuted).Render(count)
	if m.loading {
		right = m.spinner.View() + " " + right
	}

	// The bar is w-2 wide and carries 1 cell of horizontal padding on each
	// side, so its inner content width is w-4. Computing the gap against a
	// wider box makes the line wrap and pushes the title off screen.
	inner := w - 4
	if inner < 4 {
		inner = 4
	}
	gap := inner - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		right = ""
		gap = max(0, inner-lipgloss.Width(left))
	}

	bar := lipgloss.NewStyle().
		Background(colorSurface).
		Padding(0, 1).
		Width(w - 2).
		Render(left + strings.Repeat(" ", gap) + right)

	divider := lipgloss.NewStyle().
		Foreground(colorAccent).
		Render(strings.Repeat("─", w))

	return lipgloss.JoinVertical(lipgloss.Left, bar, divider)
}

// statusLine renders the transient toast, or "" when there is none.
func (m Model) statusLine() string {
	if m.statusMsg == "" {
		return ""
	}
	glyph, color := "✓", colorSuccess
	if m.statusIsError {
		glyph, color = "✗", colorDanger
	}
	return lipgloss.NewStyle().Foreground(color).PaddingLeft(2).Render(glyph + " " + m.statusMsg)
}

func (m Model) renderFooter(w int) string {
	parts := []string{}
	for _, kv := range footerKeys() {
		parts = append(parts, styleHelpKey.Render(kv.keys)+" "+styleHelpDesc.Render(i18n.T(kv.desc)))
	}
	help := strings.Join(parts, styleHelpDesc.Render(" · "))

	lines := []string{lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", w))}
	if line := m.statusLine(); line != "" {
		lines = append(lines, line)
	}
	lines = append(lines, lipgloss.NewStyle().PaddingLeft(2).Render(help))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ── Loading skeleton ────────────────────────────────────────────────────────

func (m Model) renderSkeleton(w, h int) string {
	const numCols = 3
	colW := (w - numCols*4) / numCols
	if colW < 20 {
		colW = 20
	}

	cols := make([]string, 0, numCols)
	for range numCols {
		lines := []string{styleSkeleton.Render(" " + strings.Repeat("─", max(4, colW-6)))}
		for used := 0; used+5 <= h-2; used += 5 {
			body := make([]string, 3)
			for i := range body {
				width := max(6, colW-10-i*4)
				body[i] = styleSkeleton.Render(strings.Repeat("░", width))
			}
			lines = append(lines, lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1).
				MarginBottom(1).
				Render(strings.Join(body, "\n")))
		}
		cols = append(cols, lipgloss.NewStyle().Width(colW).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}

// ── Empty state ─────────────────────────────────────────────────────────────

func (m Model) renderEmpty(w, h int) string {
	box := lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(colorMuted).Render("⊘ "+i18n.T("app.noissues")),
		"",
		styleHelpDesc.Render(i18n.T("empty.hint")),
	)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

// ── Board ───────────────────────────────────────────────────────────────────

func (m Model) renderBoard(w, h int) string {
	m.geom.cols = m.geom.cols[:0]
	m.geom.cards = m.geom.cards[:0]

	if len(m.columns) == 0 {
		return m.renderEmpty(w, h)
	}
	if m.narrow() {
		return m.renderBoardNarrow(w, h)
	}

	const colOverhead = 4
	const minColContent = 24

	numCols := len(m.columns)
	colContent := (w - numCols*colOverhead) / numCols
	if colContent < minColContent {
		for numCols > 1 && colContent < minColContent {
			numCols--
			colContent = (w - numCols*colOverhead) / numCols
		}
		if colContent < minColContent {
			colContent = minColContent
		}
	}

	visStart := m.colIdx - numCols/2
	if visStart < 0 {
		visStart = 0
	}
	if visStart+numCols > len(m.columns) {
		visStart = len(m.columns) - numCols
		if visStart < 0 {
			visStart = 0
		}
	}
	visEnd := visStart + numCols
	if visEnd > len(m.columns) {
		visEnd = len(m.columns)
	}
	visible := m.columns[visStart:visEnd]

	// The navigation hint sits under the board, so the columns must give up a
	// row for it; otherwise the total exceeds the board height and the header
	// scrolls off the top.
	colH := h
	showHint := len(m.columns) > len(visible)
	if showHint {
		colH--
	}

	cols := make([]string, 0, len(visible))
	x := 0
	for ci, col := range visible {
		globalIdx := visStart + ci
		active := globalIdx == m.colIdx
		accent := statusColor(col.status.Name)

		headerInner := max(4, colContent-2)
		headerText := statusDot(col.status.Name) + " " +
			truncate(col.status.Name, max(2, headerInner-8)) + " " +
			styleCountBadge.Render(fmt.Sprintf("%d", len(col.issues)))

		var header string
		if active {
			header = lipgloss.NewStyle().
				Bold(true).
				Foreground(accent).
				PaddingLeft(1).
				PaddingRight(1).
				PaddingBottom(1).
				Width(headerInner).
				Render(headerText)
		} else {
			header = styleColumnHeader.Width(headerInner).Render(headerText)
		}
		headerH := lipgloss.Height(header)

		all, start, end := m.visibleCards(col, colContent-4, max(1, colH-headerH), active)

		// Board starts at screen row 2; a column's top border sits on the first
		// board row, so its content starts one row lower.
		y := 3 + headerH
		m.geom.cols = append(m.geom.cols, hitCol{x: x, w: colContent, headerY0: 2, headerY1: 3 + headerH})

		lines := []string{header}
		if start > 0 {
			lines = append(lines, styleHelpDesc.Render("  "+i18n.T("more.up")))
			y++
		}
		for i := start; i < end; i++ {
			lines = append(lines, all[i].rendered)
			m.geom.cards = append(m.geom.cards, hitCard{col: globalIdx, index: i, y0: y, y1: y + all[i].height})
			y += all[i].height
		}
		if end < len(col.issues) {
			lines = append(lines, styleHelpDesc.Render("  "+i18n.T("more.down")))
		}

		colStyle := styleColumn
		if active {
			colStyle = styleColumnActive
		}
		cols = append(cols, colStyle.Width(colContent).Height(colH).MaxHeight(colH).
			Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
		x += colContent
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	if showHint {
		hint := " " + fmt.Sprintf(i18n.T("col.nav"), m.colIdx+1, len(m.columns)) + " "
		board = lipgloss.JoinVertical(lipgloss.Left, board, styleHelpDesc.Render(hint))
	}
	return board
}

// renderBoardNarrow shows one full-width column plus a tab strip, so a small
// terminal degrades structurally instead of squeezing every column.
func (m Model) renderBoardNarrow(w, h int) string {
	tabs := make([]string, 0, len(m.columns))
	for i, col := range m.columns {
		label := fmt.Sprintf("%s %d", truncate(col.status.Name, 14), len(col.issues))
		if i == m.colIdx {
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(statusColor(col.status.Name)).Bold(true).Underline(true).
				Render("▸ "+label))
		} else {
			tabs = append(tabs, styleHelpDesc.Render(label))
		}
	}
	strip := lipgloss.NewStyle().PaddingLeft(1).Render(strings.Join(tabs, styleHelpDesc.Render(" · ")))
	stripH := lipgloss.Height(strip)

	col := m.columns[m.colIdx]
	all, start, end := m.visibleCards(col, w-6, max(1, h-stripH-1), true)

	y := 2 + stripH
	m.geom.cols = append(m.geom.cols, hitCol{x: 0, w: w, headerY0: 2, headerY1: 2 + stripH})

	lines := []string{strip}
	if start > 0 {
		lines = append(lines, styleHelpDesc.Render("  "+i18n.T("more.up")))
		y++
	}
	for i := start; i < end; i++ {
		lines = append(lines, all[i].rendered)
		m.geom.cards = append(m.geom.cards, hitCard{col: m.colIdx, index: i, y0: y, y1: y + all[i].height})
		y += all[i].height
	}
	if end < len(col.issues) {
		lines = append(lines, styleHelpDesc.Render("  "+i18n.T("more.down")))
	}

	return lipgloss.NewStyle().Width(w).MaxHeight(h).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

type cardEntry struct {
	rendered string
	height   int
}

// visibleCards renders a column's cards and returns the window that fits,
// scrolled to keep the focused card on screen.
func (m Model) visibleCards(col kanbanColumn, contentW, availH int, active bool) (all []cardEntry, start, end int) {
	all = make([]cardEntry, len(col.issues))
	for i, issue := range col.issues {
		r := renderCard(issue, contentW, active && i == m.cardIdx)
		all[i] = cardEntry{r, lipgloss.Height(r)}
	}

	if active && m.cardIdx < len(all) {
		used := 0
		for i := m.cardIdx; i >= 0; i-- {
			used += all[i].height
			if used > availH {
				start = i + 1
				break
			}
		}
	}

	used := 0
	end = start
	for i := start; i < len(all); i++ {
		if used+all[i].height > availH {
			break
		}
		used += all[i].height
		end = i + 1
	}
	if end == start && start < len(all) {
		end = start + 1
	}
	return all, start, end
}

// ── Card ────────────────────────────────────────────────────────────────────

func renderCard(issue redmine.Issue, width int, selected bool) string {
	inner := width - 4
	if inner < 8 {
		inner = 8
	}
	body := inner - 2 // one cell for the priority rail plus its space

	rail := lipgloss.NewStyle().Foreground(priorityColor(issue.Priority.Name)).Render("▌")

	idPrefix := fmt.Sprintf("#%d ", issue.ID)
	title := truncate(issue.Subject, max(4, body-len([]rune(idPrefix))))
	line1 := rail + " " +
		lipgloss.NewStyle().Foreground(colorMuted).Render(idPrefix) +
		lipgloss.NewStyle().Bold(selected).Foreground(colorText).Render(title)

	line2 := rail + " " +
		styleProjectTag.Render(truncate(issue.Project.Name, 12)) + " " +
		priorityStyle(issue.Priority.Name).Render(priorityIcon(issue.Priority.Name)+" "+issue.Priority.Name)

	lines := []string{line1, line2}

	if label, style := humanDue(issue.DueDate); label != "" {
		lines = append(lines, rail+" "+style.Render(label))
	}
	if issue.DoneRatio > 0 {
		barW := body - 7
		if barW > 28 {
			barW = 28
		}
		if barW > 3 {
			lines = append(lines, rail+fmt.Sprintf(" %3d%% %s", issue.DoneRatio, progressBar(issue.DoneRatio, barW)))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	if selected {
		return styleCardSelected.Width(width).Render(content)
	}
	return styleCard.Width(width).Render(content)
}
