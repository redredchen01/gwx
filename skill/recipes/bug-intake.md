---
name: bug-intake
description: "Parse bug report emails from Gmail digest, extract structured bug info, and auto-inject into S0 as a bugfix SOP."
services: [gmail]
safety_tier: green
combo: [gwx-gmail-digest, sop-s0]
---

# Bug Intake Workflow

## Purpose

Turn bug report emails into structured S0 bugfix SOPs automatically. Instead of manually reading emails and copy-pasting bug descriptions, this workflow scans Gmail for bug reports, extracts key info (reproduction steps, expected vs actual), and injects them into S0 with `work_type: bugfix`.

## Trigger
User says: "bug intake", "bug email", "信件裡的 bug", "從 email 開 bug", "bug 報告轉 SOP"

## Steps

### Step 1: Search for bug report emails (🟢)
```bash
gwx gmail search "subject:(bug OR error OR issue OR 問題 OR 壞了 OR crash) -in:sent" --limit 10 --json
```

Or with date filter:
```bash
gwx gmail search "subject:(bug OR error OR issue) after:{date}" --limit 10 --json
```

### Step 2: Digest and categorize (🟢)
```bash
gwx gmail digest --limit 20 --json
```

Filter for bug-related categories. Present to user:

```
Found {N} potential bug reports:

1. [2026-03-17] from:alice@co.com — "API returns 500 on invoice creation"
2. [2026-03-16] from:bob@co.com — "Dashboard chart not loading after deploy"
3. [2026-03-15] from:ci@github.com — "Build failed: test_auth_refresh"

Select which to process (1-3, or 'all'):
```

### Step 3: Extract bug details (🟢)

For selected email(s):
```bash
gwx gmail get {message_id} --json
```

Parse email body to extract:
- **Reporter**: sender name + email
- **Summary**: email subject
- **Description**: email body (first 500 chars)
- **Reproduction steps**: look for numbered lists or "steps to reproduce"
- **Expected behavior**: look for "expected" / "should"
- **Actual behavior**: look for "actual" / "instead" / "but"
- **Environment**: look for version numbers, OS, browser info
- **Severity signal**: keywords like "blocker", "critical", "minor"

### Step 4: Compile S0 input

Structure extracted info into S0-ready format:

```markdown
## Bug Report (from email)

**Reporter**: {sender}
**Date**: {email_date}
**Subject**: {subject}

### Description
{extracted description}

### Reproduction Steps
{extracted steps or "Not provided — needs clarification"}

### Expected vs Actual
- Expected: {extracted or "Needs clarification"}
- Actual: {extracted or "Needs clarification"}

### Environment
{extracted or "Not specified"}
```

### Step 5: Inject into S0

Pass compiled bug report to S0 with pre-set work_type:

```
Skill(skill: "s0-understand", args: "{compiled bug report}\n\nwork_type: bugfix")
```

The `requirement-analyst` will:
- Skip work_type detection (already set to `bugfix`)
- Focus on reproduction steps and expected vs actual
- Ask targeted questions about missing fields
- Lean toward Quick Mode (unless cross-module impact)

## Batch Mode

If user selects multiple bugs:
1. Process each sequentially
2. For each: extract → compile → inject S0
3. If bugs are related, suggest grouping into single SOP

## Notes
- Steps 1-3 are all 🟢 read-only
- Step 5 enters S0 which has its own 🔴 gate
- If email body is too short or unclear, the requirement-analyst will ask follow-up questions
- Works with any email format — extraction is best-effort, missing fields become "Needs clarification"
