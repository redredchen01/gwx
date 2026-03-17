package auth

import "google.golang.org/api/gmail/v1"

// ServiceScopes maps service names to their required OAuth2 scopes.
// We use the most permissive scope per service to avoid re-auth issues.
var ServiceScopes = map[string][]string{
	"gmail": {
		gmail.GmailReadonlyScope,
		gmail.GmailSendScope,
		gmail.GmailModifyScope,
	},
	"calendar": {
		"https://www.googleapis.com/auth/calendar",
	},
	"drive": {
		"https://www.googleapis.com/auth/drive",
	},
	"docs": {
		"https://www.googleapis.com/auth/documents",
	},
	"sheets": {
		"https://www.googleapis.com/auth/spreadsheets",
	},
	"tasks": {
		"https://www.googleapis.com/auth/tasks",
	},
	"people": {
		"https://www.googleapis.com/auth/contacts.readonly",
	},
	"chat": {
		"https://www.googleapis.com/auth/chat.messages",
		"https://www.googleapis.com/auth/chat.spaces.readonly",
	},
}

// ReadOnlyScopes returns read-only scopes for services that support it.
var ReadOnlyScopes = map[string][]string{
	"gmail":    {gmail.GmailReadonlyScope},
	"calendar": {"https://www.googleapis.com/auth/calendar.readonly"},
	"drive":    {"https://www.googleapis.com/auth/drive.readonly"},
	"docs":     {"https://www.googleapis.com/auth/documents.readonly"},
	"sheets":   {"https://www.googleapis.com/auth/spreadsheets.readonly"},
	"tasks":    {"https://www.googleapis.com/auth/tasks.readonly"},
	"people":   {"https://www.googleapis.com/auth/contacts.readonly"},
	"chat":     {"https://www.googleapis.com/auth/chat.spaces.readonly"},
}

// AllScopes returns the union of all scopes for the given services.
func AllScopes(services []string, readOnly bool) []string {
	scopeMap := ServiceScopes
	if readOnly {
		scopeMap = ReadOnlyScopes
	}
	seen := make(map[string]bool)
	var result []string
	for _, svc := range services {
		for _, scope := range scopeMap[svc] {
			if !seen[scope] {
				seen[scope] = true
				result = append(result, scope)
			}
		}
	}
	return result
}
