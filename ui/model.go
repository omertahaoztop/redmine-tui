package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"redmine-tui/config"
	"redmine-tui/i18n"
	"redmine-tui/redmine"

	"github.com/charmbracelet/bubbles/spinner"
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
	helpView
	paletteView
)

// ── Messages ────────────────────────────────────────────────────────────────

type issuesMsg struct {
	issues []redmine.Issue
	loaded bool
}

// clearStatusMsg is dispatched after a delay to wipe a transient status line.
type clearStatusMsg struct{ token int }

// statusUpdatedMsg carries the new status name so the toast can say where the
// issue went instead of a generic confirmation.
type statusUpdatedMsg struct{ status string }

// ── Sorting ─────────────────────────────────────────────────────────────────

type sortMode int

const (
	sortServer sortMode = iota
	sortPriority
	sortDue
	sortUpdated
	sortProject
)

func (s sortMode) label() string {
	switch s {
	case sortPriority:
		return i18n.T("sort.priority")
	case sortDue:
		return i18n.T("sort.due")
	case sortUpdated:
		return i18n.T("sort.updated")
	case sortProject:
		return i18n.T("sort.project")
	default:
		return i18n.T("sort.none")
	}
}

func (s sortMode) next() sortMode {
	if s >= sortProject {
		return sortServer
	}
	return s + 1
}

// ── Mouse geometry ──────────────────────────────────────────────────────────
// geom hangs off the model by pointer so the value-receiver View can record
// hit regions that handleMouse later reads.

type hitCol struct {
	x, w     int
	headerY0 int
	headerY1 int
}

type hitCard struct {
	col    int
	index  int
	y0, y1 int
}

type geometry struct {
	cols  []hitCol
	cards []hitCard
}

// ── Model ───────────────────────────────────────────────────────────────────

type Model struct {
	client     *redmine.Client
	cfg        *config.Config
	state      sessionState
	returnTo   sessionState
	inputState int

	allIssues []redmine.Issue
	statuses  []redmine.IssueStatus
	selected  *redmine.Issue
	loaded    bool
	loading   bool
	err       error

	columns  []kanbanColumn
	colIdx   int
	cardIdx  int
	followID int

	activeFilter string
	sortMode     sortMode

	textInput    textinput.Model
	paletteInput textinput.Model
	paletteIdx   int

	logHours string
	viewport viewport.Model
	spinner  spinner.Model

	windowWidth  int
	windowHeight int

	statusMsg     string
	statusIsError bool
	statusToken   int

	geom *geometry
}

type kanbanColumn struct {
	status redmine.IssueStatus
	issues []redmine.Issue
}

func NewModel(cfg *config.Config) (Model, error) {
	i18n.SetLang(cfg.Language)

	ti := textinput.New()
	ti.Placeholder = i18n.T("time.ph.hours")
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40
	ti.Prompt = lipgloss.NewStyle().Foreground(colorAccent).Render("❯ ")

	pi := textinput.New()
	pi.Placeholder = i18n.T("palette.ph")
	pi.CharLimit = 64
	pi.Width = 44
	pi.Prompt = lipgloss.NewStyle().Foreground(colorAccent).Render("❯ ")

	vp := viewport.New(0, 0)
	vp.Style = styleDetailPanel

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = lipgloss.NewStyle().Foreground(colorAccent)

	client, err := redmine.NewClient(cfg.APIKey, cfg.Host)
	if err != nil {
		return Model{}, err
	}

	return Model{
		client:       client,
		cfg:          cfg,
		state:        kanbanView,
		loading:      true,
		textInput:    ti,
		paletteInput: pi,
		viewport:     vp,
		spinner:      sp,
		geom:         &geometry{},
	}, nil
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchIssues, m.fetchStatuses, textinput.Blink, m.spinner.Tick)
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
	switch msg := msg.(type) {

	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		return m, cmd

	case tea.MouseMsg:
		cmd := m.handleMouse(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.viewport.Width = msg.Width - 8
		m.viewport.Height = msg.Height - 10

	case issuesMsg:
		m.allIssues = msg.issues
		m.loaded = true
		m.loading = false
		m.applyFilter()

	case []redmine.IssueStatus:
		m.statuses = msg
		m.rebuildColumns(m.filtered())

	case *redmine.Issue:
		m.selected = msg
		m.viewport.SetContent(renderDetail(msg, m.viewport.Width))
		m.viewport.GotoTop()

	case clearStatusMsg:
		if msg.token == m.statusToken {
			m.statusMsg = ""
			m.statusIsError = false
		}

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case statusUpdatedMsg:
		text := i18n.T("msg.status_updated")
		if msg.status != "" {
			text = fmt.Sprintf(i18n.T("msg.moved"), msg.status)
		}
		return m, tea.Batch(m.setStatus(text, false), m.fetchIssues, m.fetchStatuses)

	case string:
		switch msg {
		case "time_logged":
			return m, m.setStatus(i18n.T("msg.time_logged"), false)
		case "vikunja_synced":
			return m, m.setStatus(i18n.T("msg.vikunja_synced"), false)
		default:
			return m, m.setStatus(msg, false)
		}

	case error:
		// Non-fatal errors go to the status bar. A startup failure keeps the
		// error view on screen so the user can fix the config and retry.
		m.loading = false
		if !m.loaded {
			m.err = msg
			return m, nil
		}
		return m, m.setStatus(msg.Error(), true)
	}

	return m, nil
}

