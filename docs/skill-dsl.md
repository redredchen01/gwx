# Skill DSL Reference

Skill DSL is a YAML-based domain-specific language for defining automated workflows in GWX. A skill is a reusable automation bundle that chains multiple tools together to accomplish a task.

## Top-Level Structure

```yaml
name: string              # Required: Unique skill identifier (lowercase, hyphens/underscores)
version: string           # Required: Semantic version (e.g., "1.0")
description: string       # Optional: Human-readable description
inputs: [Input]           # Optional: Input parameters the skill accepts
steps: [Step]             # Required: Sequential or DAG-based workflow steps
output: string            # Optional: Output template (Handlebars syntax)
meta: object              # Optional: Metadata (author, tags, etc.)
```

## Input Parameters

Inputs define what data a skill expects from the caller.

```yaml
inputs:
  - name: string           # Required: Parameter name
    type: string           # Required: "string", "number", "boolean", "array", "object"
    required: boolean      # Optional: Default false
    default: any           # Optional: Default value if not provided
    description: string    # Optional: Help text
```

### Example

```yaml
inputs:
  - name: email_query
    type: string
    required: true
    description: "Gmail search query (e.g., 'label:inbox from:boss')"

  - name: sheet_id
    type: string
    required: false
    default: "my-default-sheet"
    description: "Google Sheet ID for results"
```

## Step Definition

Steps are the building blocks of a skill. Each step invokes a tool and passes arguments.

```yaml
steps:
  - id: string             # Required: Unique step ID within the skill
    tool: string           # Required: Tool name (e.g., "gmail.list", "shell", "http")
    args: object           # Optional: Arguments for the tool
    store: string          # Optional: Variable name to store the result
    on_fail: string        # Optional: "abort" (default), "continue", "retry"
    Parallel: boolean      # Optional: Run in parallel (default false)
    DependsOn: [string]    # Optional: List of step IDs this step depends on
```

## Execution Models

### Sequential Execution

Steps run one after another in the order defined. Later steps can reference earlier step outputs.

```yaml
steps:
  - id: fetch_emails
    tool: gmail.list
    args:
      query: "{{.inputs.search_query}}"
    store: emails

  - id: format_results
    tool: transform
    args:
      input: "{{.steps.fetch_emails}}"
      format: "json"
```

### Parallel Execution

Steps with `Parallel: true` run concurrently. Use when steps are independent.

```yaml
steps:
  - id: fetch_gmail
    tool: gmail.list
    Parallel: true
    args:
      query: "{{.inputs.query}}"

  - id: fetch_calendar
    tool: calendar.list
    Parallel: true
    args:
      days: 7

  - id: merge
    tool: transform
    args:
      gmail: "{{.steps.fetch_gmail}}"
      calendar: "{{.steps.fetch_calendar}}"
```

### DAG (Directed Acyclic Graph) Execution

Use `DependsOn` to define complex dependency graphs. Steps only run once all dependencies complete.

```yaml
steps:
  - id: fetch_data
    tool: api.get
    args:
      url: "{{.inputs.api_url}}"

  - id: validate
    tool: transform
    DependsOn: [fetch_data]
    args:
      input: "{{.steps.fetch_data}}"
      schema: "strict"

  - id: persist
    tool: sheets.append
    DependsOn: [validate]
    args:
      sheet_id: "{{.inputs.sheet_id}}"
      data: "{{.steps.validate}}"
```

## Tool Reference

### Built-in Tools

#### `transform`

Manipulate data using Jq-style transformations.

```yaml
- id: transform_data
  tool: transform
  args:
    input: "{{.steps.previous_step}}"
    filter: ".[] | select(.status == 'active')"
    format: "json"
```

#### `echo`

Return a static value (useful for testing).

```yaml
- id: echo_test
  tool: echo
  args:
    message: "Hello, {{.inputs.name}}!"
```

#### `shell`

Execute a shell command (requires explicit enablement for security).

```yaml
- id: run_script
  tool: shell
  args:
    command: "curl https://api.example.com/data"
    timeout: 30
```

#### `http`

Make HTTP requests (GET, POST, PUT, DELETE).

```yaml
- id: api_call
  tool: http
  args:
    method: POST
    url: "{{.inputs.api_endpoint}}"
    headers:
      Authorization: "Bearer {{.env.API_KEY}}"
    body: "{{.steps.prepare_payload}}"
```

