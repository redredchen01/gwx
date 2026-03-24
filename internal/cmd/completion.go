package cmd

import "fmt"

// CompletionCmd generates shell completion scripts.
type CompletionCmd struct {
	Bash CompletionBashCmd `cmd:"" help:"Generate bash completion script"`
	Zsh  CompletionZshCmd  `cmd:"" help:"Generate zsh completion script"`
	Fish CompletionFishCmd `cmd:"" help:"Generate fish completion script"`
}

// CompletionBashCmd generates bash completion.
type CompletionBashCmd struct{}

func (c *CompletionBashCmd) Run(rctx *RunContext) error {
	script := `# gwx bash completion
# Usage: eval "$(gwx completion bash)"
_gwx_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local commands="gmail calendar drive docs sheets tasks contacts chat analytics searchconsole slides forms bigquery github slack notion config auth onboard agent schema version skill workflow standup meeting-prep find context pipe mcp-server completion doctor"

    if [ ${COMP_CWORD} -eq 1 ]; then
        COMPREPLY=($(compgen -W "${commands}" -- "${cur}"))
        return
    fi

    local service="${COMP_WORDS[1]}"
    case "${service}" in
        gmail) COMPREPLY=($(compgen -W "list get search labels send draft reply digest archive label forward" -- "${cur}")) ;;
        calendar) COMPREPLY=($(compgen -W "agenda list create update delete find-slot" -- "${cur}")) ;;
        drive) COMPREPLY=($(compgen -W "list search upload download share mkdir" -- "${cur}")) ;;
        docs) COMPREPLY=($(compgen -W "get create append" -- "${cur}")) ;;
        sheets) COMPREPLY=($(compgen -W "get read write append clear create batch-get batch-update" -- "${cur}")) ;;
        tasks) COMPREPLY=($(compgen -W "list get create update complete delete lists" -- "${cur}")) ;;
        contacts) COMPREPLY=($(compgen -W "list search get create update delete" -- "${cur}")) ;;
        chat) COMPREPLY=($(compgen -W "spaces list send" -- "${cur}")) ;;
        analytics) COMPREPLY=($(compgen -W "report realtime properties" -- "${cur}")) ;;
        searchconsole) COMPREPLY=($(compgen -W "query sites inspect" -- "${cur}")) ;;
        slides) COMPREPLY=($(compgen -W "get create add-slide update-text" -- "${cur}")) ;;
        forms) COMPREPLY=($(compgen -W "get responses response" -- "${cur}")) ;;
        bigquery) COMPREPLY=($(compgen -W "query datasets tables describe" -- "${cur}")) ;;
        github) COMPREPLY=($(compgen -W "login logout status repos issues pulls" -- "${cur}")) ;;
        slack) COMPREPLY=($(compgen -W "login status channels send messages" -- "${cur}")) ;;
        notion) COMPREPLY=($(compgen -W "login status search page create" -- "${cur}")) ;;
        config) COMPREPLY=($(compgen -W "get set list" -- "${cur}")) ;;
        auth) COMPREPLY=($(compgen -W "login logout status" -- "${cur}")) ;;
        skill) COMPREPLY=($(compgen -W "list inspect validate run create test install remove" -- "${cur}")) ;;
        workflow) COMPREPLY=($(compgen -W "test-matrix sprint-board spec-health bug-intake context-boost review-notify parallel-schedule email-from-doc sheet-to-email weekly-digest" -- "${cur}")) ;;
        completion) COMPREPLY=($(compgen -W "bash zsh fish" -- "${cur}")) ;;
    esac
}
complete -F _gwx_completions gwx`
	fmt.Println(script)
	return nil
}

// CompletionZshCmd generates zsh completion.
type CompletionZshCmd struct{}

