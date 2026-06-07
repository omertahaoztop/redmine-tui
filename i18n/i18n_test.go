package i18n

import "testing"

func TestTranslationAndFallback(t *testing.T) {
	SetLang("tr")
	if got := T("key.quit"); got != "çıkış" {
		t.Errorf("TR key.quit = %q, want çıkış", got)
	}

	SetLang("en")
	if got := T("key.quit"); got != "quit" {
		t.Errorf("EN key.quit = %q, want quit", got)
	}

	// Unknown language falls back to English.
	SetLang("xx")
	if Current() != EN {
		t.Errorf("unknown lang should fall back to EN, got %s", Current())
	}

	// Missing key returns the key itself.
	if got := T("totally.missing"); got != "totally.missing" {
		t.Errorf("missing key = %q, want echo of key", got)
	}
}
