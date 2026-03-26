package auth

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

// Note: auth_test.go already calls keyring.MockInit() in its init(), so the
// in-memory mock backend is active for all tests in this package.
// Under the mock, keyring.Get("gwx-probe", "probe") returns ErrNotFound,
// which SelectBackend interprets as "keyring available".
//
// Therefore:
//   - When GWX_CREDENTIAL_BACKEND is empty AND no SSH env vars are set,
//     the keyring probe succeeds → KeyringStore.
//   - When GWX_CREDENTIAL_BACKEND is empty AND SSH_TTY is set,
//     the probe still succeeds (mock), so the result is still KeyringStore.
//     The SSH path is only reachable when the keyring probe fails (real env).
//
// The SSH-session path is verified via the headless branch: DISPLAY="" forces
// FileStore only when the probe also fails, but that's a runtime-only scenario.
// We test it by simulating the probe failure inline with a helper.

// TestDetector_EnvVarFile verifies that GWX_CREDENTIAL_BACKEND=file
// forces FileStore regardless of keyring availability.
func TestDetector_EnvVarFile(t *testing.T) {
	t.Setenv("GWX_CREDENTIAL_BACKEND", "file")
	store := SelectBackend()
	if _, ok := store.(*FileStore); !ok {
		t.Fatalf("expected *FileStore, got %T", store)
	}
}

// TestDetector_EnvVarKeyring verifies that GWX_CREDENTIAL_BACKEND=keyring
// forces KeyringStore regardless of environment.
func TestDetector_EnvVarKeyring(t *testing.T) {
	t.Setenv("GWX_CREDENTIAL_BACKEND", "keyring")
	store := SelectBackend()
	if _, ok := store.(*KeyringStore); !ok {
		t.Fatalf("expected *KeyringStore, got %T", store)
	}
}

// TestDetector_EnvVarInvalid verifies that an invalid GWX_CREDENTIAL_BACKEND
// value logs a warning and falls back to FileStore.
func TestDetector_EnvVarInvalid(t *testing.T) {
	t.Setenv("GWX_CREDENTIAL_BACKEND", "bogus")
	store := SelectBackend()
	if _, ok := store.(*FileStore); !ok {
		t.Fatalf("expected *FileStore (invalid env → fallback), got %T", store)
	}
}

// TestDetector_AutoDetect_KeyringAvailable verifies that when no env var is set
// and the keyring probe succeeds (ErrNotFound = keyring works, mock is active),
// SelectBackend returns KeyringStore.
func TestDetector_AutoDetect_KeyringAvailable(t *testing.T) {
	t.Setenv("GWX_CREDENTIAL_BACKEND", "")
	// keyring.MockInit() is active (from auth_test.go init),
	// so the probe returns ErrNotFound → keyring available → KeyringStore.
	store := SelectBackend()
	if _, ok := store.(*KeyringStore); !ok {
		t.Fatalf("expected *KeyringStore when keyring available, got %T", store)
	}
}

// TestDetector_SSHSession verifies the priority ordering:
// with GWX_CREDENTIAL_BACKEND=="" and SSH_TTY set, keyring probe still
// runs first. Under the mock it succeeds, so we get KeyringStore.
// This confirms that SSH detection only fires when probe truly fails.
func TestDetector_SSHSession(t *testing.T) {
	t.Setenv("GWX_CREDENTIAL_BACKEND", "")
	t.Setenv("SSH_TTY", "/dev/pts/0")
	// mock keyring is active; probe returns ErrNotFound (= available)
	// Priority: probe > SSH detection → KeyringStore
	store := SelectBackend()
	if _, ok := store.(*KeyringStore); !ok {
		t.Fatalf("expected *KeyringStore (probe wins over SSH_TTY in mock env), got %T", store)
	}
}

// TestDetector_selectBackendLogic exercises the internal branching logic
// directly via the exported function using env-var overrides.
// Confirms that file override beats keyring availability and SSH session.
func TestDetector_FileOverrideBeatsSSH(t *testing.T) {
	t.Setenv("GWX_CREDENTIAL_BACKEND", "file")
	t.Setenv("SSH_TTY", "/dev/pts/0")
	store := SelectBackend()
	if _, ok := store.(*FileStore); !ok {
		t.Fatalf("expected *FileStore (env var beats SSH), got %T", store)
	}
}

// TestDetector_NoDisplay_NoKeyring verifies that when DISPLAY is empty AND the
// keyring probe fails (daemon unavailable), SelectBackend falls back to *FileStore.
// MockInitWithError forces the probe to return a non-ErrNotFound error, which
// causes SelectBackend to reach the headless/no-keyring branch.
func TestDetector_NoDisplay_NoKeyring(t *testing.T) {
	// Restore functional mock when test ends.
	t.Cleanup(func() { keyring.MockInit() })

	// Simulate broken keyring: probe returns unexpected error.
	keyring.MockInitWithError(errors.New("no keyring daemon"))

	t.Setenv("GWX_CREDENTIAL_BACKEND", "")
	t.Setenv("DISPLAY", "")
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")

	store := SelectBackend()
	if _, ok := store.(*FileStore); !ok {
		t.Fatalf("expected *FileStore (no display + no keyring), got %T", store)
	}
}
