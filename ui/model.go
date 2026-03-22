package ui

import (
	"fmt"
	"os"
	"os/exec"
	"redmine-tui/config"
	"redmine-tui/redmine"
	"redmine-tui/sync"
	"redmine-tui/vikunja"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ──────────────────────────────────────────────────────────────────────────────

var (
	colorBg      = lipgloss.AdaptiveColor{Light: "#f5f5f5", Dark: "#1a1a2e"}
	colorSurface = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#16213e"}
	colorBorder  = lipgloss.AdaptiveColor{Light: "#d0d0d0", Dark: "#2a2a4a"}
	colorAccent  = lipgloss.AdaptiveColor{Light: "#6c63ff", Dark: "#7c6fff"}
	colorText    = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#e8e8f0"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "#888899", Dark: "#6060a0"}
	colorSuccess = lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"}
	colorWarn    = lipgloss.AdaptiveColor{Light: "#f59e0b", Dark: "#fbbf24"}
	colorDanger  = lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"}
	colorInfo    = lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"}
	colorPurple  = lipgloss.AdaptiveColor{Light: "#8b5cf6", Dark: "#a78bfa"}

	styleColumnHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText).
				Background(colorSurface).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(colorAccent).
				PaddingLeft(1).
				PaddingRight(1)

	styleColumnHeaderActive = styleColumnHeader.
				Foreground(colorAccent).
				Bold(true)

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
			PaddingRight(1)

	styleCardSelected = styleCard.
				BorderForeground(colorAccent).
				Background(lipgloss.AdaptiveColor{Light: "#f0eeff", Dark: "#1e1a40"})

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

	styleTag = lipgloss.NewStyle().
			Foreground(colorSurface).
			Background(colorAccent).
			PaddingLeft(1).
			PaddingRight(1)

	styleProjectTag = lipgloss.NewStyle().
			Foreground(colorSurface).
			Background(colorInfo).
			PaddingLeft(1).
			PaddingRight(1)

	styleCountBadge = lipgloss.NewStyle().
			Foreground(colorSurface).
			Background(colorMuted).
			PaddingLeft(1).
			PaddingRight(1)

	stylePriorityUrgent = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	stylePriorityHigh   = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	stylePriorityNormal = lipgloss.NewStyle().Foreground(colorInfo)
	stylePriorityLow    = lipgloss.NewStyle().Foreground(colorMuted)

	styleOverdue  = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	styleDueToday = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	styleDueSoon  = lipgloss.NewStyle().Foreground(colorInfo)
)

// ── Priority / due helpers ─────────────────────────────────────────────────────

func priorityStyle(name string) lipgloss.Style {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(n, "kritik"):
		return stylePriorityUrgent
	case strings.Contains(n, "yüksek") || strings.Contains(n, "high"):
		return stylePriorityHigh
	case n == "normal" || strings.Contains(n, "orta"):
		return stylePriorityNormal
	case strings.Contains(n, "düşük") || strings.Contains(n, "low") || strings.Contains(n, "belirsiz"):
		return stylePriorityLow
	default:
		return stylePriorityLow
	}
}

func priorityIcon(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(n, "kritik"):
		return "▲▲"
	case strings.Contains(n, "yüksek") || strings.Contains(n, "high"):
		return "▲"
	case n == "normal" || strings.Contains(n, "orta"):
		return "●"
	case strings.Contains(n, "düşük") || strings.Contains(n, "low") || strings.Contains(n, "belirsiz"):
		return "▽"
	default:
		return "▽"
	}
}

func trackerIcon(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bug", "hata":
		return "🐞 "
	case "feature", "özellik", "yeni özellik":
		return "✨ "
	case "support", "destek":
		return "🛋 "
	case "task", "görev":
		return "☑ "
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
		return "⏰ Bugün", styleDueToday
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
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	color := colorSuccess
	if ratio < 30 {
		color = colorMuted
	} else if ratio < 70 {
		color = colorInfo
	}
	return lipgloss.NewStyle().Foreground(color).Render(bar)
}

// ── Session state ───────────────────────────────────────────────────────────────────

type sessionState int

const (
	kanbanView sessionState = iota
	detailView
	statusView
	timeInputView
	searchView
)

// ── Model ──────────────────────────────────────────────────────────────────────────────

