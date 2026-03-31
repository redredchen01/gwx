# gwx Skills

Skills are YAML pipelines that chain gwx commands into reusable workflows. This guide covers scaffolding, managing, and discovering skills.

## Quick Start: Create a New Skill

Use `gwx skill new` to scaffold a skill without writing YAML from scratch:

```bash
# Create a basic skill
gwx skill new my-automation

# With description and tools
gwx skill new my-automation -d "Automate email tasks" -t gmail.list -t sheets.append

# Output to file
gwx skill new my-automation -o ~/.config/gwx/skills/my-automation.yaml

# Preview YAML
gwx skill new my-automation -t gmail.list | jq .data.yaml
```

This generates a valid, ready-to-use YAML template with input parameters, steps, and output template.

## Structure

```yaml
name: my-skill
version: "1.0"
description: What the skill does

inputs:
  - name: query
    type: string
    required: true
    description: Search term

steps:
  - id: search
    tool: gmail.search
    args:
      query: "{{.query}}"
    store: results

  - id: summarize
    tool: gmail.read
    args:
      id: "{{.results.messages.0.id}}"
    on_fail: skip

output: "{{.results}}"

meta:
  author: your-name
  tags: gmail,search
```

## Installing Skills

```bash
# From a local file
gwx skill install ./my-skill.yaml

# From a URL
gwx skill install https://raw.githubusercontent.com/user/repo/main/skills/foo.yaml

# From a GitHub Gist
gwx skill install https://gist.github.com/user/abc123
```

## Discovering Skills

### Skill Marketplace v2

Search and browse available skills from the marketplace:

```bash
# Search by keyword
gwx skill search gmail
gwx skill search "email automation"

# Search in remote catalog
gwx skill search gmail --source remote

# Browse by tag
gwx skill browse gmail
gwx skill browse sheets
gwx skill browse calendar

# View all available tags
gwx skill browse
```

## Managing Skills

```bash
# List all installed skills
gwx skill list

# Inspect a skill (show inputs, steps, output)
gwx skill inspect my-skill

# Validate before sharing
gwx skill validate ./my-skill.yaml

# Remove a skill
gwx skill remove my-skill

# Install from marketplace
gwx skill install gmail-daily-digest
```

## Skill Locations

- **User skills**: `~/.config/gwx/skills/` (installed via `gwx skill install`)
- **Project skills**: `./skills/` in your project root (committed to git)

Project skills override user skills with the same name.

## Advanced Features

### DAG (Directed Acyclic Graph) Workflows

Define complex dependencies between steps for flexible execution models:

```yaml
steps:
  - id: fetch_emails
    tool: gmail.list
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

  - id: merge_results
    tool: transform
    DependsOn: [fetch_emails, fetch_calendar, fetch_drive]
    args:
      gmail: "{{.steps.fetch_emails}}"
      calendar: "{{.steps.fetch_calendar}}"
      drive: "{{.steps.fetch_drive}}"
```

See `/docs/dag-workflows.md` for detailed DAG patterns and examples.

### Custom Tools: Shell and HTTP

Skills support two custom tool types for maximum flexibility:

#### Shell Commands

Execute system commands with variables and error handling:

```yaml
steps:
  - id: process_data
    tool: shell
    args:
      command: "cat /tmp/data.csv | sort -u | wc -l"
      timeout: 60
```

#### HTTP Requests

Make REST API calls with auth, headers, and body:

```yaml
steps:
  - id: fetch_api
    tool: http
    args:
      method: GET
      url: "https://api.example.com/data"
      headers:
        Authorization: "Bearer {{.env.API_KEY}}"
      timeout: 30
```

See `/docs/custom-tools.md` for security best practices and examples.

## Complete Documentation

- **[Skill DSL Reference](/docs/skill-dsl.md)** — Complete YAML schema, all field types, variable substitution
- **[DAG Workflows Guide](/docs/dag-workflows.md)** — Fan-out/fan-in, filters, pipelines, testing
- **[Custom Tools Guide](/docs/custom-tools.md)** — Shell and HTTP usage, security, error handling

## Sharing Skills

1. Write a YAML file following the structure above.
2. Validate it: `gwx skill validate ./my-skill.yaml`
3. Share via:
   - Git repository (link to the raw YAML)
   - GitHub Gist
   - Marketplace submission (coming soon)
   - Direct file transfer

## Examples

Browse the `skills/` directory in this repository for production-ready examples:

- `gmail-daily-digest.yaml` — Fan-out/fan-in DAG with parallel fetches
- `github-issue-reporter.yaml` — HTTP requests with error handling
- `data-pipeline.yaml` — Multi-stage ETL with validation and storage

## Testing Skills

Use built-in test utilities to validate your skills:

```go
// Go code example
func TestMySkill(t *testing.T) {
    mc := testutil.NewMockCaller()
    mc.On("gmail.list", emails, nil)

    skill := testutil.NewSkillBuilder("my-skill").
        WithStep("fetch", "gmail.list", map[string]string{}).
        Build()

    // Run assertions...
}
```

See `/docs/` for complete testing examples.
