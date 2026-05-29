package store_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
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

func TestInsertAndGetRide_NullableFields(t *testing.T) {
	s := openTestStore(t)
	hr, power, cal := 155, 250, 800
	ride := parser.Ride{
		Filename:     "nullables.gpx",
		RecordedAt:   time.Now(),
		SourceFormat: "gpx",
		AvgHRBPM:    &hr,
		MaxHRBPM:    &hr,
		AvgPowerW:   &power,
		MaxPowerW:   &power,
		Calories:    &cal,
	}
	id, err := s.InsertRide(ride)
	if err != nil {
		t.Fatalf("InsertRide: %v", err)
	}
	got, err := s.GetRide(id)
	if err != nil {
		t.Fatalf("GetRide: %v", err)
	}
	if got.AvgHRBPM == nil || *got.AvgHRBPM != hr {
		t.Errorf("AvgHRBPM: got %v want %d", got.AvgHRBPM, hr)
	}
	if got.AvgPowerW == nil || *got.AvgPowerW != power {
		t.Errorf("AvgPowerW: got %v want %d", got.AvgPowerW, power)
	}
	if got.Calories == nil || *got.Calories != cal {
		t.Errorf("Calories: got %v want %d", got.Calories, cal)
	}
}

func TestGetRide_NotFound(t *testing.T) {
	s := openTestStore(t)
	_, err := s.GetRide(9999)
	if err == nil {
		t.Fatal("expected error for non-existent ride, got nil")
	}
}

