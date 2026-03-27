package api

import (
	"encoding/base64"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"google.golang.org/api/gmail/v1"
)

// GmailService wraps Gmail API operations.
type GmailService struct {
	client *Client

	labelMu     sync.Mutex
	labelCache  map[string]string
	labelExpiry time.Time
	labelGroup  singleflight.Group
}

// NewGmailService creates a Gmail service wrapper.
func NewGmailService(client *Client) *GmailService {
	return &GmailService{client: client}
}

// MessageSummary is a simplified message representation for agent consumption.
type MessageSummary struct {
	ID       string   `json:"id"`
	ThreadID string   `json:"thread_id"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Subject  string   `json:"subject"`
	Date     string   `json:"date"`
	Snippet  string   `json:"snippet"`
	Labels   []string `json:"labels"`
	Unread   bool     `json:"unread"`
}

// MessageDetail is a full message representation.
type MessageDetail struct {
	MessageSummary
	Body     string            `json:"body"`
	BodyHTML string            `json:"body_html,omitempty"`
	Headers  map[string]string `json:"headers"`
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
	ReplyTo     string   `json:"reply_to,omitempty"`    // message ID to reply to
	ReplyAll    bool     `json:"reply_all,omitempty"`
	ThreadID    string   `json:"thread_id,omitempty"`
}

// SendResult is returned after sending or drafting.
type SendResult struct {
	MessageID string   `json:"message_id"`
	ThreadID  string   `json:"thread_id"`
	LabelIDs  []string `json:"label_ids,omitempty"`
}

// DigestGroup is a sender-grouped summary for digest view.
type DigestGroup struct {
	Sender   string   `json:"sender"`
	Count    int      `json:"count"`
	Unread   int      `json:"unread"`
	Subjects []string `json:"subjects"`
	Category string   `json:"category"` // "ci_notification", "newsletter", "personal", "transactional"
}

// DigestResult is the output of DigestMessages.
type DigestResult struct {
	TotalMessages int           `json:"total_messages"`
	TotalUnread   int           `json:"total_unread"`
	Groups        []DigestGroup `json:"groups"`
	Summary       string        `json:"summary"`
}

// extractBody recursively searches a message part for the given MIME type and returns decoded text.
// Used by both gmail_read.go (messageToDetail) and is a shared helper across the package.
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
