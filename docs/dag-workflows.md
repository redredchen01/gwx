# DAG Workflows Guide

A Directed Acyclic Graph (DAG) workflow lets you define complex dependencies between steps. Instead of running steps sequentially or in parallel, you can create precise control over which steps must complete before others begin.

## Why Use DAGs?

- **Explicit dependencies** — Clearly state which steps depend on which other steps
- **Parallelism** — Independent steps run concurrently, reducing total execution time
- **Error recovery** — Skip failed steps while continuing with downstream tasks
- **Complex logic** — Model real-world workflows with multiple parallel branches and convergence points

## Core Concepts

### Sequential Workflow (Default)

Steps run one at a time, in order:

```
Step 1 → Step 2 → Step 3 → Step 4
```

```yaml
steps:
  - id: step_1
    tool: gmail.list

  - id: step_2
    tool: transform
    # Depends on step_1 implicitly (previous step)

  - id: step_3
    tool: sheets.append
    # Depends on step_2 implicitly

  - id: step_4
    tool: echo
    # Depends on step_3 implicitly
```

### Parallel Workflow

All independent steps run simultaneously:

```
    ↙ Step 2 ↘
Step 1 →       → Step 4
    ↘ Step 3 ↙
```

```yaml
steps:
  - id: step_1
    tool: api.get

  - id: step_2
    tool: gmail.list
    Parallel: true
    # Does NOT depend on step_1

  - id: step_3
    tool: calendar.list
    Parallel: true
    # Does NOT depend on step_1

  - id: step_4
    tool: transform
    # Implicitly depends on steps 2 and 3 (they finished before this)
```

### DAG Workflow

Explicitly declare which steps depend on which:

```
Step 1 → Step 2 → Step 4
         ↓
         Step 3 ↘
```

```yaml
steps:
  - id: step_1
    tool: api.get
    store: data

  - id: step_2
    tool: transform
    DependsOn: [step_1]
    args:
      input: "{{.steps.step_1}}"

  - id: step_3
    tool: validate
    DependsOn: [step_1]  # Also depends on step_1
    args:
      input: "{{.steps.step_1}}"

  - id: step_4
    tool: merge
    DependsOn: [step_2, step_3]  # Waits for both step_2 and step_3
    args:
      data1: "{{.steps.step_2}}"
      data2: "{{.steps.step_3}}"
```

## Common Patterns

### Fan-Out / Fan-In

Fetch data from multiple sources in parallel, then merge:

```
        ↙ Gmail ↘
        ↓       ↓
Start → Calendar → Merge → Output
        ↓       ↓
        ↙ Drive ↘
```

```yaml
name: multi-source-digest
version: "1.0"
description: "Collect from Gmail, Calendar, and Drive"

steps:
  - id: start
    tool: echo
    args:
      message: "Collecting data..."

  - id: fetch_gmail
    tool: gmail.list
    Parallel: true
    args:
      query: "is:unread"

  - id: fetch_calendar
    tool: calendar.list
    Parallel: true
    args:
      days: 7

  - id: fetch_drive
    tool: drive.list
    Parallel: true
    args:
      folder: "root"

  - id: merge_results
    tool: transform
    DependsOn: [fetch_gmail, fetch_calendar, fetch_drive]
    args:
      gmail: "{{.steps.fetch_gmail}}"
      calendar: "{{.steps.fetch_calendar}}"
      drive: "{{.steps.fetch_drive}}"
      format: "json"

  - id: output
    tool: sheets.append
    DependsOn: [merge_results]
    args:
      sheet_id: "{{.inputs.sheet_id}}"
      data: "{{.steps.merge_results}}"
```

### Filter Chain

Process data through a sequence of filters, each depending on the previous:

```
Data → Filter 1 → Filter 2 → Filter 3 → Output
```

```yaml
name: email-cleanup
version: "1.0"

steps:
  - id: fetch_emails
    tool: gmail.list
    args:
      query: "is:unread"

  - id: remove_spam
    tool: transform
    DependsOn: [fetch_emails]
    args:
      input: "{{.steps.fetch_emails}}"
      filter: ".[] | select(.spam == false)"

  - id: extract_important
    tool: transform
    DependsOn: [remove_spam]
    args:
      input: "{{.steps.remove_spam}}"
      filter: ".[] | select(.starred == true)"

  - id: prioritize
    tool: transform
    DependsOn: [extract_important]
    args:
      input: "{{.steps.extract_important}}"
      sort_by: ".priority"
      sort_order: "descending"

output: |
  {{.steps.prioritize}}
```

### Conditional Fallback

Use `on_fail: continue` with DAG to provide fallback steps:

```
        ↙ Try Primary ↘
Data → Check → Merge → Output
        ↘ Try Fallback ↗
```

```yaml
name: resilient-fetch
version: "1.0"

steps:
  - id: prepare
    tool: echo
    args:
      message: "Starting fetch"

  - id: try_primary_api
    tool: http
    DependsOn: [prepare]
    on_fail: continue  # Don't stop on failure
    args:
      method: GET
      url: "https://primary.example.com/data"
      timeout: 5

  - id: try_fallback_api
    tool: http
    DependsOn: [prepare]  # Also depends on prepare, not on try_primary_api
    args:
      method: GET
      url: "https://fallback.example.com/data"
      timeout: 5

  - id: merge_results
    tool: transform
    DependsOn: [try_primary_api, try_fallback_api]
    args:
      primary: "{{.steps.try_primary_api}}"
      fallback: "{{.steps.try_fallback_api}}"
      prefer: "primary"
```

