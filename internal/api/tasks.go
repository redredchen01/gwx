package api

import (
	"context"
	"fmt"
	"time"

	tasks "google.golang.org/api/tasks/v1"
)

// TasksService wraps Google Tasks API operations.
type TasksService struct {
	client *Client
}

// NewTasksService creates a Tasks service wrapper.
func NewTasksService(client *Client) *TasksService {
	return &TasksService{client: client}
}

func (ts *TasksService) service(ctx context.Context) (*tasks.Service, error) {
	svc, err := ts.client.GetOrCreateService("tasks:v1", func() (any, error) {
		opts, err := ts.client.ClientOptions(ctx, "tasks")
		if err != nil {
			return nil, err
		}
		return tasks.NewService(ctx, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("create tasks service: %w", err)
	}
	return svc.(*tasks.Service), nil
}

// TaskItem is a simplified task representation.
type TaskItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Notes     string `json:"notes,omitempty"`
	Status    string `json:"status"` // "needsAction" or "completed"
	Due       string `json:"due,omitempty"`
	Completed string `json:"completed,omitempty"`
	Updated   string `json:"updated"`
	Position  string `json:"position,omitempty"`
	Parent    string `json:"parent,omitempty"`
}

// TaskListInfo holds task list metadata.
type TaskListInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ListTaskLists returns all task lists.
func (ts *TasksService) ListTaskLists(ctx context.Context) ([]TaskListInfo, error) {
	if err := ts.client.WaitRate(ctx, "tasks"); err != nil {
		return nil, err
	}

	svc, err := ts.service(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := svc.Tasklists.List().Do()
	if err != nil {
		return nil, fmt.Errorf("list task lists: %w", err)
	}

	var result []TaskListInfo
	for _, tl := range resp.Items {
		result = append(result, TaskListInfo{ID: tl.Id, Title: tl.Title})
	}
	return result, nil
}

// ListTasks lists tasks in a task list.
func (ts *TasksService) ListTasks(ctx context.Context, taskListID string, showCompleted bool) ([]TaskItem, error) {
	if !ts.client.NoCache {
		key := cacheKey("tasks", "ListTasks", taskListID, showCompleted)
		if cached, ok := ts.client.cache.Get(key); ok {
			return cached.([]TaskItem), nil
		}
	}

	if err := ts.client.WaitRate(ctx, "tasks"); err != nil {
		return nil, err
	}

	svc, err := ts.service(ctx)
	if err != nil {
		return nil, err
	}

	if taskListID == "" {
		taskListID = "@default"
	}

	call := svc.Tasks.List(taskListID).ShowCompleted(showCompleted)

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	var items []TaskItem
	for _, t := range resp.Items {
		items = append(items, taskToItem(t))
	}
	if !ts.client.NoCache {
		key := cacheKey("tasks", "ListTasks", taskListID, showCompleted)
		ts.client.cache.Set(key, items, 5*time.Minute)
	}
	return items, nil
}

// CreateTask creates a new task.
func (ts *TasksService) CreateTask(ctx context.Context, taskListID string, title, notes, due string) (*TaskItem, error) {
	if err := ts.client.WaitRate(ctx, "tasks"); err != nil {
		return nil, err
	}

	svc, err := ts.service(ctx)
	if err != nil {
		return nil, err
	}

	if taskListID == "" {
		taskListID = "@default"
	}

	task := &tasks.Task{
		Title: title,
		Notes: notes,
	}
	if due != "" {
		task.Due = due + "T00:00:00.000Z" // Tasks API expects RFC3339
	}

	created, err := svc.Tasks.Insert(taskListID, task).Do()
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	item := taskToItem(created)
	return &item, nil
}

// CompleteTask marks a task as completed.
func (ts *TasksService) CompleteTask(ctx context.Context, taskListID, taskID string) (*TaskItem, error) {
	if err := ts.client.WaitRate(ctx, "tasks"); err != nil {
		return nil, err
	}

	svc, err := ts.service(ctx)
	if err != nil {
		return nil, err
	}

	if taskListID == "" {
		taskListID = "@default"
	}

	task, err := svc.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	task.Status = "completed"
	updated, err := svc.Tasks.Update(taskListID, taskID, task).Do()
	if err != nil {
		return nil, fmt.Errorf("complete task: %w", err)
	}

	item := taskToItem(updated)
	return &item, nil
}

// DeleteTask deletes a task.
func (ts *TasksService) DeleteTask(ctx context.Context, taskListID, taskID string) error {
	if err := ts.client.WaitRate(ctx, "tasks"); err != nil {
		return err
	}

	svc, err := ts.service(ctx)
	if err != nil {
		return err
	}

	if taskListID == "" {
		taskListID = "@default"
	}

	if err := svc.Tasks.Delete(taskListID, taskID).Do(); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

func taskToItem(t *tasks.Task) TaskItem {
	item := TaskItem{
		ID:       t.Id,
		Title:    t.Title,
		Notes:    t.Notes,
		Status:   t.Status,
		Due:      t.Due,
		Updated:  t.Updated,
		Position: t.Position,
		Parent:   t.Parent,
	}
	if t.Completed != nil {
		item.Completed = *t.Completed
	}
	return item
}
