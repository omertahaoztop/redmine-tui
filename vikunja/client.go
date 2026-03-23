package vikunja

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ── Types ────────────────────────────────────────────────────────────────────────────────

type Task struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Done      bool   `json:"done"`
	BucketID  int64  `json:"bucket_id"`
	ProjectID int64  `json:"project_id"`
}

type KanbanBucket struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Tasks []Task `json:"tasks"`
}

type View struct {
	ID              int64  `json:"id"`
	ViewKind        string `json:"view_kind"`
	DoneBucketID    int64  `json:"done_bucket_id"`
	DefaultBucketID int64  `json:"default_bucket_id"`
}

type Client struct {
	BaseURL  string
	Token    string
	Username string
	Password string
	Client   *http.Client
}

// ── Constructor ──────────────────────────────────────────────────────────────────────────

// ensureAPIv1 normalises the base URL to end with /api/v1.
func ensureAPIv1(url string) string {
	clean := strings.TrimRight(url, "/")
	if !strings.HasPrefix(clean, "http://") && !strings.HasPrefix(clean, "https://") {
		clean = "https://" + clean
	}
	if strings.HasSuffix(clean, "/api/v1") {
		return clean
	}
	if strings.HasSuffix(clean, "/api") {
		return clean + "/v1"
	}
	return clean + "/api/v1"
}

func NewClient(baseURL, token, username, password string) *Client {
	return &Client{
		BaseURL:  ensureAPIv1(baseURL),
		Token:    token,
		Username: username,
		Password: password,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// ── Auth ─────────────────────────────────────────────────────────────────────────────────

// Login authenticates with username/password if no API token was provided.
func (c *Client) Login() error {
	if c.Token != "" {
		return nil
	}

	url := fmt.Sprintf("%s/login", c.BaseURL)
	payload := map[string]string{
		"username": c.Username,
		"password": c.Password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.Client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	token, ok := result["token"].(string)
	if !ok {
		return fmt.Errorf("token not found in login response")
	}

	c.Token = token
	return nil
}

// ── HTTP helper ──────────────────────────────────────────────────────────────────────────

func (c *Client) doRequest(method, url string, payload interface{}) (*http.Response, error) {
	var req *http.Request
	var err error

	if payload != nil {
		b, merr := json.Marshal(payload)
		if merr != nil {
			return nil, merr
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(b))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	return c.Client.Do(req)
}

// ── Views ────────────────────────────────────────────────────────────────────────────────

// GetViews returns all views for the given project.
func (c *Client) GetViews(projectID int64) ([]View, error) {
	url := fmt.Sprintf("%s/projects/%d/views", c.BaseURL, projectID)

	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch views: %d", resp.StatusCode)
	}

	var views []View
	if err := json.NewDecoder(resp.Body).Decode(&views); err != nil {
		return nil, err
	}

	return views, nil
}

// FindKanbanView returns the first kanban-type view in the given project.
func (c *Client) FindKanbanView(projectID int64) (*View, error) {
	views, err := c.GetViews(projectID)
	if err != nil {
		return nil, err
	}

	for _, v := range views {
		if v.ViewKind == "kanban" {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("no kanban view found for project %d", projectID)
}

// ── Tasks ────────────────────────────────────────────────────────────────────────────────

func (c *Client) GetBucketTasks(projectID, viewID, bucketID int64) ([]Task, error) {
	var allTasks []Task
	page := 1
	const perPage = 50

	for {
		url := fmt.Sprintf("%s/projects/%d/views/%d/tasks?page=%d&per_page=%d", c.BaseURL, projectID, viewID, page, perPage)

		resp, err := c.doRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch tasks: %d", resp.StatusCode)
		}

		var buckets []KanbanBucket
		if err := json.NewDecoder(resp.Body).Decode(&buckets); err != nil {
			resp.Body.Close()
			return nil, err
		}

		totalPagesStr := resp.Header.Get("X-Pagination-Total-Pages")
		resp.Body.Close()

		for _, b := range buckets {
			if b.ID == bucketID {
				allTasks = append(allTasks, b.Tasks...)
				break
			}
		}

		totalPages := 1
		if totalPagesStr != "" {
			if tp, err := strconv.Atoi(totalPagesStr); err == nil {
				totalPages = tp
			}
		}

		if page >= totalPages {
			break
		}

		page++
	}

	return allTasks, nil
}

// CreateTask creates a new task in the project and places it into the specified bucket.
func (c *Client) CreateTask(projectID, viewID, bucketID int64, title string) error {
	createURL := fmt.Sprintf("%s/projects/%d/tasks", c.BaseURL, projectID)
	payload := map[string]interface{}{
		"title":     title,
		"bucket_id": bucketID,
	}

	resp, err := c.doRequest("PUT", createURL, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("create task failed with status: %d, body: %s", resp.StatusCode, buf.String())
	}

	return nil
}

// DeleteTask removes a task by ID.
func (c *Client) DeleteTask(taskID int64) error {
	url := fmt.Sprintf("%s/tasks/%d", c.BaseURL, taskID)

	resp, err := c.doRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete task failed with status: %d", resp.StatusCode)
	}

	return nil
}

// MoveTask moves an existing task into a different bucket.
func (c *Client) MoveTask(projectID, viewID, taskID, targetBucketID int64) error {
	url := fmt.Sprintf("%s/projects/%d/views/%d/buckets/%d", c.BaseURL, projectID, viewID, targetBucketID)
	payload := map[string]interface{}{
		"task_id": taskID,
	}

	resp, err := c.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusPreconditionFailed {
		return fmt.Errorf("move task failed with status: %d", resp.StatusCode)
	}

	return nil
}
