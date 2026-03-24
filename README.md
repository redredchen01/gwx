# gwx — Google Workspace CLI for Humans and Agents

A unified CLI + MCP server for Google Workspace, GitHub, Slack, and Notion — Gmail, Calendar, Drive, Docs, Sheets, Tasks, Contacts, Chat, Analytics, Search Console, Slides, Forms, BigQuery, GitHub, Slack, Notion. Designed for both human use and LLM agent integration.

**140 CLI commands · 123 MCP tools · 16 services**

## Install

```bash
# npm (recommended — auto-downloads pre-built binary)
npm install -g gwx-cli

# Go
go install github.com/redredchen01/gwx/cmd/gwx@latest

# From source
git clone https://github.com/redredchen01/gwx.git
cd gwx && make install
```

## Quick Start

```bash
# 1. Set up Google Cloud credentials (interactive wizard)
gwx onboard

# 2. Start using
gwx gmail list --limit 5
gwx calendar agenda
gwx drive list
gwx sheets read SHEET_ID "A1:C10"
gwx docs get DOC_ID

# 3. Shortcuts
gwx ls                         # → drive list
gwx search "keyword"           # → gmail search
gwx send --to a@b.com ...     # → gmail send
gwx find "topic"               # → unified search (Gmail + Drive)
gwx context "project"          # → gather context (Gmail + Drive + Calendar)
```

## Commands (140)

| Service | Commands |
|---------|----------|
| **Gmail** (11) | `list` `get` `search` `labels` `send` `draft` `reply` `digest` `archive` `label` `forward` |
| **Calendar** (6) | `agenda` `list` `create` `update` `delete` `find-slot` |
| **Drive** (6) | `list` `search` `upload` `download` `share` `mkdir` |
| **Docs** (8) | `get` `create` `append` `search` `replace` `template` `from-sheet` `export` |
| **Sheets** (15) | `read` `info` `describe` `stats` `search` `filter` `diff` `append` `smart-append` `update` `clear` `copy-tab` `export` `import` `create` |
| **Tasks** (5) | `list` `lists` `create` `complete` `delete` |
| **Contacts** (3) | `list` `search` `get` |
| **Chat** (3) | `spaces` `send` `messages` |
| **Analytics** (4) | `report` `realtime` `properties` `audiences` |
| **Search Console** (5) | `query` `sites` `inspect` `sitemaps` `index-status` |
| **Slides** (6) | `get` `list` `create` `duplicate` `export` `from-sheet` |
| **Forms** (3) | `get` `responses` `response` |
| **BigQuery** (4) | `query` `datasets` `tables` `describe` |
| **GitHub** (10) | `login` `logout` `status` `repos` `issues` `pulls` `pull` `runs` `notify` `create issue` |
| **Slack** (7) | `login` `status` `channels` `send` `messages` `search` `users` |
| **Notion** (7) | `login` `status` `search` `page` `create` `databases` `query` |
| **Skill** (8) | `list` `inspect` `validate` `run` `create` `test` `install` `remove` |
| **Config** (3) | `set` `get` `list` |
| **Workflow** (13) | `standup` `meeting-prep` + `workflow` subgroup: `weekly-digest` `context-boost` `bug-intake` `test-matrix` `spec-health` `sprint-board` `review-notify` `email-from-doc` `sheet-to-email` `parallel-schedule` |
| **Cross-service** (2) | `find` (unified search) · `context` (gather context) |
| **Pipeline** (1) | `pipe` (chain commands via JSON stdin/stdout) |
| **System** (9) | `auth login/logout/status` `onboard` `agent exit-codes` `schema` `mcp-server` `version` |

## Highlights

