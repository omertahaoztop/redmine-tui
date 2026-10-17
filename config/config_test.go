package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// The working directory is not a trusted config source: a file dropped next to
// the binary (a cloned repo, a shared temp dir) can name the host that receives
// credentials inherited from the environment.
func TestWorkingDirectoryConfigIsIgnored(t *testing.T) {
	viper.Reset()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("REDMINE_TUI_CONFIG", "")
	t.Setenv("REDMINE_HOST", "")
	t.Setenv("REDMINE_API_KEY", "")

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".redmine-tui.yaml"),
		[]byte("host: \"http://evil.example\"\napi_key: \"stolen\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Host != "" || cfg.APIKey != "" {
		t.Fatalf("working-directory config was loaded: host=%q api_key=%q", cfg.Host, cfg.APIKey)
	}
}

// A config outside $HOME and /etc/default has to be selected explicitly.
func TestExplicitConfigPathIsUsed(t *testing.T) {
	viper.Reset()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("REDMINE_HOST", "")
	t.Setenv("REDMINE_API_KEY", "")

	path := filepath.Join(t.TempDir(), "custom.yaml")
	if err := os.WriteFile(path, []byte("host: \"redmine.example.com\"\napi_key: \"k\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REDMINE_TUI_CONFIG", path)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Host != "redmine.example.com" || cfg.APIKey != "k" {
		t.Fatalf("explicit config not used: host=%q api_key=%q", cfg.Host, cfg.APIKey)
	}
}
