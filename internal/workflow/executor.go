package workflow

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/redredchen01/gwx/internal/output"
)

// Action describes a side-effect to execute.
type Action struct {
	Name        string                                         // e.g. "send_email", "create_event"
	Description string                                         // human-readable preview line
	Fn          func(ctx context.Context) (interface{}, error) // actual execution
}

// ExecuteOpts controls execution behavior.
type ExecuteOpts struct {
	Execute bool // --execute flag
	NoInput bool // --no-input flag (skip confirmation)
	IsMCP   bool // MCP mode: never execute
}

// ExecuteResult holds outcomes of action execution.
type ExecuteResult struct {
	Executed  bool           `json:"executed"`
	Cancelled bool           `json:"cancelled,omitempty"`
	Reason    string         `json:"reason,omitempty"`
	Actions   []ActionResult `json:"actions,omitempty"`
}

// ActionResult holds the outcome of a single action.
type ActionResult struct {
	Name    string      `json:"name"`
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Dispatch shows preview, asks confirmation if TTY, then executes actions.
// If IsMCP=true or Execute=false, returns without executing.
func Dispatch(ctx context.Context, actions []Action, opts ExecuteOpts) (*ExecuteResult, error) {
	// MCP mode: never execute
	if opts.IsMCP {
		return &ExecuteResult{Executed: false, Reason: "mcp_read_only"}, nil
	}

	// No --execute flag
	if !opts.Execute {
		return &ExecuteResult{Executed: false, Reason: "no_execute_flag"}, nil
	}

	// TTY confirmation (unless --no-input)
	if !opts.NoInput && output.IsTTY() {
		// Print preview to stderr
		fmt.Fprintf(os.Stderr, "\n--- Action Preview ---\n")
		for i, a := range actions {
			fmt.Fprintf(os.Stderr, "  %d. [%s] %s\n", i+1, a.Name, a.Description)
		}
		fmt.Fprintf(os.Stderr, "\nExecute %d action(s)? [y/N] ", len(actions))

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			return &ExecuteResult{Cancelled: true, Reason: "user declined"}, nil
		}
	}

	// Execute all actions
	results := make([]ActionResult, 0, len(actions))
	for _, a := range actions {
		val, err := a.Fn(ctx)
		ar := ActionResult{Name: a.Name, Success: err == nil}
		if err != nil {
			ar.Error = err.Error()
		} else {
			ar.Result = val
		}
		results = append(results, ar)
	}

	return &ExecuteResult{Executed: true, Actions: results}, nil
}
