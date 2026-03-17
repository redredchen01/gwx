package cmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Binary path — built by TestMain
var binaryPath string

func TestMain(m *testing.M) {
	// Build binary for integration tests
	tmpDir, err := os.MkdirTemp("", "gwx-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = tmpDir + "/gwx"
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/gwx/")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + err.Error() + "\n" + string(out))
	}

	os.Exit(m.Run())
}

// cleanEnv returns an environment that isolates from the real OS keyring.
func cleanEnv(t *testing.T, extra ...string) []string {
	t.Helper()
	env := []string{
		"HOME=" + t.TempDir(),
		"PATH=" + os.Getenv("PATH"),
		"GWX_AUTO_JSON=1",
	}
	return append(env, extra...)
}

func runGWX(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = cleanEnv(t)

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return outBuf.String(), errBuf.String(), exitCode
}

func TestCLI_Version(t *testing.T) {
	stdout, _, code := runGWX(t, "version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, stdout)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", resp["status"])
	}
	data := resp["data"].(map[string]interface{})
	if data["version"] != "0.6.0" {
		t.Fatalf("expected version 0.6.0, got %v", data["version"])
	}
}

func TestCLI_AgentExitCodes(t *testing.T) {
	stdout, _, code := runGWX(t, "agent", "exit-codes")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	if data["10"] != "auth_required" {
		t.Fatalf("expected code 10=auth_required, got %v", data["10"])
	}
}

func TestCLI_AuthStatus_Unauthenticated(t *testing.T) {
	_, stderr, code := runGWX(t, "auth", "status")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(stderr), &resp); err != nil {
		t.Fatalf("invalid JSON in stderr: %v\nraw: %s", err, stderr)
	}
	if resp["status"] != "error" {
		t.Fatalf("expected error status, got %v", resp["status"])
	}
	errObj := resp["error"].(map[string]interface{})
	if errObj["name"] != "auth_required" {
		t.Fatalf("expected auth_required, got %v", errObj["name"])
	}
}

func TestCLI_GmailList_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "gmail", "list")
	if code != 10 {
		t.Fatalf("expected exit 10 (auth_required), got %d", code)
	}
}

func TestCLI_GmailSearch_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "gmail", "search", "test query")
	if code != 10 {
		t.Fatalf("expected exit 10 (auth_required), got %d", code)
	}
}

func TestCLI_GmailGet_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "gmail", "get", "msg123")
	if code != 10 {
		t.Fatalf("expected exit 10 (auth_required), got %d", code)
	}
}

func TestCLI_Allowlist_Denied(t *testing.T) {
	cmd := exec.Command(binaryPath, "gmail", "list")
	cmd.Env = cleanEnv(t, "GWX_ENABLE_COMMANDS=calendar.*")

	var errBuf strings.Builder
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	if exitCode != 12 {
		t.Fatalf("expected exit 12 (permission_denied), got %d\nstderr: %s", exitCode, errBuf.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(errBuf.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	errObj := resp["error"].(map[string]interface{})
	if errObj["name"] != "permission_denied" {
		t.Fatalf("expected permission_denied, got %v", errObj["name"])
	}
}

func TestCLI_Allowlist_Allowed(t *testing.T) {
	// gmail.list is allowed, but still fails on auth (exit 10, not 12)
	cmd := exec.Command(binaryPath, "gmail", "list")
	cmd.Env = cleanEnv(t, "GWX_ENABLE_COMMANDS=gmail.*")

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	// Should pass allowlist (exit 10 = auth, not 12 = permission)
	if exitCode != 10 {
		t.Fatalf("expected exit 10 (auth, passed allowlist), got %d", exitCode)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	cmd := exec.Command(binaryPath, "nonexistent")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	// Kong exits with code 1 for unknown commands
	if exitErr.ExitCode() == 0 {
		t.Fatal("expected non-zero exit for unknown command")
	}
}

func TestCLI_Help(t *testing.T) {
	cmd := exec.Command(binaryPath, "--help")
	out, err := cmd.CombinedOutput()
	// --help exits with 0
	if err != nil {
		// Kong may exit with 0 or 1 for help, just check output
	}
	output := string(out)
	if !strings.Contains(output, "Google Workspace CLI") {
		t.Fatalf("help should contain description, got: %s", output)
	}
	if !strings.Contains(output, "gmail") {
		t.Fatalf("help should list gmail command, got: %s", output)
	}
}

func TestCLI_GmailHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "gmail", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !strings.Contains(output, "list") {
		t.Fatalf("gmail help should list 'list' subcommand, got: %s", output)
	}
	if !strings.Contains(output, "search") {
		t.Fatalf("gmail help should list 'search' subcommand, got: %s", output)
	}
}

func TestCLI_OutputFormat_JSON(t *testing.T) {
	stdout, _, code := runGWX(t, "--format", "json", "version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("--format json should produce valid JSON: %v", err)
	}
}

func TestCLI_OutputFormat_Plain(t *testing.T) {
	stdout, _, code := runGWX(t, "--format", "plain", "version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if len(stdout) == 0 {
		t.Fatal("plain output should not be empty")
	}
}

// --- Phase 2: Calendar tests ---

func TestCLI_CalendarAgenda_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "calendar", "agenda")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_CalendarList_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "calendar", "list", "--from", "2026-03-17", "--to", "2026-03-18")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_CalendarCreate_DryRun(t *testing.T) {
	// DryRun still requires auth first — expect 10
	_, _, code := runGWX(t, "calendar", "create", "--title", "Test", "--start", "2026-03-17T10:00:00Z", "--end", "2026-03-17T11:00:00Z", "--dry-run")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_CalendarHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "calendar", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, sub := range []string{"agenda", "list", "create", "update", "delete", "find-slot"} {
		if !strings.Contains(output, sub) {
			t.Errorf("calendar help should list %q subcommand", sub)
		}
	}
}

func TestCLI_CalendarAllowlist(t *testing.T) {
	cmd := exec.Command(binaryPath, "calendar", "agenda")
	cmd.Env = cleanEnv(t, "GWX_ENABLE_COMMANDS=gmail.*")
	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	if exitCode != 12 {
		t.Fatalf("expected exit 12 (permission_denied), got %d", exitCode)
	}
}

// --- Phase 2: Drive tests ---

func TestCLI_DriveList_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "drive", "list")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_DriveSearch_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "drive", "search", "name contains 'test'")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_DriveHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "drive", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, sub := range []string{"list", "search", "upload", "download", "share", "mkdir"} {
		if !strings.Contains(output, sub) {
			t.Errorf("drive help should list %q subcommand", sub)
		}
	}
}

