//go:build !darwin && !linux && !windows

package auth

func machineID() (string, error) {
	return hostnameID()
}
