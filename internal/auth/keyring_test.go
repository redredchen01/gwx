package auth

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func init() {
	// Use the in-memory mock backend so tests don't touch the real OS keyring.
	// Safe to call multiple times; go-keyring deduplicates mock init.
	keyring.MockInit()
}

func TestKeyringStore_SaveProviderToken(t *testing.T) {
	ks := &KeyringStore{}

	if err := ks.SaveProviderToken("github", "user1", "ghp_test123"); err != nil {
		t.Fatalf("SaveProviderToken failed: %v", err)
	}

	// Roundtrip: load it back and verify.
	tok, err := ks.LoadProviderToken("github", "user1")
	if err != nil {
		t.Fatalf("LoadProviderToken after save failed: %v", err)
	}
	if tok != "ghp_test123" {
		t.Fatalf("expected ghp_test123, got %q", tok)
	}
}

func TestKeyringStore_LoadProviderToken_Missing(t *testing.T) {
	ks := &KeyringStore{}

	_, err := ks.LoadProviderToken("nonexistent-provider", "nonexistent-account")
	if err == nil {
		t.Fatal("expected error for missing provider token, got nil")
	}
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestKeyringStore_DeleteProviderToken(t *testing.T) {
	ks := &KeyringStore{}

	if err := ks.SaveProviderToken("slack", "del-test", "xoxb-delete-me"); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := ks.DeleteProviderToken("slack", "del-test"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := ks.LoadProviderToken("slack", "del-test")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound after delete, got %v", err)
	}
}

func TestKeyringStore_HasProviderToken(t *testing.T) {
	ks := &KeyringStore{}

	// Before save: should be false.
	if ks.HasProviderToken("notion", "has-test") {
		t.Fatal("expected false before save")
	}

	if err := ks.SaveProviderToken("notion", "has-test", "ntn_token"); err != nil {
		t.Fatalf("save: %v", err)
	}

	// After save: should be true.
	if !ks.HasProviderToken("notion", "has-test") {
		t.Fatal("expected true after save")
	}

	if err := ks.DeleteProviderToken("notion", "has-test"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// After delete: should be false again.
	if ks.HasProviderToken("notion", "has-test") {
		t.Fatal("expected false after delete")
	}
}
