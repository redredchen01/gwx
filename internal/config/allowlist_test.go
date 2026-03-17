package config

import (
	"os"
	"testing"
)

func TestAllowlist_NilAllowsAll(t *testing.T) {
	var a *Allowlist = nil
	if !a.IsAllowed("anything") {
		t.Fatal("nil allowlist should allow everything")
	}
}

func TestAllowlist_ExactMatch(t *testing.T) {
	a := &Allowlist{patterns: []string{"gmail.list", "calendar.list"}}
	if !a.IsAllowed("gmail.list") {
		t.Fatal("exact match should be allowed")
	}
	if !a.IsAllowed("calendar.list") {
		t.Fatal("exact match should be allowed")
	}
	if a.IsAllowed("gmail.send") {
		t.Fatal("non-matching command should be denied")
	}
}

func TestAllowlist_WildcardMatch(t *testing.T) {
	a := &Allowlist{patterns: []string{"gmail.*", "drive.*"}}
	if !a.IsAllowed("gmail.list") {
		t.Fatal("wildcard should match gmail.list")
	}
	if !a.IsAllowed("gmail.send") {
		t.Fatal("wildcard should match gmail.send")
	}
	if !a.IsAllowed("drive.upload") {
		t.Fatal("wildcard should match drive.upload")
	}
	if a.IsAllowed("calendar.list") {
		t.Fatal("calendar should not match gmail.* or drive.*")
	}
}

func TestAllowlist_MixedPatterns(t *testing.T) {
	a := &Allowlist{patterns: []string{"gmail.*", "calendar.list", "drive.search"}}
	if !a.IsAllowed("gmail.send") {
		t.Fatal("should match gmail wildcard")
	}
	if !a.IsAllowed("calendar.list") {
		t.Fatal("should match exact calendar.list")
	}
	if a.IsAllowed("calendar.create") {
		t.Fatal("should not match calendar.create")
	}
	if !a.IsAllowed("drive.search") {
		t.Fatal("should match exact drive.search")
	}
	if a.IsAllowed("drive.upload") {
		t.Fatal("should not match drive.upload")
	}
}

func TestLoadAllowlist_Empty(t *testing.T) {
	os.Unsetenv(envEnableCommands)
	a := LoadAllowlist()
	if a != nil {
		t.Fatal("empty env should return nil allowlist")
	}
}

func TestLoadAllowlist_All(t *testing.T) {
	os.Setenv(envEnableCommands, "all")
	defer os.Unsetenv(envEnableCommands)
	a := LoadAllowlist()
	if a != nil {
		t.Fatal("'all' should return nil allowlist")
	}
}

func TestLoadAllowlist_Star(t *testing.T) {
	os.Setenv(envEnableCommands, "*")
	defer os.Unsetenv(envEnableCommands)
	a := LoadAllowlist()
	if a != nil {
		t.Fatal("'*' should return nil allowlist")
	}
}

func TestLoadAllowlist_CommaSeparated(t *testing.T) {
	os.Setenv(envEnableCommands, "gmail.list, calendar.*, drive.search")
	defer os.Unsetenv(envEnableCommands)
	a := LoadAllowlist()
	if a == nil {
		t.Fatal("should return non-nil allowlist")
	}
	if !a.IsAllowed("gmail.list") {
		t.Fatal("should allow gmail.list")
	}
	if !a.IsAllowed("calendar.create") {
		t.Fatal("should allow calendar wildcard")
	}
	if a.IsAllowed("drive.upload") {
		t.Fatal("should deny drive.upload")
	}
}
