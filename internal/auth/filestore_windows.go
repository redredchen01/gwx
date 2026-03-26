//go:build windows

package auth

import (
	"fmt"
	"os/exec"
	"strings"
)

// windowsMachineID reads the MachineGuid from the Windows registry.
// Falls back to `reg query` to avoid CGO dependency.
func windowsMachineID() (string, error) {
	out, err := exec.Command(
		"reg", "query",
		`HKLM\SOFTWARE\Microsoft\Cryptography`,
		"/v", "MachineGuid",
	).Output()
	if err != nil {
		return "", fmt.Errorf("reg query: %w", err)
	}
	// Output looks like:
	//     MachineGuid    REG_SZ    xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MachineGuid") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[len(parts)-1], nil
			}
		}
	}
	return "", fmt.Errorf("MachineGuid not found in reg output")
}
