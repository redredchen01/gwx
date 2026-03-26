package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const slackBaseURL = "https://slack.com/api/"

// SlackClient provides access to the Slack Web API using a bot token.
type SlackClient struct {
	token string
	http  *http.Client
}

// NewSlackClient creates a Slack API client with the given bot token.
func NewSlackClient(token string) *SlackClient {
	return &SlackClient{
		token: token,
		http:  &http.Client{Transport: NewBaseTransport()},
	}
}

// ListChannels returns up to limit channels visible to the bot.
func (s *SlackClient) ListChannels(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	params := url.Values{
		"types":            {"public_channel,private_channel"},
		"exclude_archived": {"true"},
		"limit":            {fmt.Sprintf("%d", limit)},
	}
	resp, err := s.get(ctx, "conversations.list", params)
	if err != nil {
		return nil, err
	}
	channels, _ := toSlice(resp, "channels")
	return channels, nil
}

// SendMessage posts a text message to a channel.
func (s *SlackClient) SendMessage(ctx context.Context, channel, text string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"channel": channel,
		"text":    text,
	}
	resp, err := s.post(ctx, "chat.postMessage", body)
	if err != nil {
		return nil, err
	}
	msg, _ := resp["message"].(map[string]interface{})
	result := map[string]interface{}{
		"ok":      resp["ok"],
		"channel": resp["channel"],
		"ts":      resp["ts"],
	}
	if msg != nil {
		result["message"] = msg
	}
	return result, nil
}

// ListMessages returns recent messages from a channel.
func (s *SlackClient) ListMessages(ctx context.Context, channel string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{
		"channel": {channel},
		"limit":   {fmt.Sprintf("%d", limit)},
	}
	resp, err := s.get(ctx, "conversations.history", params)
	if err != nil {
		return nil, err
	}
	messages, _ := toSlice(resp, "messages")
	return messages, nil
}

// SearchMessages searches messages across the workspace.
// Requires the search:read scope on the bot token.
func (s *SlackClient) SearchMessages(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{
		"query": {query},
		"count": {fmt.Sprintf("%d", limit)},
	}
	resp, err := s.get(ctx, "search.messages", params)
	if err != nil {
		return nil, err
	}
	// search.messages returns { messages: { matches: [...] } }
	messagesObj, _ := resp["messages"].(map[string]interface{})
	if messagesObj == nil {
		return nil, nil
	}
	matches, _ := toSlice(messagesObj, "matches")
	return matches, nil
}

// ListUsers returns up to limit workspace members.
func (s *SlackClient) ListUsers(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	params := url.Values{
		"limit": {fmt.Sprintf("%d", limit)},
	}
	resp, err := s.get(ctx, "users.list", params)
	if err != nil {
		return nil, err
	}
	members, _ := toSlice(resp, "members")
	return members, nil
}

// GetUser returns profile information for a single user.
func (s *SlackClient) GetUser(ctx context.Context, userID string) (map[string]interface{}, error) {
	params := url.Values{
		"user": {userID},
	}
	resp, err := s.get(ctx, "users.info", params)
	if err != nil {
		return nil, err
	}
	user, _ := resp["user"].(map[string]interface{})
	if user == nil {
		return nil, fmt.Errorf("user %s not found in response", userID)
	}
	return user, nil
}

// --- internal helpers ---

func (s *SlackClient) get(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
	u := slackBaseURL + method
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("slack: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)

	return s.do(req)
}

func (s *SlackClient) post(ctx context.Context, method string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("slack: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackBaseURL+method, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("slack: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	return s.do(req)
}

func (s *SlackClient) do(req *http.Request) (map[string]interface{}, error) {
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("slack: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("slack: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("slack: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("slack: parse response: %w", err)
	}

	ok, _ := result["ok"].(bool)
	if !ok {
		errMsg, _ := result["error"].(string)
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return nil, fmt.Errorf("slack API error: %s", errMsg)
	}

	return result, nil
}

// toSlice extracts a []map[string]interface{} from a response field.
func toSlice(m map[string]interface{}, key string) ([]map[string]interface{}, error) {
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
