package auth

import (
	"testing"
)

func TestAllScopes_SingleService(t *testing.T) {
	scopes := AllScopes([]string{"gmail"}, false)
	if len(scopes) == 0 {
		t.Fatal("gmail should have scopes")
	}
	// Gmail has 3 scopes
	if len(scopes) != 3 {
		t.Fatalf("expected 3 gmail scopes, got %d: %v", len(scopes), scopes)
	}
}

func TestAllScopes_MultipleServices(t *testing.T) {
	scopes := AllScopes([]string{"gmail", "calendar", "drive"}, false)
	if len(scopes) < 5 {
		t.Fatalf("expected at least 5 scopes for gmail+calendar+drive, got %d", len(scopes))
	}
}

func TestAllScopes_NoDuplicates(t *testing.T) {
	scopes := AllScopes([]string{"gmail", "gmail", "calendar"}, false)
	seen := make(map[string]bool)
	for _, s := range scopes {
		if seen[s] {
			t.Fatalf("duplicate scope: %s", s)
		}
		seen[s] = true
	}
}

func TestAllScopes_ReadOnly(t *testing.T) {
	full := AllScopes([]string{"gmail"}, false)
	ro := AllScopes([]string{"gmail"}, true)

	if len(ro) >= len(full) {
		t.Fatalf("readonly scopes (%d) should be fewer than full scopes (%d)", len(ro), len(full))
	}

	// Readonly gmail should be just 1 scope
	if len(ro) != 1 {
		t.Fatalf("expected 1 readonly gmail scope, got %d: %v", len(ro), ro)
	}
}

func TestAllScopes_UnknownService(t *testing.T) {
	scopes := AllScopes([]string{"nonexistent"}, false)
	if len(scopes) != 0 {
		t.Fatalf("unknown service should return empty scopes, got %v", scopes)
	}
}

func TestAllScopes_AllServicesHaveScopes(t *testing.T) {
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat"}
	for _, svc := range services {
		scopes := AllScopes([]string{svc}, false)
		if len(scopes) == 0 {
			t.Errorf("service %q has no scopes defined", svc)
		}
	}
}

func TestAllScopes_AllServicesHaveReadOnlyScopes(t *testing.T) {
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat"}
	for _, svc := range services {
		scopes := AllScopes([]string{svc}, true)
		if len(scopes) == 0 {
			t.Errorf("service %q has no readonly scopes defined", svc)
		}
	}
}

func TestServiceScopes_ContainsExpectedScopes(t *testing.T) {
	// Spot check a few known scopes
	gmailScopes := ServiceScopes["gmail"]
	found := false
	for _, s := range gmailScopes {
		if s == "https://www.googleapis.com/auth/gmail.readonly" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("gmail scopes should contain gmail.readonly")
	}

	calScopes := ServiceScopes["calendar"]
	if len(calScopes) == 0 || calScopes[0] != "https://www.googleapis.com/auth/calendar" {
		t.Fatalf("calendar scope unexpected: %v", calScopes)
	}
}
