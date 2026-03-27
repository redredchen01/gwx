package api

import (
	"context"
	"fmt"
	"sync"

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

	// Parallel fetch with semaphore to respect rate limits
	const concurrency = 10
	sem := make(chan struct{}, concurrency)
	summaries := make([]MessageSummary, len(resp.Messages))
	var wg sync.WaitGroup

	for i, msg := range resp.Messages {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				summaries[idx] = MessageSummary{
					ID:      id,
					Snippet: fmt.Sprintf("[error: %v]", ctx.Err()),
				}
				return
			}

			if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
				summaries[idx] = MessageSummary{
					ID:      id,
					Snippet: fmt.Sprintf("[error: rate limit: %v]", err),
				}
				return
			}
			detail, err := svc.Users.Messages.Get("me", id).
				Format("metadata").
				MetadataHeaders("From", "To", "Subject", "Date").
				Do()
			if err != nil {
				summaries[idx] = MessageSummary{
					ID:      id,
					Snippet: fmt.Sprintf("[error: %v]", err),
				}
				return
			}
			summaries[idx] = messageToSummary(detail)
		}(i, msg.Id)
	}
	wg.Wait()

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

	labels := make([]map[string]string, 0, len(resp.Labels))
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
