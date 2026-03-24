package cmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// --- Schema ---

func TestCLI_Schema(t *testing.T) {
	stdout, _, code := runGWX(t, "schema")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("schema should produce valid JSON: %v\nraw: %s", err, stdout)
	}
	// Must have top-level keys
	for _, key := range []string{"commands", "by_service", "total", "services"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("schema output missing key %q", key)
		}
	}
	// Commands must be a non-empty array
	cmds, ok := parsed["commands"].([]interface{})
	if !ok || len(cmds) == 0 {
		t.Fatal("commands should be a non-empty array")
	}
	// Total must match the actual count
	total, ok := parsed["total"].(float64)
	if !ok || int(total) != len(cmds) {
		t.Errorf("total=%v, len(commands)=%d", total, len(cmds))
	}
}

func TestCLI_Schema_CommandsSorted(t *testing.T) {
	stdout, _, code := runGWX(t, "schema")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var parsed map[string]interface{}
	json.Unmarshal([]byte(stdout), &parsed)
	cmds := parsed["commands"].([]interface{})

	var names []string
	for _, c := range cmds {
		m := c.(map[string]interface{})
		names = append(names, m["name"].(string))
	}
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("commands not sorted: %s comes after %s", names[i], names[i-1])
			break
		}
	}
}

func TestCLI_Schema_CommandFields(t *testing.T) {
	stdout, _, _ := runGWX(t, "schema")
	var parsed map[string]interface{}
	json.Unmarshal([]byte(stdout), &parsed)
	cmds := parsed["commands"].([]interface{})
	first := cmds[0].(map[string]interface{})

	// Each command must have these fields
	for _, field := range []string{"name", "service", "description", "safety_tier", "requires_auth"} {
		if _, ok := first[field]; !ok {
			t.Errorf("command missing field %q", field)
		}
	}
}

// --- Config set/get ---

func TestCLI_ConfigSet_DryRun(t *testing.T) {
	stdout, _, code := runGWX(t, "--dry-run", "config", "set", "test.key", "test.value")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	if data["dry_run"] != "config.set" {
		t.Errorf("expected dry_run=config.set, got %v", data["dry_run"])
	}
}

func TestCLI_ConfigGet_DryRun(t *testing.T) {
	stdout, _, code := runGWX(t, "--dry-run", "config", "get", "test.key")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var resp map[string]interface{}
	json.Unmarshal([]byte(stdout), &resp)
	data := resp["data"].(map[string]interface{})
	if data["dry_run"] != "config.get" {
		t.Errorf("expected dry_run=config.get, got %v", data["dry_run"])
	}
}

func TestCLI_ConfigList_DryRun(t *testing.T) {
	stdout, _, code := runGWX(t, "--dry-run", "config", "list")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var resp map[string]interface{}
	json.Unmarshal([]byte(stdout), &resp)
	data := resp["data"].(map[string]interface{})
	if data["dry_run"] != "config.list" {
		t.Errorf("expected dry_run=config.list, got %v", data["dry_run"])
	}
}

func TestCLI_ConfigSetGet_RoundTrip(t *testing.T) {
	// Config set/get uses isolated HOME from cleanEnv, which means
	// each runGWX call gets a DIFFERENT temp dir. To test round-trip,
	// we need to use the same HOME for both set and get.
	tmpHome := t.TempDir()
	env := []string{
		"HOME=" + tmpHome,
		"PATH=" + os.Getenv("PATH"),
		"GWX_AUTO_JSON=1",
	}

	// Set a value
	setCmd := exec.Command(binaryPath, "config", "set", "test.roundtrip", "hello42")
	setCmd.Env = env
	if out, err := setCmd.CombinedOutput(); err != nil {
		t.Fatalf("config set failed: %v\n%s", err, out)
	}

	// Get it back (same HOME)
	getCmd := exec.Command(binaryPath, "config", "get", "test.roundtrip")
	getCmd.Env = env
	out, err := getCmd.Output()
	if err != nil {
		t.Fatalf("config get failed: %v", err)
	}
	var resp map[string]interface{}
	json.Unmarshal(out, &resp)
	data := resp["data"].(map[string]interface{})
	if data["value"] != "hello42" {
		t.Errorf("expected value=hello42, got %v", data["value"])
	}
}

