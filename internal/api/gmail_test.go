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
