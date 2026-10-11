package redmine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	c := NewClient("secret", srv.URL)
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

	c := NewClient("key", srv.URL)
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
