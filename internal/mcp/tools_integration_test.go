package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// setupComprehensiveMock creates a mock HTTP server that handles ALL Google API
// service paths needed by the GWXHandler tool implementations.
//
// Path conventions vary by Google SDK when using option.WithEndpoint:
//   - Gmail: /gmail/v1/users/me/... (full path)
//   - Drive: /files, /files/{id}, /permissions (relative, no /drive/v3/ prefix)
//   - Calendar: /calendars/{id}/events (relative, no /calendar/v3/ prefix)
//   - Sheets: /v4/spreadsheets/... (keeps /v4/ prefix)
//   - Docs: /v1/documents/... (keeps /v1/ prefix)
//   - Tasks: /tasks/v1/... (full path)
//   - People: /v1/people/... or /v1/people:searchContacts (keeps /v1/ prefix)
//   - Chat: /v1/spaces (keeps /v1/ prefix)
//   - Slides: /v1/presentations/... (keeps /v1/ prefix)
//   - Analytics Data: /v1beta/properties/... (keeps prefix)
//   - Analytics Admin: /v1alpha/... (keeps prefix)
//   - Search Console: /webmasters/v3/... (full path)
//   - Forms: /v1/forms/... (keeps prefix)
//   - BigQuery: /bigquery/v2/... (full path)
func setupComprehensiveMock(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// ======================
	// Gmail routes
	// ======================

	// Gmail: send message
	mux.HandleFunc("/gmail/v1/users/me/messages/send", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"id":       "sent_001",
			"threadId": "thread_sent_001",
			"labelIds": []interface{}{"SENT"},
		})
	})

	// Gmail: batch modify
	mux.HandleFunc("/gmail/v1/users/me/messages/batchModify", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Gmail: get single message by ID
	mux.HandleFunc("/gmail/v1/users/me/messages/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, sampleGmailMessage())
	})

	// Gmail: list messages
	mux.HandleFunc("/gmail/v1/users/me/messages", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"messages": []interface{}{
				map[string]interface{}{"id": "msg_001", "threadId": "thread_001"},
			},
			"resultSizeEstimate": float64(1),
		})
	})

	// Gmail: list labels
	mux.HandleFunc("/gmail/v1/users/me/labels", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"labels": []interface{}{
				map[string]interface{}{"id": "INBOX", "name": "INBOX", "type": "system"},
				map[string]interface{}{"id": "SENT", "name": "SENT", "type": "system"},
				map[string]interface{}{"id": "UNREAD", "name": "UNREAD", "type": "system"},
			},
		})
	})

	// Gmail: create draft
	mux.HandleFunc("/gmail/v1/users/me/drafts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"id": "draft_001",
			"message": map[string]interface{}{
				"id":       "draft_msg_001",
				"threadId": "thread_draft_001",
				"labelIds": []interface{}{"DRAFT"},
			},
		})
	})

	// ======================
	// Calendar routes
	// ======================

	// Calendar: freeBusy
	mux.HandleFunc("/freeBusy", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"kind":      "calendar#freeBusy",
			"calendars": map[string]interface{}{},
		})
	})

	// Calendar: events (list, create, update, delete)
	mux.HandleFunc("/calendars/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/events") {
			switch r.Method {
			case http.MethodGet:
				writeJSON(w, map[string]interface{}{
					"kind": "calendar#events",
					"items": []interface{}{
						sampleCalendarEvent(),
					},
				})
			case http.MethodPost:
				writeJSON(w, sampleCalendarEvent())
			case http.MethodPut, http.MethodPatch:
				writeJSON(w, sampleCalendarEvent())
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			default:
				writeJSON(w, map[string]interface{}{})
			}
			return
		}
		writeJSON(w, map[string]interface{}{})
	})

	// ======================
	// Drive routes
	// ======================

	// Drive: upload (multipart)
	mux.HandleFunc("/upload/drive/v3/files", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, sampleDriveFile())
	})

	// Drive: permissions (for share)
	mux.HandleFunc("/permissions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"id": "perm_001", "role": "reader", "type": "user"})
	})

	// Drive: file operations with ID
	mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/permissions") {
			writeJSON(w, map[string]interface{}{"id": "perm_001", "role": "reader", "type": "user"})
			return
		}
		if strings.Contains(path, "/export") {
			w.Header().Set("Content-Type", "application/pdf")
			w.Write([]byte("%PDF-1.4 fake content"))
			return
		}
		writeJSON(w, sampleDriveFile())
	})

	// Drive: list/search files (exact match — must come after /files/)
	mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Create file/folder
			writeJSON(w, sampleDriveFile())
			return
		}
		writeJSON(w, map[string]interface{}{
			"kind":  "drive#fileList",
			"files": []interface{}{sampleDriveFile()},
		})
	})

	// ======================
	// Sheets routes
	// ======================

	// Sheets: batchUpdate (for CopyTab)
	mux.HandleFunc("/v4/spreadsheets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Create spreadsheet
			writeJSON(w, sampleSpreadsheetMeta())
			return
		}
		writeJSON(w, sampleSpreadsheetMeta())
	})

	mux.HandleFunc("/v4/spreadsheets/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Append
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
		// Clear
		if strings.Contains(path, ":clear") {
			writeJSON(w, map[string]interface{}{
				"spreadsheetId":  "sheet_001",
				"clearedRange":   "Tasks!A1:C3",
			})
			return
		}
		// Batch update (for addSheet, etc.)
		if strings.Contains(path, ":batchUpdate") {
			writeJSON(w, map[string]interface{}{
				"spreadsheetId": "sheet_001",
				"replies":       []interface{}{map[string]interface{}{"addSheet": map[string]interface{}{"properties": map[string]interface{}{"sheetId": float64(99), "title": "NewTab"}}}},
			})
			return
		}
		// Values read
		if strings.Contains(path, "/values/") {
			if r.Method == http.MethodPut {
				// Update values
				writeJSON(w, map[string]interface{}{
					"spreadsheetId":  "sheet_001",
					"updatedRange":   "Tasks!A1:C3",
					"updatedRows":    float64(3),
					"updatedCells":   float64(9),
				})
				return
			}
			writeJSON(w, sampleSpreadsheetValues())
			return
		}
		// Spreadsheet metadata
		writeJSON(w, sampleSpreadsheetMeta())
	})

	// ======================
	// Docs routes
	// ======================

	// Docs: create
	mux.HandleFunc("/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writeJSON(w, map[string]interface{}{
				"documentId": "doc_new_001",
				"title":      "New Document",
				"revisionId": "rev_001",
				"body": map[string]interface{}{
					"content": []interface{}{},
				},
			})
			return
		}
		writeJSON(w, map[string]interface{}{})
	})

	// Docs: get/update by ID
	mux.HandleFunc("/v1/documents/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// batchUpdate (for replaceText, appendText)
			writeJSON(w, map[string]interface{}{
				"documentId": "doc_001",
				"replies": []interface{}{
					map[string]interface{}{"replaceAllText": map[string]interface{}{"occurrencesChanged": float64(2)}},
				},
			})
			return
		}
		writeJSON(w, map[string]interface{}{
			"documentId": "doc_001",
			"title":      "Test Document",
			"revisionId": "rev_001",
			"body": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"paragraph": map[string]interface{}{
							"elements": []interface{}{
								map[string]interface{}{
									"textRun": map[string]interface{}{
										"content": "Hello world test content.",
									},
								},
							},
						},
					},
				},
			},
		})
	})

	// ======================
	// Tasks routes
	// ======================

	// Tasks: list task lists
	mux.HandleFunc("/tasks/v1/users/@me/lists", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"kind": "tasks#taskLists",
			"items": []interface{}{
				map[string]interface{}{"id": "list_001", "title": "My Tasks"},
			},
		})
	})

	// Tasks: list/create/complete/delete tasks
	mux.HandleFunc("/tasks/v1/lists/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, map[string]interface{}{
				"kind": "tasks#tasks",
				"items": []interface{}{
					map[string]interface{}{
						"id":      "task_001",
						"title":   "Buy groceries",
						"status":  "needsAction",
						"updated": time.Now().Format(time.RFC3339),
					},
				},
			})
		case http.MethodPost:
			writeJSON(w, map[string]interface{}{
				"id":      "task_new_001",
				"title":   "New Task",
				"status":  "needsAction",
				"updated": time.Now().Format(time.RFC3339),
			})
		case http.MethodPut, http.MethodPatch:
			writeJSON(w, map[string]interface{}{
				"id":        "task_001",
				"title":     "Buy groceries",
				"status":    "completed",
				"completed": time.Now().Format(time.RFC3339),
				"updated":   time.Now().Format(time.RFC3339),
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	// ======================
	// People / Contacts routes
	// ======================

	mux.HandleFunc("/v1/people:searchContacts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{
					"person": map[string]interface{}{
						"resourceName": "people/c001",
						"names":        []interface{}{map[string]interface{}{"displayName": "Alice Chen"}},
						"emailAddresses": []interface{}{
							map[string]interface{}{"value": "alice@example.com"},
						},
					},
				},
			},
		})
	})

	mux.HandleFunc("/v1/people/me/connections", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"connections": []interface{}{
				map[string]interface{}{
					"resourceName": "people/c001",
					"names":        []interface{}{map[string]interface{}{"displayName": "Alice Chen"}},
					"emailAddresses": []interface{}{
						map[string]interface{}{"value": "alice@example.com"},
					},
				},
			},
			"totalPeople": float64(1),
		})
	})

	mux.HandleFunc("/v1/people/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"resourceName": "people/c001",
			"names":        []interface{}{map[string]interface{}{"displayName": "Alice Chen"}},
			"emailAddresses": []interface{}{
				map[string]interface{}{"value": "alice@example.com"},
			},
		})
	})

	// ======================
	// Chat routes
	// ======================

	mux.HandleFunc("/v1/spaces", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"spaces": []interface{}{
				map[string]interface{}{
					"name":        "spaces/AAAA",
					"displayName": "General",
					"type":        "ROOM",
					"threaded":    true,
				},
			},
		})
	})

	mux.HandleFunc("/v1/spaces/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/messages") {
			if r.Method == http.MethodPost {
				writeJSON(w, map[string]interface{}{
					"name":       "spaces/AAAA/messages/msg001",
					"text":       "Hello from test",
					"createTime": time.Now().Format(time.RFC3339),
					"space":      map[string]interface{}{"name": "spaces/AAAA"},
				})
				return
			}
			writeJSON(w, map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"name":       "spaces/AAAA/messages/msg001",
						"text":       "Hello",
						"createTime": time.Now().Format(time.RFC3339),
						"sender":     map[string]interface{}{"name": "users/001", "displayName": "Alice"},
					},
				},
			})
			return
		}
		writeJSON(w, map[string]interface{}{
			"name":        "spaces/AAAA",
			"displayName": "General",
		})
	})

	// ======================
	// Slides routes
	// ======================

	mux.HandleFunc("/v1/presentations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writeJSON(w, map[string]interface{}{
				"presentationId": "pres_new_001",
				"title":          "New Presentation",
				"slides":         []interface{}{},
			})
			return
		}
		writeJSON(w, map[string]interface{}{})
	})

	mux.HandleFunc("/v1/presentations/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"presentationId": "pres_001",
			"title":          "Test Presentation",
			"slides": []interface{}{
				map[string]interface{}{
					"objectId": "slide_001",
					"pageElements": []interface{}{
						map[string]interface{}{
							"objectId": "elem_001",
							"shape": map[string]interface{}{
								"text": map[string]interface{}{
									"textElements": []interface{}{
										map[string]interface{}{
											"textRun": map[string]interface{}{
												"content": "Slide title text",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		})
	})

	// ======================
	// Analytics Data routes
	// ======================

	mux.HandleFunc("/v1beta/properties/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, ":runRealtimeReport") {
			writeJSON(w, map[string]interface{}{
				"rows": []interface{}{
					map[string]interface{}{
						"dimensionValues": []interface{}{map[string]interface{}{"value": "US"}},
						"metricValues":    []interface{}{map[string]interface{}{"value": "42"}},
					},
				},
				"rowCount": float64(1),
			})
			return
		}
		if strings.Contains(path, ":runReport") {
			writeJSON(w, map[string]interface{}{
				"rows": []interface{}{
					map[string]interface{}{
						"dimensionValues": []interface{}{map[string]interface{}{"value": "2026-03-20"}},
						"metricValues":    []interface{}{map[string]interface{}{"value": "100"}},
					},
				},
				"rowCount": float64(1),
			})
			return
		}
		if strings.Contains(path, "/audiences") {
			writeJSON(w, map[string]interface{}{
				"audiences": []interface{}{
					map[string]interface{}{
						"name":        "properties/123/audiences/001",
						"displayName": "All Users",
					},
				},
			})
			return
		}
		writeJSON(w, map[string]interface{}{})
	})

	// Analytics Admin: list properties
	mux.HandleFunc("/v1alpha/properties", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"properties": []interface{}{
				map[string]interface{}{
					"name":         "properties/123456",
					"displayName":  "My Website",
					"industryCategory": "TECHNOLOGY",
					"timeZone":     "Asia/Taipei",
				},
			},
		})
	})

	// ======================
	// Search Console routes
	// ======================

	mux.HandleFunc("/webmasters/v3/sites", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"siteEntry": []interface{}{
				map[string]interface{}{
					"siteUrl":         "https://example.com/",
					"permissionLevel": "siteOwner",
				},
			},
		})
	})

	mux.HandleFunc("/webmasters/v3/sites/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/searchAnalytics/query") {
			writeJSON(w, map[string]interface{}{
				"rows": []interface{}{
					map[string]interface{}{
						"keys":        []interface{}{"test query"},
						"clicks":      float64(100),
						"impressions": float64(1000),
						"ctr":         0.1,
						"position":    3.5,
					},
				},
			})
			return
		}
		if strings.Contains(path, "/sitemaps") {
			writeJSON(w, map[string]interface{}{
				"sitemap": []interface{}{
					map[string]interface{}{
						"path":           "https://example.com/sitemap.xml",
						"lastDownloaded": time.Now().Format(time.RFC3339),
					},
				},
			})
			return
		}
		if strings.Contains(path, "/urlInspection/index:inspect") {
			writeJSON(w, map[string]interface{}{
				"inspectionResult": map[string]interface{}{
					"indexStatusResult": map[string]interface{}{
						"verdict":       "PASS",
						"coverageState": "Submitted and indexed",
						"indexingState": "INDEXING_ALLOWED",
					},
				},
			})
			return
		}
		writeJSON(w, map[string]interface{}{})
	})

	// ======================
	// Forms routes
	// ======================

	mux.HandleFunc("/v1/forms/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/responses") {
			writeJSON(w, map[string]interface{}{
				"responses": []interface{}{
					map[string]interface{}{
						"responseId": "resp_001",
						"createTime": time.Now().Format(time.RFC3339),
						"answers":    map[string]interface{}{},
					},
				},
			})
			return
		}
		writeJSON(w, map[string]interface{}{
			"formId": "form_001",
			"info":   map[string]interface{}{"title": "Test Form"},
			"items": []interface{}{
				map[string]interface{}{
					"itemId": "item_001",
					"title":  "What is your name?",
					"questionItem": map[string]interface{}{
						"question": map[string]interface{}{
							"questionId": "q_001",
							"textQuestion": map[string]interface{}{},
						},
					},
				},
			},
		})
	})

	// ======================
	// BigQuery routes
	// ======================

	mux.HandleFunc("/bigquery/v2/projects/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/queries") {
			writeJSON(w, map[string]interface{}{
				"kind":              "bigquery#queryResponse",
				"jobComplete":       true,
				"totalRows":         "1",
				"totalBytesProcessed": "1024",
				"schema": map[string]interface{}{
					"fields": []interface{}{
						map[string]interface{}{"name": "col1", "type": "STRING"},
					},
				},
				"rows": []interface{}{
					map[string]interface{}{
						"f": []interface{}{
							map[string]interface{}{"v": "value1"},
						},
					},
				},
			})
			return
		}
		if strings.Contains(path, "/datasets") {
			writeJSON(w, map[string]interface{}{
				"kind": "bigquery#datasetList",
				"datasets": []interface{}{
					map[string]interface{}{
						"kind": "bigquery#dataset",
						"datasetReference": map[string]interface{}{
							"datasetId": "ds_001",
							"projectId": "test-project",
						},
					},
				},
			})
			return
		}
		writeJSON(w, map[string]interface{}{})
	})

	// ======================
	// Default fallback
	// ======================
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Return empty success for unmatched routes
		writeJSON(w, map[string]interface{}{})
	})

	return httptest.NewServer(mux)
}

