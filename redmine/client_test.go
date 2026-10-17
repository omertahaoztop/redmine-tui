package redmine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Cloudflare-fronted Redmine instances reject the Go default User-Agent with
// "error code: 1010" before the API key is checked, so every request must
// carry a browser-like one.
func TestRequestsCarryBrowserUserAgent(t *testing.T) {
	var ua, key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		key = r.Header.Get("X-Redmine-API-Key")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IssuesResponse{})
	}))
	defer srv.Close()

	c, err := NewClient("secret", srv.URL)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := c.GetAssignedIssues(); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if ua == "" || strings.HasPrefix(ua, "Go-http-client") {
		t.Fatalf("User-Agent not overridden: %q", ua)
	}
	if key != "secret" {
		t.Fatalf("API key header missing: %q", key)
	}
}

// SearchIssues takes user input; it must be URL-encoded so "&" or "#" cannot
// inject extra filter parameters.
func TestSearchIssuesEncodesQuery(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.Query().Get("subject")+"|"+r.URL.Query().Get("description"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IssuesResponse{Issues: []Issue{{ID: 1, Subject: "x"}}})
	}))
	defer srv.Close()

	c, err := NewClient("key", srv.URL)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := c.SearchIssues("a&b#c d"); err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(got))
	}
	if got[0] != "~a&b#c d|" || got[1] != "|~a&b#c d" {
		t.Fatalf("query mangled: %q %q", got[0], got[1])
	}
}

// GetAssignedIssues feeds reconciliation in sync, which treats an issue missing
// from the list as closed and deletes its Vikunja task — so every page must come
// back, not just the first one. The server here caps the page size below the
// limit the client asks for, which must not look like the end of the collection.
func TestGetAssignedIssuesWalksEveryPage(t *testing.T) {
	const (
		total   = 150
		pageCap = 40
	)
	var offsets []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		offsets = append(offsets, r.URL.Query().Get("offset"))

		var issues []Issue
		for i := offset; i < offset+pageCap && i < total; i++ {
			issues = append(issues, Issue{ID: i, Subject: fmt.Sprintf("issue %d", i)})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IssuesResponse{Issues: issues, TotalCount: total, Offset: offset, Limit: pageCap})
	}))
	defer srv.Close()

	c, err := NewClient("key", srv.URL)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	got, err := c.GetAssignedIssues()
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(got) != total {
		t.Fatalf("got %d issues, want %d", len(got), total)
	}
	if want := []string{"0", "40", "80", "120"}; !slices.Equal(offsets, want) {
		t.Fatalf("pages requested = %v, want %v", offsets, want)
	}
}

// The offset walk can hand back an issue from an earlier page when the
// assigned/open set changes between requests; sync must not see it twice.
func TestGetAssignedIssuesDropsRepeatedPages(t *testing.T) {
	pages := [][]int{{1, 2, 3}, {3, 4, 5}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var issues []Issue
		for _, id := range pages[offset/3] {
			issues = append(issues, Issue{ID: id, Subject: fmt.Sprintf("issue %d", id)})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IssuesResponse{Issues: issues, TotalCount: 5, Offset: offset, Limit: 3})
	}))
	defer srv.Close()

	c, err := NewClient("key", srv.URL)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	got, err := c.GetAssignedIssues()
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	seen := make(map[int]bool)
	for _, issue := range got {
		if seen[issue.ID] {
			t.Fatalf("issue %d returned twice", issue.ID)
		}
		seen[issue.ID] = true
	}
	if len(got) != 5 {
		t.Fatalf("got %d issues, want 5", len(got))
	}
}

// Issue text is authored by other people and is drawn straight to the terminal:
// an ESC sequence in a subject can drive it (OSC 52 clipboard writes among
// others) and a newline in a title forges an extra log record.
func TestRemoteTextLosesControlCharacters(t *testing.T) {
	const osc = "\x1b]52;c;cGF3bmVk\x07"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("offset") != "0" {
			json.NewEncoder(w).Encode(IssuesResponse{TotalCount: 1})
			return
		}
		json.NewEncoder(w).Encode(IssuesResponse{Issues: []Issue{{
			ID:          1,
			Subject:     osc + "real\nsubject",
			Description: "line one\r\nline two\ttabbed",
			Journals:    []Journal{{Notes: osc + "note"}},
		}}, TotalCount: 1})
	}))
	defer srv.Close()

	c, err := NewClient("key", srv.URL)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	issues, err := c.GetAssignedIssues()
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	got := issues[0]
	// The ESC introducer is what makes a terminal act on the sequence, so no
	// control byte may survive; the residue is inert text.
	for name, s := range map[string]string{
		"subject":     got.Subject,
		"description": got.Description,
		"notes":       got.Journals[0].Notes,
	} {
		if strings.IndexFunc(s, func(r rune) bool { return r != '\n' && (r < 0x20 || r == 0x7f) }) >= 0 {
			t.Fatalf("%s kept a control character: %q", name, s)
		}
	}
	if !strings.HasSuffix(got.Subject, "real\nsubject") {
		t.Errorf("subject = %q, want the visible text preserved", got.Subject)
	}
	if want := "line one\nline two tabbed"; got.Description != want {
		t.Errorf("description = %q, want %q", got.Description, want)
	}
	if want := "]52;c;cGF3bmVknote"; got.Journals[0].Notes != want {
		t.Errorf("notes = %q, want %q", got.Journals[0].Notes, want)
	}
}

// When the description search fails, SearchIssues falls back to the subject
// results; that path must sanitize too.
func TestSearchIssuesFallbackIsSanitized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("description") != "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IssuesResponse{Issues: []Issue{{ID: 1, Subject: "\x1b]52;c;x\x07subject"}}})
	}))
	defer srv.Close()

	c, err := NewClient("key", srv.URL)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	issues, err := c.SearchIssues("q")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if strings.IndexFunc(issues[0].Subject, func(r rune) bool { return r < 0x20 }) >= 0 {
		t.Fatalf("control character survived the fallback path: %q", issues[0].Subject)
	}
}

// The API key rides on every request, so a remote plaintext host is refused.
func TestNewClientRejectsPlaintextRemoteHost(t *testing.T) {
	if _, err := NewClient("key", "http://redmine.example.com"); err == nil {
		t.Fatal("plaintext remote host accepted")
	}
	c, err := NewClient("key", "redmine.example.com")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if want := "https://redmine.example.com"; c.Host != want {
		t.Fatalf("host = %q, want %q", c.Host, want)
	}
}