// openSearch switches to the search overlay and focuses its input.
func (m *Model) openSearch() tea.Cmd {
	m.returnTo = m.state
	m.state = searchView
	m.inputState = 2
	m.textInput.Reset()
	m.textInput.Placeholder = i18n.T("search.ph")
	m.textInput.Width = 50
	m.textInput.Focus()
	return textinput.Blink
}

// ── Key routing ─────────────────────────────────────────────────────────────

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	k := msg.String()

	if m.err != nil {
		switch k {
		case "ctrl+c", "q", "esc":
			return tea.Quit
		case "r":
			m.err = nil
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.fetchIssues, m.fetchStatuses)
		}
		return nil
	}

	// Overlays that own their keys.
	switch m.state {
	case timeInputView, searchView, paletteView:
		return m.updateTextInput(msg, k)
	case statusView:
		return m.updateStatusPicker(k)
	}

	if k == "ctrl+c" {
		return tea.Quit
	}

	if m.state == helpView {
		switch k {
		case "?", "esc", "q", "enter":
			m.state = kanbanView
		}
		return nil
	}

	switch k {
	case "ctrl+p", ":":
		return m.openPalette()
	case "ctrl+f":
		return m.openSearch()
	case "?":
		m.state = helpView
		return nil
	case "esc":
		m.state = kanbanView
		return nil
	}

	// Issue actions are identical on the board and in the detail view.
	if cmd, handled := m.issueAction(k); handled {
		return cmd
	}

	switch m.state {
	case detailView:
		return m.updateDetail(msg)
	default:
		return m.updateKanban(k)
	}
}

// updateStatusPicker drives the status overlay: pick a status, apply it, or go back.
func (m *Model) updateStatusPicker(k string) tea.Cmd {
	switch k {
	case "esc", "q":
		m.state = m.returnTo
	case "enter":
		if len(m.statuses) == 0 || m.colIdx >= len(m.statuses) || m.selected == nil {
			return nil
		}
		// Return to the board so the moved card is visible straight away.
		st := m.statuses[m.colIdx]
		m.state = kanbanView
		m.followID = m.selected.ID
		return m.updateStatus(m.selected.ID, st.ID, st.Name)
	case "left", "h":
		if m.colIdx > 0 {
			m.colIdx--
		}
	case "right", "l":
		if m.colIdx < len(m.statuses)-1 {
			m.colIdx++
		}
	}
	return nil
}

// updateTextInput drives the three views that capture free text.
func (m *Model) updateTextInput(msg tea.KeyMsg, k string) tea.Cmd {
	switch m.state {

	case timeInputView:
		if k == "esc" {
			m.state = m.returnTo
			m.textInput.Reset()
			return nil
		}
		if msg.Type == tea.KeyEnter {
			if m.inputState == 0 {
				hours := strings.TrimSpace(m.textInput.Value())
				if !validHours(hours) {
					return m.setStatus(i18n.T("time.invalid"), true)
				}
				m.logHours = hours
				m.inputState = 1
				m.textInput.Reset()
				m.textInput.Placeholder = i18n.T("time.ph.comments")
				m.textInput.Width = 50
				return nil
			}
			comments := m.textInput.Value()
			issueID := 0
			if m.selected != nil {
				issueID = m.selected.ID
			}
			m.state = m.returnTo
			m.textInput.Reset()
			if issueID != 0 {
				return m.logTime(issueID, m.logHours, comments)
			}
			return nil
		}

	case searchView:
		if k == "esc" {
			m.state = m.returnTo
			m.textInput.Reset()
			return nil
		}
		if msg.Type == tea.KeyEnter {
			query := strings.TrimSpace(m.textInput.Value())
			m.state = kanbanView
			m.textInput.Reset()
			if query == "" {
				m.applyFilter()
				return nil
			}
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.searchIssues(query))
		}

	case paletteView:
		if k == "esc" {
			m.state = kanbanView
			return nil
		}
		cmds := m.paletteMatches()
		switch k {
		case "up", "ctrl+k":
			if m.paletteIdx > 0 {
				m.paletteIdx--
			}
			return nil
		case "down", "ctrl+j":
			if m.paletteIdx < len(cmds)-1 {
				m.paletteIdx++
			}
			return nil
		case "enter":
			if m.paletteIdx >= len(cmds) {
				m.state = kanbanView
				return nil
			}
			cmd := cmds[m.paletteIdx].run(m)
			if m.state == paletteView {
				m.state = kanbanView
			}
			return cmd
		}
		m.paletteIdx = 0
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	if m.state == paletteView {
		m.paletteInput, cmd = m.paletteInput.Update(msg)
	}
	return cmd
}

