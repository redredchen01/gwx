package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"google.golang.org/api/sheets/v4"
)

// --- S1: Stats ---

// ColumnStat holds stats for one column.
type ColumnStat struct {
	Header      string         `json:"header"`
	Index       int            `json:"index"`
	Filled      int            `json:"filled"`
	Empty       int            `json:"empty"`
	FillRate    string         `json:"fill_rate"`
	ValueCounts map[string]int `json:"value_counts,omitempty"` // for enum-like columns
}

// SheetStats holds overall sheet statistics.
type SheetStats struct {
	SheetName string       `json:"sheet_name"`
	TotalRows int          `json:"total_rows"`
	TotalCols int          `json:"total_cols"`
	Columns   []ColumnStat `json:"columns"`
	Summary   string       `json:"summary"`
}

// StatsRange computes statistics for a sheet range.
func (ss *SheetsService) StatsRange(ctx context.Context, spreadsheetID, statsRange string) (*SheetStats, error) {
	if statsRange == "" {
		info, err := ss.GetInfo(ctx, spreadsheetID)
		if err != nil {
			return nil, err
		}
		if len(info.Sheets) > 0 {
			statsRange = info.Sheets[0].Title
		}
	}

	data, err := ss.ReadRange(ctx, spreadsheetID, statsRange)
	if err != nil {
		return nil, err
	}

	if len(data.Values) < 2 {
		return nil, fmt.Errorf("need at least header + 1 data row")
	}

	headers := data.Values[0]
	rows := data.Values[1:]

	var columns []ColumnStat
	var summaryParts []string

	for i, header := range headers {
		headerStr := fmt.Sprintf("%v", header)
		cs := ColumnStat{
			Header: headerStr,
			Index:  i,
		}

		valueCounts := make(map[string]int)
		for _, row := range rows {
			var val string
			if i < len(row) {
				val = strings.TrimSpace(fmt.Sprintf("%v", row[i]))
			}
			if val == "" {
				cs.Empty++
			} else {
				cs.Filled++
				valueCounts[val]++
			}
		}

		cs.FillRate = fmt.Sprintf("%d/%d", cs.Filled, len(rows))

		// Include value counts if there are <= 10 unique values (enum-like)
		if len(valueCounts) <= 10 && len(valueCounts) > 0 {
			cs.ValueCounts = valueCounts
			// Add to summary if it looks like a status column
			if len(valueCounts) <= 5 {
				var parts []string
				for v, c := range valueCounts {
					parts = append(parts, fmt.Sprintf("%s: %d", v, c))
				}
				summaryParts = append(summaryParts, fmt.Sprintf("%s → %s", headerStr, strings.Join(parts, ", ")))
			}
		}

		columns = append(columns, cs)
	}

	summary := fmt.Sprintf("%d rows, %d columns. ", len(rows), len(headers))
	if len(summaryParts) > 0 {
		summary += strings.Join(summaryParts, "; ")
	}

	return &SheetStats{
		SheetName: statsRange,
		TotalRows: len(rows),
		TotalCols: len(headers),
		Columns:   columns,
		Summary:   summary,
	}, nil
}

// --- S2: Diff ---

// DiffChange represents one changed row.
type DiffChange struct {
	Type    string        `json:"type"` // "added", "removed", "modified"
	Key     string        `json:"key"`  // first column value (identifier)
	RowA    []interface{} `json:"row_a,omitempty"`
	RowB    []interface{} `json:"row_b,omitempty"`
	Changes []CellChange  `json:"changes,omitempty"`
}