type Model struct {
	client     *redmine.Client
	cfg        *config.Config
	state      sessionState
	inputState int

	allIssues []redmine.Issue
	statuses  []redmine.IssueStatus
	selected  *redmine.Issue
	loaded    bool
	err       error

	columns []kanbanColumn
	colIdx  int
	cardIdx int

	activeFilter string

	textInput textinput.Model
	logHours  string
	viewport  viewport.Model

	windowWidth  int
	windowHeight int

	statusMsg     string
	statusIsError bool
}

type kanbanColumn struct {
	status redmine.IssueStatus
	issues []redmine.Issue
}

type issuesMsg struct {
	issues []redmine.Issue
	loaded bool
}

func NewModel(cfg *config.Config) Model {
	client := redmine.NewClient(cfg.APIKey, cfg.Host)

	ti := textinput.New()
	ti.Placeholder = "Enter hours (e.g. 1.5)"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	vp := viewport.New(0, 0)
	vp.Style = styleDetailPanel

	return Model{
		client:    client,
		cfg:       cfg,
		state:     kanbanView,
		textInput: ti,
		viewport:  vp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchIssues, m.fetchStatuses, textinput.Blink)
}

// ── Update ──────────────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+f" {
			m.state = searchView
			m.inputState = 2
			m.textInput.Reset()
			m.textInput.Placeholder = "Search issues..."
			m.textInput.Width = 50
			m.textInput.Focus()
			return m, textinput.Blink
		}

		switch m.state {
		case detailView:
			switch msg.String() {
			case "esc", "q":
				m.state = kanbanView
			case "s":
				if m.selected != nil {
					for i, s := range m.statuses {
						if s.ID == m.selected.Status.ID {
							m.colIdx = i
							break
						}
					}
					m.state = statusView
				}
			case "t":
				if m.selected != nil {
					m.state = timeInputView
					m.inputState = 0
					m.textInput.Reset()
					m.textInput.Placeholder = "Enter hours (e.g. 1.5)"
					m.textInput.Width = 20
					m.textInput.Focus()
					return m, textinput.Blink
				}
			case "y":
				if m.selected != nil {
					clipboard.WriteAll(m.selected.Subject)
					m.statusMsg = "Copied to clipboard!"
				}
			default:
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
			return m, nil

		case statusView:
			switch msg.String() {
			case "esc":
				if m.selected != nil {
					m.state = detailView
				} else {
					m.state = kanbanView
				}
				return m, nil
			case "enter":
				if len(m.statuses) > 0 && m.colIdx < len(m.statuses) {
					st := m.statuses[m.colIdx]
					issueID := 0
					if m.selected != nil {
						issueID = m.selected.ID
					}
					if issueID != 0 {
						m.state = kanbanView
						return m, m.updateStatus(issueID, st.ID)
					}
				}
			case "left", "h":
				if m.colIdx > 0 {
					m.colIdx--
				}
			case "right", "l":
				if m.colIdx < len(m.statuses)-1 {
					m.colIdx++
				}
			}
			return m, nil

		case timeInputView:
			switch msg.String() {
			case "esc":
				m.state = detailView
				m.textInput.Reset()
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				if m.inputState == 0 {
					m.logHours = m.textInput.Value()
					m.inputState = 1
					m.textInput.Reset()
					m.textInput.Placeholder = "Enter comments..."
					m.textInput.Width = 50
					return m, nil
				}
				comments := m.textInput.Value()
				issueID := 0
				if m.selected != nil {
					issueID = m.selected.ID
				}
				if issueID != 0 {
					m.state = detailView
					m.textInput.Reset()
					return m, m.logTime(issueID, m.logHours, comments)
				}
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case searchView:
			switch msg.String() {
			case "esc":
				m.state = kanbanView
				m.textInput.Reset()
				m.rebuildColumns(m.allIssues)
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				query := m.textInput.Value()
				m.state = kanbanView
				m.textInput.Reset()
				return m, m.searchIssues(query)
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case kanbanView:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "left", "h":
				if m.colIdx > 0 {
					m.colIdx--
					m.cardIdx = 0
				}
			case "right", "l":
				if m.colIdx < len(m.columns)-1 {
					m.colIdx++
					m.cardIdx = 0
				}
			case "up", "k":
				if m.cardIdx > 0 {
					m.cardIdx--
				}
			case "down", "j":
				col := m.currentColumn()
				if col != nil && m.cardIdx < len(col.issues)-1 {
					m.cardIdx++
				}
			case "enter", " ":
				issue := m.selectedIssue()
				if issue != nil {
					m.selected = issue
					m.state = detailView
					m.viewport.SetContent("Loading details...")
					m.viewport.GotoTop()
					return m, m.fetchIssueDetails(issue.ID)
				}
			case "s":
				issue := m.selectedIssue()
				if issue != nil {
					m.selected = issue
					for i, s := range m.statuses {
						if s.ID == issue.Status.ID {
							m.colIdx = i
							break
						}
					}
					m.state = statusView
				}
			case "t":
				issue := m.selectedIssue()
				if issue != nil {
					m.selected = issue
					m.state = timeInputView
					m.inputState = 0
					m.textInput.Reset()
					m.textInput.Placeholder = "Enter hours (e.g. 1.5)"
					m.textInput.Width = 20
					m.textInput.Focus()
					return m, textinput.Blink
				}
			case "y":
				issue := m.selectedIssue()
				if issue != nil {
					clipboard.WriteAll(issue.Subject)
					m.statusMsg = "Copied to clipboard!"
				}
			case "f":
				projects := m.uniqueProjects()
				if len(projects) == 0 {
					return m, nil
				}
				cur := m.activeFilter
				idx := -1
				for i, p := range projects {
					if p == cur {
						idx = i
						break
					}
				}
				if idx == -1 || idx >= len(projects)-1 {
					m.activeFilter = ""
				} else {
					m.activeFilter = projects[idx+1]
				}
				m.applyFilter()
				m.colIdx = 0
				m.cardIdx = 0
				if m.activeFilter == "" {
					m.statusMsg = "Filter: All projects"
				} else {
					m.statusMsg = fmt.Sprintf("Filter: %s", m.activeFilter)
				}
			case "r":
				return m, tea.Batch(m.fetchIssues, m.fetchStatuses)
			case "e":
				err := m.exportToHTML()
				if err != nil {
					m.statusMsg = fmt.Sprintf("Export error: %v", err)
				} else {
					m.statusMsg = "Exported to redmine_issues.html"
				}
			case "v":
				m.statusMsg = "Syncing with Vikunja..."
				return m, m.syncVikunja()
			}
		}

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.viewport.Width = msg.Width - 8
		m.viewport.Height = msg.Height - 10

	case issuesMsg:
		m.allIssues = msg.issues
		m.loaded = true
		if m.activeFilter != "" {
			m.applyFilter()
		} else {
			m.rebuildColumns(msg.issues)
		}

	case []redmine.IssueStatus:
		m.statuses = msg
		m.rebuildColumns(m.allIssues)

	case *redmine.Issue:
		m.selected = msg
		m.viewport.SetContent(renderDetail(msg, m.viewport.Width))
		m.viewport.GotoTop()

	case string:
		switch msg {
		case "status_updated":
			m.statusMsg = "Status updated!"
			m.statusIsError = false
			return m, tea.Batch(m.fetchIssues, m.fetchStatuses)
		case "time_logged":
			m.statusMsg = "Time logged!"
			m.statusIsError = false
		case "vikunja_synced":
			m.statusMsg = "Synced with Vikunja!"
			m.statusIsError = false
		default:
			m.statusMsg = msg
			m.statusIsError = false
		}

	case error:
		// Non-fatal errors go to status bar; fatal startup errors quit
		if !m.loaded {
			m.err = msg
			return m, tea.Quit
		}
		m.statusMsg = msg.Error()
		m.statusIsError = true
	}

	return m, cmd
}

