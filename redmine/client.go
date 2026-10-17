package redmine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"redmine-tui/internal/netguard"
	"strconv"
	"strings"
	"time"
)

type IssueStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type IssueStatusesResponse struct {
	IssueStatuses []IssueStatus `json:"issue_statuses"`
}

type Issue struct {
	// Preserve the source title for reconciliation, which historically matches
	// Vikunja tasks by the exact Redmine subject rather than its display text.
	originalSubject string

	ID          int       `json:"id"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	DoneRatio   int       `json:"done_ratio"`
	DueDate     string    `json:"due_date"`
	StartDate   string    `json:"start_date"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
	Tracker     struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"tracker"`
	Status struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"status"`
	Priority struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"priority"`
	Project struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
	Author struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"author"`
	AssignedTo struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"assigned_to"`
	Journals []Journal `json:"journals"`
}

// OriginalSubject returns the unmodified subject used to match Vikunja tasks.
// It may contain terminal controls and must be quoted when included in errors.
func (i Issue) OriginalSubject() string {
	if i.originalSubject != "" {
		return i.originalSubject
	}
	return i.Subject
}

type Journal struct {
	ID   int `json:"id"`
	User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"user"`
	Notes     string          `json:"notes"`
	CreatedOn time.Time       `json:"created_on"`
	Details   []JournalDetail `json:"details"`
}