func TestListRides_Pagination(t *testing.T) {
	s := openTestStore(t)
	for i := 0; i < 5; i++ {
		r := parser.Ride{
			Filename:     fmt.Sprintf("ride%d.gpx", i),
			RecordedAt:   time.Now().Add(time.Duration(i) * time.Hour),
			SourceFormat: "gpx",
		}
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	rides, total, err := s.ListRides(store.RideFilters{Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 5 {
		t.Errorf("total: got %d want 5", total)
	}
	if len(rides) != 2 {
		t.Errorf("page 1: got %d rides want 2", len(rides))
	}
	rides2, _, err := s.ListRides(store.RideFilters{Page: 2, Limit: 2})
	if err != nil {
		t.Fatalf("ListRides page 2: %v", err)
	}
	if len(rides2) != 2 {
		t.Errorf("page 2: got %d rides want 2", len(rides2))
	}
}

func TestListRides_EmptyResult(t *testing.T) {
	s := openTestStore(t)
	year := 1990
	rides, total, err := s.ListRides(store.RideFilters{Year: &year, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 0 {
		t.Errorf("total: got %d want 0", total)
	}
	if rides == nil {
		t.Error("expected empty slice, got nil")
	}
}

func TestGetStats_FiltersByYearAndMonth(t *testing.T) {
	s := openTestStore(t)

	// Use a fixed date well within any month boundary.
	fixedNow := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	year, month := 2025, 6

	rides := []parser.Ride{
		{Filename: "this_month.gpx", RecordedAt: fixedNow, DistanceM: 30000, DurationS: 3600, ElevationGainM: 400, SourceFormat: "gpx"},
		{Filename: "last_year.gpx", RecordedAt: fixedNow.AddDate(-1, 0, 0), DistanceM: 20000, DurationS: 2400, ElevationGainM: 200, SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	stats, err := s.GetStats(store.StatsFilters{Year: &year, Month: &month})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.RideCount != 1 {
		t.Errorf("expected 1 ride this month, got %d", stats.RideCount)
	}
	if stats.TotalDistanceM != 30000 {
		t.Errorf("expected distance 30000, got %v", stats.TotalDistanceM)
	}
	if stats.TotalDurationS != 3600 {
		t.Errorf("expected duration 3600, got %d", stats.TotalDurationS)
	}
	if stats.TotalElevationM != 400 {
		t.Errorf("expected elevation 400, got %v", stats.TotalElevationM)
	}
}

func TestGetStats_AllTime(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{
			Filename:       "ride_2023.gpx",
			RecordedAt:     time.Date(2023, 3, 10, 9, 0, 0, 0, time.UTC),
			DistanceM:      20000,
			DurationS:      2400,
			ElevationGainM: 200,
			SourceFormat:   "gpx",
		},
		{
			Filename:       "ride_2025.gpx",
			RecordedAt:     time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
			DistanceM:      30000,
			DurationS:      3600,
			ElevationGainM: 400,
			SourceFormat:   "gpx",
		},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	stats, err := s.GetStats(store.StatsFilters{})
	if err != nil {
		t.Fatalf("GetStats all-time: %v", err)
	}
	if stats.RideCount != 2 {
		t.Errorf("RideCount: got %d, want 2", stats.RideCount)
	}
	if stats.TotalDistanceM != 50000 {
		t.Errorf("TotalDistanceM: got %v, want 50000", stats.TotalDistanceM)
	}
	if stats.TotalDurationS != 6000 {
		t.Errorf("TotalDurationS: got %d, want 6000", stats.TotalDurationS)
	}
	if stats.TotalElevationM != 600 {
		t.Errorf("TotalElevationM: got %v, want 600", stats.TotalElevationM)
	}
}

func TestDeleteRide(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "delete_me.gpx")

	// Insert streams for the ride.
	hr := 150
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, HRBPM: &hr},
		{RideID: rideID, Timestamp: time.Now().Add(time.Second), ElapsedS: 1, HRBPM: &hr},
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	if err := s.DeleteRide(rideID); err != nil {
		t.Fatalf("DeleteRide: %v", err)
	}

	// Ride should be gone.
	if _, err := s.GetRide(rideID); err == nil {
		t.Error("expected GetRide to return error after delete, got nil")
	}

	// Streams should be gone.
	pts, err := s.GetStreams(rideID, "hr")
	if err != nil {
		t.Fatalf("GetStreams after delete: %v", err)
	}
	if len(pts) != 0 {
		t.Errorf("expected 0 stream points after delete, got %d", len(pts))
	}
}

func TestDeleteRide_NotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.DeleteRide(9999)
	if err == nil {
		t.Fatal("expected error deleting non-existent ride, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestDeleteAll(t *testing.T) {
	s := openTestStore(t)

	// Insert two rides each with streams.
	hr := 160
	for i, name := range []string{"ride_a.gpx", "ride_b.gpx"} {
		id := insertTestRide(t, s, name)
		streams := []parser.Stream{
			{RideID: id, Timestamp: time.Now().Add(time.Duration(i) * time.Second), ElapsedS: i, HRBPM: &hr},
		}
		if err := s.InsertStreams(streams); err != nil {
			t.Fatalf("InsertStreams: %v", err)
		}
	}

	if err := s.DeleteAll(); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}

	rides, total, err := s.ListRides(store.RideFilters{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("ListRides after DeleteAll: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 rides after DeleteAll, got total=%d", total)
	}
	if len(rides) != 0 {
		t.Errorf("expected empty rides slice after DeleteAll, got %d", len(rides))
	}
}

func TestGetStats_Empty(t *testing.T) {
	s := openTestStore(t)
	year, month := 2025, 1
	stats, err := s.GetStats(store.StatsFilters{Year: &year, Month: &month})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.RideCount != 0 {
		t.Errorf("expected 0 rides, got %d", stats.RideCount)
	}
	if stats.TotalDistanceM != 0 {
		t.Errorf("expected 0 distance, got %v", stats.TotalDistanceM)
	}
	if stats.TotalDurationS != 0 {
		t.Errorf("expected 0 duration, got %d", stats.TotalDurationS)
	}
	if stats.TotalElevationM != 0 {
		t.Errorf("expected 0 elevation, got %v", stats.TotalElevationM)
	}
}

func TestGetRideByPosition_HappyPath(t *testing.T) {
	s := openTestStore(t)

	idNewest, err := s.InsertRide(parser.Ride{
		Filename:     "newest.gpx",
		RecordedAt:   time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		SourceFormat: "gpx",
	})
	if err != nil {
		t.Fatalf("insert newest: %v", err)
	}
	_, err = s.InsertRide(parser.Ride{
		Filename:     "oldest.gpx",
		RecordedAt:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		SourceFormat: "gpx",
	})
	if err != nil {
		t.Fatalf("insert oldest: %v", err)
	}

	got, err := s.GetRideByPosition(1)
	if err != nil {
		t.Fatalf("GetRideByPosition(1): %v", err)
	}
	if got.ID != idNewest {
		t.Errorf("position 1: got DB ID %d, want %d", got.ID, idNewest)
	}
	if got.Position != 1 {
		t.Errorf("Position field: got %d, want 1", got.Position)
	}
}

func TestGetRideByPosition_OutOfRange(t *testing.T) {
	s := openTestStore(t)
	_, err := s.InsertRide(parser.Ride{
		Filename:     "only.gpx",
		RecordedAt:   time.Now(),
		SourceFormat: "gpx",
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	_, err = s.GetRideByPosition(99)
	if err == nil {
		t.Fatal("expected error for out-of-range position, got nil")
	}
	if !strings.Contains(err.Error(), "position 99") {
		t.Errorf("expected error to mention position 99, got: %v", err)
	}
}

func TestGetRideByPosition_AfterDelete(t *testing.T) {
	s := openTestStore(t)

	idA, err := s.InsertRide(parser.Ride{Filename: "a.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert a: %v", err)
	}
	idB, err := s.InsertRide(parser.Ride{Filename: "b.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert b: %v", err)
	}
	idC, err := s.InsertRide(parser.Ride{Filename: "c.gpx", RecordedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert c: %v", err)
	}
	_ = idA

	// Before delete: position 2 is B
	got, err := s.GetRideByPosition(2)
	if err != nil {
		t.Fatalf("GetRideByPosition(2) before delete: %v", err)
	}
	if got.ID != idB {
		t.Errorf("before delete: position 2 got ID %d, want %d (B)", got.ID, idB)
	}

	if err := s.DeleteRide(idB); err != nil {
		t.Fatalf("DeleteRide: %v", err)
	}

	// After delete: position 2 is now C
	got, err = s.GetRideByPosition(2)
	if err != nil {
		t.Fatalf("GetRideByPosition(2) after delete: %v", err)
	}
	if got.ID != idC {
		t.Errorf("after delete: position 2 got ID %d, want %d (C)", got.ID, idC)
	}
}

func TestGetRideByPosition_PositionShiftOnMidInsert(t *testing.T) {
	s := openTestStore(t)

	idRecent, err := s.InsertRide(parser.Ride{Filename: "recent.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert recent: %v", err)
	}
	idOld, err := s.InsertRide(parser.Ride{Filename: "old.gpx", RecordedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert old: %v", err)
	}
	_ = idRecent

	// Before mid-insert: position 2 is old
	got, err := s.GetRideByPosition(2)
	if err != nil {
		t.Fatalf("GetRideByPosition(2) before mid-insert: %v", err)
	}
	if got.ID != idOld {
		t.Errorf("before mid-insert: position 2 got ID %d, want %d (old)", got.ID, idOld)
	}

	// Insert a ride whose date falls between the two
	_, err = s.InsertRide(parser.Ride{Filename: "mid.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert mid: %v", err)
	}

	// After mid-insert: position 2 is mid, position 3 is old
	got, err = s.GetRideByPosition(2)
	if err != nil {
		t.Fatalf("GetRideByPosition(2) after mid-insert: %v", err)
	}
	if got.Filename != "mid.gpx" {
		t.Errorf("after mid-insert: position 2 filename = %q, want \"mid.gpx\"", got.Filename)
	}

	got, err = s.GetRideByPosition(3)
	if err != nil {
		t.Fatalf("GetRideByPosition(3) after mid-insert: %v", err)
	}
	if got.ID != idOld {
		t.Errorf("after mid-insert: position 3 got ID %d, want %d (old)", got.ID, idOld)
	}
}

func TestListRides_HasPositions(t *testing.T) {
	s := openTestStore(t)

	_, err := s.InsertRide(parser.Ride{Filename: "newest.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert newest: %v", err)
	}
	_, err = s.InsertRide(parser.Ride{Filename: "middle.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert middle: %v", err)
	}
	_, err = s.InsertRide(parser.Ride{Filename: "oldest.gpx", RecordedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert oldest: %v", err)
	}

	rides, _, err := s.ListRides(store.RideFilters{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if len(rides) != 3 {
		t.Fatalf("expected 3 rides, got %d", len(rides))
	}
	if rides[0].Position != 1 {
		t.Errorf("rides[0].Position = %d, want 1", rides[0].Position)
	}
	if rides[1].Position != 2 {
		t.Errorf("rides[1].Position = %d, want 2", rides[1].Position)
	}
	if rides[2].Position != 3 {
		t.Errorf("rides[2].Position = %d, want 3", rides[2].Position)
	}
	if rides[0].Filename != "newest.gpx" {
		t.Errorf("rides[0].Filename = %q, want \"newest.gpx\"", rides[0].Filename)
	}
}

func TestListRides_FilteredPositionsAreGlobal(t *testing.T) {
	// Verify the documented behavior: "Position reflects its global rank across all rides,
	// even when filters are applied."
	s := openTestStore(t)

	// Insert 4 rides across different months
	if _, err := s.InsertRide(parser.Ride{Filename: "mar15.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"}); err != nil {
		t.Fatalf("insert mar15: %v", err)
	}
	if _, err := s.InsertRide(parser.Ride{Filename: "feb10.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"}); err != nil {
		t.Fatalf("insert feb10: %v", err)
	}
	if _, err := s.InsertRide(parser.Ride{Filename: "jan25.gpx", RecordedAt: time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"}); err != nil {
		t.Fatalf("insert jan25: %v", err)
	}
	if _, err := s.InsertRide(parser.Ride{Filename: "jan10.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"}); err != nil {
		t.Fatalf("insert jan10: %v", err)
	}

	// Filter by January only
	year, month := 2024, 1
	rides, total, err := s.ListRides(store.RideFilters{Year: &year, Month: &month, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}

	if len(rides) != 2 {
		t.Fatalf("expected 2 January rides, got %d", len(rides))
	}
	if total != 2 {
		t.Errorf("total count = %d, want 2 (filtered count, not global)", total)
	}

	// Critical: According to the docstring, positions should be GLOBAL.
	// jan25.gpx should have global position 3 (mar15=1, feb10=2, jan25=3, jan10=4)
	// jan10.gpx should have global position 4.
	if rides[0].Position != 3 {
		t.Errorf("rides[0] (jan25).Position = %d, want 3 (global rank)", rides[0].Position)
	}
	if rides[1].Position != 4 {
		t.Errorf("rides[1] (jan10).Position = %d, want 4 (global rank)", rides[1].Position)
	}
}

func TestGetRecords_Empty(t *testing.T) {
	s := openTestStore(t)
	recs, err := s.GetRecords(store.RecordsFilters{})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if recs.LongestDistanceM != nil {
		t.Error("expected nil LongestDistanceM")
	}
	if recs.LongestDurationS != nil {
		t.Error("expected nil LongestDurationS")
	}
	if recs.MostElevationGainM != nil {
		t.Error("expected nil MostElevationGainM")
	}
	if recs.HighestAvgPowerW != nil {
		t.Error("expected nil HighestAvgPowerW")
	}
	if recs.HighestAvgSpeedMPS != nil {
		t.Error("expected nil HighestAvgSpeedMPS")
	}
	if recs.HighestAvgHRBPM != nil {
		t.Error("expected nil HighestAvgHRBPM")
	}
	if recs.HighestMaxSpeedMPS != nil {
		t.Error("expected nil HighestMaxSpeedMPS")
	}
	if recs.MostCaloriesKcal != nil {
		t.Error("expected nil MostCaloriesKcal")
	}
	if recs.HighestAltitudeM != nil {
		t.Error("expected nil HighestAltitudeM")
	}
}

func TestGetRecords_AllFields(t *testing.T) {
	s := openTestStore(t)

	hr, power, cal := 155, 250, 800
	alt := 1200.0
	rideDate := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

	id, err := s.InsertRide(parser.Ride{
		Filename:       "full.gpx",
		RecordedAt:     rideDate,
		DistanceM:      50000,
		DurationS:      7200,
		ElevationGainM: 1500,
		AvgSpeedMPS:    10.0,
		MaxSpeedMPS:    20.0,
		AvgHRBPM:       &hr,
		MaxHRBPM:       &hr,
		AvgPowerW:      &power,
		MaxPowerW:      &power,
		Calories:       &cal,
		SourceFormat:   "gpx",
	})
	if err != nil {
		t.Fatalf("InsertRide: %v", err)
	}
	if err := s.InsertStreams([]parser.Stream{
		{RideID: id, Timestamp: rideDate, ElapsedS: 0, AltitudeM: &alt},
	}); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	recs, err := s.GetRecords(store.RecordsFilters{})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	check := func(name string, pr *store.PersonalRecord, wantVal float64) {
		t.Helper()
		if pr == nil {
			t.Errorf("%s: got nil, want %v", name, wantVal)
			return
		}
		if pr.RawValue != wantVal {
			t.Errorf("%s.RawValue: got %v, want %v", name, pr.RawValue, wantVal)
		}
		if pr.Date.Format("2006-01-02") != "2024-06-15" {
			t.Errorf("%s.Date: got %v, want 2024-06-15", name, pr.Date)
		}
	}

	check("LongestDistanceM", recs.LongestDistanceM, 50000)
	check("LongestDurationS", recs.LongestDurationS, 7200)
	check("MostElevationGainM", recs.MostElevationGainM, 1500)
	check("HighestAvgPowerW", recs.HighestAvgPowerW, 250)
	check("HighestAvgSpeedMPS", recs.HighestAvgSpeedMPS, 10.0)
	check("HighestAvgHRBPM", recs.HighestAvgHRBPM, 155)
	check("HighestMaxSpeedMPS", recs.HighestMaxSpeedMPS, 20.0)
	check("MostCaloriesKcal", recs.MostCaloriesKcal, 800)
	check("HighestAltitudeM", recs.HighestAltitudeM, 1200)
}

func TestGetRecords_MissingNullable(t *testing.T) {
	s := openTestStore(t)

	// Insert a ride with no nullable fields and no stream data.
	_, err := s.InsertRide(parser.Ride{
		Filename:       "basic.gpx",
		RecordedAt:     time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		DistanceM:      30000,
		DurationS:      3600,
		ElevationGainM: 500,
		AvgSpeedMPS:    8.0,
		MaxSpeedMPS:    15.0,
		SourceFormat:   "gpx",
	})
	if err != nil {
		t.Fatalf("InsertRide: %v", err)
	}

	recs, err := s.GetRecords(store.RecordsFilters{})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	// Non-nullable fields should be populated.
	if recs.LongestDistanceM == nil {
		t.Error("expected LongestDistanceM to be set")
	}
	if recs.LongestDurationS == nil {
		t.Error("expected LongestDurationS to be set")
	}
	if recs.MostElevationGainM == nil {
		t.Error("expected MostElevationGainM to be set")
	}
	if recs.HighestAvgSpeedMPS == nil {
		t.Error("expected HighestAvgSpeedMPS to be set")
	}
	if recs.HighestMaxSpeedMPS == nil {
		t.Error("expected HighestMaxSpeedMPS to be set")
	}

	// Nullable fields should be nil.
	if recs.HighestAvgPowerW != nil {
		t.Errorf("expected nil HighestAvgPowerW, got %+v", recs.HighestAvgPowerW)
	}
	if recs.HighestAvgHRBPM != nil {
		t.Errorf("expected nil HighestAvgHRBPM, got %+v", recs.HighestAvgHRBPM)
	}
	if recs.MostCaloriesKcal != nil {
		t.Errorf("expected nil MostCaloriesKcal, got %+v", recs.MostCaloriesKcal)
	}
	if recs.HighestAltitudeM != nil {
		t.Errorf("expected nil HighestAltitudeM, got %+v", recs.HighestAltitudeM)
	}
}

func TestGetRecords_YearFilter(t *testing.T) {
	s := openTestStore(t)

	// 2024 ride: farther
	if _, err := s.InsertRide(parser.Ride{
		Filename:     "2024.gpx",
		RecordedAt:   time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		DistanceM:    100000,
		DurationS:    3600,
		SourceFormat: "gpx",
	}); err != nil {
		t.Fatalf("insert 2024: %v", err)
	}
	// 2025 ride: shorter
	if _, err := s.InsertRide(parser.Ride{
		Filename:     "2025.gpx",
		RecordedAt:   time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		DistanceM:    30000,
		DurationS:    1800,
		SourceFormat: "gpx",
	}); err != nil {
		t.Fatalf("insert 2025: %v", err)
	}

	year := 2025
	recs, err := s.GetRecords(store.RecordsFilters{Year: &year})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if recs.LongestDistanceM == nil {
		t.Fatal("expected LongestDistanceM to be set")
	}
	if recs.LongestDistanceM.RawValue != 30000 {
		t.Errorf("LongestDistanceM: got %v, want 30000", recs.LongestDistanceM.RawValue)
	}
	if recs.LongestDistanceM.Date.Year() != 2025 {
		t.Errorf("LongestDistanceM.Date.Year: got %d, want 2025", recs.LongestDistanceM.Date.Year())
	}
}

func TestGetRecords_PicksMax(t *testing.T) {
	s := openTestStore(t)

	dateA := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	dateB := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)

	// Ride A: longer distance, slower speed.
	if _, err := s.InsertRide(parser.Ride{
		Filename:       "ride_a.gpx",
		RecordedAt:     dateA,
		DistanceM:      100000,
		DurationS:      3600,
		ElevationGainM: 500,
		AvgSpeedMPS:    5.0,
		MaxSpeedMPS:    10.0,
		SourceFormat:   "gpx",
	}); err != nil {
		t.Fatalf("insert ride A: %v", err)
	}
	// Ride B: shorter distance, faster speed.
	if _, err := s.InsertRide(parser.Ride{
		Filename:       "ride_b.gpx",
		RecordedAt:     dateB,
		DistanceM:      50000,
		DurationS:      7200,
		ElevationGainM: 200,
		AvgSpeedMPS:    20.0,
		MaxSpeedMPS:    40.0,
		SourceFormat:   "gpx",
	}); err != nil {
		t.Fatalf("insert ride B: %v", err)
	}

	recs, err := s.GetRecords(store.RecordsFilters{})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	// Distance record must come from ride A.
	if recs.LongestDistanceM == nil {
		t.Fatal("expected LongestDistanceM to be set")
	}
	if recs.LongestDistanceM.RawValue != 100000 {
		t.Errorf("LongestDistanceM: got %v, want 100000", recs.LongestDistanceM.RawValue)
	}
	if !recs.LongestDistanceM.Date.Equal(dateA) {
		t.Errorf("LongestDistanceM.Date: got %v, want %v", recs.LongestDistanceM.Date, dateA)
	}

	// Speed record must come from ride B.
	if recs.HighestAvgSpeedMPS == nil {
		t.Fatal("expected HighestAvgSpeedMPS to be set")
	}
	if recs.HighestAvgSpeedMPS.RawValue != 20.0 {
		t.Errorf("HighestAvgSpeedMPS: got %v, want 20.0", recs.HighestAvgSpeedMPS.RawValue)
	}
	if !recs.HighestAvgSpeedMPS.Date.Equal(dateB) {
		t.Errorf("HighestAvgSpeedMPS.Date: got %v, want %v", recs.HighestAvgSpeedMPS.Date, dateB)
	}
}

func TestListRides_FromFilter(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "mar.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "jun.gpx", RecordedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, total, err := s.ListRides(store.RideFilters{From: &from, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2", total)
	}
	if len(result) != 2 {
		t.Errorf("len: got %d, want 2", len(result))
	}
	for _, r := range result {
		if r.RecordedAt.Before(from) {
			t.Errorf("ride %q has date %v before --from %v", r.Filename, r.RecordedAt, from)
		}
	}
}

func TestListRides_ToFilter(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "mar.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "jun.gpx", RecordedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	to := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	result, total, err := s.ListRides(store.RideFilters{To: &to, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2", total)
	}
	if len(result) != 2 {
		t.Errorf("len: got %d, want 2", len(result))
	}
}

func TestListRides_FromToFilter(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "mar.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "jun.gpx", RecordedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "dec.gpx", RecordedAt: time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	result, total, err := s.ListRides(store.RideFilters{From: &from, To: &to, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2 (mar+jun)", total)
	}
	if len(result) != 2 {
		t.Errorf("len: got %d, want 2", len(result))
	}
}

func TestListRides_ToFilter_Inclusive(t *testing.T) {
	s := openTestStore(t)

	// A ride recorded exactly on the --to date must be included.
	boundary := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	rides := []parser.Ride{
		{Filename: "on_boundary.gpx", RecordedAt: boundary, SourceFormat: "gpx"},
		{Filename: "after_boundary.gpx", RecordedAt: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	to := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	result, total, err := s.ListRides(store.RideFilters{To: &to, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 1 {
		t.Errorf("total: got %d, want 1 (boundary date must be included)", total)
	}
	if len(result) == 0 || result[0].Filename != "on_boundary.gpx" {
		t.Errorf("expected on_boundary.gpx to be included")
	}
}
