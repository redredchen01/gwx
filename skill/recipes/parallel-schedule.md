---
name: parallel-schedule
description: "Auto-schedule code review meetings for parallel-develop worktrees using gwx calendar find-slot."
services: [calendar, docs]
safety_tier: green/yellow
combo: [gwx-calendar, sop-parallel-develop]
---

# Parallel Schedule Workflow

## Purpose

When running `/parallel-develop` with multiple worktrees, each feature needs a code review meeting after S4. This workflow automatically finds free time slots for all reviewers and creates calendar events with pre-filled context from each feature's brief_spec.

## Trigger
User says: "parallel schedule", "排 review 會議", "schedule reviews", "幫我排並行開發的 review", "並行排程"

## Prerequisites
- Active `/parallel-develop` session with 2+ worktrees
- gwx authenticated with Calendar access
- Reviewer email addresses known

## Steps

### Step 1: Scan active worktrees (local)

```bash
git worktree list --porcelain
```

For each worktree, read its sdd_context:
```
dev/specs/{feature}/sdd_context.json → extract:
  - feature name
  - current_stage
  - brief_spec summary (§1 one-liner)
```

Filter for worktrees at S4 completed or S5 pending.

### Step 2: Collect reviewer info

Ask user (or read from project config):
- Reviewer emails: `reviewer1@co.com, reviewer2@co.com`
- Review duration: default 30 minutes per feature
- Preferred time window: default "next 3 business days"

### Step 3: Find free slots (🟢)

For each feature needing review:
```bash
gwx calendar find-slot --attendees "{reviewer_emails}" --duration 30m --days 3 --json
```

Propose a schedule:
```
Proposed Review Schedule:

1. feature/invoice — Mon 10:00-10:30 (all reviewers free)
2. feature/auth-refactor — Mon 14:00-14:30
3. feature/dashboard — Tue 10:00-10:30

Accept? (yes / adjust)
```

### Step 4: Create calendar events (🟡 confirm)

For each accepted slot:
```bash
gwx calendar create \
  --title "Code Review: {feature_name}" \
  --start "{start_datetime}" \
  --end "{end_datetime}" \
  --attendees "{reviewer_emails}" \
  --json
```

### Step 5: Attach context to each event (🟡)

Create a briefing doc per review:
```bash
gwx docs create --title "Review Brief: {feature_name}" --body "{brief_spec_summary}\n\nBranch: {branch}\nScope: {file_list}\nKey decisions: {from dev_spec}" --json
```

Link doc in calendar event description.

### Step 6: Summary

```
Review Schedule Created:

| Feature | Date | Time | Reviewers | Brief Doc |
|---------|------|------|-----------|-----------|
| invoice | Mon 3/19 | 10:00-10:30 | alice, bob | [link] |
| auth-refactor | Mon 3/19 | 14:00-14:30 | alice, bob | [link] |
| dashboard | Tue 3/20 | 10:00-10:30 | alice, bob | [link] |
```

## Integration with `/parallel-develop`

The parallel-develop workflow can auto-trigger this when all worktrees reach S4:
1. Detect all worktrees at S4 completed
2. Prompt: "All features ready for review. Run `/parallel-schedule` to book review meetings?"
3. If yes, chain into this workflow

## Notes
- Step 3 is 🟢 read-only (find-slot)
- Steps 4-5 are 🟡 (create events/docs — confirm before execute)
- If reviewers have no common free slots, suggest extending the search window
- Works without docs creation — calendar events alone are sufficient
- If only one worktree, simplifies to a single find-slot + create
