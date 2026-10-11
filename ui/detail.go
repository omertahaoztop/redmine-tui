package ui

import (
	"fmt"
	"strings"

	"redmine-tui/i18n"
	"redmine-tui/redmine"

	"github.com/charmbracelet/lipgloss"
)

// ── Detail view ──────────────────────────────────────────────────────────────

func (m Model) viewDetail() string {
	if m.selected == nil {
		return ""
	}
	w := m.windowWidth
	if w == 0 {
		w = 100
	}

	m.viewport.Width = w - 8
	m.viewport.Height = m.windowHeight - 6
	m.viewport.SetContent(renderDetail(m.selected, m.viewport.Width))

	rows := []string{
		overlayHeader(i18n.T("detail.title"), i18n.T("hint.detail"), colorAccent, w),
		lipgloss.NewStyle().Padding(0, 2).Render(m.viewport.View()),
	}
	if line := m.statusLine(); line != "" {
		rows = append(rows, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderDetail(issue *redmine.Issue, width int) string {
	label := func(k string) string { return styleDetailLabel.Render(i18n.T(k) + ":") }
	val := lipgloss.NewStyle().Foreground(colorText).Render

	title := styleDetailTitle.Render(fmt.Sprintf("[#%d] %s", issue.ID, issue.Subject))

	status := statusDot(issue.Status.Name) + " " +
		lipgloss.NewStyle().Foreground(statusColor(issue.Status.Name)).Render(issue.Status.Name)

	prio := priorityStyle(issue.Priority.Name).Render(priorityIcon(issue.Priority.Name) + " " + issue.Priority.Name)

	due := val("—")
	if d, style := humanDue(issue.DueDate); d != "" {
		due = style.Render(d)
	}

	assignee := val("—")
	if issue.AssignedTo.Name != "" {
		assignee = val(issue.AssignedTo.Name)
	}

	updated := val(issue.UpdatedOn.Format("2006-01-02 15:04")) +
		"  " + styleHelpDesc.Render("("+relativeTime(issue.UpdatedOn)+")")

	meta := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("%s %s    %s %s    %s %s",
			label("label.project"), styleProjectTag.Render(issue.Project.Name),
			label("label.status"), status,
			label("label.tracker"), val(trackerIcon(issue.Tracker.Name)+issue.Tracker.Name),
		),
		fmt.Sprintf("%s %s    %s %s    %s %s",
			label("label.priority"), prio,
			label("label.due"), due,
			label("label.author"), val(issue.Author.Name),
		),
		fmt.Sprintf("%s %s    %s %s",
			label("label.assignee"), assignee,
			label("label.updated"), updated,
		),
		fmt.Sprintf("%s %s",
			label("label.created"), val(issue.CreatedOn.Format("2006-01-02 15:04")),
		),
	)
	if issue.DoneRatio > 0 {
		meta += "\n" + styleDetailLabel.Render(i18n.T("label.progress")+": ") +
			fmt.Sprintf("%d%% %s", issue.DoneRatio, progressBar(issue.DoneRatio, 20))
	}

	divider := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", max(4, width-4)))

	desc := lipgloss.NewStyle().Foreground(colorText).Render(renderMarkdownLite(issue.Description))
	if strings.TrimSpace(issue.Description) == "" {
		desc = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(i18n.T("detail.nodesc"))
	}

	sections := []string{
		title, meta, divider,
		styleDetailLabel.Render(i18n.T("label.description")+":") + "\n" + desc,
	}

	notes := make([]redmine.Journal, 0, len(issue.Journals))
	for _, j := range issue.Journals {
		if j.Notes != "" {
			notes = append(notes, j)
		}
	}
	if len(notes) > 0 {
		sections = append(sections, divider, styleDetailLabel.Render(i18n.T("label.history")+":"))
		for _, j := range notes {
			sections = append(sections, renderJournal(j))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderJournal draws one note on a thin timeline rail.
func renderJournal(j redmine.Journal) string {
	rail := lipgloss.NewStyle().Foreground(colorBorder).Render("│")
	node := lipgloss.NewStyle().Foreground(colorAccent).Render("◆")

	who := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(j.User.Name)
	when := styleHelpDesc.Render(j.CreatedOn.Format("2006-01-02 15:04") + " · " + relativeTime(j.CreatedOn))

	lines := []string{rail + " " + node + " " + who + "  " + when}
	for _, line := range strings.Split(renderMarkdownLite(j.Notes), "\n") {
		lines = append(lines, rail+"   "+line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
