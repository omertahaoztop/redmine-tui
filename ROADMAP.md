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
- [x] **Command Palette** — `:` / `Ctrl+p`, fuzzy-searched actions with their shortcuts.
- [x] **Help Overlay** — `?`, keymap grouped by task.
- [x] **One-key Status Move** — `H` / `L` moves the focused issue a status back or forward, with the cursor following the card.
- [x] **Sorting** — `S` cycles priority → due date → updated → project → server order.
- [x] **Status Colors** — columns and cards take the semantic color of their status.
- [x] **Human Dates** — relative timestamps in detail and history; "3d left" / "2d overdue" on cards.
- [x] **Markdown-lite** — headings, bullets, bold, inline code and links in descriptions and notes.
- [x] **Narrow Terminal Mode** — one column plus a tab strip below 80 columns.
- [x] **Loading & Empty States** — skeleton board with a header spinner; empty state that teaches the keys.
- [x] **Opt-in Mouse** — `mouse: true` for click-to-select, column headers and wheel scrolling.
- [x] **Security Hardening** — dependency CVEs patched, URL-encoded search, HTML-escaped export, config permission warning.
- [x] **CI** — GitHub Actions for build, vet, and test.

## Planned

- [ ] **Create Issue** — `n` to create a new issue (`POST /issues.json`).
- [ ] **Standalone Comment** — add a note without changing status.
- [ ] **Edit Done Ratio** — adjust progress from the detail view.
- [ ] **Toggle Closed Issues** — show closed/all, not just open.
- [ ] **Confirmation Dialog** — guard destructive status/time actions.
- [ ] **Offline Cache** — show last fetch instantly, refresh in background.
- [ ] **Configurable Theme** — user-defined colors via config.
- [ ] **More Languages** — community translations beyond EN/TR.
