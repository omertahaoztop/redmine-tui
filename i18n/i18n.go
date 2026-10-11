package i18n

import "strings"

// Lang is a supported language code.
type Lang string

const (
	EN Lang = "en"
	TR Lang = "tr"
)

var current = EN

var catalog = map[Lang]map[string]string{
	EN: {
		// App / headers
		"app.title":    "Redmine Kanban",
		"detail.title": "Issue Detail",
		"status.title": "Change Status",
		"time.title":   "Log Time",
		"search.title": "Search Issues",
		"app.loading":  "Loading issues...",
		"app.noissues": "No issues found",
		"app.issues":   "issues",
		"error":        "Error",

		// Detail labels
		"label.project":     "Project",
		"label.status":      "Status",
		"label.tracker":     "Tracker",
		"label.priority":    "Priority",
		"label.due":         "Due",
		"label.author":      "Author",
		"label.assignee":    "Assignee",
		"label.created":     "Created",
		"label.updated":     "Updated",
		"label.progress":    "Progress",
		"label.description": "Description",
		"label.history":     "History",
		"detail.nodesc":     "(no description)",

		// Status / time / search hints
		"hint.detail":      "esc/q: back · s: status · t: time · y: copy · o: open · ↑↓: scroll",
		"hint.status":      "←→: select · enter: apply · esc: back",
		"hint.time":        "enter: next · esc: back",
		"hint.search":      "enter: search · esc: cancel",
		"hint.help":        "?: close · esc: back",
		"hint.palette":     "enter: run · esc: cancel",
		"time.step1":       "Step 1/2: Hours",
		"time.step2":       "Step 2/2: Comments",
		"time.ph.hours":    "Enter hours (e.g. 1.5)",
		"time.ph.comments": "Enter comments...",
		"search.ph":        "Search issues...",
		"search.label":     "Search description:",

		// Footer keys
		"key.col":     "col",
		"key.card":    "card",
		"key.detail":  "detail",
		"key.status":  "status",
		"key.time":    "time",
		"key.filter":  "filter",
		"key.copy":    "copy",
		"key.refresh": "refresh",
		"key.export":  "export",
		"key.vikunja": "sync",
		"key.search":  "search",
		"key.open":    "open",
		"key.quit":    "quit",

		// Status messages
		"msg.copied":         "Copied to clipboard!",
		"msg.status_updated": "Status updated!",
		"msg.time_logged":    "Time logged!",
		"msg.vikunja_synced": "Synced with Vikunja!",
		"msg.syncing":        "Syncing with Vikunja...",
		"msg.exported":       "Exported to redmine_issues.html",
		"msg.filter.all":     "Filter: All projects",
		"msg.filter":         "Filter: %s",
		"msg.due.today":      "Today",
		"msg.opening":        "Opening in browser...",
		"col.nav":            "col %d/%d  ←→ to navigate",
		"more.up":            "↑ more",
		"more.down":          "↓ more",

		// Command palette
		"palette.title":    "Commands",
		"palette.ph":       "Type a command...",
		"palette.noresult": "No matching command",

		// Help overlay
		"help.title":     "Key Bindings",
		"help.group.nav": "Navigation",
		"help.group.act": "Issue actions",
		"help.group.vw":  "Views & tools",
		"help.group.glo": "Global",

		"key.help":    "help",
		"key.palette": "palette",
		"key.sort":    "sort",
		"key.move":    "move status",
		"key.top":     "top",
		"key.bottom":  "bottom",
		"key.back":    "back",

		// Sorting
		"sort.priority": "priority",
		"sort.due":      "due date",
		"sort.updated":  "updated",
		"sort.project":  "project",
		"msg.sort":      "Sort: %s",

		// Relative time
		"rel.now":       "just now",
		"rel.minutes":   "%dm ago",
		"rel.hours":     "%dh ago",
		"rel.yesterday": "yesterday",
		"rel.days":      "%dd ago",

		// Due date, human form
		"due.over":  "%dd overdue",
		"due.today": "due today",
		"due.left":  "%dd left",

		// Empty / error recovery
		"empty.hint": "r refresh · ^f search · f filter · ? help",
		"err.hint":   "Check your API key and host, then press r to retry · q to quit.",

		// Palette commands
		"cmd.refresh": "Refresh issues",
		"cmd.sync":    "Sync to Vikunja",
		"cmd.export":  "Export board to HTML",
		"cmd.search":  "Search issues",
		"cmd.filter":  "Cycle project filter",
		"cmd.sort":    "Cycle sort order",
		"cmd.help":    "Show key bindings",
		"cmd.detail":  "Open issue detail",
		"cmd.status":  "Change issue status",
		"cmd.time":    "Log time to issue",
		"cmd.copy":    "Copy issue subject",
		"cmd.open":    "Open issue in browser",
		"cmd.quit":    "Quit",

		"msg.moved":    "Moved to %s",
		"msg.nomove":   "No previous/next status",
		"sort.none":    "server order",
		"time.invalid": "Invalid hours — enter a number (e.g. 1.5)",
	},
	TR: {
		"app.title":    "Redmine Kanban",
		"detail.title": "İş Detayı",
		"status.title": "Durum Değiştir",
		"time.title":   "Zaman Kaydet",
		"search.title": "İş Ara",
		"app.loading":  "İşler yükleniyor...",
		"app.noissues": "İş bulunamadı",
		"app.issues":   "iş",
		"error":        "Hata",

		"label.project":     "Proje",
		"label.status":      "Durum",
		"label.tracker":     "Tür",
		"label.priority":    "Öncelik",
		"label.due":         "Bitiş",
		"label.author":      "Oluşturan",
		"label.assignee":    "Atanan",
		"label.created":     "Oluşturuldu",
		"label.updated":     "Güncellendi",
		"label.progress":    "İlerleme",
		"label.description": "Açıklama",
		"label.history":     "Geçmiş",
		"detail.nodesc":     "(açıklama yok)",

		"hint.detail":      "esc/q: geri · s: durum · t: zaman · y: kopyala · o: aç · ↑↓: kaydır",
		"hint.status":      "←→: seç · enter: uygula · esc: geri",
		"hint.time":        "enter: ileri · esc: geri",
		"hint.search":      "enter: ara · esc: iptal",
		"hint.help":        "?: kapat · esc: geri",
		"hint.palette":     "enter: çalıştır · esc: iptal",
		"time.step1":       "Adım 1/2: Saat",
		"time.step2":       "Adım 2/2: Yorum",
		"time.ph.hours":    "Saat girin (örn. 1.5)",
		"time.ph.comments": "Yorum girin...",
		"search.ph":        "İş ara...",
		"search.label":     "Açıklamada ara:",

		"key.col":     "sütun",
		"key.card":    "kart",
		"key.detail":  "detay",
		"key.status":  "durum",
		"key.time":    "zaman",
		"key.filter":  "filtre",
		"key.copy":    "kopyala",
		"key.refresh": "yenile",
		"key.export":  "dışa aktar",
		"key.vikunja": "eşitle",
		"key.search":  "ara",
		"key.open":    "aç",
		"key.quit":    "çıkış",

		"msg.copied":         "Panoya kopyalandı!",
		"msg.status_updated": "Durum güncellendi!",
		"msg.time_logged":    "Zaman kaydedildi!",
		"msg.vikunja_synced": "Vikunja ile eşitlendi!",
		"msg.syncing":        "Vikunja ile eşitleniyor...",
		"msg.exported":       "redmine_issues.html dosyasına aktarıldı",
		"msg.filter.all":     "Filtre: Tüm projeler",
		"msg.filter":         "Filtre: %s",
		"msg.due.today":      "Bugün",
		"msg.opening":        "Tarayıcıda açılıyor...",
		"col.nav":            "sütun %d/%d  ←→ ile gez",
		"more.up":            "↑ daha",
		"more.down":          "↓ daha",

		"palette.title":    "Komutlar",
		"palette.ph":       "Komut yazın...",
		"palette.noresult": "Eşleşen komut yok",

		"help.title":     "Tuş Kısayolları",
		"help.group.nav": "Gezinme",
		"help.group.act": "İş eylemleri",
		"help.group.vw":  "Görünüm ve araçlar",
		"help.group.glo": "Genel",

		"key.help":    "yardım",
		"key.palette": "komut paleti",
		"key.sort":    "sırala",
		"key.move":    "durum taşı",
		"key.top":     "başa",
		"key.bottom":  "sona",
		"key.back":    "geri",

		"sort.priority": "öncelik",
		"sort.due":      "bitiş tarihi",
		"sort.updated":  "güncellenme",
		"sort.project":  "proje",
		"msg.sort":      "Sıralama: %s",

		"rel.now":       "az önce",
		"rel.minutes":   "%d dk önce",
		"rel.hours":     "%d sa önce",
		"rel.yesterday": "dün",
		"rel.days":      "%d gün önce",

		"due.over":  "%d gün gecikti",
		"due.today": "bugün bitiyor",
		"due.left":  "%d gün kaldı",

		"empty.hint": "r yenile · ^f ara · f filtre · ? yardım",
		"err.hint":   "API anahtarını ve sunucu adresini kontrol et, sonra r ile yenile · q ile çık.",

		"cmd.refresh": "İşleri yenile",
		"cmd.sync":    "Vikunja ile eşitle",
		"cmd.export":  "Panoyu HTML'e aktar",
		"cmd.search":  "İş ara",
		"cmd.filter":  "Proje filtresini değiştir",
		"cmd.sort":    "Sıralamayı değiştir",
		"cmd.help":    "Tuş kısayollarını göster",
		"cmd.detail":  "İş detayını aç",
		"cmd.status":  "İş durumunu değiştir",
		"cmd.time":    "İşe zaman kaydet",
		"cmd.copy":    "İş konusunu kopyala",
		"cmd.open":    "İşi tarayıcıda aç",
		"cmd.quit":    "Çıkış",

		"msg.moved":    "%s durumuna taşındı",
		"msg.nomove":   "Önceki/sonraki durum yok",
		"sort.none":    "sunucu sırası",
		"time.invalid": "Geçersiz saat — bir sayı girin (örn. 1.5)",
	},
}

// SetLang sets the active language. Unknown values fall back to English.
func SetLang(code string) {
	switch Lang(strings.ToLower(strings.TrimSpace(code))) {
	case TR:
		current = TR
	default:
		current = EN
	}
}

// Current returns the active language code.
func Current() Lang { return current }

// T returns the translation for key in the active language, falling back to
// English and finally to the key itself.
func T(key string) string {
	if v, ok := catalog[current][key]; ok {
		return v
	}
	if v, ok := catalog[EN][key]; ok {
		return v
	}
	return key
}
