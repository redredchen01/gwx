package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupAPITestServer creates a comprehensive mock HTTP server for testing
// all Google API service wrapper functions.
func setupAPITestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// === Gmail ===
	mux.HandleFunc("/gmail/v1/users/me/messages/send", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"id": "sent_001", "threadId": "thread_sent_001", "labelIds": []interface{}{"SENT"},
		})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages/batchModify", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/gmail/v1/users/me/messages/", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, sampleGmailMsg())
	})
	mux.HandleFunc("/gmail/v1/users/me/messages", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"messages":           []interface{}{map[string]interface{}{"id": "msg_001", "threadId": "thread_001"}},
			"resultSizeEstimate": float64(1),
		})
	})
	mux.HandleFunc("/gmail/v1/users/me/labels", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"labels": []interface{}{
				map[string]interface{}{"id": "INBOX", "name": "INBOX", "type": "system"},
				map[string]interface{}{"id": "SENT", "name": "SENT", "type": "system"},
				map[string]interface{}{"id": "UNREAD", "name": "UNREAD", "type": "system"},
			},
		})
	})
	mux.HandleFunc("/gmail/v1/users/me/drafts", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"id": "draft_001",
			"message": map[string]interface{}{"id": "draft_msg_001", "threadId": "thread_draft", "labelIds": []interface{}{"DRAFT"}},
		})
	})

	// === Calendar ===
	mux.HandleFunc("/freeBusy", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{"kind": "calendar#freeBusy", "calendars": map[string]interface{}{}})
	})
	mux.HandleFunc("/calendars/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/events") {
			switch r.Method {
			case http.MethodGet:
				apiJSON(w, map[string]interface{}{
					"kind":  "calendar#events",
					"items": []interface{}{sampleCalEvt()},
				})
			case http.MethodPost:
				apiJSON(w, sampleCalEvt())
			case http.MethodPut, http.MethodPatch:
				apiJSON(w, sampleCalEvt())
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			}
			return
		}
		apiJSON(w, map[string]interface{}{})
	})

	// === Drive ===
	mux.HandleFunc("/upload/drive/v3/files", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, sampleDrvFile())
	})
	mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/permissions") {
			apiJSON(w, map[string]interface{}{"id": "perm_001", "role": "reader", "type": "user"})
			return
		}
		if strings.Contains(path, "/export") {
			w.Header().Set("Content-Type", "application/pdf")
			w.Write([]byte("%PDF-1.4 fake"))
			return
		}
		apiJSON(w, sampleDrvFile())
	})
	mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiJSON(w, sampleDrvFile())
			return
		}
		apiJSON(w, map[string]interface{}{
			"kind": "drive#fileList", "files": []interface{}{sampleDrvFile()},
		})
	})

	// === Sheets ===
	mux.HandleFunc("/v4/spreadsheets", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, sampleSsMeta())
	})
	mux.HandleFunc("/v4/spreadsheets/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, ":append") {
			apiJSON(w, map[string]interface{}{
				"spreadsheetId": "sheet_001",
				"updates": map[string]interface{}{
					"updatedRange": "Tasks!A4:C4", "updatedRows": float64(1), "updatedCells": float64(3),
				},
			})
			return
		}
		if strings.Contains(path, ":clear") {
			apiJSON(w, map[string]interface{}{"spreadsheetId": "sheet_001", "clearedRange": "Tasks!A1:C3"})
			return
		}
		if strings.Contains(path, ":batchUpdate") {
			apiJSON(w, map[string]interface{}{
				"spreadsheetId": "sheet_001",
				"replies":       []interface{}{map[string]interface{}{"addSheet": map[string]interface{}{"properties": map[string]interface{}{"sheetId": float64(99), "title": "NewTab"}}}},
			})
			return
		}
		if strings.Contains(path, "/values/") {
			if r.Method == http.MethodPut {
				apiJSON(w, map[string]interface{}{
					"spreadsheetId": "sheet_001", "updatedRange": "Tasks!A1:C3",
					"updatedRows": float64(3), "updatedCells": float64(9),
				})
				return
			}
			apiJSON(w, sampleSsValues())
			return
		}
		apiJSON(w, sampleSsMeta())
	})

	// === Docs ===
	mux.HandleFunc("/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"documentId": "doc_new_001", "title": "New Document", "revisionId": "rev_001",
			"body": map[string]interface{}{"content": []interface{}{}},
		})
	})
	mux.HandleFunc("/v1/documents/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiJSON(w, map[string]interface{}{
				"documentId": "doc_001",
				"replies": []interface{}{
					map[string]interface{}{"replaceAllText": map[string]interface{}{"occurrencesChanged": float64(2)}},
				},
			})
			return
		}
		apiJSON(w, map[string]interface{}{
			"documentId": "doc_001", "title": "Test Doc", "revisionId": "rev_001",
			"body": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"paragraph": map[string]interface{}{
							"elements": []interface{}{
								map[string]interface{}{"textRun": map[string]interface{}{"content": "Hello world test."}},
							},
						},
					},
				},
			},
		})
	})

	// === Tasks ===
	mux.HandleFunc("/tasks/v1/users/@me/lists", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"kind":  "tasks#taskLists",
			"items": []interface{}{map[string]interface{}{"id": "list_001", "title": "My Tasks"}},
		})
	})
	mux.HandleFunc("/tasks/v1/lists/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			apiJSON(w, map[string]interface{}{
				"kind": "tasks#tasks",
				"items": []interface{}{
					map[string]interface{}{
						"id": "task_001", "title": "Buy groceries",
						"status": "needsAction", "updated": time.Now().Format(time.RFC3339),
					},
				},
			})
		case http.MethodPost:
			apiJSON(w, map[string]interface{}{
				"id": "task_new", "title": "New Task",
				"status": "needsAction", "updated": time.Now().Format(time.RFC3339),
			})
		case http.MethodPut, http.MethodPatch:
			apiJSON(w, map[string]interface{}{
				"id": "task_001", "title": "Buy groceries",
				"status": "completed", "updated": time.Now().Format(time.RFC3339),
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	// === Contacts / People ===
	mux.HandleFunc("/v1/people:searchContacts", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{
					"person": map[string]interface{}{
						"resourceName":   "people/c001",
						"names":          []interface{}{map[string]interface{}{"displayName": "Alice Chen"}},
						"emailAddresses": []interface{}{map[string]interface{}{"value": "alice@example.com"}},
					},
				},
			},
		})
	})
	mux.HandleFunc("/v1/people/me/connections", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"connections": []interface{}{
				map[string]interface{}{
					"resourceName":   "people/c001",
					"names":          []interface{}{map[string]interface{}{"displayName": "Alice Chen"}},
					"emailAddresses": []interface{}{map[string]interface{}{"value": "alice@example.com"}},
				},
			},
			"totalPeople": float64(1),
		})
	})
	mux.HandleFunc("/v1/people/", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"resourceName":   "people/c001",
			"names":          []interface{}{map[string]interface{}{"displayName": "Alice Chen"}},
			"emailAddresses": []interface{}{map[string]interface{}{"value": "alice@example.com"}},
		})
	})

	// === Chat ===
	mux.HandleFunc("/v1/spaces", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"spaces": []interface{}{
				map[string]interface{}{"name": "spaces/AAAA", "displayName": "General", "type": "ROOM", "threaded": true},
			},
		})
	})
	mux.HandleFunc("/v1/spaces/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/messages") {
			if r.Method == http.MethodPost {
				apiJSON(w, map[string]interface{}{
					"name": "spaces/AAAA/messages/msg001", "text": "Hello",
					"createTime": time.Now().Format(time.RFC3339),
					"space":      map[string]interface{}{"name": "spaces/AAAA"},
				})
				return
			}
			apiJSON(w, map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"name": "spaces/AAAA/messages/msg001", "text": "Hello",
						"createTime": time.Now().Format(time.RFC3339),
						"sender":     map[string]interface{}{"name": "users/001", "displayName": "Alice"},
					},
				},
			})
			return
		}
		apiJSON(w, map[string]interface{}{"name": "spaces/AAAA", "displayName": "General"})
	})

	// === Slides ===
	mux.HandleFunc("/v1/presentations", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"presentationId": "pres_new", "title": "New Pres", "slides": []interface{}{},
		})
	})
	mux.HandleFunc("/v1/presentations/", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{
			"presentationId": "pres_001", "title": "Test Pres",
			"slides": []interface{}{
				map[string]interface{}{
					"objectId":     "slide_001",
					"pageElements": []interface{}{},
				},
			},
		})
	})

	// === Default ===
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		apiJSON(w, map[string]interface{}{})
	})

	return httptest.NewServer(mux)
}

func apiJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func sampleGmailMsg() map[string]interface{} {
	return map[string]interface{}{
		"id": "msg_001", "threadId": "thread_001",
		"snippet": "Hey let's sync.", "labelIds": []interface{}{"INBOX", "UNREAD"},
		"internalDate": "1711180800000", "sizeEstimate": 1234,
		"payload": map[string]interface{}{
			"mimeType": "text/plain",
			"headers": []interface{}{
				map[string]interface{}{"name": "From", "value": "alice@example.com"},
				map[string]interface{}{"name": "To", "value": "bob@example.com"},
				map[string]interface{}{"name": "Subject", "value": "Q2 Sync"},
				map[string]interface{}{"name": "Date", "value": "Mon, 23 Mar 2026 10:00:00 +0800"},
			},
			"body": map[string]interface{}{
				"size": 15, "data": "SGV5IGxldCdzIHN5bmMu",
			},
		},
	}
}

func sampleCalEvt() map[string]interface{} {
	return map[string]interface{}{
		"id": "evt_001", "summary": "Sprint Planning", "status": "confirmed",
		"start":    map[string]interface{}{"dateTime": "2026-03-23T10:00:00+08:00"},
		"end":      map[string]interface{}{"dateTime": "2026-03-23T11:00:00+08:00"},
		"location": "Room A",
		"organizer": map[string]interface{}{"email": "pm@example.com"},
		"attendees": []interface{}{map[string]interface{}{"email": "dev@example.com"}},
		"htmlLink":  "https://calendar.google.com/event?eid=evt_001",
	}
}

