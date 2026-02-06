package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
	"go.ngs.io/google-mcp-server/config"
)

// MCPServer represents the MCP server
type MCPServer struct {
	config    *config.Config
	services  map[string]ServiceHandler
	toolMap   map[string]ServiceHandler // O(1) tool name → service lookup
	conn      *jsonrpc2.Conn
	mu        sync.RWMutex
	tools     []Tool
	resources []Resource
}

// ServiceHandler represents a service that provides tools and resources
type ServiceHandler interface {
	GetTools() []Tool
	GetResources() []Resource
	HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error)
	HandleResourceCall(ctx context.Context, uri string) (interface{}, error)
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema represents the JSON schema for tool input
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property represents a property in the input schema
type Property struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Items       *Property `json:"items,omitempty"`
	Enum        []string  `json:"enum,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// maxMessageSize is the maximum allowed size for a single JSON-RPC message (10MB)
const maxMessageSize = 10 * 1024 * 1024

// NewMCPServer creates a new MCP server
func NewMCPServer(cfg *config.Config) *MCPServer {
	return &MCPServer{
		config:    cfg,
		services:  make(map[string]ServiceHandler),
		toolMap:   make(map[string]ServiceHandler),
		tools:     []Tool{},
		resources: []Resource{},
	}
}

// RegisterService registers a service handler
func (s *MCPServer) RegisterService(name string, handler ServiceHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.services[name]; exists {
		fmt.Fprintf(os.Stderr, "Warning: service %q already registered, overwriting\n", name)
	}

	s.services[name] = handler

	// Add tools from the service and build tool-to-service map for O(1) lookup
	tools := handler.GetTools()
	s.tools = append(s.tools, tools...)
	for _, tool := range tools {
		s.toolMap[tool.Name] = handler
	}

	// Add resources from the service
	resources := handler.GetResources()
	s.resources = append(s.resources, resources...)
}

// Start starts the MCP server
func (s *MCPServer) Start() error {
	// Create JSON-RPC connection using stdio
	handler := &Handler{server: s}

	// Create a newline-delimited JSON stream for MCP
	stream := NewNewlineDelimitedStream(os.Stdin, os.Stdout)

	conn := jsonrpc2.NewConn(
		context.Background(),
		stream,
		handler,
	)

	s.conn = conn

	// Wait for connection to close
	<-conn.DisconnectNotify()
	return nil
}

// Stop gracefully shuts down the MCP server
func (s *MCPServer) Stop() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// NewlineDelimitedStream implements jsonrpc2.ObjectStream for newline-delimited JSON
type NewlineDelimitedStream struct {
	scanner *bufio.Scanner
	writer  io.Writer
	mu      sync.Mutex
}

// NewNewlineDelimitedStream creates a new newline-delimited JSON stream
func NewNewlineDelimitedStream(r io.Reader, w io.Writer) *NewlineDelimitedStream {
	scanner := bufio.NewScanner(r)
	// Start with 64KB buffer, grow up to maxMessageSize; rejects oversized messages
	// without allocating them fully (prevents OOM from malicious input)
	scanner.Buffer(make([]byte, 0, 64*1024), maxMessageSize)
	return &NewlineDelimitedStream{
		scanner: scanner,
		writer:  w,
	}
}

// ReadObject reads a newline-delimited JSON object
func (s *NewlineDelimitedStream) ReadObject(v interface{}) error {
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return err
		}
		return io.EOF
	}
	return json.Unmarshal(s.scanner.Bytes(), v)
}

// WriteObject writes a newline-delimited JSON object
func (s *NewlineDelimitedStream) WriteObject(v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	_, err = s.writer.Write(data)
	if err != nil {
		return err
	}

	_, err = s.writer.Write([]byte("\n"))
	return err
}

// Close closes the stream
func (s *NewlineDelimitedStream) Close() error {
	// Don't close stdin/stdout
	return nil
}

// Handler handles JSON-RPC requests
type Handler struct {
	server *MCPServer
}

func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	switch req.Method {
	case "initialize":
		h.handleInitialize(ctx, conn, req)
	case "initialized":
		// Client confirms initialization
	case "tools/list":
		h.handleToolsList(ctx, conn, req)
	case "tools/call":
		h.handleToolCall(ctx, conn, req)
	case "resources/list":
		h.handleResourcesList(ctx, conn, req)
	case "resources/read":
		h.handleResourceRead(ctx, conn, req)
	case "completion/complete":
		h.handleCompletion(ctx, conn, req)
	default:
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeMethodNotFound,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
	}
}

func (h *Handler) handleInitialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Roots    interface{} `json:"roots,omitempty"`
			Sampling interface{} `json:"sampling,omitempty"`
		} `json:"capabilities"`
		ClientInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}

	if req.Params == nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing parameters",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid parameters",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	response := struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools     interface{} `json:"tools,omitempty"`
			Resources interface{} `json:"resources,omitempty"`
			Prompts   interface{} `json:"prompts,omitempty"`
			Logging   interface{} `json:"logging,omitempty"`
		} `json:"capabilities"`
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}{
		ProtocolVersion: "2024-11-05",
		ServerInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			Name:    "google-mcp-server",
			Version: VERSION,
		},
	}

	// Set capabilities
	response.Capabilities.Tools = struct{}{}
	response.Capabilities.Resources = struct{}{}

	if err := conn.Reply(ctx, req.ID, response); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
	}
}

func (h *Handler) handleToolsList(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.server.mu.RLock()
	tools := h.server.tools
	h.server.mu.RUnlock()

	response := struct {
		Tools []Tool `json:"tools"`
	}{
		Tools: tools,
	}

	if err := conn.Reply(ctx, req.ID, response); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
	}
}

func (h *Handler) handleToolCall(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if req.Params == nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing parameters",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid parameters",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	// Find the appropriate service handler via O(1) map lookup
	h.server.mu.RLock()
	handler, exists := h.server.toolMap[params.Name]
	h.server.mu.RUnlock()

	if !exists {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeMethodNotFound,
			Message: fmt.Sprintf("tool not found: %s", params.Name),
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	// Call the tool
	result, err := handler.HandleToolCall(ctx, params.Name, params.Arguments)
	if err != nil {
		// Log full error to stderr, return generic message to client
		fmt.Fprintf(os.Stderr, "Error in tool %s: %v\n", params.Name, err)
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInternalError,
			Message: "internal error",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	// Check if result is already a JSON string
	var responseText string
	switch v := result.(type) {
	case string:
		responseText = v
	case []byte:
		responseText = string(v)
	default:
		// Convert to JSON if not already a string
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			responseText = fmt.Sprintf("%v", result)
		} else {
			responseText = string(jsonBytes)
		}
	}

	response := struct {
		Content []struct {
			Type string      `json:"type"`
			Text string      `json:"text,omitempty"`
			Data interface{} `json:"data,omitempty"`
		} `json:"content"`
		IsError bool `json:"isError,omitempty"`
	}{
		Content: []struct {
			Type string      `json:"type"`
			Text string      `json:"text,omitempty"`
			Data interface{} `json:"data,omitempty"`
		}{
			{
				Type: "text",
				Text: responseText,
			},
		},
		IsError: false,
	}

	if err := conn.Reply(ctx, req.ID, response); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
	}
}

func (h *Handler) handleResourcesList(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.server.mu.RLock()
	resources := h.server.resources
	h.server.mu.RUnlock()

	response := struct {
		Resources []Resource `json:"resources"`
	}{
		Resources: resources,
	}

	if err := conn.Reply(ctx, req.ID, response); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
	}
}

func (h *Handler) handleResourceRead(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		URI string `json:"uri"`
	}

	if req.Params == nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing parameters",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid parameters",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	// Find the appropriate service handler
	h.server.mu.RLock()
	var handler ServiceHandler
	for _, service := range h.server.services {
		resources := service.GetResources()
		for _, resource := range resources {
			if resource.URI == params.URI {
				handler = service
				break
			}
		}
		if handler != nil {
			break
		}
	}
	h.server.mu.RUnlock()

	if handler == nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeMethodNotFound,
			Message: fmt.Sprintf("resource not found: %s", params.URI),
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	// Read the resource
	result, err := handler.HandleResourceCall(ctx, params.URI)
	if err != nil {
		// Log full error to stderr, return generic message to client
		fmt.Fprintf(os.Stderr, "Error reading resource %s: %v\n", params.URI, err)
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInternalError,
			Message: "internal error",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	response := struct {
		Contents []struct {
			URI      string `json:"uri"`
			MimeType string `json:"mimeType,omitempty"`
			Text     string `json:"text,omitempty"`
		} `json:"contents"`
	}{
		Contents: []struct {
			URI      string `json:"uri"`
			MimeType string `json:"mimeType,omitempty"`
			Text     string `json:"text,omitempty"`
		}{
			{
				URI:      params.URI,
				MimeType: "text/plain",
				Text:     fmt.Sprintf("%v", result),
			},
		},
	}

	if err := conn.Reply(ctx, req.ID, response); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
	}
}

func (h *Handler) handleCompletion(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		Ref struct {
			Type string `json:"type"`
			Name string `json:"name,omitempty"`
			URI  string `json:"uri,omitempty"`
		} `json:"ref"`
		Argument struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"argument"`
	}

	if req.Params == nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing parameters",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		if err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid parameters",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
		}
		return
	}

	// For now, return empty completions
	response := struct {
		Completion struct {
			Values []string `json:"values"`
		} `json:"completion"`
	}{
		Completion: struct {
			Values []string `json:"values"`
		}{
			Values: []string{},
		},
	}

	if err := conn.Reply(ctx, req.ID, response); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending reply: %v\n", err)
	}
}