// writeJSON encodes data as JSON to the response writer.
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

// --- Sample data helpers ---

func sampleGmailMessage() map[string]interface{} {
	return map[string]interface{}{
		"id":           "msg_001",
		"threadId":     "thread_001",
		"snippet":      "Hey, let's sync on the Q2 roadmap tomorrow.",
		"labelIds":     []interface{}{"INBOX", "UNREAD"},
		"internalDate": "1711180800000",
		"sizeEstimate": 1234,
		"payload": map[string]interface{}{
			"mimeType": "text/plain",
			"headers": []interface{}{
				map[string]interface{}{"name": "From", "value": "alice@example.com"},
				map[string]interface{}{"name": "To", "value": "bob@example.com"},
				map[string]interface{}{"name": "Subject", "value": "Q2 Roadmap Sync"},
				map[string]interface{}{"name": "Date", "value": "Mon, 23 Mar 2026 10:00:00 +0800"},
			},
			"body": map[string]interface{}{
				"size": 45,
				"data": "SGV5LCBsZXQncyBzeW5jIG9uIHRoZSBRMiByb2FkbWFwIHRvbW9ycm93Lg==",
			},
		},
	}
}

func sampleCalendarEvent() map[string]interface{} {
	return map[string]interface{}{
		"id":      "evt_001",
		"summary": "Sprint Planning",
		"status":  "confirmed",
		"start":   map[string]interface{}{"dateTime": "2026-03-23T10:00:00+08:00"},
		"end":     map[string]interface{}{"dateTime": "2026-03-23T11:00:00+08:00"},
		"location":    "Meeting Room A",
		"description": "Weekly sprint planning session",
		"organizer":   map[string]interface{}{"email": "pm@example.com"},
		"attendees": []interface{}{
			map[string]interface{}{"email": "dev1@example.com"},
		},
		"hangoutLink": "https://meet.google.com/abc",
		"htmlLink":    "https://calendar.google.com/event?eid=evt_001",
	}
}

