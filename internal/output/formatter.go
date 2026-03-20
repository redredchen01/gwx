package output

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
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
	Action  string `json:"action,omitempty"`
}

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
		p.printPlain(data)
	case FormatTable:
		p.printTable(data)
	}
}

// printPlain outputs data in a human-readable plain text format.
func (p *Printer) printPlain(data interface{}) {
	// Convert to map via JSON roundtrip
	raw, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(p.Writer, "%v\n", data)
		return
	}

	// Try as map
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err == nil {
		p.printPlainMap(m, "")
		return
	}

	// Try as array
	var arr []interface{}
	if err := json.Unmarshal(raw, &arr); err == nil {
		for i, item := range arr {
			if i > 0 {
				fmt.Fprintln(p.Writer, "---")
			}
			if im, ok := item.(map[string]interface{}); ok {
				p.printPlainMap(im, "")
			} else {
				fmt.Fprintf(p.Writer, "%v\n", item)
			}
		}
		return
	}

	fmt.Fprintf(p.Writer, "%v\n", data)
}

func (p *Printer) printPlainMap(m map[string]interface{}, indent string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		switch val := v.(type) {
		case map[string]interface{}:
			fmt.Fprintf(p.Writer, "%s%s:\n", indent, k)
			p.printPlainMap(val, indent+"  ")
		case []interface{}:
			fmt.Fprintf(p.Writer, "%s%s: (%d items)\n", indent, k, len(val))
			for _, item := range val {
				if im, ok := item.(map[string]interface{}); ok {
					// Print key fields inline
					summary := mapSummary(im)
					fmt.Fprintf(p.Writer, "%s  - %s\n", indent, summary)
				} else {
					fmt.Fprintf(p.Writer, "%s  - %v\n", indent, item)
				}
			}
		default:
			fmt.Fprintf(p.Writer, "%s%s: %v\n", indent, k, v)
		}
	}
}

// mapSummary picks the most useful fields from a map for one-line display.
func mapSummary(m map[string]interface{}) string {
	// Priority fields to show
	keys := []string{"subject", "title", "name", "summary", "id", "email", "status"}
	var parts []string
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil && fmt.Sprintf("%v", v) != "" {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		if len(parts) >= 3 {
			break
		}
	}
	if len(parts) == 0 {
		// Fallback: first 3 fields
		for k, v := range m {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			if len(parts) >= 3 {
				break
			}
		}
	}
	return strings.Join(parts, "  ")
}

// printTable outputs data as a formatted table using tabwriter.
func (p *Printer) printTable(data interface{}) {
	raw, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(p.Writer, "%v\n", data)
		return
	}

	// Try to find an array in the data (common pattern: {"messages": [...], "count": N})
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err == nil {
		// Find the largest array field (deterministic: sort keys, pick largest)
		sortedKeys := make([]string, 0, len(m))
		for k := range m {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)
		var bestArr []interface{}
		for _, k := range sortedKeys {
			if arr, ok := m[k].([]interface{}); ok && len(arr) > len(bestArr) {
				bestArr = arr
			}
		}
		if len(bestArr) > 0 {
			p.renderArrayAsTable(bestArr)
			return
		}
		// No array found — render map as key-value table
		var headers []string
		var values []string
		for _, k := range sortedKeys {
			headers = append(headers, k)
			values = append(values, fmt.Sprintf("%v", m[k]))
		}
		p.Table(headers, [][]string{values})
		return
	}

	// Direct array
	var arr []interface{}
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		p.renderArrayAsTable(arr)
		return
	}

	fmt.Fprintf(p.Writer, "%v\n", data)
}

// renderArrayAsTable renders an array of objects as a table.
func (p *Printer) renderArrayAsTable(arr []interface{}) {
	if len(arr) == 0 {
		return
	}

	// Extract headers from first element
	first, ok := arr[0].(map[string]interface{})
	if !ok {
		for _, item := range arr {
			fmt.Fprintf(p.Writer, "%v\n", item)
		}
		return
	}

	// Collect headers (skip nested objects/arrays for table display), sorted for deterministic output
	var headers []string
	for k, v := range first {
		switch v.(type) {
		case map[string]interface{}, []interface{}:
			continue // skip nested
		default:
			headers = append(headers, k)
		}
	}
	sort.Strings(headers)

	// Build rows
	var rows [][]string
	for _, item := range arr {
		im, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		var row []string
		for _, h := range headers {
			val := ""
			if v, ok := im[h]; ok && v != nil {
				s := fmt.Sprintf("%v", v)
				if len(s) > 60 {
					s = s[:57] + "..."
				}
				val = s
			}
			row = append(row, val)
		}
		rows = append(rows, row)
	}

	p.Table(headers, rows)
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
	action := suggestedAction(code, msg)

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
		if action != "" {
			resp.Error.Action = action
		}
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(resp) //nolint:errcheck
	default:
		slog.Error("output error", "message", msg)
		if action != "" {
			fmt.Fprintf(os.Stderr, "  Fix: %s\n", action)
		}
	}
	return code
}

// suggestedAction returns a user-friendly fix suggestion based on error code and message.
func suggestedAction(code int, msg string) string {
	switch code {
	case exitcode.AuthRequired:
		return "Run 'gwx onboard' to set up credentials, or 'gwx auth login' to sign in."
	case exitcode.AuthExpired:
		return "Run 'gwx auth login' to refresh your token."
	case exitcode.PermissionDenied:
		if strings.Contains(msg, "allowlist") {
			return "Add this command to GWX_ENABLE_COMMANDS or remove the restriction."
		}
		return "You may need to re-authorize with additional scopes: 'gwx auth login'"
	case exitcode.NotFound:
		return "Check the ID/path and try again. Use 'gwx <service> list' to find valid IDs."
	case exitcode.RateLimited:
		return "Wait 30 seconds and retry. Google API quota may be exhausted."
	case exitcode.CircuitOpen:
		return "Google API is unstable. Wait 30 seconds for the circuit breaker to recover."
	case exitcode.Conflict:
		return "Resource was modified by another process. Retry your operation."
	case exitcode.InvalidInput:
		return "Check your command arguments. Run '<command> --help' for usage."
	}
	return ""
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
