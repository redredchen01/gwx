---
name: email-from-doc
description: "Draft an email using content from a Google Doc as the body."
services: [docs, gmail]
safety_tier: yellow
---

# Email from Doc Workflow

## Trigger
User says: "用那份文件寄信", "email from doc", "把文件內容寄出去"

## Steps

### Step 1: Get the document content
```bash
gwx docs get DOC_ID --json
```
Extract the plain text body from the response.

### Step 2: Draft the email (🟡 confirm)
Show the user:
- To: {recipients}
- Subject: {doc title or user-specified}
- Body: {first 200 chars of doc content}...

After confirmation:
```bash
gwx gmail draft --to "recipient@example.com" --subject "{doc_title}" --body "{doc_body}" --json
```

Or if user wants to send directly (🔴 hard gate):
```bash
gwx gmail send --to "recipient@example.com" --subject "{doc_title}" --body "{doc_body}" --json
```

## Notes
- Step 1 is 🟢 auto-execute
- Step 2 defaults to draft (🟡), escalates to send (🔴) only if user explicitly asks
- Truncate very long documents to first 5000 chars for email body