// issueAction runs the keys that act on the focused issue, wherever it is
// focused. It reports whether the key belonged to this group.
func (m *Model) issueAction(k string) (tea.Cmd, bool) {
	switch k {
	case "s":
		issue := m.issueTarget()
		if issue == nil {
			return nil, true
		}
		m.selected = issue
		m.colIdx = max(0, m.statusIndex(issue.Status.ID))
		m.returnTo = m.state
		m.state = statusView
		return nil, true

	case "t":
		issue := m.issueTarget()
		if issue == nil {
			return nil, true
		}
		m.selected = issue
		m.returnTo = m.state
		m.state = timeInputView
		m.inputState = 0
		m.textInput.Reset()
		m.textInput.Placeholder = i18n.T("time.ph.hours")
		m.textInput.Width = 20
		m.textInput.Focus()
		return textinput.Blink, true

	case "y":
		issue := m.issueTarget()
		if issue == nil {
			return nil, true
		}
		_ = clipboardWrite(issue.Subject)
		return m.setStatus(i18n.T("msg.copied"), false), true

	case "o":
		issue := m.issueTarget()
		if issue == nil {
			return nil, true
		}
		_ = openBrowser(fmt.Sprintf("%s/issues/%d", m.client.Host, issue.ID))
		return m.setStatus(i18n.T("msg.opening"), false), true

	case "H", "L":
		issue := m.issueTarget()
		if issue == nil {
			return nil, true
		}
		idx := m.statusIndex(issue.Status.ID)
		target := idx - 1
		if k == "L" {
			target = idx + 1
		}
		if idx < 0 || target < 0 || target >= len(m.statuses) {
			return m.setStatus(i18n.T("msg.nomove"), true), true
		}
		st := m.statuses[target]
		m.followID = issue.ID
		return m.updateStatus(issue.ID, st.ID, st.Name), true
	}
	return nil, false
}

func (m *Model) updateDetail(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q":
		m.state = kanbanView
		return nil
	case "g", "home":
		m.viewport.GotoTop()
		return nil
	case "G", "end":
		m.viewport.GotoBottom()
		return nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

func (m *Model) updateKanban(k string) tea.Cmd {
	switch k {
	case "q":
		return tea.Quit

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
		if col := m.currentColumn(); col != nil && m.cardIdx < len(col.issues)-1 {
			m.cardIdx++
		}
	case "g", "home":
		m.cardIdx = 0
	case "G", "end":
		if col := m.currentColumn(); col != nil && len(col.issues) > 0 {
			m.cardIdx = len(col.issues) - 1
		}

	case "enter", " ":
		issue := m.selectedIssue()
		if issue == nil {
			return nil
		}
		m.selected = issue
		m.state = detailView
		m.viewport.SetContent(i18n.T("app.loading"))
		m.viewport.GotoTop()
		return m.fetchIssueDetails(issue.ID)

	case "f":
		projects := m.uniqueProjects()
		if len(projects) == 0 {
			return nil
		}
		idx := -1
		for i, p := range projects {
			if p == m.activeFilter {
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
			return m.setStatus(i18n.T("msg.filter.all"), false)
		}
		return m.setStatus(fmt.Sprintf(i18n.T("msg.filter"), m.activeFilter), false)

	case "S":
		m.sortMode = m.sortMode.next()
		m.applyFilter()
		m.cardIdx = 0
		return m.setStatus(fmt.Sprintf(i18n.T("msg.sort"), m.sortMode.label()), false)

	case "r":
		m.loading = true
		return tea.Batch(m.spinner.Tick, m.fetchIssues, m.fetchStatuses)

	case "e":
		if err := m.exportToHTML(); err != nil {
			return m.setStatus(fmt.Sprintf("Export error: %v", err), true)
		}
		return m.setStatus(i18n.T("msg.exported"), false)

	case "v":
		return tea.Batch(
			m.setStatus(i18n.T("msg.syncing"), false),
			m.syncVikunja(),
		)
	}
	return nil
}

// ── Mouse ───────────────────────────────────────────────────────────────────

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if m.state != kanbanView || !m.cfg.Mouse || !m.loaded {
		return nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.cardIdx > 0 {
			m.cardIdx--
		}
		return nil
	case tea.MouseButtonWheelDown:
		if col := m.currentColumn(); col != nil && m.cardIdx < len(col.issues)-1 {
			m.cardIdx++
		}
		return nil
	}

	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return nil
	}

	for i, c := range m.geom.cols {
		if msg.X >= c.x && msg.X < c.x+c.w && msg.Y >= c.headerY0 && msg.Y < c.headerY1 {
			m.colIdx = i
			m.cardIdx = 0
			return nil
		}
	}
	for _, c := range m.geom.cards {
		if msg.Y < c.y0 || msg.Y >= c.y1 || c.col >= len(m.geom.cols) {
			continue
		}
		col := m.geom.cols[c.col]
		if msg.X < col.x || msg.X >= col.x+col.w {
			continue
		}
		if m.colIdx == c.col && m.cardIdx == c.index {
			// A second click on the focused card opens it.
			issue := m.selectedIssue()
			if issue == nil {
				return nil
			}
			m.selected = issue
			m.state = detailView
			m.viewport.SetContent(i18n.T("app.loading"))
			m.viewport.GotoTop()
			return m.fetchIssueDetails(issue.ID)
		}
		m.colIdx = c.col
		m.cardIdx = c.index
		return nil
	}
	return nil
}

