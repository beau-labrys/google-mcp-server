package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"go.ngs.io/google-mcp-server/auth"
	"go.ngs.io/google-mcp-server/server"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// MultiAccountHandler implements the ServiceHandler interface with multi-account support
type MultiAccountHandler struct {
	accountManager *auth.AccountManager
	defaultClient  *Client // For backward compatibility
}

// NewMultiAccountHandler creates a new multi-account aware Tasks handler
func NewMultiAccountHandler(accountManager *auth.AccountManager, defaultClient *Client) *MultiAccountHandler {
	return &MultiAccountHandler{
		accountManager: accountManager,
		defaultClient:  defaultClient,
	}
}

// GetTools returns the available Tasks tools with multi-account support
func (h *MultiAccountHandler) GetTools() []server.Tool {
	return []server.Tool{
		// Task List Tools
		{
			Name:        "tasks_list_tasklists",
			Description: "List all task lists for the authenticated user",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
					},
				},
			},
		},
		{
			Name:        "tasks_list_tasklists_all_accounts",
			Description: "List all task lists from all authenticated accounts",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
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
					"account": {
						Type:        "string",
						Description: "Email address of the account to use (optional)",
					},
				},
				Required: []string{"tasklist_id"},
			},
		},
	}
}

// GetResources returns available resources (none for Tasks)
func (h *MultiAccountHandler) GetResources() []server.Resource {
	return []server.Resource{}
}

// HandleResourceCall handles resource calls (not implemented for Tasks)
func (h *MultiAccountHandler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	return nil, fmt.Errorf("resources not supported for tasks service")
}

// getClientForAccount gets or creates a tasks client for the specified account
func (h *MultiAccountHandler) getClientForAccount(ctx context.Context, email string) (*Client, error) {
	// If no email specified, use default client
	if email == "" && h.defaultClient != nil {
		return h.defaultClient, nil
	}

	// Get account from manager
	account, err := h.accountManager.GetAccount(email)
	if err != nil {
		// If specific account not found, try to find one
		if email == "" {
			accounts := h.accountManager.ListAccounts()
			if len(accounts) > 0 {
				account = accounts[0]
			} else {
				return nil, fmt.Errorf("no authenticated accounts available")
			}
		} else {
			return nil, err
		}
	}

	// Create tasks service for this account
	service, err := tasks.NewService(ctx, option.WithHTTPClient(account.OAuthClient.GetHTTPClient()))
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks service: %w", err)
	}

	return &Client{service: service}, nil
}