### MCP Tools

Invoke tools from registered MCP servers. Tool names follow the format `service.action`.

```yaml
- id: send_email
  tool: gmail.send
  args:
    to: "user@example.com"
    subject: "Report"
    body: "{{.steps.report}}"

- id: create_sheet
  tool: sheets.create
  args:
    title: "Results"
    rows: "{{.steps.data}}"

- id: list_files
  tool: drive.list
  args:
    folder: "root"
    query: "mimeType = 'application/vnd.google-apps.spreadsheet'"
```

## Variable Substitution

Skills support Handlebars-style templating to reference inputs and step outputs.

### Syntax

- `{{.inputs.parameter_name}}` — Reference an input parameter
- `{{.steps.step_id}}` — Reference the output of a step
- `{{.env.VARIABLE_NAME}}` — Reference environment variables
- `{{#if condition}}...{{/if}}` — Conditional rendering

### Examples

```yaml
args:
  # Reference input
  query: "{{.inputs.search_query}}"

  # Reference previous step
  data: "{{.steps.fetch_data}}"

  # Reference environment
  api_key: "{{.env.API_KEY}}"

  # Complex template
  message: "Results for {{.inputs.query}}: {{.steps.count}} items found"
```

## Error Handling

The `on_fail` field controls what happens when a step fails:

- `abort` (default) — Stop skill execution and report error
- `continue` — Ignore the error and proceed to next step
- `retry` — Automatically retry up to 3 times

```yaml
- id: optional_step
  tool: http
  on_fail: continue
  args:
    url: "{{.inputs.fallback_url}}"
```

## Output Template

The `output` field specifies what the skill returns to the caller. It uses Handlebars syntax.

```yaml
output: |
  {
    "success": true,
    "data": {{.steps.final_result}},
    "count": {{.steps.row_count}}
  }
```

If no `output` is specified, the skill returns the output of the last step.

## Metadata

The `meta` section stores optional metadata about the skill.

```yaml
meta:
  author: "email@example.com"
  tags: "gmail,sheets,automation"
  version_history: "1.0: initial release"
```

## Complete Example

```yaml
name: daily-email-digest
version: "1.0"
description: "Collect unread emails and append to a summary sheet"

inputs:
  - name: hours_back
    type: number
    required: false
    default: 24
    description: "How many hours back to look for emails"

  - name: sheet_id
    type: string
    required: true
    description: "Google Sheet ID to append results to"

steps:
  - id: fetch_emails
    tool: gmail.list
    args:
      query: "is:unread newer_than:{{.inputs.hours_back}}h"
    store: emails

  - id: extract_subjects
    tool: transform
    DependsOn: [fetch_emails]
    args:
      input: "{{.steps.fetch_emails}}"
      filter: ".[] | {from: .from, subject: .subject, received: .date}"

  - id: append_to_sheet
    tool: sheets.append
    DependsOn: [extract_subjects]
    args:
      sheet_id: "{{.inputs.sheet_id}}"
      data: "{{.steps.extract_subjects}}"
    on_fail: continue

output: |
  Processed {{.steps.fetch_emails | length}} emails and appended to {{.inputs.sheet_id}}
```

## Best Practices

1. **Use meaningful step IDs** — Make your automation self-documenting
2. **Provide descriptions** — Help future maintainers understand intent
3. **Handle errors gracefully** — Use `on_fail: continue` for optional steps
4. **Store intermediate results** — Use `store:` to make debugging easier
5. **Document complex templates** — Add comments explaining variable references
6. **Validate inputs** — Define required inputs clearly
7. **Version your skills** — Use semantic versioning to track changes

## Validation

Use the `gwx skill validate` command to check YAML syntax:

```bash
gwx skill validate my-skill.yaml
gwx skill validate -
```

## Testing

Use the test utilities in `internal/testutil/` to write automated tests for your skills:

```go
func TestMySkill(t *testing.T) {
    mc := testutil.NewMockCaller()
    mc.On("gmail.list", emails, nil)
    mc.On("sheets.append", "row_123", nil)

    skill := testutil.NewSkillBuilder("test").
        WithStep("fetch", "gmail.list", map[string]string{}).
        Build()
}
```
