package api

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ObsidianVault provides operations on a local Obsidian vault directory.
// No network calls — all operations are pure filesystem reads/writes.
type ObsidianVault struct {
	path       string
	dailyFolder string // relative folder for daily notes (default: root)
}

// NewObsidianVault creates a vault handle after validating the path.
// The path must exist and contain an .obsidian/ subfolder.
func NewObsidianVault(vaultPath string) (*ObsidianVault, error) {
	if vaultPath == "" {
		return nil, fmt.Errorf("obsidian: vault path is empty")
	}

	absPath, err := filepath.Abs(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("obsidian: resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("obsidian: vault path does not exist: %s", absPath)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("obsidian: vault path is not a directory: %s", absPath)
	}

	dotObsidian := filepath.Join(absPath, ".obsidian")
	if _, err := os.Stat(dotObsidian); err != nil {
		return nil, fmt.Errorf("obsidian: not a valid vault (missing .obsidian/ folder): %s", absPath)
	}

	return &ObsidianVault{path: absPath}, nil
}

// SetDailyFolder configures the relative folder for daily notes.
func (v *ObsidianVault) SetDailyFolder(folder string) {
	v.dailyFolder = folder
}

// safePath validates that a relative note path stays within the vault.
// Returns the absolute path or an error if traversal is detected.
func (v *ObsidianVault) safePath(notePath string) (string, error) {
	if notePath == "" {
		return "", fmt.Errorf("obsidian: note path is empty")
	}

	// Block path traversal
	cleaned := filepath.Clean(notePath)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("obsidian: path traversal not allowed: %s", notePath)
	}

	abs := filepath.Join(v.path, cleaned)

	// Double-check the resolved path is under the vault root
	rel, err := filepath.Rel(v.path, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("obsidian: path traversal not allowed: %s", notePath)
	}

	return abs, nil
}

// isSkippedDir returns true for directories that should be excluded from scans.
func isSkippedDir(name string) bool {
	return name == ".obsidian" || name == ".trash" || name == ".git"
}

// ListNotes lists .md files in the vault, optionally filtered by folder.
func (v *ObsidianVault) ListNotes(folder string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}

	root := v.path
	if folder != "" {
		safe, err := v.safePath(folder)
		if err != nil {
			return nil, err
		}
		root = safe
	}

	var notes []map[string]interface{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if isSkippedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		if len(notes) >= limit {
			return filepath.SkipAll
		}

		rel, _ := filepath.Rel(v.path, path)
		info, infoErr := d.Info()

		entry := map[string]interface{}{
			"path":  rel,
			"title": strings.TrimSuffix(d.Name(), ".md"),
		}
		if infoErr == nil {
			entry["modified"] = info.ModTime().Format(time.RFC3339)
			entry["size"] = info.Size()
		}

		// Quick tag scan from first 50 lines
		tags := quickTags(path)
		if len(tags) > 0 {
			entry["tags"] = tags
		}

		notes = append(notes, entry)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("obsidian: list notes: %w", err)
	}
	return notes, nil
}

// SearchNotes searches all .md files for a content match (case-insensitive).
func (v *ObsidianVault) SearchNotes(query string, limit int) ([]map[string]interface{}, error) {
	if query == "" {
		return nil, fmt.Errorf("obsidian: search query is empty")
	}
	if limit <= 0 {
		limit = 10
	}

	lowerQuery := strings.ToLower(query)
	var results []map[string]interface{}

	err := filepath.WalkDir(v.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isSkippedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		if len(results) >= limit {
			return filepath.SkipAll
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(data)
		if !strings.Contains(strings.ToLower(content), lowerQuery) {
			return nil
		}

		rel, _ := filepath.Rel(v.path, path)

		// Find matching lines with context
		var matches []string
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), lowerQuery) {
				lineNum := i + 1
				match := fmt.Sprintf("L%d: %s", lineNum, strings.TrimSpace(line))
				if len(match) > 200 {
					match = match[:200] + "..."
				}
				matches = append(matches, match)
				if len(matches) >= 3 {
					break
				}
			}
		}

		results = append(results, map[string]interface{}{
			"path":    rel,
			"title":   strings.TrimSuffix(d.Name(), ".md"),
			"matches": matches,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("obsidian: search: %w", err)
	}
	return results, nil
}

