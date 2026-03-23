package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redredchen01/gwx/internal/api"
)

// TestPathDiscovery verifies the URL paths that Google SDK generates
// when using option.WithEndpoint. This is a regression test — if Google
// updates their client libraries to use different path conventions, this
// test will catch it so MockAPI routing can be updated.
func TestPathDiscovery(t *testing.T) {
	var paths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		w.Header().Set("Content-Type", "application/json")

		path := r.URL.Path
		switch {
		case path == "/files":
			json.NewEncoder(w).Encode(map[string]interface{}{"kind": "drive#fileList", "files": []interface{}{}})
		case path == "/calendars/primary/events":
			json.NewEncoder(w).Encode(map[string]interface{}{"kind": "calendar#events", "items": []interface{}{}})
		case path == "/gmail/v1/users/me/messages":
			json.NewEncoder(w).Encode(map[string]interface{}{"messages": []interface{}{}, "resultSizeEstimate": 0})
		case path == "/gmail/v1/users/me/labels":
			json.NewEncoder(w).Encode(map[string]interface{}{"labels": []interface{}{}})
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	}))
	defer srv.Close()

	client := api.NewTestClient(srv.Client(), srv.URL)

	// Drive
	driveSvc := api.NewDriveService(client)
	driveSvc.ListFiles(t.Context(), "", 5)

	// Calendar
	calSvc := api.NewCalendarService(client)
	calSvc.Agenda(t.Context(), 1)

	// Gmail
	gmailSvc := api.NewGmailService(client)
	gmailSvc.ListMessages(t.Context(), "", nil, 5, false)

	// Gmail labels
	gmailSvc.ListLabels(t.Context())

	// Verify expected paths
	expected := map[string]bool{
		"GET /files":                         false,
		"GET /calendars/primary/events":      false,
		"GET /gmail/v1/users/me/messages":    false,
		"GET /gmail/v1/users/me/labels":      false,
	}
	for _, p := range paths {
		if _, ok := expected[p]; ok {
			expected[p] = true
		}
	}
	for path, found := range expected {
		if !found {
			t.Errorf("expected path %q not seen in requests. Got: %v", path, paths)
		}
	}
}