// ── Kanban helpers ───────────────────────────────────────────────────────────────────

func (m *Model) rebuildColumns(issues []redmine.Issue) {
	if len(m.statuses) == 0 {
		seen := map[int]bool{}
		cols := []kanbanColumn{}
		for _, issue := range issues {
			if !seen[issue.Status.ID] {
				seen[issue.Status.ID] = true
				cols = append(cols, kanbanColumn{
					status: redmine.IssueStatus{ID: issue.Status.ID, Name: issue.Status.Name},
				})
			}
		}
		for i := range cols {
			for _, issue := range issues {
				if issue.Status.ID == cols[i].status.ID {
					cols[i].issues = append(cols[i].issues, issue)
				}
			}
		}
		m.columns = cols
		return
	}

	cols := make([]kanbanColumn, len(m.statuses))
	for i, st := range m.statuses {
		cols[i] = kanbanColumn{status: st}
		for _, issue := range issues {
			if issue.Status.ID == st.ID {
				cols[i].issues = append(cols[i].issues, issue)
			}
		}
	}
	nonEmpty := []kanbanColumn{}
	for _, col := range cols {
		if len(col.issues) > 0 {
			nonEmpty = append(nonEmpty, col)
		}
	}
	m.columns = nonEmpty
	if m.colIdx >= len(m.columns) {
		m.colIdx = max(0, len(m.columns)-1)
	}
}

