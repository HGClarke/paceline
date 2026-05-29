package cmd

import (
	"testing"
	"time"
)

func TestParseDateRange_BothValid(t *testing.T) {
	from, to, err := parseDateRange("2025-01-01", "2025-03-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from == nil || from.Format("2006-01-02") != "2025-01-01" {
		t.Errorf("from: got %v, want 2025-01-01", from)
	}
	if to == nil || to.Format("2006-01-02") != "2025-03-31" {
		t.Errorf("to: got %v, want 2025-03-31", to)
	}
}

func TestParseDateRange_FromOnly(t *testing.T) {
	from, to, err := parseDateRange("2025-06-01", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from == nil {
		t.Error("expected non-nil from")
	}
	if to != nil {
		t.Errorf("expected nil to, got %v", to)
	}
}

func TestParseDateRange_ToOnly(t *testing.T) {
	from, to, err := parseDateRange("", "2025-06-30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from != nil {
		t.Errorf("expected nil from, got %v", from)
	}
	if to == nil {
		t.Error("expected non-nil to")
	}
}

func TestParseDateRange_BothEmpty(t *testing.T) {
	from, to, err := parseDateRange("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from != nil || to != nil {
		t.Errorf("expected both nil, got from=%v to=%v", from, to)
	}
}

func TestParseDateRange_InvalidFrom(t *testing.T) {
	_, _, err := parseDateRange("not-a-date", "2025-03-31")
	if err == nil {
		t.Fatal("expected error for invalid --from, got nil")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestParseDateRange_InvalidTo(t *testing.T) {
	_, _, err := parseDateRange("2025-01-01", "bad")
	if err == nil {
		t.Fatal("expected error for invalid --to, got nil")
	}
}

func TestParseDateRange_FromAfterTo(t *testing.T) {
	_, _, err := parseDateRange("2025-06-01", "2025-01-01")
	if err == nil {
		t.Fatal("expected error when --from is after --to, got nil")
	}
	want := "--from must not be after --to"
	if err.Error() != want {
		t.Errorf("error message: got %q, want %q", err.Error(), want)
	}
}

func TestParseDateRange_SameDay(t *testing.T) {
	// Same day for --from and --to is valid (single-day range).
	from, to, err := parseDateRange("2025-03-15", "2025-03-15")
	if err != nil {
		t.Fatalf("unexpected error for same-day range: %v", err)
	}
	if from == nil {
		t.Fatal("expected from to be set")
	}
	if to == nil {
		t.Fatal("expected to to be set")
	}
	if !from.Equal(*to) {
		t.Errorf("expected from == to, got from=%v to=%v", from, to)
	}
}

func TestDateRangeLabel_Both(t *testing.T) {
	from := mustParseDate("2025-01-01")
	to := mustParseDate("2025-03-31")
	got := dateRangeLabel(&from, &to)
	want := "2025-01-01 to 2025-03-31"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDateRangeLabel_FromOnly(t *testing.T) {
	from := mustParseDate("2025-06-01")
	got := dateRangeLabel(&from, nil)
	want := "from 2025-06-01"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDateRangeLabel_ToOnly(t *testing.T) {
	to := mustParseDate("2025-06-30")
	got := dateRangeLabel(nil, &to)
	want := "to 2025-06-30"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDateRangeLabel_Neither(t *testing.T) {
	got := dateRangeLabel(nil, nil)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
