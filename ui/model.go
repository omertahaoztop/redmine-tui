package ui

import (
	"fmt"
	"os"
	"os/exec"
	"redmine-tui/config"
	"redmine-tui/planka"
	"redmine-tui/redmine"
	"redmine-tui/sync"
	"runtime"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type sessionState int

const (
	listView sessionState = iota
	detailView
	statusView
	timeInputView
	projectFilterView
	searchView
)

type item struct {
	issue redmine.Issue
}

func (i item) Title() string { return i.issue.Subject }
func (i item) Description() string {
	return fmt.Sprintf("%s | %s", i.issue.Project.Name, i.issue.Status.Name)
}
func (i item) FilterValue() string { return i.issue.Subject }

type statusItem struct {
	status redmine.IssueStatus
}

func (i statusItem) Title() string       { return i.status.Name }
func (i statusItem) Description() string { return fmt.Sprintf("ID: %d", i.status.ID) }
func (i statusItem) FilterValue() string { return i.status.Name }

type projectItem struct {
	name string
	id   int
}

func (i projectItem) Title() string       { return i.name }
func (i projectItem) Description() string { return "" }
func (i projectItem) FilterValue() string { return i.name }

type Model struct {
	list         list.Model
	statusList   list.Model
	projectList  list.Model
	allIssues    []redmine.Issue
	viewport     viewport.Model
	textInput    textinput.Model
	client       *redmine.Client
	cfg          *config.Config
	state        sessionState
	inputState   int // 0: Hours, 1: Comments, 2: Search Query
	logHours     string
	selected     *redmine.Issue
	loaded       bool
	err          error
	windowWidth  int
	windowHeight int
}

func NewModel(cfg *config.Config) Model {
	client := redmine.NewClient(cfg.APIKey, cfg.Host)
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Assigned Issues (Space: Details, s: Status, t: Log Time, f: Filter, Ctrl+f: Search, e: Export, p: Sync Planka)"

	sl := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	sl.Title = "Select New Status"

	pl := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	pl.Title = "Filter by Project"

	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	ti := textinput.New()
	ti.Placeholder = "Enter hours (e.g. 1.5)"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return Model{
		client:      client,
		cfg:         cfg,
		list:        l,
		statusList:  sl,
		projectList: pl,
		viewport:    vp,
		textInput:   ti,
		state:       listView,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchIssues, textinput.Blink)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "ctrl+f" {
			m.state = searchView
			m.inputState = 2 // Search Query
			m.textInput.Reset()
			m.textInput.Placeholder = "Search description..."
			m.textInput.Width = 50
			m.textInput.Focus()
			return m, textinput.Blink
		}

		if m.state == detailView {
			if msg.String() == "esc" {
				m.state = listView
				return m, nil
			}
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		if m.state == statusView {
			if msg.String() == "esc" {
				m.state = listView
				return m, nil
			}
			if msg.String() == "enter" {
				if i, ok := m.statusList.SelectedItem().(statusItem); ok {
					var issueID int
					if m.selected != nil {
						issueID = m.selected.ID
					} else if lItem, ok := m.list.SelectedItem().(item); ok {
						issueID = lItem.issue.ID
					}

					if issueID != 0 {
						return m, tea.Batch(
							m.updateStatus(issueID, i.status.ID),
							m.list.NewStatusMessage(fmt.Sprintf("Updating status to %s...", i.status.Name)),
						)
					}
				}
			}
			m.statusList, cmd = m.statusList.Update(msg)
			return m, cmd
		}

		if m.state == projectFilterView {
			if msg.String() == "esc" {
				m.state = listView
				return m, nil
			}
			if msg.String() == "enter" {
				if i, ok := m.projectList.SelectedItem().(projectItem); ok {
					var filteredItems []list.Item
					if i.name == "All Projects" {
						filteredItems = make([]list.Item, len(m.allIssues))
						for idx, issue := range m.allIssues {
							filteredItems[idx] = item{issue: issue}
						}
						m.list.Title = "Assigned Issues (All Projects)"
					} else {
						for _, issue := range m.allIssues {
							if issue.Project.ID == i.id {
								filteredItems = append(filteredItems, item{issue: issue})
							}
						}
						m.list.Title = fmt.Sprintf("Assigned Issues (%s)", i.name)
					}
					m.list.SetItems(filteredItems)
					m.state = listView
					return m, nil
				}
			}
			m.projectList, cmd = m.projectList.Update(msg)
			return m, cmd
		}

		if m.state == timeInputView || m.state == searchView {
			if msg.String() == "esc" {
				m.state = listView
				m.textInput.Reset()
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				if m.state == searchView { // Search submitted
					query := m.textInput.Value()
					m.state = listView
					m.textInput.Reset()
					return m, tea.Batch(
						m.searchIssues(query),
						m.list.NewStatusMessage(fmt.Sprintf("Searching for '%s'...", query)),
					)
				}

				if m.inputState == 0 { // Hours entered
					m.logHours = m.textInput.Value()
					m.inputState = 1
					m.textInput.Reset()
					m.textInput.Placeholder = "Enter comments..."
					m.textInput.Width = 50
					return m, nil
				} else { // Comments entered
					comments := m.textInput.Value()
					var issueID int
					if m.selected != nil {
						issueID = m.selected.ID
					} else if lItem, ok := m.list.SelectedItem().(item); ok {
						issueID = lItem.issue.ID
					}

					if issueID != 0 {
						m.state = listView
						m.textInput.Reset()
						return m, tea.Batch(
							m.logTime(issueID, m.logHours, comments),
							m.list.NewStatusMessage("Logging time..."),
						)
					}
				}
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// List View Controls
		if msg.String() == "enter" {
			if i, ok := m.list.SelectedItem().(item); ok {
				clipboard.WriteAll(i.issue.Subject)
				return m, m.list.NewStatusMessage("Copied to clipboard!")
			}
		}

		if msg.String() == " " { // Space to open details
			if i, ok := m.list.SelectedItem().(item); ok {
				m.state = detailView
				m.selected = &i.issue // Keep basic info until details load
				m.viewport.SetContent("Loading details...")
				return m, m.fetchIssueDetails(i.issue.ID)
			}
		}

		if msg.String() == "s" { // Status selection
			if i, ok := m.list.SelectedItem().(item); ok {
				m.selected = &i.issue // Track which issue we are editing
				m.state = statusView
				if len(m.statusList.Items()) == 0 {
					return m, m.fetchStatuses
				}
				return m, nil
			}
		}

		if msg.String() == "t" { // Time logging
			if i, ok := m.list.SelectedItem().(item); ok {
				m.selected = &i.issue
				m.state = timeInputView
				m.inputState = 0
				m.textInput.Placeholder = "Enter hours (e.g. 1.5)"
				m.textInput.Width = 20
				m.textInput.Focus()
				return m, textinput.Blink
			}
		}

		if msg.String() == "f" { // Filter by project
			m.state = projectFilterView

			// Extract unique projects
			projects := make(map[int]string)
			for _, issue := range m.allIssues {
				projects[issue.Project.ID] = issue.Project.Name
			}

			items := []list.Item{projectItem{name: "All Projects", id: 0}}
			for id, name := range projects {
				items = append(items, projectItem{name: name, id: id})
			}
			m.projectList.SetItems(items)
			return m, nil
		}

		// Handle 'e' for Export
		if msg.String() == "e" {
			err := m.exportToHTML()
			if err != nil {
				return m, m.list.NewStatusMessage(fmt.Sprintf("Error exporting: %v", err))
			}
			return m, m.list.NewStatusMessage("Exported to redmine_issues.html and opened in browser")
		}

		// Handle 'p' for Sync to Planka
		if msg.String() == "p" {
			return m, tea.Batch(
				m.syncPlanka(),
				m.list.NewStatusMessage("Syncing with Planka..."),
			)
		}

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.statusList.SetSize(msg.Width-h, msg.Height-v)
		m.projectList.SetSize(msg.Width-h, msg.Height-v)

		// Update viewport size
		m.viewport.Width = msg.Width - h
		m.viewport.Height = msg.Height - v

	case []redmine.Issue:
		m.allIssues = msg // Store all issues
		items := make([]list.Item, len(msg))
		for i, issue := range msg {
			items[i] = item{issue: issue}
		}
		m.list.SetItems(items)
		m.loaded = true

		if m.state == searchView { // Return to list view after search results loaded
			m.state = listView
		}

		// If we updated status, return to list view
		if m.state == statusView {
			m.state = listView
		}

	case *redmine.Issue: // Detail fetched
		m.selected = msg
		m.viewport.SetContent(renderDetail(msg, m.viewport.Width))

	case []redmine.IssueStatus: // Statuses fetched
		items := make([]list.Item, len(msg))
		for i, s := range msg {
			items[i] = statusItem{status: s}
		}
		m.statusList.SetItems(items)

	case string: // Status update success (custom msg)
		if msg == "status_updated" {
			m.state = listView
			return m, tea.Batch(
				m.list.NewStatusMessage("Status updated!"),
				m.fetchIssues, // Refresh list
			)
		} else if msg == "time_logged" {
			return m, m.list.NewStatusMessage("Time logged successfully!")
		} else if msg == "planka_synced" {
			return m, m.list.NewStatusMessage("Successfully synced with Planka!")
		}

	case error:
		m.err = msg
		return m, tea.Quit
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func renderDetail(issue *redmine.Issue, width int) string {
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render

	content := fmt.Sprintf("%s\n%s\n\n", titleStyle(issue.Subject), infoStyle(fmt.Sprintf("ID: %d | Status: %s | Priority: %s", issue.ID, issue.Status.Name, issue.Priority.Name)))

	content += fmt.Sprintf("Project: %s\n", issue.Project.Name)
	content += fmt.Sprintf("Author: %s\n", issue.Author.Name)
	content += fmt.Sprintf("Created: %s\n\n", issue.CreatedOn.Format("2006-01-02 15:04"))

	content += lipgloss.NewStyle().Bold(true).Render("Description:") + "\n"
	content += issue.Description + "\n\n"

	if len(issue.Journals) > 0 {
		content += lipgloss.NewStyle().Bold(true).Render("History:") + "\n"
		for _, j := range issue.Journals {
			if j.Notes != "" {
				content += fmt.Sprintf("--- \n%s (%s):\n%s\n", j.User.Name, j.CreatedOn.Format("01/02 15:04"), j.Notes)
			}
		}
	}

	return content
}

func (m Model) exportToHTML() error {
	filename := "redmine_issues.html"
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	htmlContent := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Redmine Issues</title>
    <style>
        body { font-family: sans-serif; padding: 20px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:hover { background-color: #f5f5f5; }
        .copy-btn { margin-bottom: 20px; padding: 10px 20px; background: #007bff; color: white; border: none; cursor: pointer; border-radius: 5px;}
        .copy-btn:hover { background: #0056b3; }
    </style>
</head>
<body>
    <button class="copy-btn" onclick="copyTable()">Copy Table</button>
    <table id="issuesTable">
        <thead>
            <tr>
                <th>ID</th>
                <th>Project</th>
                <th>Priority</th>
                <th>Status</th>
                <th>Subject</th>
            </tr>
        </thead>
        <tbody>
`
	if _, err := f.WriteString(htmlContent); err != nil {
		return err
	}

	for _, i := range m.list.Items() {
		if itm, ok := i.(item); ok {
			row := fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
				itm.issue.ID,
				itm.issue.Project.Name,
				itm.issue.Priority.Name,
				itm.issue.Status.Name,
				itm.issue.Subject,
			)
			if _, err := f.WriteString(row); err != nil {
				return err
			}
		}
	}

	footer := `
        </tbody>
    </table>
    <script>
        function copyTable() {
            var range = document.createRange();
            range.selectNode(document.getElementById("issuesTable"));
            window.getSelection().removeAllRanges(); 
            window.getSelection().addRange(range); 
            document.execCommand("copy");
            window.getSelection().removeAllRanges();
            alert("Table copied to clipboard!");
        }
    </script>
</body>
</html>`
	if _, err := f.WriteString(footer); err != nil {
		return err
	}

	return openBrowser(filename)
}

func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	if !m.loaded {
		return "Loading issues..."
	}

	if m.state == detailView {
		return docStyle.Render(m.viewport.View())
	}

	if m.state == statusView {
		return docStyle.Render(m.statusList.View())
	}

	if m.state == projectFilterView {
		return docStyle.Render(m.projectList.View())
	}

	if m.state == searchView {
		return docStyle.Render(fmt.Sprintf(
			"Search Issues\n\n%s\n\n%s",
			"Enter search query (searches description):",
			m.textInput.View(),
		))
	}

	return docStyle.Render(m.list.View())
}

func (m Model) fetchIssues() tea.Msg {
	issues, err := m.client.GetAssignedIssues()
	if err != nil {
		return err
	}
	return issues
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

func (m Model) fetchStatuses() tea.Msg {
	statuses, err := m.client.GetIssueStatuses()
	if err != nil {
		return err
	}
	return statuses
}

func (m Model) updateStatus(issueID, statusID int) tea.Cmd {
	return func() tea.Msg {
		err := m.client.UpdateIssueStatus(issueID, statusID)
		if err != nil {
			return err
		}
		return "status_updated"
	}
}

func (m Model) logTime(issueID int, hours string, comments string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.LogTime(issueID, hours, comments)
		if err != nil {
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
		return issues
	}
}

func (m Model) syncPlanka() tea.Cmd {
	return func() tea.Msg {
		plankaClient := planka.NewClient(m.cfg.Planka.BaseURL, m.cfg.Planka.Username, m.cfg.Planka.Password)
		if err := plankaClient.Login(); err != nil {
			return fmt.Errorf("planka login failed: %w", err)
		}

		if err := sync.SyncIssuesToPlanka(m.client, plankaClient, m.cfg); err != nil {
			return err
		}

		return "planka_synced"
	}
}
