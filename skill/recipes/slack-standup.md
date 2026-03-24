---
name: slack-standup
description: "Cross-platform standup: pull Gmail digest + Calendar agenda, format as a concise standup report, and post to a Slack channel."
services: [gmail, calendar, slack]
safety_tier: red
combo: [gwx-multi, cross-platform]
---

# Slack Standup Workflow

## Purpose

Generate a daily standup report from Google Workspace data and post it to Slack. Combines:
1. **Gmail digest** — unread/recent email summary
2. **Calendar agenda** — today's meetings
3. **Slack delivery** — post the formatted report to a channel

## Trigger
User says: "slack standup", "post standup to slack", "站會發到 slack", "standup slack"

## Steps

### Step 1: Gather Gmail digest (read-only)
```bash
gwx gmail digest --limit 20 --unread -f json
```

Extract:
- Total unread count
- Top senders
- Key subjects requiring action

### Step 2: Gather Calendar agenda (read-only)
```bash
gwx calendar agenda --days 1 -f json
```

Extract:
- Meeting count
- Meeting titles + times
- Attendee lists

### Step 3: Format standup report

```
Daily Standup — {today's date}

📧 Email Summary
• {unread_count} unread emails
• Key threads: {top subjects}
• Action needed from: {senders}

📅 Today's Meetings
• {time}: {meeting title} ({attendee count} people)
• {time}: {meeting title}

✅ Plan
• Review and respond to {high_priority_count} urgent emails
• Attend {meeting_count} meetings
• {custom items if provided}
```

### Step 4: Post to Slack (requires confirmation)

```bash
gwx slack send -c {CHANNEL_ID} "{standup_report}"
```

**This step requires explicit user confirmation** with:
- Target channel ID or name
- Preview of the message before sending

## Notes
- Steps 1-2 are read-only Google Workspace operations
- Step 4 is a write operation that requires user confirmation
- If Slack is not authenticated, display the report in terminal instead
- Channel must be specified by the user (no default)
- The report format can be customized by the user
