package parser_test

import (
	"os"
	"testing"

	"github.com/HGClarke/paceline/internal/parser"
)

func TestParseGPX(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample.gpx")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	ride, streams, err := parser.ParseGPX("sample.gpx", data)
	if err != nil {
		t.Fatalf("ParseGPX: %v", err)
	}

	if ride.SourceFormat != "gpx" {
		t.Errorf("source_format: got %q want %q", ride.SourceFormat, "gpx")
	}
	if ride.Filename != "sample.gpx" {
		t.Errorf("filename: got %q want %q", ride.Filename, "sample.gpx")
	}
	if len(streams) != 3 {
		t.Errorf("expected 3 stream points, got %d", len(streams))
	}
	if ride.RecordedAt.IsZero() {
		t.Error("RecordedAt is zero")
	}
	if ride.DistanceM <= 0 {
		t.Errorf("expected positive distance, got %v", ride.DistanceM)
	}
}
