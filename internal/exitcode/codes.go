package exitcode

import "fmt"

// Stable exit codes for agent automation.
// These codes MUST NOT change once released.
const (
	OK               = 0
	GeneralError     = 1
	UsageError       = 2
	AuthRequired     = 10
	AuthExpired      = 11
	PermissionDenied = 12
	NotFound         = 20
	Conflict         = 21
	RateLimited      = 30
	CircuitOpen      = 31
	InvalidInput     = 40
	DryRunSuccess    = 50
)

var descriptions = map[int]string{
	OK:               "success",
	GeneralError:     "general_error",
	UsageError:       "usage_error",
	AuthRequired:     "auth_required",
	AuthExpired:      "auth_expired",
	PermissionDenied: "permission_denied",
	NotFound:         "not_found",
	Conflict:         "conflict",
	RateLimited:      "rate_limited",
	CircuitOpen:      "circuit_open",
	InvalidInput:     "invalid_input",
	DryRunSuccess:    "dry_run_success",
}

// Description returns the stable string name for an exit code.
func Description(code int) string {
	if d, ok := descriptions[code]; ok {
		return d
	}
	return fmt.Sprintf("unknown_%d", code)
}

// All returns all defined exit codes with descriptions.
func All() map[int]string {
	out := make(map[int]string, len(descriptions))
	for k, v := range descriptions {
		out[k] = v
	}
	return out
}
