package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redredchen01/gwx/internal/auth"
)

const testCredJSON = `{"installed":{"client_id":"test-id","client_secret":"test-secret","project_id":"proj1"}}`

// --- resolveCredentials ---

func TestResolveCredentials_JSONFlagTakesPriority(t *testing.T) {
	// --json flag should win over env vars
	t.Setenv("GWX_OAUTH_JSON", `{"installed":{"client_id":"env-id","client_secret":"s"}}`)
	cmd := &OnboardCmd{JSON: testCredJSON}

	cred, source, err := cmd.resolveCredentials(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "flag" {
		t.Errorf("source = %q, want flag", source)
	}
	if !strings.Contains(string(cred), "test-id") {
		t.Error("expected --json flag value, got env value")
	}
}

func TestResolveCredentials_JSONFlagInvalid(t *testing.T) {
	cmd := &OnboardCmd{JSON: "not-json"}
	_, _, err := cmd.resolveCredentials(false)
	if err == nil {
		t.Fatal("expected error for invalid JSON flag")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error %q should mention --json", err)
	}
}

func TestResolveCredentials_PipeReadsStdin(t *testing.T) {
	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	// Write test JSON and close writer
	w.WriteString(testCredJSON)
	w.Close()

	cmd := &OnboardCmd{}
	cred, source, resolveErr := cmd.resolveCredentials(true)
	if resolveErr != nil {
		t.Fatalf("unexpected error: %v", resolveErr)
	}
	if source != "pipe" {
		t.Errorf("source = %q, want pipe", source)
	}
	if !strings.Contains(string(cred), "test-id") {
		t.Error("expected pipe JSON content")
	}
}

func TestResolveCredentials_PipeEmpty(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	w.Close() // empty pipe

	cmd := &OnboardCmd{}
	_, _, resolveErr := cmd.resolveCredentials(true)
	if resolveErr == nil {
		t.Fatal("expected error for empty pipe")
	}
	if !strings.Contains(resolveErr.Error(), "empty") {
		t.Errorf("error %q should mention empty", resolveErr)
	}
}

func TestResolveCredentials_PipeInvalidJSON(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	w.WriteString("not json at all")
	w.Close()

	cmd := &OnboardCmd{}
	_, _, resolveErr := cmd.resolveCredentials(true)
	if resolveErr == nil {
		t.Fatal("expected error for invalid pipe JSON")
	}
	if !strings.Contains(resolveErr.Error(), "invalid JSON") {
		t.Errorf("error %q should mention invalid JSON", resolveErr)
	}
}

func TestResolveCredentials_EnvJSONFallback(t *testing.T) {
	t.Setenv("GWX_OAUTH_JSON", testCredJSON)

	cmd := &OnboardCmd{} // no flag, isPipe=false
	cred, source, err := cmd.resolveCredentials(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "env" {
		t.Errorf("source = %q, want env", source)
	}
	if !strings.Contains(string(cred), "test-id") {
		t.Error("expected env JSON content")
	}
}

func TestResolveCredentials_EnvFileFallback(t *testing.T) {
	tmp := t.TempDir()
	credFile := filepath.Join(tmp, "creds.json")
	if err := os.WriteFile(credFile, []byte(testCredJSON), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	t.Setenv("GWX_OAUTH_FILE", credFile)

	cmd := &OnboardCmd{}
	cred, source, err := cmd.resolveCredentials(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "env-file" {
		t.Errorf("source = %q, want env-file", source)
	}
	if !strings.Contains(string(cred), "test-id") {
		t.Error("expected file JSON content")
	}
}

func TestResolveCredentials_EnvFileMissing(t *testing.T) {
	t.Setenv("GWX_OAUTH_FILE", "/nonexistent/path/creds.json")

	cmd := &OnboardCmd{}
	_, _, err := cmd.resolveCredentials(false)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "GWX_OAUTH_FILE") {
		t.Errorf("error %q should mention GWX_OAUTH_FILE", err)
	}
}

func TestResolveCredentials_NoneReturnsNil(t *testing.T) {
	// No flag, no pipe, no env — should return nil
	cmd := &OnboardCmd{}
	cred, source, err := cmd.resolveCredentials(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred != nil {
		t.Errorf("expected nil cred, got %d bytes", len(cred))
	}
	if source != "" {
		t.Errorf("source = %q, want empty", source)
	}
}

func TestResolveCredentials_JSONFlagOverPipe(t *testing.T) {
	// --json flag should take priority even when isPipe is true
	// (we don't read stdin when --json is set)
	cmd := &OnboardCmd{JSON: testCredJSON}
	cred, source, err := cmd.resolveCredentials(true) // isPipe=true but --json wins
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "flag" {
		t.Errorf("source = %q, want flag", source)
	}
	if !strings.Contains(string(cred), "test-id") {
		t.Error("expected flag value")
	}
}

// --- resolveServices ---

func TestResolveServices_FlagSplitsAndTrims(t *testing.T) {
	cmd := &OnboardCmd{Services: "gmail, calendar , drive"}
	svc := cmd.resolveServices()
	if len(svc) != 3 {
		t.Fatalf("len = %d, want 3", len(svc))
	}
	if svc[0] != "gmail" || svc[1] != "calendar" || svc[2] != "drive" {
		t.Errorf("services = %v", svc)
	}
}

func TestResolveServices_EnvFallback(t *testing.T) {
	t.Setenv("GWX_SERVICES", "gmail,sheets")
	cmd := &OnboardCmd{} // no flag
	svc := cmd.resolveServices()
	if len(svc) != 2 {
		t.Fatalf("len = %d, want 2", len(svc))
	}
	if svc[0] != "gmail" || svc[1] != "sheets" {
		t.Errorf("services = %v", svc)
	}
}

func TestResolveServices_FlagOverridesEnv(t *testing.T) {
	t.Setenv("GWX_SERVICES", "gmail,sheets")
	cmd := &OnboardCmd{Services: "drive"}
	svc := cmd.resolveServices()
	if len(svc) != 1 || svc[0] != "drive" {
		t.Errorf("services = %v, want [drive]", svc)
	}
}

func TestResolveServices_NoneReturnsDefault(t *testing.T) {
	cmd := &OnboardCmd{}
	svc := cmd.resolveServices()
	if len(svc) != len(defaultServices) {
		t.Errorf("len = %d, want %d (default)", len(svc), len(defaultServices))
	}
}

// --- resolveAuthMethod ---

func TestResolveAuthMethod_FlagPriority(t *testing.T) {
	t.Setenv("GWX_AUTH_METHOD", "remote")
	cmd := &OnboardCmd{AuthMethod: "browser"}
	m := cmd.resolveAuthMethod()
	if m != "browser" {
		t.Errorf("method = %q, want browser", m)
	}
}

func TestResolveAuthMethod_EnvFallback(t *testing.T) {
	t.Setenv("GWX_AUTH_METHOD", "manual")
	cmd := &OnboardCmd{}
	m := cmd.resolveAuthMethod()
	if m != "manual" {
		t.Errorf("method = %q, want manual", m)
	}
}

func TestResolveAuthMethod_NoneReturnsEmpty(t *testing.T) {
	cmd := &OnboardCmd{}
	m := cmd.resolveAuthMethod()
	if m != "" {
		t.Errorf("method = %q, want empty", m)
	}
}

func TestResolveAuthMethod_CaseInsensitive(t *testing.T) {
	cmd := &OnboardCmd{AuthMethod: "BROWSER"}
	m := cmd.resolveAuthMethod()
	if m != "browser" {
		t.Errorf("method = %q, want browser", m)
	}
}

// --- OnboardCmd DryRun integration ---

func TestOnboardCmd_DryRunWithJSONFlag(t *testing.T) {
	rctx, buf := newDryRunContext(t)

	cmd := &OnboardCmd{
		JSON:       testCredJSON,
		AuthMethod: "browser", // avoid pipe conflict
	}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
	out := buf.String()
	if !strings.Contains(out, `"dry_run"`) {
		t.Error("output should contain dry_run")
	}
	if !strings.Contains(out, `"source":"flag"`) && !strings.Contains(out, `"source": "flag"`) {
		t.Errorf("output should contain source=flag, got: %s", out)
	}
}

func TestOnboardCmd_DryRunWithEnvJSON(t *testing.T) {
	t.Setenv("GWX_OAUTH_JSON", testCredJSON)
	rctx, buf := newDryRunContext(t)

	cmd := &OnboardCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
	out := buf.String()
	if !strings.Contains(out, `"dry_run"`) {
		t.Error("output should contain dry_run")
	}
}

func TestOnboardCmd_DryRunNoCredentialsErrors(t *testing.T) {
	rctx, _ := newDryRunContext(t)

	cmd := &OnboardCmd{}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for dry-run without credentials")
	}
	if !strings.Contains(err.Error(), "dry-run requires credentials") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOnboardCmd_DryRunWithServicesFlag(t *testing.T) {
	rctx, buf := newDryRunContext(t)

	cmd := &OnboardCmd{
		JSON:     testCredJSON,
		Services: "gmail,drive",
	}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "gmail") || !strings.Contains(out, "drive") {
		t.Errorf("output should contain gmail and drive: %s", out)
	}
	// Should NOT contain all default services
	if strings.Contains(out, "searchconsole") {
		t.Error("output should not contain searchconsole when --services is specified")
	}
}

func TestOnboardCmd_InvalidAuthMethod(t *testing.T) {
	cmd := &OnboardCmd{AuthMethod: "invalid"}
	m := cmd.resolveAuthMethod()
	switch m {
	case "browser", "b", "manual", "m", "remote", "r", "":
		t.Error("should not be valid")
	default:
		// expected: invalid method
	}
}

// --- SA-D6-007 (P0): pipe + DryRun Run() integration ---

func TestOnboardCmd_DryRunWithPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	w.WriteString(testCredJSON)
	w.Close()

	rctx, buf := newDryRunContext(t)
	cmd := &OnboardCmd{AuthMethod: "browser"} // explicit auth to avoid pipe+remote conflict
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"dry_run"`) {
		t.Error("output should contain dry_run")
	}
	if !strings.Contains(out, `"pipe"`) {
		t.Errorf("output should contain source=pipe, got: %s", out)
	}
}

// --- SA-D6-008 (P0): pipe + remote conflict ---

func TestOnboardCmd_PipeRemoteConflict(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	w.WriteString(testCredJSON)
	w.Close()

	rctx, _ := newTestRunContext(t)
	rctx.Auth = auth.NewManager()
	// No AuthMethod → defaults to "remote" for non-interactive → conflict with pipe
	cmd := &OnboardCmd{}
	runErr := cmd.Run(rctx)
	if runErr == nil {
		t.Fatal("expected error for pipe + remote conflict")
	}
	errMsg := runErr.Error()
	if !strings.Contains(errMsg, "pipe mode conflicts") {
		t.Errorf("error should mention pipe conflict, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "browser") || !strings.Contains(errMsg, "manual") {
		t.Errorf("error should suggest alternatives, got: %s", errMsg)
	}
}

// --- SA-D6-009 (P1): invalid --auth-method at Run() level ---

func TestOnboardCmd_InvalidAuthMethodRunLevel(t *testing.T) {
	rctx, _ := newDryRunContext(t)
	// DryRun stops before auth method check, so use non-DryRun with real auth
	rctx.DryRun = false
	rctx.Auth = auth.NewManager()

	cmd := &OnboardCmd{JSON: testCredJSON, AuthMethod: "invalid"}
	runErr := cmd.Run(rctx)
	if runErr == nil {
		t.Fatal("expected error for invalid auth method")
	}
	if !strings.Contains(runErr.Error(), "invalid --auth-method") {
		t.Errorf("error should mention invalid auth method, got: %v", runErr)
	}
}

// --- SA-D6-005 (P2): --json flag priority over pipe at Run() level ---

func TestOnboardCmd_JSONFlagPriorityOverPipeRunLevel(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	w.WriteString("garbage that would fail json.Valid")
	w.Close()

	rctx, buf := newDryRunContext(t)
	// --json flag should win, pipe garbage should be ignored
	cmd := &OnboardCmd{JSON: testCredJSON}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"flag"`) {
		t.Errorf("source should be flag, got: %s", out)
	}
}