### Smart Sheets
```bash
# Analyze column structure before writing
gwx sheets describe SHEET_ID
# → [0] 花名 (freetext, REQUIRED)
# → [2] 完成情况 (enum): 已完成 / 持续中 / DONE

# Validate + append (catches wrong enum values, missing required fields)
gwx sheets smart-append SHEET_ID "Sheet1!A:F" --values '[["Alice","Plan X","已完成","","",""]]'

# Column statistics
gwx sheets stats SHEET_ID
# → 完成情况 → 已完成: 9, 持续中: 5, DONE: 2

# Compare two weeks
gwx sheets diff SHEET_ID --from "第1周" --to "第2周"
# → 高睿: 完成情况 已完成 → 10%

# Export / Import
gwx sheets export SHEET_ID "A:D" --export-format csv -o report.csv
gwx sheets import SHEET_ID "A1" -i data.csv --import-format csv
```

### Gmail Intelligence
```bash
# Smart digest — groups by sender, categorizes CI/transactional/personal
gwx gmail digest --limit 30
# → 14 CI notifications (can batch archive). 3 personal.

# Batch archive noisy notifications
gwx gmail archive "subject:Run failed" --limit 50

# Batch label — add/remove labels on matching messages
gwx gmail label "from:github" --add CI --remove INBOX --limit 100

# Forward a message
gwx gmail forward MESSAGE_ID --to colleague@company.com
```

### Docs Template Engine
```bash
# Create documents from templates with {{var}} replacement
gwx docs template TEMPLATE_DOC_ID -v '{"name":"Alice","date":"2026-03-17"}'
```

### Google Analytics 4
```bash
# GA4 report — sessions by country for last 7 days
gwx analytics report --metrics sessions,activeUsers --dimensions country --start-date 7daysAgo

# Real-time active users
gwx analytics realtime --metrics activeUsers

# List all GA4 properties
gwx analytics properties

# Set default property (one-time)
gwx config set analytics.default-property properties/123456
```

### Google Search Console
```bash
# Search performance — top queries last 28 days
gwx searchconsole query --start-date 2026-02-19 --dimensions query --limit 20

# Check URL index status
gwx searchconsole inspect --site https://example.com https://example.com/page

# List sitemaps
gwx searchconsole sitemaps --site https://example.com

# Set default site (one-time)
gwx config set searchconsole.default-site https://example.com
```

### Google Slides
```bash
# List presentations
gwx slides list

# Get presentation structure
gwx slides get PRESENTATION_ID

# Create a new presentation
gwx slides create --title "Q2 Report"

# Export to PDF
gwx slides export PRESENTATION_ID --export-format pdf -o report.pdf

# Duplicate a presentation
gwx slides duplicate PRESENTATION_ID --title "Copy of Report"

# Generate from Sheet data + template (replaces {{placeholders}})
gwx slides from-sheet --template TEMPLATE_ID --sheet-id SHEET_ID --range "A:D"
```

### Google Forms
```bash
# Get form structure
gwx forms get FORM_ID

# List all responses
gwx forms responses FORM_ID --limit 50

# Get a single response
gwx forms response FORM_ID RESPONSE_ID
```

### BigQuery
```bash
# Run a SQL query
gwx bigquery query "SELECT * FROM dataset.table LIMIT 10" --project my-project

# List datasets
gwx bigquery datasets --project my-project

# List tables in a dataset
gwx bigquery tables --project my-project --dataset my_dataset

# Describe a table's schema
gwx bigquery describe my_table --project my-project --dataset my_dataset

# Set default project (one-time)
gwx config set bigquery.default-project my-project
```

### GitHub Integration
```bash
# Authenticate with personal access token
gwx github login --token ghp_xxx

# List your repositories
gwx github repos --limit 10

# List issues / pull requests
gwx github issues owner/repo --state open
gwx github pulls owner/repo

# Get a specific PR with details
gwx github pull owner/repo 42

# List CI workflow runs
gwx github runs owner/repo

# List notifications
gwx github notify

# Create an issue
gwx github create issue owner/repo --title "Bug report" --body "Details..." --labels bug,urgent
```