func (m *Model) applyFilter() {
	filtered := m.allIssues
	if m.activeFilter != "" {
		filtered = []redmine.Issue{}
		for _, issue := range m.allIssues {
			if issue.Project.Name == m.activeFilter {
				filtered = append(filtered, issue)
			}
		}
	}
	m.rebuildColumns(filtered)
}

func (m *Model) uniqueProjects() []string {
	seen := map[string]bool{}
	out := []string{}
	for _, issue := range m.allIssues {
		if !seen[issue.Project.Name] {
			seen[issue.Project.Name] = true
			out = append(out, issue.Project.Name)
		}
	}
	return out
}

func (m *Model) currentColumn() *kanbanColumn {
	if len(m.columns) == 0 || m.colIdx >= len(m.columns) {
		return nil
	}
	return &m.columns[m.colIdx]
}

func (m *Model) selectedIssue() *redmine.Issue {
	col := m.currentColumn()
	if col == nil || len(col.issues) == 0 || m.cardIdx >= len(col.issues) {
		return nil
	}
	return &col.issues[m.cardIdx]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── View ────────────────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.err != nil {
		return styleDetailPanel.Render(
			styleDetailTitle.Render("Error") + "\n\n" +
				lipgloss.NewStyle().Foreground(colorDanger).Render(m.err.Error()),
		)
	}
	switch m.state {
	case detailView:
		return m.viewDetail()
	case statusView:
		return m.viewStatusPicker()
	case timeInputView:
		return m.viewTimeInput()
	case searchView:
		return m.viewSearch()
	default:
		return m.viewKanban()
	}
}

// ── Kanban view ────────────────────────────────────────────────────────────────────────

