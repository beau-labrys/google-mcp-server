package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"go.ngs.io/google-mcp-server/server"
)

// Handler implements the ServiceHandler interface for Tasks
type Handler struct {
	client *Client
}

// NewHandler creates a new Tasks handler
func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

// GetTools returns the available Tasks tools
func (h *Handler) GetTools() []server.Tool {
	return []server.Tool{
		// Task List Tools
		{
			Name:        "tasks_list_tasklists",
			Description: "List all task lists for the authenticated user",
			InputSchema: server.InputSchema{
				Type:       "object",
				Properties: map[string]server.Property{},
			},
		},
		{
			Name:        "tasks_get_tasklist",
			Description: "Get details of a specific task list",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list",
					},
				},
				Required: []string{"tasklist_id"},
			},
		},
		{
			Name:        "tasks_create_tasklist",
			Description: "Create a new task list",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"title": {
						Type:        "string",
						Description: "Title of the new task list",
					},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "tasks_update_tasklist",
			Description: "Update an existing task list",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list to update",
					},
					"title": {
						Type:        "string",
						Description: "New title for the task list",
					},
				},
				Required: []string{"tasklist_id", "title"},
			},
		},
		{
			Name:        "tasks_delete_tasklist",
			Description: "Delete a task list",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list to delete",
					},
				},
				Required: []string{"tasklist_id"},
			},
		},
		// Task Tools
		{
			Name:        "tasks_list_tasks",
			Description: "List tasks in a task list",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list (use 'default' for the primary task list)",
					},
					"show_completed": {
						Type:        "boolean",
						Description: "Whether to show completed tasks (default: false)",
					},
					"show_hidden": {
						Type:        "boolean",
						Description: "Whether to show hidden tasks (default: false)",
					},
					"max_results": {
						Type:        "number",
						Description: "Maximum number of tasks to return",
					},
					"due_min": {
						Type:        "string",
						Description: "Lower bound for task due date (RFC3339 format)",
					},
					"due_max": {
						Type:        "string",
						Description: "Upper bound for task due date (RFC3339 format)",
					},
				},
				Required: []string{"tasklist_id"},
			},
		},
		{
			Name:        "tasks_get_task",
			Description: "Get details of a specific task",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list",
					},
					"task_id": {
						Type:        "string",
						Description: "The ID of the task",
					},
				},
				Required: []string{"tasklist_id", "task_id"},
			},
		},
		{
			Name:        "tasks_create_task",
			Description: "Create a new task in a task list",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list (use 'default' for the primary task list)",
					},
					"title": {
						Type:        "string",
						Description: "Title of the task",
					},
					"notes": {
						Type:        "string",
						Description: "Additional notes or description for the task",
					},
					"due": {
						Type:        "string",
						Description: "Due date (RFC3339 format, e.g., 2025-02-06T00:00:00Z or just 2025-02-06)",
					},
					"parent": {
						Type:        "string",
						Description: "Parent task ID to create this as a subtask",
					},
				},
				Required: []string{"tasklist_id", "title"},
			},
		},
		{
			Name:        "tasks_update_task",
			Description: "Update an existing task",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list",
					},
					"task_id": {
						Type:        "string",
						Description: "The ID of the task to update",
					},
					"title": {
						Type:        "string",
						Description: "New title for the task",
					},
					"notes": {
						Type:        "string",
						Description: "New notes for the task",
					},
					"due": {
						Type:        "string",
						Description: "New due date (RFC3339 format)",
					},
					"status": {
						Type:        "string",
						Description: "Task status: 'needsAction' or 'completed'",
						Enum:        []string{"needsAction", "completed"},
					},
				},
				Required: []string{"tasklist_id", "task_id"},
			},
		},
		{
			Name:        "tasks_delete_task",
			Description: "Delete a task",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list",
					},
					"task_id": {
						Type:        "string",
						Description: "The ID of the task to delete",
					},
				},
				Required: []string{"tasklist_id", "task_id"},
			},
		},
		{
			Name:        "tasks_complete_task",
			Description: "Mark a task as completed",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list",
					},
					"task_id": {
						Type:        "string",
						Description: "The ID of the task to complete",
					},
				},
				Required: []string{"tasklist_id", "task_id"},
			},
		},
		{
			Name:        "tasks_move_task",
			Description: "Move a task to a new position (reorder or change parent)",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list",
					},
					"task_id": {
						Type:        "string",
						Description: "The ID of the task to move",
					},
					"parent": {
						Type:        "string",
						Description: "New parent task ID (empty string to make it a top-level task)",
					},
					"previous": {
						Type:        "string",
						Description: "ID of the task to position after (empty for first position)",
					},
				},
				Required: []string{"tasklist_id", "task_id"},
			},
		},
		{
			Name:        "tasks_clear_completed",
			Description: "Remove all completed tasks from a task list",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"tasklist_id": {
						Type:        "string",
						Description: "The ID of the task list to clear completed tasks from",
					},
				},
				Required: []string{"tasklist_id"},
			},
		},
	}
}

