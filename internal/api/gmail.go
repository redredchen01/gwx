package api

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// GmailService wraps Gmail API operations.
type GmailService struct {
	client *Client
}

// NewGmailService creates a Gmail service wrapper.
func NewGmailService(client *Client) *GmailService {
	return &GmailService{client: client}
}

// MessageSummary is a simplified message representation for agent consumption.
type MessageSummary struct {
	ID      string   `json:"id"`
	ThreadID string  `json:"thread_id"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	Subject string   `json:"subject"`
	Date    string   `json:"date"`
	Snippet string   `json:"snippet"`
	Labels  []string `json:"labels"`
	Unread  bool     `json:"unread"`
}

// MessageDetail is a full message representation.
type MessageDetail struct {
	MessageSummary
	Body     string            `json:"body"`
	BodyHTML string            `json:"body_html,omitempty"`
	Headers  map[string]string `json:"headers"`
}

// ListMessages lists messages with optional filters.
func (gs *GmailService) ListMessages(ctx context.Context, query string, labelIDs []string, maxResults int64, unreadOnly bool) ([]MessageSummary, int64, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return nil, 0, err
	}

	opts, err := gs.client.ClientOptions(ctx, "gmail")
	if err != nil {
		return nil, 0, err
	}

	svc, err := gmail.NewService(ctx, opts...)
	if err != nil {
		return nil, 0, fmt.Errorf("create gmail service: %w", err)
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

	opts, err := gs.client.ClientOptions(ctx, "gmail")
	if err != nil {
		return nil, err
	}

	svc, err := gmail.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
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

func extractBody(payload *gmail.MessagePart, mimeType string) string {
	if payload.MimeType == mimeType && payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err != nil {
			// Try with padding fix — Gmail sometimes omits padding
			decoded, err = base64.RawURLEncoding.DecodeString(payload.Body.Data)
			if err != nil {
				return payload.Body.Data // fallback to raw
			}
		}
		return string(decoded)
	}
	for _, part := range payload.Parts {
		if body := extractBody(part, mimeType); body != "" {
			return body
		}
	}
	return ""
}

// SendInput holds parameters for sending an email.
type SendInput struct {
	To          []string `json:"to"`
	CC          []string `json:"cc,omitempty"`
	BCC         []string `json:"bcc,omitempty"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	BodyHTML    string   `json:"body_html,omitempty"`
	Attachments []string `json:"attachments,omitempty"` // file paths
	ReplyTo     string   `json:"reply_to,omitempty"`     // message ID to reply to
	ReplyAll    bool     `json:"reply_all,omitempty"`
	ThreadID    string   `json:"thread_id,omitempty"`
}

// SendResult is returned after sending or drafting.
type SendResult struct {
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id"`
	LabelIDs  []string `json:"label_ids,omitempty"`
}

// SendMessage sends an email.
func (gs *GmailService) SendMessage(ctx context.Context, input *SendInput) (*SendResult, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return nil, err
	}

	opts, err := gs.client.ClientOptions(ctx, "gmail")
	if err != nil {
		return nil, err
	}

	svc, err := gmail.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}

	raw, err := buildRawMessage(input)
	if err != nil {
		return nil, fmt.Errorf("build message: %w", err)
	}

	msg := &gmail.Message{
		Raw:      raw,
		ThreadId: input.ThreadID,
	}

	sent, err := svc.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	return &SendResult{
		MessageID: sent.Id,
		ThreadID:  sent.ThreadId,
		LabelIDs:  sent.LabelIds,
	}, nil
}

// CreateDraft creates a draft email.
func (gs *GmailService) CreateDraft(ctx context.Context, input *SendInput) (*SendResult, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return nil, err
	}

	opts, err := gs.client.ClientOptions(ctx, "gmail")
	if err != nil {
		return nil, err
	}

	svc, err := gmail.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}

	raw, err := buildRawMessage(input)
	if err != nil {
		return nil, fmt.Errorf("build message: %w", err)
	}

	draft := &gmail.Draft{
		Message: &gmail.Message{
			Raw:      raw,
			ThreadId: input.ThreadID,
		},
	}

	created, err := svc.Users.Drafts.Create("me", draft).Do()
	if err != nil {
		return nil, fmt.Errorf("create draft: %w", err)
	}

	return &SendResult{
		MessageID: created.Message.Id,
		ThreadID:  created.Message.ThreadId,
	}, nil
}

