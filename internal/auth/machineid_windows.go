//go:build windows

package auth

func machineID() (string, error) {
	id, err := windowsMachineID()
	if err == nil && id != "" {
		return id, nil
	}
	return hostnameID()
}
