package auth

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// newTestFileStore creates a FileStore with a random 32-byte key and temp dir.
func newTestFileStore(t *testing.T) *FileStore {
	t.Helper()
	dir := t.TempDir()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return NewFileStoreWithKey(dir, key)
}

func TestFileStore_SaveLoadToken_Roundtrip(t *testing.T) {
	fs := newTestFileStore(t)
	token := &oauth2.Token{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Round(time.Second),
	}

	if err := fs.SaveToken("user@example.com", token); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	got, err := fs.LoadToken("user@example.com")
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}

	if got.AccessToken != token.AccessToken {
		t.Errorf("AccessToken: got %q want %q", got.AccessToken, token.AccessToken)
	}
	if got.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken: got %q want %q", got.RefreshToken, token.RefreshToken)
	}
	if !got.Expiry.Equal(token.Expiry) {
		t.Errorf("Expiry: got %v want %v", got.Expiry, token.Expiry)
	}
}

func TestFileStore_SaveLoadCredentials_Roundtrip(t *testing.T) {
	fs := newTestFileStore(t)
	creds := &OAuthCredentials{
		ClientID:     "client-id-123",
		ClientSecret: "secret-xyz",
		ProjectID:    "my-project",
	}

	if err := fs.SaveCredentials("default", creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	got, err := fs.LoadCredentials("default")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	if got.ClientID != creds.ClientID {
		t.Errorf("ClientID: got %q want %q", got.ClientID, creds.ClientID)
	}
	if got.ClientSecret != creds.ClientSecret {
		t.Errorf("ClientSecret: got %q want %q", got.ClientSecret, creds.ClientSecret)
	}
	if got.ProjectID != creds.ProjectID {
		t.Errorf("ProjectID: got %q want %q", got.ProjectID, creds.ProjectID)
	}
}

func TestFileStore_SaveLoadProviderToken_Roundtrip(t *testing.T) {
	fs := newTestFileStore(t)

	if err := fs.SaveProviderToken("github", "alice", "ghp_token123"); err != nil {
		t.Fatalf("SaveProviderToken: %v", err)
	}

	got, err := fs.LoadProviderToken("github", "alice")
	if err != nil {
		t.Fatalf("LoadProviderToken: %v", err)
	}
	if got != "ghp_token123" {
		t.Errorf("got %q want %q", got, "ghp_token123")
	}

	// HasProviderToken
	if !fs.HasProviderToken("github", "alice") {
		t.Error("HasProviderToken should return true")
	}
	if fs.HasProviderToken("github", "nonexistent") {
		t.Error("HasProviderToken should return false for nonexistent")
	}

	// DeleteProviderToken
	if err := fs.DeleteProviderToken("github", "alice"); err != nil {
		t.Fatalf("DeleteProviderToken: %v", err)
	}
	if fs.HasProviderToken("github", "alice") {
		t.Error("token should be deleted")
	}
}

func TestFileStore_CorruptedFile(t *testing.T) {
	fs := newTestFileStore(t)

	// Write random garbage to credentials.enc
	encPath := filepath.Join(fs.dir, "credentials.enc")
	garbage := make([]byte, 64)
	if _, err := rand.Read(garbage); err != nil {
		t.Fatalf("rand: %v", err)
	}
	if err := os.WriteFile(encPath, garbage, 0600); err != nil {
		t.Fatalf("write corrupted file: %v", err)
	}

	_, err := fs.LoadToken("any")
	if err == nil {
		t.Fatal("expected error for corrupted file, got nil")
	}
	if !errors.Is(err, ErrCredentialCorrupted) && !errors.Is(err, ErrKeyMismatch) {
		t.Errorf("expected ErrCredentialCorrupted or ErrKeyMismatch, got: %v", err)
	}
}

func TestFileStore_KeyMismatch(t *testing.T) {
	dir := t.TempDir()
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	if _, err := rand.Read(key1); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatal(err)
	}

	fs1 := NewFileStoreWithKey(dir, key1)
	fs2 := NewFileStoreWithKey(dir, key2)

	// Save with key1
	if err := fs1.SaveToken("user", &oauth2.Token{AccessToken: "tok"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	// Load with key2 — should fail
	_, err := fs2.LoadToken("user")
	if err == nil {
		t.Fatal("expected error for key mismatch, got nil")
	}
	if !errors.Is(err, ErrCredentialCorrupted) && !errors.Is(err, ErrKeyMismatch) {
		t.Errorf("expected ErrCredentialCorrupted or ErrKeyMismatch, got: %v", err)
	}
}

func TestFileStore_ChmodPermissions(t *testing.T) {
	fs := newTestFileStore(t)

	if err := fs.SaveToken("user", &oauth2.Token{AccessToken: "tok"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	encPath := filepath.Join(fs.dir, "credentials.enc")
	info, err := os.Stat(encPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected 0600, got %04o", perm)
	}
}

func TestFileStore_EnsurePermissions_Fix644(t *testing.T) {
	fs := newTestFileStore(t)

	if err := fs.SaveToken("user", &oauth2.Token{AccessToken: "tok"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	encPath := filepath.Join(fs.dir, "credentials.enc")
	// Manually set to 644
	if err := os.Chmod(encPath, 0644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	// Any operation should fix permissions
	if err := fs.SaveToken("user2", &oauth2.Token{AccessToken: "tok2"}); err != nil {
		t.Fatalf("SaveToken after chmod: %v", err)
	}

	info, err := os.Stat(encPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected 0600 after fix, got %04o", perm)
	}
}

func TestFileStore_AtomicWrite(t *testing.T) {
	fs := newTestFileStore(t)

	if err := fs.SaveToken("user", &oauth2.Token{AccessToken: "tok"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	// .tmp file must not remain
	tmpPath := filepath.Join(fs.dir, "credentials.enc.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error(".tmp file should not remain after SaveToken")
	}
}

func TestFileStore_TokenNotFound(t *testing.T) {
	fs := newTestFileStore(t)

	_, err := fs.LoadToken("nonexistent")
	if err == nil {
		t.Fatal("expected ErrTokenNotFound, got nil")
	}
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got: %v", err)
	}
}

func TestFileStore_DeleteToken(t *testing.T) {
	fs := newTestFileStore(t)

	tok := &oauth2.Token{AccessToken: "to-delete"}
	if err := fs.SaveToken("u", tok); err != nil {
		t.Fatal(err)
	}
	if err := fs.DeleteToken("u"); err != nil {
		t.Fatal(err)
	}
	_, err := fs.LoadToken("u")
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound after delete, got: %v", err)
	}
}
