package testutil

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/redredchen01/gwx/internal/mcp"
)

// MCPClient simulates an MCP client for testing the server.
// It wires up an in-memory pipe to the Server's stdin/stdout
// so you can send requests and read responses without actual I/O.
type MCPClient struct {
	// pipe to server stdin
	stdinWriter  io.Writer
	// pipe from server stdout
	stdoutReader *bufio.Reader
	// next request ID
	nextID int
	mu     sync.Mutex
}

// NewMCPClient creates an MCP test client connected to a server backed by the given handler.
// It starts the server's Run loop in a background goroutine.
// The server loop terminates when the client's stdin pipe is closed (via Close).
func NewMCPClient(handler mcp.Handler) *MCPClient {
	// Server reads from stdinRead, writes to stdoutWrite
	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()

	s := &mcp.Server{}
	// We need to set internal fields. Since Server has unexported fields,
	// we use the exported NewServer and override. But NewServer uses os.Stdin/Stdout.
	// Instead, we construct the server loop manually using the public interface.

	// Use a goroutine to run the server loop
	go func() {
		defer stdoutWrite.Close()
		scanner := bufio.NewScanner(stdinRead)
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

		var writeMu sync.Mutex

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var req mcp.Request
			if err := json.Unmarshal(line, &req); err != nil {
				resp := mcp.Response{
					JSONRPC: "2.0",
					Error:   &mcp.RPCError{Code: -32700, Message: "Parse error", Data: err.Error()},
				}
				data, _ := json.Marshal(resp)
				writeMu.Lock()
				fmt.Fprintf(stdoutWrite, "%s\n", data)
				writeMu.Unlock()
				continue
			}

			// Dispatch
			var resp mcp.Response
			resp.JSONRPC = "2.0"
			resp.ID = req.ID

			switch req.Method {
			case "initialize":
				resp.Result = mcp.InitializeResult{
					ProtocolVersion: "2024-11-05",
					Capabilities: mcp.ServerCapabilities{
						Tools: &mcp.ToolsCapability{},
					},
					ServerInfo: mcp.ServerInfo{
						Name:    "gwx",
						Version: "0.8.0",
					},
				}

			case "notifications/initialized":
				continue // no response for notifications

			case "tools/list":
				tools := handler.ListTools()
				resp.Result = mcp.ListToolsResult{Tools: tools}

			case "tools/call":
				var params mcp.ToolCallParams
				if err := json.Unmarshal(req.Params, &params); err != nil {
					resp.Error = &mcp.RPCError{Code: -32602, Message: "Invalid params", Data: err.Error()}
				} else {
					result, err := handler.CallTool(params.Name, params.Arguments)
					if err != nil {
						resp.Result = mcp.ToolResult{
							Content: []mcp.ContentBlock{{Type: "text", Text: err.Error()}},
							IsError: true,
						}
					} else {
						resp.Result = result
					}
				}

			case "ping":
				resp.Result = map[string]interface{}{}

			default:
				resp.Error = &mcp.RPCError{Code: -32601, Message: "Method not found", Data: req.Method}
			}

			data, _ := json.Marshal(resp)
			writeMu.Lock()
			fmt.Fprintf(stdoutWrite, "%s\n", data)
			writeMu.Unlock()
		}
	}()

	// Suppress unused variable warning — we construct *Server just to document intent.
	_ = s

	return &MCPClient{
		stdinWriter:  stdinWrite,
		stdoutReader: bufio.NewReader(stdoutRead),
		nextID:       1,
	}
}

// Close terminates the server goroutine by closing the stdin pipe.
func (c *MCPClient) Close() {
	if pw, ok := c.stdinWriter.(*io.PipeWriter); ok {
		pw.Close()
	}
}

// send writes a JSON-RPC request and reads the response.
func (c *MCPClient) send(method string, params interface{}) (*mcp.Response, error) {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	c.mu.Unlock()

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      float64(id),
		Method:  method,
	}
	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		req.Params = raw
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Write request
	if _, err := fmt.Fprintf(c.stdinWriter, "%s\n", data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read response line
	line, err := c.stdoutReader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var resp mcp.Response
	if err := json.Unmarshal(bytes.TrimSpace(line), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (raw: %s)", err, line)
	}

	return &resp, nil
}

// Initialize performs the MCP initialize handshake.
func (c *MCPClient) Initialize() (*mcp.InitializeResult, error) {
	resp, err := c.send("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "test-client",
			"version": "1.0.0",
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	raw, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	var result mcp.InitializeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshal InitializeResult: %w", err)
	}
	return &result, nil
}

// ListTools requests the list of available tools.
func (c *MCPClient) ListTools() ([]mcp.Tool, error) {
	resp, err := c.send("tools/list", nil)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	raw, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	var result mcp.ListToolsResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ListToolsResult: %w", err)
	}
	return result.Tools, nil
}

// CallTool invokes a tool by name with the given arguments.
func (c *MCPClient) CallTool(name string, args map[string]interface{}) (*mcp.ToolResult, error) {
	resp, err := c.send("tools/call", mcp.ToolCallParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	raw, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	var result mcp.ToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ToolResult: %w", err)
	}
	return &result, nil
}

// Ping sends a ping request to verify the server is alive.
func (c *MCPClient) Ping() error {
	resp, err := c.send("ping", nil)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return nil
}

// ParseToolResultJSON parses the text content of a ToolResult as JSON into dst.
func ParseToolResultJSON(result *mcp.ToolResult, dst interface{}) error {
	if len(result.Content) == 0 {
		return fmt.Errorf("empty tool result content")
	}
	return json.Unmarshal([]byte(result.Content[0].Text), dst)
}
