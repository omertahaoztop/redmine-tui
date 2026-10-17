# Redmine TUI

[![CI](https://github.com/omertahaoztop/redmine-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/omertahaoztop/redmine-tui/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/omertahaoztop/redmine-tui)](go.mod)
[![Release](https://img.shields.io/github/v/release/omertahaoztop/redmine-tui)](https://github.com/omertahaoztop/redmine-tui/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![AUR](https://img.shields.io/aur/version/redmine-tui?label=AUR&logo=archlinux&color=1793D1)](https://aur.archlinux.org/packages/redmine-tui/)
[![AUR (bin)](https://img.shields.io/aur/version/redmine-tui-bin?label=AUR%20(bin)&logo=archlinux&color=1793D1)](https://aur.archlinux.org/packages/redmine-tui-bin/)

A fast, modern Terminal User Interface (TUI) for [Redmine](https://www.redmine.org/), built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). View assigned issues on a kanban board, log time, change status, filter, search, export to HTML, and sync assigned tasks to [Vikunja](https://vikunja.io/).

## Features

- **Kanban Board** — Issues assigned to you, grouped by status, with priority rails, due-date warnings, and progress bars.
- **Status Colors** — Each column and card carries the semantic color of its status (new, in progress, feedback, resolved, closed).
- **Issue Details** — Press `Space` for full description and comment history, with relative timestamps and light markdown rendering.
- **Log Time** — Press `t` to log hours and comments (with input validation).
- **Change Status** — Press `s` to pick a status, or `H` / `L` to move the issue one status back or forward in a single keypress.
- **Command Palette** — Press `:` (or `Ctrl+p`) to fuzzy-search every action; each entry shows its shortcut.
- **Help Overlay** — Press `?` for the full keymap, grouped by task.
- **Sorting** — Press `S` to cycle priority → due date → updated → project → server order.
- **Filter & Search** — Press `f` to cycle projects, `Ctrl+f` to search subjects and descriptions.
- **Open in Browser** — Press `o` to open the selected issue in Redmine.
- **Export** — Press `e` to export the current board to a styled HTML file.
- **Sync to Vikunja** — Press `v` to sync your assigned Redmine issues to a Vikunja kanban bucket.
- **Narrow Terminals** — Below 80 columns the board collapses to one column with a tab strip instead of squeezing every column.
- **Mouse (opt-in)** — `mouse: true` adds click-to-select, column headers, and wheel scrolling; off by default so terminal text selection keeps working.
- **Multi-language** — English (default) and Turkish, selectable via config/env.
- **CLI Sync** — Run with `--sync` for a headless one-shot sync (great for cron).

Sync requires all assigned open issues to fit in one Redmine response (up to 100,
or the server's lower page limit). If the response is incomplete or its count
cannot be verified, sync stops before changing Vikunja tasks. The board can
still load multiple pages. This prevents task removal caused by issues shifting
between pages during a sync.

## Installation

### From source

```bash
git clone https://github.com/omertahaoztop/redmine-tui.git
cd redmine-tui
make build      # or: go build -o redmine-tui .
make run        # or: ./redmine-tui
```

### Archlinux

```bash
# Source release
yay -S redmine-tui

# Binary release
yay -S redmine-tui-bin
```

## Configuration

Configuration is loaded (in order) from:

1. `$HOME/.redmine-tui.yaml`
2. `/etc/default/redmine-tui.yaml`

The working directory is deliberately not searched: a config file dropped next to the binary (a cloned repo, a shared temp dir) could point `host` or `vikunja.base_url` at a host you did not choose while your credentials come from the environment. Set `REDMINE_TUI_CONFIG` to an explicit path to load a config from anywhere else.

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
mouse: false             # true: click to select cards, wheel to scroll

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

`host` and `vikunja.base_url` must use `https://` unless they address the local machine: the Redmine API key and the Vikunja token or password are attached to every request, so a remote plaintext endpoint is refused. `REDMINE_TUI_CONFIG` selects a config file outside `$HOME` and `/etc/default`.

```bash
export REDMINE_API_KEY="your_redmine_api_key"
export REDMINE_HOST="redmine.example.com"
export REDMINE_TUI_LANG="en"
export REDMINE_TUI_MOUSE="false"

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
| `H` / `L` | Move issue to the previous / next status |
| `S` | Cycle sort order |
| `g` / `G` | Jump to first / last card |
| `:` or `Ctrl+p` | Command palette |
| `?` | Key bindings |
| `Esc` | Go back / cancel input |
| `q` / `Ctrl+c` | Quit |

## Cron Example

Run the sync every day at 18:00:

```bash
0 18 * * * /path/to/redmine-tui --sync
```

Make sure credentials are available in the cron environment or via `.redmine-tui.yaml`.

## Requirements

- Go 1.26+ (the module's own requirement; only needed to build from source)
- A Redmine instance with REST API enabled and an API key.
- (Optional) A Vikunja instance for the sync feature.

## License

[MIT](LICENSE)
