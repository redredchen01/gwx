---
name: bq-daily
description: "Run a daily BigQuery report: execute a predefined SQL query and format the results into a summary."
services: [bigquery]
safety_tier: green
---

# BigQuery Daily Report Workflow

## Trigger
User says: "bq daily", "bigquery report", "daily query", "每日查詢", "BQ 報表"

## Steps

### Step 1: List available datasets
```bash
gwx bigquery datasets --project {project_id} --json
```
Confirm the target dataset exists.

### Step 2: Run the daily query
```bash
gwx bigquery query "SELECT date, COUNT(*) as events, COUNT(DISTINCT user_id) as users FROM \`{project}.{dataset}.{table}\` WHERE date = CURRENT_DATE() - 1 GROUP BY date" --project {project_id} --limit 100 --json
```
Execute the daily summary query.

### Step 3: Compile report

```
# Daily BQ Report — {yesterday's date}

## Query Results
- Project: {project_id}
- Rows returned: {N}

## Summary Table
| Column 1 | Column 2 | ... |
|----------|----------|-----|
| value    | value    | ... |

## Notes
- Query processed {bytes} bytes
- Cache hit: {yes/no}
```

## Notes
- All read-only operations (bigquery.readonly scope)
- Replace the SQL template with your actual daily query
- Set default project: `gwx config set bigquery.default-project <id>`
