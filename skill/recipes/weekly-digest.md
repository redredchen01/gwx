---
name: weekly-digest
description: "Generate a weekly digest: unread emails summary, upcoming events, pending tasks."
services: [gmail, calendar, tasks]
safety_tier: green
---

# Weekly Digest Workflow

## Trigger
User says: "週報整理", "weekly digest", "這週摘要", "幫我整理一下"

## Steps

### Step 1: Unread email summary
```bash
gwx gmail list --unread --limit 20 --json
```
Group by sender, count per sender, list subjects.

### Step 2: This week's events
```bash
gwx calendar agenda --days 7 --json
```
List events chronologically.

### Step 3: Pending tasks
```bash
gwx tasks list --json
```
List incomplete tasks with due dates.

### Step 4: Compile digest
Present:

```
# Weekly Digest — {date range}

## 📬 Unread Emails ({count})
- {sender1}: {count} messages — {subjects}
- {sender2}: {count} messages — {subjects}

## 📅 Upcoming Events ({count})
- {day}: {time} {title} ({attendee count} attendees)
- ...

## ✅ Pending Tasks ({count})
- [ ] {task1} (due: {date})
- [ ] {task2}
- ...
```

## Notes
- All operations are 🟢 read-only
- Suitable for running on schedule or on-demand