// GetResources returns available resources (none for Tasks)
func (h *Handler) GetResources() []server.Resource {
	return []server.Resource{}
}

// HandleResourceCall handles resource calls (not implemented for Tasks)
func (h *Handler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	return nil, fmt.Errorf("resources not supported for tasks service")
}

// HandleToolCall handles a tool call for Tasks service
func (h *Handler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	switch name {
	// Task List operations
	case "tasks_list_tasklists":
		return h.handleListTaskLists(ctx)

	case "tasks_get_tasklist":
		var args struct {
			TaskListID string `json:"tasklist_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleGetTaskList(ctx, args.TaskListID)

	case "tasks_create_tasklist":
		var args struct {
			Title string `json:"title"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleCreateTaskList(ctx, args.Title)

	case "tasks_update_tasklist":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			Title      string `json:"title"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleUpdateTaskList(ctx, args.TaskListID, args.Title)

	case "tasks_delete_tasklist":
		var args struct {
			TaskListID string `json:"tasklist_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleDeleteTaskList(ctx, args.TaskListID)

	// Task operations
	case "tasks_list_tasks":
		var args struct {
			TaskListID    string `json:"tasklist_id"`
			ShowCompleted bool   `json:"show_completed"`
			ShowHidden    bool   `json:"show_hidden"`
			MaxResults    int64  `json:"max_results"`
			DueMin        string `json:"due_min"`
			DueMax        string `json:"due_max"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleListTasks(ctx, args.TaskListID, args.ShowCompleted, args.ShowHidden, args.MaxResults, args.DueMin, args.DueMax)

	case "tasks_get_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			TaskID     string `json:"task_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleGetTask(ctx, args.TaskListID, args.TaskID)

	case "tasks_create_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			Title      string `json:"title"`
			Notes      string `json:"notes"`
			Due        string `json:"due"`
			Parent     string `json:"parent"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleCreateTask(ctx, args.TaskListID, args.Title, args.Notes, args.Due, args.Parent)

	case "tasks_update_task":
		var args struct {
			TaskListID string  `json:"tasklist_id"`
			TaskID     string  `json:"task_id"`
			Title      *string `json:"title,omitempty"`
			Notes      *string `json:"notes,omitempty"`
			Due        *string `json:"due,omitempty"`
			Status     *string `json:"status,omitempty"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleUpdateTask(ctx, args.TaskListID, args.TaskID, args.Title, args.Notes, args.Due, args.Status)

	case "tasks_delete_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			TaskID     string `json:"task_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleDeleteTask(ctx, args.TaskListID, args.TaskID)

	case "tasks_complete_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			TaskID     string `json:"task_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleCompleteTask(ctx, args.TaskListID, args.TaskID)

	case "tasks_move_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			TaskID     string `json:"task_id"`
			Parent     string `json:"parent"`
			Previous   string `json:"previous"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleMoveTask(ctx, args.TaskListID, args.TaskID, args.Parent, args.Previous)

	case "tasks_clear_completed":
		var args struct {
			TaskListID string `json:"tasklist_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleClearCompleted(ctx, args.TaskListID)

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// --- Handler implementations ---

func (h *Handler) handleListTaskLists(ctx context.Context) (interface{}, error) {
	taskLists, err := h.client.ListTaskLists()
	if err != nil {
		return nil, err
	}

	// Format response
	result := make([]map[string]interface{}, len(taskLists))
	for i, tl := range taskLists {
		result[i] = map[string]interface{}{
			"id":      tl.Id,
			"title":   tl.Title,
			"updated": tl.Updated,
		}
	}

	return map[string]interface{}{
		"tasklists": result,
		"count":     len(result),
	}, nil
}

func (h *Handler) handleGetTaskList(ctx context.Context, taskListID string) (interface{}, error) {
	taskList, err := h.client.GetTaskList(taskListID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      taskList.Id,
		"title":   taskList.Title,
		"updated": taskList.Updated,
	}, nil
}

func (h *Handler) handleCreateTaskList(ctx context.Context, title string) (interface{}, error) {
	taskList, err := h.client.CreateTaskList(title)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      taskList.Id,
		"title":   taskList.Title,
		"message": fmt.Sprintf("Task list '%s' created successfully", title),
	}, nil
}

func (h *Handler) handleUpdateTaskList(ctx context.Context, taskListID, title string) (interface{}, error) {
	taskList, err := h.client.UpdateTaskList(taskListID, title)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      taskList.Id,
		"title":   taskList.Title,
		"message": "Task list updated successfully",
	}, nil
}

func (h *Handler) handleDeleteTaskList(ctx context.Context, taskListID string) (interface{}, error) {
	if err := h.client.DeleteTaskList(taskListID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":  "deleted",
		"message": "Task list deleted successfully",
	}, nil
}

func (h *Handler) resolveTaskListID(taskListID string) (string, error) {
	if taskListID == "default" || taskListID == "" {
		defaultList, err := h.client.GetDefaultTaskList()
		if err != nil {
			return "", err
		}
		return defaultList.Id, nil
	}
	return taskListID, nil
}

func (h *Handler) handleListTasks(ctx context.Context, taskListID string, showCompleted, showHidden bool, maxResults int64, dueMin, dueMax string) (interface{}, error) {
	resolvedID, err := h.resolveTaskListID(taskListID)
	if err != nil {
		return nil, err
	}

	opts := &ListTasksOptions{
		ShowCompleted: showCompleted,
		ShowHidden:    showHidden,
		MaxResults:    maxResults,
		DueMin:        dueMin,
		DueMax:        dueMax,
	}

	tasks, err := h.client.ListTasks(resolvedID, opts)
	if err != nil {
		return nil, err
	}

	// Format response
	result := make([]map[string]interface{}, len(tasks))
	for i, t := range tasks {
		result[i] = formatTask(t)
	}

	return map[string]interface{}{
		"tasks":       result,
		"count":       len(result),
		"tasklist_id": resolvedID,
	}, nil
}

func (h *Handler) handleGetTask(ctx context.Context, taskListID, taskID string) (interface{}, error) {
	resolvedID, err := h.resolveTaskListID(taskListID)
	if err != nil {
		return nil, err
	}

	task, err := h.client.GetTask(resolvedID, taskID)
	if err != nil {
		return nil, err
	}

	return formatTask(task), nil
}

func (h *Handler) handleCreateTask(ctx context.Context, taskListID, title, notes, due, parent string) (interface{}, error) {
	resolvedID, err := h.resolveTaskListID(taskListID)
	if err != nil {
		return nil, err
	}

	opts := &CreateTaskOptions{
		Title:  title,
		Notes:  notes,
		Due:    due,
		Parent: parent,
	}

	task, err := h.client.CreateTask(resolvedID, opts)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["message"] = fmt.Sprintf("Task '%s' created successfully", title)
	return result, nil
}

func (h *Handler) handleUpdateTask(ctx context.Context, taskListID, taskID string, title, notes, due, status *string) (interface{}, error) {
	resolvedID, err := h.resolveTaskListID(taskListID)
	if err != nil {
		return nil, err
	}

	opts := &UpdateTaskOptions{
		Title:  title,
		Notes:  notes,
		Due:    due,
		Status: status,
	}

	task, err := h.client.UpdateTask(resolvedID, taskID, opts)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["message"] = "Task updated successfully"
	return result, nil
}

func (h *Handler) handleDeleteTask(ctx context.Context, taskListID, taskID string) (interface{}, error) {
	resolvedID, err := h.resolveTaskListID(taskListID)
	if err != nil {
		return nil, err
	}

	if err := h.client.DeleteTask(resolvedID, taskID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":  "deleted",
		"task_id": taskID,
		"message": "Task deleted successfully",
	}, nil
}

func (h *Handler) handleCompleteTask(ctx context.Context, taskListID, taskID string) (interface{}, error) {
	resolvedID, err := h.resolveTaskListID(taskListID)
	if err != nil {
		return nil, err
	}

	task, err := h.client.CompleteTask(resolvedID, taskID)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["message"] = "Task marked as completed"
	return result, nil
}

func (h *Handler) handleMoveTask(ctx context.Context, taskListID, taskID, parent, previous string) (interface{}, error) {
	resolvedID, err := h.resolveTaskListID(taskListID)
	if err != nil {
		return nil, err
	}

	task, err := h.client.MoveTask(resolvedID, taskID, parent, previous)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["message"] = "Task moved successfully"
	return result, nil
}

func (h *Handler) handleClearCompleted(ctx context.Context, taskListID string) (interface{}, error) {
	resolvedID, err := h.resolveTaskListID(taskListID)
	if err != nil {
		return nil, err
	}

	if err := h.client.ClearCompleted(resolvedID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":      "cleared",
		"tasklist_id": resolvedID,
		"message":     "All completed tasks cleared successfully",
	}, nil
}

// formatTask formats a task for JSON response
func formatTask(t interface{}) map[string]interface{} {
	// Use JSON marshal/unmarshal for generic formatting
	data := make(map[string]interface{})
	jsonData, _ := json.Marshal(t)
	_ = json.Unmarshal(jsonData, &data)

	// Clean up the response by removing empty fields
	result := make(map[string]interface{})
	for k, v := range data {
		if v != nil && v != "" {
			result[k] = v
		}
	}

	return result
}
