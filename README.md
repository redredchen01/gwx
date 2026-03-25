# gwx — 17 Services, One CLI

把散落在各平台的資料，用一個命令列統一操作。人能用，AI 也能用。

```
Google:   Gmail · Calendar · Drive · Docs · Sheets · Tasks
          Contacts · Chat · Analytics · Search Console · Slides
          Forms · BigQuery
外部:     GitHub · Slack · Notion · Obsidian
```

**150+ CLI commands · 133 MCP tools · 22 YAML skills · 17 services**

### 三種用法

```bash
# 1. 人打指令
gwx gmail list --limit 5
gwx calendar agenda
gwx github pulls owner/repo

# 2. AI 透過 MCP 直接呼叫（133 個工具，Claude/Codex 自動可用）
gwx mcp-server

# 3. YAML Skill 自動化（不寫程式，串多個平台一鍵跑）
gwx skill run google-morning-brief     # 收信 + 行程 + 待辦
gwx skill run cross-full-context -p keyword=invoice  # 5 平台並行搜
```

## Install

```bash
# One-liner (macOS/Linux — pre-built binary to /usr/local/bin)
curl -fsSL https://raw.githubusercontent.com/redredchen01/gwx/main/install-bin.sh | sudo bash

# npm (auto-downloads pre-built binary)
npm install -g gwx-cli

# Go
go install github.com/redredchen01/gwx/cmd/gwx@latest

# Homebrew
brew install redredchen01/tap/gwx

# From source
git clone https://github.com/redredchen01/gwx.git
cd gwx && make install
```

## Quick Start

```bash
# 1. Set up credentials (interactive wizard)
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

## Commands

| Service | Count | Commands |
|---------|-------|----------|
| **Gmail** | 11 | `list` `get` `search` `labels` `send` `draft` `reply` `digest` `archive` `label` `forward` |
| **Calendar** | 6 | `agenda` `list` `create` `update` `delete` `find-slot` |
| **Drive** | 6 | `list` `search` `upload` `download` `share` `mkdir` |
| **Docs** | 8 | `get` `create` `append` `search` `replace` `template` `from-sheet` `export` |
| **Sheets** | 15 | `read` `info` `describe` `stats` `search` `filter` `diff` `append` `smart-append` `update` `clear` `copy-tab` `export` `import` `create` |
| **Tasks** | 5 | `list` `lists` `create` `complete` `delete` |
| **Contacts** | 3 | `list` `search` `get` |
| **Chat** | 3 | `spaces` `send` `messages` |
| **Analytics** | 4 | `report` `realtime` `properties` `audiences` |
| **Search Console** | 5 | `query` `sites` `inspect` `sitemaps` `index-status` |
| **Slides** | 6 | `get` `list` `create` `duplicate` `export` `from-sheet` |
| **Forms** | 3 | `get` `responses` `response` |
| **BigQuery** | 4 | `query` `datasets` `tables` `describe` |
| **GitHub** | 10 | `login` `logout` `status` `repos` `issues` `pulls` `pull` `runs` `notify` `create issue` |
| **Slack** | 7 | `login` `status` `channels` `send` `messages` `search` `users` |
| **Notion** | 7 | `login` `status` `search` `page` `create` `databases` `query` |
| **Obsidian** | 10 | `setup` `list` `search` `read` `create` `append` `daily` `tags` `recent` `folders` |
| **Skill** | 8 | `list` `inspect` `validate` `run` `create` `test` `install` `remove` |
| **Config** | 3 | `set` `get` `list` |
| **Workflow** | 13 | `standup` `meeting-prep` + `workflow` subgroup: `weekly-digest` `context-boost` `bug-intake` `test-matrix` `spec-health` `sprint-board` `review-notify` `email-from-doc` `sheet-to-email` `parallel-schedule` |
| **Cross-service** | 2 | `find` (unified search) · `context` (gather context) |
| **Pipeline** | 1 | `pipe` (chain commands via JSON stdin/stdout) |
| **System** | 12 | `auth login/logout/status` `onboard` `agent exit-codes` `schema` `mcp-server` `version` `doctor` `completion bash/zsh/fish` |

## Highlights

### Smart Sheets

```bash
# Analyze column structure before writing
gwx sheets describe SHEET_ID
# → [0] Name (freetext, REQUIRED)  [2] Status (enum): Done / In Progress

# Validate + append (catches wrong enum values, missing required fields)
gwx sheets smart-append SHEET_ID "Sheet1!A:F" --values '[["Alice","Plan X","Done","","",""]]'

# Column statistics + tab comparison
gwx sheets stats SHEET_ID
gwx sheets diff SHEET_ID --from "Week1" --to "Week2"

# Export / Import
gwx sheets export SHEET_ID "A:D" --export-format csv -o report.csv
gwx sheets import SHEET_ID "A1" -i data.csv --import-format csv
```

### Gmail Intelligence

```bash
# Smart digest — groups by sender, categorizes CI/transactional/personal
gwx gmail digest --limit 30

