package ui

import (
	"fmt"
	"redmine-tui/i18n"
	"redmine-tui/redmine"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Kanban view ──────────────────────────────────────────────────────────────

func (m Model) viewKanban() string {
	if !m.loaded {
		return lipgloss.Place(
			m.windowWidth, m.windowHeight,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(colorAccent).Render("◌")+" "+i18n.T("app.loading"),
		)
	}

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
	board := m.renderBoard(w, boardH)

	return lipgloss.JoinVertical(lipgloss.Left, header, board, footer)
}

func (m Model) renderHeader(w int) string {
	titleStyled := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("◈ " + i18n.T("app.title"))
	filterStyled := ""
	if m.activeFilter != "" {
		filterStyled = "  " + styleProjectTag.Render(truncate(m.activeFilter, 24))
	}
	totalText := fmt.Sprintf("%d %s", len(m.allIssues), i18n.T("app.issues"))
	totalStyled := lipgloss.NewStyle().Foreground(colorMuted).Render(totalText)

	leftStyled := titleStyled + filterStyled

	inner := w - 2
	if inner < 4 {
		inner = 4
	}
	leftW := lipgloss.Width(leftStyled)
	rightW := lipgloss.Width(totalStyled)
	gap := inner - leftW - rightW
	if gap < 1 {
		gap = 0
		totalStyled = ""
		gap = inner - leftW
		if gap < 0 {
			gap = 0
		}
	}

	content := leftStyled + strings.Repeat(" ", gap) + totalStyled

	bar := lipgloss.NewStyle().
		Background(colorSurface).
		Padding(0, 1).
		Width(w - 2).
		Render(content)

	divider := lipgloss.NewStyle().
		Foreground(colorAccent).
		Render(strings.Repeat("─", w))

	return lipgloss.JoinVertical(lipgloss.Left, bar, divider)
}

func (m Model) renderFooter(w int) string {
	keys := []struct{ k, d string }{
		{"←→", i18n.T("key.col")}, {"↑↓", i18n.T("key.card")}, {"↵", i18n.T("key.detail")},
		{"s", i18n.T("key.status")}, {"t", i18n.T("key.time")}, {"f", i18n.T("key.filter")},
		{"o", i18n.T("key.open")}, {"y", i18n.T("key.copy")}, {"r", i18n.T("key.refresh")},
		{"e", i18n.T("key.export")}, {"v", i18n.T("key.vikunja")}, {"^f", i18n.T("key.search")},
		{"q", i18n.T("key.quit")},
	}
	parts := []string{}
	for _, kv := range keys {
		parts = append(parts, styleHelpKey.Render(kv.k)+" "+styleHelpDesc.Render(kv.d))
	}
	help := strings.Join(parts, styleHelpDesc.Render(" · "))

	divider := lipgloss.NewStyle().
		Foreground(colorBorder).
		Render(strings.Repeat("─", w))

	lines := []string{divider}
	if m.statusMsg != "" {
		var statusLine string
		if m.statusIsError {
			statusLine = lipgloss.NewStyle().Foreground(colorDanger).PaddingLeft(2).Render("✗ " + m.statusMsg)
		} else {
			statusLine = lipgloss.NewStyle().Foreground(colorSuccess).PaddingLeft(2).Render("✓ " + m.statusMsg)
		}
		lines = append(lines, statusLine)
	}
	lines = append(lines, lipgloss.NewStyle().PaddingLeft(2).Render(help))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderBoard(w, h int) string {
	if len(m.columns) == 0 {
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(colorMuted).Render("⊘ "+i18n.T("app.noissues")))
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
	visibleCols := m.columns[visStart:visEnd]

	cols := []string{}
	for ci, col := range visibleCols {
		globalIdx := visStart + ci
		active := globalIdx == m.colIdx

		count := styleCountBadge.Render(fmt.Sprintf("%d", len(col.issues)))
		headerInner := colContent - 2
		if headerInner < 4 {
			headerInner = 4
		}
		headerText := truncate(col.status.Name, headerInner-4) + " " + count
		var header string
		if active {
			header = styleColumnHeaderActive.Width(headerInner).Render(headerText)
		} else {
			header = styleColumnHeader.Width(headerInner).Render(headerText)
		}

		availH := h - lipgloss.Height(header)
		if availH < 1 {
			availH = 1
		}

		type cardEntry struct {
			rendered string
			height   int
		}
		all := make([]cardEntry, len(col.issues))
		for i, issue := range col.issues {
			selectedCard := active && i == m.cardIdx
			r := renderCard(issue, colContent-4, selectedCard)
			all[i] = cardEntry{r, lipgloss.Height(r)}
		}

		cardStart := 0
		if active && m.cardIdx < len(all) {
			usedH := 0
			for i := m.cardIdx; i >= 0; i-- {
				usedH += all[i].height
				if usedH > availH {
					cardStart = i + 1
					break
				}
			}
		}

		usedH := 0
		cardEnd := cardStart
		for i := cardStart; i < len(all); i++ {
			if usedH+all[i].height > availH {
				break
			}
			usedH += all[i].height
			cardEnd = i + 1
		}
		if cardEnd == cardStart && cardStart < len(all) {
			cardEnd = cardStart + 1
		}

		cards := []string{header}
		if cardStart > 0 {
			cards = append(cards, lipgloss.NewStyle().Foreground(colorMuted).Render("  "+i18n.T("more.up")))
		}
		for i := cardStart; i < cardEnd; i++ {
			cards = append(cards, all[i].rendered)
		}
		if cardEnd < len(col.issues) {
			cards = append(cards, lipgloss.NewStyle().Foreground(colorMuted).Render("  "+i18n.T("more.down")))
		}

		cardContent := lipgloss.JoinVertical(lipgloss.Left, cards...)

		var colStyle lipgloss.Style
		if active {
			colStyle = styleColumnActive.Width(colContent).Height(h).MaxHeight(h)
		} else {
			colStyle = styleColumn.Width(colContent).Height(h).MaxHeight(h)
		}
		cols = append(cols, colStyle.Render(cardContent))
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

	if len(m.columns) > len(visibleCols) {
		hint := " " + fmt.Sprintf(i18n.T("col.nav"), m.colIdx+1, len(m.columns)) + " "
		hintLine := lipgloss.NewStyle().Foreground(colorMuted).Render(hint)
		board = lipgloss.JoinVertical(lipgloss.Left, board, hintLine)
	}

	return board
}

func renderCard(issue redmine.Issue, width int, selected bool) string {
	inner := width - 4
	if inner < 6 {
		inner = 6
	}

	idPrefix := fmt.Sprintf("#%d ", issue.ID)
	titleMax := inner - len([]rune(idPrefix))
	if titleMax < 4 {
		titleMax = 4
	}
	title := truncate(issue.Subject, titleMax)
	idStr := lipgloss.NewStyle().Foreground(colorMuted).Render(idPrefix)
	titleStr := lipgloss.JoinHorizontal(lipgloss.Top,
		idStr,
		lipgloss.NewStyle().Bold(selected).Foreground(colorText).Render(title),
	)

	projStr := styleProjectTag.Render(truncate(issue.Project.Name, 12))

	pStyle := priorityStyle(issue.Priority.Name)
	prioStr := pStyle.Render(priorityIcon(issue.Priority.Name) + " " + issue.Priority.Name)

	dueLabel, dueStyle := dueDateStyle(issue.DueDate)

	line2 := lipgloss.JoinHorizontal(lipgloss.Center, projStr, " ", prioStr)
	if dueLabel != "" {
		line2 = lipgloss.JoinHorizontal(lipgloss.Center, projStr, " ", prioStr, " ", dueStyle.Render(dueLabel))
	}

	lines := []string{titleStr, line2}

	if issue.DoneRatio > 0 {
		barW := inner - 5
		if barW > 0 {
			lines = append(lines, fmt.Sprintf("%3d%% %s", issue.DoneRatio, progressBar(issue.DoneRatio, barW)))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	if selected {
		return styleCardSelected.Width(width).Render(content)
	}
	return styleCard.Width(width).Render(content)
}

// ── Detail view ──────────────────────────────────────────────────────────────

func (m Model) viewDetail() string {
	if m.selected == nil {
		return ""
	}
	w := m.windowWidth
	if w == 0 {
		w = 100
	}

	header := lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorAccent).
		Bold(true).
		Width(w).
		Padding(0, 2).
		Render("◈ " + i18n.T("detail.title") + "  " +
			styleHelpDesc.Render(i18n.T("hint.detail")))

	divider := lipgloss.NewStyle().Foreground(colorAccent).Render(strings.Repeat("─", w))

	m.viewport.Width = w - 8
	m.viewport.Height = m.windowHeight - 6
	m.viewport.SetContent(renderDetail(m.selected, m.viewport.Width))

	return lipgloss.JoinVertical(lipgloss.Left,
		header, divider,
		lipgloss.NewStyle().Padding(0, 2).Render(m.viewport.View()),
	)
}

func renderDetail(issue *redmine.Issue, width int) string {
	label := func(k string) string { return styleDetailLabel.Render(i18n.T(k) + ":") }
	val := lipgloss.NewStyle().Foreground(colorText).Render

	title := styleDetailTitle.Render(fmt.Sprintf("[#%d] %s", issue.ID, issue.Subject))

	pStyle := priorityStyle(issue.Priority.Name)
	prio := pStyle.Render(priorityIcon(issue.Priority.Name) + " " + issue.Priority.Name)

	dueLabel, dueStyle := dueDateStyle(issue.DueDate)
	due := val("—")
	if dueLabel != "" {
		due = dueStyle.Render(dueLabel)
	}

	bar := ""
	if issue.DoneRatio > 0 {
		bar = fmt.Sprintf("%d%% %s", issue.DoneRatio, progressBar(issue.DoneRatio, 20))
	}

	assignee := val("—")
	if issue.AssignedTo.Name != "" {
		assignee = val(issue.AssignedTo.Name)
	}

	meta := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("%s %s    %s %s    %s %s",
			label("label.project"), styleProjectTag.Render(issue.Project.Name),
			label("label.status"), styleTag.Render(issue.Status.Name),
			label("label.tracker"), val(trackerIcon(issue.Tracker.Name)+issue.Tracker.Name),
		),
		fmt.Sprintf("%s %s    %s %s    %s %s",
			label("label.priority"), prio,
			label("label.due"), due,
			label("label.author"), val(issue.Author.Name),
		),
		fmt.Sprintf("%s %s    %s %s",
			label("label.assignee"), assignee,
			label("label.created"), val(issue.CreatedOn.Format("2006-01-02 15:04")),
		),
		fmt.Sprintf("%s %s",
			label("label.updated"), val(issue.UpdatedOn.Format("2006-01-02 15:04")),
		),
	)
	if bar != "" {
		meta += "\n" + styleDetailLabel.Render(i18n.T("label.progress")+": ") + bar
	}

	divider := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", width-4))

	cleanDesc := strings.ReplaceAll(issue.Description, "\r\n", "\n")
	cleanDesc = strings.ReplaceAll(cleanDesc, "\r", "\n")
	desc := lipgloss.NewStyle().Foreground(colorText).Render(cleanDesc)
	if issue.Description == "" {
		desc = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(i18n.T("detail.nodesc"))
	}

	sections := []string{title, meta, divider, styleDetailLabel.Render(i18n.T("label.description")+":") + "\n" + desc}

	hasJournals := false
	for _, j := range issue.Journals {
		if j.Notes != "" {
			hasJournals = true
			break
		}
	}
	if hasJournals {
		sections = append(sections, divider, styleDetailLabel.Render(i18n.T("label.history")+":"))
		for _, j := range issue.Journals {
			if j.Notes == "" {
				continue
			}
			notes := strings.ReplaceAll(j.Notes, "\r\n", "\n")
			notes = strings.ReplaceAll(notes, "\r", "\n")
			who := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(j.User.Name)
			when := lipgloss.NewStyle().Foreground(colorMuted).Render(j.CreatedOn.Format("2006-01-02 15:04"))
			headerLine := who + "  " + when
			noteBody := lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2).Render(notes)
			entry := lipgloss.JoinVertical(lipgloss.Left, headerLine, noteBody)
			sections = append(sections, entry)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// ── Status picker ─────────────────────────────────────────────────────────────

func (m Model) viewStatusPicker() string {
	w := m.windowWidth
	if w == 0 {
		w = 100
	}

	header := lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorPurple).
		Bold(true).
		Width(w).
		Padding(0, 2).
		Render("◈ " + i18n.T("status.title") + "  " + styleHelpDesc.Render(i18n.T("hint.status")))
	divider := lipgloss.NewStyle().Foreground(colorPurple).Render(strings.Repeat("─", w))

	title := ""
	if m.selected != nil {
		title = lipgloss.NewStyle().
			Foreground(colorText).
			PaddingLeft(2).
			Render(fmt.Sprintf("#%d %s", m.selected.ID, m.selected.Subject))
	}

	cards := []string{}
	for i, st := range m.statuses {
		active := i == m.colIdx
		var card string
		if active {
			card = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPurple).
				Foreground(colorPurple).
				Bold(true).
				Padding(1, 2).
				Render("● " + st.Name)
		} else {
			card = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Foreground(colorMuted).
				Padding(1, 2).
				Render("○ " + st.Name)
		}
		cards = append(cards, card)
	}

	board := lipgloss.NewStyle().PaddingLeft(2).PaddingTop(1).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, cards...))

	return lipgloss.JoinVertical(lipgloss.Left, header, divider, title, board)
}

// ── Time input ─────────────────────────────────────────────────────────────────

func (m Model) viewTimeInput() string {
	w := m.windowWidth
	if w == 0 {
		w = 80
	}

	header := lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorInfo).
		Bold(true).
		Width(w).
		Padding(0, 2).
		Render("◈ " + i18n.T("time.title") + "  " + styleHelpDesc.Render(i18n.T("hint.time")))
	divider := lipgloss.NewStyle().Foreground(colorInfo).Render(strings.Repeat("─", w))

	step := i18n.T("time.step1")
	if m.inputState == 1 {
		step = i18n.T("time.step2")
	}

	panel := styleInputPanel.
		BorderForeground(colorInfo).
		Render(
			lipgloss.NewStyle().Foreground(colorInfo).Bold(true).Render(step) + "\n\n" +
				m.textInput.View(),
		)

	return lipgloss.JoinVertical(lipgloss.Left,
		header, divider,
		lipgloss.NewStyle().Padding(2, 4).Render(panel),
	)
}

// ── Search ───────────────────────────────────────────────────────────────────

func (m Model) viewSearch() string {
	w := m.windowWidth
	if w == 0 {
		w = 80
	}

	header := lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorWarn).
		Bold(true).
		Width(w).
		Padding(0, 2).
		Render("◈ " + i18n.T("search.title") + "  " + styleHelpDesc.Render(i18n.T("hint.search")))
	divider := lipgloss.NewStyle().Foreground(colorWarn).Render(strings.Repeat("─", w))

	panel := styleInputPanel.
		BorderForeground(colorWarn).
		Render(
			lipgloss.NewStyle().Foreground(colorWarn).Bold(true).Render(i18n.T("search.label")) + "\n\n" +
				m.textInput.View(),
		)

	return lipgloss.JoinVertical(lipgloss.Left,
		header, divider,
		lipgloss.NewStyle().Padding(2, 4).Render(panel),
	)
}
