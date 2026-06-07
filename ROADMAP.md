# Redmine TUI Roadmap & To-Do

This document tracks the planned features and improvements for the Redmine TUI project.

## Done

- [x] **Detail View** — full description, comment history, metadata.
- [x] **Status Updates & Workflow** — `PUT /issues/{id}.json`, status picker.
- [x] **Time Logging** — modal for hours + comments, with input validation.
- [x] **Advanced Filtering & Search** — project filter, server-side subject/description search.
- [x] **Configuration Management** — `~/.redmine-tui.yaml`, `/etc/default`, env vars.
- [x] **Vikunja Integration** — sync assigned issues to a kanban bucket; CLI `--sync` for cron.
- [x] **Modern UI** — refreshed palette, rounded cards, badges, progress bars.
- [x] **Multi-language** — English + Turkish, locale-driven labels and messages.
- [x] **Open in Browser** — jump to an issue in Redmine (`o`).
- [x] **Transient Status Messages** — auto-clearing toast feedback.
- [x] **Rune-safe Truncation** — no broken multibyte characters in cards.

## Planned

- [ ] **Create Issue** — `n` to create a new issue (`POST /issues.json`).
- [ ] **Standalone Comment** — add a note without changing status.
- [ ] **Edit Done Ratio** — adjust progress from the detail view.
- [ ] **Toggle Closed Issues** — show closed/all, not just open.
- [ ] **Confirmation Dialog** — guard destructive status/time actions.
- [ ] **Offline Cache** — show last fetch instantly, refresh in background.
- [ ] **Configurable Theme** — user-defined colors via config.
- [ ] **More Languages** — community translations beyond EN/TR.
- [ ] **CI** — GitHub Actions for build, vet, and test.
