package mcp

import (
	"context"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

type obsidianProvider struct{}

func (obsidianProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "obsidian_list",
			Description: "List notes in an Obsidian vault. No auth needed — local filesystem only.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"folder": {Type: "string", Description: "Filter by folder (relative to vault root)"},
					"limit":  {Type: "integer", Description: "Max notes (default 20)"},
				},
			},
		},
		{
			Name:        "obsidian_search",
			Description: "Search note content in an Obsidian vault (case-insensitive full-text search).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Search query"},
					"limit": {Type: "integer", Description: "Max results (default 10)"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "obsidian_read",
			Description: "Read an Obsidian note. Returns content, frontmatter, tags, and wiki links.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "Note path relative to vault root (e.g. 'Projects/my-note.md')"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "obsidian_create",
			Description: "Create a new note in the Obsidian vault. CAUTION: Creates a real file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":    {Type: "string", Description: "Note path relative to vault root"},
					"content": {Type: "string", Description: "Note content (Markdown)"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "obsidian_append",
			Description: "Append text to an existing Obsidian note. CAUTION: Modifies a real file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "Note path relative to vault root"},
					"text": {Type: "string", Description: "Text to append"},
				},
				Required: []string{"path", "text"},
			},
		},
		{
			Name:        "obsidian_daily",
			Description: "Create or append to today's daily note (YYYY-MM-DD.md). CAUTION: Modifies a real file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"text": {Type: "string", Description: "Text for daily note"},
				},
				Required: []string{"text"},
			},
		},
		{
			Name:        "obsidian_tags",
			Description: "List all #tags across the Obsidian vault with occurrence counts.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "obsidian_search_tag",
			Description: "Find all notes containing a specific #tag.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"tag":   {Type: "string", Description: "Tag to search for (with or without # prefix)"},
					"limit": {Type: "integer", Description: "Max results (default 10)"},
				},
				Required: []string{"tag"},
			},
		},
		{
			Name:        "obsidian_recent",
			Description: "List recently modified notes in the Obsidian vault (most recent first).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max notes (default 10)"},
				},
			},
		},
		{
			Name:        "obsidian_folders",
			Description: "List all folders in the Obsidian vault.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
	}
}

// loadObsidianVault loads the vault path from config and creates a vault handle.
// Independent of Google auth — no API client needed.
func loadObsidianVault() (*api.ObsidianVault, error) {
	vaultPath, err := config.Get("obsidian.vault")
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if vaultPath == "" {
		return nil, fmt.Errorf("vault path not configured. Run 'gwx obsidian setup /path/to/vault' or 'gwx config set obsidian.vault /path/to/vault'")
	}

	vault, err := api.NewObsidianVault(vaultPath)
	if err != nil {
		return nil, err
	}

	// Apply daily folder config if set
	dailyFolder, _ := config.Get("obsidian.daily-folder")
	if dailyFolder != "" {
		vault.SetDailyFolder(dailyFolder)
	}

	return vault, nil
}

func (obsidianProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"obsidian_list":       obsidianList,
		"obsidian_search":     obsidianSearch,
		"obsidian_read":       obsidianRead,
		"obsidian_create":     obsidianCreate,
		"obsidian_append":     obsidianAppend,
		"obsidian_daily":      obsidianDaily,
		"obsidian_tags":       obsidianTags,
		"obsidian_search_tag": obsidianSearchTag,
		"obsidian_recent":     obsidianRecent,
		"obsidian_folders":    obsidianFolders,
	}
}

func obsidianList(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	notes, err := vault.ListNotes(strArg(args, "folder"), intArg(args, "limit", 20))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"notes": notes, "count": len(notes)})
}

func obsidianSearch(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	results, err := vault.SearchNotes(strArg(args, "query"), intArg(args, "limit", 10))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"query": strArg(args, "query"), "results": results, "count": len(results)})
}

func obsidianRead(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	note, err := vault.ReadNote(strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	return jsonResult(note)
}

func obsidianCreate(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	result, err := vault.CreateNote(strArg(args, "path"), strArg(args, "content"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func obsidianAppend(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	result, err := vault.AppendNote(strArg(args, "path"), strArg(args, "text"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func obsidianDaily(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	result, err := vault.DailyNote(strArg(args, "text"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func obsidianTags(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	tags, err := vault.ListTags()
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"tags": tags, "count": len(tags)})
}

func obsidianSearchTag(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	results, err := vault.SearchByTag(strArg(args, "tag"), intArg(args, "limit", 10))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"tag": strArg(args, "tag"), "results": results, "count": len(results)})
}

func obsidianRecent(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	notes, err := vault.RecentNotes(intArg(args, "limit", 10))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"notes": notes, "count": len(notes)})
}

func obsidianFolders(_ context.Context, args map[string]interface{}) (*ToolResult, error) {
	vault, err := loadObsidianVault()
	if err != nil {
		return nil, err
	}
	folders, err := vault.ListFolders()
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"folders": folders, "count": len(folders)})
}

func init() { RegisterProvider(obsidianProvider{}) }
