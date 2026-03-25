# gwx Skills

19 curated YAML skills. Each composes 2+ MCP tools into a reusable workflow.

**Principle**: Single-tool wrappers belong in CLI, not skills. A skill must combine multiple tools.

## Google Workspace (9)

| Skill | Steps | What it does |
|-------|-------|-------------|
| `google-morning-brief` | 3 | Unread emails + today's calendar + pending tasks |
| `google-invoice-log` | 2 | Search Gmail for invoices → append to Sheets |
| `google-meeting-notes` | 2 | Today's meetings → create Google Doc |
| `google-doc-from-sheet` | 2 | Read spreadsheet → generate Google Doc |
| `google-contact-export` | 2 | List contacts → append to Sheets |
| `google-bq-to-sheet` | 2 | BigQuery SQL → save to Sheets |
| `google-seo-daily` | 3 | Search Console + GA4 → save to Sheets |
| `google-task-report` | 2 | List task lists + fetch tasks |
| `cross-daily-report` | 2 | Chains morning-brief + seo-daily (sub-skill) |

## GitHub Integration (3)

| Skill | Steps | What it does |
|-------|-------|-------------|
| `github-issue-to-sheet` | 3 | Issues → transform → Google Sheets |
| `github-pr-to-slack` | 3 | Open PRs → transform → Slack notification |
| `github-ci-alert` | 3 | CI runs → conditional Slack + Gmail alert |

## Cross-Provider (7)

| Skill | Steps | What it does |
|-------|-------|-------------|
| `cross-client-360` | 3 | Gmail + Drive + Contacts parallel search |
| `cross-github-standup` | 3 | GitHub PRs + Gmail + Calendar parallel standup |
| `cross-full-context` | 5 | Gmail + Drive + Slack + Notion + GitHub parallel search |
| `cross-weekly-sync` | 4 | GA4 + GitHub + Gmail + Calendar weekly digest |
| `cross-onboard-checklist` | 5 | Drive + Docs + Calendar + Slack + Gmail onboarding |
| `cross-form-to-slack` | 3 | Forms responses → transform → Slack |
| `cross-notion-to-sheet` | 2 | Notion DB → Google Sheets sync |

## Quick Start

```bash
gwx skill list                         # List all
gwx skill run google-morning-brief     # Run
gwx skill test google-morning-brief    # Test with mock data
gwx skill create my-skill              # Scaffold
```
