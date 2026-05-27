package store_test

import (
	"testing"
	"time"

	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
)

func insertTestRide(t *testing.T, s *store.Store, filename string) int64 {
	t.Helper()
	id, err := s.InsertRide(parser.Ride{
		Filename:     filename,
		RecordedAt:   time.Now(),
		SourceFormat: "gpx",
	})
	if err != nil {
		t.Fatalf("insertTestRide: %v", err)
	}
	return id
}

func TestInsertAndGetStreams(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "stream_test.gpx")

	power := 200
	hr := 150
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, PowerW: &power, HRBPM: &hr},
		{RideID: rideID, Timestamp: time.Now().Add(time.Second), ElapsedS: 1, PowerW: &power, HRBPM: &hr},
	}

	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	got, err := s.GetStreams(rideID, "power")
	if err != nil {
		t.Fatalf("GetStreams: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 streams, got %d", len(got))
	}
	if got[0].PowerW == nil || *got[0].PowerW != 200 {
		t.Errorf("expected power 200, got %v", got[0].PowerW)
	}
}

func TestGetStreams_FieldFallback(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "fallback_test.gpx")

	hr := 155
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, HRBPM: &hr},
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	// power requested but not present — should return empty
	got, err := s.GetStreams(rideID, "power")
	if err != nil {
		t.Fatalf("GetStreams power: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 power streams, got %d", len(got))
	}

	// hr is present
	got, err = s.GetStreams(rideID, "hr")
	if err != nil {
		t.Fatalf("GetStreams hr: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 hr stream, got %d", len(got))
	}
}

func TestAvailableFields(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "fields_test.gpx")

	hr := 140
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, HRBPM: &hr},
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	fields, err := s.AvailableFields(rideID)
	if err != nil {
		t.Fatalf("AvailableFields: %v", err)
	}
	if len(fields) != 1 || fields[0] != "hr" {
		t.Errorf("expected [hr], got %v", fields)
	}
}
