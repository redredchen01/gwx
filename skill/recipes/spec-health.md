---
name: spec-health
description: "Track spec-audit results across features in a Google Sheet — visualize quality trends, P0/P1/P2 distribution, and audit convergence history."
services: [sheets]
safety_tier: green/yellow
combo: [gwx-sheets, sop-spec-audit]
---

# Spec Health Dashboard Workflow

## Purpose

Maintain a persistent Google Sheet that tracks spec-audit results across all features over time. Instead of looking at individual audit reports, get a bird's-eye view of code quality trends, recurring problem areas, and convergence velocity.

## Trigger
User says: "spec health", "品質儀表板", "spec 健康", "audit 追蹤", "quality dashboard", "品質趨勢"

## Steps

### Step 1: Create or locate dashboard Sheet (🟡)

**First time:**
```bash
gwx sheets create --title "Spec Health Dashboard" --json
```

**Subsequent:**
```bash
gwx drive search "name contains 'Spec Health Dashboard'" --limit 3 --json
```

### Step 2: Initialize structure (🟡)

**Tab 1: Audit Log** — one row per audit run
```bash
gwx sheets update {SHEET_ID} "Audit Log!A1:I1" --values '[["Date", "Feature", "Spec Mode", "Round", "P0", "P1", "P2", "Status", "Duration (min)"]]' --json
```

**Tab 2: Feature Summary** — one row per feature, latest state
```bash
gwx sheets update {SHEET_ID} "Feature Summary!A1:H1" --values '[["Feature", "Total Audits", "Last Audit", "Current P0", "Current P1", "Current P2", "Convergence Rounds", "Health"]]' --json
```

**Tab 3: Trend** — weekly aggregates
```bash
gwx sheets update {SHEET_ID} "Trend!A1:F1" --values '[["Week", "Features Audited", "Total P0", "Total P1", "Total P2", "Avg Convergence Rounds"]]' --json
```

### Step 3: Record audit results (🟡)

After each `/spec-audit` or `/audit-converge` run, append results:

```bash
gwx sheets smart-append {SHEET_ID} "Audit Log!A:I" --values '[
  ["{date}", "{feature}", "{spec_mode}", "{round}", {p0}, {p1}, {p2}, "{pass|fail}", {duration}]
]' --json
```

Update Feature Summary row:
```bash
gwx sheets update {SHEET_ID} "Feature Summary!{row}" --values '[["{feature}", {total_audits}, "{last_date}", {p0}, {p1}, {p2}, {rounds}, "{health_emoji}"]]' --json
```

Health emoji logic:
- P0=0, P1=0, P2≤2 → ✅ Healthy
- P0=0, P1≤2 → 🟡 Acceptable
- P0>0 → 🔴 Critical

### Step 4: Generate insights (🟢)

```bash
gwx sheets stats {SHEET_ID} --tab "Audit Log" --json
```

Produce insights:
```
Spec Health Report:
- Total features tracked: {N}
- Features at ✅: {N} | 🟡: {N} | 🔴: {N}
- Most audited feature: {name} ({N} rounds)
- Highest P0 rate: {feature} — investigate root cause
- Average convergence: {N} rounds
- Trend: P0 count {↑|↓|→} vs last week
```

### Step 5: Diff between periods (🟢)

```bash
gwx sheets diff {SHEET_ID} --tab1 "Week 11" --tab2 "Week 12" --json
```

Shows which features improved/regressed.

## Integration Points

### After `/spec-audit`:
Automatically append results to the dashboard.

### After `/audit-converge`:
Update convergence round count and final P0/P1/P2.

### Weekly review:
Run `/spec-health` standalone to see trends and generate weekly quality report.

## Notes
- Step 1 is one-time (🟡)
- Step 3 runs after each audit (🟡 for writes)
- Steps 4-5 are read-only (🟢)
- Sheet ID should be stored in a project-level config for persistence
- `smart-append` validates column structure automatically
- Falls back gracefully if gwx not connected — audit results still live in sdd_context
