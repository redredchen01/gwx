package mcp

import (
	"context"

	"github.com/redredchen01/gwx/internal/workflow"
)

// WorkflowTools returns 19 MCP tool definitions for workflow operations.
// All tools are read-only — they never execute actions.
func WorkflowTools() []Tool {
	return []Tool{
		// FA-B: Data Aggregation
		{
			Name:        "workflow_standup",
			Description: "Generate a daily standup report aggregating Git activity, Gmail, Calendar, and Tasks. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"days": {Type: "integer", Description: "Days of git history (default 1)"},
				},
			},
		},
		{
			Name:        "workflow_meeting_prep",
			Description: "Prepare context for an upcoming meeting: attendees, recent emails, related docs. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"meeting": {Type: "string", Description: "Meeting title or keyword to match"},
					"days":    {Type: "integer", Description: "Days ahead to search (default 1)"},
				},
				Required: []string{"meeting"},
			},
		},
		{
			Name:        "workflow_weekly_digest",
			Description: "Generate a weekly digest of email, meetings, and completed tasks. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"weeks": {Type: "integer", Description: "Number of weeks (default 1)"},
				},
			},
		},
		{
			Name:        "workflow_context_boost",
			Description: "Deep context gathering for a topic across Gmail, Drive, Calendar, and Contacts. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"topic": {Type: "string", Description: "Topic to gather context for"},
					"days":  {Type: "integer", Description: "Days of history (default 14)"},
					"limit": {Type: "integer", Description: "Max results per service (default 10)"},
				},
				Required: []string{"topic"},
			},
		},
		{
			Name:        "workflow_bug_intake",
			Description: "Search for bug-related emails, docs, and git history. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"bug_id": {Type: "string", Description: "Bug ID or keyword to search"},
					"after":  {Type: "string", Description: "Date filter (e.g. 2026/03/15)"},
				},
			},
		},
		// FA-C: Sheets State — Test Matrix
		{
			Name:        "workflow_test_matrix_init",
			Description: "Preview test matrix Sheet creation. Read-only in MCP mode (does not create Sheet).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature": {Type: "string", Description: "Feature name for the Sheet title"},
				},
				Required: []string{"feature"},
			},
		},
		{
			Name:        "workflow_test_matrix_sync",
			Description: "Preview test matrix sync. Read-only in MCP mode (does not write to Sheet).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"sheet_id": {Type: "string", Description: "Sheet ID"},
					"file":     {Type: "string", Description: "Test results file path"},
				},
				Required: []string{"sheet_id", "file"},
			},
		},
		{
			Name:        "workflow_test_matrix_stats",
			Description: "Get test matrix statistics from a Sheet. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"sheet_id": {Type: "string", Description: "Sheet ID"},
				},
				Required: []string{"sheet_id"},
			},
		},
		// FA-C: Sheets State — Spec Health
		{
			Name:        "workflow_spec_health_init",
			Description: "Preview spec health Sheet creation. Read-only in MCP mode (does not create Sheet).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature": {Type: "string", Description: "Feature name for the Sheet title"},
				},
				Required: []string{"feature"},
			},
		},
		{
			Name:        "workflow_spec_health_record",
			Description: "Preview spec health record. Read-only in MCP mode (does not write to Sheet).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"sheet_id":    {Type: "string", Description: "Sheet ID"},
					"spec_folder": {Type: "string", Description: "Spec folder path"},
				},
				Required: []string{"sheet_id", "spec_folder"},
			},
		},
		{
			Name:        "workflow_spec_health_stats",
			Description: "Get spec health statistics from a Sheet. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"sheet_id":    {Type: "string", Description: "Sheet ID"},
					"spec_folder": {Type: "string", Description: "Spec folder path (optional filter)"},
				},
				Required: []string{"sheet_id"},
			},
		},
		// FA-C: Sheets State — Sprint Board
		{
			Name:        "workflow_sprint_board_init",
			Description: "Preview sprint board Sheet creation. Read-only in MCP mode (does not create Sheet).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature": {Type: "string", Description: "Feature name for the Sheet title"},
				},
				Required: []string{"feature"},
			},
		},
		{
			Name:        "workflow_sprint_board_ticket",
			Description: "Preview ticket creation. Read-only in MCP mode (does not write to Sheet).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"sheet_id": {Type: "string", Description: "Sheet ID"},
					"title":    {Type: "string", Description: "Ticket title"},
					"assignee": {Type: "string", Description: "Assignee"},
					"priority": {Type: "string", Description: "Priority (P0-P3)"},
				},
				Required: []string{"sheet_id", "title"},
			},
		},
		{
			Name:        "workflow_sprint_board_stats",
			Description: "Get sprint board statistics from a Sheet. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"sheet_id": {Type: "string", Description: "Sheet ID"},
				},
				Required: []string{"sheet_id"},
			},
		},
		// FA-D: Action Workflows (preview only in MCP)
		{
			Name:        "workflow_review_notify",
			Description: "Preview review notification. Read-only in MCP mode (does not send notification).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spec_folder": {Type: "string", Description: "Spec folder path"},
					"reviewers":   {Type: "string", Description: "Comma-separated reviewer emails"},
					"channel":     {Type: "string", Description: "Notification channel (email or chat:spaces/XXX)"},
				},
				Required: []string{"spec_folder", "reviewers"},
			},
		},
		{
			Name:        "workflow_email_from_doc",
			Description: "Preview email from Google Doc. Read-only in MCP mode (does not send email).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"doc_id":     {Type: "string", Description: "Google Doc ID"},
					"recipients": {Type: "string", Description: "Comma-separated recipient emails"},
					"subject":    {Type: "string", Description: "Email subject override"},
				},
				Required: []string{"doc_id", "recipients"},
			},
		},
		{
			Name:        "workflow_sheet_to_email",
			Description: "Preview personalized emails from Sheet data. Read-only in MCP mode (does not send emails). Hard limit: 50 rows.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"sheet_id":    {Type: "string", Description: "Sheet ID"},
					"range":       {Type: "string", Description: "Sheet range (e.g. Sheet1!A:F)"},
					"email_col":   {Type: "integer", Description: "Column index for email (0-based)"},
					"subject_col": {Type: "integer", Description: "Column index for subject (0-based)"},
					"body_col":    {Type: "integer", Description: "Column index for body (0-based)"},
				},
				Required: []string{"sheet_id", "range"},
			},
		},
		{
			Name:        "workflow_parallel_schedule",
			Description: "Preview parallel 1-on-1 review scheduling. Read-only in MCP mode (does not create events).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title":      {Type: "string", Description: "Meeting title"},
					"attendees":  {Type: "string", Description: "Comma-separated attendee emails"},
					"duration":   {Type: "string", Description: "Meeting duration (e.g. 30m, 1h)"},
					"days_ahead": {Type: "integer", Description: "Days ahead to search (default 7)"},
				},
				Required: []string{"title", "attendees", "duration"},
			},
		},
		// Alias
		{
			Name:        "workflow_digest",
			Description: "Alias for workflow_weekly_digest. Generate a weekly digest. Read-only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"weeks": {Type: "integer", Description: "Number of weeks (default 1)"},
				},
			},
		},
	}
}

