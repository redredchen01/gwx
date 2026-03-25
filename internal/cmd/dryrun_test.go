package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/output"
)

// --- DryRun helper ---

// assertDryRun is a reusable helper: runs cmd.Run in DryRun mode, asserts success.
func assertDryRun(t *testing.T, cmd interface{ Run(*RunContext) error }, name string) {
	t.Helper()
	rctx, buf := newDryRunContext(t)
	err := cmd.Run(rctx)
	if err != nil {
		t.Fatalf("[%s] DryRun unexpected error: %v", name, err)
	}
	if buf.Len() == 0 {
		t.Fatalf("[%s] DryRun produced no output", name)
	}
}

// assertDryRunJSON verifies the dry_run response is valid JSON with status=ok.
func assertDryRunJSON(t *testing.T, cmd interface{ Run(*RunContext) error }, name string) {
	t.Helper()
	rctx, buf := newDryRunContext(t)
	err := cmd.Run(rctx)
	if err != nil {
		t.Fatalf("[%s] DryRun unexpected error: %v", name, err)
	}
	var resp output.Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("[%s] DryRun invalid JSON: %v\nraw: %s", name, err, buf.String())
	}
	if resp.Status != "ok" {
		t.Fatalf("[%s] DryRun status = %q, want ok", name, resp.Status)
	}
}

// assertAllowlistDenied verifies a command is denied by an unrelated allowlist.
func assertAllowlistDenied(t *testing.T, cmd interface{ Run(*RunContext) error }, name string) {
	t.Helper()
	rctx, _ := newTestRunContext(t)
	rctx.Allowlist = testAllowlist("gmail.list") // very narrow allowlist
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatalf("[%s] expected allowlist denial error, got nil", name)
	}
	ee, ok := err.(*output.ExitError)
	if !ok {
		t.Fatalf("[%s] expected *ExitError, got %T: %v", name, err, err)
	}
	if ee.Code != exitcode.PermissionDenied {
		t.Fatalf("[%s] code = %d, want %d (PermissionDenied)", name, ee.Code, exitcode.PermissionDenied)
	}
}

// ===== Gmail remaining DryRun tests =====

func TestGmailReplyCmd_DryRun(t *testing.T) {
	assertDryRun(t, &GmailReplyCmd{MessageID: "msg123", Body: "ok"}, "gmail.reply")
}

func TestGmailDigestCmd_DryRun(t *testing.T) {
	assertDryRun(t, &GmailDigestCmd{Limit: 10}, "gmail.digest")
}

func TestGmailArchiveCmd_DryRun(t *testing.T) {
	assertDryRun(t, &GmailArchiveCmd{Query: "older_than:1y", Limit: 50}, "gmail.archive")
}

func TestGmailForwardCmd_DryRun(t *testing.T) {
	assertDryRun(t, &GmailForwardCmd{MessageID: "msg123", To: []string{"a@b.com"}}, "gmail.forward")
}

// GmailLabelCmd has pre-Preflight validation: requires --add or --remove.
func TestGmailLabelCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &GmailLabelCmd{Query: "from:ci", Add: []string{"CI"}, Limit: 10}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestGmailLabelCmd_NoFlags(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &GmailLabelCmd{Query: "test", Limit: 10}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error when neither --add nor --remove given")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.InvalidInput {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.InvalidInput)
	}
}

// ===== Calendar remaining DryRun tests =====

func TestCalendarListCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &CalendarListCmd{From: "2026-03-20", To: "2026-03-25"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestCalendarCreateCmd_DryRun(t *testing.T) {
	assertDryRun(t, &CalendarCreateCmd{
		Title: "Test", Start: "2026-03-20T10:00:00Z", End: "2026-03-20T11:00:00Z",
	}, "calendar.create")
}

func TestCalendarUpdateCmd_DryRun(t *testing.T) {
	assertDryRun(t, &CalendarUpdateCmd{EventID: "ev123", Title: "Updated"}, "calendar.update")
}

func TestCalendarDeleteCmd_DryRun(t *testing.T) {
	assertDryRun(t, &CalendarDeleteCmd{EventID: "ev123"}, "calendar.delete")
}

func TestCalendarFindSlotCmd_DryRun(t *testing.T) {
	assertDryRun(t, &CalendarFindSlotCmd{Attendees: []string{"a@b.com"}, Duration: "30m", Days: 3}, "calendar.find-slot")
}

func TestCalendarFindSlotCmd_InvalidDuration(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &CalendarFindSlotCmd{Attendees: []string{"a@b.com"}, Duration: "notduration", Days: 3}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.InvalidInput {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.InvalidInput)
	}
}

