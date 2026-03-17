package api

import (
	"context"
	"fmt"
	"strings"
)

// ColumnRule describes the inferred fill rule for a column.
type ColumnRule struct {
	Index       int           `json:"index"`
	Header      string        `json:"header"`
	Type        string        `json:"type"`         // "enum", "freetext", "url", "number", "empty"
	Required    bool          `json:"required"`
	FillRate    string        `json:"fill_rate"`     // e.g. "5/5"
	EnumValues  []string      `json:"enum_values,omitempty"`  // if type=enum
	Samples     []string      `json:"samples,omitempty"`      // if type=freetext
	Description string        `json:"description"`  // human-readable hint
}

// SheetSchema describes the fill structure of a sheet.
type SheetSchema struct {
	SpreadsheetID string       `json:"spreadsheet_id"`
	SheetName     string       `json:"sheet_name"`
	Columns       []ColumnRule `json:"columns"`
	ColumnCount   int          `json:"column_count"`
	DataRows      int          `json:"data_rows"`
	Instructions  string       `json:"instructions"` // agent-readable fill guide
}

// DescribeSheet analyzes a sheet's structure and infers fill rules for each column.
func (ss *SheetsService) DescribeSheet(ctx context.Context, spreadsheetID, sheetRange string, sampleRows int) (*SheetSchema, error) {
	if sampleRows <= 0 {
		sampleRows = 20
	}

	// If no range specified, auto-detect first sheet
	if sheetRange == "" {
		info, err := ss.GetInfo(ctx, spreadsheetID)
		if err != nil {
			return nil, err
		}
		if len(info.Sheets) > 0 {
			sheetRange = info.Sheets[0].Title
		} else {
			sheetRange = "Sheet1"
		}
	}

	data, err := ss.ReadRange(ctx, spreadsheetID, sheetRange)
	if err != nil {
		return nil, err
	}

	if len(data.Values) < 1 {
		return nil, fmt.Errorf("sheet is empty, no headers found")
	}

	headers := data.Values[0]
	rows := data.Values[1:]
	if len(rows) > sampleRows {
		rows = rows[:sampleRows]
	}

	var columns []ColumnRule
	for i, header := range headers {
		headerStr := fmt.Sprintf("%v", header)
		col := analyzeColumn(i, headerStr, rows)
		columns = append(columns, col)
	}

	schema := &SheetSchema{
		SpreadsheetID: spreadsheetID,
		SheetName:     sheetRange,
		Columns:       columns,
		ColumnCount:   len(columns),
		DataRows:      len(rows),
		Instructions:  generateFillInstructions(columns),
	}

	return schema, nil
}

// ValidateRow validates a proposed row against the sheet schema.
func ValidateRow(schema *SheetSchema, row []interface{}) *ValidationResult {
	result := &ValidationResult{Valid: true}

	for _, col := range schema.Columns {
		var value string
		if col.Index < len(row) {
			value = fmt.Sprintf("%v", row[col.Index])
		}

		issue := validateCell(col, value)
		if issue != "" {
			result.Valid = false
			result.Issues = append(result.Issues, ValidationIssue{
				Column:  col.Header,
				Index:   col.Index,
				Value:   value,
				Message: issue,
			})
		}
	}

	return result
}

// ValidationResult is the output of ValidateRow.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Issues []ValidationIssue `json:"issues,omitempty"`
}

// ValidationIssue describes a single validation problem.
type ValidationIssue struct {
	Column  string `json:"column"`
	Index   int    `json:"index"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// --- Internal analysis functions ---

func analyzeColumn(index int, header string, rows [][]interface{}) ColumnRule {
	col := ColumnRule{
		Index:  index,
		Header: header,
	}

	// Collect values from this column
	var values []string
	filled := 0
	for _, row := range rows {
		if index < len(row) {
			v := strings.TrimSpace(fmt.Sprintf("%v", row[index]))
			if v != "" {
				values = append(values, v)
				filled++
			}
		}
	}

	total := len(rows)
	col.FillRate = fmt.Sprintf("%d/%d", filled, total)
	col.Required = filled == total && total > 0

	if len(values) == 0 {
		col.Type = "empty"
		col.Description = "Column is empty in all sampled rows"
		return col
	}

	// Detect type
	unique := uniqueValues(values)

	// Check if all are URLs
	urlCount := 0
	for _, v := range values {
		if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
			urlCount++
		}
	}
	if urlCount > len(values)/2 {
		col.Type = "url"
		col.Samples = truncateSlice(unique, 3)
		col.Description = fmt.Sprintf("URL field. %d/%d filled.", filled, total)
		return col
	}

	// Check if all are numbers
	numCount := 0
	for _, v := range values {
		if isNumeric(v) {
			numCount++
		}
	}
	if numCount == len(values) {
		col.Type = "number"
		col.Samples = truncateSlice(unique, 5)
		col.Description = fmt.Sprintf("Numeric field. %d/%d filled.", filled, total)
		return col
	}

	// Enum: few unique values relative to total
	if len(unique) <= 5 && len(unique) < len(values) {
		col.Type = "enum"
		col.EnumValues = unique
		col.Description = fmt.Sprintf("Select from: %s. %d/%d filled.", strings.Join(unique, " / "), filled, total)
		return col
	}

	// Default: freetext
	col.Type = "freetext"
	col.Samples = truncateSlice(unique, 3)

	// Truncate long samples
	for i, s := range col.Samples {
		if len(s) > 60 {
			col.Samples[i] = s[:57] + "..."
		}
	}

	optionalHint := ""
	headerLower := strings.ToLower(header)
	if strings.Contains(headerLower, "选填") || strings.Contains(headerLower, "optional") {
		col.Required = false
		optionalHint = " (optional)"
	}

	col.Description = fmt.Sprintf("Free text%s. %d/%d filled.", optionalHint, filled, total)
	return col
}

func validateCell(col ColumnRule, value string) string {
	if col.Required && value == "" {
		return fmt.Sprintf("required field '%s' is empty", col.Header)
	}

	if value == "" {
		return "" // optional and empty = ok
	}

	switch col.Type {
	case "enum":
		found := false
		for _, ev := range col.EnumValues {
			if strings.EqualFold(value, ev) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Sprintf("value '%s' not in allowed values: %s", value, strings.Join(col.EnumValues, ", "))
		}
	case "url":
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Sprintf("expected URL, got '%s'", value)
		}
	case "number":
		if !isNumeric(value) {
			return fmt.Sprintf("expected number, got '%s'", value)
		}
	}

	return ""
}

func generateFillInstructions(columns []ColumnRule) string {
	var sb strings.Builder
	sb.WriteString("Fill guide:\n")
	for _, col := range columns {
		req := "optional"
		if col.Required {
			req = "REQUIRED"
		}
		sb.WriteString(fmt.Sprintf("  [%d] %s (%s, %s): %s\n", col.Index, col.Header, col.Type, req, col.Description))
	}
	return sb.String()
}

func uniqueValues(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func truncateSlice(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	dotCount := 0
	for i, c := range s {
		if c == '-' && i == 0 {
			continue
		}
		if c == '.' {
			dotCount++
			if dotCount > 1 {
				return false
			}
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