// ReplyMessage replies to a message. Fetches original headers for In-Reply-To/References.
func (gs *GmailService) ReplyMessage(ctx context.Context, messageID string, input *SendInput) (*SendResult, error) {
	// Fetch original message for threading headers
	original, err := gs.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("fetch original message: %w", err)
	}

	// Set threading
	input.ThreadID = original.ThreadID
	input.ReplyTo = messageID

	// Set In-Reply-To and References from original
	if input.Subject == "" {
		subj := original.Subject
		if !strings.HasPrefix(strings.ToLower(subj), "re:") {
			subj = "Re: " + subj
		}
		input.Subject = subj
	}

	// If reply-all, populate To/CC from original
	if input.ReplyAll && len(input.To) == 0 {
		if original.Headers["Reply-To"] != "" {
			input.To = []string{original.Headers["Reply-To"]}
		} else {
			input.To = []string{original.From}
		}
		if cc := original.Headers["Cc"]; cc != "" {
			input.CC = append(input.CC, cc)
		}
	} else if len(input.To) == 0 {
		if original.Headers["Reply-To"] != "" {
			input.To = []string{original.Headers["Reply-To"]}
		} else {
			input.To = []string{original.From}
		}
	}

	return gs.SendMessage(ctx, input)
}

// buildRawMessage constructs a base64url-encoded RFC 822 message.
func buildRawMessage(input *SendInput) (string, error) {
	hasAttachments := len(input.Attachments) > 0
	hasHTML := input.BodyHTML != ""

	var buf bytes.Buffer

	if hasAttachments {
		return buildMultipartMessage(input)
	}

	// Simple message (no attachments)
	writeHeader(&buf, "To", strings.Join(input.To, ", "))
	if len(input.CC) > 0 {
		writeHeader(&buf, "Cc", strings.Join(input.CC, ", "))
	}
	if len(input.BCC) > 0 {
		writeHeader(&buf, "Bcc", strings.Join(input.BCC, ", "))
	}
	writeHeader(&buf, "Subject", input.Subject)

	if hasHTML {
		// Multipart alternative for text + HTML
		boundary := generateBoundary()
		writeHeader(&buf, "MIME-Version", "1.0")
		writeHeader(&buf, "Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", boundary))
		buf.WriteString("\r\n")

		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		buf.WriteString(input.Body)
		buf.WriteString("\r\n")

		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		buf.WriteString(input.BodyHTML)
		buf.WriteString("\r\n")

		buf.WriteString("--" + boundary + "--\r\n")
	} else {
		writeHeader(&buf, "MIME-Version", "1.0")
		writeHeader(&buf, "Content-Type", "text/plain; charset=utf-8")
		buf.WriteString("\r\n")
		buf.WriteString(input.Body)
	}

	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

// buildMultipartMessage builds a MIME multipart message with attachments.
func buildMultipartMessage(input *SendInput) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Write headers into a preamble buffer
	var preamble bytes.Buffer
	writeHeader(&preamble, "To", strings.Join(input.To, ", "))
	if len(input.CC) > 0 {
		writeHeader(&preamble, "Cc", strings.Join(input.CC, ", "))
	}
	if len(input.BCC) > 0 {
		writeHeader(&preamble, "Bcc", strings.Join(input.BCC, ", "))
	}
	writeHeader(&preamble, "Subject", input.Subject)
	writeHeader(&preamble, "MIME-Version", "1.0")
	writeHeader(&preamble, "Content-Type", fmt.Sprintf("multipart/mixed; boundary=%q", writer.Boundary()))
	preamble.WriteString("\r\n")

	// Body part
	bodyHeader := make(textproto.MIMEHeader)
	bodyHeader.Set("Content-Type", "text/plain; charset=utf-8")
	part, err := writer.CreatePart(bodyHeader)
	if err != nil {
		return "", err
	}
	part.Write([]byte(input.Body)) //nolint:errcheck

	// Attachment parts
	for _, path := range input.Attachments {
		fi, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("stat attachment %s: %w", path, err)
		}
		if fi.Size() > 25*1024*1024 {
			return "", fmt.Errorf("attachment %s is %d bytes, exceeds Gmail 25MB limit", path, fi.Size())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read attachment %s: %w", path, err)
		}

		filename := filepath.Base(path)
		mimeType := mime.TypeByExtension(filepath.Ext(path))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		attHeader := make(textproto.MIMEHeader)
		attHeader.Set("Content-Type", mimeType)
		attHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		attHeader.Set("Content-Transfer-Encoding", "base64")

		attPart, err := writer.CreatePart(attHeader)
		if err != nil {
			return "", err
		}

		encoder := base64.NewEncoder(base64.StdEncoding, attPart)
		encoder.Write(data) //nolint:errcheck
		encoder.Close()     //nolint:errcheck
	}

	writer.Close() //nolint:errcheck

	// Combine preamble + multipart body
	var final bytes.Buffer
	io.Copy(&final, &preamble) //nolint:errcheck
	io.Copy(&final, &buf)      //nolint:errcheck

	return base64.URLEncoding.EncodeToString(final.Bytes()), nil
}

func generateBoundary() string {
	b := make([]byte, 16)
	crand.Read(b) //nolint:errcheck
	return fmt.Sprintf("gwx_%x", b)
}

func writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

// ListLabels returns all Gmail labels.
func (gs *GmailService) ListLabels(ctx context.Context) ([]map[string]string, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return nil, err
	}

	opts, err := gs.client.ClientOptions(ctx, "gmail")
	if err != nil {
		return nil, err
	}

	svc, err := gmail.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
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
