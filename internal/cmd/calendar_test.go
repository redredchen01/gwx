package cmd

import (
	"testing"
	"time"
)

func TestParseTime_RFC3339(t *testing.T) {
	parsed, err := parseTime("2026-03-17T10:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Year() != 2026 || parsed.Month() != 3 || parsed.Day() != 17 {
		t.Fatalf("wrong date: %v", parsed)
	}
}

func TestParseTime_DateOnly(t *testing.T) {
	parsed, err := parseTime("2026-03-17")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Year() != 2026 || parsed.Month() != 3 || parsed.Day() != 17 {
		t.Fatalf("wrong date: %v", parsed)
	}
}

func TestParseTime_Today(t *testing.T) {
	parsed, err := parseTime("today")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	now := time.Now()
	if parsed.Year() != now.Year() || parsed.Month() != now.Month() || parsed.Day() != now.Day() {
		t.Fatalf("'today' should be today, got %v", parsed)
	}
	if parsed.Hour() != 0 || parsed.Minute() != 0 {
		t.Fatalf("'today' should be start of day, got %v", parsed)
	}
}

func TestParseTime_Tomorrow(t *testing.T) {
	parsed, err := parseTime("tomorrow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tomorrow := time.Now().AddDate(0, 0, 1)
	if parsed.Day() != tomorrow.Day() {
		t.Fatalf("'tomorrow' should be tomorrow, got %v", parsed)
	}
}

func TestParseTime_Invalid(t *testing.T) {
	_, err := parseTime("not-a-date")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}
