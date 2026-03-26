//go:build linux

package auth

import "os"

func machineID() (string, error) {
	data, err := os.ReadFile("/etc/machine-id")
	if err == nil && len(data) > 0 {
		return string(data), nil
	}
	return hostnameID()
}