// CallWorkflowTool routes workflow tool calls.
// Returns (result, error, handled). handled=true means the tool name was recognised.
func (h *GWXHandler) CallWorkflowTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error, bool) {
	switch name {
	// FA-B
	case "workflow_standup":
		return h.workflowStandup(ctx, args)
	case "workflow_meeting_prep":
		return h.workflowMeetingPrep(ctx, args)
	case "workflow_weekly_digest", "workflow_digest":
		return h.workflowWeeklyDigest(ctx, args)
	case "workflow_context_boost":
		return h.workflowContextBoost(ctx, args)
	case "workflow_bug_intake":
		return h.workflowBugIntake(ctx, args)
	// FA-C: Test Matrix
	case "workflow_test_matrix_init":
		return h.workflowTestMatrixInit(ctx, args)
	case "workflow_test_matrix_sync":
		return h.workflowTestMatrixSync(ctx, args)
	case "workflow_test_matrix_stats":
		return h.workflowTestMatrixStats(ctx, args)
	// FA-C: Spec Health
	case "workflow_spec_health_init":
		return h.workflowSpecHealthInit(ctx, args)
	case "workflow_spec_health_record":
		return h.workflowSpecHealthRecord(ctx, args)
	case "workflow_spec_health_stats":
		return h.workflowSpecHealthStats(ctx, args)
	// FA-C: Sprint Board
	case "workflow_sprint_board_init":
		return h.workflowSprintBoardInit(ctx, args)
	case "workflow_sprint_board_ticket":
		return h.workflowSprintBoardTicket(ctx, args)
	case "workflow_sprint_board_stats":
		return h.workflowSprintBoardStats(ctx, args)
	// FA-D
	case "workflow_review_notify":
		return h.workflowReviewNotify(ctx, args)
	case "workflow_email_from_doc":
		return h.workflowEmailFromDoc(ctx, args)
	case "workflow_sheet_to_email":
		return h.workflowSheetToEmail(ctx, args)
	case "workflow_parallel_schedule":
		return h.workflowParallelSchedule(ctx, args)
	default:
		return nil, nil, false
	}
}

