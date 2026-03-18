package output

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/redredchen01/gwx/internal/exitcode"
)

// Format represents the output format.
type Format int

const (
	FormatJSON  Format = iota
	FormatPlain
	FormatTable
)

// ParseFormat parses a format string.
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "plain":
		return FormatPlain
	case "table":
		return FormatTable
	default:
		return FormatJSON
	}
}

// Response is the unified JSON envelope for all gwx output.
type Response struct {
	Status string      `json:"status"`          // "ok" or "error"
	Data   interface{} `json:"data,omitempty"`
	Error  *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo carries structured error details.
type ErrorInfo struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// Printer handles formatted output.
// Printer handles formatted output.
type Printer struct {
	Format Format
	Writer io.Writer
	Fields []string // if set, filter JSON output to these top-level keys
}

// NewPrinter creates a printer with the given format.
func NewPrinter(f Format) *Printer {
	return &Printer{Format: f, Writer: os.Stdout}
}

// Success prints a success response.
func (p *Printer) Success(data interface{}) {
	if len(p.Fields) > 0 {
		data = filterFields(data, p.Fields)
	}

	switch p.Format {
	case FormatJSON:
		resp := Response{Status: "ok", Data: data}
		enc := json.NewEncoder(p.Writer)
		enc.SetIndent("", "  ")
		enc.Encode(resp) //nolint:errcheck
	case FormatPlain:
		fmt.Fprintf(p.Writer, "%v\n", data)
	case FormatTable:
		fmt.Fprintf(p.Writer, "%v\n", data)
	}
}

// filterFields keeps only the specified keys from a map or struct (via JSON roundtrip).
func filterFields(data interface{}, fields []string) interface{} {
	// Convert to map via JSON roundtrip
	raw, err := json.Marshal(data)
	if err != nil {
		return data
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		// Might be a slice — try filtering each element
		var arr []interface{}
		if err := json.Unmarshal(raw, &arr); err != nil {
			return data
		}
		var filtered []interface{}
		for _, item := range arr {
			filtered = append(filtered, filterFields(item, fields))
		}
		return filtered
	}

	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[f] = true
	}

	result := make(map[string]interface{})
	for k, v := range m {
		if fieldSet[k] {
			result[k] = v
		}
	}
	return result
}

// Err prints an error response and returns the exit code.
func (p *Printer) Err(code int, msg string) int {
	switch p.Format {
	case FormatJSON:
		resp := Response{
			Status: "error",
			Error: &ErrorInfo{
				Code:    code,
				Name:    exitcode.Description(code),
				Message: msg,
			},
		}
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(resp) //nolint:errcheck
	default:
		slog.Error("output error", "message", msg)
	}
	return code
}

// Table prints data as a table. headers are column names, rows are string slices.
func (p *Printer) Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(p.Writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	fmt.Fprintln(w, strings.Repeat("─\t", len(headers)))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// ErrExit prints an error and returns it as a special ExitError for the CLI to handle.
func (p *Printer) ErrExit(code int, msg string) error {
	p.Err(code, msg)
	return &ExitError{Code: code, Msg: msg}
}

// ExitError carries an exit code through the error chain.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string { return e.Msg }

// IsTTY checks if stdout is a terminal.
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
