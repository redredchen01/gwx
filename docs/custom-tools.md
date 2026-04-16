# Custom Tools Guide

GWX supports custom tools beyond the standard MCP-based integrations: **shell** and **http**. This guide covers when to use them, how to configure them safely, and best practices.

## Shell Tool

The `shell` tool executes system commands. It's powerful but requires careful security considerations.

### Basic Usage

```yaml
steps:
  - id: run_curl
    tool: shell
    args:
      command: "curl https://api.example.com/data"
      timeout: 30
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `command` | string | Yes | Shell command to execute |
| `timeout` | number | No | Timeout in seconds (default: 60) |
| `shell` | string | No | Shell to use: `bash`, `sh`, `zsh` (default: system default) |
| `env` | object | No | Environment variables to set |
| `cwd` | string | No | Working directory |

### Examples

#### Simple Command

```yaml
- id: get_timestamp
  tool: shell
  args:
    command: "date +%s"
```

#### With Environment Variables

```yaml
- id: deploy
  tool: shell
  args:
    command: "deploy.sh"
    env:
      ENV: "production"
      API_KEY: "{{.env.DEPLOY_KEY}}"
```

#### With Working Directory

```yaml
- id: build
  tool: shell
  args:
    command: "npm run build"
    cwd: "/home/user/project"
    timeout: 300  # 5 minutes
```

#### Piped Commands

```yaml
- id: process_logs
  tool: shell
  args:
    command: |
      cat /var/log/app.log |
      grep "ERROR" |
      wc -l
```

### Security Best Practices

#### 1. Never Embed Secrets in Commands

```yaml
# BAD: Secret hardcoded
- id: call_api
  tool: shell
  args:
    command: "curl -H 'Authorization: Bearer sk-123456' https://api.example.com"

# GOOD: Use environment variables
- id: call_api
  tool: shell
  args:
    command: "curl -H 'Authorization: Bearer $API_KEY' https://api.example.com"
    env:
      API_KEY: "{{.env.API_KEY}}"
```

#### 2. Validate User Input

```yaml
# BAD: Direct interpolation
- id: risky
  tool: shell
  args:
    command: "rm -rf {{.inputs.path}}"

# GOOD: Validate and whitelist
- id: safe
  tool: shell
  args:
    command: |
      if [[ "{{.inputs.path}}" =~ ^[a-zA-Z0-9_/.-]+$ ]]; then
        rm -rf "{{.inputs.path}}"
      else
        echo "Invalid path"
        exit 1
      fi
```

#### 3. Use Absolute Paths

```yaml
# BAD: Relative path
- id: run_script
  tool: shell
  args:
    command: "./deploy.sh"

# GOOD: Absolute path
- id: run_script
  tool: shell
  args:
    command: "/opt/scripts/deploy.sh"
```

#### 4. Timeout Critical Operations

```yaml
# BAD: Could hang forever
- id: wait_for_service
  tool: shell
  args:
    command: "while ! ping google.com; do sleep 1; done"

# GOOD: Reasonable timeout
- id: wait_for_service
  tool: shell
  args:
    command: "timeout 60 bash -c 'while ! ping -c 1 google.com; do sleep 1; done'"
    timeout: 65  # Skill timeout > command timeout
```

#### 5. Check Exit Codes

```yaml
- id: critical_operation
  tool: shell
  on_fail: abort  # Stop if exit code is non-zero
  args:
    command: "critical-command.sh && echo 'Success'"
```

## HTTP Tool

The `http` tool makes HTTP requests (GET, POST, PUT, DELETE, PATCH).

### Basic Usage

```yaml
steps:
  - id: fetch_data
    tool: http
    args:
      method: GET
      url: "https://api.example.com/data"
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `method` | string | Yes | HTTP method (GET, POST, PUT, DELETE, PATCH) |
| `url` | string | Yes | Full URL to call |
| `headers` | object | No | HTTP headers to send |
| `body` | string/object | No | Request body (POST/PUT) |
| `query` | object | No | Query string parameters |
| `timeout` | number | No | Timeout in seconds (default: 30) |
| `follow_redirects` | boolean | No | Follow HTTP redirects (default: true) |
| `auth` | object | No | Basic auth credentials |