func (c *CompletionZshCmd) Run(rctx *RunContext) error {
	script := `#compdef gwx
# gwx zsh completion
# Usage: eval "$(gwx completion zsh)"

_gwx() {
    local -a commands
    commands=(
        'gmail:Gmail operations'
        'calendar:Calendar operations'
        'drive:Google Drive operations'
        'docs:Google Docs operations'
        'sheets:Google Sheets operations'
        'tasks:Google Tasks operations'
        'contacts:Contacts operations'
        'chat:Google Chat operations'
        'analytics:Google Analytics 4 operations'
        'searchconsole:Google Search Console operations'
        'slides:Google Slides operations'
        'forms:Google Forms operations'
        'bigquery:BigQuery operations'
        'github:GitHub operations'
        'slack:Slack operations'
        'notion:Notion operations'
        'config:Configuration management'
        'auth:Authentication management'
        'onboard:Interactive setup wizard'
        'agent:Agent automation helpers'
        'schema:Print full command schema'
        'version:Print version'
        'skill:Skill DSL operations'
        'workflow:Workflow commands'
        'standup:Daily standup report'
        'meeting-prep:Prepare context for a meeting'
        'find:Search across Gmail + Drive + Contacts'
        'context:Gather all context for a topic'
        'pipe:Chain gwx commands via JSON pipeline'
        'mcp-server:Start MCP server for Claude integration'
        'completion:Generate shell completion scripts'
        'doctor:Diagnose configuration and connectivity'
    )

    _arguments -C \
        '(-f --format)'{-f,--format}'[Output format]:format:(json plain table)' \
        '(-a --account)'{-a,--account}'[Account email to use]:account:' \
        '--fields[Comma-separated fields to include in output]:fields:' \
        '--dry-run[Validate without executing]' \
        '--no-input[Disable interactive prompts]' \
        '--no-cache[Disable caching]' \
        '1:command:->cmd' \
        '*::arg:->args'

    case "$state" in
        cmd)
            _describe 'command' commands
            ;;
        args)
            case "${words[1]}" in
                gmail) _describe 'subcommand' '(list get search labels send draft reply digest archive label forward)' ;;
                calendar) _describe 'subcommand' '(agenda list create update delete find-slot)' ;;
                drive) _describe 'subcommand' '(list search upload download share mkdir)' ;;
                docs) _describe 'subcommand' '(get create append)' ;;
                sheets) _describe 'subcommand' '(get read write append clear create batch-get batch-update)' ;;
                tasks) _describe 'subcommand' '(list get create update complete delete lists)' ;;
                contacts) _describe 'subcommand' '(list search get create update delete)' ;;
                chat) _describe 'subcommand' '(spaces list send)' ;;
                analytics) _describe 'subcommand' '(report realtime properties)' ;;
                searchconsole) _describe 'subcommand' '(query sites inspect)' ;;
                slides) _describe 'subcommand' '(get create add-slide update-text)' ;;
                forms) _describe 'subcommand' '(get responses response)' ;;
                bigquery) _describe 'subcommand' '(query datasets tables describe)' ;;
                github) _describe 'subcommand' '(login logout status repos issues pulls)' ;;
                slack) _describe 'subcommand' '(login status channels send messages)' ;;
                notion) _describe 'subcommand' '(login status search page create)' ;;
                config) _describe 'subcommand' '(get set list)' ;;
                auth) _describe 'subcommand' '(login logout status)' ;;
                skill) _describe 'subcommand' '(list inspect validate run create test install remove)' ;;
                workflow) _describe 'subcommand' '(test-matrix sprint-board spec-health bug-intake context-boost review-notify parallel-schedule email-from-doc sheet-to-email weekly-digest)' ;;
                completion) _describe 'subcommand' '(bash zsh fish)' ;;
            esac
            ;;
    esac
}

_gwx "$@"`
	fmt.Println(script)
	return nil
}

// CompletionFishCmd generates fish completion.
type CompletionFishCmd struct{}

