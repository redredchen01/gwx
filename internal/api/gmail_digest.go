package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// DigestMessages fetches recent messages and groups them by sender with categorization.
func (gs *GmailService) DigestMessages(ctx context.Context, maxMessages int64, unreadOnly bool) (*DigestResult, error) {
	messages, _, err := gs.ListMessages(ctx, "", nil, maxMessages, unreadOnly)
	if err != nil {
		return nil, err
	}

	// Group by sender
	type group struct {
		sender   string
		subjects []string
		unread   int
	}
	groupMap := make(map[string]*group)
	var order []string

	for _, m := range messages {
		sender := m.From
		// Extract just the name/email
		if idx := strings.Index(sender, " <"); idx > 0 {
			sender = sender[:idx]
		}

		g, ok := groupMap[sender]
		if !ok {
			g = &group{sender: sender}
			groupMap[sender] = g
			order = append(order, sender)
		}
		g.subjects = append(g.subjects, m.Subject)
		if m.Unread {
			g.unread++
		}
	}

	totalUnread := 0
	var groups []DigestGroup
	for _, key := range order {
		g := groupMap[key]
		cat := categorizeGroup(g.sender, g.subjects)
		dg := DigestGroup{
			Sender:   g.sender,
			Count:    len(g.subjects),
			Unread:   g.unread,
			Category: cat,
		}
		// Deduplicate similar subjects (e.g., CI notifications)
		if cat == "ci_notification" && len(g.subjects) > 3 {
			dg.Subjects = []string{fmt.Sprintf("[%d similar CI notifications]", len(g.subjects))}
		} else {
			dg.Subjects = g.subjects
		}
		totalUnread += g.unread
		groups = append(groups, dg)
	}

	// Sort: personal first, then by count descending
	sortGroups(groups)

	// Generate summary
	summary := generateDigestSummary(groups, len(messages), totalUnread)

	return &DigestResult{
		TotalMessages: len(messages),
		TotalUnread:   totalUnread,
		Groups:        groups,
		Summary:       summary,
	}, nil
}

func categorizeGroup(sender string, subjects []string) string {
	senderLower := strings.ToLower(sender)

	// Check subjects for CI/CD patterns (sender might be the user's own name for GitHub)
	ciPatterns := 0
	devPatterns := 0
	for _, s := range subjects {
		sl := strings.ToLower(s)
		if strings.Contains(sl, "run failed") || strings.Contains(sl, "build failed") ||
			strings.Contains(sl, "pipeline failed") || strings.Contains(sl, "ci -") {
			ciPatterns++
		}
		if strings.Contains(sl, "[") && strings.Contains(sl, "]") &&
			(strings.Contains(sl, "pr") || strings.Contains(sl, "issue") ||
				strings.Contains(sl, "push") || strings.Contains(sl, "run")) {
			devPatterns++
		}
	}
	if ciPatterns > 0 && ciPatterns >= len(subjects)/2 {
		return "ci_notification"
	}
	if devPatterns > 0 && devPatterns >= len(subjects)/2 {
		return "dev_notification"
	}

	// CI/CD by sender name
	if strings.Contains(senderLower, "github") || strings.Contains(senderLower, "gitlab") ||
		strings.Contains(senderLower, "circleci") || strings.Contains(senderLower, "jenkins") {
		return "dev_notification"
	}
	// Newsletters / automated
	if strings.Contains(senderLower, "noreply") || strings.Contains(senderLower, "no-reply") ||
		strings.Contains(senderLower, "newsletter") || strings.Contains(senderLower, "digest") {
		return "newsletter"
	}
	// Transactional (npm, billing, etc.)
	if strings.Contains(senderLower, "npm") || strings.Contains(senderLower, "billing") ||
		strings.Contains(senderLower, "receipt") || strings.Contains(senderLower, "invoice") {
		return "transactional"
	}
	return "personal"
}

func sortGroups(groups []DigestGroup) {
	categoryOrder := map[string]int{"personal": 0, "transactional": 1, "dev_notification": 2, "ci_notification": 3, "newsletter": 4}
	sort.Slice(groups, func(i, j int) bool {
		oi := categoryOrder[groups[i].Category]
		oj := categoryOrder[groups[j].Category]
		if oi != oj {
			return oi < oj
		}
		return groups[i].Count > groups[j].Count
	})
}

func generateDigestSummary(groups []DigestGroup, total, unread int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d messages (%d unread). ", total, unread))

	ciCount := 0
	personalCount := 0
	for _, g := range groups {
		switch g.Category {
		case "ci_notification":
			ciCount += g.Count
		case "personal":
			personalCount += g.Count
		}
	}
	if personalCount > 0 {
		sb.WriteString(fmt.Sprintf("%d personal. ", personalCount))
	}
	if ciCount > 0 {
		sb.WriteString(fmt.Sprintf("%d CI notifications (can batch archive). ", ciCount))
	}
	return sb.String()
}
