package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <file|directory>",
	Short: "Import ride file(s) into the database",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	path := args[0]
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %q: %w", path, err)
	}

	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	var files []string
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, filepath.Join(path, e.Name()))
			}
		}
	} else {
		files = []string{path}
	}

	var imported, skipped int
	var errs []string

	for _, f := range files {
		if err := importFile(s, f); err != nil {
			skipped++
			errs = append(errs, fmt.Sprintf("%s — %v", filepath.Base(f), err))
		} else {
			imported++
		}
	}

	fmt.Printf("%d imported, %d skipped\n", imported, skipped)
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "  skipped: %s\n", e)
	}
	return nil
}

func importFile(s *store.Store, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	filename := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	var ride *parser.Ride
	var streams []parser.Stream

	switch ext {
	case ".gpx":
		ride, streams, err = parser.ParseGPX(filename, data)
	case ".tcx":
		ride, streams, err = parser.ParseTCX(filename, data)
	case ".fit":
		ride, streams, err = parser.ParseFIT(filename, data)
	default:
		return fmt.Errorf("unsupported format %q", ext)
	}
	if err != nil {
		return err
	}

	id, err := s.InsertRide(*ride)
	if err != nil {
		return err
	}
	if id == 0 {
		fmt.Printf("  already imported: %s\n", filename)
		return nil
	}

	for i := range streams {
		streams[i].RideID = id
	}
	if err := s.InsertStreams(streams); err != nil {
		return fmt.Errorf("insert streams: %w", err)
	}

	fmt.Printf("  imported: %s (id=%d, %d points)\n", filename, id, len(streams))
	return nil
}
