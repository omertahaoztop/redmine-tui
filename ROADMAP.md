# Redmine TUI Roadmap & To-Do

This document tracks the planned features and improvements for the Redmine TUI project.

- [x] **Detail View**
    - [x] Allow users to press a key (e.g., `Enter` or `Space`) to open detail view
    - [x] Display full Description
    - [x] Show recent journals/updates
    - [x] List attachments

- [x] **Status Updates & Workflow**
    - [x] Implement `PUT /issues/{id}.json` in API client
    - [x] "Mark as In Progress" shortcut
    - [x] "Close Issue" shortcut

- [x] **Time Logging**
    - [x] Create modal to input hours and comments
    - [x] Shortcut to log time to the currently selected issue

- [x] **Advanced Filtering & Search**
    - [x] Filter by Project (Client-side)
    - [x] Fuzzy search (built-in via '/')
    - [x] Text search within descriptions (Server-side via 'Ctrl+f')

- [x] **Configuration Management**
    - [x] Support for `~/.redmine-tui.yaml` or `.env` file
    - [x] Load API Key & Host from config/envd credentials (Host, API Key) to config

- [x] **Planka Integration (Advanced)**
    - [x] Research Planka API
    - [x] Implement "Move to Planka" action (Sync)
    - [x] Automatically create a card in Planka from assigned issues
    - [x] CLI/Cron support for auto-sync (`--sync`)
