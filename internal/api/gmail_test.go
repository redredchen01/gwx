package api

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRawMessage_PlainText(t *testing.T) {
	input := &SendInput{
		To:      []string{"alice@example.com"},
		Subject: "Test Subject",
		Body:    "Hello, world!",
	}

	raw, err := buildRawMessage(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	decoded, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("invalid base64url: %v", err)
	}

	msg := string(decoded)
	if !strings.Contains(msg, "To: alice@example.com") {
		t.Fatalf("missing To header: %s", msg)
	}
	if !strings.Contains(msg, "Subject: Test Subject") {
		t.Fatalf("missing Subject header: %s", msg)
	}
	if !strings.Contains(msg, "Hello, world!") {
		t.Fatalf("missing body: %s", msg)
	}
	if !strings.Contains(msg, "text/plain") {
		t.Fatalf("missing content-type: %s", msg)
	}
}

func TestBuildRawMessage_WithCC(t *testing.T) {
	input := &SendInput{
		To:      []string{"alice@example.com", "bob@example.com"},
		CC:      []string{"charlie@example.com"},
		BCC:     []string{"secret@example.com"},
		Subject: "Multi-recipient",
		Body:    "Test",
	}

	raw, err := buildRawMessage(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	decoded, _ := base64.URLEncoding.DecodeString(raw)
	msg := string(decoded)

	if !strings.Contains(msg, "alice@example.com, bob@example.com") {
		t.Fatalf("To should contain both recipients: %s", msg)
	}
	if !strings.Contains(msg, "Cc: charlie@example.com") {
		t.Fatalf("missing CC: %s", msg)
	}
	if !strings.Contains(msg, "Bcc: secret@example.com") {
		t.Fatalf("missing BCC: %s", msg)
	}
}

func TestBuildRawMessage_HTMLAlternative(t *testing.T) {
	input := &SendInput{
		To:       []string{"alice@example.com"},
		Subject:  "HTML Test",
		Body:     "Plain text version",
		BodyHTML: "<h1>HTML version</h1>",
	}

	raw, err := buildRawMessage(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	decoded, _ := base64.URLEncoding.DecodeString(raw)
	msg := string(decoded)

	if !strings.Contains(msg, "multipart/alternative") {
		t.Fatalf("should be multipart/alternative: %s", msg)
	}
	if !strings.Contains(msg, "Plain text version") {
		t.Fatalf("missing plain text part: %s", msg)
	}
	if !strings.Contains(msg, "<h1>HTML version</h1>") {
		t.Fatalf("missing HTML part: %s", msg)
	}
}

func TestBuildMultipartMessage_WithAttachment(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("attachment content"), 0644); err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	input := &SendInput{
		To:          []string{"alice@example.com"},
		Subject:     "With Attachment",
		Body:        "See attached",
		Attachments: []string{tmpFile},
	}

	raw, err := buildRawMessage(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	decoded, _ := base64.URLEncoding.DecodeString(raw)
	msg := string(decoded)

	if !strings.Contains(msg, "multipart/mixed") {
		t.Fatalf("should be multipart/mixed: %s", msg)
	}
	if !strings.Contains(msg, "test.txt") {
		t.Fatalf("should contain attachment filename: %s", msg)
	}
	if !strings.Contains(msg, "See attached") {
		t.Fatalf("should contain body: %s", msg)
	}
}

func TestBuildRawMessage_EmptyBody(t *testing.T) {
	input := &SendInput{
		To:      []string{"alice@example.com"},
		Subject: "Empty",
		Body:    "",
	}

	raw, err := buildRawMessage(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == "" {
		t.Fatal("raw should not be empty even with empty body")
	}
}

func TestBuildMultipartMessage_MissingAttachment(t *testing.T) {
	input := &SendInput{
		To:          []string{"alice@example.com"},
		Subject:     "Missing",
		Body:        "Test",
		Attachments: []string{"/nonexistent/file.txt"},
	}

	_, err := buildRawMessage(input)
	if err == nil {
		t.Fatal("expected error for missing attachment")
	}
}

func TestBuildMultipartMessage_OversizedAttachment(t *testing.T) {
	// Create a file that reports >25MB via Stat
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "big.bin")
	// Write just enough to trigger the size check
	f, _ := os.Create(tmpFile)
	f.Truncate(26 * 1024 * 1024) // 26MB sparse file
	f.Close()

	input := &SendInput{
		To:          []string{"alice@example.com"},
		Subject:     "Too Big",
		Body:        "Test",
		Attachments: []string{tmpFile},
	}

	_, err := buildRawMessage(input)
	if err == nil {
		t.Fatal("expected error for oversized attachment")
	}
	if !strings.Contains(err.Error(), "25MB") {
		t.Fatalf("error should mention 25MB limit, got: %v", err)
	}
}

func TestBuildRawMessage_RandomBoundary(t *testing.T) {
	input := &SendInput{
		To:       []string{"alice@example.com"},
		Subject:  "Boundary Test",
		Body:     "Plain",
		BodyHTML: "<b>HTML</b>",
	}

	raw1, _ := buildRawMessage(input)
	raw2, _ := buildRawMessage(input)

	// Two messages should have different boundaries (random)
	if raw1 == raw2 {
		t.Fatal("boundary should be random, but two messages are identical")
	}
}

func TestGenerateBoundary_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		b := generateBoundary()
		if seen[b] {
			t.Fatalf("duplicate boundary generated: %s", b)
		}
		seen[b] = true
	}
}

func TestParseValuesJSON_Valid(t *testing.T) {
	values, err := ParseValuesJSON(`[["a",1],["b",2]]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(values))
	}
}

func TestParseValuesJSON_Invalid(t *testing.T) {
	_, err := ParseValuesJSON(`not json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseValuesJSON_Empty(t *testing.T) {
	values, err := ParseValuesJSON(`[]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(values))
	}
}
