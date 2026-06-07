# Redmine TUI

[![CI](https://github.com/omertahaoztop/redmine-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/omertahaoztop/redmine-tui/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/omertahaoztop/redmine-tui)](https://goreportcard.com/report/github.com/omertahaoztop/redmine-tui)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A fast, modern Terminal User Interface (TUI) for [Redmine](https://www.redmine.org/), built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). View assigned issues on a kanban board, log time, change status, filter, search, export to HTML, and sync assigned tasks to [Vikunja](https://vikunja.io/).

## Features

- **Kanban Board** — Issues assigned to you, grouped by status, with priority colors, due-date warnings, and progress bars.
- **Issue Details** — Press `Space` for full description and comment history.
- **Log Time** — Press `t` to log hours and comments (with input validation).
- **Change Status** — Press `s` to move an issue through your workflow.
- **Filter** — Press `f` to cycle through projects.
- **Search** — Press `Ctrl+f` to search issue subjects and descriptions.
- **Open in Browser** — Press `o` to open the selected issue in Redmine.
- **Export** — Press `e` to export the current board to a styled HTML file.
- **Sync to Vikunja** — Press `v` to sync your assigned Redmine issues to a Vikunja kanban bucket.
- **Multi-language** — English (default) and Turkish, selectable via config/env.
- **CLI Sync** — Run with `--sync` for a headless one-shot sync (great for cron).

## Installation

```bash
git clone https://github.com/omertahaoztop/redmine-tui.git
cd redmine-tui
make build      # or: go build -o redmine-tui .
make run        # or: ./redmine-tui
```

## Configuration

Configuration is loaded (in order) from:

1. `$HOME/.redmine-tui.yaml`
2. `./.redmine-tui.yaml`
3. `/etc/default/redmine-tui.yaml`

Environment variables override file values. Copy the sample to get started:

```bash
cp .redmine-tui.example.yaml ~/.redmine-tui.yaml
# then edit it with your own credentials
```

### Config File (`.redmine-tui.yaml`)

```yaml
api_key: "your_redmine_api_key"
host: "redmine.example.com"
language: "en"           # "en" or "tr"

vikunja:
  base_url: "https://vikunja.example.com"   # /api/v1 is appended automatically
  token: "tk_xxxxxxxxxxxxxxxxxxxxxxxx"       # or use username/password below
  username: "your_username"
  password: "your_password"
  project_id: 5
  view_id: 20            # optional; auto-detected kanban view if omitted
  bucket_id: 34          # source bucket for "assigned" tasks
  done_bucket_id: 39     # optional; closed issues move here instead of being deleted
```

### Environment Variables

```bash
export REDMINE_API_KEY="your_redmine_api_key"
export REDMINE_HOST="redmine.example.com"
export REDMINE_TUI_LANG="en"

export VIKUNJA_API_URL="https://vikunja.example.com"
export VIKUNJA_TOKEN="tk_xxxxxxxx"
export VIKUNJA_USERNAME="your_username"
export VIKUNJA_PASSWORD="your_password"
export VIKUNJA_PROJECT_ID="5"
export VIKUNJA_VIEW_ID="20"
export VIKUNJA_BUCKET_ID="34"
export VIKUNJA_DONE_BUCKET_ID="39"
```

### Finding Vikunja Project / Bucket IDs

- **Project ID** — Open the project in Vikunja; it's the number in the URL (`.../projects/5/...`).
- **View / Bucket IDs** — Open the kanban view, then check the API call `GET /api/v1/projects/{id}/views/{viewId}/tasks` in your browser's network tab. If `view_id` is omitted, the app auto-detects the first kanban view.

## Key Bindings

| Key | Action |
| --- | --- |
| `←→` / `h l` | Move between columns |
| `↑↓` / `k j` | Move between cards |
| `Space` / `Enter` | View issue details |
| `s` | Change issue status |
| `t` | Log time to issue |
| `f` | Cycle project filter |
| `Ctrl+f` | Search issues |
| `o` | Open issue in browser |
| `y` | Copy subject to clipboard |
| `r` | Refresh |
| `e` | Export to HTML |
| `v` | Sync to Vikunja |
| `Esc` | Go back / cancel input |
| `q` / `Ctrl+c` | Quit |

## Cron Example

Run the sync every day at 18:00:

```bash
0 18 * * * /path/to/redmine-tui --sync
```

Make sure credentials are available in the cron environment or via `.redmine-tui.yaml`.

## Requirements

- Go 1.21+
- A Redmine instance with REST API enabled and an API key.
- (Optional) A Vikunja instance for the sync feature.

## License

[MIT](LICENSE)