func TestCLI_ConfigList(t *testing.T) {
	// config list should work even with empty config (returns empty preferences)
	stdout, _, code := runGWX(t, "config", "list")
	if code != 0 {
		t.Fatalf("config list exit %d", code)
	}
	var resp map[string]interface{}
	json.Unmarshal([]byte(stdout), &resp)
	data := resp["data"].(map[string]interface{})
	if _, ok := data["preferences"]; !ok {
		t.Error("config list should have 'preferences' key")
	}
	if _, ok := data["count"]; !ok {
		t.Error("config list should have 'count' key")
	}
}

// --- Agent ---

func TestCLI_AgentExitCodes_AllCodes(t *testing.T) {
	stdout, _, code := runGWX(t, "agent", "exit-codes")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var resp map[string]interface{}
	json.Unmarshal([]byte(stdout), &resp)
	data := resp["data"].(map[string]interface{})

	// Verify all known exit codes are present
	expectedCodes := map[string]string{
		"0":  "success",
		"1":  "general_error",
		"2":  "usage_error",
		"10": "auth_required",
		"11": "auth_expired",
		"12": "permission_denied",
		"20": "not_found",
		"21": "conflict",
		"30": "rate_limited",
		"31": "circuit_open",
		"40": "invalid_input",
		"50": "dry_run_success",
	}
	for code, name := range expectedCodes {
		if data[code] != name {
			t.Errorf("exit code %s: expected %q, got %v", code, name, data[code])
		}
	}
}

func TestCLI_AgentHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "agent", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !strings.Contains(output, "exit-codes") {
		t.Fatal("agent help should list exit-codes subcommand")
	}
}

// --- Workflow help ---

func TestCLI_WorkflowHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "workflow", "--help")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	for _, sub := range []string{"weekly-digest", "context-boost", "bug-intake", "test-matrix", "spec-health", "sprint-board", "review-notify", "email-from-doc", "sheet-to-email", "parallel-schedule"} {
		if !strings.Contains(output, sub) {
			t.Errorf("workflow help should list %q subcommand", sub)
		}
	}
}

// --- Workflow unauthenticated tests ---

func TestCLI_WorkflowWeeklyDigest_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "weekly-digest")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_WorkflowContextBoost_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "context-boost", "test-topic")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_WorkflowBugIntake_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "bug-intake")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_WorkflowTestMatrix_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "test-matrix", "init")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_WorkflowSpecHealth_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "spec-health", "init")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_WorkflowSprintBoard_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "sprint-board", "init")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_WorkflowEmailFromDoc_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "email-from-doc", "--doc-id", "abc")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_WorkflowSheetToEmail_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "sheet-to-email", "--sheet-id", "abc", "--range", "A:B", "--email-col", "0", "--subject-col", "1", "--body-col", "2")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_WorkflowParallelSchedule_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "workflow", "parallel-schedule", "--title", "test", "--attendees", "a@b.com", "--duration", "30m")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

// --- Standup/MeetingPrep unauthenticated ---

func TestCLI_Standup_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "standup")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_MeetingPrep_Unauthenticated(t *testing.T) {
	// meeting-prep requires a positional arg (meeting title)
	_, _, code := runGWX(t, "meeting-prep", "standup")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

// --- Find/Context unauthenticated ---

func TestCLI_Find_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "find", "test-query")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

func TestCLI_Context_Unauthenticated(t *testing.T) {
	_, _, code := runGWX(t, "context", "test-topic")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d", code)
	}
}

// --- Analytics/SearchConsole/Slides unauthenticated ---

func TestCLI_AnalyticsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "analytics", "--help")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Fatal("analytics help should produce output")
	}
}

func TestCLI_SearchConsoleHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "searchconsole", "--help")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Fatal("searchconsole help should produce output")
	}
}

func TestCLI_SlidesHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "slides", "--help")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Fatal("slides help should produce output")
	}
}

// --- Allowlist cross-cutting ---

func TestCLI_WorkflowAllowlist_Denied(t *testing.T) {
	commands := []struct {
		args []string
	}{
		{[]string{"workflow", "weekly-digest"}},
		{[]string{"workflow", "context-boost", "topic"}},
		{[]string{"workflow", "bug-intake"}},
		{[]string{"standup"}},
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

// --- Dry-run tests ---

func TestCLI_Version_DryRun(t *testing.T) {
	// Version doesn't use Preflight, so dry-run still returns version
	stdout, _, code := runGWX(t, "--dry-run", "version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var resp map[string]interface{}
	json.Unmarshal([]byte(stdout), &resp)
	if resp["status"] != "ok" {
		t.Fatal("version should succeed even with dry-run")
	}
}
