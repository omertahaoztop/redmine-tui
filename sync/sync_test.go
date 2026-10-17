package sync

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	stdsync "sync"
	"sync/atomic"
	"testing"

	"redmine-tui/config"
	"redmine-tui/redmine"
	"redmine-tui/vikunja"
)

// An issue leaving the first page shifts the next offset past another live
// issue. Sync must abort before that second read and before any Vikunja access.
func TestSyncRejectsNonSnapshotCollection(t *testing.T) {
	for _, doneBucketID := range []int64{0, 4} {
		t.Run(fmt.Sprintf("done=%d", doneBucketID), func(t *testing.T) {
			var redmineReads, vikunjaRequests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != "/issues.json" {
					vikunjaRequests.Add(1)
					if err := json.NewEncoder(w).Encode([]vikunja.KanbanBucket{{ID: 3, Tasks: []vikunja.Task{
						{ID: 1, Title: "first"}, {ID: 2, Title: "still open"}, {ID: 3, Title: "last"},
					}}}); err != nil {
						t.Error(err)
					}
					return
				}
				redmineReads.Add(1)
				offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
				if err != nil {
					t.Error(err)
				}
				page := redmine.IssuesResponse{Issues: []redmine.Issue{{ID: 1, Subject: "first"}}, TotalCount: 3, Limit: 1}
				if offset > 0 {
					// Issue 1 is now closed. Offset 1 skips issue 2, which
					// remains eligible and must keep its existing task.
					page = redmine.IssuesResponse{Issues: []redmine.Issue{{ID: 3, Subject: "last"}}, TotalCount: 2, Offset: offset, Limit: 1}
				}
				if err := json.NewEncoder(w).Encode(page); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			rc, err := redmine.NewClient("key", srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			vc, err := vikunja.NewClient(srv.URL, "token", "", "")
			if err != nil {
				t.Fatal(err)
			}
			cfg := &config.Config{Vikunja: config.Vikunja{ProjectID: 1, ViewID: 2, BucketID: 3, DoneBucketID: doneBucketID}}
			err = SyncIssuesToVikunja(rc, vc, cfg)
			if err == nil {
				t.Error("sync accepted an inconsistent collection")
			}
			if got := redmineReads.Load(); got != 1 {
				t.Errorf("Redmine reads = %d, want one", got)
			}
			if got := vikunjaRequests.Load(); got != 0 {
				t.Errorf("Vikunja requests = %d, want none", got)
			}
		})
	}
}

// Existing tasks keep their identity even when two subjects have the same
// sanitized display text. Missing tasks must also retain the original title.
func TestSyncPreservesOriginalSubjects(t *testing.T) {
	for _, existing := range []bool{true, false} {
		for _, doneBucketID := range []int64{0, 4} {
			t.Run(fmt.Sprintf("existing=%t/done=%d", existing, doneBucketID), func(t *testing.T) {
				subjects := []string{"a\tb", "a b", "\x1b]52;c;x\aoriginal\nsubject"}
				var issues []redmine.Issue
				var tasks []vikunja.Task
				for i, subject := range subjects {
					issues = append(issues, redmine.Issue{ID: i + 1, Subject: subject})
					if existing {
						tasks = append(tasks, vikunja.Task{ID: int64(i + 1), Title: subject})
					}
				}
				var created, unexpected []string
				var mu stdsync.Mutex
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == http.MethodGet && r.URL.Path == "/issues.json":
						if err := json.NewEncoder(w).Encode(redmine.IssuesResponse{Issues: issues, TotalCount: len(issues)}); err != nil {
							t.Error(err)
						}
					case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/1/views/2/tasks":
						if err := json.NewEncoder(w).Encode([]vikunja.KanbanBucket{{ID: 3, Tasks: tasks}}); err != nil {
							t.Error(err)
						}
					case r.Method == http.MethodPut && r.URL.Path == "/api/v1/projects/1/tasks":
						var task vikunja.Task
						if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
							t.Error(err)
						}
						created = append(created, task.Title)
						tasks = append(tasks, vikunja.Task{ID: int64(len(tasks) + 1), Title: task.Title})
						w.WriteHeader(http.StatusCreated)
					default:
						unexpected = append(unexpected, r.Method+" "+r.URL.Path)
						w.WriteHeader(http.StatusNoContent)
					}
				}))
				defer srv.Close()
				rc, err := redmine.NewClient("key", srv.URL)
				if err != nil {
					t.Fatal(err)
				}
				vc, err := vikunja.NewClient(srv.URL, "token", "", "")
				if err != nil {
					t.Fatal(err)
				}
				cfg := &config.Config{Vikunja: config.Vikunja{ProjectID: 1, ViewID: 2, BucketID: 3, DoneBucketID: doneBucketID}}
				// Repeated syncs must preserve the same tasks and metadata.
				for range 2 {
					if err := SyncIssuesToVikunja(rc, vc, cfg); err != nil {
						t.Fatal(err)
					}
				}
				srv.Close()
				var want []string
				if !existing {
					want = slices.Clone(subjects)
				}
				slices.Sort(created)
				slices.Sort(want)
				if !slices.Equal(created, want) || len(unexpected) != 0 {
					t.Fatalf("created titles = %q, want %q; unexpected mutations = %q", created, want, unexpected)
				}
			})
		}
	}
}

func TestSyncEscapesSubjectInErrors(t *testing.T) {
	const subject = "\x1b]52;c;x\aoriginal\nsubject"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/issues.json":
			if err := json.NewEncoder(w).Encode(redmine.IssuesResponse{Issues: []redmine.Issue{{ID: 1, Subject: subject}}, TotalCount: 1}); err != nil {
				t.Error(err)
			}
		case "/api/v1/projects/1/views/2/tasks":
			if err := json.NewEncoder(w).Encode([]vikunja.KanbanBucket{{ID: 3}}); err != nil {
				t.Error(err)
			}
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	rc, err := redmine.NewClient("key", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	vc, err := vikunja.NewClient(srv.URL, "token", "", "")
	if err != nil {
		t.Fatal(err)
	}
	err = SyncIssuesToVikunja(rc, vc, &config.Config{Vikunja: config.Vikunja{ProjectID: 1, ViewID: 2, BucketID: 3}})
	if err == nil {
		t.Fatal("expected create failure")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%q", subject)) {
		t.Fatalf("error must quote the original subject: %q", err)
	}
}