// ── Kanban helpers ──────────────────────────────────────────────────────────

func (m *Model) rebuildColumns(issues []redmine.Issue) {
	var cols []kanbanColumn
	if len(m.statuses) == 0 {
		seen := map[int]bool{}
		for _, issue := range issues {
			if !seen[issue.Status.ID] {
				seen[issue.Status.ID] = true
				cols = append(cols, kanbanColumn{
					status: redmine.IssueStatus{ID: issue.Status.ID, Name: issue.Status.Name},
				})
			}
		}
	} else {
		cols = make([]kanbanColumn, len(m.statuses))
		for i, st := range m.statuses {
			cols[i] = kanbanColumn{status: st}
		}
	}

	for _, issue := range issues {
		for i := range cols {
			if cols[i].status.ID == issue.Status.ID {
				cols[i].issues = append(cols[i].issues, issue)
				break
			}
		}
	}

	if m.sortMode != sortServer {
		less := m.lessFunc()
		for i := range cols {
			issues := cols[i].issues
			sort.SliceStable(issues, func(a, b int) bool { return less(issues[a], issues[b]) })
		}
	}

	nonEmpty := make([]kanbanColumn, 0, len(cols))
	for _, col := range cols {
		if len(col.issues) > 0 {
			nonEmpty = append(nonEmpty, col)
		}
	}
	m.columns = nonEmpty
	if m.colIdx >= len(m.columns) {
		m.colIdx = max(0, len(m.columns)-1)
	}

	// A card just moved with H/L keeps the cursor: otherwise the next press
	// would act on whichever card slid into the vacated slot.
	if m.followID != 0 {
		for ci := range m.columns {
			for ii, issue := range m.columns[ci].issues {
				if issue.ID == m.followID {
					m.colIdx, m.cardIdx = ci, ii
				}
			}
		}
		m.followID = 0
	}
}

func (m *Model) lessFunc() func(a, b redmine.Issue) bool {
	switch m.sortMode {
	case sortPriority:
		return func(a, b redmine.Issue) bool {
			return classifyPriority(a.Priority.Name) < classifyPriority(b.Priority.Name)
		}
	case sortDue:
		return func(a, b redmine.Issue) bool {
			if a.DueDate == "" {
				return false
			}
			if b.DueDate == "" {
				return true
			}
			return a.DueDate < b.DueDate
		}
	case sortUpdated:
		return func(a, b redmine.Issue) bool { return a.UpdatedOn.After(b.UpdatedOn) }
	default:
		return func(a, b redmine.Issue) bool {
			if a.Project.Name != b.Project.Name {
				return a.Project.Name < b.Project.Name
			}
			return a.ID < b.ID
		}
	}
}

func (m Model) filtered() []redmine.Issue {
	if m.activeFilter == "" {
		return m.allIssues
	}
	out := make([]redmine.Issue, 0, len(m.allIssues))
	for _, issue := range m.allIssues {
		if issue.Project.Name == m.activeFilter {
			out = append(out, issue)
		}
	}
	return out
}

func (m *Model) applyFilter() {
	m.rebuildColumns(m.filtered())
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

// issueTarget is the issue the shared action keys act on.
func (m *Model) issueTarget() *redmine.Issue {
	if m.state == detailView {
		return m.selected
	}
	return m.selectedIssue()
}

func (m *Model) statusIndex(id int) int {
	for i, s := range m.statuses {
		if s.ID == id {
			return i
		}
	}
	return -1
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
		return m.viewError()
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
	case helpView:
		return m.viewHelp()
	case paletteView:
		return m.viewPalette()
	default:
		return m.viewKanban()
	}
}
