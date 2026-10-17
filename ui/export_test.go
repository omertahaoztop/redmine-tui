package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"redmine-tui/config"
	"redmine-tui/redmine"
)

// exportDir returns a temporary working directory whose browser opener is
// stubbed out, and makes it the process working directory for the test.
func exportDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "xdg-open"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	t.Chdir(dir)

	return dir
}

// exportToHTML interpolates issue-controlled text; it must be HTML-escaped.
func TestExportEscapesHTML(t *testing.T) {
	dir := exportDir(t)

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

	b, err := os.ReadFile(filepath.Join(dir, exportFile))
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

// os.Create would follow a symlink at the export path and truncate whatever it
// points at, letting a shared or cloned working directory choose the victim.
func TestExportDoesNotFollowSymlink(t *testing.T) {
	dir := exportDir(t)

	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("important"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(dir, exportFile)); err != nil {
		// Windows needs a privilege for this; any other failure is a real one.
		if runtime.GOOS == "windows" && os.IsPermission(err) {
			t.Skipf("symlinks unavailable: %v", err)
		}
		t.Fatal(err)
	}

	var iss redmine.Issue
	iss.ID = 1
	iss.Subject = "subject"

	m := Model{cfg: &config.Config{}, allIssues: []redmine.Issue{iss}}
	if err := m.exportToHTML(); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	if b, err := os.ReadFile(victim); err != nil || string(b) != "important" {
		t.Fatalf("symlink target was written through: %q (%v)", b, err)
	}
	info, err := os.Lstat(filepath.Join(dir, exportFile))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("export path is still a symlink")
	}
}