// --- FA-B handlers ---

func (h *GWXHandler) workflowStandup(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunStandup(ctx, h.client, workflow.StandupOpts{
		Days:  intArg(args, "days", 1),
		IsMCP: true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowMeetingPrep(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunMeetingPrep(ctx, h.client, workflow.MeetingPrepOpts{
		Meeting: strArg(args, "meeting"),
		Days:    intArg(args, "days", 1),
		IsMCP:   true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowWeeklyDigest(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunWeeklyDigest(ctx, h.client, workflow.WeeklyDigestOpts{
		Weeks: intArg(args, "weeks", 1),
		IsMCP: true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowContextBoost(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunContextBoost(ctx, h.client, workflow.ContextBoostOpts{
		Topic: strArg(args, "topic"),
		Days:  intArg(args, "days", 14),
		Limit: intArg(args, "limit", 10),
		IsMCP: true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowBugIntake(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunBugIntake(ctx, h.client, workflow.BugIntakeOpts{
		BugID: strArg(args, "bug_id"),
		After: strArg(args, "after"),
		IsMCP: true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

// --- FA-C: Test Matrix handlers ---

func (h *GWXHandler) workflowTestMatrixInit(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunTestMatrix(ctx, h.client, workflow.TestMatrixOpts{
		Action:  "init",
		Feature: strArg(args, "feature"),
		IsMCP:   true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowTestMatrixSync(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunTestMatrix(ctx, h.client, workflow.TestMatrixOpts{
		Action:  "sync",
		SheetID: strArg(args, "sheet_id"),
		File:    strArg(args, "file"),
		IsMCP:   true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowTestMatrixStats(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunTestMatrix(ctx, h.client, workflow.TestMatrixOpts{
		Action:  "stats",
		SheetID: strArg(args, "sheet_id"),
		IsMCP:   true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

// --- FA-C: Spec Health handlers ---

func (h *GWXHandler) workflowSpecHealthInit(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunSpecHealth(ctx, h.client, workflow.SpecHealthOpts{
		Action:  "init",
		Feature: strArg(args, "feature"),
		IsMCP:   true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowSpecHealthRecord(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunSpecHealth(ctx, h.client, workflow.SpecHealthOpts{
		Action:     "record",
		SheetID:    strArg(args, "sheet_id"),
		SpecFolder: strArg(args, "spec_folder"),
		IsMCP:      true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowSpecHealthStats(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunSpecHealth(ctx, h.client, workflow.SpecHealthOpts{
		Action:     "stats",
		SheetID:    strArg(args, "sheet_id"),
		SpecFolder: strArg(args, "spec_folder"),
		IsMCP:      true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

// --- FA-C: Sprint Board handlers ---

func (h *GWXHandler) workflowSprintBoardInit(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunSprintBoard(ctx, h.client, workflow.SprintBoardOpts{
		Action:  "init",
		Feature: strArg(args, "feature"),
		IsMCP:   true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowSprintBoardTicket(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunSprintBoard(ctx, h.client, workflow.SprintBoardOpts{
		Action:   "ticket",
		SheetID:  strArg(args, "sheet_id"),
		Title:    strArg(args, "title"),
		Assignee: strArg(args, "assignee"),
		Priority: strArg(args, "priority"),
		IsMCP:    true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowSprintBoardStats(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunSprintBoard(ctx, h.client, workflow.SprintBoardOpts{
		Action:  "stats",
		SheetID: strArg(args, "sheet_id"),
		IsMCP:   true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

// --- FA-D handlers ---

func (h *GWXHandler) workflowReviewNotify(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunReviewNotify(ctx, h.client, workflow.ReviewNotifyOpts{
		SpecFolder: strArg(args, "spec_folder"),
		Reviewers:  splitArg(args, "reviewers"),
		Channel:    strArg(args, "channel"),
		IsMCP:      true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowEmailFromDoc(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunEmailFromDoc(ctx, h.client, workflow.EmailFromDocOpts{
		DocID:      strArg(args, "doc_id"),
		Recipients: splitArg(args, "recipients"),
		Subject:    strArg(args, "subject"),
		IsMCP:      true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowSheetToEmail(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunSheetToEmail(ctx, h.client, workflow.SheetToEmailOpts{
		SheetID:    strArg(args, "sheet_id"),
		Range:      strArg(args, "range"),
		EmailCol:   intArg(args, "email_col", 0),
		SubjectCol: intArg(args, "subject_col", 1),
		BodyCol:    intArg(args, "body_col", 2),
		IsMCP:      true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) workflowParallelSchedule(ctx context.Context, args map[string]interface{}) (*ToolResult, error, bool) {
	result, err := workflow.RunParallelSchedule(ctx, h.client, workflow.ParallelScheduleOpts{
		Title:     strArg(args, "title"),
		Attendees: splitArg(args, "attendees"),
		Duration:  strArg(args, "duration"),
		DaysAhead: intArg(args, "days_ahead", 7),
		IsMCP:     true,
	})
	if err != nil {
		return nil, err, true
	}
	r, err := jsonResult(result)
	return r, err, true
}

func (h *GWXHandler) registerWorkflowTools(r map[string]ToolHandler) {
	r["workflow_standup"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowStandup(ctx, args)
		return result, err
	}
	r["workflow_meeting_prep"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowMeetingPrep(ctx, args)
		return result, err
	}
	r["workflow_weekly_digest"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowWeeklyDigest(ctx, args)
		return result, err
	}
	r["workflow_digest"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowWeeklyDigest(ctx, args)
		return result, err
	}
	r["workflow_context_boost"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowContextBoost(ctx, args)
		return result, err
	}
	r["workflow_bug_intake"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowBugIntake(ctx, args)
		return result, err
	}
	r["workflow_test_matrix_init"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowTestMatrixInit(ctx, args)
		return result, err
	}
	r["workflow_test_matrix_sync"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowTestMatrixSync(ctx, args)
		return result, err
	}
	r["workflow_test_matrix_stats"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowTestMatrixStats(ctx, args)
		return result, err
	}
	r["workflow_spec_health_init"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowSpecHealthInit(ctx, args)
		return result, err
	}
	r["workflow_spec_health_record"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowSpecHealthRecord(ctx, args)
		return result, err
	}
	r["workflow_spec_health_stats"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowSpecHealthStats(ctx, args)
		return result, err
	}
	r["workflow_sprint_board_init"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowSprintBoardInit(ctx, args)
		return result, err
	}
	r["workflow_sprint_board_ticket"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowSprintBoardTicket(ctx, args)
		return result, err
	}
	r["workflow_sprint_board_stats"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowSprintBoardStats(ctx, args)
		return result, err
	}
	r["workflow_review_notify"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowReviewNotify(ctx, args)
		return result, err
	}
	r["workflow_email_from_doc"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowEmailFromDoc(ctx, args)
		return result, err
	}
	r["workflow_sheet_to_email"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowSheetToEmail(ctx, args)
		return result, err
	}
	r["workflow_parallel_schedule"] = func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		result, err, _ := h.workflowParallelSchedule(ctx, args)
		return result, err
	}
}
