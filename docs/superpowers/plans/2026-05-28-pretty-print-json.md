# Pretty-Print JSON Output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace compact single-line JSON output with human-readable indented JSON across all `--json` output paths.

**Architecture:** All JSON output lives in `internal/display/table.go`. Four functions use `json.NewEncoder(w).Encode(x)` — each gets replaced with `json.MarshalIndent` + `fmt.Fprintln`. Tests are added/updated in `internal/display/display_test.go` to assert on indented output.

**Tech Stack:** Go standard library (`encoding/json`, `fmt`).

---

### Task 1: Write failing tests for indented JSON output

**Files:**
- Modify: `internal/display/display_test.go`

- [ ] **Step 1: Add indentation assertion to the existing `TestPrintRecords_JSON`**

Open `internal/display/display_test.go`. At the end of `TestPrintRecords_JSON` (currently ends at line 289), add one more assertion inside the function body before the closing `}`:

```go
if !strings.Contains(output, "\n  ") {
    t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
}
```

The full function after the change:

```go
func TestPrintRecords_JSON(t *testing.T) {
	var buf bytes.Buffer
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	recs := store.Records{
		LongestDistanceM: &store.PersonalRecord{RawValue: 50000, Date: date},
	}
	PrintRecords(&buf, recs, "all time", true, "metric")
	output := buf.String()
	if !strings.Contains(output, `"longest_distance_m"`) {
		t.Errorf("expected JSON key 'longest_distance_m', got:\n%s", output)
	}
	if !strings.Contains(output, "50000") {
		t.Errorf("expected value 50000 in JSON, got:\n%s", output)
	}
	if !strings.Contains(output, "null") {
		t.Errorf("expected null fields in JSON for absent records, got:\n%s", output)
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}
```

- [ ] **Step 2: Add `TestPrintRideList_JSON` after `TestPrintRideList_Imperial`**

Append after `TestPrintRideList_Imperial` (currently ends ~line 121):

```go
func TestPrintRideList_JSON(t *testing.T) {
	var buf bytes.Buffer
	rides := []parser.Ride{
		{
			Position:    1,
			RecordedAt:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			DistanceM:   30000,
			DurationS:   3600,
			AvgSpeedMPS: 8.3,
		},
	}
	PrintRideList(&buf, rides, 1, 1, 10, true, "metric")
	output := buf.String()
	if !strings.Contains(output, `"distance_m"`) {
		t.Errorf("expected JSON key 'distance_m', got:\n%s", output)
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}
```

- [ ] **Step 3: Add `TestPrintRideDetail_JSON` after `TestPrintRideList_JSON`**

```go
func TestPrintRideDetail_JSON(t *testing.T) {
	var buf bytes.Buffer
	r := parser.Ride{
		RecordedAt:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		DistanceM:   50000,
		DurationS:   7200,
		AvgSpeedMPS: 6.9,
	}
	PrintRideDetail(&buf, r, true, "metric")
	output := buf.String()
	if !strings.Contains(output, `"distance_m"`) {
		t.Errorf("expected JSON key 'distance_m', got:\n%s", output)
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}
```

- [ ] **Step 4: Add `TestPrintStats_JSON` after `TestPrintRideDetail_JSON`**

```go
func TestPrintStats_JSON(t *testing.T) {
	var buf bytes.Buffer
	st := store.Stats{
		RideCount:      5,
		TotalDistanceM: 100000,
		TotalDurationS: 18000,
	}
	PrintStats(&buf, st, "all time", true, "metric")
	output := buf.String()
	if !strings.Contains(output, `"RideCount"`) {
		t.Errorf("expected JSON key 'RideCount', got:\n%s", output)
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}
```

- [ ] **Step 5: Run the new tests to confirm they fail**

```bash
go test ./internal/display/... -run "TestPrintRideList_JSON|TestPrintRideDetail_JSON|TestPrintStats_JSON|TestPrintRecords_JSON" -v
```

Expected: the `\n  ` indentation assertions FAIL with output like:
```
--- FAIL: TestPrintRecords_JSON (0.00s)
    display_test.go:XXX: expected indented JSON (newline + 2 spaces), got:
    {"longest_distance_m":{"raw_value":50000,...}}
```

---

### Task 2: Replace `json.NewEncoder` with `json.MarshalIndent` in all 4 functions

**Files:**
- Modify: `internal/display/table.go`

- [ ] **Step 1: Replace JSON output in `PrintRideList`**

In `internal/display/table.go`, find the `PrintRideList` JSON block (lines 18–21):

```go
if jsonOut {
    _ = json.NewEncoder(w).Encode(rides) // write error on stdout is unrecoverable
    return
}
```

Replace with:

```go
if jsonOut {
    b, _ := json.MarshalIndent(rides, "", "  ")
    fmt.Fprintln(w, string(b))
    return
}
```

- [ ] **Step 2: Replace JSON output in `PrintRideDetail`**

Find the `PrintRideDetail` JSON block (lines 48–51):

```go
if jsonOut {
    _ = json.NewEncoder(w).Encode(r) // write error on stdout is unrecoverable
    return
}
```

Replace with:

```go
if jsonOut {
    b, _ := json.MarshalIndent(r, "", "  ")
    fmt.Fprintln(w, string(b))
    return
}
```

- [ ] **Step 3: Replace JSON output in `PrintStats`**

Find the `PrintStats` JSON block (lines 89–92):

```go
if jsonOut {
    _ = json.NewEncoder(w).Encode(st) // write error on stdout is unrecoverable
    return
}
```

Replace with:

```go
if jsonOut {
    b, _ := json.MarshalIndent(st, "", "  ")
    fmt.Fprintln(w, string(b))
    return
}
```

- [ ] **Step 4: Replace JSON output in `PrintRecords`**

Find the `PrintRecords` JSON block (lines 114–117):

```go
if jsonOut {
    _ = json.NewEncoder(w).Encode(recs) // write error on stdout is unrecoverable
    return
}
```

Replace with:

```go
if jsonOut {
    b, _ := json.MarshalIndent(recs, "", "  ")
    fmt.Fprintln(w, string(b))
    return
}
```

- [ ] **Step 5: Remove the now-unused `json` import alias if needed — verify the import is still used**

`json.MarshalIndent` still uses `"encoding/json"`, so no import changes are needed.

- [ ] **Step 6: Run the JSON tests to confirm they now pass**

```bash
go test ./internal/display/... -run "TestPrintRideList_JSON|TestPrintRideDetail_JSON|TestPrintStats_JSON|TestPrintRecords_JSON" -v
```

Expected:
```
--- PASS: TestPrintRideList_JSON (0.00s)
--- PASS: TestPrintRideDetail_JSON (0.00s)
--- PASS: TestPrintStats_JSON (0.00s)
--- PASS: TestPrintRecords_JSON (0.00s)
PASS
```

- [ ] **Step 7: Run the full test suite and linter**

```bash
make all
```

Expected: all tests pass, vet clean, lint clean.

- [ ] **Step 8: Commit**

```bash
git add internal/display/table.go internal/display/display_test.go
git commit -m "feat: pretty-print JSON output with 2-space indentation"
```
