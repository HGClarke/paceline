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

func TestGetPowerCurve_NoPower(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "pc_no_power.gpx")

	hr := 140
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, HRBPM: &hr},
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	curve, err := s.GetPowerCurve(rideID)
	if err != nil {
		t.Fatalf("GetPowerCurve: %v", err)
	}
	if len(curve.Table) != 0 || len(curve.Chart) != 0 {
		t.Errorf("expected empty curve, got table=%d chart=%d", len(curve.Table), len(curve.Chart))
	}
}

func TestGetPowerCurve_Basic(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "pc_basic.gpx")

	// Build a 300-second stream with varying power so MMP has something to find.
	streams := make([]parser.Stream, 300)
	for i := range streams {
		p := 100 + i // power increases over time
		streams[i] = parser.Stream{
			RideID:    rideID,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			ElapsedS:  i,
			PowerW:    &p,
		}
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	curve, err := s.GetPowerCurve(rideID)
	if err != nil {
		t.Fatalf("GetPowerCurve: %v", err)
	}

	// Table should contain entries for 5s, 30s, 1min, 5min (durations ≤ 299s).
	if len(curve.Table) == 0 {
		t.Fatal("expected non-empty table")
	}

	// 5-second MMP should be the highest: last 5 points have power 395..399, avg ~397.
	if curve.Table[0].DurationS != 5 {
		t.Errorf("first table entry duration = %d, want 5", curve.Table[0].DurationS)
	}
	if curve.Table[0].PowerW < curve.Table[len(curve.Table)-1].PowerW {
		t.Errorf("power curve should be non-increasing: 5s=%d, last=%d",
			curve.Table[0].PowerW, curve.Table[len(curve.Table)-1].PowerW)
	}

	// 60min duration (3600s) must not appear since the ride is only 299s.
	for _, pt := range curve.Table {
		if pt.DurationS == 3600 {
			t.Errorf("60min entry should not appear for a 299s ride")
		}
	}

	// Chart should be non-empty and monotonically non-increasing.
	if len(curve.Chart) == 0 {
		t.Error("expected non-empty chart")
	}
	for i := 1; i < len(curve.Chart); i++ {
		if curve.Chart[i].PowerW > curve.Chart[i-1].PowerW {
			t.Errorf("chart power increased at index %d: %d > %d",
				i, curve.Chart[i].PowerW, curve.Chart[i-1].PowerW)
		}
	}
}

func TestGetPowerCurve_ConstantPower(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "pc_constant.gpx")

	const constPower = 250
	streams := make([]parser.Stream, 3600)
	for i := range streams {
		p := constPower
		streams[i] = parser.Stream{
			RideID:    rideID,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			ElapsedS:  i,
			PowerW:    &p,
		}
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	curve, err := s.GetPowerCurve(rideID)
	if err != nil {
		t.Fatalf("GetPowerCurve: %v", err)
	}

	for _, pt := range curve.Table {
		if pt.PowerW != constPower {
			t.Errorf("constant power: duration %ds expected %dW, got %dW", pt.DurationS, constPower, pt.PowerW)
		}
	}
}

func TestGetGPSPoints_NoData(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "gps_no_data.gpx")
	speed := 5.0
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, SpeedMPS: &speed},
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	pts, err := s.GetGPSPoints(rideID)
	if err != nil {
		t.Fatalf("GetGPSPoints: %v", err)
	}
	if len(pts) != 0 {
		t.Errorf("expected empty points, got %d", len(pts))
	}
}

func TestGetGPSPoints_WithData(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "gps_with_data.gpx")
	// Insert rows out of order: elapsed_s=1 first, then elapsed_s=0
	// to verify ORDER BY elapsed_s ASC is actually sorting them.
	lat1 := 51.5080
	lon1 := -0.1270
	lat0 := 51.5074
	lon0 := -0.1278
	alt := 120.5
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now().Add(time.Second), ElapsedS: 1, Lat: &lat1, Lon: &lon1},
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, Lat: &lat0, Lon: &lon0, AltitudeM: &alt},
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	pts, err := s.GetGPSPoints(rideID)
	if err != nil {
		t.Fatalf("GetGPSPoints: %v", err)
	}
	if len(pts) != 2 {
		t.Errorf("expected 2 points, got %d", len(pts))
	}

	if pts[0].ElapsedS != 0 {
		t.Errorf("point[0].ElapsedS = %d, want 0", pts[0].ElapsedS)
	}
	if pts[0].Lat < 51.5073 || pts[0].Lat > 51.5075 {
		t.Errorf("point[0].Lat = %f, want ~51.5074", pts[0].Lat)
	}
	if pts[0].Lon < -0.1279 || pts[0].Lon > -0.1277 {
		t.Errorf("point[0].Lon = %f, want ~-0.1278", pts[0].Lon)
	}
	if pts[0].AltitudeM == nil {
		t.Error("point[0].AltitudeM should not be nil")
	} else if *pts[0].AltitudeM < 120.4 || *pts[0].AltitudeM > 120.6 {
		t.Errorf("point[0].AltitudeM = %f, want ~120.5", *pts[0].AltitudeM)
	}

	if pts[1].ElapsedS != 1 {
		t.Errorf("point[1].ElapsedS = %d, want 1", pts[1].ElapsedS)
	}
	if pts[1].AltitudeM != nil {
		t.Error("point[1].AltitudeM should be nil")
	}
}
