# gwx Skills

Skills are YAML pipelines that chain gwx commands into reusable workflows.

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

## Managing Skills

```bash
# List all installed skills
gwx skill list

# Inspect a skill
gwx skill inspect my-skill

# Validate before sharing
gwx skill validate ./my-skill.yaml

# Remove a skill
gwx skill remove my-skill
```

## Skill Locations

- **User skills**: `~/.config/gwx/skills/` (installed via `gwx skill install`)
- **Project skills**: `./skills/` in your project root (committed to git)

Project skills override user skills with the same name.

## Sharing Skills

1. Write a YAML file following the structure above.
2. Validate it: `gwx skill validate my-skill.yaml`
3. Share via:
   - Git repository (link to the raw YAML)
   - GitHub Gist
   - Direct file transfer
