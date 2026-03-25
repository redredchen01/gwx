package api

import (
	"context"
	"fmt"

	"google.golang.org/api/gmail/v1"
)

func (gs *GmailService) service(ctx context.Context) (*gmail.Service, error) {
	svc, err := gs.client.GetOrCreateService("gmail:v1", func() (any, error) {
		opts, err := gs.client.ClientOptions(ctx, "gmail")
		if err != nil {
			return nil, err
		}
		return gmail.NewService(ctx, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}
	return svc.(*gmail.Service), nil
}

// ListMessages lists messages with optional filters.
func (gs *GmailService) ListMessages(ctx context.Context, query string, labelIDs []string, maxResults int64, unreadOnly bool) ([]MessageSummary, int64, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return nil, 0, err
	}

	svc, err := gs.service(ctx)
	if err != nil {
		return nil, 0, err
	}

	call := svc.Users.Messages.List("me")
	if query != "" {
		call = call.Q(query)
	}
	if unreadOnly {
		call = call.LabelIds("UNREAD")
	}
	for _, lid := range labelIDs {
		call = call.LabelIds(lid)
	}
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, 0, fmt.Errorf("list messages: %w", err)
	}

	var summaries []MessageSummary
	for _, msg := range resp.Messages {
		if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
			return summaries, resp.ResultSizeEstimate, err
		}
		detail, err := svc.Users.Messages.Get("me", msg.Id).
			Format("metadata").
			MetadataHeaders("From", "To", "Subject", "Date").
			Do()
		if err != nil {
			// Include a stub with just the ID so the caller knows it was skipped
			summaries = append(summaries, MessageSummary{
				ID:      msg.Id,
				Snippet: fmt.Sprintf("[error: %v]", err),
			})
			continue
		}
		summaries = append(summaries, messageToSummary(detail))
	}

	return summaries, resp.ResultSizeEstimate, nil
}

// GetMessage retrieves a single message by ID.
func (gs *GmailService) GetMessage(ctx context.Context, messageID string) (*MessageDetail, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return nil, err
	}

	svc, err := gs.service(ctx)
	if err != nil {
		return nil, err
	}

	msg, err := svc.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	detail := messageToDetail(msg)
	return &detail, nil
}

// SearchMessages searches messages using Gmail query syntax.
func (gs *GmailService) SearchMessages(ctx context.Context, query string, maxResults int64) ([]MessageSummary, int64, error) {
	return gs.ListMessages(ctx, query, nil, maxResults, false)
}

// ListLabels returns all Gmail labels.
func (gs *GmailService) ListLabels(ctx context.Context) ([]map[string]string, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return nil, err
	}

	svc, err := gs.service(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}

	var labels []map[string]string
	for _, l := range resp.Labels {
		labels = append(labels, map[string]string{
			"id":   l.Id,
			"name": l.Name,
			"type": l.Type,
		})
	}
	return labels, nil
}

func messageToSummary(msg *gmail.Message) MessageSummary {
	s := MessageSummary{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
		Labels:   msg.LabelIds,
	}

	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "From":
			s.From = h.Value
		case "To":
			s.To = h.Value
		case "Subject":
			s.Subject = h.Value
		case "Date":
			s.Date = h.Value
		}
	}

	for _, l := range msg.LabelIds {
		if l == "UNREAD" {
			s.Unread = true
			break
		}
	}

	return s
}

func messageToDetail(msg *gmail.Message) MessageDetail {
	d := MessageDetail{
		MessageSummary: messageToSummary(msg),
		Headers:        make(map[string]string),
	}

	if msg.Payload != nil {
		for _, h := range msg.Payload.Headers {
			d.Headers[h.Name] = h.Value
		}
		d.Body = extractBody(msg.Payload, "text/plain")
		d.BodyHTML = extractBody(msg.Payload, "text/html")
	}

	return d
}