// CellChange describes a single cell change.
type CellChange struct {
	Column string `json:"column"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// SheetDiff holds diff results between two ranges.
type SheetDiff struct {
	RangeA   string       `json:"range_a"`
	RangeB   string       `json:"range_b"`
	Changes  []DiffChange `json:"changes"`
	Summary  string       `json:"summary"`
	Added    int          `json:"added"`
	Removed  int          `json:"removed"`
	Modified int          `json:"modified"`
}

// DiffRanges compares two ranges (e.g., two week tabs) using the first column as key.
func (ss *SheetsService) DiffRanges(ctx context.Context, spreadsheetID, rangeA, rangeB string) (*SheetDiff, error) {
	// Parallel read both ranges
	var dataA, dataB *SheetData
	var errA, errB error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		dataA, errA = ss.ReadRange(ctx, spreadsheetID, rangeA)
	}()
	go func() {
		defer wg.Done()
		dataB, errB = ss.ReadRange(ctx, spreadsheetID, rangeB)
	}()
	wg.Wait()

	if errA != nil {
		return nil, fmt.Errorf("read range A: %w", errA)
	}
	if errB != nil {
		return nil, fmt.Errorf("read range B: %w", errB)
	}

	if len(dataA.Values) < 1 || len(dataB.Values) < 1 {
		return nil, fmt.Errorf("both ranges need at least headers")
	}

	headersA := dataA.Values[0]
	rowsA := dataA.Values[1:]
	rowsB := dataB.Values[1:]

	// Index by first column (key)
	mapA := indexByKey(rowsA)
	mapB := indexByKey(rowsB)

	var changes []DiffChange
	added, removed, modified := 0, 0, 0

	// Check for modifications and removals
	for key, rowA := range mapA {
		rowB, exists := mapB[key]
		if !exists {
			changes = append(changes, DiffChange{Type: "removed", Key: key, RowA: rowA})
			removed++
			continue
		}
		// Compare cells
		var cellChanges []CellChange
		maxLen := len(rowA)
		if len(rowB) > maxLen {
			maxLen = len(rowB)
		}
		for i := 1; i < maxLen; i++ { // skip first column (key)
			valA := cellStr(rowA, i)
			valB := cellStr(rowB, i)
			if valA != valB {
				colName := ""
				if i < len(headersA) {
					colName = fmt.Sprintf("%v", headersA[i])
				}
				cellChanges = append(cellChanges, CellChange{Column: colName, From: valA, To: valB})
			}
		}
		if len(cellChanges) > 0 {
			changes = append(changes, DiffChange{Type: "modified", Key: key, RowA: rowA, RowB: rowB, Changes: cellChanges})
			modified++
		}
	}

	// Check for additions
	for key, rowB := range mapB {
		if _, exists := mapA[key]; !exists {
			changes = append(changes, DiffChange{Type: "added", Key: key, RowB: rowB})
			added++
		}
	}

	summary := fmt.Sprintf("%d added, %d removed, %d modified (out of %d → %d rows)", added, removed, modified, len(rowsA), len(rowsB))

	return &SheetDiff{
		RangeA:   rangeA,
		RangeB:   rangeB,
		Changes:  changes,
		Summary:  summary,
		Added:    added,
		Removed:  removed,
		Modified: modified,
	}, nil
}

// --- S3: Copy Tab ---

// CopyTab copies a source tab's headers + first column to a new tab.
func (ss *SheetsService) CopyTab(ctx context.Context, spreadsheetID, sourceTab, newTabName string) error {
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return err
	}

	// Read source headers + first column
	data, err := ss.ReadRange(ctx, spreadsheetID, sourceTab)
	if err != nil {
		return fmt.Errorf("read source tab: %w", err)
	}
	if len(data.Values) < 1 {
		return fmt.Errorf("source tab is empty")
	}

	opts, err := ss.client.ClientOptions(ctx, "sheets")
	if err != nil {
		return err
	}
	svc, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return fmt.Errorf("create sheets service: %w", err)
	}

	// Create new sheet tab
	addReq := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: newTabName,
					},
				},
			},
		},
	}
	if _, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, addReq).Do(); err != nil {
		return fmt.Errorf("create new tab: %w", err)
	}

	// Prepare data: headers + key column only
	var newData [][]interface{}
	newData = append(newData, data.Values[0]) // headers
	for _, row := range data.Values[1:] {
		if len(row) > 0 && fmt.Sprintf("%v", row[0]) != "" {
			// Keep first column, empty rest
			newRow := make([]interface{}, len(data.Values[0]))
			newRow[0] = row[0]
			newData = append(newData, newRow)
		}
	}

	// Write to new tab
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return err
	}
	vr := &sheets.ValueRange{Values: newData}
	if _, err := svc.Spreadsheets.Values.Update(spreadsheetID, newTabName+"!A1", vr).
		ValueInputOption("RAW").Do(); err != nil {
		return fmt.Errorf("write to new tab: %w", err)
	}

	return nil
}

// --- S4: Export ---

// ExportCSV writes range data as CSV to a writer.
func (ss *SheetsService) ExportCSV(ctx context.Context, spreadsheetID, exportRange string, w io.Writer) (int, error) {
	data, err := ss.ReadRange(ctx, spreadsheetID, exportRange)
	if err != nil {
		return 0, err
	}

	writer := csv.NewWriter(w)
	for _, row := range data.Values {
		var record []string
		for _, cell := range row {
			record = append(record, fmt.Sprintf("%v", cell))
		}
		if err := writer.Write(record); err != nil {
			return 0, fmt.Errorf("write csv: %w", err)
		}
	}
	writer.Flush()
	return len(data.Values), writer.Error()
}

// ExportJSON writes range data as JSON to a writer.
func (ss *SheetsService) ExportJSON(ctx context.Context, spreadsheetID, exportRange string, w io.Writer) (int, error) {
	data, err := ss.ReadRange(ctx, spreadsheetID, exportRange)
	if err != nil {
		return 0, err
	}

	if len(data.Values) < 2 {
		return 0, fmt.Errorf("need at least header + data rows")
	}

	headers := data.Values[0]
	rows := data.Values[1:]

	var records []map[string]interface{}
	for _, row := range rows {
		record := make(map[string]interface{})
		for i, header := range headers {
			key := fmt.Sprintf("%v", header)
			if i < len(row) {
				record[key] = row[i]
			} else {
				record[key] = ""
			}
		}
		records = append(records, record)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(records); err != nil {
		return 0, err
	}
	return len(records), nil
}

// --- Helpers ---

func indexByKey(rows [][]interface{}) map[string][]interface{} {
	m := make(map[string][]interface{})
	for _, row := range rows {
		if len(row) > 0 {
			key := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
			if key != "" {
				m[key] = row
			}
		}
	}
	return m
}

func cellStr(row []interface{}, index int) string {
	if index >= len(row) {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", row[index]))
}

// --- S5: Import ---

// ImportCSV reads CSV from a reader and appends to a sheet.
func (ss *SheetsService) ImportCSV(ctx context.Context, spreadsheetID, importRange string, r io.Reader, hasHeader bool) (*SheetAppendResult, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV is empty")
	}

	var values [][]interface{}
	start := 0
	if hasHeader {
		start = 1
	}
	for i := start; i < len(records); i++ {
		var row []interface{}
		for _, cell := range records[i] {
			row = append(row, cell)
		}
		values = append(values, row)
	}

	return ss.AppendValues(ctx, spreadsheetID, importRange, values)
}

// ImportJSON reads JSON array of objects and appends to a sheet.
func (ss *SheetsService) ImportJSON(ctx context.Context, spreadsheetID, importRange string, r io.Reader) (*SheetAppendResult, error) {
	var records []map[string]interface{}
	if err := json.NewDecoder(r).Decode(&records); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("JSON array is empty")
	}

	// Collect all keys for headers
	keySet := make(map[string]bool)
	var keys []string
	for _, rec := range records {
		for k := range rec {
			if !keySet[k] {
				keySet[k] = true
				keys = append(keys, k)
			}
		}
	}

	// Build header row + data rows
	var values [][]interface{}
	var headerRow []interface{}
	for _, k := range keys {
		headerRow = append(headerRow, k)
	}
	values = append(values, headerRow)

	for _, rec := range records {
		var row []interface{}
		for _, k := range keys {
			row = append(row, fmt.Sprintf("%v", rec[k]))
		}
		values = append(values, row)
	}

	return ss.AppendValues(ctx, spreadsheetID, importRange, values)
}

// ImportFromFile reads CSV or JSON from a file and imports to a sheet.
func (ss *SheetsService) ImportFromFile(ctx context.Context, spreadsheetID, importRange, format, path string) (*SheetAppendResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	switch strings.ToLower(format) {
	case "csv":
		return ss.ImportCSV(ctx, spreadsheetID, importRange, f, true)
	case "json":
		return ss.ImportJSON(ctx, spreadsheetID, importRange, f)
	default:
		return nil, fmt.Errorf("unsupported format %q (use csv or json)", format)
	}
}

// ExportToFile is a convenience wrapper that writes to a file.
func (ss *SheetsService) ExportToFile(ctx context.Context, spreadsheetID, exportRange, format, path string) (int, error) {
	var w io.Writer
	if path == "" || path == "-" {
		w = os.Stdout
	} else {
		f, err := os.Create(path)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		w = f
	}

	switch strings.ToLower(format) {
	case "csv":
		return ss.ExportCSV(ctx, spreadsheetID, exportRange, w)
	case "json":
		return ss.ExportJSON(ctx, spreadsheetID, exportRange, w)
	default:
		return 0, fmt.Errorf("unsupported format %q (use csv or json)", format)
	}
}