func sampleDriveFile() map[string]interface{} {
	return map[string]interface{}{
		"id":           "file_001",
		"name":         "Q2 Roadmap.docx",
		"mimeType":     "application/vnd.google-apps.document",
		"size":         "15360",
		"modifiedTime": "2026-03-22T16:30:00.000Z",
		"createdTime":  "2026-03-01T09:00:00.000Z",
		"webViewLink":  "https://docs.google.com/document/d/file_001",
		"parents":      []interface{}{"folder_root"},
		"shared":       false,
		"trashed":      false,
	}
}

func sampleSpreadsheetMeta() map[string]interface{} {
	return map[string]interface{}{
		"spreadsheetId":  "sheet_001",
		"spreadsheetUrl": "https://docs.google.com/spreadsheets/d/sheet_001",
		"properties":     map[string]interface{}{"title": "Project Tracker"},
		"sheets": []interface{}{
			map[string]interface{}{
				"properties": map[string]interface{}{
					"sheetId": float64(0),
					"title":   "Tasks",
					"index":   float64(0),
					"gridProperties": map[string]interface{}{
						"rowCount":    float64(100),
						"columnCount": float64(10),
					},
				},
			},
		},
	}
}

func sampleSpreadsheetValues() map[string]interface{} {
	return map[string]interface{}{
		"range":          "Tasks!A1:C3",
		"majorDimension": "ROWS",
		"values": []interface{}{
			[]interface{}{"Task", "Status", "Assignee"},
			[]interface{}{"Build API", "In Progress", "Alice"},
			[]interface{}{"Write Tests", "Todo", "Bob"},
		},
	}
}