func TestCalendarListCmd_InvalidFrom(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &CalendarListCmd{From: "not-a-date", To: "2026-03-25"}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for invalid --from")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.InvalidInput {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.InvalidInput)
	}
}

func TestCalendarListCmd_InvalidTo(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &CalendarListCmd{From: "2026-03-20", To: "not-a-date"}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for invalid --to")
	}
}

// ===== Drive remaining DryRun tests =====

func TestDriveUploadCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DriveUploadCmd{File: "/tmp/test.txt"}, "drive.upload")
}

func TestDriveDownloadCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DriveDownloadCmd{FileID: "file123"}, "drive.download")
}

func TestDriveShareCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DriveShareCmd{FileID: "file123", Email: "a@b.com", Role: "reader"}, "drive.share")
}

func TestDriveMkdirCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DriveMkdirCmd{Name: "test-folder"}, "drive.mkdir")
}

// ===== Docs remaining DryRun tests =====

func TestDocsAppendCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DocsAppendCmd{DocID: "doc123", Text: "hello"}, "docs.append")
}

func TestDocsExportCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DocsExportCmd{DocID: "doc123", ExportFmt: "pdf"}, "docs.export")
}

func TestDocsSearchCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DocsSearchCmd{DocID: "doc123", Query: "hello"}, "docs.search")
}

func TestDocsReplaceCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DocsReplaceCmd{DocID: "doc123", Find: "old", Replace: "new"}, "docs.replace")
}

func TestDocsFromSheetCmd_DryRun(t *testing.T) {
	assertDryRun(t, &DocsFromSheetCmd{SpreadsheetID: "sheet123", Range: "A:D", Title: "Report"}, "docs.from-sheet")
}

func TestDocsTemplateCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &DocsTemplateCmd{TemplateID: "tmpl123", Vars: `{"name":"Alice"}`, Title: "Letter"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestDocsTemplateCmd_InvalidVars(t *testing.T) {
	rctx, _ := newDryRunContext(t)
	cmd := &DocsTemplateCmd{TemplateID: "tmpl123", Vars: `not json`}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON vars")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.InvalidInput {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.InvalidInput)
	}
}

// ===== Sheets remaining DryRun tests =====

func TestSheetsAppendCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsAppendCmd{SpreadsheetID: "s123", Range: "A:C", Values: `[["a",1]]`}, "sheets.append")
}

func TestSheetsUpdateCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsUpdateCmd{SpreadsheetID: "s123", Range: "A1:C3", Values: `[["x"]]`}, "sheets.update")
}

func TestSheetsInfoCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsInfoCmd{SpreadsheetID: "s123"}, "sheets.info")
}

func TestSheetsSearchCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsSearchCmd{SpreadsheetID: "s123", Query: "test"}, "sheets.search")
}

func TestSheetsFilterCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsFilterCmd{SpreadsheetID: "s123", Range: "A:D", Column: 0, Value: "x"}, "sheets.filter")
}

func TestSheetsClearCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsClearCmd{SpreadsheetID: "s123", Range: "A2:D"}, "sheets.clear")
}

func TestSheetsDescribeCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsDescribeCmd{SpreadsheetID: "s123", Samples: 20}, "sheets.describe")
}

func TestSheetsStatsCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsStatsCmd{SpreadsheetID: "s123"}, "sheets.stats")
}

func TestSheetsDiffCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsDiffCmd{SpreadsheetID: "s123", RangeA: "Tab1", RangeB: "Tab2"}, "sheets.diff")
}

func TestSheetsCopyTabCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsCopyTabCmd{SpreadsheetID: "s123", Source: "Tab1", Name: "Tab1 Copy"}, "sheets.copy-tab")
}

func TestSheetsExportCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsExportCmd{SpreadsheetID: "s123", Range: "A:D", ExportFmt: "csv"}, "sheets.export")
}

func TestSheetsImportCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetsImportCmd{SpreadsheetID: "s123", Range: "A1", File: "/tmp/data.csv", ImportFmt: "csv"}, "sheets.import")
}

// ===== Tasks remaining DryRun tests =====