### Examples

#### GET Request

```yaml
- id: fetch_github_user
  tool: http
  args:
    method: GET
    url: "https://api.github.com/users/{{.inputs.username}}"
    headers:
      Accept: "application/json"
```

#### POST with JSON Body

```yaml
- id: create_issue
  tool: http
  args:
    method: POST
    url: "https://api.github.com/repos/{{.inputs.owner}}/{{.inputs.repo}}/issues"
    headers:
      Authorization: "token {{.env.GITHUB_TOKEN}}"
      Content-Type: "application/json"
    body: |
      {
        "title": "{{.inputs.title}}",
        "body": "{{.inputs.body}}"
      }
```

#### Query Parameters

```yaml
- id: search_github
  tool: http
  args:
    method: GET
    url: "https://api.github.com/search/repositories"
    query:
      q: "language:go stars:>1000"
      sort: "stars"
      order: "desc"
    headers:
      Authorization: "token {{.env.GITHUB_TOKEN}}"
```

#### Basic Authentication

```yaml
- id: call_private_api
  tool: http
  args:
    method: GET
    url: "https://internal-api.example.com/data"
    auth:
      username: "{{.env.API_USER}}"
      password: "{{.env.API_PASSWORD}}"
```

#### Conditional Requests

```yaml
- id: check_deployment
  tool: http
  on_fail: continue  # Don't fail if endpoint returns error
  args:
    method: GET
    url: "https://api.example.com/status"
    timeout: 10
```

### Security Best Practices

#### 1. Use HTTPS Only

```yaml
# BAD: Unencrypted
- id: bad_request
  tool: http
  args:
    method: GET
    url: "http://api.example.com/data"  # Not encrypted

# GOOD: Use HTTPS
- id: good_request
  tool: http
  args:
    method: GET
    url: "https://api.example.com/data"
```

#### 2. Validate URLs

```yaml
# BAD: Could be injected
- id: risky
  tool: http
  args:
    method: GET
    url: "{{.inputs.endpoint}}"

# GOOD: Validate before use
inputs:
  - name: service_name
    type: string
    required: true
    description: "Service name (whitelist: github, gitlab)"

steps:
  - id: validate_and_call
    tool: shell
    args:
      command: |
        case "{{.inputs.service_name}}" in
          github) URL="https://api.github.com/status" ;;
          gitlab) URL="https://gitlab.com/api/v4/health" ;;
          *) echo "Invalid service"; exit 1 ;;
        esac
        echo "$URL"
```

#### 3. Handle Sensitive Response Data

```yaml
# BAD: Returns entire response (might contain secrets)
- id: get_config
  tool: http
  args:
    method: GET
    url: "https://config-api.example.com/settings"

# GOOD: Extract only needed fields
- id: get_config
  tool: http
  args:
    method: GET
    url: "https://config-api.example.com/settings"

- id: extract_public_config
  tool: transform
  DependsOn: [get_config]
  args:
    input: "{{.steps.get_config}}"
    filter: "{version: .version, enabled: .enabled}"  # Only public fields
```

#### 4. Protect Authentication Tokens

```yaml
# BAD: Token in plain view
- id: call_api
  tool: http
  args:
    method: GET
    url: "https://api.example.com/data"
    headers:
      Authorization: "Bearer your-secret-token-here"

# GOOD: Use environment variables
- id: call_api
  tool: http
  args:
    method: GET
    url: "https://api.example.com/data"
    headers:
      Authorization: "Bearer {{.env.API_TOKEN}}"
```

#### 5. Timeout External Calls

