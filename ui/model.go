package ui

import (
	"fmt"
	"os"
	"os/exec"
	"redmine-tui/config"
	"redmine-tui/i18n"
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

// ── Session state ───────────────────────────────────────────────────────────

type sessionState int

const (
	kanbanView sessionState = iota
	detailView
	statusView
	timeInputView
	searchView
)

// ── Messages ────────────────────────────────────────────────────────────────

type issuesMsg struct {
	issues []redmine.Issue
	loaded bool
}

// clearStatusMsg is dispatched after a delay to wipe a transient status line.
type clearStatusMsg struct{ token int }

// ── Model ───────────────────────────────────────────────────────────────────

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
	statusToken   int
}

type kanbanColumn struct {
	status redmine.IssueStatus
	issues []redmine.Issue
}

func NewModel(cfg *config.Config) Model {
	i18n.SetLang(cfg.Language)

	client := redmine.NewClient(cfg.APIKey, cfg.Host)

	ti := textinput.New()
	ti.Placeholder = i18n.T("time.ph.hours")
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40
	ti.Prompt = lipgloss.NewStyle().Foreground(colorAccent).Render("❯ ")

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

// setStatus records a transient status message and returns a cmd that clears it.
func (m *Model) setStatus(msg string, isErr bool) tea.Cmd {
	m.statusMsg = msg
	m.statusIsError = isErr
	m.statusToken++
	token := m.statusToken
	delay := 4 * time.Second
	if isErr {
		delay = 6 * time.Second
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return clearStatusMsg{token: token}
	})
}

// ── Update ──────────────────────────────────────────────────────────────────

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
			m.textInput.Placeholder = i18n.T("search.ph")
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
					m.textInput.Placeholder = i18n.T("time.ph.hours")
					m.textInput.Width = 20
					m.textInput.Focus()
					return m, textinput.Blink
				}
			case "y":
				if m.selected != nil {
					clipboard.WriteAll(m.selected.Subject)
					return m, m.setStatus(i18n.T("msg.copied"), false)
				}
			case "o":
				if m.selected != nil {
					openBrowser(fmt.Sprintf("%s/issues/%d", m.client.Host, m.selected.ID))
					return m, m.setStatus(i18n.T("msg.opening"), false)
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
					hours := strings.TrimSpace(m.textInput.Value())
					if !validHours(hours) {
						return m, m.setStatus("Invalid hours — enter a number (e.g. 1.5)", true)
					}
					m.logHours = hours
					m.inputState = 1
					m.textInput.Reset()
					m.textInput.Placeholder = i18n.T("time.ph.comments")
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
					m.viewport.SetContent(i18n.T("app.loading"))
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
					m.textInput.Placeholder = i18n.T("time.ph.hours")
					m.textInput.Width = 20
					m.textInput.Focus()
					return m, textinput.Blink
				}
			case "y":
				issue := m.selectedIssue()
				if issue != nil {
					clipboard.WriteAll(issue.Subject)
					return m, m.setStatus(i18n.T("msg.copied"), false)
				}
			case "o":
				issue := m.selectedIssue()
				if issue != nil {
					openBrowser(fmt.Sprintf("%s/issues/%d", m.client.Host, issue.ID))
					return m, m.setStatus(i18n.T("msg.opening"), false)
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
					return m, m.setStatus(i18n.T("msg.filter.all"), false)
				}
				return m, m.setStatus(fmt.Sprintf(i18n.T("msg.filter"), m.activeFilter), false)
			case "r":
				return m, tea.Batch(m.fetchIssues, m.fetchStatuses)
			case "e":
				if err := m.exportToHTML(); err != nil {
					return m, m.setStatus(fmt.Sprintf("Export error: %v", err), true)
				}
				return m, m.setStatus(i18n.T("msg.exported"), false)
			case "v":
				return m, tea.Batch(
					m.setStatus(i18n.T("msg.syncing"), false),
					m.syncVikunja(),
				)
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

	case clearStatusMsg:
		if msg.token == m.statusToken {
			m.statusMsg = ""
			m.statusIsError = false
		}

	case string:
		switch msg {
		case "status_updated":
			return m, tea.Batch(
				m.setStatus(i18n.T("msg.status_updated"), false),
				m.fetchIssues, m.fetchStatuses,
			)
		case "time_logged":
			return m, m.setStatus(i18n.T("msg.time_logged"), false)
		case "vikunja_synced":
			return m, m.setStatus(i18n.T("msg.vikunja_synced"), false)
		default:
			return m, m.setStatus(msg, false)
		}

	case error:
		// Non-fatal errors go to status bar; fatal startup errors quit
		if !m.loaded {
			m.err = msg
			return m, tea.Quit
		}
		return m, m.setStatus(msg.Error(), true)
	}

	return m, cmd
}

