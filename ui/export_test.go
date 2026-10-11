package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"redmine-tui/config"
	"redmine-tui/redmine"
)

// exportToHTML interpolates issue-controlled text; it must be HTML-escaped.
func TestExportEscapesHTML(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "xdg-open"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	var iss redmine.Issue
	iss.ID = 7
	iss.Subject = `<script>alert(1)</script>`
	iss.Status.Name = `Open"><script>x</script>`
	iss.Project.Name = `<b>proj</b>`
	iss.Priority.Name = `<i>urgent</i>`

	m := Model{cfg: &config.Config{}, allIssues: []redmine.Issue{iss}}
	if err := m.exportToHTML(); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "redmine_issues.html"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	for _, bad := range []string{"<script>", "<b>proj</b>", "<i>urgent</i>", `<img`} {
		if strings.Contains(out, bad) {
			t.Fatalf("unescaped markup leaked into export: %q", bad)
		}
	}
	for _, want := range []string{"&lt;script&gt;", "&lt;b&gt;proj&lt;/b&gt;"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected escaped %q in export", want)
		}
	}
}
