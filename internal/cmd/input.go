package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// readJSONInput reads JSON from --json flag, stdin pipe, or interactive paste.
// Returns (data, source, error). source is "flag", "pipe", or "paste".
func readJSONInput(jsonFlag string, isPipe bool) ([]byte, string, error) {
	// 1. --json flag (highest priority)
	if jsonFlag != "" {
		data := []byte(jsonFlag)
		if !json.Valid(data) {
			return nil, "", fmt.Errorf("--json: invalid JSON")
		}
		return data, "flag", nil
	}

	// 2. stdin pipe (non-TTY)
	if isPipe {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, "", fmt.Errorf("read stdin: %w", err)
		}
		data = bytes.TrimSpace(data)
		if len(data) == 0 {
			return nil, "", fmt.Errorf("stdin pipe is empty")
		}
		if !json.Valid(data) {
			return nil, "", fmt.Errorf("stdin: invalid JSON")
		}
		return data, "pipe", nil
	}

	// 3. Interactive paste (TTY)
	fmt.Fprint(os.Stderr, "Paste token JSON (starts with '{'):\n> ")
	reader := bufio.NewReader(os.Stdin)
	firstLine, err := reader.ReadString('\n')
	if err != nil && firstLine == "" {
		return nil, "", fmt.Errorf("read input: %w", err)
	}
	firstLine = strings.TrimSpace(firstLine)
	if !strings.HasPrefix(firstLine, "{") {
		return nil, "", fmt.Errorf("expected JSON starting with '{', got: %s", firstLine)
	}
	data, err := readPastedJSON(firstLine, reader)
	if err != nil {
		return nil, "", fmt.Errorf("read pasted JSON: %w", err)
	}
	return data, "paste", nil
}
