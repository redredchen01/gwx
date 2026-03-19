package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/redredchen01/gwx/internal/exitcode"
)

// PipeCmd chains gwx commands using JSON stdin/stdout piping.
type PipeCmd struct {
	Pipeline string `arg:"" help:"Pipeline expression: 'cmd1 | cmd2 | cmd3' (e.g. 'gmail search invoice | sheets append SHEET_ID A:C')"`
}

func (c *PipeCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "pipe"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	stages := parsePipeline(c.Pipeline)
	if len(stages) == 0 {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, "empty pipeline")
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":  true,
			"command":  "pipe",
			"stages":   stages,
			"count":    len(stages),
		})
		return nil
	}

	// Execute pipeline: each stage feeds JSON to the next
	var lastOutput []byte

	for i, stage := range stages {
		args := buildGwxArgs(stage, rctx)

		gwxPath, err := os.Executable()
		if err != nil {
			gwxPath = "gwx"
		}

		cmd := exec.CommandContext(rctx.Context, gwxPath, args...)
		cmd.Stderr = os.Stderr

		// Pipe previous output as stdin (for stages after the first)
		if i > 0 && len(lastOutput) > 0 {
			cmd.Stdin = strings.NewReader(string(lastOutput))
			// Set env to signal piped input
			cmd.Env = append(os.Environ(), "GWX_PIPE_INPUT=1")
		}

		output, err := cmd.Output()
		if err != nil {
			return rctx.Printer.ErrExit(exitcode.GeneralError,
				fmt.Sprintf("stage %d (%s) failed: %v", i+1, stage, err))
		}
		lastOutput = output
	}

	// Output the final result
	if len(lastOutput) > 0 {
		// Try to pretty-print if valid JSON
		var parsed interface{}
		if err := json.Unmarshal(lastOutput, &parsed); err == nil {
			rctx.Printer.Success(parsed)
		} else {
			fmt.Fprint(rctx.Printer.Writer, string(lastOutput))
		}
	}

	return nil
}

// parsePipeline splits a pipeline string by '|' and trims each stage.
func parsePipeline(s string) []string {
	parts := strings.Split(s, "|")
	var stages []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			stages = append(stages, p)
		}
	}
	return stages
}

// buildGwxArgs converts a stage string into gwx CLI arguments.
// Adds --format json to ensure machine-readable output between stages.
func buildGwxArgs(stage string, rctx *RunContext) []string {
	// Split respecting quoted strings
	parts := splitArgs(stage)

	// Always force JSON output for pipeline stages
	args := append(parts, "--format", "json")

	// Pass through account flag
	if rctx.Account != "default" {
		args = append(args, "--account", rctx.Account)
	}

	return args
}

// splitArgs splits a command string into arguments, respecting quoted strings.
func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		} else if c == '"' || c == '\'' {
			inQuote = true
			quoteChar = c
		} else if c == ' ' || c == '\t' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