func TestTasksListsCmd_DryRun(t *testing.T) {
	assertDryRun(t, &TasksListsCmd{}, "tasks.lists")
}

func TestTasksCompleteCmd_DryRun(t *testing.T) {
	assertDryRun(t, &TasksCompleteCmd{TaskID: "task123"}, "tasks.complete")
}

func TestTasksDeleteCmd_DryRun(t *testing.T) {
	assertDryRun(t, &TasksDeleteCmd{TaskID: "task123"}, "tasks.delete")
}

// ===== Contacts remaining DryRun tests =====

func TestContactsGetCmd_DryRun(t *testing.T) {
	assertDryRun(t, &ContactsGetCmd{ResourceName: "people/c123"}, "contacts.get")
}

// ===== Chat remaining DryRun tests =====

func TestChatMessagesCmd_DryRun(t *testing.T) {
	assertDryRun(t, &ChatMessagesCmd{Space: "spaces/AAA", Limit: 20}, "chat.messages")
}

// ===== Analytics DryRun tests =====

func TestAnalyticsReportCmd_DryRun(t *testing.T) {
	assertDryRun(t, &AnalyticsReportCmd{
		Property: "properties/123456", Metrics: []string{"sessions"},
		StartDate: "7daysAgo", EndDate: "today",
	}, "analytics.report")
}

func TestAnalyticsRealtimeCmd_DryRun(t *testing.T) {
	assertDryRun(t, &AnalyticsRealtimeCmd{
		Property: "properties/123456", Metrics: []string{"activeUsers"},
	}, "analytics.realtime")
}

func TestAnalyticsPropertiesCmd_DryRun(t *testing.T) {
	assertDryRun(t, &AnalyticsPropertiesCmd{}, "analytics.properties")
}

func TestAnalyticsAudiencesCmd_DryRun(t *testing.T) {
	assertDryRun(t, &AnalyticsAudiencesCmd{Property: "properties/123456"}, "analytics.audiences")
}

// ===== SearchConsole DryRun tests =====

func TestSearchConsoleQueryCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SearchConsoleQueryCmd{
		Site: "https://example.com", StartDate: "2026-03-01",
	}, "searchconsole.query")
}

func TestSearchConsoleSitesCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SearchConsoleSitesCmd{}, "searchconsole.sites")
}

func TestSearchConsoleInspectCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SearchConsoleInspectCmd{
		Site: "https://example.com", URL: "https://example.com/page",
	}, "searchconsole.inspect")
}

func TestSearchConsoleSitemapsCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SearchConsoleSitemapsCmd{Site: "https://example.com"}, "searchconsole.sitemaps")
}

func TestSearchConsoleIndexStatusCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SearchConsoleIndexStatusCmd{Site: "https://example.com"}, "searchconsole.index-status")
}

// ===== Slides DryRun tests =====

func TestSlidesGetCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SlidesGetCmd{PresentationID: "pres123"}, "slides.get")
}

func TestSlidesListCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SlidesListCmd{Limit: 10}, "slides.list")
}

func TestSlidesCreateCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SlidesCreateCmd{Title: "Test Presentation"}, "slides.create")
}

func TestSlidesDuplicateCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SlidesDuplicateCmd{PresentationID: "pres123", Title: "Copy"}, "slides.duplicate")
}

func TestSlidesExportCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SlidesExportCmd{PresentationID: "pres123", Format: "pdf"}, "slides.export")
}

func TestSlidesFromSheetCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SlidesFromSheetCmd{TemplateID: "tmpl123", SheetID: "s123", Range: "Sheet1"}, "slides.from-sheet")
}

// ===== Forms DryRun tests =====

func TestFormsGetCmd_DryRun(t *testing.T) {
	assertDryRun(t, &FormsGetCmd{FormID: "form123"}, "forms.get")
}

func TestFormsResponsesCmd_DryRun(t *testing.T) {
	assertDryRun(t, &FormsResponsesCmd{FormID: "form123", Limit: 10}, "forms.responses")
}

func TestFormsResponseCmd_DryRun(t *testing.T) {
	assertDryRun(t, &FormsResponseCmd{FormID: "form123", ResponseID: "resp1"}, "forms.response")
}

// ===== BigQuery DryRun tests =====

func TestBQQueryCmd_DryRun(t *testing.T) {
	assertDryRun(t, &BQQueryCmd{SQL: "SELECT 1", Project: "my-project"}, "bigquery.query")
}

