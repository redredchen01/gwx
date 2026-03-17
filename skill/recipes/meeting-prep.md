---
name: meeting-prep
description: "Prepare a briefing for upcoming meetings: fetch events, gather attendee context from recent emails, find related Drive docs."
services: [calendar, gmail, drive]
safety_tier: green
---

# Meeting Prep Workflow

## Trigger
User says: "幫我準備明天的會議", "meeting prep", "會議準備"

## Steps

### Step 1: Fetch upcoming events
```bash
gwx calendar agenda --days 1 --json
```
Parse the events. For each event with attendees, continue to Step 2.

### Step 2: Gather email context per meeting
For each attendee email in the event:
```bash
gwx gmail search "from:{attendee_email}" --limit 3 --json
```
Collect recent email subjects and snippets for context.

### Step 3: Find related documents
Use the meeting title to search Drive:
```bash
gwx drive search "name contains '{meeting_title_keyword}'" --limit 5 --json
```

### Step 4: Compile briefing
Present a structured briefing per meeting:

```
## Meeting: {title}
- Time: {start} - {end}
- Location: {location}
- Attendees: {list}

### Recent Context
- From {attendee1}: {recent email subjects}
- From {attendee2}: {recent email subjects}

### Related Documents
- {doc_name} (last modified {date})
```

## Notes
- All operations are 🟢 read-only — no confirmation needed
- Rate limiter handles the burst of API calls automatically
- Skip attendees that are the user's own email
