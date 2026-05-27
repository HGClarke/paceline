package parser_test

import (
	"os"
	"testing"

	"github.com/HGClarke/paceline/internal/parser"
)

func TestParseFIT(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample.fit")
	if err != nil {
		t.Skip("testdata/sample.fit not present — copy a .fit file to run this test")
	}

	ride, streams, err := parser.ParseFIT("sample.fit", data)
	if err != nil {
		t.Fatalf("ParseFIT: %v", err)
	}

	if ride.SourceFormat != "fit" {
		t.Errorf("source_format: got %q want fit", ride.SourceFormat)
	}
	if ride.RecordedAt.IsZero() {
		t.Error("RecordedAt is zero")
	}
	if len(streams) == 0 {
		t.Error("expected stream points, got none")
	}
}
