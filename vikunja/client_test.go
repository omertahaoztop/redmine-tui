package vikunja

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"redmine-tui/internal/netguard"
	"strings"
	"testing"
)

// A bearer token (or the password on login) rides on every request, so a
// plaintext endpoint may only be the local machine.
func TestNewClientRejectsPlaintextRemoteEndpoint(t *testing.T) {
	rejected := []string{
		"http://vikunja.example.com",
		"http://10.0.0.5:3456",
		"http://10.0.0.5:3456/api/v1",
	}
	for _, u := range rejected {
		if _, err := NewClient(u, "tk", "", ""); err == nil {
			t.Errorf("NewClient(%q) accepted a plaintext remote endpoint", u)
		}
	}

	accepted := map[string]string{
		"https://vikunja.example.com": "https://vikunja.example.com/api/v1",
		"vikunja.example.com":         "https://vikunja.example.com/api/v1",
		"http://127.0.0.1:3456":       "http://127.0.0.1:3456/api/v1",
		"http://localhost:3456/api":   "http://localhost:3456/api/v1",
	}
	for in, want := range accepted {
		c, err := NewClient(in, "tk", "", "")
		if err != nil {
			t.Errorf("NewClient(%q) failed: %v", in, err)
			continue
		}
		if c.BaseURL != want {
			t.Errorf("NewClient(%q).BaseURL = %q, want %q", in, c.BaseURL, want)
		}
	}
}

func TestGetViewsDecodesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tk" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]View{{ID: 7, ViewKind: "kanban"}})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "tk", "", "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	views, err := c.GetViews(1)
	if err != nil {
		t.Fatalf("GetViews failed: %v", err)
	}
	if len(views) != 1 || views[0].ID != 7 {
		t.Fatalf("views = %+v", views)
	}
}

// net/http gunzips transparently, so without a budget on the decoded size a
// small substituted response can expand into an arbitrarily large allocation.
func TestDecodedResponseIsBounded(t *testing.T) {
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	for _, part := range [][]byte{
		[]byte(`[{"id":1,"title":"`),
		bytes.Repeat([]byte("a"), netguard.MaxResponseBytes+1),
		[]byte(`","tasks":[]}]`),
	} {
		// A truncated fixture would make the test pass for the wrong reason.
		if _, err := zw.Write(part); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if compressed.Len() > 1<<20 {
		t.Fatalf("fixture is not actually compressed: %d bytes", compressed.Len())
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		if _, err := w.Write(compressed.Bytes()); err != nil {
			t.Errorf("delivering fixture: %v", err)
		}
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "tk", "", "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := c.GetBucketTasks(1, 2, 3); err == nil {
		t.Fatal("a response larger than the decode budget was accepted")
	}
}

// A remote error body reaches log records and the TUI status bar, so it has to
// arrive as one bounded, escaped line.
func TestDiagnosticBodyEscapesAndBounds(t *testing.T) {
	got := diagnosticBody(strings.NewReader("create failed\n\x1b]52;c;cGF3bmVk\x07"))
	if strings.IndexFunc(got, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		t.Fatalf("control characters survived: %q", got)
	}
	if !strings.Contains(got, `\n`) {
		t.Fatalf("line break not visible in %q", got)
	}

	long := diagnosticBody(strings.NewReader(strings.Repeat("x", 4*maxDiagnosticBytes)))
	if !strings.HasSuffix(long, " (truncated)") {
		t.Fatalf("truncation not reported: %q", long)
	}
	if len(long) > maxDiagnosticBytes+32 {
		t.Fatalf("diagnostic body not bounded: %d bytes", len(long))
	}
}