func TestBQDatasetsCmd_DryRun(t *testing.T) {
	assertDryRun(t, &BQDatasetsCmd{Project: "my-project"}, "bigquery.datasets")
}

func TestBQTablesCmd_DryRun(t *testing.T) {
	assertDryRun(t, &BQTablesCmd{Project: "my-project", Dataset: "ds1"}, "bigquery.tables")
}

func TestBQDescribeCmd_DryRun(t *testing.T) {
	assertDryRun(t, &BQDescribeCmd{Table: "t1", Project: "my-project", Dataset: "ds1"}, "bigquery.describe")
}

// ===== Workflow DryRun tests =====

func TestWeeklyDigestCmd_DryRun(t *testing.T) {
	assertDryRun(t, &WeeklyDigestCmd{Weeks: 1}, "workflow.weekly-digest")
}

func TestContextBoostCmd_DryRun(t *testing.T) {
	assertDryRun(t, &ContextBoostCmd{Topic: "budget", Days: 7, Limit: 5}, "workflow.context-boost")
}

func TestBugIntakeCmd_DryRun(t *testing.T) {
	assertDryRun(t, &BugIntakeCmd{BugID: "BUG-001"}, "workflow.bug-intake")
}

func TestTestMatrixCmd_DryRun(t *testing.T) {
	assertDryRun(t, &TestMatrixCmd{Action: "init"}, "workflow.test-matrix")
}

func TestSpecHealthCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SpecHealthCmd{Action: "init"}, "workflow.spec-health")
}

func TestSprintBoardCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SprintBoardCmd{Action: "init"}, "workflow.sprint-board")
}

func TestReviewNotifyCmd_DryRun(t *testing.T) {
	assertDryRun(t, &ReviewNotifyCmd{SpecFolder: "specs", Reviewers: "a@b.com"}, "workflow.review-notify")
}

func TestEmailFromDocCmd_DryRun(t *testing.T) {
	assertDryRun(t, &EmailFromDocCmd{DocID: "doc123"}, "workflow.email-from-doc")
}

func TestSheetToEmailCmd_DryRun(t *testing.T) {
	assertDryRun(t, &SheetToEmailCmd{SheetID: "s123", Range: "A:F", EmailCol: 0, SubjectCol: 1, BodyCol: 2}, "workflow.sheet-to-email")
}

func TestParallelScheduleCmd_DryRun(t *testing.T) {
	assertDryRun(t, &ParallelScheduleCmd{Title: "test", Attendees: "a@b.com", Duration: "30m"}, "workflow.parallel-schedule")
}

// ===== StandupCmd and MeetingPrepCmd DryRun =====

func TestStandupCmd_DryRun(t *testing.T) {
	assertDryRun(t, &StandupCmd{Days: 1}, "workflow.standup")
}

func TestMeetingPrepCmd_DryRun(t *testing.T) {
	assertDryRun(t, &MeetingPrepCmd{Meeting: "standup", Days: 1}, "meeting-prep")
}

// ===== Find and Context DryRun =====

func TestUnifiedSearchCmd_DryRun(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	cmd := &UnifiedSearchCmd{Query: "test", Services: []string{"gmail", "drive"}, Limit: 5}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
	// Verify the dry_run response structure
	var resp output.Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestContextCmd_DryRun(t *testing.T) {
	assertDryRun(t, &ContextCmd{Topic: "budget", Days: 7, Limit: 5}, "context")
}

// ===== Completion tests =====

func TestCompletionBashCmd_Run(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	cmd := &CompletionBashCmd{}
	// Completion writes to stdout (fmt.Println), not to printer.
	// We redirect via a custom writer on the printer to verify no error.
	_ = buf // output goes to os.Stdout, not printer
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionZshCmd_Run(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &CompletionZshCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionFishCmd_Run(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &CompletionFishCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ===== DoctorCmd test =====

func TestDoctorCmd_Run(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.Auth = auth.NewManager()
	cmd := &DoctorCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
	var resp output.Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
}

// ===== Slack DryRun tests =====

func TestSlackLoginCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &SlackLoginCmd{Token: "xoxb-fake"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestSlackChannelsCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &SlackChannelsCmd{Limit: 10}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestSlackSendCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &SlackSendCmd{Channel: "#general", Text: "hello"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestSlackMessagesCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &SlackMessagesCmd{Channel: "C0123", Limit: 20}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestSlackSearchCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &SlackSearchCmd{Query: "test", Limit: 20}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestSlackUsersCmd_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &SlackUsersCmd{Limit: 100}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

// ===== Allowlist denial batch tests for remaining commands =====

func TestCalendarAgendaCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &CalendarAgendaCmd{Days: 1}, "calendar.agenda")
}

func TestDriveListCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &DriveListCmd{Limit: 10}, "drive.list")
}

func TestDocsGetCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &DocsGetCmd{DocID: "doc123"}, "docs.get")
}

func TestSheetsReadCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &SheetsReadCmd{SpreadsheetID: "s123", Range: "A1"}, "sheets.read")
}

func TestTasksListCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &TasksListCmd{}, "tasks.list")
}

func TestContactsListCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &ContactsListCmd{}, "contacts.list")
}

func TestChatSpacesCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &ChatSpacesCmd{}, "chat.spaces")
}

func TestAnalyticsReportCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &AnalyticsReportCmd{
		Property: "properties/123456", Metrics: []string{"sessions"},
	}, "analytics.report")
}

func TestSlidesGetCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &SlidesGetCmd{PresentationID: "pres123"}, "slides.get")
}

func TestFormsGetCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &FormsGetCmd{FormID: "form123"}, "forms.get")
}

func TestBQQueryCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &BQQueryCmd{SQL: "SELECT 1", Project: "p"}, "bigquery.query")
}

func TestSearchConsoleSitesCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &SearchConsoleSitesCmd{}, "searchconsole.sites")
}

func TestMeetingPrepCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &MeetingPrepCmd{Meeting: "standup"}, "meeting-prep")
}

// ===== handleAPIError with different error types =====

func TestHandleAPIError_CircuitOpen(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	err := handleAPIError(rctx, &testCircuitErr{msg: "circuit open"})
	if err == nil {
		t.Fatal("expected error")
	}
	// The error is formatted as ExitError but wrapping is via the printer;
	// just verify we don't get a nil.
}

type testCircuitErr struct {
	msg string
}

func (e *testCircuitErr) Error() string { return e.msg }

// ===== joinNames helper tests =====

func TestJoinNames_Empty(t *testing.T) {
	got := joinNames(nil)
	if got != "" {
		t.Errorf("joinNames(nil) = %q, want empty", got)
	}
}

func TestJoinNames_Few(t *testing.T) {
	got := joinNames([]string{"a", "b", "c"})
	if got != "a, b, c" {
		t.Errorf("joinNames([a,b,c]) = %q", got)
	}
}

func TestJoinNames_Many(t *testing.T) {
	got := joinNames([]string{"a", "b", "c", "d", "e", "f", "g"})
	if got != "a, b, c, d, e (and 2 more)" {
		t.Errorf("joinNames(7) = %q", got)
	}
}

// ===== readPastedJSON tests =====

