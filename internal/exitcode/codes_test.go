package exitcode

import "testing"

func TestDescription_KnownCodes(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{OK, "success"},
		{GeneralError, "general_error"},
		{AuthRequired, "auth_required"},
		{AuthExpired, "auth_expired"},
		{PermissionDenied, "permission_denied"},
		{NotFound, "not_found"},
		{RateLimited, "rate_limited"},
		{CircuitOpen, "circuit_open"},
		{InvalidInput, "invalid_input"},
		{DryRunSuccess, "dry_run_success"},
	}
	for _, tt := range tests {
		got := Description(tt.code)
		if got != tt.want {
			t.Errorf("Description(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestDescription_Unknown(t *testing.T) {
	got := Description(999)
	if got != "unknown_999" {
		t.Errorf("Description(999) = %q, want 'unknown_999'", got)
	}
}

func TestAll_ContainsAllCodes(t *testing.T) {
	all := All()
	expected := []int{OK, GeneralError, UsageError, AuthRequired, AuthExpired,
		PermissionDenied, NotFound, Conflict, RateLimited, CircuitOpen,
		InvalidInput, DryRunSuccess}
	for _, code := range expected {
		if _, ok := all[code]; !ok {
			t.Errorf("All() missing code %d", code)
		}
	}
}

func TestAll_ReturnsCopy(t *testing.T) {
	a := All()
	a[999] = "injected"
	b := All()
	if _, ok := b[999]; ok {
		t.Fatal("All() should return a copy, not the original map")
	}
}