// ReadNote reads a note's content and parses frontmatter, links, and tags.
func (v *ObsidianVault) ReadNote(notePath string) (map[string]interface{}, error) {
	abs, err := v.safePath(notePath)
	if err != nil {
		return nil, err
	}

	// Ensure .md extension
	if !strings.HasSuffix(abs, ".md") {
		abs += ".md"
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("obsidian: note not found: %s", notePath)
		}
		return nil, fmt.Errorf("obsidian: read note: %w", err)
	}

	content := string(data)
	rel, _ := filepath.Rel(v.path, abs)
	name := filepath.Base(rel)

	result := map[string]interface{}{
		"path":    rel,
		"title":   strings.TrimSuffix(name, ".md"),
		"content": content,
	}

	// Parse YAML frontmatter
	if fm := parseFrontmatter(content); fm != nil {
		result["frontmatter"] = fm
	}

	// Extract [[wiki links]]
	links := extractLinks(content)
	if len(links) > 0 {
		result["links"] = links
	}

	// Extract #tags
	tags := extractTags(content)
	if len(tags) > 0 {
		result["tags"] = tags
	}

	return result, nil
}

// CreateNote creates a new .md file at the specified path.
func (v *ObsidianVault) CreateNote(notePath, content string) (map[string]interface{}, error) {
	if !strings.HasSuffix(notePath, ".md") {
		notePath += ".md"
	}

	abs, err := v.safePath(notePath)
	if err != nil {
		return nil, err
	}

	// Check if already exists
	if _, err := os.Stat(abs); err == nil {
		return nil, fmt.Errorf("obsidian: note already exists: %s", notePath)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("obsidian: create directory: %w", err)
	}

	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("obsidian: create note: %w", err)
	}

	rel, _ := filepath.Rel(v.path, abs)
	return map[string]interface{}{
		"created": true,
		"path":    rel,
		"title":   strings.TrimSuffix(filepath.Base(rel), ".md"),
		"size":    len(content),
	}, nil
}

// AppendNote appends text to an existing note.
func (v *ObsidianVault) AppendNote(notePath, text string) (map[string]interface{}, error) {
	if !strings.HasSuffix(notePath, ".md") {
		notePath += ".md"
	}

	abs, err := v.safePath(notePath)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return nil, fmt.Errorf("obsidian: note not found: %s", notePath)
	}

	f, err := os.OpenFile(abs, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("obsidian: open note: %w", err)
	}
	defer f.Close()

	// Ensure we start on a new line
	appendText := "\n" + text
	if _, err := f.WriteString(appendText); err != nil {
		return nil, fmt.Errorf("obsidian: append note: %w", err)
	}

	rel, _ := filepath.Rel(v.path, abs)
	return map[string]interface{}{
		"appended": true,
		"path":     rel,
		"added":    len(text),
	}, nil
}

// DailyNote creates or appends to today's daily note.
// Filename is YYYY-MM-DD.md in the configured daily folder.
func (v *ObsidianVault) DailyNote(content string) (map[string]interface{}, error) {
	today := time.Now().Format("2006-01-02")
	filename := today + ".md"

	var notePath string
	if v.dailyFolder != "" {
		notePath = filepath.Join(v.dailyFolder, filename)
	} else {
		notePath = filename
	}

	abs, err := v.safePath(notePath)
	if err != nil {
		return nil, err
	}

	rel, _ := filepath.Rel(v.path, abs)

	// If file exists, append
	if _, err := os.Stat(abs); err == nil {
		f, openErr := os.OpenFile(abs, os.O_APPEND|os.O_WRONLY, 0644)
		if openErr != nil {
			return nil, fmt.Errorf("obsidian: open daily note: %w", openErr)
		}
		defer f.Close()

		appendText := "\n" + content
		if _, writeErr := f.WriteString(appendText); writeErr != nil {
			return nil, fmt.Errorf("obsidian: append daily note: %w", writeErr)
		}

		return map[string]interface{}{
			"action": "appended",
			"path":   rel,
			"date":   today,
			"added":  len(content),
		}, nil
	}

	// Create new daily note
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("obsidian: create daily folder: %w", err)
	}

	header := fmt.Sprintf("# %s\n\n%s", today, content)
	if err := os.WriteFile(abs, []byte(header), 0644); err != nil {
		return nil, fmt.Errorf("obsidian: create daily note: %w", err)
	}

	return map[string]interface{}{
		"action": "created",
		"path":   rel,
		"date":   today,
		"size":   len(header),
	}, nil
}

// tagPattern matches #tags in markdown content, excluding headings.
var tagPattern = regexp.MustCompile(`(?:^|[ \t])#([a-zA-Z0-9_/]+)`)