```yaml
- id: external_service
  tool: http
  args:
    method: GET
    url: "https://external-api.example.com/data"
    timeout: 15  # Don't wait forever
```

#### 6. Validate Response Format

```yaml
- id: fetch_json
  tool: http
  args:
    method: GET
    url: "https://api.example.com/data"
    headers:
      Accept: "application/json"

- id: validate_response
  tool: transform
  DependsOn: [fetch_json]
  args:
    input: "{{.steps.fetch_json}}"
    filter: "if type == \"object\" then . else error(\"Invalid response format\") end"
```

## Choosing Between Shell and HTTP

### Use HTTP When:

- Calling REST APIs or webhooks
- Need built-in authentication handling
- Working with structured data (JSON)
- Error handling is important

### Use Shell When:

- Need to run system commands
- Combining multiple tools with pipes
- Running scripts or binaries
- Need complex string processing

### Example Comparison

```yaml
# Task: Fetch data from API and save to file

# Shell approach
- id: fetch_and_save
  tool: shell
  args:
    command: "curl -s https://api.example.com/data | jq . > /tmp/data.json"

# HTTP + Shell approach (safer)
- id: fetch
  tool: http
  args:
    method: GET
    url: "https://api.example.com/data"

- id: save
  tool: shell
  args:
    command: "cat > /tmp/data.json"
    env:
      DATA: "{{.steps.fetch}}"
```

## Error Handling

### Shell Errors

```yaml
- id: operation
  tool: shell
  on_fail: abort  # Fail on non-zero exit
  args:
    command: "critical-operation.sh"
```

### HTTP Errors

```yaml
- id: api_call
  tool: http
  on_fail: continue  # Don't fail on 4xx/5xx
  args:
    method: GET
    url: "https://api.example.com/optional-data"

- id: fallback
  tool: http
  DependsOn: [api_call]
  args:
    method: GET
    url: "https://fallback-api.example.com/data"
```

## Debugging

### See Command Output

```yaml
- id: debug
  tool: shell
  args:
    command: "echo 'Debug: {{.inputs.value}}' && your-command.sh"
```

### Test HTTP Connectivity

```bash
gwx skill new test-http -t http
gwx skill validate test-http.yaml
gwx skill run test-http.yaml --dry-run
```

### Check Response Headers

```yaml
- id: inspect_response
  tool: http
  args:
    method: GET
    url: "https://api.example.com/data"
    headers:
      User-Agent: "gwx/1.0"  # Helps with debugging
```

## Performance Considerations

### Parallel HTTP Requests

```yaml
- id: fetch_user1
  tool: http
  Parallel: true
  args:
    method: GET
    url: "https://api.github.com/users/user1"

- id: fetch_user2
  tool: http
  Parallel: true
  args:
    method: GET
    url: "https://api.github.com/users/user2"

- id: merge
  tool: transform
  DependsOn: [fetch_user1, fetch_user2]
  args:
    user1: "{{.steps.fetch_user1}}"
    user2: "{{.steps.fetch_user2}}"
```

### Rate Limiting

```yaml
- id: batch_requests
  tool: shell
  args:
    command: |
      for i in {1..100}; do
        curl -s "https://api.example.com/item/$i"
        sleep 0.1  # Rate limit: 10 requests/second
      done
```

## Examples

See `skills/` directory for complete, production-ready examples:

- `gmail-daily-digest.yaml` — Uses http to fetch config, shell for processing
- `github-issue-reporter.yaml` — Multiple http requests with error handling
- `data-pipeline.yaml` — Complex DAG with mixed shell and http tools

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Command not found | Use absolute path, check `cwd` |
| Timeout | Increase `timeout` value, check network |
| 401 Unauthorized | Verify token, check `Authorization` header |
| 404 Not Found | Validate URL, check parameters |
| Shell exit code 1 | Set `on_fail: continue`, check script |
| Large response | Stream to file instead of storing in variable |
