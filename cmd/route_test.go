package cmd

import (
	"strings"
	"testing"

	"github.com/HGClarke/paceline/internal/store"
)

func TestBuildOSMURL_MultiPoint(t *testing.T) {
	pts := []store.GPSPoint{
		{Lat: 51.50, Lon: -0.13},
		{Lat: 51.51, Lon: -0.12},
		{Lat: 51.49, Lon: -0.14},
	}
	got := buildOSMURL(pts)
	want := "https://www.openstreetmap.org/?bbox=-0.140000,51.490000,-0.120000,51.510000"
	if got != want {
		t.Errorf("buildOSMURL = %q, want %q", got, want)
	}
}

func TestBuildOSMURL_Empty(t *testing.T) {
	got := buildOSMURL(nil)
	if got != "" {
		t.Errorf("buildOSMURL(nil) = %q, want empty string", got)
	}
}

func TestBuildOSMURL_SinglePoint(t *testing.T) {
	pts := []store.GPSPoint{{Lat: 51.50, Lon: -0.13}}
	got := buildOSMURL(pts)
	if !strings.Contains(got, "openstreetmap.org") {
		t.Errorf("buildOSMURL single point = %q, expected OSM URL", got)
	}
	if !strings.Contains(got, "51.500000") {
		t.Errorf("buildOSMURL single point = %q, expected lat 51.500000", got)
	}
	if !strings.Contains(got, "-0.130000") {
		t.Errorf("buildOSMURL single point = %q, expected lon -0.130000", got)
	}
}