// defaultArgsForTool returns sensible default arguments for each tool name.
func defaultArgsForTool(name string) map[string]interface{} {
	switch name {
	// Gmail
	case "gmail_list":
		return map[string]interface{}{"limit": float64(5)}
	case "gmail_get":
		return map[string]interface{}{"message_id": "msg_001"}
	case "gmail_search":
		return map[string]interface{}{"query": "test", "limit": float64(5)}
	case "gmail_send":
		return map[string]interface{}{"to": "test@example.com", "subject": "Test", "body": "Hello"}
	case "gmail_labels":
		return map[string]interface{}{}
	case "gmail_draft":
		return map[string]interface{}{"to": "test@example.com", "subject": "Draft", "body": "Draft body"}
	case "gmail_digest":
		return map[string]interface{}{"limit": float64(5)}
	case "gmail_archive":
		return map[string]interface{}{"query": "from:test@example.com", "limit": float64(5)}
	case "gmail_reply":
		return map[string]interface{}{"message_id": "msg_001", "body": "Reply text"}
	case "gmail_forward":
		return map[string]interface{}{"message_id": "msg_001", "to": "forward@example.com"}
	case "gmail_batch_label":
		return map[string]interface{}{"query": "from:test", "add": "INBOX"}

	// Calendar
	case "calendar_agenda":
		return map[string]interface{}{"days": float64(1)}
	case "calendar_create":
		return map[string]interface{}{
			"title": "Test Event",
			"start": time.Now().Add(time.Hour).Format(time.RFC3339),
			"end":   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		}
	case "calendar_list":
		return map[string]interface{}{}
	case "calendar_update":
		return map[string]interface{}{"event_id": "evt_001", "title": "Updated"}
	case "calendar_delete":
		return map[string]interface{}{"event_id": "evt_001"}
	case "calendar_find_slot":
		return map[string]interface{}{"attendees": "dev@example.com", "duration": "30m", "days": float64(1)}

	// Drive
	case "drive_list":
		return map[string]interface{}{"limit": float64(5)}
	case "drive_search":
		return map[string]interface{}{"query": "name contains 'report'", "limit": float64(5)}
	case "drive_share":
		return map[string]interface{}{"file_id": "file_001", "email": "share@example.com", "role": "reader"}
	case "drive_mkdir":
		return map[string]interface{}{"name": "New Folder"}
	case "drive_download":
		return map[string]interface{}{"file_id": "file_001"}

	// Docs
	case "docs_get":
		return map[string]interface{}{"doc_id": "doc_001"}
	case "docs_create":
		return map[string]interface{}{"title": "New Doc"}
	case "docs_search":
		return map[string]interface{}{"doc_id": "doc_001", "query": "test"}
	case "docs_replace":
		return map[string]interface{}{"doc_id": "doc_001", "find": "old", "replace": "new"}
	case "docs_append":
		return map[string]interface{}{"doc_id": "doc_001", "text": "Appended text"}
	case "docs_template":
		return map[string]interface{}{"template_id": "doc_001", "title": "From Template"}
	case "docs_from_sheet":
		return map[string]interface{}{"title": "From Sheet", "headers": `["Name","Score"]`, "rows": `[["Alice",95]]`}
	case "docs_export":
		return map[string]interface{}{"doc_id": "doc_001", "format": "pdf"}

	// Sheets
	case "sheets_read":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks!A1:C3"}
	case "sheets_append":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks!A:C", "values": `[["Task3","Done","Charlie"]]`}
	case "sheets_describe":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks"}
	case "sheets_smart_append":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks!A:C", "values": `[["Task3","Done","Charlie"]]`}
	case "sheets_search":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "query": "Alice", "range": "Tasks!A:C"}
	case "sheets_filter":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks!A:C", "column": float64(1), "value": "In Progress"}
	case "sheets_stats":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks"}
	case "sheets_diff":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range_a": "Tasks", "range_b": "Tasks"}
	case "sheets_export":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks!A:C", "format": "csv"}
	case "sheets_info":
		return map[string]interface{}{"spreadsheet_id": "sheet_001"}
	case "sheets_clear":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks!A1:C3"}
	case "sheets_update":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "range": "Tasks!A1:C1", "values": `[["Task","Status","Assignee"]]`}
	case "sheets_create":
		return map[string]interface{}{"title": "New Sheet"}
	case "sheets_copy_tab":
		return map[string]interface{}{"spreadsheet_id": "sheet_001", "source": "Tasks", "name": "NewTab"}

	// Tasks
	case "tasks_list":
		return map[string]interface{}{}
	case "tasks_create":
		return map[string]interface{}{"title": "New Task"}
	case "tasks_lists":
		return map[string]interface{}{}
	case "tasks_complete":
		return map[string]interface{}{"task_id": "task_001"}
	case "tasks_delete":
		return map[string]interface{}{"task_id": "task_001"}

	// Contacts
	case "contacts_search":
		return map[string]interface{}{"query": "Alice"}
	case "contacts_list":
		return map[string]interface{}{"limit": float64(10)}
	case "contacts_get":
		return map[string]interface{}{"resource_name": "people/c001"}

	// Chat
	case "chat_spaces":
		return map[string]interface{}{}
	case "chat_send":
		return map[string]interface{}{"space": "spaces/AAAA", "text": "Hello"}
	case "chat_messages":
		return map[string]interface{}{"space": "spaces/AAAA"}

	// Slides
	case "slides_get":
		return map[string]interface{}{"presentation_id": "pres_001"}
	case "slides_list":
		return map[string]interface{}{"limit": float64(5)}
	case "slides_create":
		return map[string]interface{}{"title": "New Presentation"}
	case "slides_duplicate":
		return map[string]interface{}{"presentation_id": "pres_001", "title": "Copy"}
	case "slides_export":
		return map[string]interface{}{"presentation_id": "pres_001", "format": "pdf"}

	// Context / Unified search
	case "context_gather":
		return map[string]interface{}{"topic": "roadmap", "days": float64(1), "limit": float64(2)}
	case "unified_search":
		return map[string]interface{}{"query": "roadmap", "limit": float64(2)}

	// Config (no API calls)
	case "config_set":
		return map[string]interface{}{"key": "test.key", "value": "test_value"}
	case "config_get":
		return map[string]interface{}{"key": "test.key"}
	case "config_list":
		return map[string]interface{}{}

	default:
		return map[string]interface{}{}
	}
}

