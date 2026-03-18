---
name: review-notify
description: "Push S5 code review results to Google Chat or email immediately after review completes."
services: [chat, gmail]
safety_tier: red
combo: [gwx-chat, sop-s5]
---

# Review Notify Workflow

## Purpose

After S5 code review (R1→R2→R3) completes, immediately push a structured summary to the team's Google Chat space or via email. No one has to wait for the developer to report — the review result arrives in real-time.

## Trigger
User says: "review notify", "推送 review 結果", "通知 review", "review 結果推到 chat", "notify team"

## Prerequisites
- S5 code review completed (sdd_context has `stages.s5.output`)
- gwx authenticated with Chat or Gmail access
- Target channel/email configured

## Steps

### Step 1: Read S5 review output (local)

```bash
# Read from sdd_context
cat dev/specs/{feature}/sdd_context.json
```

Extract from `stages.s5.output`:
- `recommendation`: pass / conditional_pass / fix_required
- `findings`: array of { severity, description, file, line }
- `p0_count`, `p1_count`, `p2_count`
- `review_duration`

### Step 2: Compile notification

```
📋 Code Review Complete: {feature_name}

Result: {✅ PASS | ⚠️ CONDITIONAL PASS | ❌ FIX REQUIRED}
Branch: {branch}
Reviewer: Claude R1→R2→R3

Findings: {p0} P0 · {p1} P1 · {p2} P2

{if P0 > 0:}
🔴 P0 Issues (blocking):
  - {description} ({file}:{line})
{endif}

{if P1 > 0:}
🟡 P1 Issues:
  - {description} ({file}:{line})
{endif}

Next: {S6 testing | Fix required → back to S4}
```

### Step 3: Deliver (🔴 hard gate)

**Option A — Google Chat:**
```bash
gwx chat send {SPACE_NAME} --text "{notification}" --json
```

**Option B — Email:**
```bash
gwx gmail send --to "{team_email}" --subject "Review: {feature_name} — {PASS|FAIL}" --body "{notification}" --json
```

**Option C — Draft (🟡):**
```bash
gwx gmail draft --to "{team_email}" --subject "Review: {feature_name} — {PASS|FAIL}" --body "{notification}" --json
```

**MUST show full notification content and get explicit confirmation before sending.**

### Step 4: Record delivery

Update sdd_context:
```json
{
  "stages": {
    "s5": {
      "output": {
        "notification_sent": true,
        "notification_channel": "{chat|email}",
        "notification_timestamp": "{ISO 8601}"
      }
    }
  }
}
```

## Auto-trigger from S5

When S5 completes and a notification target is configured:
1. S5 Gate resolves (pass / conditional_pass / fix_required)
2. Prompt: "Review complete. Push to {channel}? (yes/no)"
3. If yes → execute Steps 2-4

## Configuration

Store default notification target in project config:
```json
{
  "review_notify": {
    "channel": "chat:spaces/AAAA",
    "notify_on": ["fix_required", "pass"],
    "skip_on": ["conditional_pass"]
  }
}
```

## Notes
- Step 1 is local (no network)
- Step 3 is 🔴 (sending externally) — always requires explicit confirmation
- Option C (draft) is 🟡 — lower friction alternative
- Notification is concise — full details remain in sdd_context
- If review had no findings (P0=P1=0), notification is a simple "✅ PASS"
