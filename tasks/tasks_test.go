package tasks

import (
	"context"
	"encoding/json"
	"testing"

	"go.ngs.io/google-mcp-server/server"
)

// TestHandlerGetTools verifies that the handler returns the expected tools
func TestHandlerGetTools(t *testing.T) {
	// Create a handler with nil client (just testing tool definitions)
	handler := &Handler{client: nil}

	tools := handler.GetTools()

	// Verify we have the expected number of tools
	expectedToolCount := 13 // All tools except the multi-account one
	if len(tools) != expectedToolCount {
		t.Errorf("Expected %d tools, got %d", expectedToolCount, len(tools))
	}

	// Verify tool names
	expectedTools := map[string]bool{
		"tasks_list_tasklists":  true,
		"tasks_get_tasklist":    true,
		"tasks_create_tasklist": true,
		"tasks_update_tasklist": true,
		"tasks_delete_tasklist": true,
		"tasks_list_tasks":      true,
		"tasks_get_task":        true,
		"tasks_create_task":     true,
		"tasks_update_task":     true,
		"tasks_delete_task":     true,
		"tasks_complete_task":   true,
		"tasks_move_task":       true,
		"tasks_clear_completed": true,
	}

	for _, tool := range tools {
		if !expectedTools[tool.Name] {
			t.Errorf("Unexpected tool: %s", tool.Name)
		}
		delete(expectedTools, tool.Name)
	}

	if len(expectedTools) > 0 {
		for name := range expectedTools {
			t.Errorf("Missing expected tool: %s", name)
		}
	}
}

// TestMultiAccountHandlerGetTools verifies multi-account handler returns expected tools
func TestMultiAccountHandlerGetTools(t *testing.T) {
	// Create a multi-account handler with nil dependencies (just testing tool definitions)
	handler := &MultiAccountHandler{
		accountManager: nil,
		defaultClient:  nil,
	}

	tools := handler.GetTools()

	// Verify we have the expected number of tools (14 including the _all_accounts tool)
	expectedToolCount := 14
	if len(tools) != expectedToolCount {
		t.Errorf("Expected %d tools, got %d", expectedToolCount, len(tools))
	}

	// Verify the multi-account specific tool exists
	hasAllAccountsTool := false
	for _, tool := range tools {
		if tool.Name == "tasks_list_tasklists_all_accounts" {
			hasAllAccountsTool = true
			break
		}
	}

	if !hasAllAccountsTool {
		t.Error("Missing tasks_list_tasklists_all_accounts tool")
	}
}