func (m Model) viewKanban() string {
	if !m.loaded {
		return lipgloss.Place(
			m.windowWidth, m.windowHeight,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(colorAccent).Render("⦾")+" Loading issues...",
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

	// header=2 lines (bar+divider), footer=2-3 lines (divider+help[+status])
	// use fixed offsets to avoid ANSI-width mis-measurement
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
	titleText := "◈ Redmine Kanban"
	totalText := fmt.Sprintf("%d issues", len(m.allIssues))

	// Render styled parts
	titleStyled := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(titleText)
	filterStyled := ""
	if m.activeFilter != "" {
		filterStyled = "  " + styleProjectTag.Render(m.activeFilter)
	}
	totalStyled := lipgloss.NewStyle().Foreground(colorMuted).Render(totalText)

	leftStyled := titleStyled + filterStyled

	// Compute ANSI-aware widths for gap (inner area = w-2, since Padding(0,1) adds 2)
	inner := w - 2
	if inner < 4 {
		inner = 4
	}
	leftW := lipgloss.Width(leftStyled)
	rightW := lipgloss.Width(totalStyled)
	gap := inner - leftW - rightW
	if gap < 1 {
		// Not enough room — drop the right-side counter rather than overflow
		gap = 0
		totalStyled = ""
		rightW = 0
		gap = inner - leftW
		if gap < 0 {
			gap = 0
		}
	}

	content := leftStyled + strings.Repeat(" ", gap) + totalStyled

	// Width(w-2): Padding(0,1) adds 2, so final rendered width = w
	bar := lipgloss.NewStyle().
		Background(colorSurface).
		Padding(0, 1).
		Width(w - 2).
		Render(content)

	divider := lipgloss.NewStyle().
		Foreground(colorAccent).
		Render(strings.Repeat("-", w))

	return lipgloss.JoinVertical(lipgloss.Left, bar, divider)
}
func (m Model) renderFooter(w int) string {
	keys := []struct{ k, d string }{
		{"←→", "col"}, {"↑↓", "card"}, {"↵", "detail"},
		{"s", "status"}, {"t", "time"}, {"f", "filter"},
		{"y", "copy"}, {"r", "refresh"}, {"e", "export"},
		{"v", "vikunja"}, {"^f", "search"}, {"q", "quit"},
	}
	parts := []string{}
	for _, kv := range keys {
		parts = append(parts, styleHelpKey.Render(kv.k)+" "+styleHelpDesc.Render(kv.d))
	}
	help := strings.Join(parts, styleHelpDesc.Render(" · "))

	divider := lipgloss.NewStyle().
		Foreground(colorBorder).
		Render(strings.Repeat("-", w))

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
			lipgloss.NewStyle().Foreground(colorMuted).Render("No issues found"))
	}

	// colOverhead: RoundedBorder(left+right=2) + PaddingLeft+Right(2) = 4
	const colOverhead = 4
	// minColContent is the narrowest usable card content area
	const minColContent = 24

	numCols := len(m.columns)

	// How many columns actually fit?
	// Each col takes (colContent + colOverhead) columns.
	// Start with equal distribution, then clamp.
	colContent := (w - numCols*colOverhead) / numCols
	if colContent < minColContent {
		// Try fewer visible columns
		for numCols > 1 && colContent < minColContent {
			numCols--
			colContent = (w - numCols*colOverhead) / numCols
		}
		if colContent < minColContent {
			colContent = minColContent
		}
	}

	// Visible column window: center around active column
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
		headerText := col.status.Name + " " + count
		var header string
		if active {
			header = styleColumnHeaderActive.Width(headerInner).Render(headerText)
		} else {
			header = styleColumnHeader.Width(headerInner).Render(headerText)
		}

		// Available card area = board height minus this column's real header height.
		availH := h - lipgloss.Height(header)
		if availH < 1 {
			availH = 1
		}

		// Pre-render all cards to know their heights.
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

		// Find the scroll window: binary-search for the smallest cardStart
		// such that cards[cardStart..cardIdx] fit within availH.
		cardStart := 0
		if active && m.cardIdx < len(all) {
			// Walk backwards from cardIdx, accumulating height.
			usedH := 0
			for i := m.cardIdx; i >= 0; i-- {
				usedH += all[i].height
				if usedH > availH {
					cardStart = i + 1
					break
				}
			}
		}

		// Fill forward from cardStart until availH is exhausted.
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
			// At minimum show one card even if it overflows.
			cardEnd = cardStart + 1
		}

		cards := []string{header}
		if cardStart > 0 {
			cards = append(cards, lipgloss.NewStyle().Foreground(colorMuted).Render("  ↑ more"))
		}
		for i := cardStart; i < cardEnd; i++ {
			cards = append(cards, all[i].rendered)
		}
		if cardEnd < len(col.issues) {
			cards = append(cards, lipgloss.NewStyle().Foreground(colorMuted).Render("  ↓ more"))
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

	// If total visible columns < total columns, show navigation hint
	if len(m.columns) > len(visibleCols) {
		hint := fmt.Sprintf(" col %d/%d  ←→ to navigate ", m.colIdx+1, len(m.columns))
		hintLine := lipgloss.NewStyle().Foreground(colorMuted).Render(hint)
		board = lipgloss.JoinVertical(lipgloss.Left, board, hintLine)
	}

	return board
}

