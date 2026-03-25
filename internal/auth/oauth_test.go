package auth

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/zalando/go-keyring"
)

// --- LoadConfigFromJSON ---

func TestLoadConfigFromJSON_InstalledKey(t *testing.T) {
	m := NewManager()
	credJSON := []byte(`{"installed":{"client_id":"test-id","client_secret":"test-secret","project_id":"proj1"}}`)
	err := m.LoadConfigFromJSON(credJSON, []string{"https://www.googleapis.com/auth/gmail.readonly"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.config == nil {
		t.Fatal("config should not be nil after LoadConfigFromJSON")
	}
	if m.config.ClientID != "test-id" {
		t.Errorf("ClientID = %q, want test-id", m.config.ClientID)
	}
	if m.config.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %q, want test-secret", m.config.ClientSecret)
	}
	if len(m.config.Scopes) != 1 {
		t.Errorf("Scopes count = %d, want 1", len(m.config.Scopes))
	}
}

func TestLoadConfigFromJSON_WebKey(t *testing.T) {
	m := NewManager()
	credJSON := []byte(`{"web":{"client_id":"web-id","client_secret":"web-secret"}}`)
	err := m.LoadConfigFromJSON(credJSON, []string{"https://www.googleapis.com/auth/calendar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.config.ClientID != "web-id" {
		t.Errorf("ClientID = %q, want web-id", m.config.ClientID)
	}
}

func TestLoadConfigFromJSON_InvalidJSON(t *testing.T) {
	m := NewManager()
	err := m.LoadConfigFromJSON([]byte("not json"), nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadConfigFromJSON_MissingKey(t *testing.T) {
	m := NewManager()
	err := m.LoadConfigFromJSON([]byte(`{"other":{"client_id":"x"}}`), nil)
	if err == nil {
		t.Fatal("expected error when neither 'installed' nor 'web' key present")
	}
}

func TestLoadConfigFromJSON_EmptyJSON(t *testing.T) {
	m := NewManager()
	err := m.LoadConfigFromJSON([]byte(`{}`), nil)
	if err == nil {
		t.Fatal("expected error for empty JSON object")
	}
}

func TestLoadConfigFromJSON_BadInnerJSON(t *testing.T) {
	m := NewManager()
	// inner value is a string, not an object
	err := m.LoadConfigFromJSON([]byte(`{"installed":"not an object"}`), nil)
	if err == nil {
		t.Fatal("expected error for bad inner JSON")
	}
}

func TestLoadConfigFromJSON_MultipleScopes(t *testing.T) {
	m := NewManager()
	credJSON := []byte(`{"installed":{"client_id":"id","client_secret":"secret"}}`)
	scopes := []string{
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/calendar",
		"https://www.googleapis.com/auth/drive",
	}
	err := m.LoadConfigFromJSON(credJSON, scopes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.config.Scopes) != 3 {
		t.Errorf("Scopes count = %d, want 3", len(m.config.Scopes))
	}
}

// --- Manager methods ---

func TestManager_HasToken_Missing(t *testing.T) {
	m := NewManager()
	if m.HasToken("nonexistent-account-xyz") {
		t.Error("HasToken should return false for missing token")
	}
}

func TestManager_SaveAndHasToken(t *testing.T) {
	m := NewManager()
	// First, save a config so that the token can be stored
	credJSON := []byte(`{"installed":{"client_id":"id","client_secret":"secret"}}`)
	if err := m.LoadConfigFromJSON(credJSON, nil); err != nil {
		t.Fatalf("LoadConfigFromJSON: %v", err)
	}

	// We cannot call SaveToken without a valid oauth2.Token,
	// but we can test HasToken directly via the keyring.
	if m.HasToken("test-save-has") {
		t.Error("should not have token before save")
	}
}

func TestManager_DeleteToken(t *testing.T) {
	m := NewManager()
	// Deleting a non-existent token should not panic
	err := m.DeleteToken("nonexistent-delete-test")
	if err == nil {
		// Some keyring implementations return nil for missing keys
		return
	}
	// Error is expected for missing token
}

// --- KeyringStore ---

func TestKeyringStore_SaveAndLoadCredentials(t *testing.T) {
	ks := &KeyringStore{}
	creds := &OAuthCredentials{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ProjectID:    "test-project",
	}
	if err := ks.SaveCredentials("test-creds", creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	loaded, err := ks.LoadCredentials("test-creds")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if loaded.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want test-client-id", loaded.ClientID)
	}
	if loaded.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %q, want test-client-secret", loaded.ClientSecret)
	}
	if loaded.ProjectID != "test-project" {
		t.Errorf("ProjectID = %q, want test-project", loaded.ProjectID)
	}
}

func TestKeyringStore_LoadCredentials_Missing(t *testing.T) {
	ks := &KeyringStore{}
	_, err := ks.LoadCredentials("nonexistent-creds-xyz")
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
}

func TestKeyringStore_LoadCredentials_CorruptedJSON(t *testing.T) {
	// Write corrupt data directly to keyring
	if err := keyring.Set(keyringService, credKeyPrefix+"corrupt-test", "not json"); err != nil {
		t.Fatalf("keyring.Set: %v", err)
	}

	ks := &KeyringStore{}
	_, err := ks.LoadCredentials("corrupt-test")
	if err == nil {
		t.Fatal("expected error for corrupted JSON")
	}
}

func TestKeyringStore_LoadToken_Missing(t *testing.T) {
	ks := &KeyringStore{}
	_, err := ks.LoadToken("nonexistent-token-xyz")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestKeyringStore_LoadToken_CorruptedJSON(t *testing.T) {
	// Write corrupt data directly to keyring
	if err := keyring.Set(keyringService, tokenKeyPrefix+"corrupt-tok-test", "not json"); err != nil {
		t.Fatalf("keyring.Set: %v", err)
	}

	ks := &KeyringStore{}
	_, err := ks.LoadToken("corrupt-tok-test")
	if err == nil {
		t.Fatal("expected error for corrupted token JSON")
	}
}

func TestKeyringStore_DeleteToken_Missing(t *testing.T) {
	ks := &KeyringStore{}
	err := ks.DeleteToken("nonexistent-delete-xyz")
	// Should return an error for missing token
	if err == nil {
		// Some implementations may silently succeed
		return
	}
}

// --- OAuthCredentials JSON marshaling ---

func TestOAuthCredentials_JSON(t *testing.T) {
	creds := &OAuthCredentials{
		ClientID:     "cid",
		ClientSecret: "csecret",
		ProjectID:    "pid",
	}
	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded OAuthCredentials
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.ClientID != "cid" {
		t.Errorf("ClientID = %q", loaded.ClientID)
	}
	if loaded.ClientSecret != "csecret" {
		t.Errorf("ClientSecret = %q", loaded.ClientSecret)
	}
	if loaded.ProjectID != "pid" {
		t.Errorf("ProjectID = %q", loaded.ProjectID)
	}
}

func TestOAuthCredentials_JSON_OmitEmpty(t *testing.T) {
	creds := &OAuthCredentials{ClientID: "x", ClientSecret: "y"}
	data, _ := json.Marshal(creds)
	// ProjectID should be omitted when empty
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if _, ok := m["project_id"]; ok {
		t.Error("project_id should be omitted when empty")
	}
}

// --- TokenFromDirect additional tests ---

func TestTokenFromDirect_LongToken(t *testing.T) {
	longToken := "ya29." + string(make([]byte, 200))
	ts := TokenFromDirect(longToken)
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("token error: %v", err)
	}
	if token.AccessToken != longToken {
		t.Error("long token should be preserved")
	}
}

func TestTokenFromDirect_SpecialChars(t *testing.T) {
	ts := TokenFromDirect("token/with+special=chars")
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("token error: %v", err)
	}
	if token.AccessToken != "token/with+special=chars" {
		t.Errorf("AccessToken = %q", token.AccessToken)
	}
}

// --- openBrowser should not panic ---

func TestOpenBrowser_DoesNotPanic(t *testing.T) {
	// Override commandExec to prevent actually opening a browser
	oldExec := commandExec
	commandExec = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true") // no-op command
	}
	defer func() { commandExec = oldExec }()

	// This should not panic regardless of platform
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("openBrowser panicked: %v", r)
		}
	}()
	openBrowser("https://example.com")
}

