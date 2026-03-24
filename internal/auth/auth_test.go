package auth

import (
	"testing"
)

// --- extractAuthCode ---

func TestExtractAuthCode_FullURL(t *testing.T) {
	url := "http://127.0.0.1:1/callback?state=abc123&code=4/0AbcDef_GhiJkl&scope=openid"
	code := extractAuthCode(url)
	if code != "4/0AbcDef_GhiJkl" {
		t.Errorf("extractAuthCode(url) = %q, want '4/0AbcDef_GhiJkl'", code)
	}
}

func TestExtractAuthCode_URLWithoutScope(t *testing.T) {
	url := "http://127.0.0.1:1/callback?code=MYCODE123"
	code := extractAuthCode(url)
	if code != "MYCODE123" {
		t.Errorf("extractAuthCode(url) = %q, want 'MYCODE123'", code)
	}
}

func TestExtractAuthCode_RawCode(t *testing.T) {
	code := extractAuthCode("4/0AbcDef_GhiJkl")
	if code != "4/0AbcDef_GhiJkl" {
		t.Errorf("extractAuthCode(raw) = %q, want '4/0AbcDef_GhiJkl'", code)
	}
}

func TestExtractAuthCode_Empty(t *testing.T) {
	code := extractAuthCode("")
	if code != "" {
		t.Errorf("extractAuthCode('') = %q, want ''", code)
	}
}

func TestExtractAuthCode_CodeWithSpaces(t *testing.T) {
	// Code followed by whitespace
	url := "http://127.0.0.1:1/callback?code=MYCODE stuff"
	code := extractAuthCode(url)
	if code != "MYCODE" {
		t.Errorf("extractAuthCode(url with space) = %q, want 'MYCODE'", code)
	}
}

func TestExtractAuthCode_CodeWithNewline(t *testing.T) {
	input := "http://127.0.0.1:1/callback?code=CODE123\n"
	code := extractAuthCode(input)
	if code != "CODE123" {
		t.Errorf("extractAuthCode(with newline) = %q, want 'CODE123'", code)
	}
}

func TestExtractAuthCode_CodeWithCarriageReturn(t *testing.T) {
	input := "http://127.0.0.1:1/callback?code=CODE456\r\n"
	code := extractAuthCode(input)
	if code != "CODE456" {
		t.Errorf("extractAuthCode(with CR) = %q, want 'CODE456'", code)
	}
}

// --- indexOf ---

func TestIndexOf_Found(t *testing.T) {
	if idx := indexOf("hello world", "world"); idx != 6 {
		t.Errorf("indexOf = %d, want 6", idx)
	}
}

func TestIndexOf_NotFound(t *testing.T) {
	if idx := indexOf("hello", "xyz"); idx != -1 {
		t.Errorf("indexOf = %d, want -1", idx)
	}
}

func TestIndexOf_AtStart(t *testing.T) {
	if idx := indexOf("hello", "hell"); idx != 0 {
		t.Errorf("indexOf = %d, want 0", idx)
	}
}

func TestIndexOf_AtEnd(t *testing.T) {
	if idx := indexOf("hello", "llo"); idx != 2 {
		t.Errorf("indexOf = %d, want 2", idx)
	}
}

func TestIndexOf_EmptySubstring(t *testing.T) {
	if idx := indexOf("hello", ""); idx != 0 {
		t.Errorf("indexOf(hello, '') = %d, want 0", idx)
	}
}

func TestIndexOf_EmptyString(t *testing.T) {
	if idx := indexOf("", "hello"); idx != -1 {
		t.Errorf("indexOf('', hello) = %d, want -1", idx)
	}
}

func TestIndexOf_Equal(t *testing.T) {
	if idx := indexOf("hello", "hello"); idx != 0 {
		t.Errorf("indexOf(hello, hello) = %d, want 0", idx)
	}
}

// --- AllScopes additional tests ---

func TestAllScopes_EmptyServices(t *testing.T) {
	scopes := AllScopes([]string{}, false)
	if len(scopes) != 0 {
		t.Fatalf("empty services should return empty scopes, got %v", scopes)
	}
}

