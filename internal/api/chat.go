package api

import (
	"context"
	"fmt"

	"google.golang.org/api/chat/v1"
)

// ChatService wraps Google Chat API operations.
type ChatService struct {
	client *Client
}

// NewChatService creates a Chat service wrapper.
func NewChatService(client *Client) *ChatService {
	return &ChatService{client: client}
}

func (cs *ChatService) service(ctx context.Context) (*chat.Service, error) {
	svc, err := cs.client.GetOrCreateService("chat:v1", func() (any, error) {
		opts, err := cs.client.ClientOptions(ctx, "chat")
		if err != nil {
			return nil, err
		}
		return chat.NewService(ctx, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("create chat service: %w", err)
	}
	return svc.(*chat.Service), nil
}

// SpaceSummary is a simplified space representation.
type SpaceSummary struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Threaded    bool   `json:"threaded"`
}

// ChatMessageResult is returned after sending a message.
type ChatMessageResult struct {
	Name       string `json:"name"`
	Text       string `json:"text"`
	CreateTime string `json:"create_time"`
	Space      string `json:"space"`
}

// ListSpaces lists Chat spaces the user is a member of.
func (cs *ChatService) ListSpaces(ctx context.Context, maxResults int) ([]SpaceSummary, error) {
	if err := cs.client.WaitRate(ctx, "chat"); err != nil {
		return nil, err
	}

	svc, err := cs.service(ctx)
	if err != nil {
		return nil, err
	}

	call := svc.Spaces.List()
	if maxResults > 0 {
		call = call.PageSize(int64(maxResults))
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list spaces: %w", err)
	}

	var spaces []SpaceSummary
	for _, s := range resp.Spaces {
		spaces = append(spaces, SpaceSummary{
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Type:        s.Type,
			Threaded:    s.Threaded,
		})
	}
	return spaces, nil
}

// SendMessage sends a text message to a Chat space.
func (cs *ChatService) SendMessage(ctx context.Context, spaceName string, text string) (*ChatMessageResult, error) {
	if err := cs.client.WaitRate(ctx, "chat"); err != nil {
		return nil, err
	}

	svc, err := cs.service(ctx)
	if err != nil {
		return nil, err
	}

	msg := &chat.Message{Text: text}

	sent, err := svc.Spaces.Messages.Create(spaceName, msg).Do()
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	return &ChatMessageResult{
		Name:       sent.Name,
		Text:       sent.Text,
		CreateTime: sent.CreateTime,
		Space:      sent.Space.Name,
	}, nil
}

// ListMessages lists recent messages in a Chat space.
func (cs *ChatService) ListMessages(ctx context.Context, spaceName string, maxResults int) ([]ChatMessageResult, error) {
	if err := cs.client.WaitRate(ctx, "chat"); err != nil {
		return nil, err
	}

	svc, err := cs.service(ctx)
	if err != nil {
		return nil, err
	}

	call := svc.Spaces.Messages.List(spaceName)
	if maxResults > 0 {
		call = call.PageSize(int64(maxResults))
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	var messages []ChatMessageResult
	for _, m := range resp.Messages {
		spaceName := ""
		if m.Space != nil {
			spaceName = m.Space.Name
		}
		messages = append(messages, ChatMessageResult{
			Name:       m.Name,
			Text:       m.Text,
			CreateTime: m.CreateTime,
			Space:      spaceName,
		})
	}
	return messages, nil
}
