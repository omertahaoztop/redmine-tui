package redmine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

type IssuesResponse struct {
	Issues []Issue `json:"issues"`
}

type IssueResponse struct {
	Issue Issue `json:"issue"`
}

type Client struct {
	APIKey string
	Host   string
	Client *http.Client
}

func normalizeHost(host string) string {
	// If host already has a scheme, use it as-is (after trimming trailing slash)
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return strings.TrimRight(host, "/")
	}
	// No scheme: default to https
	return "https://" + strings.TrimRight(host, "/")
}

func NewClient(apiKey, host string) *Client {
	return &Client{
		APIKey: apiKey,
		Host:   normalizeHost(host),
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetAssignedIssues() ([]Issue, error) {
	// Filtering by assigned_to_id=me, status=open, and increasing limit to 100
	url := fmt.Sprintf("%s/issues.json?assigned_to_id=me&status_id=open&limit=100", c.Host)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Redmine-API-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result IssuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Issues, nil
}

func (c *Client) GetIssueDetails(id int) (*Issue, error) {
	url := fmt.Sprintf("%s/issues/%d.json?include=journals", c.Host, id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Redmine-API-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Issue, nil
}

func (c *Client) GetIssueStatuses() ([]IssueStatus, error) {
	url := fmt.Sprintf("%s/issue_statuses.json", c.Host)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Redmine-API-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result IssueStatusesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.IssueStatuses, nil
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

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("X-Redmine-API-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

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

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("X-Redmine-API-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

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
	url := fmt.Sprintf("%s/issues.json?assigned_to_id=me&status_id=open&limit=100&description=~%s", c.Host, query)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Redmine-API-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result IssuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Issues, nil
}
