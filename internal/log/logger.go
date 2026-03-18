// Package log provides structured logging helpers for gwx.
//
// MCP server mode must never write to stdout (stdout carries JSON-RPC).
// All log output goes to stderr only.
package log

import (
	"log/slog"
	"os"
)

// SetupMCPLogger returns a JSON-format slog.Logger that writes to stderr.
// Use this in MCP server mode where stdout is reserved for JSON-RPC traffic.
func SetupMCPLogger() *slog.Logger {
	h := slog.NewJSONHandler(os.Stderr, nil)
	return slog.New(h)
}

// SetupCLILogger returns a logger that writes to stderr.
// It uses Text handler when stderr is an interactive terminal,
// and JSON handler otherwise (piped / agent-friendly).
func SetupCLILogger() *slog.Logger {
	var h slog.Handler
	if isTerminal(os.Stderr) {
		h = slog.NewTextHandler(os.Stderr, nil)
	} else {
		h = slog.NewJSONHandler(os.Stderr, nil)
	}
	return slog.New(h)
}

// isTerminal reports whether f is connected to an interactive terminal.
// Uses os.ModeCharDevice — no external dependency required.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
