package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".obsidian"), 0755)
	// Create some test notes
	os.WriteFile(filepath.Join(dir, "note1.md"), []byte("---\ntitle: Test\ntags: [go, cli]\n---\n# Hello\nSome content with #tag1"), 0644)
	os.MkdirAll(filepath.Join(dir, "folder1"), 0755)
	os.WriteFile(filepath.Join(dir, "folder1", "note2.md"), []byte("# Second\nContent with [[note1]] link"), 0644)
	return dir
}

// ---------------------------------------------------------------------------
// NewObsidianVault
// ---------------------------------------------------------------------------

func TestNewObsidianVault_ValidPath(t *testing.T) {
	dir := setupTestVault(t)
	v, err := NewObsidianVault(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if v == nil {
		t.Fatal("expected non-nil vault")
	}
}

func TestNewObsidianVault_InvalidPath_NoObsidianDir(t *testing.T) {
	dir := t.TempDir() // no .obsidian/ inside
	_, err := NewObsidianVault(dir)
	if err == nil {
		t.Fatal("expected error for missing .obsidian/")
	}
	if !strings.Contains(err.Error(), "not a valid vault") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestNewObsidianVault_NonexistentPath(t *testing.T) {
	_, err := NewObsidianVault("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestNewObsidianVault_EmptyPath(t *testing.T) {
	_, err := NewObsidianVault("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestNewObsidianVault_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "notadir")
	os.WriteFile(f, []byte("hi"), 0644)

	_, err := NewObsidianVault(f)
	if err == nil {
		t.Fatal("expected error for file path")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListNotes
// ---------------------------------------------------------------------------

func TestListNotes_ListsFiles(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	notes, err := v.ListNotes("", 0)
	if err != nil {
		t.Fatalf("ListNotes error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}

	// Check that both notes are present
	paths := make(map[string]bool)
	for _, n := range notes {
		paths[n["path"].(string)] = true
	}
	if !paths["note1.md"] {
		t.Error("expected note1.md in results")
	}
	if !paths[filepath.Join("folder1", "note2.md")] {
		t.Errorf("expected folder1/note2.md in results")
	}
}

func TestListNotes_FolderFilter(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	notes, err := v.ListNotes("folder1", 0)
	if err != nil {
		t.Fatalf("ListNotes with folder filter error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note in folder1, got %d", len(notes))
	}
	if notes[0]["title"] != "note2" {
		t.Errorf("expected title 'note2', got %v", notes[0]["title"])
	}
}

func TestListNotes_RespectsLimit(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	notes, err := v.ListNotes("", 1)
	if err != nil {
		t.Fatalf("ListNotes with limit error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note with limit=1, got %d", len(notes))
	}
}

func TestListNotes_SkipsObsidianAndTrash(t *testing.T) {
	dir := setupTestVault(t)
	// Create files inside .obsidian/ and .trash/
	os.WriteFile(filepath.Join(dir, ".obsidian", "hidden.md"), []byte("# Hidden"), 0644)
	os.MkdirAll(filepath.Join(dir, ".trash"), 0755)
	os.WriteFile(filepath.Join(dir, ".trash", "deleted.md"), []byte("# Deleted"), 0644)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "gitfile.md"), []byte("# Git"), 0644)

	v, _ := NewObsidianVault(dir)
	notes, err := v.ListNotes("", 100)
	if err != nil {
		t.Fatalf("ListNotes error: %v", err)
	}

	for _, n := range notes {
		p := n["path"].(string)
		if strings.HasPrefix(p, ".obsidian") || strings.HasPrefix(p, ".trash") || strings.HasPrefix(p, ".git") {
			t.Errorf("should not include hidden/skipped file: %s", p)
		}
	}
	// Only the 2 original notes
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes (skipping hidden dirs), got %d", len(notes))
	}
}

func TestListNotes_DefaultLimit(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	// limit=0 → defaults to 20
	notes, err := v.ListNotes("", 0)
	if err != nil {
		t.Fatalf("ListNotes error: %v", err)
	}
	// We only have 2 notes, so result should be 2 (not 20)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

// ---------------------------------------------------------------------------
// SearchNotes
// ---------------------------------------------------------------------------

func TestSearchNotes_FindsMatchingContent(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	results, err := v.SearchNotes("Hello", 0)
	if err != nil {
		t.Fatalf("SearchNotes error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'Hello', got %d", len(results))
	}
	if results[0]["title"] != "note1" {
		t.Errorf("expected note1, got %v", results[0]["title"])
	}
	matches := results[0]["matches"].([]string)
	if len(matches) == 0 {
		t.Error("expected at least one match line")
	}
}

func TestSearchNotes_CaseInsensitive(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	results, err := v.SearchNotes("hello", 0) // lowercase
	if err != nil {
		t.Fatalf("SearchNotes error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for case-insensitive 'hello', got %d", len(results))
	}
}

func TestSearchNotes_Limit(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	// "content" appears in both files
	results, err := v.SearchNotes("content", 1)
	if err != nil {
		t.Fatalf("SearchNotes error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with limit=1, got %d", len(results))
	}
}

func TestSearchNotes_EmptyQuery(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.SearchNotes("", 0)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSearchNotes_NoMatch(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	results, err := v.SearchNotes("xyznonexistent", 0)
	if err != nil {
		t.Fatalf("SearchNotes error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// ReadNote
// ---------------------------------------------------------------------------

func TestReadNote_ReadsContent(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	result, err := v.ReadNote("note1")
	if err != nil {
		t.Fatalf("ReadNote error: %v", err)
	}
	content := result["content"].(string)
	if !strings.Contains(content, "# Hello") {
		t.Error("expected content to contain '# Hello'")
	}
	if result["title"] != "note1" {
		t.Errorf("expected title 'note1', got %v", result["title"])
	}
}

func TestReadNote_ParsesFrontmatter(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	result, err := v.ReadNote("note1.md")
	if err != nil {
		t.Fatalf("ReadNote error: %v", err)
	}
	fm, ok := result["frontmatter"]
	if !ok {
		t.Fatal("expected frontmatter to be present")
	}
	fmMap := fm.(map[string]string)
	if fmMap["title"] != "Test" {
		t.Errorf("expected frontmatter title 'Test', got %v", fmMap["title"])
	}
}

func TestReadNote_ExtractsTagsAndLinks(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	// note1 has #tag1
	r1, _ := v.ReadNote("note1")
	tags, ok := r1["tags"]
	if !ok {
		t.Fatal("expected tags in note1")
	}
	tagList := tags.([]string)
	found := false
	for _, tag := range tagList {
		if tag == "#tag1" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected #tag1 in tags, got %v", tagList)
	}

	// note2 has [[note1]] link
	r2, _ := v.ReadNote("folder1/note2")
	links, ok := r2["links"]
	if !ok {
		t.Fatal("expected links in note2")
	}
	linkList := links.([]string)
	if len(linkList) == 0 || linkList[0] != "note1" {
		t.Errorf("expected link to 'note1', got %v", linkList)
	}
}

func TestReadNote_NotFound(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.ReadNote("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent note")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadNote_AutoAddsExtension(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	// Should work without .md extension
	r1, err := v.ReadNote("note1")
	if err != nil {
		t.Fatalf("ReadNote without .md error: %v", err)
	}
	// Should also work with .md extension
	r2, err := v.ReadNote("note1.md")
	if err != nil {
		t.Fatalf("ReadNote with .md error: %v", err)
	}
	if r1["content"] != r2["content"] {
		t.Error("expected same content with and without .md")
	}
}

// ---------------------------------------------------------------------------
// CreateNote
// ---------------------------------------------------------------------------

func TestCreateNote_CreatesFile(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	result, err := v.CreateNote("newnote", "# New Note\nContent here")
	if err != nil {
		t.Fatalf("CreateNote error: %v", err)
	}
	if result["created"] != true {
		t.Error("expected created=true")
	}
	if result["path"] != "newnote.md" {
		t.Errorf("expected path 'newnote.md', got %v", result["path"])
	}

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(dir, "newnote.md"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(data) != "# New Note\nContent here" {
		t.Errorf("unexpected file content: %s", data)
	}
}

func TestCreateNote_AutoAddsMdExtension(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	result, err := v.CreateNote("autoext", "content")
	if err != nil {
		t.Fatalf("CreateNote error: %v", err)
	}
	if result["path"] != "autoext.md" {
		t.Errorf("expected path 'autoext.md', got %v", result["path"])
	}

	// With .md should also work
	result2, err := v.CreateNote("explicit.md", "content2")
	if err != nil {
		t.Fatalf("CreateNote error: %v", err)
	}
	if result2["path"] != "explicit.md" {
		t.Errorf("expected path 'explicit.md', got %v", result2["path"])
	}
}

func TestCreateNote_AlreadyExists(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.CreateNote("note1", "duplicate")
	if err == nil {
		t.Fatal("expected error for already existing note")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateNote_CreatesParentDir(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	result, err := v.CreateNote("subfolder/deep/newnote", "content")
	if err != nil {
		t.Fatalf("CreateNote with nested dir error: %v", err)
	}
	expected := filepath.Join("subfolder", "deep", "newnote.md")
	if result["path"] != expected {
		t.Errorf("expected path %q, got %v", expected, result["path"])
	}
}

func TestCreateNote_PathTraversal(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.CreateNote("../escape", "evil")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AppendNote
// ---------------------------------------------------------------------------

func TestAppendNote_AppendsText(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	result, err := v.AppendNote("note1", "Appended text")
	if err != nil {
		t.Fatalf("AppendNote error: %v", err)
	}
	if result["appended"] != true {
		t.Error("expected appended=true")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "note1.md"))
	if !strings.HasSuffix(string(data), "\nAppended text") {
		t.Errorf("expected appended text at end, got: %s", data)
	}
}

func TestAppendNote_NoteMustExist(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.AppendNote("nonexistent", "text")
	if err == nil {
		t.Fatal("expected error for nonexistent note")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppendNote_AutoAddsMdExtension(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	// Should work without .md
	_, err := v.AppendNote("note1", "extra")
	if err != nil {
		t.Fatalf("AppendNote without .md error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// DailyNote
// ---------------------------------------------------------------------------

func TestDailyNote_CreatesNewNote(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	result, err := v.DailyNote("Today's entry")
	if err != nil {
		t.Fatalf("DailyNote error: %v", err)
	}
	if result["action"] != "created" {
		t.Errorf("expected action 'created', got %v", result["action"])
	}

	today := time.Now().Format("2006-01-02")
	if result["date"] != today {
		t.Errorf("expected date %s, got %v", today, result["date"])
	}

	// Verify file content
	data, err := os.ReadFile(filepath.Join(dir, today+".md"))
	if err != nil {
		t.Fatalf("daily note file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# "+today) {
		t.Errorf("expected header with date, got: %s", content)
	}
	if !strings.Contains(content, "Today's entry") {
		t.Errorf("expected content 'Today's entry', got: %s", content)
	}
}

func TestDailyNote_AppendsToExisting(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	// Create first
	_, err := v.DailyNote("First entry")
	if err != nil {
		t.Fatalf("DailyNote create error: %v", err)
	}

	// Append
	result, err := v.DailyNote("Second entry")
	if err != nil {
		t.Fatalf("DailyNote append error: %v", err)
	}
	if result["action"] != "appended" {
		t.Errorf("expected action 'appended', got %v", result["action"])
	}

	today := time.Now().Format("2006-01-02")
	data, _ := os.ReadFile(filepath.Join(dir, today+".md"))
	content := string(data)
	if !strings.Contains(content, "First entry") || !strings.Contains(content, "Second entry") {
		t.Errorf("expected both entries, got: %s", content)
	}
}

func TestDailyNote_UsesConfiguredFolder(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)
	v.SetDailyFolder("daily")

	result, err := v.DailyNote("In subfolder")
	if err != nil {
		t.Fatalf("DailyNote with folder error: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	expected := filepath.Join("daily", today+".md")
	if result["path"] != expected {
		t.Errorf("expected path %q, got %v", expected, result["path"])
	}

	// Verify directory was created
	_, err = os.Stat(filepath.Join(dir, "daily"))
	if err != nil {
		t.Errorf("expected daily/ directory to be created: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListTags
// ---------------------------------------------------------------------------

func TestListTags_FindsTagsWithCounts(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	tags, err := v.ListTags()
	if err != nil {
		t.Fatalf("ListTags error: %v", err)
	}
	if len(tags) == 0 {
		t.Fatal("expected at least one tag")
	}

	// #tag1 should be found in note1.md
	foundTag1 := false
	for _, tag := range tags {
		if tag["tag"] == "#tag1" {
			foundTag1 = true
			count := tag["count"].(int)
			if count < 1 {
				t.Errorf("expected count >= 1 for #tag1, got %d", count)
			}
		}
	}
	if !foundTag1 {
		t.Error("expected to find #tag1")
	}
}

func TestListTags_MultipleFilesCountsAggregate(t *testing.T) {
	dir := setupTestVault(t)
	// Add another file with #tag1
	os.WriteFile(filepath.Join(dir, "note3.md"), []byte("Content with #tag1 again"), 0644)

	v, _ := NewObsidianVault(dir)
	tags, err := v.ListTags()
	if err != nil {
		t.Fatalf("ListTags error: %v", err)
	}

	for _, tag := range tags {
		if tag["tag"] == "#tag1" {
			count := tag["count"].(int)
			if count < 2 {
				t.Errorf("expected count >= 2 for #tag1 across files, got %d", count)
			}
			return
		}
	}
	t.Error("expected to find #tag1")
}

// ---------------------------------------------------------------------------
// SearchByTag
// ---------------------------------------------------------------------------

func TestSearchByTag_FindsNotes(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	results, err := v.SearchByTag("tag1", 0)
	if err != nil {
		t.Fatalf("SearchByTag error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 note with #tag1, got %d", len(results))
	}
	if results[0]["title"] != "note1" {
		t.Errorf("expected note1, got %v", results[0]["title"])
	}
}

func TestSearchByTag_NormalizesHashPrefix(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	// Both with and without # should work
	r1, _ := v.SearchByTag("#tag1", 0)
	r2, _ := v.SearchByTag("tag1", 0)

	if len(r1) != len(r2) {
		t.Errorf("expected same results with/without # prefix: %d vs %d", len(r1), len(r2))
	}
}

func TestSearchByTag_EmptyTag(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.SearchByTag("", 0)
	if err == nil {
		t.Fatal("expected error for empty tag")
	}
}

func TestSearchByTag_Limit(t *testing.T) {
	dir := setupTestVault(t)
	// Add more notes with same tag
	os.WriteFile(filepath.Join(dir, "note3.md"), []byte("has #tag1"), 0644)
	os.WriteFile(filepath.Join(dir, "note4.md"), []byte("also #tag1"), 0644)

	v, _ := NewObsidianVault(dir)
	results, err := v.SearchByTag("tag1", 1)
	if err != nil {
		t.Fatalf("SearchByTag error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with limit=1, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// ListFolders
// ---------------------------------------------------------------------------

func TestListFolders_ListsDirectories(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	folders, err := v.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders error: %v", err)
	}

	// Should include folder1 but not .obsidian, .trash, .git
	found := false
	for _, f := range folders {
		if f == "folder1" {
			found = true
		}
		if strings.HasPrefix(f, ".") {
			t.Errorf("should not include hidden folder: %s", f)
		}
	}
	if !found {
		t.Error("expected folder1 in results")
	}
}

func TestListFolders_ExcludesHiddenDirs(t *testing.T) {
	dir := setupTestVault(t)
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0755)
	os.MkdirAll(filepath.Join(dir, ".trash"), 0755)

	v, _ := NewObsidianVault(dir)
	folders, err := v.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders error: %v", err)
	}

	for _, f := range folders {
		if strings.HasPrefix(f, ".") {
			t.Errorf("should not include hidden folder: %s", f)
		}
	}
}

func TestListFolders_Sorted(t *testing.T) {
	dir := setupTestVault(t)
	os.MkdirAll(filepath.Join(dir, "zebra"), 0755)
	os.MkdirAll(filepath.Join(dir, "alpha"), 0755)

	v, _ := NewObsidianVault(dir)
	folders, err := v.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders error: %v", err)
	}

	for i := 1; i < len(folders); i++ {
		if folders[i] < folders[i-1] {
			t.Errorf("folders not sorted: %v comes before %v", folders[i-1], folders[i])
		}
	}
}

// ---------------------------------------------------------------------------
// RecentNotes
// ---------------------------------------------------------------------------

func TestRecentNotes_SortedByModTime(t *testing.T) {
	dir := setupTestVault(t)
	// Create a newer file by touching it after a brief pause
	newerPath := filepath.Join(dir, "newer.md")
	os.WriteFile(newerPath, []byte("# Newer"), 0644)
	// Set note1.md to an older time
	oldTime := time.Now().Add(-1 * time.Hour)
	os.Chtimes(filepath.Join(dir, "note1.md"), oldTime, oldTime)

	v, _ := NewObsidianVault(dir)
	notes, err := v.RecentNotes(0)
	if err != nil {
		t.Fatalf("RecentNotes error: %v", err)
	}
	if len(notes) < 2 {
		t.Fatalf("expected at least 2 notes, got %d", len(notes))
	}

	// Most recent should be first
	firstModStr := notes[0]["modified"].(string)
	secondModStr := notes[1]["modified"].(string)
	firstMod, _ := time.Parse(time.RFC3339, firstModStr)
	secondMod, _ := time.Parse(time.RFC3339, secondModStr)

	if firstMod.Before(secondMod) {
		t.Errorf("expected first note to be more recent than second: %v vs %v", firstMod, secondMod)
	}
}

func TestRecentNotes_RespectsLimit(t *testing.T) {
	dir := setupTestVault(t)
	os.WriteFile(filepath.Join(dir, "note3.md"), []byte("# Three"), 0644)
	os.WriteFile(filepath.Join(dir, "note4.md"), []byte("# Four"), 0644)

	v, _ := NewObsidianVault(dir)
	notes, err := v.RecentNotes(2)
	if err != nil {
		t.Fatalf("RecentNotes error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes with limit=2, got %d", len(notes))
	}
}

func TestRecentNotes_DefaultLimit(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	// limit=0 → defaults to 10; we only have 2 files
	notes, err := v.RecentNotes(0)
	if err != nil {
		t.Fatalf("RecentNotes error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

// ---------------------------------------------------------------------------
// Path traversal
// ---------------------------------------------------------------------------

func TestPathTraversal_Blocked(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	cases := []struct {
		name string
		path string
	}{
		{"dotdot", "../escape"},
		{"nested dotdot", "foo/../../escape"},
		{"absolute escape", "foo/../../../etc/passwd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := v.ReadNote(tc.path)
			if err == nil {
				t.Errorf("expected error for path traversal with %q", tc.path)
			}
			if !strings.Contains(err.Error(), "traversal") {
				t.Errorf("expected traversal error, got: %v", err)
			}
		})
	}
}

func TestPathTraversal_CreateNote(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.CreateNote("../../etc/evil", "pwned")
	if err == nil {
		t.Fatal("expected error for path traversal in CreateNote")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPathTraversal_AppendNote(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.AppendNote("../../etc/passwd", "evil")
	if err == nil {
		t.Fatal("expected error for path traversal in AppendNote")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPathTraversal_ListNotes(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.ListNotes("../../../etc", 10)
	if err == nil {
		t.Fatal("expected error for path traversal in ListNotes")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func TestParseFrontmatter(t *testing.T) {
	content := "---\ntitle: Hello\ntags: [a, b]\n---\n# Content"
	fm := parseFrontmatter(content)
	if fm == nil {
		t.Fatal("expected frontmatter")
	}
	if fm["title"] != "Hello" {
		t.Errorf("expected title 'Hello', got %v", fm["title"])
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	fm := parseFrontmatter("# Just a heading\nSome content")
	if fm != nil {
		t.Errorf("expected nil for no frontmatter, got %v", fm)
	}
}

func TestExtractLinks(t *testing.T) {
	content := "Link to [[page1]] and [[page2|alias]] and [[page1]] again"
	links := extractLinks(content)

	if len(links) != 2 {
		t.Fatalf("expected 2 unique links, got %d: %v", len(links), links)
	}
	if links[0] != "page1" {
		t.Errorf("expected first link 'page1', got %v", links[0])
	}
	if links[1] != "page2" {
		t.Errorf("expected second link 'page2' (alias stripped), got %v", links[1])
	}
}

func TestExtractTags(t *testing.T) {
	content := "Some text with #tag1 and #tag2 and #tag1 again\nAlso #nested/tag"
	tags := extractTags(content)

	if len(tags) != 3 {
		t.Fatalf("expected 3 unique tags, got %d: %v", len(tags), tags)
	}

	expected := map[string]bool{"#tag1": true, "#tag2": true, "#nested/tag": true}
	for _, tag := range tags {
		if !expected[tag] {
			t.Errorf("unexpected tag: %s", tag)
		}
	}
}

func TestSafePath_EmptyPath(t *testing.T) {
	dir := setupTestVault(t)
	v, _ := NewObsidianVault(dir)

	_, err := v.safePath("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestIsSkippedDir(t *testing.T) {
	if !isSkippedDir(".obsidian") {
		t.Error(".obsidian should be skipped")
	}
	if !isSkippedDir(".trash") {
		t.Error(".trash should be skipped")
	}
	if !isSkippedDir(".git") {
		t.Error(".git should be skipped")
	}
	if isSkippedDir("folder1") {
		t.Error("folder1 should not be skipped")
	}
}
