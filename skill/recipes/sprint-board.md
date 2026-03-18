---
name: sprint-board
description: "Use a Google Sheet as a lightweight Kanban/Sprint board — S0 creates tickets, S4 updates status, sheets stats provides burn-down, copy-tab archives sprints."
services: [sheets]
safety_tier: green/yellow
combo: [gwx-sheets, sop-pipeline]
---

# Sprint Board Workflow

## Purpose

Replace heavyweight project management tools (Jira, Linear) with a Google Sheet that stays in sync with the SOP pipeline. Every S0 creates a ticket, every S4 updates its status, and `gwx sheets stats` gives instant burn-down metrics. Zero cost, zero context-switching.

## Trigger
User says: "sprint board", "看板", "kanban", "sprint sheet", "專案追蹤", "建看板", "project board"

## Steps

### Step 1: Create or locate Sprint Board (🟡)

**First time:**
```bash
gwx sheets create --title "Sprint Board — {sprint_name}" --json
```

**Subsequent:**
```bash
gwx drive search "name contains 'Sprint Board'" --limit 3 --json
```

### Step 2: Initialize structure (🟡)

```bash
gwx sheets update {SHEET_ID} "Backlog!A1:J1" --values '[["Ticket", "Feature", "Type", "Priority", "Status", "Assignee", "Created", "Updated", "Branch", "Notes"]]' --json
```

**Column definitions:**
| Column | Values | Source |
|--------|--------|--------|
| Ticket | SOP-001, SOP-002 | Auto-increment |
| Feature | Feature name | S0 brief_spec §1 |
| Type | feature / bugfix / refactor / investigation | S0 work_type |
| Priority | P0 / P1 / P2 | User-set |
| Status | backlog / in-progress / review / testing / done / blocked | SOP stage mapping |
| Assignee | Name | User-set |
| Created | Date | S0 timestamp |
| Updated | Date | Last stage transition |
| Branch | Branch name | Git branch |
| Notes | Free text | Blockers, decisions |

**Stage → Status mapping:**
| SOP Stage | Board Status |
|-----------|-------------|
| S0 confirmed | backlog |
| S1-S3 | in-progress |
| S4 | in-progress |
| S5 | review |
| S6 | testing |
| S7 completed | done |
| Any repair loop exceeded | blocked |

### Step 3: S0 — Create ticket (🟡)

When S0 Gate is confirmed:
```bash
gwx sheets smart-append {SHEET_ID} "Backlog!A:J" --values '[
  ["SOP-{N}", "{feature_name}", "{work_type}", "{priority}", "backlog", "{assignee}", "{date}", "{date}", "{branch}", ""]
]' --json
```

### Step 4: Stage transitions — Update status (🟡)

When SOP advances stages:
```bash
gwx sheets update {SHEET_ID} "Backlog!E{row}:F{row}" --values '[["{new_status}", "{date}"]]' --json
```

Examples:
- S4 starts → `in-progress`
- S5 starts → `review`
- S5 fix_required → `in-progress` + Notes: "Review round {N}"
- S6 starts → `testing`
- S6 repair loop exceeded → `blocked` + Notes: "S6 repair loop 3x"
- S7 done → `done`

### Step 5: Burn-down and metrics (🟢)

```bash
gwx sheets stats {SHEET_ID} --json
```

Output:
```
Sprint Progress:
- Status → done: 7, in-progress: 3, review: 1, testing: 2, backlog: 4, blocked: 1
- Type → feature: 10, bugfix: 5, refactor: 3
- Priority → P0: 2, P1: 8, P2: 8

Velocity: 7 done / 18 total = 38.9%
Blocked: SOP-012 (S6 repair loop)
```

### Step 6: Sprint archive (🟡)

At sprint end:
```bash
# Copy current sprint as archive
gwx sheets copy-tab {SHEET_ID} --from "Backlog" --to "Sprint {N} Archive" --json

# Clear done items from Backlog, keep in-progress
gwx sheets update {SHEET_ID} "Backlog!..." --values '...' --json
```

### Step 7: Cross-sprint diff (🟢)

```bash
gwx sheets diff {SHEET_ID} --tab1 "Sprint 1 Archive" --tab2 "Sprint 2 Archive" --json
```

Shows velocity changes, recurring blocked items, type distribution shifts.

## Multi-user Support

Share the Sheet with team members:
```bash
# 🔴 Hard gate
gwx drive share {SHEET_ID} --email "team@co.com" --role writer --json
```

Everyone sees real-time updates. No repo access needed.

## Integration with SOP Pipeline

The Sprint Board is a **passive receiver** — SOP stages push updates to it:

```
S0 Gate ✅ → smart-append new ticket
S4 start → update status = in-progress
S5 start → update status = review
S5 fix_required → update status = in-progress
S6 start → update status = testing
S6 blocked → update status = blocked
S7 done → update status = done
```

## Notes
- Read operations (stats, diff) are 🟢
- Write operations (create ticket, update status) are 🟡
- Share is 🔴
- `smart-append` validates column structure
- Sheet ID should be persisted in project-level config
- If gwx not connected, SOP runs normally — board is enhancement, not dependency
- `copy-tab` preserves header + structure for sprint archives
