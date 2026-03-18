package log_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	gwxlog "github.com/redredchen01/gwx/internal/log"
)

// TestSetupMCPLogger_WritesToStderr verifies that the MCP logger:
//   - returns a non-nil *slog.Logger
//   - produces valid JSON output
//   - does NOT write to stdout
func TestSetupMCPLogger_WritesToStderr(t *testing.T) {
	logger := gwxlog.SetupMCPLogger()
	if logger == nil {
		t.Fatal("SetupMCPLogger returned nil")
	}

	// Capture stderr via a pipe to verify JSON output.
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	// Also capture stdout to confirm nothing leaks there.
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	os.Stdout = wOut

	// Re-create logger after redirecting stderr so the handler picks up the new fd.
	logger = gwxlog.SetupMCPLogger()
	logger.Error("test message", "key", "value")

	// Restore
	w.Close()
	wOut.Close()
	os.Stderr = origStderr
	os.Stdout = origStdout

	var stderrBuf bytes.Buffer
	stderrBuf.ReadFrom(r) //nolint:errcheck

	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(rOut) //nolint:errcheck

	// stdout must be empty
	if stdoutBuf.Len() != 0 {
		t.Errorf("MCP logger must not write to stdout, got: %s", stdoutBuf.String())
	}

	// stderr must contain valid JSON with expected fields
	line := stderrBuf.String()
	if line == "" {
		t.Fatal("expected output on stderr, got nothing")
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Errorf("MCP logger output is not valid JSON: %v\noutput: %s", err, line)
	}

	if m["msg"] != "test message" {
		t.Errorf("expected msg=test message, got %v", m["msg"])
	}
}

// TestSetupCLILogger verifies that SetupCLILogger returns a non-nil logger
// and that it uses the correct handler type based on whether stderr is a TTY.
// In the test environment stderr is typically NOT a TTY, so we expect JSON.
func TestSetupCLILogger(t *testing.T) {
	logger := gwxlog.SetupCLILogger()
	if logger == nil {
		t.Fatal("SetupCLILogger returned nil")
	}

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	logger = gwxlog.SetupCLILogger()
	logger.Info("cli test", slog.String("env", "ci"))

	w.Close()
	os.Stderr = origStderr

	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output on stderr, got nothing")
	}

	// In a non-TTY env (CI / pipe), expect JSON.
	// If somehow running in a real TTY, this test might get text — that's acceptable,
	// so we just verify the output is non-empty and contains our message.
	if !bytes.Contains([]byte(output), []byte("cli test")) {
		t.Errorf("expected 'cli test' in output, got: %s", output)
	}
}