### Slack Integration
```bash
# Authenticate with bot token
gwx slack login xoxb-xxx

# List channels
gwx slack channels

# Send a message
gwx slack send "Hello team" --channel "#general"

# Read channel history
gwx slack messages C01234567 --limit 20

# Search across workspace
gwx slack search "deploy error"

# List workspace users
gwx slack users
```

### Notion Integration
```bash
# Authenticate with integration token
gwx notion login ntn_xxx

# Search pages
gwx notion search "project plan"

# Get a page
gwx notion page PAGE_ID

# List databases
gwx notion databases

# Query a database with filter
gwx notion query DATABASE_ID --filter '{"property":"Status","select":{"equals":"Done"}}'

# Create a page in a database
gwx notion create --parent DATABASE_ID --title "New item"
```

### Built-in Workflows
```bash
# Daily standup — aggregate Git + Gmail + Calendar + Tasks
gwx standup

# Push standup to Google Chat
gwx standup --execute --push chat:spaces/AAAA

# Meeting prep — attendees, recent emails, related docs
gwx meeting-prep "Weekly Sync"

# Weekly digest
gwx workflow weekly-digest

# Deep context gathering
gwx workflow context-boost "Project X"

# Test matrix in Sheets (init → sync → stats)
gwx workflow test-matrix init --feature "invoice"
gwx workflow test-matrix stats

# Sprint board in Sheets
gwx workflow sprint-board init --feature "Q2"
gwx workflow sprint-board ticket --title "Fix login" --priority P1

# Send review notification (requires --execute)
gwx workflow review-notify --spec-folder dev/specs/xxx --reviewers a@co.com --execute --channel email

# Mail merge from Sheets (hard limit: 50 rows)
gwx workflow sheet-to-email --sheet-id XXX --range "A:F" --email-col 0 --subject-col 1 --body-col 2 --execute
```

> All workflows default to **read-only** (JSON output). Add `--execute` for actions. MCP tools are always read-only.

### Cross-Service Operations
```bash
# Search across Gmail + Drive simultaneously
gwx find "keyword"

# Gather all context for a topic (Gmail + Drive + Calendar)
gwx context "project-name" --days 7
```

### Command Pipeline
```bash
# Chain commands — each stage passes JSON to the next
gwx pipe "gmail search 'invoice' | sheets append SHEET_ID A:C"

# Multi-stage pipeline
gwx pipe "calendar agenda --days 7 | docs create --title 'Weekly Report'"
```

> Each stage runs as a subprocess with `--format json`. Output of stage N becomes stdin of stage N+1.

### Skill DSL — YAML-Defined Workflows

Define multi-step workflows in YAML — no Go code, no recompilation:

```yaml
# skills/morning-brief.yaml
name: morning-brief
description: "Daily briefing — inbox + calendar + tasks"
inputs:
  - name: email-limit
    type: int
    default: "10"
steps:
  - id: inbox
    tool: gmail_list
    args: { limit: "{{.input.email-limit}}", unread: "true" }
  - id: today
    tool: calendar_agenda
    args: { days: "1" }
    on_fail: skip
  - id: tasks
    tool: tasks_list
    args: { show_completed: "false" }
    on_fail: skip
```

```bash
# Manage skills
gwx skill list                           # List installed skills
gwx skill validate skills/my-skill.yaml  # Validate YAML
gwx skill inspect morning-brief          # Show skill details
gwx skill run morning-brief -p email-limit=20  # Run a skill
gwx skill create my-new-skill            # Scaffold a new skill
gwx skill test morning-brief             # Test with mock data
gwx skill install ./path/to/skill.yaml   # Install from file or URL
gwx skill remove old-skill               # Remove an installed skill

# Skills auto-register as MCP tools (skill_morning-brief)
# Drop YAML files in ./skills/ or ~/.config/gwx/skills/
```

**28 built-in skills**: morning-brief, client-360, invoice-log, seo-daily, meeting-notes, email-digest, drive-audit, sheet-compare, contact-export, task-report, chat-summary, ga4-realtime, doc-from-sheet, github-pr-digest, github-issue-triage, slack-standup, slack-channel-archive, notion-weekly, notion-inbox, bq-daily-report, forms-survey-summary, cross-provider-standup, multi-inbox-digest, and more.