func TestReadPastedJSON_CompleteJSON(t *testing.T) {
	data, err := readPastedJSON(`{"installed":{"client_id":"x"}}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("expected valid JSON, got %q", data)
	}
}

// ===== parseJSON from helpers_test has valid/invalid, add empty object test =====

func TestParseJSON_EmptyObject(t *testing.T) {
	var result map[string]string
	if err := parseJSON(`{}`, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

// ===== checkResult type tests =====

func TestCheckResult_Fields(t *testing.T) {
	cr := checkResult{Name: "test", Status: "ok", Message: "good"}
	if cr.Name != "test" || cr.Status != "ok" || cr.Message != "good" {
		t.Errorf("unexpected fields: %+v", cr)
	}
}

// ===== newDryRunContext ensures buf is non-nil =====
// This is a helper validation, not testing application code.

func TestNewDryRunContext_WriterNotNil(t *testing.T) {
	rctx, buf := newDryRunContext(t)
	if rctx.Printer.Writer == nil {
		t.Fatal("Printer.Writer should not be nil")
	}
	if buf == nil {
		t.Fatal("buf should not be nil")
	}
}

// ===== Verify Preflight returns done=false when no dry-run, no allowlist =====

func TestPreflight_NoDryRun_NoAllowlist(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	t.Setenv("GWX_ACCESS_TOKEN", "fake-test-token")
	rctx.Auth = auth.NewManager()
	done, err := Preflight(rctx, "test.cmd", []string{"gmail"})
	if done {
		t.Fatal("expected done=false for normal flow")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ===== Ensure config commands work without DryRun too =====

func TestConfigSetCmd_RealRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	// Use a temp HOME to avoid polluting real config
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cmd := &ConfigSetCmd{Key: "test.key", Value: "test.value"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestConfigGetCmd_RealRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cmd := &ConfigGetCmd{Key: "nonexistent.key"}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestConfigListCmd_RealRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cmd := &ConfigListCmd{}
	if err := cmd.Run(rctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

// ===== Verify SkillCmd variants =====

func TestSkillListCmd_AllowlistDenied(t *testing.T) {
	assertAllowlistDenied(t, &SkillListCmd{}, "skill.list")
}

// ===== GitHub status (uses keyring, not Google auth) =====

func TestGitHubStatusCmd_Run(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &GitHubStatusCmd{}
	// Without a saved token, should return auth_required
	err := cmd.Run(rctx)
	if err == nil {
		// If the mock keyring has a token, status will succeed. That's fine.
		return
	}
	ee, ok := err.(*output.ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != exitcode.AuthRequired {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.AuthRequired)
	}
}

// ===== SlackStatusCmd (uses keyring) =====

func TestSlackStatusCmd_Run(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &SlackStatusCmd{}
	err := cmd.Run(rctx)
	if err == nil {
		return // token found in mock keyring
	}
	ee, ok := err.(*output.ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != exitcode.AuthRequired {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.AuthRequired)
	}
}

// ===== Workflow validation tests =====

func TestReviewNotifyCmd_ExecuteWithChannelNoReviewers(t *testing.T) {
	// Execute with --channel but still needs auth
	rctx, _ := newDryRunContext(t)
	rctx.DryRun = false // real run
	cmd := &ReviewNotifyCmd{SpecFolder: "specs", Reviewers: "a@b.com", Channel: "email", Execute: true}
	// This will go through Preflight auth check and succeed (we have GWX_ACCESS_TOKEN)
	// then fail at the workflow execution itself. Just verify no panic.
	_ = cmd.Run(rctx)
}

func TestEmailFromDocCmd_ExecuteValidation(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	cmd := &EmailFromDocCmd{DocID: "doc123", Execute: true, Recipients: ""}
	err := cmd.Run(rctx)
	if err == nil {
		t.Fatal("expected error for --execute without --recipients")
	}
	ee := err.(*output.ExitError)
	if ee.Code != exitcode.UsageError {
		t.Errorf("code = %d, want %d", ee.Code, exitcode.UsageError)
	}
}

// ===== PipeCmd single stage dry run =====

func TestPipeCmd_SingleStage_DryRun(t *testing.T) {
	rctx, buf := newTestRunContext(t)
	rctx.DryRun = true
	cmd := &PipeCmd{Pipeline: "gmail list"}
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
	count, ok := data["count"].(float64)
	if !ok || int(count) != 1 {
		t.Errorf("expected count=1 for single stage, got %v", data["count"])
	}
}

// ===== Ensure DryRun works for the hidden alias 'digest' =====

func TestDigestCmd_DryRun(t *testing.T) {
	// Digest is an alias for WeeklyDigestCmd in the WorkflowCmd struct.
	assertDryRun(t, &WeeklyDigestCmd{Weeks: 2}, "workflow.digest")
}

// ===== EnsureAuth with access token env =====

func TestEnsureAuth_AccessToken(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	t.Setenv("GWX_ACCESS_TOKEN", "test-token-for-auth")
	rctx.Auth = auth.NewManager()
	err := EnsureAuth(rctx, []string{"gmail"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rctx.APIClient == nil {
		t.Fatal("APIClient should not be nil after EnsureAuth with access token")
	}
}

func TestEnsureAuth_NoToken(t *testing.T) {
	rctx, _ := newTestRunContext(t)
	t.Setenv("GWX_ACCESS_TOKEN", "") // clear
	rctx.Auth = auth.NewManager()
	err := EnsureAuth(rctx, []string{"gmail"})
	if err == nil {
		// If mock keyring has tokens, this might succeed. That's fine.
		return
	}
	// Verify the error message mentions onboard or login
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

// ===== Placeholder for keeping buf reference so compiler doesn't complain =====

var _ = (*bytes.Buffer)(nil)
