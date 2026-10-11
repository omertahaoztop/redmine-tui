package redmine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