## MCP Server (123+ Tools)

Native Claude integration — no Bash needed. Multi-provider support across Google, GitHub, Slack, and Notion:

```bash
# Start MCP server
gwx mcp-server
```

Add to `~/.claude/settings.json`:
```json
{
  "mcpServers": {
    "gwx": {
      "command": "gwx",
      "args": ["mcp-server"]
    }
  }
}
```

All CLI commands are available as MCP tools. Claude can directly call `sheets_describe`, `gmail_digest`, `gmail_batch_label`, `analytics_report`, `searchconsole_query`, `config_set`, `workflow_standup`, `workflow_meeting_prep`, `github_repos`, `github_issues`, `github_pulls`, `slack_channels`, `slack_send`, `notion_search`, `notion_query`, `bigquery_query`, `forms_get`, `skill_run`, etc.

## Multi-Provider Auth

gwx supports multiple providers beyond Google Workspace. Each provider stores tokens independently in the OS keyring.

```bash
# Google (existing OAuth flow)
gwx onboard

# GitHub
gwx github login --token ghp_xxx

# Slack
gwx slack login xoxb-xxx

# Notion
gwx notion login ntn_xxx

# Check status for each
gwx auth status           # Google
gwx github status         # GitHub
gwx slack status          # Slack
gwx notion status         # Notion
```

## Claude Code Skill

```bash
# Install CLI + Claude Code skill + workflow recipes
./install.sh
```

Installs skill definitions to `~/.claude/commands/` and agent definitions to `~/.claude/agents/` for automatic trigger on Google Workspace keywords.

### Combo Skills — gwx × Claude Code Workflows

Eight power combos that chain gwx with Claude Code's SOP pipeline:

#### `/context-boost` — Google Workspace Context → S0

Before starting a new feature, automatically gather all related emails, docs, and calendar events — then inject them into the requirement discussion.

```bash
# In Claude Code:
/context-boost 幫我做一個 invoice 自動寄送功能
```

What happens:
1. Extracts keywords from your requirement (`invoice`, `寄送`)
2. Runs `gwx context` across Gmail + Drive + Calendar (past 14 days)
3. Deep-dives into top emails and docs
4. Compiles a Context Briefing (stakeholders, timeline signals, existing decisions)
5. Feeds it into S0 — the requirement-analyst references real data instead of guessing

> All context-gathering is read-only. Falls back to standard S0 if gwx is not connected.

#### `/test-matrix` — Google Sheets as Live Test Dashboard

Turn S6 test execution into a Google Sheet that PM/QA can follow in real-time.

```bash
# After S3 (plan) is done — initialize the matrix:
/test-matrix dev/specs/my-feature

# During S6 — sync results (auto-triggered if Sheet exists):
/test-matrix 1BxC...sheetId
```

What happens:
1. Creates a Google Sheet: `{feature} — Test Matrix`
2. Populates rows from S3's `tdd_plan` (TC-ID, task, test case, type, priority)
3. During S6, updates Status/Result columns after each test run
4. `gwx sheets stats` gives instant burn-down: `passed=12, failed=3, pending=5`
5. Exports final CSV to `dev/specs/{feature}/s6_test_results.csv`

| Column | Purpose |
|--------|---------|
| TC-ID | Unique test case ID (TC-001) |
| Task | S3 task reference (T1, T2) |
| Test Case | Test description |
| Type | unit / integration / e2e / manual |
| Status | pending / running / passed / failed / skipped |
| Result | Pass / Fail / Skip |
| Error Summary | First 100 chars of failure message |
| Fixed In | Git commit hash of the fix |

#### `/standup` — AI Daily Standup Report

Merges git activity + SOP progress + Google Workspace into one standup.

