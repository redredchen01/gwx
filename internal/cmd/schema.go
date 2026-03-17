package cmd

import (
	"encoding/json"
	"sort"
)

// SchemaCmd outputs the full command schema for agent introspection.
type SchemaCmd struct{}

type commandInfo struct {
	Name        string   `json:"name"`
	Service     string   `json:"service"`
	Description string   `json:"description"`
	SafetyTier  string   `json:"safety_tier"`
	ExampleArgs string   `json:"example_args,omitempty"`
	RequiresAuth bool   `json:"requires_auth"`
}

func (c *SchemaCmd) Run(rctx *RunContext) error {
	commands := []commandInfo{
		// Auth
		{Name: "auth.login", Service: "auth", Description: "Sign in to Google account", SafetyTier: "none"},
		{Name: "auth.logout", Service: "auth", Description: "Remove saved credentials", SafetyTier: "none"},
		{Name: "auth.status", Service: "auth", Description: "Check authentication status", SafetyTier: "none"},
		{Name: "onboard", Service: "auth", Description: "Interactive setup wizard", SafetyTier: "none"},

		// Gmail
		{Name: "gmail.list", Service: "gmail", Description: "List messages", SafetyTier: "green", ExampleArgs: "--limit 10 --unread", RequiresAuth: true},
		{Name: "gmail.get", Service: "gmail", Description: "Get a single message", SafetyTier: "green", ExampleArgs: "MESSAGE_ID", RequiresAuth: true},
		{Name: "gmail.search", Service: "gmail", Description: "Search messages", SafetyTier: "green", ExampleArgs: "\"from:user@example.com\"", RequiresAuth: true},
		{Name: "gmail.labels", Service: "gmail", Description: "List labels", SafetyTier: "green", RequiresAuth: true},
		{Name: "gmail.send", Service: "gmail", Description: "Send an email", SafetyTier: "red", ExampleArgs: "--to a@b.com --subject Hi --body Hello", RequiresAuth: true},
		{Name: "gmail.draft", Service: "gmail", Description: "Create a draft", SafetyTier: "yellow", ExampleArgs: "--to a@b.com --subject Hi --body Hello", RequiresAuth: true},
		{Name: "gmail.reply", Service: "gmail", Description: "Reply to a message", SafetyTier: "red", ExampleArgs: "MSG_ID --body \"Got it\"", RequiresAuth: true},

		// Calendar
		{Name: "calendar.agenda", Service: "calendar", Description: "Show upcoming events", SafetyTier: "green", ExampleArgs: "--days 1", RequiresAuth: true},
		{Name: "calendar.list", Service: "calendar", Description: "List events in date range", SafetyTier: "green", ExampleArgs: "--from today --to tomorrow", RequiresAuth: true},
		{Name: "calendar.create", Service: "calendar", Description: "Create an event", SafetyTier: "yellow", ExampleArgs: "--title Meeting --start 2026-03-18T09:00:00Z --end 2026-03-18T10:00:00Z", RequiresAuth: true},
		{Name: "calendar.update", Service: "calendar", Description: "Update an event", SafetyTier: "yellow", ExampleArgs: "EVENT_ID --title \"New Title\"", RequiresAuth: true},
		{Name: "calendar.delete", Service: "calendar", Description: "Delete an event", SafetyTier: "red", ExampleArgs: "EVENT_ID", RequiresAuth: true},
		{Name: "calendar.find-slot", Service: "calendar", Description: "Find free time slots", SafetyTier: "green", ExampleArgs: "--attendees a@b.com --duration 30m", RequiresAuth: true},

		// Drive
		{Name: "drive.list", Service: "drive", Description: "List files", SafetyTier: "green", ExampleArgs: "--limit 20", RequiresAuth: true},
		{Name: "drive.search", Service: "drive", Description: "Search files", SafetyTier: "green", ExampleArgs: "\"name contains 'report'\"", RequiresAuth: true},
		{Name: "drive.upload", Service: "drive", Description: "Upload a file", SafetyTier: "yellow", ExampleArgs: "file.pdf --folder FOLDER_ID", RequiresAuth: true},
		{Name: "drive.download", Service: "drive", Description: "Download a file", SafetyTier: "green", ExampleArgs: "FILE_ID -o output.pdf", RequiresAuth: true},
		{Name: "drive.share", Service: "drive", Description: "Share a file", SafetyTier: "red", ExampleArgs: "FILE_ID --email user@x.com --role reader", RequiresAuth: true},
		{Name: "drive.mkdir", Service: "drive", Description: "Create a folder", SafetyTier: "yellow", ExampleArgs: "\"New Folder\"", RequiresAuth: true},

		// Docs
		{Name: "docs.get", Service: "docs", Description: "Get document content", SafetyTier: "green", ExampleArgs: "DOC_ID", RequiresAuth: true},
		{Name: "docs.create", Service: "docs", Description: "Create a document", SafetyTier: "yellow", ExampleArgs: "--title \"My Doc\"", RequiresAuth: true},
		{Name: "docs.append", Service: "docs", Description: "Append text to document", SafetyTier: "yellow", ExampleArgs: "DOC_ID --text \"Hello\"", RequiresAuth: true},
		{Name: "docs.export", Service: "docs", Description: "Export document to file", SafetyTier: "green", ExampleArgs: "DOC_ID --export-format pdf", RequiresAuth: true},

		// Sheets
		{Name: "sheets.read", Service: "sheets", Description: "Read a range", SafetyTier: "green", ExampleArgs: "SHEET_ID \"A1:C10\"", RequiresAuth: true},
		{Name: "sheets.append", Service: "sheets", Description: "Append rows", SafetyTier: "yellow", ExampleArgs: "SHEET_ID \"A:C\" --values '[[\"a\",1]]'", RequiresAuth: true},
		{Name: "sheets.update", Service: "sheets", Description: "Update cells", SafetyTier: "yellow", ExampleArgs: "SHEET_ID \"A1:B2\" --values '[[\"x\"]]'", RequiresAuth: true},
		{Name: "sheets.create", Service: "sheets", Description: "Create spreadsheet", SafetyTier: "yellow", ExampleArgs: "--title \"My Sheet\"", RequiresAuth: true},

		// Tasks
		{Name: "tasks.list", Service: "tasks", Description: "List tasks", SafetyTier: "green", RequiresAuth: true},
		{Name: "tasks.lists", Service: "tasks", Description: "List task lists", SafetyTier: "green", RequiresAuth: true},
		{Name: "tasks.create", Service: "tasks", Description: "Create a task", SafetyTier: "yellow", ExampleArgs: "--title \"Buy milk\"", RequiresAuth: true},
		{Name: "tasks.complete", Service: "tasks", Description: "Complete a task", SafetyTier: "yellow", ExampleArgs: "TASK_ID", RequiresAuth: true},
		{Name: "tasks.delete", Service: "tasks", Description: "Delete a task", SafetyTier: "red", ExampleArgs: "TASK_ID", RequiresAuth: true},

		// Contacts
		{Name: "contacts.list", Service: "contacts", Description: "List contacts", SafetyTier: "green", RequiresAuth: true},
		{Name: "contacts.search", Service: "contacts", Description: "Search contacts", SafetyTier: "green", ExampleArgs: "\"john\"", RequiresAuth: true},
		{Name: "contacts.get", Service: "contacts", Description: "Get a contact", SafetyTier: "green", ExampleArgs: "people/c123", RequiresAuth: true},

		// Chat
		{Name: "chat.spaces", Service: "chat", Description: "List Chat spaces", SafetyTier: "green", RequiresAuth: true},
		{Name: "chat.send", Service: "chat", Description: "Send message to space", SafetyTier: "red", ExampleArgs: "spaces/AAA --text \"Hello\"", RequiresAuth: true},
		{Name: "chat.messages", Service: "chat", Description: "List messages in space", SafetyTier: "green", ExampleArgs: "spaces/AAA", RequiresAuth: true},
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	// Also output as a lookup map for agents
	byService := make(map[string][]commandInfo)
	for _, c := range commands {
		byService[c.Service] = append(byService[c.Service], c)
	}

	result := map[string]interface{}{
		"commands":    commands,
		"by_service":  byService,
		"total":       len(commands),
		"services":    []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "contacts", "chat"},
	}

	// Use raw JSON for compact output
	data, _ := json.MarshalIndent(result, "", "  ")
	rctx.Printer.Writer.Write(data)    //nolint:errcheck
	rctx.Printer.Writer.Write([]byte("\n")) //nolint:errcheck
	return nil
}