type JournalDetail struct {
	Property string `json:"property"`
	Name     string `json:"name"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// IssuesResponse carries the pagination envelope Redmine returns for every
// collection endpoint, so a caller can tell a complete list from a truncated
// one. TotalCount and Offset are pointers because the sync path must tell an
// absent field from a zero value: a response without them is indistinguishable
// from an empty collection, and reading it as "no issues" would delete or move
// every task reconciliation covers.
type IssuesResponse struct {
	Issues     []Issue `json:"issues"`
	TotalCount *int    `json:"total_count"`
	Offset     *int    `json:"offset"`
}

type IssueResponse struct {
	Issue Issue `json:"issue"`
}

// sanitizeText strips terminal control characters from remote text. Redmine
// fields are written by other people and end up in the terminal, in the HTML
// export and in log lines: an ESC in a subject can drive the terminal (OSC 52
// clipboard writes among others) and a newline in a title forges an extra log
// record. Newlines survive so multi-line descriptions still render.
func sanitizeText(s string) string {
	if strings.IndexFunc(s, isControl) < 0 {
		return s
	}
	return strings.Map(func(r rune) rune {
		// A tab cannot start an escape sequence, so keep it as the word break
		// it stood for instead of welding the words together.
		if r == '\t' {
			return ' '
		}
		if isControl(r) {
			return -1
		}
		return r
	}, s)
}

func isControl(r rune) bool {
	switch {
	case r == '\n':
		return false
	case r < 0x20, r == 0x7f:
		return true
	case r >= 0x80 && r <= 0x9f: // C1, including the 8-bit CSI 0x9b
		return true
	default:
		return false
	}
}

func sanitizeIssue(i Issue) Issue {
	if i.originalSubject == "" {
		i.originalSubject = i.Subject
	}
	i.Subject = sanitizeText(i.Subject)
	i.Description = sanitizeText(i.Description)
	i.Project.Name = sanitizeText(i.Project.Name)
	i.Status.Name = sanitizeText(i.Status.Name)
	i.Priority.Name = sanitizeText(i.Priority.Name)
	i.Tracker.Name = sanitizeText(i.Tracker.Name)
	i.Author.Name = sanitizeText(i.Author.Name)
	i.AssignedTo.Name = sanitizeText(i.AssignedTo.Name)
	for n := range i.Journals {
		i.Journals[n].Notes = sanitizeText(i.Journals[n].Notes)
		i.Journals[n].User.Name = sanitizeText(i.Journals[n].User.Name)
	}
	return i
}

func sanitizeIssues(issues []Issue) []Issue {
	for i := range issues {
		issues[i] = sanitizeIssue(issues[i])
	}
	return issues
}

type Client struct {
	APIKey string
	Host   string
	Client *http.Client
}

// normalizeHost validates the configured Redmine host and adds the default
// scheme. The API key rides on every request, so a remote plaintext endpoint
// would hand it to anyone on the path; loopback is exempt because that traffic
// never leaves the host.
func normalizeHost(host string) (string, error) {
	clean := strings.TrimRight(strings.TrimSpace(host), "/")
	if !strings.HasPrefix(clean, "http://") && !strings.HasPrefix(clean, "https://") {
		clean = "https://" + clean
	}
	u, err := url.Parse(clean)
	if err != nil {
		return "", fmt.Errorf("invalid redmine host %q: %w", host, err)
	}
	if err := netguard.RequireSecureTransport(u, host); err != nil {
		return "", err
	}
	return clean, nil
}

// userAgent is required. Cloudflare-fronted Redmine instances answer the Go
// default ("Go-http-client/1.1") with "error code: 1010" — a banned browser
// signature — before the API key is ever checked.
const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

// newRequest builds a request with the headers every Redmine call needs.
func (c *Client) newRequest(method, url string, body []byte) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Redmine-API-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	return req, nil
}

func NewClient(apiKey, host string) (*Client, error) {
	normalized, err := normalizeHost(host)
	if err != nil {
		return nil, err
	}
	return &Client{
		APIKey: apiKey,
		Host:   normalized,
		Client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: netguard.CheckRedirect},
	}, nil
}

const (
	// issuesPageSize is the largest page Redmine serves per request.
	issuesPageSize = 100
	// maxIssuesPages bounds the walk. A server that keeps answering with full
	// pages while reporting no usable total would otherwise loop forever.
	maxIssuesPages = 200
)

// GetAssignedIssues walks every page of open issues for display. The result can
// change between requests and must not be used for absence-based reconciliation.
func (c *Client) GetAssignedIssues() ([]Issue, error) {
	return c.getAssignedIssues(false)
}

// GetAssignedIssuesForSync requires the complete collection in one response.
// Without snapshot pagination, combining pages could omit a live issue whose
// task would then be deleted or moved by reconciliation.
func (c *Client) GetAssignedIssuesForSync() ([]Issue, error) {
	return c.getAssignedIssues(true)
}

func (c *Client) getAssignedIssues(singlePage bool) ([]Issue, error) {
	var all []Issue
	seen := make(map[int]struct{})

	// The offset advances by what the server actually returned, not by the page
	// size we asked for: a server may cap the page below it, and skipping the
	// gap would hide issues from the board.
	offset := 0
	for page := 0; ; page++ {
		if page == maxIssuesPages {
			return nil, fmt.Errorf("assigned issues exceed %d pages; refusing a partial collection", maxIssuesPages)
		}

		u, err := url.Parse(c.Host + "/issues.json")
		if err != nil {
			return nil, err
		}
		q := u.Query()
		q.Set("assigned_to_id", "me")
		q.Set("status_id", "open")
		q.Set("limit", strconv.Itoa(issuesPageSize))
		q.Set("offset", strconv.Itoa(offset))
		// A stable order keeps the walk deterministic. It cannot make the walk
		// atomic: a status or assignment change between requests may shift the
		// window and omit an issue. Sync therefore requires a single page.
		q.Set("sort", "id:asc")
		u.RawQuery = q.Encode()

		req, err := c.newRequest("GET", u.String(), nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.Client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
		}

		var result IssuesResponse
		if err := netguard.DecodeJSON(resp.Body, &result); err != nil {
			return nil, err
		}
		if singlePage {
			// Metadata must be present, not merely zero: an empty issues array
			// without it would otherwise pass as a complete, empty collection.
			if result.Offset == nil || *result.Offset != 0 ||
				result.TotalCount == nil || *result.TotalCount != len(result.Issues) {
				return nil, fmt.Errorf("assigned issues require a single complete page with pagination metadata for sync; refusing a non-snapshot collection")
			}
		}

		// The offset walk can return an issue from an earlier page when the
		// assigned/open set changes between requests; display it only once.
		for _, issue := range result.Issues {
			if _, seen := seen[issue.ID]; seen {
				continue
			}
			seen[issue.ID] = struct{}{}
			all = append(all, issue)
		}
		offset += len(result.Issues)

		// An empty page proves the collection is exhausted; total_count is the
		// authoritative end when the server reports it.
		if len(result.Issues) == 0 {
			break
		}
		if result.TotalCount != nil && *result.TotalCount > 0 && offset >= *result.TotalCount {
			break
		}
	}

	return sanitizeIssues(all), nil
}

func (c *Client) GetIssueDetails(id int) (*Issue, error) {
	url := fmt.Sprintf("%s/issues/%d.json?include=journals", c.Host, id)

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result IssueResponse
	if err := netguard.DecodeJSON(resp.Body, &result); err != nil {
		return nil, err
	}

	issue := sanitizeIssue(result.Issue)
	return &issue, nil
}

func (c *Client) GetIssueStatuses() ([]IssueStatus, error) {
	url := fmt.Sprintf("%s/issue_statuses.json", c.Host)

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result IssueStatusesResponse
	if err := netguard.DecodeJSON(resp.Body, &result); err != nil {
		return nil, err
	}

	statuses := make([]IssueStatus, len(result.IssueStatuses))
	for i, s := range result.IssueStatuses {
		s.Name = sanitizeText(s.Name)
		statuses[i] = s
	}
	return statuses, nil
}

func (c *Client) UpdateIssueStatus(issueID, statusID int) error {
	url := fmt.Sprintf("%s/issues/%d.json", c.Host, issueID)

	payload := map[string]interface{}{
		"issue": map[string]int{
			"status_id": statusID,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := c.newRequest("PUT", url, body)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) LogTime(issueID int, hours string, comments string) error {
	url := fmt.Sprintf("%s/time_entries.json", c.Host)

	payload := map[string]interface{}{
		"time_entry": map[string]interface{}{
			"issue_id": issueID,
			"hours":    hours,
			"comments": comments,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := c.newRequest("POST", url, body)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) SearchIssues(query string) ([]Issue, error) {
	// The query is user input, so it must be URL-encoded; building the raw
	// string would let "&" or "#" inject extra filter parameters.
	fetch := func(field string) ([]Issue, error) {
		u, err := url.Parse(c.Host + "/issues.json")
		if err != nil {
			return nil, err
		}
		q := u.Query()
		q.Set("assigned_to_id", "me")
		q.Set("status_id", "open")
		q.Set("limit", "100")
		q.Set(field, "~"+query)
		u.RawQuery = q.Encode()

		req, err := c.newRequest("GET", u.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.Client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
		}
		var result IssuesResponse
		if err := netguard.DecodeJSON(resp.Body, &result); err != nil {
			return nil, err
		}
		return sanitizeIssues(result.Issues), nil
	}

	subjectIssues, err := fetch("subject")
	if err != nil {
		return nil, err
	}
	descIssues, err := fetch("description")
	if err != nil {
		return subjectIssues, nil
	}

	seen := make(map[int]bool)
	var merged []Issue
	for _, issue := range append(subjectIssues, descIssues...) {
		if !seen[issue.ID] {
			seen[issue.ID] = true
			merged = append(merged, issue)
		}
	}
	return merged, nil
}