// HandleToolCall handles a tool call
func (h *MultiAccountHandler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	switch name {
	// Multi-account specific tools
	case "tasks_list_tasklists_all_accounts":
		return h.handleListTaskListsAllAccounts(ctx)

	// Task List operations
	case "tasks_list_tasklists":
		var args struct {
			Account string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleListTaskLists(ctx, args.Account)

	case "tasks_get_tasklist":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleGetTaskList(ctx, args.TaskListID, args.Account)

	case "tasks_create_tasklist":
		var args struct {
			Title   string `json:"title"`
			Account string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleCreateTaskList(ctx, args.Title, args.Account)

	case "tasks_update_tasklist":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			Title      string `json:"title"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleUpdateTaskList(ctx, args.TaskListID, args.Title, args.Account)

	case "tasks_delete_tasklist":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleDeleteTaskList(ctx, args.TaskListID, args.Account)

	// Task operations
	case "tasks_list_tasks":
		var args struct {
			TaskListID    string `json:"tasklist_id"`
			ShowCompleted bool   `json:"show_completed"`
			ShowHidden    bool   `json:"show_hidden"`
			MaxResults    int64  `json:"max_results"`
			DueMin        string `json:"due_min"`
			DueMax        string `json:"due_max"`
			Account       string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleListTasks(ctx, args.TaskListID, args.ShowCompleted, args.ShowHidden, args.MaxResults, args.DueMin, args.DueMax, args.Account)

	case "tasks_get_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			TaskID     string `json:"task_id"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleGetTask(ctx, args.TaskListID, args.TaskID, args.Account)

	case "tasks_create_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			Title      string `json:"title"`
			Notes      string `json:"notes"`
			Due        string `json:"due"`
			Parent     string `json:"parent"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleCreateTask(ctx, args.TaskListID, args.Title, args.Notes, args.Due, args.Parent, args.Account)

	case "tasks_update_task":
		var args struct {
			TaskListID string  `json:"tasklist_id"`
			TaskID     string  `json:"task_id"`
			Title      *string `json:"title,omitempty"`
			Notes      *string `json:"notes,omitempty"`
			Due        *string `json:"due,omitempty"`
			Status     *string `json:"status,omitempty"`
			Account    string  `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleUpdateTask(ctx, args.TaskListID, args.TaskID, args.Title, args.Notes, args.Due, args.Status, args.Account)

	case "tasks_delete_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			TaskID     string `json:"task_id"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleDeleteTask(ctx, args.TaskListID, args.TaskID, args.Account)

	case "tasks_complete_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			TaskID     string `json:"task_id"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleCompleteTask(ctx, args.TaskListID, args.TaskID, args.Account)

	case "tasks_move_task":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			TaskID     string `json:"task_id"`
			Parent     string `json:"parent"`
			Previous   string `json:"previous"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleMoveTask(ctx, args.TaskListID, args.TaskID, args.Parent, args.Previous, args.Account)

	case "tasks_clear_completed":
		var args struct {
			TaskListID string `json:"tasklist_id"`
			Account    string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return h.handleClearCompleted(ctx, args.TaskListID, args.Account)

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// --- Handler implementations ---

func (h *MultiAccountHandler) handleListTaskLists(ctx context.Context, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	taskLists, err := client.ListTaskLists()
	if err != nil {
		return nil, err
	}

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
		"account":   account,
	}, nil
}

func (h *MultiAccountHandler) handleListTaskListsAllAccounts(ctx context.Context) (interface{}, error) {
	accounts := h.accountManager.ListAccounts()
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no authenticated accounts available")
	}

	allTaskLists := make(map[string]interface{})
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, account := range accounts {
		wg.Add(1)
		go func(acc *auth.Account) {
			defer wg.Done()

			client, err := h.getClientForAccount(ctx, acc.Email)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get client for %s: %v\n", acc.Email, err)
				return
			}

			taskLists, err := client.ListTaskLists()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to list task lists for %s: %v\n", acc.Email, err)
				return
			}

			result := make([]map[string]interface{}, len(taskLists))
			for i, tl := range taskLists {
				result[i] = map[string]interface{}{
					"id":      tl.Id,
					"title":   tl.Title,
					"updated": tl.Updated,
				}
			}

			mu.Lock()
			allTaskLists[acc.Email] = map[string]interface{}{
				"account_name": acc.Name,
				"tasklists":    result,
				"count":        len(result),
			}
			mu.Unlock()
		}(account)
	}

	wg.Wait()

	return map[string]interface{}{
		"accounts":       allTaskLists,
		"total_accounts": len(accounts),
	}, nil
}

func (h *MultiAccountHandler) handleGetTaskList(ctx context.Context, taskListID, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	taskList, err := client.GetTaskList(taskListID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      taskList.Id,
		"title":   taskList.Title,
		"updated": taskList.Updated,
		"account": account,
	}, nil
}

func (h *MultiAccountHandler) handleCreateTaskList(ctx context.Context, title, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	taskList, err := client.CreateTaskList(title)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      taskList.Id,
		"title":   taskList.Title,
		"account": account,
		"message": fmt.Sprintf("Task list '%s' created successfully", title),
	}, nil
}

func (h *MultiAccountHandler) handleUpdateTaskList(ctx context.Context, taskListID, title, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	taskList, err := client.UpdateTaskList(taskListID, title)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      taskList.Id,
		"title":   taskList.Title,
		"account": account,
		"message": "Task list updated successfully",
	}, nil
}

func (h *MultiAccountHandler) handleDeleteTaskList(ctx context.Context, taskListID, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	if err := client.DeleteTaskList(taskListID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":  "deleted",
		"account": account,
		"message": "Task list deleted successfully",
	}, nil
}

func (h *MultiAccountHandler) resolveTaskListID(client *Client, taskListID string) (string, error) {
	if taskListID == "default" || taskListID == "" {
		defaultList, err := client.GetDefaultTaskList()
		if err != nil {
			return "", err
		}
		return defaultList.Id, nil
	}
	return taskListID, nil
}

func (h *MultiAccountHandler) handleListTasks(ctx context.Context, taskListID string, showCompleted, showHidden bool, maxResults int64, dueMin, dueMax, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	resolvedID, err := h.resolveTaskListID(client, taskListID)
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

	tasks, err := client.ListTasks(resolvedID, opts)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(tasks))
	for i, t := range tasks {
		result[i] = formatTask(t)
	}

	return map[string]interface{}{
		"tasks":       result,
		"count":       len(result),
		"tasklist_id": resolvedID,
		"account":     account,
	}, nil
}

func (h *MultiAccountHandler) handleGetTask(ctx context.Context, taskListID, taskID, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	resolvedID, err := h.resolveTaskListID(client, taskListID)
	if err != nil {
		return nil, err
	}

	task, err := client.GetTask(resolvedID, taskID)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["account"] = account
	return result, nil
}

func (h *MultiAccountHandler) handleCreateTask(ctx context.Context, taskListID, title, notes, due, parent, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	resolvedID, err := h.resolveTaskListID(client, taskListID)
	if err != nil {
		return nil, err
	}

	opts := &CreateTaskOptions{
		Title:  title,
		Notes:  notes,
		Due:    due,
		Parent: parent,
	}

	task, err := client.CreateTask(resolvedID, opts)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["account"] = account
	result["message"] = fmt.Sprintf("Task '%s' created successfully", title)
	return result, nil
}

func (h *MultiAccountHandler) handleUpdateTask(ctx context.Context, taskListID, taskID string, title, notes, due, status *string, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	resolvedID, err := h.resolveTaskListID(client, taskListID)
	if err != nil {
		return nil, err
	}

	opts := &UpdateTaskOptions{
		Title:  title,
		Notes:  notes,
		Due:    due,
		Status: status,
	}

	task, err := client.UpdateTask(resolvedID, taskID, opts)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["account"] = account
	result["message"] = "Task updated successfully"
	return result, nil
}

func (h *MultiAccountHandler) handleDeleteTask(ctx context.Context, taskListID, taskID, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	resolvedID, err := h.resolveTaskListID(client, taskListID)
	if err != nil {
		return nil, err
	}

	if err := client.DeleteTask(resolvedID, taskID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":  "deleted",
		"task_id": taskID,
		"account": account,
		"message": "Task deleted successfully",
	}, nil
}

func (h *MultiAccountHandler) handleCompleteTask(ctx context.Context, taskListID, taskID, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	resolvedID, err := h.resolveTaskListID(client, taskListID)
	if err != nil {
		return nil, err
	}

	task, err := client.CompleteTask(resolvedID, taskID)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["account"] = account
	result["message"] = "Task marked as completed"
	return result, nil
}

func (h *MultiAccountHandler) handleMoveTask(ctx context.Context, taskListID, taskID, parent, previous, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	resolvedID, err := h.resolveTaskListID(client, taskListID)
	if err != nil {
		return nil, err
	}

	task, err := client.MoveTask(resolvedID, taskID, parent, previous)
	if err != nil {
		return nil, err
	}

	result := formatTask(task)
	result["account"] = account
	result["message"] = "Task moved successfully"
	return result, nil
}

func (h *MultiAccountHandler) handleClearCompleted(ctx context.Context, taskListID, account string) (interface{}, error) {
	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	resolvedID, err := h.resolveTaskListID(client, taskListID)
	if err != nil {
		return nil, err
	}

	if err := client.ClearCompleted(resolvedID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":      "cleared",
		"tasklist_id": resolvedID,
		"account":     account,
		"message":     "All completed tasks cleared successfully",
	}, nil
}
