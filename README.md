# gwx — Google Workspace CLI for Humans and Agents

A unified CLI + MCP server for Google Workspace — Gmail, Calendar, Drive, Docs, Sheets, Tasks, Contacts, Chat. Designed for both human use and LLM agent integration.

**66 CLI commands · 39 MCP tools · 8 Google services**

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

## Commands (66)

| Service | Commands |
|---------|----------|
| **Gmail** (9) | `list` `get` `search` `labels` `send` `draft` `reply` `digest` `archive` |
| **Calendar** (6) | `agenda` `list` `create` `update` `delete` `find-slot` |
| **Drive** (6) | `list` `search` `upload` `download` `share` `mkdir` |
| **Docs** (8) | `get` `create` `append` `search` `replace` `template` `from-sheet` `export` |
| **Sheets** (15) | `read` `info` `describe` `stats` `search` `filter` `diff` `append` `smart-append` `update` `clear` `copy-tab` `export` `import` `create` |
| **Tasks** (5) | `list` `lists` `create` `complete` `delete` |
| **Contacts** (3) | `list` `search` `get` |
| **Chat** (3) | `spaces` `send` `messages` |
| **Cross-service** (2) | `find` (unified search) · `context` (gather context) |
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
```

### Docs Template Engine
```bash
# Create documents from templates with {{var}} replacement
gwx docs template TEMPLATE_DOC_ID -v '{"name":"Alice","date":"2026-03-17"}'
```

### Cross-Service Operations
```bash
# Search across Gmail + Drive simultaneously
gwx find "keyword"

# Gather all context for a topic (Gmail + Drive + Calendar)
gwx context "project-name" --days 7
```

## MCP Server (39 Tools)

Native Claude integration — no Bash needed:

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

All 66 CLI commands are available as MCP tools. Claude can directly call `sheets_describe`, `gmail_digest`, `context_gather`, etc.

## Claude Code Skill

```bash
# Install CLI + Claude Code skill + workflow recipes
./install.sh
```

Installs skill definitions to `~/.claude/commands/` and agent definitions to `~/.claude/agents/` for automatic trigger on Google Workspace keywords.

### Combo Skills — gwx × Claude Code Workflows

Three power combos that chain gwx with Claude Code's SOP pipeline:

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

> All context-gathering is 🟢 read-only. Falls back to standard S0 if gwx is not connected.

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
| Result | ✅ / ❌ / ⏭️ |
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

### Workflow Recipes

| Recipe | Trigger | Services | Tier |
|--------|---------|----------|------|
| meeting-prep | "meeting prep" | Calendar + Gmail + Drive | 🟢 |
| weekly-digest | "weekly digest" | Gmail + Calendar + Tasks | 🟢 |
| standup-report | "standup" | Gmail + Calendar + Tasks | 🟢 |
| email-from-doc | "email from doc" | Docs + Gmail | 🟡/🔴 |
| sheet-to-email | "mail merge" | Sheets + Gmail | 🔴 |
| **context-boost** | "context boost" | Gmail + Drive + Calendar | 🟢 |
| **test-matrix** | "test matrix" | Sheets | 🟢/🟡 |
| **standup** | "standup" | Gmail + Calendar + Tasks + Git + Chat | 🟢/🔴 |

### Safety Tiers

| Tier | Operations | Behavior |
|------|-----------|----------|
| 🟢 Green | Read-only (list, get, search, stats, describe) | Auto-execute |
| 🟡 Yellow | Create/modify (create, update, draft, append, import) | Confirm before execute |
| 🔴 Red | Destructive/external (send, delete, share, archive) | Hard gate, explicit approval |
| ⛔ Blocked | Permanent delete, ownership transfer | Never execute |

### Agent Sandbox
```bash
export GWX_ENABLE_COMMANDS="gmail.*,calendar.list,sheets.read,sheets.describe"
```

## Resilience

- **Rate Limiter** — Per-service token bucket (Sheets 0.8 QPS, Gmail 4 QPS, Drive 8 QPS)
- **Retry Transport** — 429 exponential backoff with Retry-After header, 5xx fixed retry
- **Circuit Breaker** — Opens after 5 consecutive failures, auto-recovers after 30s

## Security

- **OS Keyring** — OAuth tokens stored in macOS Keychain / Linux Secret Service / Windows Credential Manager. Never written to disk files
- **CSRF Protection** — 128-bit crypto/rand state for OAuth flow
- **Drive Query Injection** — Folder ID validation prevents query injection
- **Sheets Formula Protection** — Auto-escapes `=`, `+`, `-`, `@` prefixed values
- **Attachment Size Limit** — 25MB check before reading into memory

## Authentication

```bash
# Interactive setup (browser OAuth — requests all 8 service scopes)
gwx onboard

# Headless (loopback redirect on random port)
gwx auth login --manual

# CI/CD (direct access token)
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
