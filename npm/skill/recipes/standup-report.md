---
name: standup-report
description: "Generate a daily standup report from yesterday's activity: sent emails, attended meetings, completed tasks."
services: [gmail, calendar, tasks]
safety_tier: green
---

# Standup Report Workflow

## Trigger
User says: "standup", "daily report", "昨天做了什麼", "站立會議報告"

## Steps

### Step 1: Yesterday's sent emails
```bash
gwx gmail search "in:sent after:{yesterday} before:{today}" --limit 20 --json
```
List subjects of emails sent yesterday.

### Step 2: Yesterday's meetings
```bash
gwx calendar list --from {yesterday} --to {today} --json
```
List meetings attended.

### Step 3: Recently completed tasks
```bash
gwx tasks list --show-completed --json
```
Filter for tasks completed yesterday (check `completed` timestamp).

### Step 4: Compile standup

```
# Standup — {today's date}

## Done yesterday
- Sent {N} emails: {key subjects}
- Attended {N} meetings: {meeting titles}
- Completed {N} tasks: {task titles}

## Plan for today
(User fills in or agent suggests based on today's calendar)

## Blockers
(User fills in)
```

## Notes
- All 🟢 read-only operations
- Date placeholders: {yesterday} = yesterday's date in YYYY/MM/DD, {today} = today