func (c *CompletionFishCmd) Run(rctx *RunContext) error {
	script := `# gwx fish completion
# Usage: gwx completion fish | source

# Disable file completions by default
complete -c gwx -f

# Global flags
complete -c gwx -l format -s f -d "Output format" -ra "json plain table"
complete -c gwx -l account -s a -d "Account email to use"
complete -c gwx -l fields -d "Comma-separated fields to include in output"
complete -c gwx -l dry-run -d "Validate without executing"
complete -c gwx -l no-input -d "Disable interactive prompts"
complete -c gwx -l no-cache -d "Disable caching"

# Top-level commands
complete -c gwx -n "__fish_use_subcommand" -a gmail -d "Gmail operations"
complete -c gwx -n "__fish_use_subcommand" -a calendar -d "Calendar operations"
complete -c gwx -n "__fish_use_subcommand" -a drive -d "Google Drive operations"
complete -c gwx -n "__fish_use_subcommand" -a docs -d "Google Docs operations"
complete -c gwx -n "__fish_use_subcommand" -a sheets -d "Google Sheets operations"
complete -c gwx -n "__fish_use_subcommand" -a tasks -d "Google Tasks operations"
complete -c gwx -n "__fish_use_subcommand" -a contacts -d "Contacts operations"
complete -c gwx -n "__fish_use_subcommand" -a chat -d "Google Chat operations"
complete -c gwx -n "__fish_use_subcommand" -a analytics -d "Google Analytics 4 operations"
complete -c gwx -n "__fish_use_subcommand" -a searchconsole -d "Google Search Console operations"
complete -c gwx -n "__fish_use_subcommand" -a slides -d "Google Slides operations"
complete -c gwx -n "__fish_use_subcommand" -a forms -d "Google Forms operations"
complete -c gwx -n "__fish_use_subcommand" -a bigquery -d "BigQuery operations"
complete -c gwx -n "__fish_use_subcommand" -a github -d "GitHub operations"
complete -c gwx -n "__fish_use_subcommand" -a slack -d "Slack operations"
complete -c gwx -n "__fish_use_subcommand" -a notion -d "Notion operations"
complete -c gwx -n "__fish_use_subcommand" -a config -d "Configuration management"
complete -c gwx -n "__fish_use_subcommand" -a auth -d "Authentication management"
complete -c gwx -n "__fish_use_subcommand" -a onboard -d "Interactive setup wizard"
complete -c gwx -n "__fish_use_subcommand" -a agent -d "Agent automation helpers"
complete -c gwx -n "__fish_use_subcommand" -a schema -d "Print full command schema"
complete -c gwx -n "__fish_use_subcommand" -a version -d "Print version"
complete -c gwx -n "__fish_use_subcommand" -a skill -d "Skill DSL operations"
complete -c gwx -n "__fish_use_subcommand" -a workflow -d "Workflow commands"
complete -c gwx -n "__fish_use_subcommand" -a standup -d "Daily standup report"
complete -c gwx -n "__fish_use_subcommand" -a meeting-prep -d "Prepare context for a meeting"
complete -c gwx -n "__fish_use_subcommand" -a find -d "Search across Gmail + Drive + Contacts"
complete -c gwx -n "__fish_use_subcommand" -a context -d "Gather all context for a topic"
complete -c gwx -n "__fish_use_subcommand" -a pipe -d "Chain gwx commands via JSON pipeline"
complete -c gwx -n "__fish_use_subcommand" -a mcp-server -d "Start MCP server for Claude integration"
complete -c gwx -n "__fish_use_subcommand" -a completion -d "Generate shell completion scripts"
complete -c gwx -n "__fish_use_subcommand" -a doctor -d "Diagnose configuration and connectivity"

# Gmail subcommands
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a list -d "List messages"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a get -d "Get a message"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a search -d "Search messages"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a labels -d "List labels"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a send -d "Send an email"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a draft -d "Create a draft"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a reply -d "Reply to a message"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a digest -d "Email digest"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a archive -d "Archive a message"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a label -d "Apply label"
complete -c gwx -n "__fish_seen_subcommand_from gmail" -a forward -d "Forward a message"

# Calendar subcommands
complete -c gwx -n "__fish_seen_subcommand_from calendar" -a agenda -d "Show agenda"
complete -c gwx -n "__fish_seen_subcommand_from calendar" -a list -d "List events"
complete -c gwx -n "__fish_seen_subcommand_from calendar" -a create -d "Create an event"
complete -c gwx -n "__fish_seen_subcommand_from calendar" -a update -d "Update an event"
complete -c gwx -n "__fish_seen_subcommand_from calendar" -a delete -d "Delete an event"
complete -c gwx -n "__fish_seen_subcommand_from calendar" -a find-slot -d "Find available time slots"

# Drive subcommands
complete -c gwx -n "__fish_seen_subcommand_from drive" -a list -d "List files"
complete -c gwx -n "__fish_seen_subcommand_from drive" -a search -d "Search files"
complete -c gwx -n "__fish_seen_subcommand_from drive" -a upload -d "Upload a file"
complete -c gwx -n "__fish_seen_subcommand_from drive" -a download -d "Download a file"
complete -c gwx -n "__fish_seen_subcommand_from drive" -a share -d "Share a file"
complete -c gwx -n "__fish_seen_subcommand_from drive" -a mkdir -d "Create a folder"

# Skill subcommands
complete -c gwx -n "__fish_seen_subcommand_from skill" -a list -d "List all loaded skills"
complete -c gwx -n "__fish_seen_subcommand_from skill" -a inspect -d "Show details of a skill"
complete -c gwx -n "__fish_seen_subcommand_from skill" -a validate -d "Validate a skill YAML file"
complete -c gwx -n "__fish_seen_subcommand_from skill" -a run -d "Run a skill by name"
complete -c gwx -n "__fish_seen_subcommand_from skill" -a create -d "Create a new skill scaffold"
complete -c gwx -n "__fish_seen_subcommand_from skill" -a test -d "Test a skill with mock data"
complete -c gwx -n "__fish_seen_subcommand_from skill" -a install -d "Install a skill from file or URL"
complete -c gwx -n "__fish_seen_subcommand_from skill" -a remove -d "Remove an installed skill"

# Completion subcommands
complete -c gwx -n "__fish_seen_subcommand_from completion" -a bash -d "Generate bash completion"
complete -c gwx -n "__fish_seen_subcommand_from completion" -a zsh -d "Generate zsh completion"
complete -c gwx -n "__fish_seen_subcommand_from completion" -a fish -d "Generate fish completion"`
	fmt.Println(script)
	return nil
}
