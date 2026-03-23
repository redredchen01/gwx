package testutil

// SampleGmailMessage returns a Gmail API-format message suitable for mock responses.
// The structure matches what gmail/v1.Message looks like when serialized to JSON.
func SampleGmailMessage() map[string]interface{} {
	return map[string]interface{}{
		"id":                "msg_001",
		"threadId":          "thread_001",
		"snippet":           "Hey, let's sync on the Q2 roadmap tomorrow.",
		"labelIds":          []interface{}{"INBOX", "UNREAD"},
		"internalDate":      "1711180800000",
		"sizeEstimate":      1234,
		"historyId":         "12345",
		"resultSizeEstimate": float64(1),
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

// SampleGmailMessage2 returns a second distinct Gmail message for list tests.
func SampleGmailMessage2() map[string]interface{} {
	return map[string]interface{}{
		"id":                "msg_002",
		"threadId":          "thread_002",
		"snippet":           "Invoice #1042 is attached.",
		"labelIds":          []interface{}{"INBOX"},
		"internalDate":      "1711267200000",
		"sizeEstimate":      2048,
		"historyId":         "12346",
		"resultSizeEstimate": float64(1),
		"payload": map[string]interface{}{
			"mimeType": "text/plain",
			"headers": []interface{}{
				map[string]interface{}{"name": "From", "value": "billing@vendor.com"},
				map[string]interface{}{"name": "To", "value": "bob@example.com"},
				map[string]interface{}{"name": "Subject", "value": "Invoice #1042"},
				map[string]interface{}{"name": "Date", "value": "Tue, 24 Mar 2026 09:00:00 +0800"},
			},
			"body": map[string]interface{}{
				"size": 26,
				"data": "SW52b2ljZSAjMTA0MiBpcyBhdHRhY2hlZC4=",
			},
		},
	}
}

// SampleGmailLabels returns a Gmail labels response body.
func SampleGmailLabels() map[string]interface{} {
	return map[string]interface{}{
		"labels": []interface{}{
			map[string]interface{}{"id": "INBOX", "name": "INBOX", "type": "system"},
			map[string]interface{}{"id": "SENT", "name": "SENT", "type": "system"},
			map[string]interface{}{"id": "UNREAD", "name": "UNREAD", "type": "system"},
			map[string]interface{}{"id": "Label_1", "name": "Work", "type": "user"},
		},
	}
}

// SampleCalendarEvent returns a Calendar API-format event.
// The structure matches what calendar/v3.Event looks like when serialized to JSON.
func SampleCalendarEvent() map[string]interface{} {
	return map[string]interface{}{
		"id":      "evt_001",
		"summary": "Sprint Planning",
		"status":  "confirmed",
		"start": map[string]interface{}{
			"dateTime": "2026-03-23T10:00:00+08:00",
		},
		"end": map[string]interface{}{
			"dateTime": "2026-03-23T11:00:00+08:00",
		},
		"location":    "Meeting Room A",
		"description": "Weekly sprint planning session",
		"organizer": map[string]interface{}{
			"email": "pm@example.com",
		},
		"attendees": []interface{}{
			map[string]interface{}{"email": "dev1@example.com"},
			map[string]interface{}{"email": "dev2@example.com"},
		},
		"hangoutLink": "https://meet.google.com/abc-defg-hij",
		"htmlLink":    "https://calendar.google.com/event?eid=evt_001",
	}
}

// SampleCalendarEvent2 returns a second distinct calendar event.
func SampleCalendarEvent2() map[string]interface{} {
	return map[string]interface{}{
		"id":      "evt_002",
		"summary": "1:1 with Manager",
		"status":  "confirmed",
		"start": map[string]interface{}{
			"dateTime": "2026-03-23T14:00:00+08:00",
		},
		"end": map[string]interface{}{
			"dateTime": "2026-03-23T14:30:00+08:00",
		},
		"organizer": map[string]interface{}{
			"email": "manager@example.com",
		},
		"htmlLink": "https://calendar.google.com/event?eid=evt_002",
	}
}

// SampleDriveFile returns a Drive API-format file.
// The structure matches what drive/v3.File looks like when serialized to JSON.
func SampleDriveFile() map[string]interface{} {
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

// SampleDriveFile2 returns a second distinct Drive file.
func SampleDriveFile2() map[string]interface{} {
	return map[string]interface{}{
		"id":           "file_002",
		"name":         "Budget 2026.xlsx",
		"mimeType":     "application/vnd.google-apps.spreadsheet",
		"size":         "28672",
		"modifiedTime": "2026-03-21T11:00:00.000Z",
		"createdTime":  "2026-02-15T14:00:00.000Z",
		"webViewLink":  "https://docs.google.com/spreadsheets/d/file_002",
		"parents":      []interface{}{"folder_root"},
		"shared":       true,
		"trashed":      false,
	}
}

// SampleSpreadsheet returns a Sheets API-format spreadsheet metadata response.
func SampleSpreadsheet() map[string]interface{} {
	return map[string]interface{}{
		"spreadsheetId":  "sheet_001",
		"spreadsheetUrl": "https://docs.google.com/spreadsheets/d/sheet_001",
		"properties": map[string]interface{}{
			"title": "Project Tracker",
		},
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
			map[string]interface{}{
				"properties": map[string]interface{}{
					"sheetId": float64(1),
					"title":   "Backlog",
					"index":   float64(1),
					"gridProperties": map[string]interface{}{
						"rowCount":    float64(50),
						"columnCount": float64(6),
					},
				},
			},
		},
	}
}

// SampleSpreadsheetValues returns a Sheets values response for range reads.
func SampleSpreadsheetValues() map[string]interface{} {
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
