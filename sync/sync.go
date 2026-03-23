package sync

import (
	"fmt"
	"redmine-tui/config"
	"redmine-tui/redmine"
	"redmine-tui/vikunja"
)

func SyncIssuesToVikunja(redmineClient *redmine.Client, vikunjaClient *vikunja.Client, cfg *config.Config) error {
	issues, err := redmineClient.GetAssignedIssues()
	if err != nil {
		return fmt.Errorf("failed to fetch redmine issues: %w", err)
	}

	viewID := cfg.Vikunja.ViewID
	if viewID == 0 {
		view, err := vikunjaClient.FindKanbanView(cfg.Vikunja.ProjectID)
		if err != nil {
			return fmt.Errorf("failed to find kanban view: %w", err)
		}
		viewID = view.ID
	}

	bucketTasks, err := vikunjaClient.GetBucketTasks(cfg.Vikunja.ProjectID, viewID, cfg.Vikunja.BucketID)
	if err != nil {
		return fmt.Errorf("failed to fetch vikunja tasks: %w", err)
	}

	tasksByTitle := make(map[string][]vikunja.Task)
	for _, task := range bucketTasks {
		tasksByTitle[task.Title] = append(tasksByTitle[task.Title], task)
	}

	vikunjaMap := make(map[string]vikunja.Task)
	for title, group := range tasksByTitle {
		vikunjaMap[title] = group[0]
		for _, t := range group[1:] {
			_ = vikunjaClient.DeleteTask(t.ID)
		}
	}

	redmineMap := make(map[string]redmine.Issue)
	for _, issue := range issues {
		redmineMap[issue.Subject] = issue
	}

	for subject := range redmineMap {
		if _, exists := vikunjaMap[subject]; !exists {
			if err := vikunjaClient.CreateTask(cfg.Vikunja.ProjectID, viewID, cfg.Vikunja.BucketID, subject); err != nil {
				return fmt.Errorf("failed to create task '%s': %w", subject, err)
			}
		}
	}

	for title, task := range vikunjaMap {
		if _, exists := redmineMap[title]; !exists {
			if cfg.Vikunja.DoneBucketID != 0 {
				if err := vikunjaClient.MoveTask(cfg.Vikunja.ProjectID, viewID, task.ID, cfg.Vikunja.DoneBucketID); err != nil {
					return fmt.Errorf("failed to move task '%s' to done bucket: %w", title, err)
				}
			} else {
				if err := vikunjaClient.DeleteTask(task.ID); err != nil {
					return fmt.Errorf("failed to delete task '%s': %w", title, err)
				}
			}
		}
	}

	return nil
}
