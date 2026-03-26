package auth

import (
	"errors"
	"log"
	"os"

	"github.com/zalando/go-keyring"
)

// SelectBackend detects the environment and returns the appropriate TokenStore.
// Priority: GWX_CREDENTIAL_BACKEND env var > keyring probe > DISPLAY/SSH_TTY detection.
//
// The function never returns an error; it falls back to FileStore on any failure.
func SelectBackend() TokenStore {
	// 1. ENV VAR override
	if val := os.Getenv("GWX_CREDENTIAL_BACKEND"); val != "" {
		switch val {
		case "file":
			return mustFileStore()
		case "keyring":
			log.Printf("auth: using keyring backend")
			return &KeyringStore{}
		default:
			log.Printf("WARN: invalid GWX_CREDENTIAL_BACKEND=%q, using file", val)
			return mustFileStore()
		}
	}

	// 2. Keyring probe: try to get a nonexistent key.
	//    ErrNotFound means keyring is operational (key simply doesn't exist yet).
	//    Any other error means keyring is broken / unavailable.
	_, probeErr := keyring.Get("gwx-probe", "probe")
	if probeErr == nil || errors.Is(probeErr, keyring.ErrNotFound) {
		log.Printf("auth: using keyring backend")
		return &KeyringStore{}
	}

	// 3. Environment detection: headless or SSH session → FileStore
	if os.Getenv("DISPLAY") == "" {
		return mustFileStore()
	}
	if os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" {
		return mustFileStore()
	}

	// 4. One more keyring attempt (X11 forwarding / unexpected env).
	//    If it still fails, fall through to FileStore.
	_, probeErr2 := keyring.Get("gwx-probe", "probe")
	if probeErr2 == nil || errors.Is(probeErr2, keyring.ErrNotFound) {
		log.Printf("auth: using keyring backend")
		return &KeyringStore{}
	}

	// 5. Default fallback
	return mustFileStore()
}

// mustFileStore creates a FileStore and panics only if the OS cannot provide
// a config directory (which should never happen on any supported platform).
func mustFileStore() *FileStore {
	fs, err := NewFileStore()
	if err != nil {
		// This should not happen: os.UserConfigDir() failed on a supported OS.
		// Panic is acceptable here — the system is fundamentally broken.
		log.Panicf("auth: cannot create file store: %v", err)
	}
	log.Printf("auth: using file backend (%s)", fs.dir)
	return fs
}
