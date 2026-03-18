---
name: test-matrix
description: "Use Google Sheets as a live test tracking matrix for S6 — auto-populate test cases from S3 plan, update results in real-time, provide stats and diff for PM/QA visibility."
services: [sheets]
safety_tier: green/yellow
combo: [gwx-sheets, sop-s6]
---

# Test Matrix Workflow

## Purpose

Turn a Google Sheet into a **live test tracking dashboard** that stays in sync with the SOP pipeline. S3 populates test cases, S6 updates results, anyone with Sheet access can see real-time progress — no repo access needed.

## Trigger
User says: "test matrix", "測試追蹤", "測試 sheet", "建測試追蹤表", "sync tests to sheet"

## Prerequisites
- gwx authenticated (`gwx auth status`)
- An active SOP with S3 completed (has `s3_implementation_plan.md` with tdd_plan)
- Google Sheets API access

## Steps

### Step 1: Create or locate Sheet (🟡 confirm)

**New Sheet:**
```bash
gwx sheets create --title "{feature_name} — Test Matrix" --json
```

**Or reuse existing:**
```bash
gwx drive search "name contains 'Test Matrix'" --limit 5 --json
```

Save the `SHEET_ID` for subsequent steps.

### Step 2: Initialize Sheet structure (🟡)

Set up the header row:
```bash
gwx sheets update {SHEET_ID} "A1:J1" --values '[["TC-ID", "Task", "Test Case", "Type", "Priority", "Status", "Result", "Error Summary", "Fixed In", "Last Run"]]' --json
```

**Column definitions:**
| Column | Purpose | Values |
|--------|---------|--------|
| TC-ID | Unique test case ID | TC-001, TC-002... |
| Task | S3 task reference | T1, T2... |
| Test Case | Test description | From tdd_plan.test_cases |
| Type | Test type | unit / integration / e2e / manual |
| Priority | Test priority | P0 / P1 / P2 |
| Status | Current status | pending / running / passed / failed / skipped |
| Result | Pass/fail result | ✅ / ❌ / ⏭️ |
| Error Summary | Failure message | First 100 chars of error |
| Fixed In | Fix commit hash | git short hash |
| Last Run | Timestamp | ISO 8601 |

### Step 3: Populate from S3 plan (🟡)

Read `s3_implementation_plan.md` and extract all `tdd_plan` entries:

```bash
# For each task with tdd_plan:
gwx sheets smart-append {SHEET_ID} "A:J" --values '[
  ["TC-001", "T1", "UserService.create should validate email", "unit", "P0", "pending", "", "", "", ""],
  ["TC-002", "T1", "UserService.create should hash password", "unit", "P1", "pending", "", "", "", ""]
]' --json
```

Use `smart-append` to validate column structure automatically.

### Step 4: S6 real-time sync (🟡)

During S6 test execution, update results after each test run:

```bash
# Update single test case result
gwx sheets update {SHEET_ID} "F{row}:J{row}" --values '[["passed", "✅", "", "", "2026-03-18T14:30:00"]]' --json

# Or for failure:
gwx sheets update {SHEET_ID} "F{row}:J{row}" --values '[["failed", "❌", "Expected 200 got 500", "", "2026-03-18T14:30:00"]]' --json
```

After each test batch completes:
```bash
# Get current stats
gwx sheets stats {SHEET_ID} --json
```

Output example:
```
Status: passed=12, failed=3, pending=5, skipped=1
Result: ✅=12, ❌=3, ⏭️=1
Type: unit=15, integration=4, e2e=2
```

### Step 5: Defect loop tracking

When S6 enters repair loop:

```bash
# After fix, update the Fixed In column
gwx sheets update {SHEET_ID} "I{row}" --values '[["abc1234"]]' --json

# Re-run and update result
gwx sheets update {SHEET_ID} "F{row}:J{row}" --values '[["passed", "✅", "", "abc1234", "2026-03-18T15:00:00"]]' --json
```

### Step 6: Final report (🟢)

After S6 completes:

```bash
# Full stats
gwx sheets stats {SHEET_ID} --json

# Diff from initial state (if using copy-tab for baseline)
gwx sheets diff {SHEET_ID} --tab1 "Baseline" --tab2 "Current" --json

# Export for archival
gwx sheets export {SHEET_ID} --format csv --output "dev/specs/{feature}/s6_test_results.csv"
```

## Integration with S6

The `test-engineer` agent should:
1. **Before running tests**: Read the Sheet to know what to test
2. **After each test command**: Update the corresponding row
3. **On failure**: Fill Error Summary column
4. **On fix**: Fill Fixed In column and re-run
5. **On completion**: Run `stats` and include Sheet link in S6 report

## Sheet Sharing

After creation, optionally share with team:
```bash
# 🔴 Hard gate — requires explicit confirmation
gwx drive share {SHEET_ID} --email "team@example.com" --role writer --json
```

## Notes
- Step 1-3 are one-time setup (🟡 confirm for create/write operations)
- Step 4-5 are repeated during S6 (🟡 for writes)
- Step 6 is read-only (🟢)
- `smart-append` validates that data matches the header schema
- `stats` provides instant burn-down visibility
- `diff` lets you compare before/after repair loops
- Sheet URL should be recorded in `sdd_context.json` under `stages.s6.output.test_matrix_url`
