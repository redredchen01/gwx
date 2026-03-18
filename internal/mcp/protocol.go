package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// JSON-RPC 2.0 types for MCP protocol

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP-specific types

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo        `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]Property    `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default,omitempty"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// Server runs the MCP stdio server loop.
type Server struct {
	handler Handler
	reader  *bufio.Reader
	writer  io.Writer
	mu      sync.Mutex
}

// Handler processes MCP tool calls.
type Handler interface {
	ListTools() []Tool
	CallTool(name string, args map[string]interface{}) (*ToolResult, error)
}

// NewServer creates a new MCP server.
func NewServer(h Handler) *Server {
	return &Server{
		handler: h,
		reader:  bufio.NewReader(os.Stdin),
		writer:  os.Stdout,
	}
}

// Run starts the MCP server loop.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.reader)
	// MCP messages can be large
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(&req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req *Request) {
	switch req.Method {
	case "initialize":
		s.sendResult(req.ID, InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
			ServerInfo: ServerInfo{
				Name:    "gwx",
				Version: "0.8.0",
			},
		})

	case "notifications/initialized":
		// No response needed for notifications

	case "tools/list":
		tools := s.handler.ListTools()
		s.sendResult(req.ID, ListToolsResult{Tools: tools})

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(req.ID, -32602, "Invalid params", err.Error())
			return
		}

		result, err := s.handler.CallTool(params.Name, params.Arguments)
		if err != nil {
			s.sendResult(req.ID, ToolResult{
				Content: []ContentBlock{{Type: "text", Text: err.Error()}},
				IsError: true,
			})
			return
		}
		s.sendResult(req.ID, result)

	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})

	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal result", "error", err)
		s.sendError(id, -32603, "Internal error", "failed to marshal result")
		return
	}
	s.mu.Lock()
	fmt.Fprintf(s.writer, "%s\n", data)
	s.mu.Unlock()
}

func (s *Server) sendError(id interface{}, code int, message, data string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message, Data: data},
	}
	d, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal error response", "error", err)
		return
	}
	s.mu.Lock()
	fmt.Fprintf(s.writer, "%s\n", d)
	s.mu.Unlock()
}