```bash
# Display in terminal (default):
/standup

# Push to Google Chat (requires confirmation):
/standup chat:spaces/AAAA

# Draft as email:
/standup email:team@company.com
```

Sample output:
```
# Daily Standup — 2026-03-18

## Done (Yesterday)
### Development
- 3 commits on `feature/invoice`: add invoice model, add API endpoint, add tests
- SOP: invoice-auto-send — S4 → S5 (code review passed)

### Communication
- Sent 2 emails: "Invoice API spec review", "Sprint planning follow-up"
- Attended 1 meeting: "Backend sync"
- Completed 1 task: "Write invoice schema migration"

## Plan (Today)
### Development
- invoice-auto-send: Continue S6 (testing)
### Meetings
- 10:00: "QA Review" (3 attendees)
### Tasks
- [ ] Add rate limiting to invoice endpoint (due: 2026-03-19)

## Blockers
- (none)
```

> Works without gwx — git-only standup still generates. Combine with `/loop 24h /standup` for daily automation.

#### `/bug-intake` — Gmail Bug Reports → S0 Bugfix

Scan Gmail for bug report emails, extract structured info, and auto-inject into S0 as a bugfix SOP.

```bash
/bug-intake                          # scan recent bug emails
/bug-intake --after 2026/03/15       # filter by date
/bug-intake OAuth token              # filter by keyword
```

What happens:
1. Searches Gmail for emails with bug/error/issue/crash in subject
2. Presents candidates — you pick which to process
3. Extracts reproduction steps, expected vs actual, environment
4. Injects into S0 with `work_type: bugfix` pre-set

> Batch mode: select multiple emails, related bugs auto-group into single SOP.

#### `/spec-health` — Spec Quality Dashboard

Track spec-audit results across all features in one Google Sheet — see quality trends over time.

```bash
/spec-health                         # view dashboard (or init on first run)
/spec-health record:dev/specs/auth   # record latest audit results
```

What happens:
1. Creates a 3-tab Sheet: Audit Log, Feature Summary, Trend
2. After each `/spec-audit`, appends P0/P1/P2 counts
3. Feature Summary shows health status: Healthy / Acceptable / Critical
4. `gwx sheets stats` gives instant quality report
5. `gwx sheets diff` compares quality across sprints

#### `/parallel-schedule` — Auto-Schedule Review Meetings

When running `/parallel-develop`, auto-find free slots and book review meetings for all worktrees.

```bash
/parallel-schedule --reviewers alice@co.com,bob@co.com
```

What happens:
1. Scans git worktrees for features at S4 completed
2. `gwx calendar find-slot` finds common free time for all reviewers
3. Proposes a schedule — you confirm
4. Creates calendar events with briefing docs attached

#### `/review-notify` — Push Review Results to Chat/Email

After S5 code review, instantly notify the team.

```bash
/review-notify chat:spaces/AAAA     # push to Google Chat
/review-notify email:team@co.com    # send via email
```

Sample notification:
```
Code Review Complete: invoice-auto-send
Result: PASS
Findings: 0 P0 · 1 P1 · 2 P2
Next: S6 testing
```

> Always requires explicit confirmation before sending (hard gate).

#### `/sprint-board` — Google Sheet as Kanban Board

Zero-cost project management. gwx is your Jira.

```bash
/sprint-board                        # init or view
/sprint-board ticket:invoice         # create ticket from S0
/sprint-board stats                  # burn-down metrics
/sprint-board archive                # archive current sprint
```

What happens:
1. Creates a Sheet with columns: Ticket, Feature, Type, Priority, Status, Assignee...
2. S0 creates tickets, S4 updates to `in-progress`, S5 to `review`, S6 to `testing`, S7 to `done`
3. `gwx sheets stats` → `done: 7, in-progress: 3, review: 1, blocked: 1`
4. `gwx sheets copy-tab` archives sprints, `diff` compares velocity across sprints

> Share the Sheet with your team — everyone sees real-time progress without repo access.

### Workflow Recipes

