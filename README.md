# gwx — Google Workspace CLI for Humans and Agents

A unified CLI for Google Workspace (Gmail, Calendar, Drive, Docs, Sheets, Tasks, Contacts, Chat) designed for both human use and LLM agent integration.

## Install

```bash
# npm (recommended)
npm install -g @redredchen01/gwx

# Go
go install github.com/redredchen01/gwx/cmd/gwx@latest

# From source
git clone https://github.com/redredchen01/gwx.git
cd gwx && make install
```

## Quick Start

```bash
# 1. Set up Google Cloud credentials
gwx onboard

# 2. Use it
gwx gmail list --limit 5
gwx calendar agenda
gwx drive list
gwx docs get DOC_ID
gwx sheets read SHEET_ID "A1:C10"
```

## Features

- **8 Google Workspace services** — Gmail, Calendar, Drive, Docs, Sheets, Tasks, Contacts, Chat
- **44 commands** — Full CRUD operations across all services
- **Agent-friendly** — JSON output, stable exit codes, command allowlist, schema introspection
- **Resilient** — Circuit breaker, retry with exponential backoff, per-service rate limiting
- **Secure** — OAuth tokens stored in OS keyring (never written to disk), CSRF-safe auth flow

## Commands

| Service | Commands |
|---------|----------|
| **Gmail** | `list` `get` `search` `labels` `send` `draft` `reply` |
| **Calendar** | `agenda` `list` `create` `update` `delete` `find-slot` |
| **Drive** | `list` `search` `upload` `download` `share` `mkdir` |
| **Docs** | `get` `create` `append` `export` |
| **Sheets** | `read` `append` `update` `create` |
| **Tasks** | `list` `lists` `create` `complete` `delete` |
| **Contacts** | `list` `search` `get` |
| **Chat** | `spaces` `send` `messages` |
| **Agent** | `exit-codes` `schema` |

## Agent Integration

### Claude Code Skill

```bash
# Install CLI + Claude Code skill
./install.sh
```

This installs:
- `gwx` binary to `$GOPATH/bin`
- Skill definition to `~/.claude/commands/google-workspace.md`
- Agent definitions to `~/.claude/agents/`
- Workflow recipes to `~/.claude/commands/`

### Safety Tiers

| Tier | Operations | Behavior |
|------|-----------|----------|
| 🟢 Green | Read-only (list, get, search) | Auto-execute |
| 🟡 Yellow | Create/modify (create, update, draft) | Confirm before execute |
| 🔴 Red | Destructive/external (send, delete, share) | Hard gate, explicit approval |
| ⛔ Blocked | Permanent delete, ownership transfer | Never execute |

### Command Allowlist (Sandbox)

```bash
# Restrict agent to read-only Gmail + Calendar
export GWX_ENABLE_COMMANDS="gmail.list,gmail.get,gmail.search,calendar.*"
```

### Schema Introspection

```bash
# Agent can discover all available commands
gwx schema
```

### Exit Codes

| Code | Name | Meaning |
|------|------|---------|
| 0 | success | Operation completed |
| 10 | auth_required | Run `gwx onboard` |
| 11 | auth_expired | Run `gwx auth login` |
| 12 | permission_denied | Scope or allowlist issue |
| 20 | not_found | Resource doesn't exist |
| 30 | rate_limited | Wait and retry |
| 31 | circuit_open | API unstable, wait 30s |
| 40 | invalid_input | Fix parameters |

## Resilience

- **Rate Limiter** — Per-service token bucket (conservative: Sheets 0.8 QPS, Gmail 4 QPS)
- **Retry Transport** — 429 exponential backoff with Retry-After, 5xx fixed retry
- **Circuit Breaker** — Opens after 5 consecutive failures, auto-recovers after 30s

## Authentication

```bash
# Interactive setup (browser OAuth)
gwx onboard

# Headless (loopback redirect, paste URL)
gwx auth login --manual

# CI/CD (direct token)
export GWX_ACCESS_TOKEN="ya29.xxx"
gwx gmail list
```

Tokens are stored in the OS keyring (macOS Keychain / Linux Secret Service / Windows Credential Manager) and never written to disk files.

## License

MIT
