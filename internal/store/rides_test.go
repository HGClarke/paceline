package store_test

import (
	"testing"
	"time"

	"github.com/hollandclarke/paceline/internal/parser"
	"github.com/hollandclarke/paceline/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestInsertAndGetRide(t *testing.T) {
	s := openTestStore(t)

	ride := parser.Ride{
		Filename:       "test.gpx",
		RecordedAt:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		DistanceM:      30000,
		DurationS:      3600,
		ElevationGainM: 500,
		AvgSpeedMPS:    8.3,
		MaxSpeedMPS:    15.0,
		SourceFormat:   "gpx",
	}

	id, err := s.InsertRide(ride)
	if err != nil {
		t.Fatalf("InsertRide: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	got, err := s.GetRide(id)
	if err != nil {
		t.Fatalf("GetRide: %v", err)
	}
	if got.Filename != ride.Filename {
		t.Errorf("filename: got %q want %q", got.Filename, ride.Filename)
	}
	if got.DistanceM != ride.DistanceM {
		t.Errorf("distance: got %v want %v", got.DistanceM, ride.DistanceM)
	}
}

func TestInsertRide_Idempotent(t *testing.T) {
	s := openTestStore(t)

	ride := parser.Ride{
		Filename:     "dup.gpx",
		RecordedAt:   time.Now(),
		SourceFormat: "gpx",
	}

	_, err := s.InsertRide(ride)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	skipped, err := s.InsertRide(ride)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if skipped != 0 {
		t.Errorf("expected 0 (skipped) on duplicate, got %d", skipped)
	}
}

func TestListRides_Filters(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "feb.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "jan2.gpx", RecordedAt: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	year, month := 2024, 1
	result, _, err := s.ListRides(store.RideFilters{Year: &year, Month: &month, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 January rides, got %d", len(result))
	}
}
