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
// Pass ":memory:" for an in-memory database.
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
			distance_m    DOUBLE NOT NULL DEFAULT 0,
			duration_s    INTEGER NOT NULL DEFAULT 0,
			elevation_gain_m DOUBLE NOT NULL DEFAULT 0,
			avg_speed_mps DOUBLE NOT NULL DEFAULT 0,
			max_speed_mps DOUBLE NOT NULL DEFAULT 0,
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
			lon         DOUBLE,
			PRIMARY KEY (ride_id, elapsed_s)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			preview := stmt
			if len(preview) > 40 {
				preview = preview[:40]
			}
			return fmt.Errorf("exec %q: %w", preview, err)
		}
	}
	return nil
}

// DefaultPath returns ~/.paceline/data.db, creating the directory if needed.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	dir := filepath.Join(home, ".paceline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create .paceline dir: %w", err)
	}
	return filepath.Join(dir, "data.db"), nil
}

// DB returns the underlying sql.DB for testing purposes.
func (s *Store) DB() *sql.DB { return s.db }
