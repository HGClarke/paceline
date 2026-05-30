package cmd

import "testing"

func TestValidateCompareArgs_WeekFlag(t *testing.T) {
	if err := validateCompareArgs(2025, 2024, 22, "", ""); err == nil {
		t.Error("expected error for --week + --compare")
	}
}

func TestValidateCompareArgs_FromFlag(t *testing.T) {
	if err := validateCompareArgs(2025, 2024, 0, "2025-01-01", ""); err == nil {
		t.Error("expected error for --from + --compare")
	}
}

func TestValidateCompareArgs_ToFlag(t *testing.T) {
	if err := validateCompareArgs(2025, 2024, 0, "", "2025-12-31"); err == nil {
		t.Error("expected error for --to + --compare")
	}
}

func TestValidateCompareArgs_SameYear(t *testing.T) {
	err := validateCompareArgs(2025, 2025, 0, "", "")
	if err == nil {
		t.Fatal("expected error for same-year compare")
	}
	want := "--compare year must differ from the primary year"
	if err.Error() != want {
		t.Errorf("error message = %q, want %q", err.Error(), want)
	}
}

func TestValidateCompareArgs_Valid(t *testing.T) {
	if err := validateCompareArgs(2025, 2024, 0, "", ""); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
