package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// --- validateTokenImport ---

func TestValidateTokenImport_Envelope(t *testing.T) {
	export := TokenExport{
		Version:    1,
		Account:    "work",
		ExportedAt: time.Now().UTC(),
		Token: &oauth2.Token{
			AccessToken:  "ya29-test",
			TokenType:    "Bearer",
			RefreshToken: "1//refresh",
			Expiry:       time.Now().Add(time.Hour),
		},
	}
	data, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	tok, account, err := validateTokenImport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "ya29-test" {
		t.Errorf("access_token = %q, want ya29-test", tok.AccessToken)
	}
	if tok.RefreshToken != "1//refresh" {
		t.Errorf("refresh_token = %q, want 1//refresh", tok.RefreshToken)
	}
	if account != "work" {
		t.Errorf("account = %q, want work", account)
	}
}

func TestValidateTokenImport_RawToken(t *testing.T) {
	raw := `{"access_token":"ya29-raw","token_type":"Bearer","refresh_token":"1//raw","expiry":"2026-12-01T00:00:00Z"}`

	tok, account, err := validateTokenImport([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "ya29-raw" {
		t.Errorf("access_token = %q, want ya29-raw", tok.AccessToken)
	}
	if account != "" {
		t.Errorf("account = %q, want empty for raw token", account)
	}
}

func TestValidateTokenImport_RefreshTokenOnly(t *testing.T) {
	raw := `{"refresh_token":"1//only-refresh"}`
	tok, _, err := validateTokenImport([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.RefreshToken != "1//only-refresh" {
		t.Errorf("refresh_token = %q, want 1//only-refresh", tok.RefreshToken)
	}
}

func TestValidateTokenImport_EmptyTokens(t *testing.T) {
	raw := `{"token_type":"Bearer"}`
	_, _, err := validateTokenImport([]byte(raw))
	if err == nil {
		t.Fatal("expected error for token with no access_token or refresh_token")
	}
	if !strings.Contains(err.Error(), "neither") {
		t.Errorf("error %q should mention 'neither'", err)
	}
}

func TestValidateTokenImport_InvalidJSON(t *testing.T) {
	_, _, err := validateTokenImport([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateTokenImport_EnvelopeEmptyToken(t *testing.T) {
	data := `{"version":1,"account":"x","token":{"token_type":"Bearer"}}`
	_, _, err := validateTokenImport([]byte(data))
	if err == nil {
		t.Fatal("expected error for envelope with empty token")
	}
}

// --- readJSONInput ---

func TestReadJSONInput_FlagPriority(t *testing.T) {
	raw := `{"access_token":"test","refresh_token":"r"}`
	data, source, err := readJSONInput(raw, true) // isPipe=true but flag should win
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "flag" {
		t.Errorf("source = %q, want flag", source)
	}
	if string(data) != raw {
		t.Errorf("data mismatch")
	}
}

func TestReadJSONInput_FlagInvalid(t *testing.T) {
	_, _, err := readJSONInput("not-json", false)
	if err == nil {
		t.Fatal("expected error for invalid JSON flag")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error %q should mention --json", err)
	}
}

// --- TokenExport round-trip ---

func TestTokenExport_RoundTrip(t *testing.T) {
	original := TokenExport{
		Version:    1,
		Account:    "personal",
		ExportedAt: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC),
		Token: &oauth2.Token{
			AccessToken:  "ya29-round",
			TokenType:    "Bearer",
			RefreshToken: "1//round",
			Expiry:       time.Date(2026, 3, 27, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	tok, account, err := validateTokenImport(data)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if account != "personal" {
		t.Errorf("account = %q, want personal", account)
	}
	if tok.AccessToken != "ya29-round" {
		t.Errorf("access_token = %q, want ya29-round", tok.AccessToken)
	}
	if tok.RefreshToken != "1//round" {
		t.Errorf("refresh_token = %q, want 1//round", tok.RefreshToken)
	}
}