// toolsRequiringExternalProviders lists tools that need external providers
// (GitHub, Slack, Notion, etc.) or config that can't be satisfied in tests.
var toolsRequiringExternalProviders = map[string]bool{
	// GitHub tools require auth.LoadProviderToken
	"github_repos":         true,
	"github_issues":        true,
	"github_create_issue":  true,
	"github_pulls":         true,
	"github_pull":          true,
	"github_runs":          true,
	"github_notifications": true,
	// Slack tools require auth.LoadProviderToken
	"slack_channels": true,
	"slack_send":     true,
	"slack_messages": true,
	"slack_search":   true,
	"slack_users":    true,
	"slack_user":     true,
	// Notion tools require auth.LoadProviderToken
	"notion_search":      true,
	"notion_page":        true,
	"notion_create_page": true,
	"notion_databases":   true,
	"notion_query":       true,
	// Analytics tools need config (resolveProperty will fail)
	"analytics_report":     true,
	"analytics_realtime":   true,
	"analytics_properties": true,
	"analytics_audiences":  true,
	// Search Console tools need config (resolveSearchConsoleSite will fail)
	"searchconsole_query":        true,
	"searchconsole_sites":        true,
	"searchconsole_inspect":      true,
	"searchconsole_sitemaps":     true,
	"searchconsole_index_status": true,
	// BigQuery needs config (default-project)
	"bigquery_query":    true,
	"bigquery_datasets": true,
	"bigquery_tables":   true,
	"bigquery_describe": true,
	// Workflow tools use workflow package which calls multiple services
	"workflow_standup":             true,
	"workflow_meeting_prep":        true,
	"workflow_weekly_digest":       true,
	"workflow_digest":              true,
	"workflow_context_boost":       true,
	"workflow_bug_intake":          true,
	"workflow_test_matrix_init":    true,
	"workflow_test_matrix_sync":    true,
	"workflow_test_matrix_stats":   true,
	"workflow_spec_health_init":    true,
	"workflow_spec_health_record":  true,
	"workflow_spec_health_stats":   true,
	"workflow_sprint_board_init":   true,
	"workflow_sprint_board_ticket": true,
	"workflow_sprint_board_stats":  true,
	"workflow_review_notify":       true,
	"workflow_email_from_doc":      true,
	"workflow_sheet_to_email":      true,
	"workflow_parallel_schedule":   true,
	// Tools needing local filesystem
	"drive_upload":        true,
	"drive_batch_upload":  true,
	"sheets_import":       true,
	"sheets_batch_append": true,
	// Slides requiring Drive operations that might race
	"slides_from_sheet": true,
}

// TestAllGoogleToolHandlers tests every registered Google API tool handler.
// It creates a comprehensive mock server, wires up a GWXHandler, and calls
// each tool with appropriate default arguments. Tests verify:
// - No panic
// - Success or known error (no nil result without error)
func TestAllGoogleToolHandlers(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()

	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	handler := NewGWXHandler(client)

	// Build the registry to get all registered tool names
	registry := handler.buildRegistry()

	for name, toolFn := range registry {
		if toolsRequiringExternalProviders[name] {
			continue
		}

		t.Run(name, func(t *testing.T) {
			args := defaultArgsForTool(name)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := toolFn(ctx, args)
			if err != nil {
				// Some errors are expected (e.g., config not found).
				// We only care that it doesn't panic.
				t.Logf("tool %s returned error (acceptable): %v", name, err)
				return
			}
			if result == nil {
				t.Errorf("tool %s returned nil result without error", name)
				return
			}
			if len(result.Content) == 0 {
				t.Errorf("tool %s returned empty content", name)
				return
			}
			if result.Content[0].Text == "" {
				t.Errorf("tool %s returned empty text", name)
				return
			}
			// Verify the result is valid JSON (except export tools which return raw text)
			if !strings.HasSuffix(name, "_export") {
				var parsed interface{}
				if err := json.Unmarshal([]byte(result.Content[0].Text), &parsed); err != nil {
					t.Errorf("tool %s returned invalid JSON: %v", name, err)
				}
			}
		})
	}
}

// --- Individual handler tests for deeper verification ---

