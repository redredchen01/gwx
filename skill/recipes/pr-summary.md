---
name: pr-summary
description: "List open pull requests for a repository and generate a summary document."
services: [github, docs]
safety_tier: yellow
---

# PR Summary Workflow

## Trigger
User says: "PR summary", "pull request report", "PR 狀態", "列出所有 PR"

## Steps

### Step 1: List open pull requests
```bash
gwx github pulls owner/repo --state open --limit 50 --json
```
Collect all open PRs with their title, author, created date, and draft status.

### Step 2: Compile summary

```
# PR Summary — {repo} — {today's date}

## Open Pull Requests ({count})

| # | Title | Author | Created | Draft |
|---|-------|--------|---------|-------|
| {number} | {title} | {user} | {created_at} | {draft} |

## Statistics
- Total open: {count}
- Drafts: {draft_count}
- Oldest open: {oldest_title} ({days} days)
```

### Step 3: Create Google Doc (optional, confirm first)
```bash
gwx docs create --title "PR Summary — {repo} — {date}" --body "{summary}" --json
```

## Notes
- Step 1 is read-only
- Step 3 requires Google Docs auth and user confirmation
- Works with any GitHub repository the user's PAT has access to
