package auth

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func init() {
	// Use the in-memory mock backend so tests don't touch the real OS keyring.
	keyring.MockInit()
	// Initialize defaultStore so package-level provider functions work.
	NewManagerWithStore(&KeyringStore{})
}

func TestSaveProviderToken_KeyFormat(t *testing.T) {
	err := SaveProviderToken("github", "user1", "ghp_abc123")
	if err != nil {
		t.Fatalf("SaveProviderToken failed: %v", err)
	}

	// Verify the token is stored under key "provider:github:user1"
	// by loading it back through the same function.
	tok, err := LoadProviderToken("github", "user1")
	if err != nil {
		t.Fatalf("LoadProviderToken failed: %v", err)
	}
	if tok != "ghp_abc123" {
		t.Fatalf("expected token=ghp_abc123, got %q", tok)
	}

	// Also verify directly via keyring.Get to confirm key format.
	direct, err := keyring.Get(keyringService, "provider:github:user1")
	if err != nil {
		t.Fatalf("keyring.Get with explicit key failed: %v", err)
	}
	if direct != "ghp_abc123" {
		t.Fatalf("expected direct read=ghp_abc123, got %q", direct)
	}
}

func TestSaveProviderToken_MultipleProviders(t *testing.T) {
	// Save tokens for different providers under same account.
	if err := SaveProviderToken("slack", "workspace1", "xoxb-slack"); err != nil {
		t.Fatalf("save slack: %v", err)
	}
	if err := SaveProviderToken("notion", "workspace1", "ntn_notion"); err != nil {
		t.Fatalf("save notion: %v", err)
	}

	slackTok, err := LoadProviderToken("slack", "workspace1")
	if err != nil {
		t.Fatalf("load slack: %v", err)
	}
	notionTok, err := LoadProviderToken("notion", "workspace1")
	if err != nil {
		t.Fatalf("load notion: %v", err)
	}

	if slackTok != "xoxb-slack" {
		t.Fatalf("expected slack token=xoxb-slack, got %q", slackTok)
	}
	if notionTok != "ntn_notion" {
		t.Fatalf("expected notion token=ntn_notion, got %q", notionTok)
	}
}

func TestHasProviderToken_Missing(t *testing.T) {
	has := HasProviderToken("nonexistent-provider", "nonexistent-account")
	if has {
		t.Fatal("expected HasProviderToken to return false for missing token")
	}
}

func TestHasProviderToken_Exists(t *testing.T) {
	if err := SaveProviderToken("github", "has-test", "token123"); err != nil {
		t.Fatalf("save: %v", err)
	}

	has := HasProviderToken("github", "has-test")
	if !has {
		t.Fatal("expected HasProviderToken to return true for existing token")
	}
}

func TestLoadProviderToken_Missing(t *testing.T) {
	_, err := LoadProviderToken("missing", "missing")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestDeleteProviderToken(t *testing.T) {
	if err := SaveProviderToken("github", "del-test", "to-delete"); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify it exists.
	if !HasProviderToken("github", "del-test") {
		t.Fatal("token should exist before delete")
	}

	if err := DeleteProviderToken("github", "del-test"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify it's gone.
	if HasProviderToken("github", "del-test") {
		t.Fatal("token should not exist after delete")
	}
}

func TestSaveProviderToken_Overwrite(t *testing.T) {
	if err := SaveProviderToken("github", "overwrite-test", "old-token"); err != nil {
		t.Fatalf("save old: %v", err)
	}
	if err := SaveProviderToken("github", "overwrite-test", "new-token"); err != nil {
		t.Fatalf("save new: %v", err)
	}

	tok, err := LoadProviderToken("github", "overwrite-test")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if tok != "new-token" {
		t.Fatalf("expected new-token after overwrite, got %q", tok)
	}
}

func TestProviderToken_NotInitialized(t *testing.T) {
	// Reset the singleton so we can test the nil-store path.
	resetDefaultStoreForTest()
	defer func() {
		// Restore so subsequent tests are not broken.
		keyring.MockInit()
		NewManagerWithStore(&KeyringStore{})
	}()

	if err := SaveProviderToken("x", "y", "z"); err == nil {
		t.Fatal("expected error when defaultStore is nil")
	}
	if _, err := LoadProviderToken("x", "y"); err == nil {
		t.Fatal("expected error when defaultStore is nil")
	}
	if err := DeleteProviderToken("x", "y"); err == nil {
		t.Fatal("expected error when defaultStore is nil")
	}
	if HasProviderToken("x", "y") {
		t.Fatal("expected false when defaultStore is nil")
	}
}

func TestDefaultStore_SyncOnce(t *testing.T) {
	// Reset so we start from a clean state.
	resetDefaultStoreForTest()
	defer func() {
		keyring.MockInit()
		NewManagerWithStore(&KeyringStore{})
	}()

	first := &KeyringStore{}
	second := &KeyringStore{}

	setDefaultStore(first)
	setDefaultStore(second) // should be ignored by sync.Once

	if getDefaultStore() != first {
		t.Fatal("setDefaultStore: second call should not override the first (sync.Once)")
	}
}

func TestNewManagerWithStore_UsesProvidedStore(t *testing.T) {
	resetDefaultStoreForTest()
	defer func() {
		keyring.MockInit()
		NewManagerWithStore(&KeyringStore{})
	}()

	mock := &mockTokenStore{}
	m := NewManagerWithStore(mock)

	if m.store != mock {
		t.Fatal("manager should use the provided store")
	}
	if getDefaultStore() != mock {
		t.Fatal("defaultStore should be set to the provided store")
	}
}

// mockTokenStore is a minimal TokenStore for injection tests.
type mockTokenStore struct {
	KeyringStore // embed to satisfy interface without implementing all methods
}
