package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var importNoRecursive bool
var importDryRun bool

var importCmd = &cobra.Command{
	Use:   "import <file|directory>",
	Short: "Import ride file(s) into the database",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().BoolVar(&importNoRecursive, "no-recursive", false, "do not recurse into subdirectories")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "show what would be imported without importing")
}

var errAlreadyExists = errors.New("already exists")

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
		files, err = collectFiles(path, !importNoRecursive)
		if err != nil {
			return err
		}
		if importDryRun {
			return runDryRun(s, files)
		}
		fmt.Printf("Importing %d files...\n", len(files))
	} else {
		if importDryRun {
			return runDryRun(s, []string{path})
		}
		files = []string{path}
	}

	var imported, alreadyExist, errs int
	for _, f := range files {
		switch err := importFile(s, f); {
		case err == nil:
			imported++
		case errors.Is(err, errAlreadyExists):
			alreadyExist++
		default:
			errs++
			fmt.Fprintf(os.Stderr, "  error: %s — %v\n", filepath.Base(f), err)
		}
	}

	fmt.Printf("Done: %d imported, %d already exist, %d errors\n", imported, alreadyExist, errs)
	return nil
}

func runDryRun(s *store.Store, files []string) error {
	fmt.Println("Dry run — no files will be imported.")
	var newCount, existCount int
	for _, f := range files {
		exists, err := s.FileExists(filepath.Base(f))
		if err != nil {
			return fmt.Errorf("check %s: %w", filepath.Base(f), err)
		}
		if exists {
			existCount++
		} else {
			newCount++
		}
	}
	fmt.Printf("Found %d files (%d new, %d already exist)\n", len(files), newCount, existCount)
	return nil
}

func collectFiles(root string, recursive bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == ".fit" || ext == ".gpx" || ext == ".tcx" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
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
		return errAlreadyExists
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
