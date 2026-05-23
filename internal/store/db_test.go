package store_test

import (
	"testing"

	"github.com/hollandclarke/paceline/internal/store"
)

func TestOpen_CreatesSchema(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if err := s.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	for _, table := range []string{"rides", "streams"} {
		var n int
		err := s.DB().QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table,
		).Scan(&n)
		if err != nil {
			t.Fatalf("check table %q: %v", table, err)
		}
		if n == 0 {
			t.Errorf("table %q was not created", table)
		}
	}
}
