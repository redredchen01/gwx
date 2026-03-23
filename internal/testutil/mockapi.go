package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// MockAPI provides a fake Google API server for testing.
// Configure the response slices before making API calls through it.
//
// Path routing is based on empirically discovered paths from the Google SDK
// when using option.WithEndpoint. Different SDKs use different path conventions:
//   - Gmail: /gmail/v1/users/me/messages (full path)
//   - Drive: /files (relative, no /drive/v3 prefix)
//   - Calendar: /calendars/primary/events (relative, no /calendar/v3 prefix)
//   - Sheets: /v4/spreadsheets/{id}/... (partial prefix)
type MockAPI struct {
	Server            *httptest.Server
	GmailMessages     []map[string]interface{}
	CalendarEvents    []map[string]interface{}
	DriveFiles        []map[string]interface{}
	SpreadsheetMeta   map[string]interface{} // response for GET /v4/spreadsheets/{id}
	SpreadsheetValues map[string]interface{} // response for GET /v4/spreadsheets/{id}/values/{range}

	mu       sync.Mutex
	requests []RecordedRequest // recorded requests for assertions
}

// RecordedRequest captures an incoming request for test assertions.
type RecordedRequest struct {
	Method string
	Path   string
	Query  string
}

// NewMockAPI creates and starts a mock Google API server.
// The caller must call Close() when done.
func NewMockAPI() *MockAPI {
	m := &MockAPI{
		GmailMessages:     []map[string]interface{}{SampleGmailMessage()},
		CalendarEvents:    []map[string]interface{}{SampleCalendarEvent()},
		DriveFiles:        []map[string]interface{}{SampleDriveFile()},
		SpreadsheetMeta:   SampleSpreadsheet(),
		SpreadsheetValues: SampleSpreadsheetValues(),
	}

	mux := http.NewServeMux()

	// =====================
	// Gmail routes (full paths — Gmail SDK keeps /gmail/v1/ prefix)
	// =====================

	// Gmail: send message — must be registered before the catch-all messages/ route
	mux.HandleFunc("/gmail/v1/users/me/messages/send", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		writeJSON(w, map[string]interface{}{
			"id":       "sent_001",
			"threadId": "thread_sent_001",
			"labelIds": []interface{}{"SENT"},
		})
	})

	// Gmail: get single message — /gmail/v1/users/me/messages/{id}
	mux.HandleFunc("/gmail/v1/users/me/messages/", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		msgID := lastPathSegment(r.URL.Path)
		for _, msg := range m.GmailMessages {
			if msg["id"] == msgID {
				writeJSON(w, msg)
				return
			}
		}
		http.Error(w, `{"error":{"code":404,"message":"Not Found"}}`, http.StatusNotFound)
	})

	// Gmail: list messages — exact path match
	mux.HandleFunc("/gmail/v1/users/me/messages", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		var msgRefs []map[string]interface{}
		for _, msg := range m.GmailMessages {
			msgRefs = append(msgRefs, map[string]interface{}{
				"id":       msg["id"],
				"threadId": msg["threadId"],
			})
		}
		writeJSON(w, map[string]interface{}{
			"messages":           msgRefs,
			"resultSizeEstimate": float64(len(m.GmailMessages)),
		})
	})

	// Gmail: list labels
	mux.HandleFunc("/gmail/v1/users/me/labels", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		writeJSON(w, SampleGmailLabels())
	})

	// Gmail: batch modify messages (for archive/mark-read)
	mux.HandleFunc("/gmail/v1/users/me/messages/batchModify", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		w.WriteHeader(http.StatusNoContent)
	})

	// =====================
	// Calendar routes (relative paths — Calendar SDK strips /calendar/v3/ prefix)
	// =====================

	// Calendar: events under any calendar ID
	mux.HandleFunc("/calendars/", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		path := r.URL.Path
		if strings.Contains(path, "/events") {
			switch r.Method {
			case http.MethodGet:
				writeJSON(w, map[string]interface{}{
					"kind":  "calendar#events",
					"items": interfaceSlice(m.CalendarEvents),
				})
			case http.MethodPost:
				event := SampleCalendarEvent()
				event["id"] = "evt_new_001"
				writeJSON(w, event)
			default:
				writeJSON(w, map[string]interface{}{})
			}
			return
		}
		http.Error(w, `{"error":{"code":404,"message":"Not Found"}}`, http.StatusNotFound)
	})

	// =====================
	// Drive routes (relative paths — Drive SDK strips /drive/v3/ prefix)
	// =====================

	// Drive: list/search files
	mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		writeJSON(w, map[string]interface{}{
			"kind":  "drive#fileList",
			"files": interfaceSlice(m.DriveFiles),
		})
	})

	// Drive: file operations with ID — /files/{id}
	mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		writeJSON(w, SampleDriveFile())
	})

	// =====================
	// Sheets routes (keeps /v4/spreadsheets/ prefix)
	// =====================

	mux.HandleFunc("/v4/spreadsheets/", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		path := r.URL.Path
		// /v4/spreadsheets/{id}/values/{range}:append — append values (POST)
		if strings.Contains(path, ":append") {
			writeJSON(w, map[string]interface{}{
				"spreadsheetId": "sheet_001",
				"updates": map[string]interface{}{
					"updatedRange": "Tasks!A4:C4",
					"updatedRows":  float64(1),
					"updatedCells": float64(3),
				},
			})
			return
		}
		// /v4/spreadsheets/{id}/values/{range} — read values
		if strings.Contains(path, "/values/") {
			writeJSON(w, m.SpreadsheetValues)
			return
		}
		// /v4/spreadsheets/{id} — spreadsheet metadata
		writeJSON(w, m.SpreadsheetMeta)
	})

	// =====================
	// Default handler for unmatched routes
	// =====================
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		http.Error(w, `{"error":{"code":404,"message":"Not Found","status":"NOT_FOUND"}}`, http.StatusNotFound)
	})

	m.Server = httptest.NewServer(mux)
	return m
}

// Close shuts down the mock server.
func (m *MockAPI) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// URL returns the base URL of the mock server.
func (m *MockAPI) URL() string {
	return m.Server.URL
}

// HTTPClient returns an *http.Client configured to talk to this mock server.
func (m *MockAPI) HTTPClient() *http.Client {
	return m.Server.Client()
}

// Requests returns a copy of all recorded requests for assertions.
func (m *MockAPI) Requests() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]RecordedRequest, len(m.requests))
	copy(cp, m.requests)
	return cp
}

// ResetRequests clears the recorded request history.
func (m *MockAPI) ResetRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = nil
}

// RequestCount returns the number of recorded requests.
func (m *MockAPI) RequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.requests)
}

func (m *MockAPI) record(r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, RecordedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.RawQuery,
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func lastPathSegment(path string) string {
	path = strings.TrimRight(path, "/")
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}

func interfaceSlice(src []map[string]interface{}) []interface{} {
	out := make([]interface{}, len(src))
	for i, v := range src {
		out[i] = v
	}
	return out
}