// ── Kanban helpers ──────────────────────────────────────────────────────────

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

func validHours(s string) bool {
	if s == "" {
		return false
	}
	dot := false
	hasDigit := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '.' || r == ',':
			if dot {
				return false
			}
			dot = true
		default:
			return false
		}
	}
	return hasDigit
}

// ── View dispatch ───────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.err != nil {
		return styleDetailPanel.Render(
			styleDetailTitle.Render(i18n.T("error")) + "\n\n" +
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

// ── Commands ────────────────────────────────────────────────────────────────

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

// ── Export ──────────────────────────────────────────────────────────────────

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
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Redmine Issues</title>
<style>
:root{--bg:#13131f;--surface:#1c1c2e;--card:#23233a;--border:#2e2e48;--accent:#a29bfe;--text:#e8e8f4;--muted:#6c6c8a}
*{box-sizing:border-box}
body{font-family:ui-sans-serif,system-ui,sans-serif;margin:0;padding:32px;background:var(--bg);color:var(--text)}
h2{color:var(--accent);font-weight:700;letter-spacing:-.02em}
.board{display:flex;gap:18px;overflow-x:auto;padding-bottom:16px}
.col{background:var(--surface);border:1px solid var(--border);border-radius:14px;padding:16px;min-width:300px;flex-shrink:0}
.col h3{margin:0 0 14px;font-size:13px;text-transform:uppercase;letter-spacing:.05em;color:var(--accent);display:flex;justify-content:space-between;align-items:center}
.col h3 span{background:var(--card);color:var(--muted);font-weight:600;padding:2px 8px;border-radius:8px;font-size:12px}
.card{background:var(--card);border:1px solid var(--border);border-radius:10px;padding:12px;margin-bottom:10px;transition:.15s}
.card:hover{border-color:var(--accent);transform:translateY(-1px);box-shadow:0 6px 20px rgba(0,0,0,.25)}
.card h4{margin:0 0 8px;font-size:13px;font-weight:600}
.card h4 .id{color:var(--muted);font-weight:400}
.tag{display:inline-block;padding:2px 8px;border-radius:6px;font-size:11px;color:#fff;background:var(--accent);margin-right:6px;font-weight:600}
.prio{font-size:12px;font-weight:600}
.prio-urgent{color:#f87171}.prio-high{color:#fbbf24}.prio-normal{color:#60a5fa}.prio-low{color:#6c6c8a}
.due{font-size:11px;color:var(--muted);margin-top:6px}
.due.overdue{color:#f87171;font-weight:600}
.progress{height:5px;background:var(--border);border-radius:99px;margin-top:8px;overflow:hidden}
.progress-fill{height:100%;background:var(--accent);border-radius:99px}
</style>
</head>
<body>
<h2>◈ Redmine Kanban</h2>
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
		html += fmt.Sprintf(`<div class="col"><h3>%s <span>%d</span></h3>`, status, len(issues))
		for _, issue := range issues {
			pClass := "prio-low"
			switch classifyPriority(issue.Priority.Name) {
			case prioUrgent:
				pClass = "prio-urgent"
			case prioHigh:
				pClass = "prio-high"
			case prioNormal:
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
  <h4><span class="id">#%d</span> %s</h4>
  <span class="tag">%s</span><span class="prio %s">%s %s</span>
  %s%s
</div>`, issue.ID, issue.Subject, issue.Project.Name, pClass, priorityIcon(issue.Priority.Name), issue.Priority.Name, due, bar)
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
