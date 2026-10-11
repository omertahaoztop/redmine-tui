package ui

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"redmine-tui/redmine"
)

// ── Export ──────────────────────────────────────────────────────────────────

func (m Model) exportToHTML() error {
	f, err := os.Create("redmine_issues.html")
	if err != nil {
		return err
	}
	defer f.Close()

	doc := `<!DOCTYPE html>
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
		doc += fmt.Sprintf(`<div class="col"><h3>%s <span>%d</span></h3>`, html.EscapeString(status), len(issues))
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
				due = fmt.Sprintf(`<div class="%s">Due: %s</div>`, dueClass, html.EscapeString(issue.DueDate))
			}
			bar := ""
			if issue.DoneRatio > 0 {
				bar = fmt.Sprintf(`<div class="progress"><div class="progress-fill" style="width:%d%%"></div></div>`, issue.DoneRatio)
			}
			doc += fmt.Sprintf(`
<div class="card">
  <h4><span class="id">#%d</span> %s</h4>
  <span class="tag">%s</span><span class="prio %s">%s %s</span>
  %s%s
</div>`, issue.ID, html.EscapeString(issue.Subject), html.EscapeString(issue.Project.Name), pClass, priorityIcon(issue.Priority.Name), html.EscapeString(issue.Priority.Name), due, bar)
		}
		doc += "</div>"
	}

	doc += `</div></body></html>`
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	// Open the absolute path: a bare relative filename is not a URL and
	// xdg-open/rundll32 resolve it against an unpredictable cwd.
	abs, err := filepath.Abs("redmine_issues.html")
	if err != nil {
		return err
	}
	return openBrowser(abs)
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
