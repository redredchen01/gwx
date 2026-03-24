---
name: notion-sync
description: "Cross-platform sync: query a Notion database and append rows to a Google Sheets spreadsheet."
services: [notion, sheets]
safety_tier: red
combo: [gwx-multi, cross-platform]
---

# Notion → Sheets Sync Workflow

## Purpose

One-way sync from a Notion database to a Google Sheets spreadsheet. Queries Notion, maps properties to columns, and appends new rows to Sheets.

## Trigger
User says: "sync notion to sheets", "notion sync", "notion 同步到 sheets", "notion-sync"

## Steps

### Step 1: Query Notion database

```bash
gwx notion query {DATABASE_ID} --limit 50 -f json
```

User must provide:
- **Database ID** (required)
- **Filter** (optional, Notion filter JSON)

Extract from each result:
- Page title
- All properties (text, number, date, select, etc.)

### Step 2: Map properties to columns

Auto-detect column mapping from the first row:
- Notion property names → Sheet column headers
- Type conversion:
  - `title` → plain text
  - `rich_text` → plain text
  - `number` → number
  - `date` → ISO date string
  - `select` / `multi_select` → comma-separated text
  - `checkbox` → TRUE/FALSE
  - `url` → URL string

Present the mapping to the user for confirmation.

### Step 3: Describe target sheet

```bash
gwx sheets describe {SPREADSHEET_ID} --range "{RANGE}" -f json
```

Validate:
- Sheet exists and is accessible
- Column count matches mapped properties
- No type conflicts

### Step 4: Append to Google Sheets (requires confirmation)

```bash
gwx sheets append {SPREADSHEET_ID} --range "{RANGE}" --values '{mapped_rows_json}'
```

**This step requires explicit user confirmation** with:
- Row count to be appended
- Preview of first 3 rows
- Target spreadsheet and range

## Notes
- Step 1 is a Notion read operation
- Step 3 is a Sheets read operation
- Step 4 is a Sheets write operation requiring confirmation
- User must provide both Notion database ID and Sheets spreadsheet ID
- This is a one-way append, not a full bidirectional sync
- Duplicate detection is not built in — user should filter in Notion or check Sheets
