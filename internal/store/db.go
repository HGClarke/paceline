package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

type Store struct {
	db *sql.DB
}

// Open opens (or creates) a DuckDB database at path.
// Pass "" or ":memory:" for an in-memory database.
func Open(path string) (*Store, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping() error { return s.db.Ping() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS rides_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS rides (
			id            BIGINT  DEFAULT nextval('rides_id_seq') PRIMARY KEY,
			filename      TEXT    UNIQUE NOT NULL,
			recorded_at   TIMESTAMP NOT NULL,
			distance_m    DOUBLE,
			duration_s    INTEGER,
			elevation_gain_m DOUBLE,
			avg_speed_mps DOUBLE,
			max_speed_mps DOUBLE,
			avg_hr_bpm    INTEGER,
			max_hr_bpm    INTEGER,
			avg_power_w   INTEGER,
			max_power_w   INTEGER,
			calories      INTEGER,
			source_format TEXT    NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS streams (
			ride_id     BIGINT    NOT NULL REFERENCES rides(id),
			timestamp   TIMESTAMP NOT NULL,
			elapsed_s   INTEGER   NOT NULL,
			speed_mps   DOUBLE,
			hr_bpm      INTEGER,
			power_w     INTEGER,
			cadence_rpm INTEGER,
			altitude_m  DOUBLE,
			lat         DOUBLE,
			lon         DOUBLE
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:40], err)
		}
	}
	return nil
}

// DefaultPath returns ~/.paceline/data.db, creating the directory if needed.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".paceline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "data.db"), nil
}
