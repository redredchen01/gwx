---
name: context-boost
description: "Inject Google Workspace context (emails, docs, calendar) into S0 requirement discussion for richer, faster requirement convergence."
services: [gmail, drive, calendar]
safety_tier: green
combo: [gwx-context, sop-s0]
---

# Context Boost Workflow

## Purpose

Before starting S0 requirement discussion, automatically gather all relevant context from Google Workspace — recent email threads, related Drive documents, upcoming meetings — and inject them as structured background for the `requirement-analyst` agent.

This turns S0 from "tell me what you want" into "here's everything I already know, let's confirm and refine."

## Trigger
User says: "context boost", "帶上下文開 SOP", "先查再做", "蒐集背景再開始", or any S0 trigger with a topic keyword + "查一下相關的"

## Steps

### Step 1: Extract topic keyword (🟢)
Parse the user's requirement description to extract 1-3 search keywords.

Example: "幫我做一個 invoice 自動寄送功能" → keywords: `invoice`, `寄送`

### Step 2: Cross-service context gather (🟢)
```bash
gwx context "{keyword}" --days 14 --json
```

This runs parallel searches across:
- **Gmail**: emails mentioning the topic (threads, senders, dates)
- **Drive**: documents related to the topic (PRDs, design docs, specs)
- **Calendar**: upcoming meetings related to the topic

### Step 3: Deep-dive on high-signal results (🟢)

For the top 3 most relevant emails:
```bash
gwx gmail get {message_id} --json
```

For the top 2 most relevant Drive docs:
```bash
gwx docs get {doc_id} --json
```

### Step 4: Compile context briefing

Structure the gathered context into a briefing block:

```markdown
## 🔍 Google Workspace Context Briefing

### 📬 Related Email Threads ({count})
| Date | From | Subject | Key Points |
|------|------|---------|------------|
| {date} | {sender} | {subject} | {snippet} |

### 📄 Related Documents ({count})
| Document | Last Modified | Summary |
|----------|--------------|---------|
| {title} | {date} | {first 100 chars} |

### 📅 Related Meetings ({count})
| Date | Title | Attendees |
|------|-------|-----------|
| {date} | {title} | {attendees} |

### 💡 Context Insights
- Stakeholders involved: {list of unique senders/attendees}
- Timeline signals: {any deadlines mentioned in emails}
- Existing decisions: {any conclusions from docs/emails}
```

### Step 5: Inject into S0

Pass the context briefing as supplementary input to S0:

```
The following Google Workspace context was gathered automatically.
Use it to:
1. Pre-fill known stakeholders and constraints
2. Reference specific email threads or docs in clarification questions
3. Identify potential conflicts with scheduled meetings
4. Skip questions whose answers are already in the context

--- BEGIN CONTEXT BRIEFING ---
{compiled briefing from Step 4}
--- END CONTEXT BRIEFING ---

User's original requirement: {original requirement text}
```

Then invoke S0 normally (Skill: s0-understand).

## Integration with S0

The `requirement-analyst` agent should:
- **Reference** context items: "In the email from Alice on 3/15, she mentioned X — is that still the requirement?"
- **Pre-fill** stakeholders from email senders and meeting attendees
- **Surface** contradictions: "The PRD says A, but the email thread suggests B — which is correct?"
- **Skip** already-answered questions: if the context clearly shows the answer, confirm rather than ask

## Notes
- All context-gathering operations are 🟢 read-only
- `gwx context` handles rate limiting internally via parallel fan-out
- If no results found for any service, skip that section silently
- Maximum context briefing length: 2000 chars (summarize if longer)
- This workflow is optional — S0 works fine without it, this just accelerates convergence