// TestToolSchemas verifies that tool schemas are properly defined
func TestToolSchemas(t *testing.T) {
	handler := &Handler{client: nil}
	tools := handler.GetTools()

	toolMap := make(map[string]server.Tool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	// Test tasks_list_tasks has all expected properties
	listTasksTool, ok := toolMap["tasks_list_tasks"]
	if !ok {
		t.Fatal("tasks_list_tasks tool not found")
	}

	expectedProps := []string{"tasklist_id", "show_completed", "show_hidden", "max_results", "due_min", "due_max"}
	for _, prop := range expectedProps {
		if _, exists := listTasksTool.InputSchema.Properties[prop]; !exists {
			t.Errorf("tasks_list_tasks missing property: %s", prop)
		}
	}

	// Verify required fields
	if len(listTasksTool.InputSchema.Required) == 0 {
		t.Error("tasks_list_tasks should have required fields")
	}

	// Test tasks_create_task has all expected properties
	createTaskTool, ok := toolMap["tasks_create_task"]
	if !ok {
		t.Fatal("tasks_create_task tool not found")
	}

	expectedCreateProps := []string{"tasklist_id", "title", "notes", "due", "parent"}
	for _, prop := range expectedCreateProps {
		if _, exists := createTaskTool.InputSchema.Properties[prop]; !exists {
			t.Errorf("tasks_create_task missing property: %s", prop)
		}
	}
}

// TestFormatTask verifies task formatting
func TestFormatTask(t *testing.T) {
	// Create a mock task-like structure
	mockTask := map[string]interface{}{
		"id":     "task123",
		"title":  "Test Task",
		"notes":  "Some notes",
		"status": "needsAction",
		"due":    "2025-02-06T00:00:00Z",
	}

	result := formatTask(mockTask)

	// Verify fields are preserved
	if result["id"] != "task123" {
		t.Errorf("Expected id 'task123', got %v", result["id"])
	}
	if result["title"] != "Test Task" {
		t.Errorf("Expected title 'Test Task', got %v", result["title"])
	}
	if result["status"] != "needsAction" {
		t.Errorf("Expected status 'needsAction', got %v", result["status"])
	}
}

// TestHandleToolCallUnknownTool verifies unknown tool handling
func TestHandleToolCallUnknownTool(t *testing.T) {
	handler := &Handler{client: nil}

	_, err := handler.HandleToolCall(context.Background(), "unknown_tool", json.RawMessage(`{}`))
	if err == nil {
		t.Error("Expected error for unknown tool")
	}

	expectedErr := "unknown tool: unknown_tool"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

// TestMultiAccountHandlerUnknownTool verifies unknown tool handling in multi-account handler
func TestMultiAccountHandlerUnknownTool(t *testing.T) {
	handler := &MultiAccountHandler{
		accountManager: nil,
		defaultClient:  nil,
	}

	_, err := handler.HandleToolCall(context.Background(), "unknown_tool", json.RawMessage(`{}`))
	if err == nil {
		t.Error("Expected error for unknown tool")
	}

	expectedErr := "unknown tool: unknown_tool"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

// TestInvalidArgumentsHandling verifies error handling for invalid arguments
func TestInvalidArgumentsHandling(t *testing.T) {
	handler := &Handler{client: nil}

	// Test with invalid JSON
	_, err := handler.HandleToolCall(context.Background(), "tasks_get_task", json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("Expected error for invalid JSON arguments")
	}
}

// TestListTasksOptionsDefault verifies default options are properly set
func TestListTasksOptionsDefault(t *testing.T) {
	opts := &ListTasksOptions{}

	// Verify defaults
	if opts.ShowCompleted {
		t.Error("ShowCompleted should default to false")
	}
	if opts.ShowHidden {
		t.Error("ShowHidden should default to false")
	}
	if opts.ShowDeleted {
		t.Error("ShowDeleted should default to false")
	}
	if opts.MaxResults != 0 {
		t.Error("MaxResults should default to 0")
	}
}

// TestCreateTaskOptions verifies CreateTaskOptions struct
func TestCreateTaskOptions(t *testing.T) {
	opts := &CreateTaskOptions{
		Title:  "Test Task",
		Notes:  "Test Notes",
		Due:    "2025-02-06T00:00:00Z",
		Parent: "parent123",
		Status: "needsAction",
	}

	if opts.Title != "Test Task" {
		t.Errorf("Expected Title 'Test Task', got '%s'", opts.Title)
	}
	if opts.Notes != "Test Notes" {
		t.Errorf("Expected Notes 'Test Notes', got '%s'", opts.Notes)
	}
	if opts.Parent != "parent123" {
		t.Errorf("Expected Parent 'parent123', got '%s'", opts.Parent)
	}
}

// TestUpdateTaskOptions verifies UpdateTaskOptions with pointers
func TestUpdateTaskOptions(t *testing.T) {
	title := "Updated Title"
	status := "completed"

	opts := &UpdateTaskOptions{
		Title:  &title,
		Notes:  nil, // Not updating notes
		Due:    nil, // Not updating due
		Status: &status,
	}

	if opts.Title == nil || *opts.Title != "Updated Title" {
		t.Error("Title should be set")
	}
	if opts.Notes != nil {
		t.Error("Notes should be nil")
	}
	if opts.Status == nil || *opts.Status != "completed" {
		t.Error("Status should be set to 'completed'")
	}
}

// TestMultiAccountHandlerAccountProperty verifies account property in tool schemas
func TestMultiAccountHandlerAccountProperty(t *testing.T) {
	handler := &MultiAccountHandler{
		accountManager: nil,
		defaultClient:  nil,
	}

	tools := handler.GetTools()

	// All tools should have an account property
	for _, tool := range tools {
		if tool.Name == "tasks_list_tasklists_all_accounts" {
			// This tool aggregates across all accounts, so no account property needed
			continue
		}

		if _, hasAccount := tool.InputSchema.Properties["account"]; !hasAccount {
			t.Errorf("Tool %s should have 'account' property", tool.Name)
		}
	}
}
