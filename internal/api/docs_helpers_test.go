package api

import (
	"testing"

	"golang.org/x/oauth2"
)

// oauth2StaticTS creates a static token source for testing.
func oauth2StaticTS(token string) oauth2.TokenSource {
	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
		TokenType:   "Bearer",
	})
}

func TestExportMimeType(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"pdf", "application/pdf"},
		{"PDF", "application/pdf"},
		{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"DOCX", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"txt", "text/plain"},
		{"html", "text/html"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := exportMimeType(tt.format)
		if got != tt.want {
			t.Errorf("exportMimeType(%q) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestNewGmailService(t *testing.T) {
	svc := NewGmailService(nil)
	if svc == nil {
		t.Fatal("NewGmailService returned nil")
	}
}

func TestNewCalendarService(t *testing.T) {
	svc := NewCalendarService(nil)
	if svc == nil {
		t.Fatal("NewCalendarService returned nil")
	}
}

func TestNewDriveService(t *testing.T) {
	svc := NewDriveService(nil)
	if svc == nil {
		t.Fatal("NewDriveService returned nil")
	}
}

func TestNewDocsService(t *testing.T) {
	svc := NewDocsService(nil)
	if svc == nil {
		t.Fatal("NewDocsService returned nil")
	}
}

func TestNewSheetsService(t *testing.T) {
	svc := NewSheetsService(nil)
	if svc == nil {
		t.Fatal("NewSheetsService returned nil")
	}
}

func TestNewTasksService(t *testing.T) {
	svc := NewTasksService(nil)
	if svc == nil {
		t.Fatal("NewTasksService returned nil")
	}
}

func TestNewContactsService(t *testing.T) {
	svc := NewContactsService(nil)
	if svc == nil {
		t.Fatal("NewContactsService returned nil")
	}
}

func TestNewChatService(t *testing.T) {
	svc := NewChatService(nil)
	if svc == nil {
		t.Fatal("NewChatService returned nil")
	}
}

// --- SanitizeValues ---

func TestSanitizeValues_FormulaInjection(t *testing.T) {
	values := [][]interface{}{
		{"=SUM(A1)", "normal", "+cmd", "-1", "@mention"},
	}
	result := SanitizeValues(values)
	row := result[0]
	tests := []struct {
		idx  int
		want string
	}{
		{0, "'=SUM(A1)"},
		{1, "normal"},
		{2, "'+cmd"},
		{3, "'-1"},
		{4, "'@mention"},
	}
	for _, tt := range tests {
		got, ok := row[tt.idx].(string)
		if !ok {
			t.Errorf("row[%d] is not a string", tt.idx)
			continue
		}
		if got != tt.want {
			t.Errorf("row[%d] = %q, want %q", tt.idx, got, tt.want)
		}
	}
}

func TestSanitizeValues_NonString(t *testing.T) {
	values := [][]interface{}{
		{42, true, nil},
	}
	result := SanitizeValues(values)
	if result[0][0] != 42 {
		t.Error("non-string values should pass through unchanged")
	}
}

func TestSanitizeValues_EmptyString(t *testing.T) {
	values := [][]interface{}{
		{""},
	}
	result := SanitizeValues(values)
	if result[0][0] != "" {
		t.Error("empty string should pass through unchanged")
	}
}

func TestSanitizeValues_SafeString(t *testing.T) {
	values := [][]interface{}{
		{"hello", "world"},
	}
	result := SanitizeValues(values)
	if result[0][0] != "hello" {
		t.Error("safe strings should pass through unchanged")
	}
}

// --- NewClient / NewTestClient ---

func TestNewClient(t *testing.T) {
	ts := oauth2StaticTS("test-token")
	c := NewClient(ts)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.cache == nil {
		t.Error("cache should not be nil")
	}
	if c.rateLimiter == nil {
		t.Error("rateLimiter should not be nil")
	}
}

func TestNewTestClient(t *testing.T) {
	c := NewTestClient(nil, "http://localhost:1234")
	if c == nil {
		t.Fatal("NewTestClient returned nil")
	}
	if !c.NoCache {
		t.Error("test client should have NoCache=true")
	}
	if c.endpoint != "http://localhost:1234" {
		t.Errorf("endpoint = %q, want http://localhost:1234", c.endpoint)
	}
}

func TestClient_Breaker(t *testing.T) {
	ts := oauth2StaticTS("test")
	c := NewClient(ts)
	cb1 := c.breaker("gmail")
	cb2 := c.breaker("gmail")
	if cb1 != cb2 {
		t.Error("same service should return same circuit breaker")
	}
	cb3 := c.breaker("drive")
	if cb1 == cb3 {
		t.Error("different services should return different circuit breakers")
	}
}

func TestClient_IsCircuitOpen(t *testing.T) {
	ts := oauth2StaticTS("test")
	c := NewClient(ts)
	if c.IsCircuitOpen("gmail") {
		t.Error("new client should have closed circuits")
	}
}
