package ui

import "redmine-tui/i18n"

// binding is one row of the help overlay and the footer hint line. It is the
// single place a key is described; dispatch lives in Model.handleKey.
type binding struct {
	keys string // display form, e.g. "←→ / h l"
	desc string // i18n key
}

type bindingGroup struct {
	title string // i18n key
	items []binding
}

func keymapGroups() []bindingGroup {
	return []bindingGroup{
		{i18n.T("help.group.nav"), []binding{
			{"←→ / h l", "key.col"},
			{"↑↓ / k j", "key.card"},
			{"g", "key.top"},
			{"G", "key.bottom"},
			{"enter / space", "key.detail"},
			{"esc", "key.back"},
		}},
		{i18n.T("help.group.act"), []binding{
			{"s", "key.status"},
			{"H / L", "key.move"},
			{"t", "key.time"},
			{"y", "key.copy"},
			{"o", "key.open"},
		}},
		{i18n.T("help.group.vw"), []binding{
			{"^f", "key.search"},
			{"f", "key.filter"},
			{"S", "key.sort"},
			{"e", "key.export"},
			{"v", "key.vikunja"},
			{"r", "key.refresh"},
		}},
		{i18n.T("help.group.glo"), []binding{
			{": / ^p", "key.palette"},
			{"?", "key.help"},
			{"q / ^c", "key.quit"},
		}},
	}
}

// footerKeys is the compact hint line: the keys worth remembering without
// opening the help overlay.
func footerKeys() []binding {
	return []binding{
		{"←→", "key.col"}, {"↑↓", "key.card"}, {"↵", "key.detail"},
		{"s", "key.status"}, {"t", "key.time"}, {"f", "key.filter"},
		{"o", "key.open"}, {"r", "key.refresh"}, {"^f", "key.search"},
		{":", "key.palette"}, {"?", "key.help"}, {"q", "key.quit"},
	}
}