func TestAllScopes_NilServices(t *testing.T) {
	scopes := AllScopes(nil, false)
	if len(scopes) != 0 {
		t.Fatalf("nil services should return empty scopes, got %v", scopes)
	}
}

func TestAllScopes_AllServicesInMap(t *testing.T) {
	// Verify all 13 services defined in ServiceScopes
	allServices := []string{
		"gmail", "calendar", "drive", "docs", "sheets", "tasks",
		"people", "chat", "analytics", "searchconsole", "slides",
		"forms", "bigquery",
	}
	for _, svc := range allServices {
		scopes := AllScopes([]string{svc}, false)
		if len(scopes) == 0 {
			t.Errorf("service %q has no scopes defined", svc)
		}
	}
}

func TestAllScopes_AllServicesReadOnly(t *testing.T) {
	allServices := []string{
		"gmail", "calendar", "drive", "docs", "sheets", "tasks",
		"people", "chat", "analytics", "searchconsole", "slides",
		"forms", "bigquery",
	}
	for _, svc := range allServices {
		scopes := AllScopes([]string{svc}, true)
		if len(scopes) == 0 {
			t.Errorf("service %q has no readonly scopes defined", svc)
		}
	}
}

func TestAllScopes_DeduplicatesSameService(t *testing.T) {
	scopes := AllScopes([]string{"gmail", "gmail", "gmail"}, false)
	seen := make(map[string]bool)
	for _, s := range scopes {
		if seen[s] {
			t.Fatalf("duplicate scope: %s", s)
		}
		seen[s] = true
	}
}

func TestAllScopes_DeduplicatesCrossService(t *testing.T) {
	// People read-only and contacts use the same scope
	scopes := AllScopes([]string{"people", "people"}, true)
	seen := make(map[string]bool)
	for _, s := range scopes {
		if seen[s] {
			t.Fatalf("duplicate scope: %s", s)
		}
		seen[s] = true
	}
}

// --- TokenFromDirect ---

func TestTokenFromDirect_NotNil(t *testing.T) {
	ts := TokenFromDirect("test-token-123")
	if ts == nil {
		t.Fatal("TokenFromDirect should not return nil")
	}
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("token error: %v", err)
	}
	if token.AccessToken != "test-token-123" {
		t.Errorf("AccessToken = %q, want test-token-123", token.AccessToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want Bearer", token.TokenType)
	}
}

func TestTokenFromDirect_EmptyToken(t *testing.T) {
	ts := TokenFromDirect("")
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("token error: %v", err)
	}
	if token.AccessToken != "" {
		t.Errorf("AccessToken = %q, want empty", token.AccessToken)
	}
}

// --- NewManager ---

func TestNewManager_NotNil(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager should not return nil")
	}
	if m.store == nil {
		t.Error("store should not be nil")
	}
}

// --- ServiceScopes map consistency ---

func TestServiceScopes_AllHaveValues(t *testing.T) {
	for svc, scopes := range ServiceScopes {
		if len(scopes) == 0 {
			t.Errorf("ServiceScopes[%q] is empty", svc)
		}
		for _, s := range scopes {
			if s == "" {
				t.Errorf("ServiceScopes[%q] contains empty string", svc)
			}
		}
	}
}

func TestReadOnlyScopes_AllHaveValues(t *testing.T) {
	for svc, scopes := range ReadOnlyScopes {
		if len(scopes) == 0 {
			t.Errorf("ReadOnlyScopes[%q] is empty", svc)
		}
		for _, s := range scopes {
			if s == "" {
				t.Errorf("ReadOnlyScopes[%q] contains empty string", svc)
			}
		}
	}
}

func TestServiceAndReadOnlyScopes_SameKeys(t *testing.T) {
	for svc := range ServiceScopes {
		if _, ok := ReadOnlyScopes[svc]; !ok {
			t.Errorf("service %q in ServiceScopes but not in ReadOnlyScopes", svc)
		}
	}
	for svc := range ReadOnlyScopes {
		if _, ok := ServiceScopes[svc]; !ok {
			t.Errorf("service %q in ReadOnlyScopes but not in ServiceScopes", svc)
		}
	}
}