func sampleDrvFile() map[string]interface{} {
	return map[string]interface{}{
		"id": "file_001", "name": "Doc.docx",
		"mimeType": "application/vnd.google-apps.document", "size": "15360",
		"modifiedTime": "2026-03-22T16:30:00.000Z", "createdTime": "2026-03-01T09:00:00.000Z",
		"webViewLink": "https://docs.google.com/d/file_001",
		"parents": []interface{}{"root"}, "shared": false, "trashed": false,
	}
}

func sampleSsMeta() map[string]interface{} {
	return map[string]interface{}{
		"spreadsheetId": "sheet_001", "spreadsheetUrl": "https://docs.google.com/spreadsheets/d/sheet_001",
		"properties": map[string]interface{}{"title": "Tracker"},
		"sheets": []interface{}{
			map[string]interface{}{
				"properties": map[string]interface{}{
					"sheetId": float64(0), "title": "Tasks", "index": float64(0),
					"gridProperties": map[string]interface{}{"rowCount": float64(100), "columnCount": float64(10)},
				},
			},
		},
	}
}

func sampleSsValues() map[string]interface{} {
	return map[string]interface{}{
		"range": "Tasks!A1:C3", "majorDimension": "ROWS",
		"values": []interface{}{
			[]interface{}{"Task", "Status", "Assignee"},
			[]interface{}{"Build API", "In Progress", "Alice"},
			[]interface{}{"Write Tests", "Todo", "Bob"},
		},
	}
}

func apiTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return NewTestClient(server.Client(), server.URL)
}

// === Gmail Service Tests ===

func TestGmailService_ListMessages(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	msgs, total, err := svc.ListMessages(context.Background(), "", nil, 10, false)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].ID != "msg_001" {
		t.Errorf("expected ID=msg_001, got %s", msgs[0].ID)
	}
	if msgs[0].Subject != "Q2 Sync" {
		t.Errorf("expected subject 'Q2 Sync', got %q", msgs[0].Subject)
	}
	if msgs[0].From != "alice@example.com" {
		t.Errorf("expected from alice@example.com, got %q", msgs[0].From)
	}
}

func TestGmailService_ListMessages_Unread(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	msgs, _, err := svc.ListMessages(context.Background(), "", nil, 5, true)
	if err != nil {
		t.Fatalf("ListMessages unread: %v", err)
	}
	if len(msgs) == 0 {
		t.Error("expected at least 1 message")
	}
}

func TestGmailService_ListMessages_WithQuery(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	msgs, _, err := svc.ListMessages(context.Background(), "from:alice", nil, 5, false)
	if err != nil {
		t.Fatalf("ListMessages with query: %v", err)
	}
	if len(msgs) == 0 {
		t.Error("expected at least 1 message")
	}
}

func TestGmailService_ListMessages_WithLabel(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	msgs, _, err := svc.ListMessages(context.Background(), "", []string{"INBOX"}, 5, false)
	if err != nil {
		t.Fatalf("ListMessages with label: %v", err)
	}
	if len(msgs) == 0 {
		t.Error("expected at least 1 message")
	}
}

func TestGmailService_GetMessage(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	msg, err := svc.GetMessage(context.Background(), "msg_001")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if msg.ID != "msg_001" {
		t.Errorf("expected ID=msg_001, got %s", msg.ID)
	}
	if msg.Subject != "Q2 Sync" {
		t.Errorf("expected subject 'Q2 Sync', got %q", msg.Subject)
	}
}

func TestGmailService_SearchMessages(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	msgs, total, err := svc.SearchMessages(context.Background(), "subject:sync", 5)
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	if total < 1 {
		t.Errorf("expected total >= 1, got %d", total)
	}
	if len(msgs) == 0 {
		t.Error("expected at least 1 message")
	}
}

