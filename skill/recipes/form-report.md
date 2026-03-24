---
name: form-report
description: "Summarize Google Forms responses: fetch form structure, list responses, and produce a statistical overview."
services: [forms]
safety_tier: green
---

# Form Report Workflow

## Trigger
User says: "form report", "form responses", "survey results", "表單報告", "問卷統計"

## Steps

### Step 1: Get form structure
```bash
gwx forms get {form_id} --json
```
Retrieve the form title and all questions.

### Step 2: List responses
```bash
gwx forms responses {form_id} --limit 100 --json
```
Fetch all responses (up to limit).

### Step 3: Compile report

```
# Form Report — {form_title}

## Overview
- Total responses: {N}
- Date range: {earliest} — {latest}

## Question Summary
For each question:
- Question: {title}
- Response breakdown:
  - Choice questions: count per option
  - Text questions: sample answers
  - Scale questions: average, min, max

## Notes
- All responses are read-only
```

## Notes
- All read-only operations
- For large forms (>1000 responses), consider using --limit to paginate
