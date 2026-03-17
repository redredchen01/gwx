---
name: sheet-to-email
description: "Read recipient data from a Google Sheet and send personalized emails to each row."
services: [sheets, gmail]
safety_tier: red
---

# Sheet to Email Workflow

## Trigger
User says: "從試算表寄信", "mail merge", "批次寄信", "sheet to email"

## Steps

### Step 1: Read the spreadsheet (🟢)
```bash
gwx sheets read SHEET_ID "A:D" --json
```
Expect columns: Name, Email, Subject, Body (or user-specified mapping).

### Step 2: Preview recipients (🔴 HARD GATE)
**MUST** show the full recipient list before proceeding:

```
Found {N} recipients:
1. {name1} <{email1}> — Subject: {subject1}
2. {name2} <{email2}> — Subject: {subject2}
...

⚠️ This will send {N} real emails. Proceed? (yes/no)
```

**Only proceed with explicit "yes".**

### Step 3: Send emails sequentially
For each row:
```bash
gwx gmail send --to "{email}" --subject "{subject}" --body "{body}" --json
```

The rate limiter automatically spaces requests (Gmail: 4 QPS).

Report progress: "Sent {i}/{N}: {email}"

### Step 4: Summary
```
✓ Sent {success_count}/{total} emails
✗ Failed: {failed_emails with error messages}
```

## Notes
- This is a 🔴 Tier 3 operation — requires explicit confirmation for the ENTIRE batch
- Never auto-execute this workflow
- If any send fails, continue with remaining and report all failures at the end
- Maximum recommended batch: 50 emails (Gmail daily limit considerations)