# Batch operations
gwx gmail archive "subject:Run failed" --limit 50
gwx gmail label "from:github" --add CI --remove INBOX --limit 100
gwx gmail forward MESSAGE_ID --to colleague@company.com
```

### Cross-Platform Operations

```bash
# Search across Gmail + Drive simultaneously
gwx find "keyword"

# Gather all context for a topic (Gmail + Drive + Calendar)
gwx context "project-name" --days 7

# Chain commands — each stage passes JSON to the next
gwx pipe "gmail search 'invoice' | sheets append SHEET_ID A:C"
```

### Built-in Workflows

```bash
# Daily standup — aggregate Git + Gmail + Calendar + Tasks
gwx standup

# Meeting prep — attendees, recent emails, related docs
gwx meeting-prep "Weekly Sync"

# More via gwx workflow: weekly-digest, context-boost, bug-intake,
# test-matrix, spec-health, sprint-board, review-notify, email-from-doc,
# sheet-to-email, parallel-schedule
```

> All workflows default to **read-only** (JSON output). Add `--execute` for write actions.

### Docs Template Engine

```bash
# Create documents from templates with {{var}} replacement
gwx docs template TEMPLATE_DOC_ID -v '{"name":"Alice","date":"2026-03-17"}'
```

### Slides from Data

```bash
# Generate slides from Sheet data + template (replaces {{placeholders}})
gwx slides from-sheet --template TEMPLATE_ID --sheet-id SHEET_ID --range "A:D"
```

## Skill DSL — YAML-Defined Workflows

Define multi-step workflows in YAML — no Go code, no recompilation. 22 built-in skills covering Google, GitHub, Slack, Notion, and Obsidian cross-service workflows:

```yaml
# skills/google-morning-brief.yaml
name: google-morning-brief
description: "Morning standup brief — unread emails, today's calendar, and pending tasks"
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
gwx skill list                           # List installed skills
gwx skill run google-morning-brief       # Run a skill
gwx skill validate skills/my-skill.yaml  # Validate YAML
gwx skill create my-new-skill            # Scaffold a new skill
```

Skills support **parallel execution**, **each loops**, **transform pipes**, **conditional steps** (`if:`), and **skill composition** (`tool: skill:<name>`). Skills auto-register as MCP tools (`skill_google-morning-brief`). See [USAGE.md](USAGE.md) for the full DSL reference and complete skill list.

## MCP Server (133 Tools)

Native Claude integration — no Bash needed:

```bash
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

All CLI commands available as MCP tools: `gmail_list`, `sheets_describe`, `analytics_report`, `github_repos`, `slack_send`, `notion_search`, `bigquery_query`, `forms_get`, `skill_run`, etc. See [USAGE.md](USAGE.md) for the full tool reference.

## Multi-Provider Auth

```bash
# Google (OAuth flow)
gwx onboard

# GitHub
gwx github login --token ghp_xxx

# Slack
gwx slack login xoxb-xxx

# Notion
gwx notion login ntn_xxx

# Check status
gwx auth status           # Google
gwx github status         # GitHub
gwx slack status          # Slack
gwx notion status         # Notion
```

Tokens stored in OS keyring (macOS Keychain / Linux Secret Service / Windows Credential Manager). Never written to disk.

## Shell Completion

```bash
eval "$(gwx completion bash)"     # Bash
eval "$(gwx completion zsh)"      # Zsh (add to ~/.zshrc)
gwx completion fish | source      # Fish
```

## Health Check

```bash
gwx doctor
```

Diagnoses all providers, config, auth status, and loaded skills in one command.

## Security

- **OS Keyring** — OAuth + API tokens stored in system keyring, never on disk
- **Multi-Provider Isolation** — Google, GitHub, Slack, Notion tokens stored separately
- **CSRF Protection** — 128-bit crypto/rand state for OAuth flow
- **Input Safety** — Drive query injection prevention, Sheets formula escaping, 25MB attachment limit
- **Rate Limiter** — Per-service token bucket (Sheets 0.8 QPS, Gmail 4 QPS, Drive 8 QPS)
- **Circuit Breaker** — Opens after 5 consecutive failures, auto-recovers after 30s
- **Retry Transport** — 429 exponential backoff with Retry-After, 5xx fixed retry

## Safety Tiers

| Tier | Operations | Behavior |
|------|-----------|----------|
| Green | Read-only (list, get, search, stats, describe) | Auto-execute |
| Yellow | Create/modify (create, update, draft, append, import) | Confirm before execute |
| Red | Destructive/external (send, delete, share, archive) | Hard gate, explicit approval |
| Blocked | Permanent delete, ownership transfer | Never execute |

## Global Flags

```
-f, --format     Output format: json, plain, table (default: json)
-a, --account    Account to use (default: "default")
    --fields     Filter output keys (e.g. --fields "count,messages")
    --dry-run    Validate without executing
    --no-input   Disable interactive prompts
```

## Claude Code Skill

```bash
./install.sh             # Install CLI + skill definitions + agent definitions
```

Installs skill definitions to `~/.claude/commands/` and agent definitions to `~/.claude/agents/` for automatic trigger on Google Workspace keywords. See [USAGE.md](USAGE.md) for combo skills and recipes.

## License

MIT
