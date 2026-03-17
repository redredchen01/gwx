package config

import (
	"os"
	"strings"
)

const envEnableCommands = "GWX_ENABLE_COMMANDS"

// Allowlist controls which commands an agent can execute.
type Allowlist struct {
	patterns []string
}

// LoadAllowlist reads the command allowlist from the environment.
// GWX_ENABLE_COMMANDS="gmail.*,calendar.list,drive.list"
// GWX_ENABLE_COMMANDS="all" or "*" disables the restriction.
func LoadAllowlist() *Allowlist {
	val := os.Getenv(envEnableCommands)
	if val == "" || val == "all" || val == "*" {
		return nil // no restriction
	}
	patterns := strings.Split(val, ",")
	for i := range patterns {
		patterns[i] = strings.TrimSpace(patterns[i])
	}
	return &Allowlist{patterns: patterns}
}

// IsAllowed checks if a command (e.g. "gmail.list") is permitted.
func (a *Allowlist) IsAllowed(command string) bool {
	if a == nil {
		return true
	}
	for _, p := range a.patterns {
		if p == command {
			return true
		}
		// Wildcard: "gmail.*" matches "gmail.list", "gmail.send", etc.
		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix(command, prefix+".") {
				return true
			}
		}
	}
	return false
}