| Recipe | Trigger | Services | Tier |
|--------|---------|----------|------|
| meeting-prep | "meeting prep" | Calendar + Gmail + Drive | Green |
| weekly-digest | "weekly digest" | Gmail + Calendar + Tasks | Green |
| standup-report | "standup" | Gmail + Calendar + Tasks | Green |
| email-from-doc | "email from doc" | Docs + Gmail | Yellow/Red |
| sheet-to-email | "mail merge" | Sheets + Gmail | Red |
| **context-boost** | "context boost" | Gmail + Drive + Calendar | Green |
| **test-matrix** | "test matrix" | Sheets | Green/Yellow |
| **standup** | "standup" | Gmail + Calendar + Tasks + Git + Chat | Green/Red |
| **bug-intake** | "bug intake" | Gmail | Green |
| **spec-health** | "spec health" | Sheets | Green/Yellow |
| **parallel-schedule** | "parallel schedule" | Calendar + Docs | Green/Yellow |
| **review-notify** | "review notify" | Chat + Gmail | Red |
| **sprint-board** | "sprint board" | Sheets | Green/Yellow |

### Safety Tiers

| Tier | Operations | Behavior |
|------|-----------|----------|
| Green | Read-only (list, get, search, stats, describe) | Auto-execute |
| Yellow | Create/modify (create, update, draft, append, import) | Confirm before execute |
| Red | Destructive/external (send, delete, share, archive) | Hard gate, explicit approval |
| Blocked | Permanent delete, ownership transfer | Never execute |

### Agent Sandbox
```bash
export GWX_ENABLE_COMMANDS="gmail.*,calendar.list,sheets.read,sheets.describe,github.repos,slack.channels"
```

## Resilience

- **Rate Limiter** — Per-service token bucket (Sheets 0.8 QPS, Gmail 4 QPS, Drive 8 QPS, Analytics 2 QPS, Search Console 2 QPS)
- **Retry Transport** — 429 exponential backoff with Retry-After header, 5xx fixed retry
- **Circuit Breaker** — Opens after 5 consecutive failures, auto-recovers after 30s

## Security

- **OS Keyring** — OAuth tokens and multi-provider API tokens stored in macOS Keychain / Linux Secret Service / Windows Credential Manager. Never written to disk files
- **Multi-Provider Token Isolation** — Google, GitHub, Slack, and Notion tokens stored under separate keyring entries. Per-account scoping prevents cross-contamination
- **CSRF Protection** — 128-bit crypto/rand state for OAuth flow
- **Drive Query Injection** — Folder ID validation prevents query injection
- **Sheets Formula Protection** — Auto-escapes `=`, `+`, `-`, `@` prefixed values
- **Attachment Size Limit** — 25MB check before reading into memory

## Authentication

```bash
# Interactive setup (browser OAuth — requests all Google service scopes)
gwx onboard
# Supports file path OR paste JSON directly (auto-detects '{' prefix)

# VPS/SSH setup (no browser on server):
gwx onboard
# Step 1: Paste credentials JSON (from Google Cloud Console)
# Step 2: Select services (Enter for all)
# Step 3: Choose (r)emote auth:
#   → Opens URL in your LOCAL browser
#   → After auth, browser shows "site can't be reached" — that's OK
#   → Copy the full URL from browser address bar, paste it back

# Non-interactive (CI — via environment variables)
export GWX_OAUTH_JSON='{"installed":{"client_id":"...","client_secret":"..."}}'
gwx onboard                    # auto-detects env var, uses remote auth

# CI/CD (direct access token, skip OAuth entirely)
export GWX_ACCESS_TOKEN="ya29.xxx"

# Check status
gwx auth status
```

## Global Flags

```
-f, --format     Output format: json, plain, table (default: json)
-a, --account    Account to use (default: "default")
    --fields     Filter output keys (e.g. --fields "count,messages")
    --dry-run    Validate without executing
    --no-input   Disable interactive prompts
```

## License

MIT