// ListTags scans all files for #tag patterns and returns unique tags with counts.
func (v *ObsidianVault) ListTags() ([]map[string]interface{}, error) {
	counts := make(map[string]int)

	err := filepath.WalkDir(v.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isSkippedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, match := range tagPattern.FindAllStringSubmatch(string(data), -1) {
			counts[match[1]]++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("obsidian: list tags: %w", err)
	}

	// Sort by count descending
	type tagCount struct {
		tag   string
		count int
	}
	var sorted []tagCount
	for t, c := range counts {
		sorted = append(sorted, tagCount{t, c})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	result := make([]map[string]interface{}, 0, len(sorted))
	for _, tc := range sorted {
		result = append(result, map[string]interface{}{
			"tag":   "#" + tc.tag,
			"count": tc.count,
		})
	}
	return result, nil
}

// SearchByTag finds all notes containing a specific #tag.
func (v *ObsidianVault) SearchByTag(tag string, limit int) ([]map[string]interface{}, error) {
	if tag == "" {
		return nil, fmt.Errorf("obsidian: tag is empty")
	}
	if limit <= 0 {
		limit = 10
	}

	// Normalize: strip leading # if present
	tag = strings.TrimPrefix(tag, "#")
	searchTag := "#" + tag

	var results []map[string]interface{}

	err := filepath.WalkDir(v.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isSkippedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		if len(results) >= limit {
			return filepath.SkipAll
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(data)
		if !strings.Contains(content, searchTag) {
			return nil
		}

		rel, _ := filepath.Rel(v.path, path)
		info, _ := d.Info()
		entry := map[string]interface{}{
			"path":  rel,
			"title": strings.TrimSuffix(d.Name(), ".md"),
		}
		if info != nil {
			entry["modified"] = info.ModTime().Format(time.RFC3339)
		}
		results = append(results, entry)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("obsidian: search by tag: %w", err)
	}
	return results, nil
}

// ListFolders returns all subfolders in the vault (excluding hidden ones).
func (v *ObsidianVault) ListFolders() ([]string, error) {
	var folders []string

	err := filepath.WalkDir(v.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path == v.path {
			return nil // skip root
		}
		if isSkippedDir(d.Name()) {
			return filepath.SkipDir
		}
		// Skip hidden directories
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		rel, _ := filepath.Rel(v.path, path)
		folders = append(folders, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("obsidian: list folders: %w", err)
	}

	sort.Strings(folders)
	return folders, nil
}

// RecentNotes returns notes sorted by modification time (most recent first).
func (v *ObsidianVault) RecentNotes(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}

	type noteEntry struct {
		path    string
		name    string
		modTime time.Time
		size    int64
	}
	var all []noteEntry

	err := filepath.WalkDir(v.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isSkippedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}

		rel, _ := filepath.Rel(v.path, path)
		all = append(all, noteEntry{
			path:    rel,
			name:    strings.TrimSuffix(d.Name(), ".md"),
			modTime: info.ModTime(),
			size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("obsidian: recent notes: %w", err)
	}

	// Sort most recent first
	sort.Slice(all, func(i, j int) bool {
		return all[i].modTime.After(all[j].modTime)
	})

	if len(all) > limit {
		all = all[:limit]
	}

	result := make([]map[string]interface{}, 0, len(all))
	for _, n := range all {
		result = append(result, map[string]interface{}{
			"path":     n.path,
			"title":    n.name,
			"modified": n.modTime.Format(time.RFC3339),
			"size":     n.size,
		})
	}
	return result, nil
}

// --- internal helpers ---

// parseFrontmatter extracts YAML frontmatter between --- markers as a raw key-value map.
// Returns nil if no frontmatter is found.
func parseFrontmatter(content string) map[string]string {
	if !strings.HasPrefix(content, "---") {
		return nil
	}

	lines := strings.SplitN(content, "\n", -1)
	if len(lines) < 3 {
		return nil
	}

	// Find closing ---
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx < 0 {
		return nil
	}

	fm := make(map[string]string)
	for _, line := range lines[1:endIdx] {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if key != "" {
				fm[key] = val
			}
		}
	}
	if len(fm) == 0 {
		return nil
	}
	return fm
}

// linkPattern matches [[wiki links]], optionally with alias [[target|alias]].
var linkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// extractLinks finds all [[wiki links]] in content.
func extractLinks(content string) []string {
	matches := linkPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var links []string
	for _, m := range matches {
		link := m[1]
		// Strip alias (everything after |)
		if idx := strings.Index(link, "|"); idx >= 0 {
			link = link[:idx]
		}
		if !seen[link] {
			seen[link] = true
			links = append(links, link)
		}
	}
	return links
}

// extractTags finds all #tags in content.
func extractTags(content string) []string {
	matches := tagPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := "#" + m[1]
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}

// quickTags reads first N lines of a file and extracts tags.
func quickTags(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var tags []string
	seen := make(map[string]bool)
	lineCount := 0

	for scanner.Scan() && lineCount < 50 {
		lineCount++
		for _, m := range tagPattern.FindAllStringSubmatch(scanner.Text(), -1) {
			tag := "#" + m[1]
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags
}
