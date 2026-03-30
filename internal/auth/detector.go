package auth

import (
	"errors"
	"log/slog"
	"os"
	"strings"

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
			backendLogf("auth: using keyring backend")
			return &KeyringStore{}
		default:
			backendLogf("WARN: invalid GWX_CREDENTIAL_BACKEND=%q, using file", val)
			return mustFileStore()
		}
	}

	// 2. Keyring probe: try to get a nonexistent key.
	//    ErrNotFound means keyring is operational (key simply doesn't exist yet).
	//    Any other error means keyring is broken / unavailable.
	_, probeErr := keyring.Get("gwx-probe", "probe")
	if probeErr == nil || errors.Is(probeErr, keyring.ErrNotFound) {
		backendLogf("auth: using keyring backend")
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
		backendLogf("auth: using keyring backend")
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
		slog.Error("auth: cannot create file store", "error", err)
		panic(err)
	}
	backendLogf("auth: using file backend (%s)", fs.dir)
	return fs
}

func backendLogf(format string, args ...interface{}) {
	if os.Getenv("GWX_AUTO_JSON") == "1" {
		return
	}
	if strings.EqualFold(os.Getenv("GWX_QUIET_BACKEND_LOGS"), "1") || strings.EqualFold(os.Getenv("GWX_QUIET_BACKEND_LOGS"), "true") {
		return
	}
	slog.Debug(format, "args", args)
}