func TestCLI_DriveAllowlist(t *testing.T) {
	cmd := exec.Command(binaryPath, "drive", "list")
	cmd.Env = cleanEnv(t, "GWX_ENABLE_COMMANDS=gmail.*")
	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	if exitCode != 12 {
		t.Fatalf("expected exit 12 (permission_denied), got %d", exitCode)
	}
}

// --- Phase 2: Gmail write tests ---

func TestCLI_GmailSend_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "gmail", "send", "--to", "a@b.com", "--subject", "test", "--body", "hi")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_GmailDraft_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "gmail", "draft", "--to", "a@b.com", "--subject", "test", "--body", "hi")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_GmailReply_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "gmail", "reply", "msg123", "--body", "ok")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_GmailSendHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "gmail", "send", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !strings.Contains(output, "--to") {
		t.Fatal("gmail send help should show --to flag")
	}
	if !strings.Contains(output, "--subject") {
		t.Fatal("gmail send help should show --subject flag")
	}
}

func TestCLI_HelpShowsAllServices(t *testing.T) {
	cmd := exec.Command(binaryPath, "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, svc := range []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "contacts", "chat"} {
		if !strings.Contains(output, svc) {
			t.Errorf("top-level help should list %q", svc)
		}
	}
}

// --- Phase 3: Docs tests ---

func TestCLI_DocsGet_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "docs", "get", "doc123")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_DocsCreate_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "docs", "create", "--title", "Test Doc")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_DocsAppend_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "docs", "append", "doc123", "--text", "hello")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_DocsExport_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "docs", "export", "doc123")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_DocsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "docs", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, sub := range []string{"get", "create", "append", "export"} {
		if !strings.Contains(output, sub) {
			t.Errorf("docs help should list %q subcommand", sub)
		}
	}
}

// --- Phase 3: Sheets tests ---

func TestCLI_SheetsRead_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "sheets", "read", "sheet123", "A1:B10")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_SheetsAppend_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "sheets", "append", "sheet123", "A:B", "--values", `[["a",1]]`)
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_SheetsCreate_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "sheets", "create", "--title", "Test Sheet")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_SheetsAppend_InvalidJSON(t *testing.T) {
	// Auth check runs before JSON parse, so without auth we get exit 10.
	// This test validates the command accepts the --values flag (no parse error from Kong).
	_, _, code := runGWX(t, "sheets", "append", "sheet123", "A:B", "--values", `not json`)
	if code != 10 {
		t.Fatalf("expected exit 10 (auth before JSON parse), got %d", code)
	}
}

func TestCLI_SheetsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "sheets", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, sub := range []string{"read", "append", "update", "create"} {
		if !strings.Contains(output, sub) {
			t.Errorf("sheets help should list %q subcommand", sub)
		}
	}
}

// --- Phase 3: Tasks tests ---

func TestCLI_TasksList_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "tasks", "list")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_TasksCreate_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "tasks", "create", "--title", "Buy milk")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_TasksComplete_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "tasks", "complete", "task123")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_TasksHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "tasks", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, sub := range []string{"list", "lists", "create", "complete", "delete"} {
		if !strings.Contains(output, sub) {
			t.Errorf("tasks help should list %q subcommand", sub)
		}
	}
}

// --- Phase 3: Contacts tests ---

func TestCLI_ContactsList_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "contacts", "list")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_ContactsSearch_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "contacts", "search", "john")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_ContactsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "contacts", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, sub := range []string{"list", "search", "get"} {
		if !strings.Contains(output, sub) {
			t.Errorf("contacts help should list %q subcommand", sub)
		}
	}
}

// --- Phase 3: Chat tests ---

func TestCLI_ChatSpaces_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "chat", "spaces")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_ChatSend_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "chat", "send", "spaces/AAA", "--text", "hello")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_ChatHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "chat", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, sub := range []string{"spaces", "send", "messages"} {
		if !strings.Contains(output, sub) {
			t.Errorf("chat help should list %q subcommand", sub)
		}
	}
}

// --- Cross-cutting: Allowlist for all services ---

func TestCLI_AllServicesAllowlistDenied(t *testing.T) {
	commands := []struct {
		args []string
	}{
		{[]string{"docs", "get", "x"}},
		{[]string{"sheets", "read", "x", "A1"}},
		{[]string{"tasks", "list"}},
		{[]string{"contacts", "list"}},
		{[]string{"chat", "spaces"}},
	}
	for _, tc := range commands {
		cmd := exec.Command(binaryPath, tc.args...)
		cmd.Env = cleanEnv(t, "GWX_ENABLE_COMMANDS=gmail.list")
		err := cmd.Run()
		exitCode := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		if exitCode != 12 {
			t.Errorf("allowlist should deny %v, got exit %d", tc.args, exitCode)
		}
	}
}
