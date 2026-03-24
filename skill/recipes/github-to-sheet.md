---
name: github-to-sheet
description: "Export GitHub issues from a repository to a Google Sheet for tracking."
services: [github, sheets]
safety_tier: yellow
---

# GitHub Issues to Sheet Workflow

## Trigger
User says: "issues to sheet", "export issues", "issues 匯出", "GitHub 議題到試算表"

## Steps

### Step 1: List issues from GitHub
```bash
gwx github issues owner/repo --state open --limit 100 --json
```
Collect all issues with number, title, state, labels, created_at, user, comments count.

### Step 2: Format as rows
Transform each issue into a row:
```json
[
  ["#", "Title", "State", "Labels", "Author", "Created", "Comments"],
  [1, "Bug in login", "open", "bug,priority", "octocat", "2026-03-20", 3]
]
```

### Step 3: Append to Google Sheet (confirm first)
```bash
gwx sheets append --spreadsheet-id SHEET_ID --range "Issues!A:G" --values '{rows_json}' --json
```

Or use smart append for validation:
```bash
gwx sheets smart-append --spreadsheet-id SHEET_ID --range "Issues!A:G" --values '{rows_json}' --json
```

## Notes
- Step 1 is read-only (GitHub)
- Step 3 modifies Google Sheet data — requires confirmation
- If the sheet doesn't exist yet, user needs to create it first with the header row
- Use `gwx sheets describe` first to validate column structure if appending to existing sheet
