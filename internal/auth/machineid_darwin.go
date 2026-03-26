//go:build darwin

package auth

import "syscall"

func machineID() (string, error) {
	uuid, err := syscall.Sysctl("kern.uuid")
	if err == nil && uuid != "" {
		return uuid, nil
	}
	return hostnameID()
}
