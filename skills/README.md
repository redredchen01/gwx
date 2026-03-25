# gwx Skills

Skills are YAML pipelines that chain gwx tools into reusable workflows.

## Skill Catalog

### Google Workspace (`google-*`)

| Skill | Services | Description |
|-------|----------|-------------|
| `google-morning-brief` | Gmail, Calendar, Tasks | Morning standup brief — unread emails, today's calendar, and pending tasks in one view. |
| `google-email-digest` | Gmail | Smart email digest — categorizes recent emails by sender and type with a summary. |
| `google-meeting-notes` | Calendar, Docs | Generate meeting note templates from upcoming calendar events. |
| `google-task-report` | Tasks | Task productivity report — lists all task lists and items. |
| `google-drive-audit` | Drive | Drive file audit — lists files sorted by modification time. |
| `google-doc-from-sheet` | Sheets, Docs | Generate a Google Doc from spreadsheet data. |
| `google-sheet-compare` | Sheets | Compare two spreadsheet tabs with cell-level diffs. |
| `google-contact-export` | Contacts, Sheets | Export contacts to a Google Sheet. |
| `google-invoice-log` | Gmail, Sheets | Search Gmail for invoices/receipts and log them to a Sheet. |
| `google-chat-summary` | Chat | Read recent messages from a Google Chat space. |
| `google-ga4-realtime` | Analytics | GA4 realtime snapshot — active users by country and device. |
| `google-seo-daily` | Search Console, Analytics, Sheets | Daily SEO snapshot — Search Console + GA4 traffic, saved to Sheets. |
| `google-bq-to-sheet` | BigQuery, Sheets | Run a BigQuery SQL query and archive results to a Sheet. |

### GitHub (`github-*`)

| Skill | Services | Description |
|-------|----------|-------------|
| `github-issue-to-sheet` | GitHub, Sheets | Export GitHub Issues to a Google Sheet for project tracking. |
| `github-pr-to-slack` | GitHub, Slack | Post open PR count to a Slack channel. |
| `github-ci-alert` | GitHub, Slack, Gmail | CI failure alerts via Slack and/or email. |

### Cross-Provider (`cross-*`)

| Skill | Services | Description |
|-------|----------|-------------|
| `cross-client-360` | Gmail, Drive, Contacts | Client 360 view — aggregates emails, files, and contacts for a client keyword. |
| `cross-full-context` | Gmail, Drive, Slack, Notion, GitHub | Cross-platform keyword search across 5 services in parallel. |
| `cross-github-standup` | GitHub, Gmail, Calendar | Developer standup — PRs + unread email + calendar in parallel. |
| `cross-daily-report` | Gmail, Calendar, Tasks, Search Console, Analytics, Sheets | Combined daily report — morning brief + SEO snapshot. |
| `cross-weekly-sync` | Gmail, Calendar, Analytics, GitHub | Automated weekly report aggregating email, events, GA4, and PRs. |
| `cross-onboard-checklist` | Drive, Docs, Calendar, Slack, Gmail | New hire onboarding — folder, welcome doc, meeting, and notifications. |
| `cross-form-to-slack` | Forms, Slack | Google Forms response notifications to Slack. |
| `cross-notion-to-sheet` | Notion, Sheets | Sync a Notion database to a Google Sheet. |

## Structure

```yaml
name: my-skill
version: "1.0"
description: "What the skill does"

meta:
  author: gwx
  category: google
  tags: gmail,sheets

inputs:
  - name: query
    type: string
    required: true
    description: "Search term"

steps:
  - id: search
    tool: gmail_search
    args:
      query: "{{.input.query}}"

  - id: save
    tool: sheets_append
    args:
      spreadsheet_id: "{{.input.sheet-id}}"
      values: "{{.steps.search}}"
    on_fail: skip

output: "{{.steps.search}}"
```

## Managing Skills

```bash
# List all installed skills
gwx skill list

# Validate a skill file
gwx skill validate ./my-skill.yaml

# Install from file or URL
gwx skill install ./my-skill.yaml
gwx skill install https://raw.githubusercontent.com/user/repo/main/skills/foo.yaml

# Inspect or remove
gwx skill inspect my-skill
gwx skill remove my-skill
```

## Skill Locations

- **User skills**: `~/.config/gwx/skills/` (installed via `gwx skill install`)
- **Project skills**: `./skills/` in your project root (committed to git)

Project skills override user skills with the same name.
