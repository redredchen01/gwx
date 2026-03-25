package cmd

import (
	"errors"
	"strings"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
	"google.golang.org/api/googleapi"
)

// handleAPIError maps Google API errors to exit codes with detailed fix suggestions.
func handleAPIError(rctx *RunContext, err error) error {
	msg := err.Error()

	var circuitErr *api.CircuitOpenError
	if errors.As(err, &circuitErr) {
		return rctx.Printer.ErrExit(exitcode.CircuitOpen, msg)
	}

	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		switch gErr.Code {
		case 401:
			detail := "Google API returned 401 (Unauthorized). "
			if strings.Contains(msg, "invalid_grant") || strings.Contains(msg, "Token has been expired") {
				detail += "Your token has expired or been revoked. Fix: gwx auth login"
			} else {
				detail += "Fix: gwx auth login"
			}
			return rctx.Printer.ErrExit(exitcode.AuthExpired, detail)
		case 403:
			detail := "Google API returned 403 (Forbidden). "
			if strings.Contains(msg, "insufficientPermissions") || strings.Contains(msg, "PERMISSION_DENIED") {
				detail += "The current OAuth token lacks required scopes. Fix: gwx auth login --services gmail,calendar,drive,docs,sheets,tasks,people,chat"
			} else if strings.Contains(msg, "dailyLimitExceeded") || strings.Contains(msg, "userRateLimitExceeded") {
				detail += "API quota exceeded. Wait a few minutes and retry."
			} else {
				detail += "Fix: gwx auth login (to re-authorize with required scopes)"
			}
			return rctx.Printer.ErrExit(exitcode.PermissionDenied, detail)
		case 404:
			detail := "Resource not found (404). "
			if strings.Contains(msg, "notFound") {
				detail += "The requested ID does not exist or you don't have access. Fix: use 'gwx <service> list' to find valid IDs"
			} else {
				detail += "Fix: verify the ID/path is correct"
			}
			return rctx.Printer.ErrExit(exitcode.NotFound, detail)
		case 429:
			return rctx.Printer.ErrExit(exitcode.RateLimited, "Google API rate limit hit (429). Wait 30 seconds and retry. If persistent, check quota at https://console.cloud.google.com/apis/dashboard")
		case 409:
			return rctx.Printer.ErrExit(exitcode.Conflict, "Resource conflict (409): the resource was modified concurrently. Retry your operation.")
		}
	}

	// Detect provider-specific auth failures from error messages.
	if isProviderAuthError(msg) {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, enrichProviderAuthError(msg))
	}

	return rctx.Printer.ErrExit(exitcode.GeneralError, msg)
}

// isProviderAuthError checks if an error message indicates a provider auth issue.
func isProviderAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "no github token") ||
		strings.Contains(lower, "no slack token") ||
		strings.Contains(lower, "no notion token") ||
		strings.Contains(lower, "not authenticated")
}

// enrichProviderAuthError adds the exact fix command to a provider auth error.
func enrichProviderAuthError(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "github"):
		return msg + ". Fix: gwx github login --token <GITHUB_TOKEN>"
	case strings.Contains(lower, "slack"):
		return msg + ". Fix: gwx slack login --token <SLACK_TOKEN>"
	case strings.Contains(lower, "notion"):
		return msg + ". Fix: gwx notion login --token <NOTION_TOKEN>"
	default:
		return msg + ". Fix: gwx onboard (or gwx auth login)"
	}
}