### Multi-Stage Pipeline

Complex workflow with validation, transformation, and storage:

```
Input → Extract → Validate → Transform → Store → Notify
```

```yaml
name: data-pipeline
version: "1.0"
description: "Complete ETL pipeline"

steps:
  - id: extract
    tool: gmail.list
    args:
      query: "has:attachment"

  - id: validate
    tool: transform
    DependsOn: [extract]
    args:
      input: "{{.steps.extract}}"
      filter: ".[] | select(.size < 10000000)"  # Reject files > 10MB

  - id: transform
    tool: transform
    DependsOn: [validate]
    args:
      input: "{{.steps.validate}}"
      format: "csv"
      include: ["from", "subject", "date"]

  - id: store
    tool: sheets.append
    DependsOn: [transform]
    args:
      sheet_id: "{{.inputs.sheet_id}}"
      data: "{{.steps.transform}}"

  - id: notify_slack
    tool: http
    DependsOn: [store]
    on_fail: continue  # Don't fail skill if notification fails
    args:
      method: POST
      url: "{{.env.SLACK_WEBHOOK}}"
      body: "{{.steps.store}}"
```

## Debugging DAG Workflows

### Visualization

View your workflow as a dependency graph:

```bash
gwx skill inspect my-workflow
```

The output shows all `DependsOn` relationships.

### Storing Intermediate Results

Use `store:` to keep step outputs for inspection:

```yaml
steps:
  - id: step_1
    tool: api.get
    store: raw_data

  - id: step_2
    tool: transform
    store: transformed_data
    DependsOn: [step_1]

  # Later steps can reference stored values
```

### Step Timing

For long-running workflows, monitor which steps block others:

```bash
gwx skill run my-workflow --verbose
```

Shows timing for each step.

## Best Practices

### 1. Make Dependencies Explicit

Instead of relying on step order, always use `DependsOn` in DAG workflows:

```yaml
# Bad: Relies on step order
steps:
  - id: fetch
    tool: api.get

  - id: process
    tool: transform

# Good: Explicit dependency
steps:
  - id: fetch
    tool: api.get

  - id: process
    tool: transform
    DependsOn: [fetch]
```

### 2. Parallelize Independent Work

```yaml
# Bad: Sequential, wastes time
steps:
  - id: fetch_gmail
    tool: gmail.list

  - id: fetch_calendar
    tool: calendar.list
    DependsOn: [fetch_gmail]

# Good: Parallel
steps:
  - id: fetch_gmail
    tool: gmail.list
    Parallel: true

  - id: fetch_calendar
    tool: calendar.list
    Parallel: true

  - id: merge
    tool: transform
    DependsOn: [fetch_gmail, fetch_calendar]
```

### 3. Fail Fast or Fail Gracefully

Decide upfront: should a step failure stop the entire workflow, or continue with fallbacks?

```yaml
steps:
  - id: critical_step
    tool: sheets.append
    on_fail: abort  # Stop the workflow

  - id: optional_step
    tool: http
    on_fail: continue  # Try fallback
    DependsOn: [critical_step]
```

### 4. Name Steps Clearly

Use step IDs that describe what happens, not order:

```yaml
# Bad
steps:
  - id: step_1
  - id: step_2

# Good
steps:
  - id: fetch_unread_emails
  - id: validate_attachments
  - id: append_to_sheet
```

### 5. Limit DAG Complexity

If your skill has more than 10-15 steps, consider breaking it into smaller skills:

```yaml
# Instead of one mega-skill with 20 steps:
# 1. Create skill: fetch-and-validate
# 2. Create skill: transform-and-store
# 3. Call them in sequence from a wrapper skill
```

## Testing DAG Workflows

```go
func TestDAGWorkflow(t *testing.T) {
    mc := testutil.NewMockCaller()
    mc.On("api.get", testData, nil)
    mc.On("transform", transformedData, nil)
    mc.On("sheets.append", "row_id", nil)

    skill := testutil.NewSkillBuilder("dag-test").
        WithDAGStep("fetch", "api.get", map[string]string{}, []string{}).
        WithDAGStep("transform", "transform", map[string]string{}, []string{"fetch"}).
        WithDAGStep("store", "sheets.append", map[string]string{}, []string{"transform"}).
        Build()

    // Verify step dependencies
    if len(skill.Steps[2].DependsOn) != 1 || skill.Steps[2].DependsOn[0] != "transform" {
        t.Fatal("store should depend on transform")
    }
}
```

## Performance Considerations

### Execution Time

DAG workflows calculate the **critical path** — the longest chain of dependent steps. This is your minimum execution time.

Example:
```yaml
# Total time: Step1 (5s) + Step2 (10s) = 15s minimum
# (Step3 and Step4 run in parallel with Step2)
```

### Resource Usage

Parallel steps consume more system resources. Monitor:

- CPU usage during parallel stages
- Memory footprint of cached step outputs
- API rate limits when parallelizing requests

### Optimization Tips

1. Parallel independent steps (biggest impact)
2. Cache expensive computations with `store:`
3. Use `on_fail: continue` to prevent cascading failures
4. Filter/validate early to reduce downstream data volume

## Advanced: Custom Tool Integration

When writing custom tools for DAG workflows, ensure they support:

- **Timeouts** — Don't run forever
- **Error reporting** — Clear error messages for debugging
- **Data consistency** — Handle partial results gracefully
- **Idempotency** — Safe to retry without side effects

See `/docs/custom-tools.md` for implementation details.
