package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	notionBaseURL = "https://api.notion.com/v1"
	notionVersion = "2022-06-28"
)

// NotionClient provides access to the Notion API using an integration token.
type NotionClient struct {
	token string
	http  *http.Client
}

// NewNotionClient creates a Notion API client with the given integration token.
func NewNotionClient(token string) *NotionClient {
	return &NotionClient{
		token: token,
		http:  &http.Client{},
	}
}

// SearchPages searches for pages and databases by title.
func (n *NotionClient) SearchPages(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	body := map[string]interface{}{
		"page_size": limit,
	}
	if query != "" {
		body["query"] = query
	}

	resp, err := n.postJSON(ctx, "/search", body)
	if err != nil {
		return nil, err
	}

	results, _ := notionToSlice(resp, "results")
	return results, nil
}

// GetPage retrieves a single page by ID.
func (n *NotionClient) GetPage(ctx context.Context, pageID string) (map[string]interface{}, error) {
	return n.getJSON(ctx, "/pages/"+pageID)
}

// CreatePage creates a new page under a parent page or database.
func (n *NotionClient) CreatePage(ctx context.Context, parentID, title string, props map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"parent": map[string]interface{}{
			"type":        "database_id",
			"database_id": parentID,
		},
		"properties": buildTitleProperty(title, props),
	}

	return n.postJSON(ctx, "/pages", body)
}

// QueryDatabase queries a database with optional filter.
func (n *NotionClient) QueryDatabase(ctx context.Context, databaseID string, filter map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	body := map[string]interface{}{
		"page_size": limit,
	}
	if filter != nil && len(filter) > 0 {
		body["filter"] = filter
	}

	resp, err := n.postJSON(ctx, "/databases/"+databaseID+"/query", body)
	if err != nil {
		return nil, err
	}

	results, _ := notionToSlice(resp, "results")
	return results, nil
}

// ListDatabases returns databases visible to the integration via the search endpoint.
func (n *NotionClient) ListDatabases(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"value":    "database",
			"property": "object",
		},
		"page_size": limit,
	}

	resp, err := n.postJSON(ctx, "/search", body)
	if err != nil {
		return nil, err
	}

	results, _ := notionToSlice(resp, "results")
	return results, nil
}

// GetDatabase retrieves a single database by ID.
func (n *NotionClient) GetDatabase(ctx context.Context, databaseID string) (map[string]interface{}, error) {
	return n.getJSON(ctx, "/databases/"+databaseID)
}

// --- internal helpers ---

func buildTitleProperty(title string, extra map[string]interface{}) map[string]interface{} {
	props := map[string]interface{}{
		"title": map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": title,
					},
				},
			},
		},
	}
	// Merge any extra properties from the caller.
	for k, v := range extra {
		if k != "title" {
			props[k] = v
		}
	}
	return props
}

func (n *NotionClient) getJSON(ctx context.Context, path string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, notionBaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("notion: create request: %w", err)
	}
	n.setHeaders(req)

	return n.doRequest(req)
}

func (n *NotionClient) postJSON(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("notion: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, notionBaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("notion: create request: %w", err)
	}
	n.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	return n.doRequest(req)
}

func (n *NotionClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Notion-Version", notionVersion)
}

func (n *NotionClient) doRequest(req *http.Request) (map[string]interface{}, error) {
	resp, err := n.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notion: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("notion: read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("notion: parse response: %w", err)
	}

	// Notion returns 200 for success, 4xx/5xx for errors.
	if resp.StatusCode >= 400 {
		errMsg, _ := result["message"].(string)
		code, _ := result["code"].(string)
		if errMsg == "" {
			errMsg = string(raw)
		}
		return nil, fmt.Errorf("notion API error (%s): %s", code, errMsg)
	}

	return result, nil
}

// notionToSlice extracts a []map[string]interface{} from a response field.
func notionToSlice(m map[string]interface{}, key string) ([]map[string]interface{}, error) {
	raw, ok := m[key]
	if !ok {
		return nil, nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, nil
	}
	result := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			result = append(result, obj)
		}
	}
	return result, nil
}