func renderCard(issue redmine.Issue, width int, selected bool) string {
	// width = colContent-4 (caller subtracts column border+padding from colContent)
	// card adds its own RoundedBorder(2) + PaddingLeft+Right(2) = 4
	// so text area = width - 4, and card rendered width = width + 4 = colContent
	inner := width - 4
	if inner < 6 {
		inner = 6
	}

	// Title line: truncate to leave room, prefix with #ID
	idPrefix := fmt.Sprintf("#%d ", issue.ID)
	titleMax := inner - len(idPrefix)
	if titleMax < 4 {
		titleMax = 4
	}
	title := issue.Subject
	if len(title) > titleMax {
		title = title[:titleMax-1] + "…"
	}
	idStr := lipgloss.NewStyle().Foreground(colorMuted).Render(idPrefix)
	titleStr := lipgloss.JoinHorizontal(lipgloss.Top,
		idStr,
		lipgloss.NewStyle().Bold(selected).Foreground(colorText).Render(title),
	)

	projStr := styleProjectTag.Render(func() string {
		p := issue.Project.Name
		if len(p) > 12 {
			p = p[:11] + "…"
		}
		return p
	}())

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

// ── Detail view ────────────────────────────────────────────────────────────────────────

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
		Render("◈ Issue Detail  " +
			styleHelpDesc.Render("esc/q: back · s: status · t: time · y: copy · ↑↓: scroll"))

	divider := lipgloss.NewStyle().Foreground(colorAccent).Render(strings.Repeat("-", w))

	m.viewport.Width = w - 8
	m.viewport.Height = m.windowHeight - 6
	m.viewport.SetContent(renderDetail(m.selected, m.viewport.Width))

	return lipgloss.JoinVertical(lipgloss.Left,
		header, divider,
		lipgloss.NewStyle().Padding(0, 2).Render(m.viewport.View()),
	)
}

func renderDetail(issue *redmine.Issue, width int) string {
	label := styleDetailLabel.Render
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
			label("Project:"), styleProjectTag.Render(issue.Project.Name),
			label("Status:"), styleTag.Render(issue.Status.Name),
			label("Tracker:"), val(trackerIcon(issue.Tracker.Name)+issue.Tracker.Name),
		),
		fmt.Sprintf("%s %s    %s %s    %s %s",
			label("Priority:"), prio,
			label("Due:"), due,
			label("Author:"), val(issue.Author.Name),
		),
		fmt.Sprintf("%s %s    %s %s",
			label("Assignee:"), assignee,
			label("Created:"), val(issue.CreatedOn.Format("2006-01-02 15:04")),
		),
		fmt.Sprintf("%s %s",
			label("Updated:"), val(issue.UpdatedOn.Format("2006-01-02 15:04")),
		),
	)
	if bar != "" {
		meta += "\n" + label("Progress: ") + bar
	}

	divider := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("-", width-4))

	// Strip Windows-style CRLF from Redmine API responses
	cleanDesc := strings.ReplaceAll(issue.Description, "\r\n", "\n")
	cleanDesc = strings.ReplaceAll(cleanDesc, "\r", "\n")
	desc := lipgloss.NewStyle().Foreground(colorText).Render(cleanDesc)
	if issue.Description == "" {
		desc = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("(no description)")
	}

	sections := []string{title, meta, divider, label("Description:") + "\n" + desc}

	hasJournals := false
	for _, j := range issue.Journals {
		if j.Notes != "" {
			hasJournals = true
			break
		}
	}
	if hasJournals {
		sections = append(sections, divider, label("History:"))
		for _, j := range issue.Journals {
			if j.Notes == "" {
				continue
			}
			// Clean CRLF, then render header and body independently
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

// ── Status picker ─────────────────────────────────────────────────────────────────

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
		Render("◈ Change Status  " + styleHelpDesc.Render("←→: select · enter: apply · esc: back"))
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

// ── Time input ─────────────────────────────────────────────────────────────────────

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
		Render("◈ Log Time  " + styleHelpDesc.Render("enter: next · esc: back"))
	divider := lipgloss.NewStyle().Foreground(colorInfo).Render(strings.Repeat("─", w))

	step := "Step 1/2: Hours"
	if m.inputState == 1 {
		step = "Step 2/2: Comments"
	}

	panel := styleInputPanel.Render(
		lipgloss.NewStyle().Foreground(colorInfo).Bold(true).Render(step) + "\n\n" +
			m.textInput.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header, divider,
		lipgloss.NewStyle().Padding(2, 4).Render(panel),
	)
}

// ── Search ───────────────────────────────────────────────────────────────────────────

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
		Render("◈ Search Issues  " + styleHelpDesc.Render("enter: search · esc: cancel"))
	divider := lipgloss.NewStyle().Foreground(colorWarn).Render(strings.Repeat("─", w))

	panel := styleInputPanel.
		BorderForeground(colorWarn).
		Render(
			lipgloss.NewStyle().Foreground(colorWarn).Bold(true).Render("Search description:") + "\n\n" +
				m.textInput.View(),
		)

	return lipgloss.JoinVertical(lipgloss.Left,
		header, divider,
		lipgloss.NewStyle().Padding(2, 4).Render(panel),
	)
}

