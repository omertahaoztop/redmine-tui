package redmine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// Cloudflare-fronted Redmine instances reject the Go default User-Agent with
// "error code: 1010" before the API key is checked, so every request must
// carry a browser-like one.
// intPtr builds the pagination envelope's optional fields.
func intPtr(v int) *int { return &v }

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

// The board must load every page, even when the server caps its page size below
// the limit the client asks for.
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
		json.NewEncoder(w).Encode(IssuesResponse{Issues: issues, TotalCount: intPtr(total), Offset: intPtr(offset)})
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
// assigned/open set changes between requests; the board must not show it twice.
func TestGetAssignedIssuesDropsRepeatedPages(t *testing.T) {
	pages := [][]int{{1, 2, 3}, {3, 4, 5}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var issues []Issue
		for _, id := range pages[offset/3] {
			issues = append(issues, Issue{ID: id, Subject: fmt.Sprintf("issue %d", id)})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IssuesResponse{Issues: issues, TotalCount: intPtr(5), Offset: intPtr(offset)})
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

func TestGetAssignedIssuesForSyncRequiresCompletePage(t *testing.T) {
	cases := []struct {
		name                 string
		count, total, offset int
		ok                   bool
	}{
		{"empty", 0, 0, 0, true},
		{"single issue", 1, 1, 0, true},
		{"full page", issuesPageSize, issuesPageSize, 0, true},
		{"second page needed", issuesPageSize, issuesPageSize + 1, 0, false},
		{"server caps page", 1, 2, 0, false},
		{"unknown total", 1, 0, 0, false},
		{"empty but incomplete", 0, 1, 0, false},
		{"negative total", 0, -1, 0, false},
		{"underreported total", 2, 1, 0, false},
		{"wrong offset", 1, 1, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) != 1 {
					t.Error("requested another page")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if r.URL.Query().Get("offset") != "0" || r.URL.Query().Get("assigned_to_id") != "me" || r.URL.Query().Get("status_id") != "open" {
					t.Errorf("unexpected collection query: %s", r.URL.RawQuery)
				}
				issues := make([]Issue, 0, tc.count)
				for id := 1; id <= tc.count; id++ {
					issues = append(issues, Issue{ID: id, Subject: "a\tb"})
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(IssuesResponse{Issues: issues, TotalCount: intPtr(tc.total), Offset: intPtr(tc.offset)}); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			c, err := NewClient("key", srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			issues, err := c.GetAssignedIssuesForSync()
			if (err == nil) != tc.ok {
				t.Fatalf("GetAssignedIssuesForSync() error = %v, want success=%t", err, tc.ok)
			}
			if !tc.ok && issues != nil {
				t.Fatalf("returned partial collection: %v", issues)
			}
			if tc.ok && len(issues) != tc.count {
				t.Fatalf("got %d issues, want %d", len(issues), tc.count)
			}
			for _, issue := range issues {
				if issue.Subject != "a b" || issue.OriginalSubject() != "a\tb" {
					t.Fatal("display sanitization or original subject was lost")
				}
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("requested %d pages, want one", got)
			}
		})
	}
}

// A server that ignores the offset must not be walked to the page cap: the same
// page repeats, so there is nothing further to collect.
func TestGetAssignedIssuesStopsWhenOffsetIsIgnored(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IssuesResponse{Issues: []Issue{{ID: 1, Subject: "x"}, {ID: 2, Subject: "y"}}})
	}))
	defer srv.Close()

	c, err := NewClient("key", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	issues, err := c.GetAssignedIssues()
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2", len(issues))
	}
	if n := calls.Load(); n > 2 {
		t.Fatalf("walked %d pages against a server that ignores the offset", n)
	}
}

// A response without pagination metadata cannot be told apart from an empty
// collection, and reading it as "no issues" would delete or move every task
// reconciliation covers. The fields must be present, not merely zero.
func TestGetAssignedIssuesForSyncRequiresPaginationMetadata(t *testing.T) {
	bodies := map[string]string{
		"no fields":    `{"issues":[]}`,
		"total only":   `{"issues":[],"total_count":0}`,
		"offset only":  `{"issues":[],"offset":0}`,
		"null fields":  `{"issues":[],"total_count":null,"offset":null}`,
		"issues only":  `{"issues":[{"id":1,"subject":"x"}]}`,
		"empty object": `{}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte(body)); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()

			c, err := NewClient("key", srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			issues, err := c.GetAssignedIssuesForSync()
			if err == nil {
				t.Fatalf("accepted a collection without pagination metadata: %v", issues)
			}
		})
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
			json.NewEncoder(w).Encode(IssuesResponse{TotalCount: intPtr(1)})
			return
		}
		json.NewEncoder(w).Encode(IssuesResponse{Issues: []Issue{{
			ID:          1,
			Subject:     osc + "real\nsubject",
			Description: "line one\r\nline two\ttabbed",
			Journals:    []Journal{{Notes: osc + "note"}},
		}}, TotalCount: intPtr(1)})
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
