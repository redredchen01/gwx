package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/output"
)

// newTestRunContext creates a RunContext suitable for in-process testing.
func newTestRunContext(t *testing.T) (*RunContext, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	printer := &output.Printer{Format: output.FormatJSON, Writer: &buf}
	return &RunContext{
		Context:   context.Background(),
		Printer:   printer,
		Account:   "default",
		DryRun:    false,
		NoInput:   true,
		Allowlist: nil, // nil = no restriction
	}, &buf
}

// --- VersionCmd ---

func TestVersionCmd_Run(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	cmd := &VersionCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp output.Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	raw, _ := json.Marshal(resp.Data)
	var data map[string]interface{}
	json.Unmarshal(raw, &data)
	if data["version"] != version {
		t.Errorf("version = %v, want %s", data["version"], version)
	}
	if data["name"] != "gwx" {
		t.Errorf("name = %v, want gwx", data["name"])
	}
}

// --- AgentExitCodesCmd ---

func TestAgentExitCodesCmd_Run(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	cmd := &AgentExitCodesCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp output.Response
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	raw, _ := json.Marshal(resp.Data)
	var data map[string]interface{}
	json.Unmarshal(raw, &data)
	// The exit code map keys are ints, but JSON marshals map[int]string to {"10":"auth_required",...}
	if len(data) < 10 {
		t.Errorf("expected at least 10 exit codes, got %d", len(data))
	}
}

// --- SchemaCmd ---

func TestSchemaCmd_Run(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	cmd := &SchemaCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("schema should produce valid JSON: %v", err)
	}
	if _, ok := parsed["commands"]; !ok {
		t.Error("missing 'commands' key")
	}
	if _, ok := parsed["total"]; !ok {
		t.Error("missing 'total' key")
	}
}

// --- ConfigSetCmd DryRun ---

func TestConfigSetCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &ConfigSetCmd{Key: "test.key", Value: "test.value"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp output.Response
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
}

// --- ConfigGetCmd DryRun ---

func TestConfigGetCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &ConfigGetCmd{Key: "test.key"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp output.Response
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
}

// --- ConfigListCmd DryRun ---

func TestConfigListCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &ConfigListCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp output.Response
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
}

// --- Allowlist blocking ---

func TestConfigSetCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &ConfigSetCmd{Key: "k", Value: "v"}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for denied command")
	}
	ee, ok := err.(*output.ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != exitcode.PermissionDenied {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.PermissionDenied)
	}
}

func TestConfigGetCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &ConfigGetCmd{Key: "k"}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for denied command")
	}
	ee, ok := err.(*output.ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != exitcode.PermissionDenied {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.PermissionDenied)
	}
}

func TestConfigListCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &ConfigListCmd{}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for denied command")
	}
	ee, ok := err.(*output.ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != exitcode.PermissionDenied {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.PermissionDenied)
	}
}

// --- Preflight ---

func TestPreflight_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	// Use access token env to avoid real auth
	t.Setenv("GWX_ACCESS_TOKEN", "fake-test-token")
	rctx.Auth = auth.NewManager()
	done, err := Preflight(rctx, "test.cmd", []string{"gmail"})
	if !done {
		t.Fatal("dry run should return done=true")
	}
	if err != nil {
		t.Fatalf("dry run should not return error, got %v", err)
	}
	if buf.Len() == 0 {
		t.Error("dry run should write output")
	}
}

func TestPreflight_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	done, err := Preflight(rctx, "drive.list", []string{"drive"})
	if !done {
		t.Fatal("denied should return done=true")
	}
	if err == nil {
		t.Fatal("denied should return error")
	}
	ee, ok := err.(*output.ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != exitcode.PermissionDenied {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.PermissionDenied)
	}
}

// TestCheckAllowlist_Allowed tests that a matching command passes.
func TestCheckAllowlist_Allowed(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	if err := CheckAllowlist(rctx, "gmail.list"); err != nil {
		t.Errorf("gmail.list should be allowed by gmail.*, got error: %v", err)
	}
}

// TestCheckAllowlist_Denied tests that a non-matching command is denied.
func TestCheckAllowlist_Denied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	if err := CheckAllowlist(rctx, "drive.list"); err == nil {
		t.Error("drive.list should be denied by gmail.* allowlist")
	}
}

// TestCheckAllowlist_ExactMatch tests that an exact match passes.
func TestCheckAllowlist_ExactMatch(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("calendar.agenda")
	if err := CheckAllowlist(rctx, "calendar.agenda"); err != nil {
		t.Errorf("exact match should be allowed: %v", err)
	}
	if err := CheckAllowlist(rctx, "calendar.list"); err == nil {
		t.Error("non-exact match should be denied")
	}
}

// --- handleAPIError ---

func TestHandleAPIError_General(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	err := handleAPIError(rctx, java_err("something broke"))
	if err == nil {
		t.Fatal("expected error")
	}
	ee, ok := err.(*output.ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != exitcode.GeneralError {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.GeneralError)
	}
}

