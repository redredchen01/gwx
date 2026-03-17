package api

import "testing"

func TestIsValidDriveID_Valid(t *testing.T) {
	valid := []string{
		"1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
		"0B1234567890abcDEF",
		"abc-def_123",
		"root",
	}
	for _, id := range valid {
		if !isValidDriveID(id) {
			t.Errorf("expected %q to be valid", id)
		}
	}
}

func TestIsValidDriveID_Invalid(t *testing.T) {
	invalid := []string{
		"abc' OR 1=1--",
		"folder id with spaces",
		"../../../etc/passwd",
		"id;drop table",
		"",
		"id\ninjection",
	}
	for _, id := range invalid {
		if isValidDriveID(id) {
			t.Errorf("expected %q to be invalid", id)
		}
	}
}
