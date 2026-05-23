package store_test

import (
	"testing"

	"github.com/hollandclarke/paceline/internal/store"
)

func TestOpen_CreatesSchema(t *testing.T) {
	s, err := store.Open("")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if err := s.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