// ── Commands ────────────────────────────────────────────────────────────────────────

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

func (m Model) updateStatus(issueID, statusID int) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.UpdateIssueStatus(issueID, statusID); err != nil {
			return err
		}
		return "status_updated"
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

// ── Export ────────────────────────────────────────────────────────────────────────────

func (m Model) exportToHTML() error {
	f, err := os.Create("redmine_issues.html")
	if err != nil {
		return err
	}
	defer f.Close()

	html := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Redmine Issues</title>
<style>
body{font-family:sans-serif;padding:20px;background:#1a1a2e;color:#e8e8f0}
.board{display:flex;gap:16px;overflow-x:auto;padding-bottom:16px}
.col{background:#16213e;border-radius:8px;padding:12px;min-width:280px;flex-shrink:0}
.col h3{margin:0 0 12px;font-size:14px;color:#7c6fff;border-bottom:2px solid #7c6fff;padding-bottom:6px}
.card{background:#0f3460;border:1px solid #2a2a4a;border-radius:6px;padding:10px;margin-bottom:8px;cursor:pointer;transition:box-shadow .15s}
.card:hover{box-shadow:0 2px 8px rgba(124,111,255,.3);border-color:#7c6fff}
.card h4{margin:0 0 6px;font-size:13px;color:#e8e8f0}
.tag{display:inline-block;padding:1px 6px;border-radius:3px;font-size:11px;color:#fff;background:#3b82f6;margin-right:4px}
.prio-urgent{color:#f87171;font-weight:bold}.prio-high{color:#fbbf24;font-weight:bold}
.prio-normal{color:#60a5fa}.prio-low{color:#6060a0}
.due{font-size:11px;color:#6060a0;margin-top:4px}
.due.overdue{color:#f87171;font-weight:bold}
.progress{height:4px;background:#2a2a4a;border-radius:2px;margin-top:6px}
.progress-fill{height:100%;background:#7c6fff;border-radius:2px}
</style>
</head>
<body>
<h2 style="color:#7c6fff">Redmine Kanban</h2>
<div class="board">
`

	cols := map[string][]redmine.Issue{}
	order := []string{}
	for _, issue := range m.allIssues {
		if _, ok := cols[issue.Status.Name]; !ok {
			order = append(order, issue.Status.Name)
		}
		cols[issue.Status.Name] = append(cols[issue.Status.Name], issue)
	}
	for _, status := range order {
		issues := cols[status]
		html += fmt.Sprintf(`<div class="col"><h3>%s <span style="color:#6060a0;font-weight:normal">(%d)</span></h3>`, status, len(issues))
		for _, issue := range issues {
			pClass := "prio-low"
			switch strings.ToLower(issue.Priority.Name) {
			case "acil", "urgent":
				pClass = "prio-urgent"
			case "yüksek", "high":
				pClass = "prio-high"
			case "normal":
				pClass = "prio-normal"
			}
			due := ""
			if issue.DueDate != "" {
				dueClass := "due"
				t, err := time.Parse("2006-01-02", issue.DueDate)
				if err == nil && t.Before(time.Now()) {
					dueClass = "due overdue"
				}
				due = fmt.Sprintf(`<div class="%s">Due: %s</div>`, dueClass, issue.DueDate)
			}
			bar := ""
			if issue.DoneRatio > 0 {
				bar = fmt.Sprintf(`<div class="progress"><div class="progress-fill" style="width:%d%%"></div></div>`, issue.DoneRatio)
			}
			html += fmt.Sprintf(`
<div class="card">
  <h4>#%d %s</h4>
  <span class="tag">%s</span><span class="%s">▲ %s</span>
  %s%s
</div>`, issue.ID, issue.Subject, issue.Project.Name, pClass, issue.Priority.Name, due, bar)
		}
		html += "</div>"
	}

	html += `</div></body></html>`
	if _, err := f.WriteString(html); err != nil {
		return err
	}
	return openBrowser("redmine_issues.html")
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
