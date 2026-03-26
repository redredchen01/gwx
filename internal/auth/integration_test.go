package auth

import (
	"crypto/rand"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// TestIntegration_FileStore_FullRoundtrip exercises all three data types
// (OAuth2 token, OAuthCredentials, provider token) through a single FileStore
// instance to confirm end-to-end serialize → encrypt → decrypt → deserialize.
func TestIntegration_FileStore_FullRoundtrip(t *testing.T) {
	dir := t.TempDir()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	fs := NewFileStoreWithKey(dir, key)

	// --- OAuth2 token ---
	want := &oauth2.Token{
		AccessToken:  "access-integration",
		RefreshToken: "refresh-integration",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Round(time.Second),
	}
	if err := fs.SaveToken("integ@example.com", want); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	gotTok, err := fs.LoadToken("integ@example.com")
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if gotTok.AccessToken != want.AccessToken {
		t.Errorf("token AccessToken: got %q want %q", gotTok.AccessToken, want.AccessToken)
	}
	if gotTok.RefreshToken != want.RefreshToken {
		t.Errorf("token RefreshToken: got %q want %q", gotTok.RefreshToken, want.RefreshToken)
	}
	if !gotTok.Expiry.Equal(want.Expiry) {
		t.Errorf("token Expiry: got %v want %v", gotTok.Expiry, want.Expiry)
	}

	// --- OAuthCredentials ---
	wantCreds := &OAuthCredentials{
		ClientID:     "integ-client-id",
		ClientSecret: "integ-client-secret",
		ProjectID:    "integ-project",
	}
	if err := fs.SaveCredentials("integ-creds", wantCreds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}
	gotCreds, err := fs.LoadCredentials("integ-creds")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if gotCreds.ClientID != wantCreds.ClientID {
		t.Errorf("creds ClientID: got %q want %q", gotCreds.ClientID, wantCreds.ClientID)
	}
	if gotCreds.ClientSecret != wantCreds.ClientSecret {
		t.Errorf("creds ClientSecret: got %q want %q", gotCreds.ClientSecret, wantCreds.ClientSecret)
	}
	if gotCreds.ProjectID != wantCreds.ProjectID {
		t.Errorf("creds ProjectID: got %q want %q", gotCreds.ProjectID, wantCreds.ProjectID)
	}

	// --- Provider token ---
	if err := fs.SaveProviderToken("github", "integ-user", "ghp_integ_token"); err != nil {
		t.Fatalf("SaveProviderToken: %v", err)
	}
	gotProv, err := fs.LoadProviderToken("github", "integ-user")
	if err != nil {
		t.Fatalf("LoadProviderToken: %v", err)
	}
	if gotProv != "ghp_integ_token" {
		t.Errorf("provider token: got %q want %q", gotProv, "ghp_integ_token")
	}
}

// TestIntegration_ProviderToken_ViaDefaultStore verifies that the package-level
// SaveProviderToken / LoadProviderToken functions route through the store
// registered by NewManagerWithStore.
func TestIntegration_ProviderToken_ViaDefaultStore(t *testing.T) {
	// Reset the singleton so this test owns the default store.
	resetDefaultStoreForTest()
	t.Cleanup(resetDefaultStoreForTest)

	dir := t.TempDir()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	fileStore := NewFileStoreWithKey(dir, key)
	NewManagerWithStore(fileStore)

	if err := SaveProviderToken("github", "user", "ghp_xxx"); err != nil {
		t.Fatalf("SaveProviderToken: %v", err)
	}
	got, err := LoadProviderToken("github", "user")
	if err != nil {
		t.Fatalf("LoadProviderToken: %v", err)
	}
	if got != "ghp_xxx" {
		t.Errorf("got %q want %q", got, "ghp_xxx")
	}
}

// TestIntegration_BackendSwitch confirms that tokens stored in one backend
// are not visible to a manager initialized with a different backend.
// This validates the isolation contract between FileStore instances.
func TestIntegration_BackendSwitch(t *testing.T) {
	// Reset singleton before and after.
	resetDefaultStoreForTest()
	t.Cleanup(resetDefaultStoreForTest)

	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	if _, err := rand.Read(key1); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatal(err)
	}

	// Backend A: first store (dir1)
	dirA := t.TempDir()
	storeA := NewFileStoreWithKey(dirA, key1)
	NewManagerWithStore(storeA)

	if err := SaveProviderToken("github", "switch-user", "token-A"); err != nil {
		t.Fatalf("SaveProviderToken on A: %v", err)
	}

	// Backend B: different dir — token saved in A is invisible here.
	resetDefaultStoreForTest()
	dirB := t.TempDir()
	storeB := NewFileStoreWithKey(dirB, key2)
	NewManagerWithStore(storeB)

	_, err := LoadProviderToken("github", "switch-user")
	if err == nil {
		t.Fatal("expected error: token from backend A should not be visible in backend B")
	}
}
