package parser_test

import (
	"os"
	"testing"

	"github.com/HGClarke/paceline/internal/parser"
)

func TestParseTCX(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample.tcx")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	ride, streams, err := parser.ParseTCX("sample.tcx", data)
	if err != nil {
		t.Fatalf("ParseTCX: %v", err)
	}

	if ride.SourceFormat != "tcx" {
		t.Errorf("source_format: got %q want %q", ride.SourceFormat, "tcx")
	}
	if ride.DurationS != 3600 {
		t.Errorf("duration: got %d want 3600", ride.DurationS)
	}
	if ride.DistanceM != 30000 {
		t.Errorf("distance: got %v want 30000", ride.DistanceM)
	}
	if ride.Calories == nil || *ride.Calories != 500 {
		t.Errorf("calories: got %v want 500", ride.Calories)
	}
	if len(streams) != 2 {
		t.Errorf("expected 2 stream points, got %d", len(streams))
	}
	if streams[0].PowerW == nil || *streams[0].PowerW != 210 {
		t.Errorf("power: got %v want 210", streams[0].PowerW)
	}
}