type java_err string

func (e java_err) Error() string { return string(e) }

// --- SkillListCmd DryRun ---

func TestSkillListCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &SkillListCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp output.Response
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
}

// --- SkillListCmd without DryRun ---

func TestSkillListCmd_Run(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	cmd := &SkillListCmd{}
	err := cmd.Run(rctx)
	// May succeed or fail depending on skill files, but should not panic
	if err != nil {
		// If it fails, just verify it's a proper exit error
		_, ok := err.(*output.ExitError)
		if !ok {
			t.Fatalf("expected *ExitError or nil, got %T: %v", err, err)
		}
	} else {
		if buf.Len() == 0 {
			t.Error("expected some output")
		}
	}
}

// --- Various commands with DryRun covering Run methods ---

func TestWeeklyDigestCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &WeeklyDigestCmd{Weeks: 1}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.PermissionDenied {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.PermissionDenied)
	}
}

func TestContextBoostCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &ContextBoostCmd{Topic: "test", Days: 7, Limit: 5}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestBugIntakeCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &BugIntakeCmd{}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestTestMatrixCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &TestMatrixCmd{Action: "init"}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestSpecHealthCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &SpecHealthCmd{Action: "init"}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestSprintBoardCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &SprintBoardCmd{Action: "init"}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestReviewNotifyCmd_ExecuteWithoutChannel(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &ReviewNotifyCmd{SpecFolder: "specs", Reviewers: "a@b.com", Execute: true}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error when --execute without --channel")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.UsageError {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.UsageError)
	}
}

func TestEmailFromDocCmd_ExecuteWithoutRecipients(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &EmailFromDocCmd{DocID: "doc123", Execute: true}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error when --execute without --recipients")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.UsageError {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.UsageError)
	}
}

func TestPipeCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &PipeCmd{Pipeline: "drive list | sheets append X A:B"}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestPipeCmd_EmptyPipeline(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &PipeCmd{Pipeline: ""}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for empty pipeline")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.InvalidInput {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.InvalidInput)
	}
}

// --- Batch DryRun tests for service commands ---
// These test the Preflight path (allowlist + auth + dry_run) for many commands.

func newDryRunContext(t *testing.T) (*RunContext, *bytes.Buffer) {
	t.Helper()
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	rctx.Auth = auth.NewManager()
	t.Setenv("GWX_ACCESS_TOKEN", "fake-test-token")
	return rctx, buf
}

func TestGmailListCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &GmailListCmd{Limit: 10}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestGmailSearchCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &GmailSearchCmd{Query: "from:test"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestGmailGetCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &GmailGetCmd{MessageID: "msg123"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestGmailSendCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &GmailSendCmd{To: []string{"a@b.com"}, Subject: "test", Body: "hello"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestGmailDraftCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &GmailDraftCmd{To: []string{"a@b.com"}, Subject: "test", Body: "hello"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestGmailLabelsCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &GmailLabelsCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestCalendarAgendaCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &CalendarAgendaCmd{Days: 1}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestDriveListCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &DriveListCmd{Limit: 10}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestDriveSearchCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &DriveSearchCmd{Query: "name contains 'test'"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestDocsGetCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &DocsGetCmd{DocID: "doc123"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestDocsCreateCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &DocsCreateCmd{Title: "My Doc"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestSheetsReadCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &SheetsReadCmd{SpreadsheetID: "sheet123", Range: "A1:B10"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestSheetsCreateCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &SheetsCreateCmd{Title: "My Sheet"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestTasksListCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &TasksListCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestTasksCreateCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &TasksCreateCmd{Title: "Buy milk"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestContactsListCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &ContactsListCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestContactsSearchCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &ContactsSearchCmd{Query: "john"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestChatSpacesCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &ChatSpacesCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestChatSendCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &ChatSendCmd{Space: "spaces/AAA", Text: "hello"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestStandupCmd_AllowlistDenied(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.*")
	cmd := &StandupCmd{Days: 1}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestPipeCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &PipeCmd{Pipeline: "gmail list | drive search report"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp output.Response
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	raw, _ := json.Marshal(resp.Data)
	var data map[string]interface{}
	json.Unmarshal(raw, &data)
	if data["dry_run"] != true {
		t.Error("expected dry_run=true")
	}
	count, ok := data["count"].(float64)
	if !ok || int(count) != 2 {
		t.Errorf("expected count=2, got %v", data["count"])
	}
}

// testAllowlist creates an Allowlist for testing using the env var mechanism.
func testAllowlist(pattern string) *config.Allowlist {
	old := os.Getenv("GWX_ENABLE_COMMANDS")
	os.Setenv("GWX_ENABLE_COMMANDS", pattern)
	al := config.LoadAllowlist()
	if old == "" {
		os.Unsetenv("GWX_ENABLE_COMMANDS")
	} else {
		os.Setenv("GWX_ENABLE_COMMANDS", old)
	}
	return al
}
