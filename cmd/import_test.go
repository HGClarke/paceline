package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCollectFiles_SingleLevel(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.fit"))
	mustWriteFile(t, filepath.Join(dir, "b.gpx"))
	mustWriteFile(t, filepath.Join(dir, "c.tcx"))
	mustWriteFile(t, filepath.Join(dir, "d.xlsx"))

	files, err := collectFiles(dir, true)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(files), files)
	}
}

func TestCollectFiles_Recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "root.fit"))
	mustWriteFile(t, filepath.Join(sub, "nested.gpx"))

	files, err := collectFiles(dir, true)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files (recursive), got %d: %v", len(files), files)
	}
}

func TestCollectFiles_NonRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "root.fit"))
	mustWriteFile(t, filepath.Join(sub, "nested.gpx"))

	files, err := collectFiles(dir, false)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (non-recursive), got %d: %v", len(files), files)
	}
	if filepath.Base(files[0]) != "root.fit" {
		t.Errorf("expected root.fit, got %q", filepath.Base(files[0]))
	}
}

func TestCollectFiles_SkipsHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "visible.fit"))
	mustWriteFile(t, filepath.Join(dir, ".hidden.fit"))

	files, err := collectFiles(dir, true)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (hidden skipped), got %d: %v", len(files), files)
	}
	if filepath.Base(files[0]) != "visible.fit" {
		t.Errorf("expected visible.fit, got %q", filepath.Base(files[0]))
	}
}

func TestCollectFiles_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "visible.fit"))
	mustWriteFile(t, filepath.Join(hiddenDir, "hidden.fit"))

	files, err := collectFiles(dir, true)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (hidden dir skipped), got %d: %v", len(files), files)
	}
}

func TestCollectFiles_CaseInsensitiveExtensions(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.FIT"))
	mustWriteFile(t, filepath.Join(dir, "b.GPX"))
	mustWriteFile(t, filepath.Join(dir, "c.TCX"))

	files, err := collectFiles(dir, true)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 files (case-insensitive), got %d: %v", len(files), files)
	}
}

func TestCollectFiles_UnsupportedExtensionSkipped(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "activity.fit"))
	mustWriteFile(t, filepath.Join(dir, "notes.txt"))
	mustWriteFile(t, filepath.Join(dir, "image.jpg"))

	files, err := collectFiles(dir, true)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(files), files)
	}
}

func TestCollectFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	files, err := collectFiles(dir, true)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestCollectFiles_DeepRecursion(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "top.fit"))
	mustWriteFile(t, filepath.Join(deep, "deep.gpx"))

	files, err := collectFiles(dir, true)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	sort.Strings(files)
	if len(files) != 2 {
		t.Errorf("expected 2 files (deep recursive), got %d: %v", len(files), files)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
