# Redmine TUI

A Terminal User Interface (TUI) for Redmine, built with Go and Bubble Tea. This tool allows you to view assigned issues, log time, change issue status, filter by projects, search issues, and sync assigned tasks to Planka.

## Features

- **View Assigned Issues**: List all issues assigned to you with status of 'Open'.
- **Issue Details**: Press `Space` to view detailed description and history of an issue.
- **Log Time**: Press `t` to log time and comments to an issue.
- **Change Status**: Press `s` to quickly update the status of an issue.
- **Filter**: Press `f` to filter issues by Project.
- **Search**: Press `Ctrl+f` to search issue descriptions.
- **Export**: Press `e` to export current view to an HTML file.
- **Sync to Planka**: Press `p` to sync your assigned Redmine issues to a Planka board list ("Üzerimdeki İşler").
- **CLI Sync**: Run with `--sync` to perform a one-time sync without the UI (useful for cron jobs).

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd redmine-tui
   ```

2. Build the binary using Make:
   ```bash
   make build
   ```
   Or manually:
   ```bash
   go build -o redmine-tui .
   ```

3. Run the application:
   ```bash
   make run
   ```

## Configuration

You can configure the application using environment variables or a configuration file. The application searches for a configuration file in the following order:

1. `$HOME/.redmine-tui.yaml`
2. `./.redmine-tui.yaml`
3. `/etc/default/redmine-tui.yaml` (or `.redmine-tui.yaml`)

### Environment Variables

```bash
# Redmine Configuration
export REDMINE_API_KEY="your_redmine_api_key"
export REDMINE_HOST="redmine.example.com"

# Planka Configuration (Optional, for Sync feature)
export PLANKA_API_URL="https://planka.example.com/api"
export PLANKA_USERNAME="your_username"
export PLANKA_PASSWORD="your_password"
```

### Config File (`.redmine-tui.yaml`)

```yaml
api_key: "your_redmine_api_key"
host: "redmine.example.com"

planka:
  base_url: "https://planka.example.com/api"
  username: "your_username"
  password: "your_password"
  board_id: "1634400507663484080"
  list_id: "1634400550629934260"
```

## Key Bindings

| Key | Action |
| --- | --- |
| `Space` | View Issue Details |
| `s` | Change Issue Status |
| `t` | Log Time to Issue |
| `f` | Filter by Project |
| `Ctrl+f`| Search Issues |
| `e` | Export to HTML |
| `p` | Sync to Planka |
| `Esc` | Go Back / Clear Input |
| `q` / `Ctrl+c` | Quit |

## Cron Job Example

To run the sync automatically every day at 18:00 (6 PM), add the following to your crontab (`crontab -e`):

```bash
0 18 * * * /path/to/redmine-tui --sync
```

Ensure environment variables are set in the cron environment or loaded from `.redmine-tui.yaml`.

## Requirements

- Go 1.21+
- A Redmine instance with API access enabled using an API Key.
- (Optional) A Planka instance for the sync feature.

## License

[MIT](LICENSE)
