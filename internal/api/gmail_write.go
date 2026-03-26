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

// SendMessage sends an email.
func (gs *GmailService) SendMessage(ctx context.Context, input *SendInput) (*SendResult, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return nil, err
	}

	svc, err := gs.service(ctx)
	if err != nil {
		return nil, err
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

	svc, err := gs.service(ctx)
	if err != nil {
		return nil, err
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

// ForwardMessage forwards a message to new recipients.
func (gs *GmailService) ForwardMessage(ctx context.Context, messageID string, to []string) (*SendResult, error) {
	// Get original message
	detail, err := gs.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("get original message: %w", err)
	}

	subject := detail.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "fwd:") {
		subject = "Fwd: " + subject
	}

	body := fmt.Sprintf("---------- Forwarded message ----------\nFrom: %s\nDate: %s\nSubject: %s\n\n%s",
		detail.From, detail.Date, detail.Subject, detail.Body)

	return gs.SendMessage(ctx, &SendInput{
		To:      to,
		Subject: subject,
		Body:    body,
	})
}

// ArchiveMessages removes INBOX label from messages matching a query.
func (gs *GmailService) ArchiveMessages(ctx context.Context, query string, maxMessages int64) (int, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return 0, err
	}

	svc, err := gs.service(ctx)
	if err != nil {
		return 0, err
	}

	// List matching messages
	call := svc.Users.Messages.List("me").Q(query)
	if maxMessages > 0 {
		call = call.MaxResults(maxMessages)
	}

	resp, err := call.Do()
	if err != nil {
		return 0, fmt.Errorf("list messages for archive: %w", err)
	}

	if len(resp.Messages) == 0 {
		return 0, nil
	}

	// Collect message IDs for batch processing
	var messageIds []string
	for _, msg := range resp.Messages {
		messageIds = append(messageIds, msg.Id)
	}

	// Process in batches of 100 (Gmail API batch limit)
	const batchSize = 100
	var archived int
	for i := 0; i < len(messageIds); i += batchSize {
		end := i + batchSize
		if end > len(messageIds) {
			end = len(messageIds)
		}

		batch := messageIds[i:end]

		// Create batch modify request
		batchReq := &gmail.BatchModifyMessagesRequest{
			Ids:            batch,
			RemoveLabelIds: []string{"INBOX", "UNREAD"},
		}

		if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
			return archived, err
		}

		if err := svc.Users.Messages.BatchModify("me", batchReq).Do(); err != nil {
			// Fallback to individual processing if batch fails
			for _, msgId := range batch {
				if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
					return archived, err
				}
				modReq := &gmail.ModifyMessageRequest{
					RemoveLabelIds: []string{"INBOX", "UNREAD"},
				}
				if _, err := svc.Users.Messages.Modify("me", msgId, modReq).Do(); err == nil {
					archived++
				}
			}
			continue
		}

		archived += len(batch)
	}

	return archived, nil
}

// MarkRead marks messages matching a query as read.
func (gs *GmailService) MarkRead(ctx context.Context, query string, maxMessages int64) (int, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return 0, err
	}

	svc, err := gs.service(ctx)
	if err != nil {
		return 0, err
	}

	call := svc.Users.Messages.List("me").Q(query)
	if maxMessages > 0 {
		call = call.MaxResults(maxMessages)
	}

	resp, err := call.Do()
	if err != nil {
		return 0, fmt.Errorf("list messages for mark-read: %w", err)
	}

	if len(resp.Messages) == 0 {
		return 0, nil
	}

	// Collect message IDs for batch processing
	var messageIds []string
	for _, msg := range resp.Messages {
		messageIds = append(messageIds, msg.Id)
	}

	// Process in batches of 100 (Gmail API batch limit)
	const batchSize = 100
	var marked int
	for i := 0; i < len(messageIds); i += batchSize {
		end := i + batchSize
		if end > len(messageIds) {
			end = len(messageIds)
		}

		batch := messageIds[i:end]

		// Create batch modify request
		batchReq := &gmail.BatchModifyMessagesRequest{
			Ids:            batch,
			RemoveLabelIds: []string{"UNREAD"},
		}

		if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
			return marked, err
		}

		if err := svc.Users.Messages.BatchModify("me", batchReq).Do(); err != nil {
			// Fallback to individual processing if batch fails
			for _, msgId := range batch {
				if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
					return marked, err
				}
				modReq := &gmail.ModifyMessageRequest{
					RemoveLabelIds: []string{"UNREAD"},
				}
				if _, err := svc.Users.Messages.Modify("me", msgId, modReq).Do(); err == nil {
					marked++
				}
			}
			continue
		}

		marked += len(batch)
	}

	return marked, nil
}

// BatchModifyLabels adds/removes labels on messages matching a query.
// Returns the number of messages modified.
func (gs *GmailService) BatchModifyLabels(ctx context.Context, query string, addLabels, removeLabels []string, maxMessages int64) (int, error) {
	if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
		return 0, err
	}

	svc, err := gs.service(ctx)
	if err != nil {
		return 0, err
	}

	// Resolve label names to IDs
	labelMap, err := gs.labelNameToID(ctx, svc)
	if err != nil {
		return 0, err
	}

	var addIDs, removeIDs []string
	for _, name := range addLabels {
		if id, ok := labelMap[strings.ToUpper(name)]; ok {
			addIDs = append(addIDs, id)
		} else if id, ok := labelMap[name]; ok {
			addIDs = append(addIDs, id)
		} else {
			return 0, fmt.Errorf("label %q not found", name)
		}
	}
	for _, name := range removeLabels {
		if id, ok := labelMap[strings.ToUpper(name)]; ok {
			removeIDs = append(removeIDs, id)
		} else if id, ok := labelMap[name]; ok {
			removeIDs = append(removeIDs, id)
		} else {
			return 0, fmt.Errorf("label %q not found", name)
		}
	}

	// List matching messages
	call := svc.Users.Messages.List("me").Q(query)
	if maxMessages > 0 {
		call = call.MaxResults(maxMessages)
	}
	resp, err := call.Do()
	if err != nil {
		return 0, fmt.Errorf("list messages: %w", err)
	}

	modified := 0
	for _, msg := range resp.Messages {
		if err := gs.client.WaitRate(ctx, "gmail"); err != nil {
			return modified, err
		}
		modReq := &gmail.ModifyMessageRequest{
			AddLabelIds:    addIDs,
			RemoveLabelIds: removeIDs,
		}
		if _, err := svc.Users.Messages.Modify("me", msg.Id, modReq).Do(); err != nil {
			continue
		}
		modified++
	}
	return modified, nil
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
	if key == "Subject" && needsMIMEEncoding(value) {
		buf.WriteString(mime.BEncoding.Encode("utf-8", value))
	} else {
		buf.WriteString(value)
	}
	buf.WriteString("\r\n")
}

func needsMIMEEncoding(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

func (gs *GmailService) labelNameToID(ctx context.Context, svc *gmail.Service) (map[string]string, error) {
	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	m := make(map[string]string)
	for _, l := range resp.Labels {
		m[l.Name] = l.Id
		m[strings.ToUpper(l.Name)] = l.Id // also index uppercase for system labels
	}
	return m, nil
}
