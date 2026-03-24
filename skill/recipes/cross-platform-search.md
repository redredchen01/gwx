---
name: cross-platform-search
description: "Search across Gmail, Slack, and Notion simultaneously. Aggregates results from all three platforms in parallel."
services: [gmail, slack, notion]
safety_tier: green
combo: [gwx-multi, cross-platform]
---

# Cross-Platform Search Workflow

## Purpose

Search for a topic across Gmail, Slack, and Notion in parallel. Returns unified results from all three platforms, making it easy to find everything related to a topic regardless of where the conversation happened.

## Trigger
User says: "search everywhere", "cross search", "全平台搜尋", "search all platforms", "find across everything"

## Steps

### Step 1: Execute parallel searches (all read-only)

Run simultaneously:

**1a. Gmail search:**
```bash
gwx gmail search "{query}" --limit 5 -f json
```

**1b. Slack search:**
```bash
gwx slack search "{query}" --limit 5 -f json
```

**1c. Notion search:**
```bash
gwx notion search "{query}" --limit 5 -f json
```

### Step 2: Aggregate results

Combine into a unified view:

```json
{
  "query": "{query}",
  "gmail": {
    "count": N,
    "messages": [
      {"subject": "...", "from": "...", "date": "...", "snippet": "..."}
    ]
  },
  "slack": {
    "count": N,
    "messages": [
      {"channel": "...", "user": "...", "text": "...", "ts": "..."}
    ]
  },
  "notion": {
    "count": N,
    "pages": [
      {"title": "...", "type": "page|database", "last_edited": "..."}
    ]
  },
  "total_results": M
}
```

### Step 3: Display results

Format by platform with clear sections:

```
Cross-Platform Search: "{query}" — {total} results

📧 Gmail ({N} results)
  • {subject} — from {sender}, {date}
  • {subject} — from {sender}, {date}

💬 Slack ({N} results)
  • #{channel}: {user} — "{text snippet}" ({date})
  • #{channel}: {user} — "{text snippet}" ({date})

📝 Notion ({N} results)
  • {page title} (last edited {date})
  • {page title} (last edited {date})
```

## Notes
- All three searches are read-only operations (green safety tier)
- Searches run in parallel for speed
- If any platform is not authenticated, skip it and note in output
- Results are limited per platform (default 5 each) to keep output manageable
- The user can adjust limits per platform if needed
- No data is written to any platform
