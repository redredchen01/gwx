package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// ObsidianCmd groups Obsidian vault operations.
// Obsidian uses local filesystem — no API auth needed.
type ObsidianCmd struct {
	Setup   ObsidianSetupCmd   `cmd:"" help:"Set vault path"`
	List    ObsidianListCmd    `cmd:"" help:"List notes"`
	Search  ObsidianSearchCmd  `cmd:"" help:"Search note content"`
	Read    ObsidianReadCmd    `cmd:"" help:"Read a note"`
	Create  ObsidianCreateCmd  `cmd:"" help:"Create a note"`
	Append  ObsidianAppendCmd  `cmd:"" help:"Append to a note"`
	Daily   ObsidianDailyCmd   `cmd:"" help:"Create/append daily note"`
	Tags    ObsidianTagsCmd    `cmd:"" help:"List all tags"`
	Recent  ObsidianRecentCmd  `cmd:"" help:"List recently modified notes"`
	Folders ObsidianFoldersCmd `cmd:"" help:"List vault folders"`
}

// loadVault reads the vault path from config and creates an ObsidianVault.
func loadVault() (*api.ObsidianVault, error) {
	vaultPath, err := config.Get("obsidian.vault")
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if vaultPath == "" {
		return nil, fmt.Errorf("vault path not configured. Run 'gwx obsidian setup /path/to/vault' first")
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

// --- Setup ---

// ObsidianSetupCmd saves the vault path to config.
type ObsidianSetupCmd struct {
	VaultPath string `arg:"" help:"Absolute path to Obsidian vault directory"`
}

func (c *ObsidianSetupCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.setup"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	// Validate the path before saving
	_, err := api.NewObsidianVault(c.VaultPath)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	if err := config.Set("obsidian.vault", c.VaultPath); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("save config: %s", err))
	}

	rctx.Printer.Success(map[string]interface{}{
		"configured": true,
		"vault_path": c.VaultPath,
	})
	return nil
}

// --- List ---

// ObsidianListCmd lists notes in the vault.
type ObsidianListCmd struct {
	Folder string `help:"Filter by folder (relative to vault root)" short:"d"`
	Limit  int    `help:"Max notes to return" default:"20" short:"n"`
}

func (c *ObsidianListCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.list"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	notes, err := vault.ListNotes(c.Folder, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"notes": notes,
		"count": len(notes),
	})
	return nil
}

// --- Search ---

// ObsidianSearchCmd searches note content.
type ObsidianSearchCmd struct {
	Query string `arg:"" help:"Search query"`
	Limit int    `help:"Max results" default:"10" short:"n"`
}

func (c *ObsidianSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	results, err := vault.SearchNotes(c.Query, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"query":   c.Query,
		"results": results,
		"count":   len(results),
	})
	return nil
}

// --- Read ---

// ObsidianReadCmd reads a single note.
type ObsidianReadCmd struct {
	NotePath string `arg:"" help:"Note path relative to vault root"`
}

func (c *ObsidianReadCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.read"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	note, err := vault.ReadNote(c.NotePath)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.NotFound, err.Error())
	}

	rctx.Printer.Success(note)
	return nil
}

// --- Create ---

// ObsidianCreateCmd creates a new note.
type ObsidianCreateCmd struct {
	NotePath string `arg:"" help:"Note path relative to vault root"`
	Content  string `help:"Note content (or pipe via stdin)" short:"c"`
}

func (c *ObsidianCreateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.create"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	content := c.Content
	if content == "" {
		// Try reading from stdin if not a TTY
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, readErr := io.ReadAll(os.Stdin)
			if readErr == nil {
				content = strings.TrimSpace(string(data))
			}
		}
	}

	result, err := vault.CreateNote(c.NotePath, content)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(result)
	return nil
}

// --- Append ---

// ObsidianAppendCmd appends text to an existing note.
type ObsidianAppendCmd struct {
	NotePath string `arg:"" help:"Note path relative to vault root"`
	Text     string `help:"Text to append" required:"" short:"t"`
}

func (c *ObsidianAppendCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.append"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	result, err := vault.AppendNote(c.NotePath, c.Text)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(result)
	return nil
}

// --- Daily ---

// ObsidianDailyCmd creates or appends to today's daily note.
type ObsidianDailyCmd struct {
	Text string `help:"Text for daily note (or pipe via stdin)" short:"t"`
}

func (c *ObsidianDailyCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.daily"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	text := c.Text
	if text == "" {
		// Try reading from stdin if not a TTY
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, readErr := io.ReadAll(os.Stdin)
			if readErr == nil {
				text = strings.TrimSpace(string(data))
			}
		}
	}
	if text == "" {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, "no text provided. Use --text or pipe via stdin")
	}

	result, err := vault.DailyNote(text)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(result)
	return nil
}

// --- Tags ---

// ObsidianTagsCmd lists all tags in the vault.
type ObsidianTagsCmd struct{}

func (c *ObsidianTagsCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.tags"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	tags, err := vault.ListTags()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"tags":  tags,
		"count": len(tags),
	})
	return nil
}

// --- Recent ---

// ObsidianRecentCmd lists recently modified notes.
type ObsidianRecentCmd struct {
	Limit int `help:"Max notes to return" default:"10" short:"n"`
}

func (c *ObsidianRecentCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.recent"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	notes, err := vault.RecentNotes(c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"notes": notes,
		"count": len(notes),
	})
	return nil
}

// --- Folders ---

// ObsidianFoldersCmd lists all folders in the vault.
type ObsidianFoldersCmd struct{}

func (c *ObsidianFoldersCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "obsidian.folders"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	vault, err := loadVault()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	folders, err := vault.ListFolders()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"folders": folders,
		"count":   len(folders),
	})
	return nil
}