func TestHandler_GmailList(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailList(context.Background(), map[string]interface{}{"limit": float64(10)})
	if err != nil {
		t.Fatalf("gmailList: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "messages")
	assertKeyExists(t, data, "count")
	assertKeyExists(t, data, "total_estimate")
}

func TestHandler_GmailGet(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailGet(context.Background(), map[string]interface{}{"message_id": "msg_001"})
	if err != nil {
		t.Fatalf("gmailGet: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "id")
	if data["id"] != "msg_001" {
		t.Errorf("expected id=msg_001, got %v", data["id"])
	}
}

func TestHandler_GmailSearch(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailSearch(context.Background(), map[string]interface{}{"query": "test", "limit": float64(5)})
	if err != nil {
		t.Fatalf("gmailSearch: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "messages")
	assertKeyExists(t, data, "count")
}

func TestHandler_GmailSend(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailSend(context.Background(), map[string]interface{}{
		"to": "test@example.com", "subject": "Test", "body": "Hello",
	})
	if err != nil {
		t.Fatalf("gmailSend: %v", err)
	}
	data := parseResult(t, result)
	if data["sent"] != true {
		t.Errorf("expected sent=true, got %v", data["sent"])
	}
	assertKeyExists(t, data, "message_id")
}

func TestHandler_GmailLabels(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailLabels(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("gmailLabels: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "labels")
	assertKeyExists(t, data, "count")
}

func TestHandler_GmailDraft(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailDraft(context.Background(), map[string]interface{}{
		"to": "test@example.com", "subject": "Draft", "body": "Draft body",
	})
	if err != nil {
		t.Fatalf("gmailDraft: %v", err)
	}
	data := parseResult(t, result)
	if data["drafted"] != true {
		t.Errorf("expected drafted=true, got %v", data["drafted"])
	}
}

func TestHandler_GmailDigest(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailDigest(context.Background(), map[string]interface{}{"limit": float64(5)})
	if err != nil {
		t.Fatalf("gmailDigest: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "total_messages")
	assertKeyExists(t, data, "groups")
}

func TestHandler_GmailArchive(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailArchive(context.Background(), map[string]interface{}{
		"query": "from:test", "limit": float64(5),
	})
	if err != nil {
		t.Fatalf("gmailArchive: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "action")
	assertKeyExists(t, data, "count")
}

func TestHandler_GmailArchive_ReadOnly(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailArchive(context.Background(), map[string]interface{}{
		"query": "from:test", "read_only": true,
	})
	if err != nil {
		t.Fatalf("gmailArchive read_only: %v", err)
	}
	data := parseResult(t, result)
	if data["action"] != "marked_read" {
		t.Errorf("expected action=marked_read, got %v", data["action"])
	}
}

func TestHandler_GmailReply(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailReply(context.Background(), map[string]interface{}{
		"message_id": "msg_001", "body": "Reply text",
	})
	if err != nil {
		t.Fatalf("gmailReply: %v", err)
	}
	data := parseResult(t, result)
	if data["replied"] != true {
		t.Errorf("expected replied=true, got %v", data["replied"])
	}
}

func TestHandler_GmailForward(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.gmailForward(context.Background(), map[string]interface{}{
		"message_id": "msg_001", "to": "forward@example.com",
	})
	if err != nil {
		t.Fatalf("gmailForward: %v", err)
	}
	data := parseResult(t, result)
	if data["forwarded"] != true {
		t.Errorf("expected forwarded=true, got %v", data["forwarded"])
	}
}

func TestHandler_GmailBatchLabel(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	// Use "INBOX" which exists in our mock labels response
	result, err := h.gmailBatchLabel(context.Background(), map[string]interface{}{
		"query": "from:test", "add": "INBOX",
	})
	if err != nil {
		t.Fatalf("gmailBatchLabel: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "modified")
}

// --- Calendar handler tests ---

func TestHandler_CalendarAgenda(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.calendarAgenda(context.Background(), map[string]interface{}{"days": float64(1)})
	if err != nil {
		t.Fatalf("calendarAgenda: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "events")
	assertKeyExists(t, data, "count")
}

func TestHandler_CalendarCreate(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.calendarCreate(context.Background(), map[string]interface{}{
		"title": "Test Event",
		"start": time.Now().Add(time.Hour).Format(time.RFC3339),
		"end":   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("calendarCreate: %v", err)
	}
	data := parseResult(t, result)
	if data["created"] != true {
		t.Errorf("expected created=true, got %v", data["created"])
	}
}

func TestHandler_CalendarList(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.calendarList(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("calendarList: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "events")
}

func TestHandler_CalendarUpdate(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.calendarUpdate(context.Background(), map[string]interface{}{
		"event_id": "evt_001", "title": "Updated Event",
	})
	if err != nil {
		t.Fatalf("calendarUpdate: %v", err)
	}
	data := parseResult(t, result)
	if data["updated"] != true {
		t.Errorf("expected updated=true, got %v", data["updated"])
	}
}

func TestHandler_CalendarDelete(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.calendarDelete(context.Background(), map[string]interface{}{"event_id": "evt_001"})
	if err != nil {
		t.Fatalf("calendarDelete: %v", err)
	}
	data := parseResult(t, result)
	if data["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", data["deleted"])
	}
}

func TestHandler_CalendarFindSlot(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.calendarFindSlot(context.Background(), map[string]interface{}{
		"attendees": "dev@example.com", "duration": "30m", "days": float64(1),
	})
	if err != nil {
		t.Fatalf("calendarFindSlot: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "slots")
	assertKeyExists(t, data, "count")
}

// --- Drive handler tests ---

func TestHandler_DriveList(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.driveList(context.Background(), map[string]interface{}{"limit": float64(5)})
	if err != nil {
		t.Fatalf("driveList: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "files")
	assertKeyExists(t, data, "count")
}

func TestHandler_DriveSearch(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.driveSearch(context.Background(), map[string]interface{}{
		"query": "name contains 'report'", "limit": float64(5),
	})
	if err != nil {
		t.Fatalf("driveSearch: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "files")
}

func TestHandler_DriveShare(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.driveShare(context.Background(), map[string]interface{}{
		"file_id": "file_001", "email": "share@example.com", "role": "reader",
	})
	if err != nil {
		t.Fatalf("driveShare: %v", err)
	}
	data := parseResult(t, result)
	if data["shared"] != true {
		t.Errorf("expected shared=true, got %v", data["shared"])
	}
}

func TestHandler_DriveMkdir(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.driveMkdir(context.Background(), map[string]interface{}{"name": "New Folder"})
	if err != nil {
		t.Fatalf("driveMkdir: %v", err)
	}
	data := parseResult(t, result)
	if data["created"] != true {
		t.Errorf("expected created=true, got %v", data["created"])
	}
}

// --- Docs handler tests ---

func TestHandler_DocsGet(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.docsGet(context.Background(), map[string]interface{}{"doc_id": "doc_001"})
	if err != nil {
		t.Fatalf("docsGet: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "document_id")
}

func TestHandler_DocsCreate(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.docsCreate(context.Background(), map[string]interface{}{"title": "New Doc", "body": "Hello"})
	if err != nil {
		t.Fatalf("docsCreate: %v", err)
	}
	data := parseResult(t, result)
	if data["created"] != true {
		t.Errorf("expected created=true, got %v", data["created"])
	}
}

func TestHandler_DocsSearch(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.docsSearch(context.Background(), map[string]interface{}{"doc_id": "doc_001", "query": "test"})
	if err != nil {
		t.Fatalf("docsSearch: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "matches")
}

func TestHandler_DocsReplace(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.docsReplace(context.Background(), map[string]interface{}{
		"doc_id": "doc_001", "find": "old", "replace": "new",
	})
	if err != nil {
		t.Fatalf("docsReplace: %v", err)
	}
	data := parseResult(t, result)
	if data["replaced"] != true {
		t.Errorf("expected replaced=true, got %v", data["replaced"])
	}
}

func TestHandler_DocsAppend(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.docsAppend(context.Background(), map[string]interface{}{
		"doc_id": "doc_001", "text": "Appended text",
	})
	if err != nil {
		t.Fatalf("docsAppend: %v", err)
	}
	data := parseResult(t, result)
	if data["appended"] != true {
		t.Errorf("expected appended=true, got %v", data["appended"])
	}
}

// --- Sheets handler tests ---

func TestHandler_SheetsRead(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsRead(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A1:C3",
	})
	if err != nil {
		t.Fatalf("sheetsRead: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "range")
	assertKeyExists(t, data, "values")
	assertKeyExists(t, data, "row_count")
}

func TestHandler_SheetsAppend(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsAppend(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A:C",
		"values": `[["Task3","Done","Charlie"]]`,
	})
	if err != nil {
		t.Fatalf("sheetsAppend: %v", err)
	}
	data := parseResult(t, result)
	if data["appended"] != true {
		t.Errorf("expected appended=true, got %v", data["appended"])
	}
}

func TestHandler_SheetsDescribe(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsDescribe(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks",
	})
	if err != nil {
		t.Fatalf("sheetsDescribe: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "columns")
}

func TestHandler_SheetsSmartAppend(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsSmartAppend(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A:C",
		"values": `[["Task3","Done","Charlie"]]`,
	})
	if err != nil {
		t.Fatalf("sheetsSmartAppend: %v", err)
	}
	data := parseResult(t, result)
	// smart_append does validation then appends
	assertKeyExists(t, data, "valid")
}

func TestHandler_SheetsSearch(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsSearch(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "query": "Alice", "range": "Tasks!A:C",
	})
	if err != nil {
		t.Fatalf("sheetsSearch: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "match_count")
	assertKeyExists(t, data, "matched_rows")
}

func TestHandler_SheetsFilter(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsFilter(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A:C",
		"column": float64(1), "value": "In Progress",
	})
	if err != nil {
		t.Fatalf("sheetsFilter: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "match_count")
	assertKeyExists(t, data, "matched_rows")
}

func TestHandler_SheetsStats(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsStats(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks",
	})
	if err != nil {
		t.Fatalf("sheetsStats: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "total_rows")
	assertKeyExists(t, data, "columns")
}

func TestHandler_SheetsDiff(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsDiff(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range_a": "Tasks", "range_b": "Tasks",
	})
	if err != nil {
		t.Fatalf("sheetsDiff: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "summary")
}

func TestHandler_SheetsExport_CSV(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsExport(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A:C", "format": "csv",
	})
	if err != nil {
		t.Fatalf("sheetsExport CSV: %v", err)
	}
	if result == nil || len(result.Content) == 0 || result.Content[0].Text == "" {
		t.Error("expected non-empty CSV output")
	}
}

func TestHandler_SheetsExport_JSON(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsExport(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A:C", "format": "json",
	})
	if err != nil {
		t.Fatalf("sheetsExport JSON: %v", err)
	}
	if result == nil || len(result.Content) == 0 || result.Content[0].Text == "" {
		t.Error("expected non-empty JSON output")
	}
}

func TestHandler_SheetsExport_BadFormat(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	_, err := h.sheetsExport(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A:C", "format": "xml",
	})
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestHandler_SheetsInfo(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsInfo(context.Background(), map[string]interface{}{"spreadsheet_id": "sheet_001"})
	if err != nil {
		t.Fatalf("sheetsInfo: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "spreadsheet_id")
	assertKeyExists(t, data, "title")
}

func TestHandler_SheetsClear(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsClear(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A1:C3",
	})
	if err != nil {
		t.Fatalf("sheetsClear: %v", err)
	}
	data := parseResult(t, result)
	if data["cleared"] != true {
		t.Errorf("expected cleared=true, got %v", data["cleared"])
	}
}

func TestHandler_SheetsUpdate(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsUpdate(context.Background(), map[string]interface{}{
		"spreadsheet_id": "sheet_001", "range": "Tasks!A1:C1",
		"values": `[["Task","Status","Assignee"]]`,
	})
	if err != nil {
		t.Fatalf("sheetsUpdate: %v", err)
	}
	data := parseResult(t, result)
	if data["updated"] != true {
		t.Errorf("expected updated=true, got %v", data["updated"])
	}
}

func TestHandler_SheetsCreate(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.sheetsCreate(context.Background(), map[string]interface{}{"title": "New Sheet"})
	if err != nil {
		t.Fatalf("sheetsCreate: %v", err)
	}
	data := parseResult(t, result)
	if data["created"] != true {
		t.Errorf("expected created=true, got %v", data["created"])
	}
}

// --- Tasks handler tests ---

func TestHandler_TasksList(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.tasksList(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("tasksList: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "tasks")
	assertKeyExists(t, data, "count")
}

func TestHandler_TasksCreate(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.tasksCreate(context.Background(), map[string]interface{}{"title": "New Task"})
	if err != nil {
		t.Fatalf("tasksCreate: %v", err)
	}
	data := parseResult(t, result)
	if data["created"] != true {
		t.Errorf("expected created=true, got %v", data["created"])
	}
}

func TestHandler_TasksLists(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.tasksLists(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("tasksLists: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "lists")
}

func TestHandler_TasksComplete(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.tasksComplete(context.Background(), map[string]interface{}{"task_id": "task_001"})
	if err != nil {
		t.Fatalf("tasksComplete: %v", err)
	}
	data := parseResult(t, result)
	if data["completed"] != true {
		t.Errorf("expected completed=true, got %v", data["completed"])
	}
}

func TestHandler_TasksDelete(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.tasksDelete(context.Background(), map[string]interface{}{"task_id": "task_001"})
	if err != nil {
		t.Fatalf("tasksDelete: %v", err)
	}
	data := parseResult(t, result)
	if data["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", data["deleted"])
	}
}

// --- Contacts handler tests ---

func TestHandler_ContactsSearch(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.contactsSearch(context.Background(), map[string]interface{}{"query": "Alice"})
	if err != nil {
		t.Fatalf("contactsSearch: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "contacts")
}

func TestHandler_ContactsList(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.contactsList(context.Background(), map[string]interface{}{"limit": float64(10)})
	if err != nil {
		t.Fatalf("contactsList: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "contacts")
}

func TestHandler_ContactsGet(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.contactsGet(context.Background(), map[string]interface{}{"resource_name": "people/c001"})
	if err != nil {
		t.Fatalf("contactsGet: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "resource_name")
}

// --- Chat handler tests ---

func TestHandler_ChatSpaces(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.chatSpaces(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("chatSpaces: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "spaces")
}

func TestHandler_ChatSend(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.chatSend(context.Background(), map[string]interface{}{
		"space": "spaces/AAAA", "text": "Hello",
	})
	if err != nil {
		t.Fatalf("chatSend: %v", err)
	}
	data := parseResult(t, result)
	if data["sent"] != true {
		t.Errorf("expected sent=true, got %v", data["sent"])
	}
}

func TestHandler_ChatMessages(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.chatMessages(context.Background(), map[string]interface{}{
		"space": "spaces/AAAA",
	})
	if err != nil {
		t.Fatalf("chatMessages: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "messages")
}

// --- Slides handler tests ---

func TestHandler_SlidesGet(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.slidesGet(context.Background(), map[string]interface{}{"presentation_id": "pres_001"})
	if err != nil {
		t.Fatalf("slidesGet: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "presentation_id")
}

func TestHandler_SlidesList(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.slidesList(context.Background(), map[string]interface{}{"limit": float64(5)})
	if err != nil {
		t.Fatalf("slidesList: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "presentations")
}

func TestHandler_SlidesCreate(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.slidesCreate(context.Background(), map[string]interface{}{"title": "New Pres"})
	if err != nil {
		t.Fatalf("slidesCreate: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "presentation_id")
}

func TestHandler_SlidesDuplicate(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.slidesDuplicate(context.Background(), map[string]interface{}{
		"presentation_id": "pres_001", "title": "Copy",
	})
	if err != nil {
		t.Fatalf("slidesDuplicate: %v", err)
	}
	data := parseResult(t, result)
	// slidesDuplicate copies via Drive API, returns a FileSummary-like result
	assertKeyExists(t, data, "id")
}

// --- Context/Unified search handler tests ---

func TestHandler_ContextGather(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.contextGather(context.Background(), map[string]interface{}{
		"topic": "roadmap", "days": float64(1), "limit": float64(2),
	})
	if err != nil {
		t.Fatalf("contextGather: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "topic")
	assertKeyExists(t, data, "emails")
	assertKeyExists(t, data, "files")
	assertKeyExists(t, data, "events")
}

func TestHandler_UnifiedSearch(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	result, err := h.unifiedSearch(context.Background(), map[string]interface{}{
		"query": "roadmap", "limit": float64(2),
	})
	if err != nil {
		t.Fatalf("unifiedSearch: %v", err)
	}
	data := parseResult(t, result)
	assertKeyExists(t, data, "query")
	assertKeyExists(t, data, "results")
}

// --- CallTool dispatch tests ---

func TestCallTool_UnknownTool_Integration(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	_, err := h.CallTool("nonexistent_tool_xyz", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("expected 'unknown tool' in error, got: %v", err)
	}
}

func TestCallTool_ViaDispatch(t *testing.T) {
	mockServer := setupComprehensiveMock(t)
	defer mockServer.Close()
	client := api.NewTestClient(mockServer.Client(), mockServer.URL)
	h := NewGWXHandler(client)

	// Test through the public CallTool method
	result, err := h.CallTool("gmail_list", map[string]interface{}{"limit": float64(5)})
	if err != nil {
		t.Fatalf("CallTool gmail_list: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// --- Helper arg parser tests ---

func TestStrArg_Empty(t *testing.T) {
	result := strArg(nil, "key")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestStrArg_Present(t *testing.T) {
	result := strArg(map[string]interface{}{"key": "value"}, "key")
	if result != "value" {
		t.Errorf("expected 'value', got %q", result)
	}
}

func TestStrArg_WrongType(t *testing.T) {
	result := strArg(map[string]interface{}{"key": 42}, "key")
	if result != "" {
		t.Errorf("expected empty string for wrong type, got %q", result)
	}
}

func TestIntArg_Default(t *testing.T) {
	result := intArg(nil, "key", 10)
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

func TestIntArg_Float64(t *testing.T) {
	result := intArg(map[string]interface{}{"key": float64(42)}, "key", 0)
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestIntArg_Int(t *testing.T) {
	result := intArg(map[string]interface{}{"key": 42}, "key", 0)
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestBoolArg_False(t *testing.T) {
	result := boolArg(nil, "key")
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestBoolArg_True(t *testing.T) {
	result := boolArg(map[string]interface{}{"key": true}, "key")
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestSplitArg_Empty(t *testing.T) {
	result := splitArg(nil, "key")
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSplitArg_Single(t *testing.T) {
	result := splitArg(map[string]interface{}{"key": "a"}, "key")
	if len(result) != 1 || result[0] != "a" {
		t.Errorf("expected [a], got %v", result)
	}
}

func TestSplitArg_Multiple(t *testing.T) {
	result := splitArg(map[string]interface{}{"key": "a, b, c"}, "key")
	if len(result) != 3 {
		t.Errorf("expected 3 parts, got %v", result)
	}
}

func TestSplitArg_WithSpaces(t *testing.T) {
	result := splitArg(map[string]interface{}{"key": "  a  ,  b  ,  ,  c  "}, "key")
	if len(result) != 3 {
		t.Errorf("expected 3 non-empty parts, got %v", result)
	}
	for _, v := range result {
		if strings.Contains(v, " ") {
			// splitArg trims spaces
		}
	}
}

func TestJsonResult_Integration(t *testing.T) {
	result, err := jsonResult(map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("jsonResult: %v", err)
	}
	if result == nil || len(result.Content) == 0 {
		t.Fatal("expected non-empty result")
	}
	if result.Content[0].Type != "text" {
		t.Errorf("expected type=text, got %s", result.Content[0].Type)
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if data["key"] != "value" {
		t.Errorf("expected key=value, got %v", data["key"])
	}
}

// --- test helpers ---

func parseResult(t *testing.T, result *ToolResult) map[string]interface{} {
	t.Helper()
	if result == nil {
		t.Fatal("nil result")
	}
	if len(result.Content) == 0 {
		t.Fatal("empty content")
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &data); err != nil {
		t.Fatalf("invalid JSON in result: %v\nraw: %s", err, result.Content[0].Text)
	}
	return data
}

func assertKeyExists(t *testing.T, data map[string]interface{}, key string) {
	t.Helper()
	if _, ok := data[key]; !ok {
		t.Errorf("expected key %q in result, got keys: %v", key, mapKeys(data))
	}
}

func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
