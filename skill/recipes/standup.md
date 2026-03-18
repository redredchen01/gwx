---
name: standup
description: "AI-powered daily standup: merge git activity, SOP progress, Google Workspace data (emails, calendar, tasks) into a structured standup report. Optionally push to Google Chat."
services: [gmail, calendar, tasks, chat]
safety_tier: green/red
combo: [gwx-multi, sop-context, git]
---

# AI Standup Workflow

## Purpose

Generate a comprehensive daily standup report by combining:
1. **Git activity** — yesterday's commits, current branch status
2. **SOP progress** — active SDD contexts, current stage, blockers
3. **Google Workspace** — sent emails, attended meetings, completed tasks
4. **Today's plan** — upcoming meetings, pending tasks, next SOP stage

This goes beyond the basic `standup-report.md` recipe by integrating development context.

## Trigger
User says: "standup", "站會", "daily standup", "今天站會", "standup report", "站會報告"

## Steps

### Step 1: Git activity (local, instant)
```bash
git log --oneline --since="yesterday" --author="$(git config user.name)" 2>/dev/null
git branch --show-current
git status --porcelain | head -5
```

Compile:
- Commits made yesterday (count + summaries)
- Current branch
- Uncommitted changes (if any)

### Step 2: SOP progress (local, instant)

Scan for active SDD contexts:
```bash
find dev/specs -name "sdd_context.json" -newer "$(date -v-7d +%Y%m%d)" 2>/dev/null
```

For each active context, extract:
- Feature name
- Current stage (s0-s7)
- Status (in_progress / completed)
- Blockers (if any repair loops exceeded)

### Step 3: Google Workspace data (🟢 all read-only)

Run in parallel:

**3a. Yesterday's sent emails:**
```bash
gwx gmail search "in:sent after:{yesterday} before:{today}" --limit 10 --json
```

**3b. Yesterday's meetings:**
```bash
gwx calendar list --from {yesterday} --to {today} --json
```

**3c. Today's meetings:**
```bash
gwx calendar agenda --days 1 --json
```

**3d. Recently completed tasks:**
```bash
gwx tasks list --show-completed --json
```

**3e. Pending tasks:**
```bash
gwx tasks list --json
```

### Step 4: Compile standup report

```markdown
# 🗓️ Daily Standup — {today's date}

## ✅ Done (Yesterday)

### Development
- {N} commits on `{branch}`: {commit summaries}
- SOP progress: {feature_name} moved from S{x} → S{y}

### Communication
- Sent {N} emails: {key subjects}
- Attended {N} meetings: {meeting titles}
- Completed {N} tasks: {task titles}

## 📋 Plan (Today)

### Development
- {feature_name}: Continue S{current_stage} → target S{next_stage}
- {uncommitted changes status}

### Meetings
- {time}: {meeting title} ({attendee count} attendees)

### Tasks
- [ ] {pending task 1} (due: {date})
- [ ] {pending task 2}

## 🚧 Blockers
- {blocker from SOP repair loops, if any}
- {blocker from failed tests, if any}
- (none) ← if no blockers
```

### Step 5: Deliver (user choice)

**Option A — Display only (default, 🟢):**
Print the standup in the terminal.

**Option B — Push to Google Chat (🔴 hard gate):**
```bash
gwx chat send {SPACE_NAME} --text "{standup_report}" --json
```
Requires explicit user confirmation with space name.

**Option C — Draft as email (🟡):**
```bash
gwx gmail draft --to "{team_email}" --subject "Standup — {date}" --body "{standup_report}" --json
```

## Notes
- Steps 1-2 are local operations, no network needed
- Step 3 is all 🟢 read-only
- Step 5 Option A is default (no confirmation needed)
- Step 5 Option B/C require confirmation per safety tiers
- If gwx is not authenticated, Steps 1-2 still produce a useful (git-only) standup
- Date handling: {yesterday} = yesterday in YYYY/MM/DD, {today} = today
- Gracefully handle missing data: if no commits, say "No commits yesterday"