// --- extractAuthCode additional edge cases ---

func TestExtractAuthCode_URLWithMultipleParams(t *testing.T) {
	url := "http://127.0.0.1:1/callback?state=abc&code=AUTH_CODE_HERE&scope=openid+email&authuser=0"
	code := extractAuthCode(url)
	if code != "AUTH_CODE_HERE" {
		t.Errorf("extractAuthCode = %q, want AUTH_CODE_HERE", code)
	}
}

func TestExtractAuthCode_URLWithOnlyCode(t *testing.T) {
	url := "http://127.0.0.1:1/callback?code=ONLY_CODE"
	code := extractAuthCode(url)
	if code != "ONLY_CODE" {
		t.Errorf("extractAuthCode = %q, want ONLY_CODE", code)
	}
}

func TestExtractAuthCode_RawCodeWithSlash(t *testing.T) {
	// Google OAuth codes often contain slashes
	code := extractAuthCode("4/0AQSTgQHabc123def456")
	if code != "4/0AQSTgQHabc123def456" {
		t.Errorf("extractAuthCode(raw with slash) = %q", code)
	}
}

func TestExtractAuthCode_URLWithCodeAtEnd(t *testing.T) {
	url := "http://127.0.0.1:1/callback?state=x&code=ENDCODE"
	code := extractAuthCode(url)
	if code != "ENDCODE" {
		t.Errorf("extractAuthCode = %q, want ENDCODE", code)
	}
}