func TestGmailService_SendMessage(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	result, err := svc.SendMessage(context.Background(), &SendInput{
		To: []string{"test@example.com"}, Subject: "Test", Body: "Hello",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if result.MessageID != "sent_001" {
		t.Errorf("expected MessageID=sent_001, got %s", result.MessageID)
	}
}

func TestGmailService_CreateDraft(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	result, err := svc.CreateDraft(context.Background(), &SendInput{
		To: []string{"test@example.com"}, Subject: "Draft", Body: "Body",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if result.MessageID == "" {
		t.Error("expected non-empty MessageID")
	}
}

func TestGmailService_ListLabels(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	labels, err := svc.ListLabels(context.Background())
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	if len(labels) != 3 {
		t.Errorf("expected 3 labels, got %d", len(labels))
	}
}

func TestGmailService_DigestMessages(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	digest, err := svc.DigestMessages(context.Background(), 10, false)
	if err != nil {
		t.Fatalf("DigestMessages: %v", err)
	}
	if digest.TotalMessages != 1 {
		t.Errorf("expected 1 total message, got %d", digest.TotalMessages)
	}
}

func TestGmailService_ArchiveMessages(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	count, err := svc.ArchiveMessages(context.Background(), "from:test", 5)
	if err != nil {
		t.Fatalf("ArchiveMessages: %v", err)
	}
	if count < 0 {
		t.Errorf("expected non-negative count, got %d", count)
	}
}

func TestGmailService_MarkRead(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	count, err := svc.MarkRead(context.Background(), "from:test", 5)
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if count < 0 {
		t.Errorf("expected non-negative count, got %d", count)
	}
}

func TestGmailService_ReplyMessage(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	result, err := svc.ReplyMessage(context.Background(), "msg_001", &SendInput{Body: "Reply"})
	if err != nil {
		t.Fatalf("ReplyMessage: %v", err)
	}
	if result.MessageID == "" {
		t.Error("expected non-empty MessageID")
	}
}

func TestGmailService_ForwardMessage(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewGmailService(client)

	result, err := svc.ForwardMessage(context.Background(), "msg_001", []string{"fwd@example.com"})
	if err != nil {
		t.Fatalf("ForwardMessage: %v", err)
	}
	if result.MessageID == "" {
		t.Error("expected non-empty MessageID")
	}
}

// === Calendar Service Tests ===

func TestCalendarService_Agenda(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewCalendarService(client)

	events, err := svc.Agenda(context.Background(), 1)
	if err != nil {
		t.Fatalf("Agenda: %v", err)
	}
	if len(events) == 0 {
		t.Error("expected at least 1 event")
	}
}

func TestCalendarService_ListEvents(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewCalendarService(client)

	now := time.Now()
	events, err := svc.ListEvents(context.Background(), "primary", now, now.AddDate(0, 0, 7), 50)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) == 0 {
		t.Error("expected at least 1 event")
	}
	if events[0].ID != "evt_001" {
		t.Errorf("expected ID=evt_001, got %s", events[0].ID)
	}
}

func TestCalendarService_CreateEvent(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewCalendarService(client)

	event, err := svc.CreateEvent(context.Background(), "primary", &EventInput{
		Title: "Test", Start: time.Now().Add(time.Hour).Format(time.RFC3339),
		End: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
}

func TestCalendarService_UpdateEvent(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewCalendarService(client)

	event, err := svc.UpdateEvent(context.Background(), "primary", "evt_001", &EventInput{Title: "Updated"})
	if err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
}

func TestCalendarService_DeleteEvent(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewCalendarService(client)

	err := svc.DeleteEvent(context.Background(), "primary", "evt_001")
	if err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}
}

func TestCalendarService_FindSlot(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewCalendarService(client)

	slots, err := svc.FindSlot(context.Background(), []string{"dev@example.com"}, 30*time.Minute, 1)
	if err != nil {
		t.Fatalf("FindSlot: %v", err)
	}
	// slots may or may not be found depending on current time, but no error means success
	_ = slots
}

func TestCalendarService_CheckConflicts(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewCalendarService(client)

	start := time.Now().Add(time.Hour).Format(time.RFC3339)
	end := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	conflicts, err := svc.CheckConflicts(context.Background(), "primary", start, end)
	if err != nil {
		t.Fatalf("CheckConflicts: %v", err)
	}
	_ = conflicts // may be empty, that's fine
}

// === Drive Service Tests ===

func TestDriveService_ListFiles(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDriveService(client)

	files, err := svc.ListFiles(context.Background(), "", 20)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].ID != "file_001" {
		t.Errorf("expected ID=file_001, got %s", files[0].ID)
	}
}

func TestDriveService_SearchFiles(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDriveService(client)

	files, err := svc.SearchFiles(context.Background(), "name contains 'report'", 10)
	if err != nil {
		t.Fatalf("SearchFiles: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected at least 1 file")
	}
}

func TestDriveService_CreateFolder(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDriveService(client)

	folder, err := svc.CreateFolder(context.Background(), "New Folder", "")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if folder == nil {
		t.Fatal("expected non-nil folder")
	}
}

func TestDriveService_ShareFile(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDriveService(client)

	err := svc.ShareFile(context.Background(), "file_001", "share@example.com", "reader")
	if err != nil {
		t.Fatalf("ShareFile: %v", err)
	}
}

func TestDriveService_CheckDownloadSize(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDriveService(client)

	err := svc.CheckDownloadSize(context.Background(), "file_001", 100*1024*1024)
	if err != nil {
		t.Fatalf("CheckDownloadSize: %v", err)
	}
}

// === Sheets Service Tests ===

func TestSheetsService_ReadRange(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	data, err := svc.ReadRange(context.Background(), "sheet_001", "Tasks!A1:C3")
	if err != nil {
		t.Fatalf("ReadRange: %v", err)
	}
	if data.RowCount != 3 {
		t.Errorf("expected 3 rows, got %d", data.RowCount)
	}
}

func TestSheetsService_AppendValues(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	result, err := svc.AppendValues(context.Background(), "sheet_001", "Tasks!A:C",
		[][]interface{}{{"Task3", "Done", "Charlie"}})
	if err != nil {
		t.Fatalf("AppendValues: %v", err)
	}
	if result.UpdatedRows != 1 {
		t.Errorf("expected 1 updated row, got %d", result.UpdatedRows)
	}
}

func TestSheetsService_UpdateValues(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	result, err := svc.UpdateValues(context.Background(), "sheet_001", "Tasks!A1:C1",
		[][]interface{}{{"Task", "Status", "Assignee"}})
	if err != nil {
		t.Fatalf("UpdateValues: %v", err)
	}
	if result.UpdatedRows != 3 {
		t.Errorf("expected 3 updated rows, got %d", result.UpdatedRows)
	}
}

func TestSheetsService_GetInfo(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	info, err := svc.GetInfo(context.Background(), "sheet_001")
	if err != nil {
		t.Fatalf("GetInfo: %v", err)
	}
	if info.SpreadsheetID != "sheet_001" {
		t.Errorf("expected spreadsheet_id=sheet_001, got %s", info.SpreadsheetID)
	}
	if info.SheetCount != 1 {
		t.Errorf("expected 1 sheet, got %d", info.SheetCount)
	}
}

func TestSheetsService_ClearRange(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	err := svc.ClearRange(context.Background(), "sheet_001", "Tasks!A1:C3")
	if err != nil {
		t.Fatalf("ClearRange: %v", err)
	}
}

func TestSheetsService_CreateSpreadsheet(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	result, err := svc.CreateSpreadsheet(context.Background(), "New Sheet")
	if err != nil {
		t.Fatalf("CreateSpreadsheet: %v", err)
	}
	if result.SpreadsheetID == "" {
		t.Error("expected non-empty spreadsheet ID")
	}
}

func TestSheetsService_SearchValues(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	result, err := svc.SearchValues(context.Background(), "sheet_001", "Tasks!A:C", "Alice")
	if err != nil {
		t.Fatalf("SearchValues: %v", err)
	}
	if result.MatchCount < 0 {
		t.Errorf("expected non-negative match count, got %d", result.MatchCount)
	}
}

func TestSheetsService_FilterRows(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	result, err := svc.FilterRows(context.Background(), "sheet_001", "Tasks!A:C", 1, "In Progress")
	if err != nil {
		t.Fatalf("FilterRows: %v", err)
	}
	if result.MatchCount < 0 {
		t.Errorf("expected non-negative match count, got %d", result.MatchCount)
	}
}

func TestSheetsService_DescribeSheet(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	schema, err := svc.DescribeSheet(context.Background(), "sheet_001", "Tasks", 20)
	if err != nil {
		t.Fatalf("DescribeSheet: %v", err)
	}
	if schema.ColumnCount != 3 {
		t.Errorf("expected 3 columns, got %d", schema.ColumnCount)
	}
}

func TestSheetsService_StatsRange(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	stats, err := svc.StatsRange(context.Background(), "sheet_001", "Tasks")
	if err != nil {
		t.Fatalf("StatsRange: %v", err)
	}
	if stats.TotalRows != 2 {
		t.Errorf("expected 2 data rows, got %d", stats.TotalRows)
	}
}

func TestSheetsService_DiffRanges(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSheetsService(client)

	diff, err := svc.DiffRanges(context.Background(), "sheet_001", "Tasks", "Tasks")
	if err != nil {
		t.Fatalf("DiffRanges: %v", err)
	}
	if diff.Summary == "" {
		t.Error("expected non-empty summary")
	}
}

// === Docs Service Tests ===

func TestDocsService_GetDocument(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDocsService(client)

	doc, err := svc.GetDocument(context.Background(), "doc_001")
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if doc.DocumentID != "doc_001" {
		t.Errorf("expected doc_001, got %s", doc.DocumentID)
	}
}

func TestDocsService_CreateDocument(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDocsService(client)

	doc, err := svc.CreateDocument(context.Background(), "New Doc", "Body text")
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil doc")
	}
}

func TestDocsService_SearchDocument(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDocsService(client)

	result, err := svc.SearchDocument(context.Background(), "doc_001", "world")
	if err != nil {
		t.Fatalf("SearchDocument: %v", err)
	}
	_ = result // result structure depends on doc content
}

func TestDocsService_ReplaceText(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDocsService(client)

	count, err := svc.ReplaceText(context.Background(), "doc_001", "old", "new")
	if err != nil {
		t.Fatalf("ReplaceText: %v", err)
	}
	if count < 0 {
		t.Errorf("expected non-negative count, got %d", count)
	}
}

func TestDocsService_AppendText(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewDocsService(client)

	err := svc.AppendText(context.Background(), "doc_001", "Appended text")
	if err != nil {
		t.Fatalf("AppendText: %v", err)
	}
}

// === Tasks Service Tests ===

func TestTasksService_ListTaskLists(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewTasksService(client)

	lists, err := svc.ListTaskLists(context.Background())
	if err != nil {
		t.Fatalf("ListTaskLists: %v", err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}
}

func TestTasksService_ListTasks(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewTasksService(client)

	tasks, err := svc.ListTasks(context.Background(), "@default", false)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) == 0 {
		t.Error("expected at least 1 task")
	}
}

func TestTasksService_CreateTask(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewTasksService(client)

	task, err := svc.CreateTask(context.Background(), "@default", "New Task", "Notes", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task == nil {
		t.Fatal("expected non-nil task")
	}
}

func TestTasksService_CompleteTask(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewTasksService(client)

	task, err := svc.CompleteTask(context.Background(), "@default", "task_001")
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	if task == nil {
		t.Fatal("expected non-nil task")
	}
}

func TestTasksService_DeleteTask(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewTasksService(client)

	err := svc.DeleteTask(context.Background(), "@default", "task_001")
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
}

// === Contacts Service Tests ===

func TestContactsService_SearchContacts(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewContactsService(client)

	contacts, err := svc.SearchContacts(context.Background(), "Alice", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(contacts) == 0 {
		t.Error("expected at least 1 contact")
	}
}

func TestContactsService_ListContacts(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewContactsService(client)

	contacts, err := svc.ListContacts(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if len(contacts) == 0 {
		t.Error("expected at least 1 contact")
	}
}

func TestContactsService_GetContact(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewContactsService(client)

	contact, err := svc.GetContact(context.Background(), "people/c001")
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	if contact == nil {
		t.Fatal("expected non-nil contact")
	}
}

// === Chat Service Tests ===

func TestChatService_ListSpaces(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewChatService(client)

	spaces, err := svc.ListSpaces(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListSpaces: %v", err)
	}
	if len(spaces) == 0 {
		t.Error("expected at least 1 space")
	}
}

func TestChatService_SendMessage(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewChatService(client)

	result, err := svc.SendMessage(context.Background(), "spaces/AAAA", "Hello")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestChatService_ListMessages(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewChatService(client)

	messages, err := svc.ListMessages(context.Background(), "spaces/AAAA", 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) == 0 {
		t.Error("expected at least 1 message")
	}
}

// === Slides Service Tests ===

func TestSlidesService_GetPresentation(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSlidesService(client)

	result, err := svc.GetPresentation(context.Background(), "pres_001")
	if err != nil {
		t.Fatalf("GetPresentation: %v", err)
	}
	if result.PresentationID != "pres_001" {
		t.Errorf("expected pres_001, got %s", result.PresentationID)
	}
}

func TestSlidesService_ListPresentations(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSlidesService(client)

	files, err := svc.ListPresentations(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListPresentations: %v", err)
	}
	_ = files // may be empty if no presentations match mime type filter
}

func TestSlidesService_CreatePresentation(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := apiTestClient(t, srv)
	svc := NewSlidesService(client)

	result, err := svc.CreatePresentation(context.Background(), "New Pres")
	if err != nil {
		t.Fatalf("CreatePresentation: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// === Client Tests ===

func TestNewTestClient_Integration(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := NewTestClient(srv.Client(), srv.URL)

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if !client.NoCache {
		t.Error("expected NoCache=true for test client")
	}
	if client.endpoint != srv.URL {
		t.Errorf("expected endpoint=%s, got %s", srv.URL, client.endpoint)
	}
}

func TestClient_ServiceInit(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := NewTestClient(srv.Client(), srv.URL)

	opts, err := client.ServiceInit(context.Background(), "gmail")
	if err != nil {
		t.Fatalf("ServiceInit: %v", err)
	}
	if len(opts) == 0 {
		t.Error("expected at least 1 client option")
	}
}

func TestClient_IsCircuitOpen_Integration(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := NewTestClient(srv.Client(), srv.URL)

	if client.IsCircuitOpen("gmail") {
		t.Error("expected circuit to be closed for new client")
	}
}

func TestClient_WaitRate(t *testing.T) {
	srv := setupAPITestServer(t)
	defer srv.Close()
	client := NewTestClient(srv.Client(), srv.URL)

	err := client.WaitRate(context.Background(), "gmail")
	if err != nil {
		t.Fatalf("WaitRate: %v", err)
	}
}

// === ParseValuesJSON Tests ===

func TestParseValuesJSON_ValidInteg(t *testing.T) {
	vals, err := ParseValuesJSON(`[["a",1],["b",2]]`)
	if err != nil {
		t.Fatalf("ParseValuesJSON: %v", err)
	}
	if len(vals) != 2 {
		t.Errorf("expected 2 rows, got %d", len(vals))
	}
}

func TestParseValuesJSON_InvalidInteg(t *testing.T) {
	_, err := ParseValuesJSON(`not json`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseValuesJSON_EmptyInteg(t *testing.T) {
	vals, err := ParseValuesJSON(`[]`)
	if err != nil {
		t.Fatalf("ParseValuesJSON: %v", err)
	}
	if len(vals) != 0 {
		t.Errorf("expected 0 rows, got %d", len(vals))
	}
}

// === ValidateRow Tests ===

func TestValidateRow_Valid(t *testing.T) {
	schema := &SheetSchema{
		Columns: []ColumnRule{
			{Index: 0, Header: "Name", Type: "freetext", Required: true},
			{Index: 1, Header: "Status", Type: "enum", EnumValues: []string{"Todo", "Done"}},
		},
		ColumnCount: 2,
	}
	row := []interface{}{"Alice", "Todo"}
	result := ValidateRow(schema, row)
	if !result.Valid {
		t.Errorf("expected valid row, got issues: %v", result.Issues)
	}
}

func TestValidateRow_InvalidEnum(t *testing.T) {
	schema := &SheetSchema{
		Columns: []ColumnRule{
			{Index: 0, Header: "Name", Type: "freetext"},
			{Index: 1, Header: "Status", Type: "enum", EnumValues: []string{"Todo", "Done"}},
		},
		ColumnCount: 2,
	}
	row := []interface{}{"Alice", "Invalid"}
	result := ValidateRow(schema, row)
	if result.Valid {
		t.Error("expected invalid row for bad enum value")
	}
}
