package tasks

import (
	"context"
	"fmt"

	"go.ngs.io/google-mcp-server/auth"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// Client wraps the Google Tasks API client
type Client struct {
	service *tasks.Service
}

// NewClient creates a new Tasks client
func NewClient(ctx context.Context, oauth *auth.OAuthClient) (*Client, error) {
	service, err := tasks.NewService(ctx, oauth.GetClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// NewClientWithHTTPClient creates a new Tasks client with an HTTP client
func NewClientWithHTTPClient(ctx context.Context, httpClient option.ClientOption) (*Client, error) {
	service, err := tasks.NewService(ctx, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// --- Task List Operations ---

// ListTaskLists lists all task lists
func (c *Client) ListTaskLists() ([]*tasks.TaskList, error) {
	var taskLists []*tasks.TaskList

	call := c.service.Tasklists.List()
	err := call.Pages(context.Background(), func(page *tasks.TaskLists) error {
		taskLists = append(taskLists, page.Items...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list task lists: %w", err)
	}

	return taskLists, nil
}

// GetTaskList gets a specific task list by ID
func (c *Client) GetTaskList(taskListID string) (*tasks.TaskList, error) {
	taskList, err := c.service.Tasklists.Get(taskListID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get task list: %w", err)
	}
	return taskList, nil
}

// CreateTaskList creates a new task list
func (c *Client) CreateTaskList(title string) (*tasks.TaskList, error) {
	taskList := &tasks.TaskList{
		Title: title,
	}

	created, err := c.service.Tasklists.Insert(taskList).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create task list: %w", err)
	}
	return created, nil
}

// UpdateTaskList updates an existing task list
func (c *Client) UpdateTaskList(taskListID, title string) (*tasks.TaskList, error) {
	taskList := &tasks.TaskList{
		Title: title,
	}

	updated, err := c.service.Tasklists.Update(taskListID, taskList).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update task list: %w", err)
	}
	return updated, nil
}

// DeleteTaskList deletes a task list
func (c *Client) DeleteTaskList(taskListID string) error {
	err := c.service.Tasklists.Delete(taskListID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete task list: %w", err)
	}
	return nil
}

// --- Task Operations ---

// ListTasksOptions contains options for listing tasks
type ListTasksOptions struct {
	ShowCompleted bool
	ShowHidden    bool
	ShowDeleted   bool
	MaxResults    int64
	DueMin        string // RFC3339 timestamp
	DueMax        string // RFC3339 timestamp
}

// ListTasks lists tasks in a task list with options
func (c *Client) ListTasks(taskListID string, opts *ListTasksOptions) ([]*tasks.Task, error) {
	call := c.service.Tasks.List(taskListID)

	if opts != nil {
		call = call.ShowCompleted(opts.ShowCompleted)
		call = call.ShowHidden(opts.ShowHidden)
		call = call.ShowDeleted(opts.ShowDeleted)

		if opts.MaxResults > 0 {
			call = call.MaxResults(opts.MaxResults)
		}
		if opts.DueMin != "" {
			call = call.DueMin(opts.DueMin)
		}
		if opts.DueMax != "" {
			call = call.DueMax(opts.DueMax)
		}
	}

	var allTasks []*tasks.Task
	err := call.Pages(context.Background(), func(page *tasks.Tasks) error {
		allTasks = append(allTasks, page.Items...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return allTasks, nil
}

// GetTask gets a specific task
func (c *Client) GetTask(taskListID, taskID string) (*tasks.Task, error) {
	task, err := c.service.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

// CreateTaskOptions contains options for creating a task
type CreateTaskOptions struct {
	Title          string
	Notes          string
	Due            string // RFC3339 date/time or date string (YYYY-MM-DD)
	Parent         string // Parent task ID for subtasks
	Status         string // "needsAction" or "completed"
	PreviousTaskID string // Task ID after which to insert
}

// CreateTask creates a new task
func (c *Client) CreateTask(taskListID string, opts *CreateTaskOptions) (*tasks.Task, error) {
	task := &tasks.Task{
		Title: opts.Title,
	}

	if opts.Notes != "" {
		task.Notes = opts.Notes
	}
	if opts.Due != "" {
		task.Due = opts.Due
	}
	if opts.Status != "" {
		task.Status = opts.Status
	} else {
		task.Status = "needsAction"
	}

	call := c.service.Tasks.Insert(taskListID, task)

	if opts.Parent != "" {
		call = call.Parent(opts.Parent)
	}
	if opts.PreviousTaskID != "" {
		call = call.Previous(opts.PreviousTaskID)
	}

	created, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	return created, nil
}

// UpdateTaskOptions contains options for updating a task
type UpdateTaskOptions struct {
	Title  *string // nil means don't update
	Notes  *string
	Due    *string
	Status *string
}

// UpdateTask updates an existing task
func (c *Client) UpdateTask(taskListID, taskID string, opts *UpdateTaskOptions) (*tasks.Task, error) {
	// First, get the current task
	task, err := c.GetTask(taskListID, taskID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if opts.Title != nil {
		task.Title = *opts.Title
	}
	if opts.Notes != nil {
		task.Notes = *opts.Notes
	}
	if opts.Due != nil {
		task.Due = *opts.Due
	}
	if opts.Status != nil {
		task.Status = *opts.Status
	}

	updated, err := c.service.Tasks.Update(taskListID, taskID, task).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	return updated, nil
}

// DeleteTask deletes a task
func (c *Client) DeleteTask(taskListID, taskID string) error {
	err := c.service.Tasks.Delete(taskListID, taskID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// CompleteTask marks a task as completed
func (c *Client) CompleteTask(taskListID, taskID string) (*tasks.Task, error) {
	status := "completed"
	return c.UpdateTask(taskListID, taskID, &UpdateTaskOptions{
		Status: &status,
	})
}

// UncompleteTask marks a task as needs action
func (c *Client) UncompleteTask(taskListID, taskID string) (*tasks.Task, error) {
	status := "needsAction"
	return c.UpdateTask(taskListID, taskID, &UpdateTaskOptions{
		Status: &status,
	})
}

// MoveTask moves a task to a new position (optionally under a new parent)
func (c *Client) MoveTask(taskListID, taskID string, parent, previous string) (*tasks.Task, error) {
	call := c.service.Tasks.Move(taskListID, taskID)

	if parent != "" {
		call = call.Parent(parent)
	}
	if previous != "" {
		call = call.Previous(previous)
	}

	moved, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to move task: %w", err)
	}
	return moved, nil
}

// ClearCompleted removes all completed tasks from a task list
func (c *Client) ClearCompleted(taskListID string) error {
	err := c.service.Tasks.Clear(taskListID).Do()
	if err != nil {
		return fmt.Errorf("failed to clear completed tasks: %w", err)
	}
	return nil
}

// GetDefaultTaskList returns the default task list (usually the first one)
func (c *Client) GetDefaultTaskList() (*tasks.TaskList, error) {
	taskLists, err := c.ListTaskLists()
	if err != nil {
		return nil, err
	}

	if len(taskLists) == 0 {
		return nil, fmt.Errorf("no task lists found")
	}

	// The first task list is typically the default one
	return taskLists[0], nil
}