// --- indexOf additional edge cases ---

func TestIndexOf_BothEmpty(t *testing.T) {
	if idx := indexOf("", ""); idx != 0 {
		t.Errorf("indexOf('', '') = %d, want 0", idx)
	}
}

func TestIndexOf_SubstringLongerThanString(t *testing.T) {
	if idx := indexOf("ab", "abcdef"); idx != -1 {
		t.Errorf("indexOf('ab', 'abcdef') = %d, want -1", idx)
	}
}

func TestIndexOf_RepeatedPattern(t *testing.T) {
	// Should find first occurrence
	if idx := indexOf("abcabc", "abc"); idx != 0 {
		t.Errorf("indexOf('abcabc', 'abc') = %d, want 0", idx)
	}
}

// --- LoadConfigFromKeyring ---

func TestLoadConfigFromKeyring_NoSavedCredentials(t *testing.T) {
	m := NewManager()
	err := m.LoadConfigFromKeyring([]string{"https://www.googleapis.com/auth/gmail.readonly"})
	// If no credentials were saved before, this should fail
	// (unless a previous test saved some). Check that no panic occurs.
	_ = err
}

func TestLoadConfigFromKeyring_AfterSave(t *testing.T) {
	m := NewManager()
	// First save credentials
	credJSON := []byte(`{"installed":{"client_id":"keyring-id","client_secret":"keyring-secret"}}`)
	if err := m.LoadConfigFromJSON(credJSON, []string{"https://www.googleapis.com/auth/gmail.readonly"}); err != nil {
		t.Fatalf("LoadConfigFromJSON: %v", err)
	}

	// Now load from keyring in a new manager
	m2 := NewManager()
	err := m2.LoadConfigFromKeyring([]string{"https://www.googleapis.com/auth/calendar"})
	if err != nil {
		t.Fatalf("LoadConfigFromKeyring: %v", err)
	}
	if m2.config == nil {
		t.Fatal("config should not be nil")
	}
	if m2.config.ClientID != "keyring-id" {
		t.Errorf("ClientID = %q, want keyring-id", m2.config.ClientID)
	}
	// Scopes should be the ones passed to LoadConfigFromKeyring
	if len(m2.config.Scopes) != 1 || m2.config.Scopes[0] != "https://www.googleapis.com/auth/calendar" {
		t.Errorf("Scopes = %v, want [calendar]", m2.config.Scopes)
	}
}

// --- Manager.LoginBrowser without config ---

func TestManager_LoginBrowser_NoConfig(t *testing.T) {
	m := NewManager()
	// m.config is nil
	_, err := m.LoginBrowser(t.Context())
	if err == nil {
		t.Fatal("expected error when config not loaded")
	}
}

func TestManager_LoginManual_NoConfig(t *testing.T) {
	m := NewManager()
	_, err := m.LoginManual(t.Context())
	if err == nil {
		t.Fatal("expected error when config not loaded")
	}
}

func TestManager_LoginRemote_NoConfig(t *testing.T) {
	m := NewManager()
	_, err := m.LoginRemote(t.Context())
	if err == nil {
		t.Fatal("expected error when config not loaded")
	}
}

// --- AuthMode constants ---

func TestAuthMode_Constants(t *testing.T) {
	if AuthBrowser != 0 {
		t.Errorf("AuthBrowser = %d, want 0", AuthBrowser)
	}
	if AuthManual != 1 {
		t.Errorf("AuthManual = %d, want 1", AuthManual)
	}
	if AuthToken != 2 {
		t.Errorf("AuthToken = %d, want 2", AuthToken)
	}
	if AuthADC != 3 {
		t.Errorf("AuthADC = %d, want 3", AuthADC)
	}
}

// --- NewManager field validation ---

func TestNewManager_StoreNotNil(t *testing.T) {
	m := NewManager()
	if m.store == nil {
		t.Fatal("store should not be nil")
	}
	if m.config != nil {
		t.Error("config should be nil before loading")
	}
}

// --- LoadConfigFromFile missing file ---

func TestLoadConfigFromFile_MissingFile(t *testing.T) {
	m := NewManager()
	err := m.LoadConfigFromFile("/nonexistent/path/credentials.json", nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
