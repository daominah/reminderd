package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daominah/reminderd/pkg/base"
	"github.com/daominah/reminderd/pkg/logic"
)

func TestWriteEntry_AppendsToDailyFile(t *testing.T) {
	// GIVEN a file store in a temp directory
	dir := t.TempDir()
	store := NewFileStore(dir)
	defer store.Close()

	// WHEN two active entries are written
	t1 := time.Date(2026, 4, 2, 10, 0, 0, 0, base.VietnamTimezone)
	t2 := time.Date(2026, 4, 2, 10, 0, 10, 0, base.VietnamTimezone)
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(t1), State: logic.Active})
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(t2), State: logic.Active})

	// THEN the daily file exists with 2 lines
	path := filepath.Join(dir, "history-2026-04-02.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected daily file to exist: %v", err)
	}
	lines := countLines(data)
	if lines != 2 {
		t.Errorf("expected 2 lines, got %d", lines)
	}
}

func TestWriteEntry_RollsOverOnNewDay(t *testing.T) {
	// GIVEN a file store with entries from day 1
	dir := t.TempDir()
	store := NewFileStore(dir)
	defer store.Close()
	day1 := time.Date(2026, 4, 2, 23, 59, 50, 0, base.VietnamTimezone)
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(day1), State: logic.Active})

	// WHEN an entry is written on the next day
	day2 := time.Date(2026, 4, 3, 0, 0, 10, 0, base.VietnamTimezone)
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(day2), State: logic.Active})

	// THEN two separate daily files exist
	file1 := filepath.Join(dir, "history-2026-04-02.jsonl")
	file2 := filepath.Join(dir, "history-2026-04-03.jsonl")
	if _, err := os.Stat(file1); err != nil {
		t.Errorf("expected day 1 file to exist: %v", err)
	}
	if _, err := os.Stat(file2); err != nil {
		t.Errorf("expected day 2 file to exist: %v", err)
	}
}

func TestCompactPrevious_KeepsFirstAndLastOfRuns(t *testing.T) {
	// GIVEN a history file with redundant middle entries
	dir := t.TempDir()
	store := NewFileStore(dir)
	defer store.Close()
	start := time.Date(2026, 4, 2, 8, 0, 0, 0, base.VietnamTimezone)
	// Write 5 consecutive active entries
	for i := range 5 {
		store.WriteEntry(logic.HistoryEntry{
			Time:  logic.FormatTime(start.Add(time.Duration(i) * 10 * time.Second)),
			State: logic.Active,
		})
	}
	// Write 3 consecutive idle entries
	for i := range 3 {
		store.WriteEntry(logic.HistoryEntry{
			Time:  logic.FormatTime(start.Add(time.Duration(50+i*10) * time.Second)),
			State: logic.Idle,
		})
	}

	// WHEN compaction runs (simulate next day so previous = 2026-04-02)
	nextDay := time.Date(2026, 4, 3, 8, 0, 0, 0, base.VietnamTimezone)
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(nextDay), State: logic.Active})
	store.CompactPrevious()

	// THEN the 2026-04-02 file has 4 lines:
	// first active, last active, first idle, last idle
	path := filepath.Join(dir, "history-2026-04-02.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read compacted file: %v", err)
	}
	lines := countLines(data)
	if lines != 4 {
		t.Errorf("expected 4 lines after compaction, got %d", lines)
	}
}

func TestReadRange_ReturnsEntriesInRange(t *testing.T) {
	// GIVEN a file store with entries across two days
	dir := t.TempDir()
	store := NewFileStore(dir)
	defer store.Close()

	day1Morning := time.Date(2026, 4, 2, 8, 0, 0, 0, base.VietnamTimezone)
	day1Afternoon := time.Date(2026, 4, 2, 14, 0, 0, 0, base.VietnamTimezone)
	day2Morning := time.Date(2026, 4, 3, 9, 0, 0, 0, base.VietnamTimezone)

	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(day1Morning), State: logic.Active})
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(day1Afternoon), State: logic.Idle})
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(day2Morning), State: logic.Active})

	// WHEN querying a range that spans both days
	start := time.Date(2026, 4, 2, 0, 0, 0, 0, base.VietnamTimezone)
	end := time.Date(2026, 4, 3, 23, 59, 59, 0, base.VietnamTimezone)
	entries, err := store.ReadRange(start, &end)
	if err != nil {
		t.Fatalf("ReadRange error: %v", err)
	}

	// THEN all 3 entries are returned
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestReadRange_FiltersOutOfRange(t *testing.T) {
	// GIVEN a file store with entries at different times
	dir := t.TempDir()
	store := NewFileStore(dir)
	defer store.Close()

	morning := time.Date(2026, 4, 2, 8, 0, 0, 0, base.VietnamTimezone)
	afternoon := time.Date(2026, 4, 2, 14, 0, 0, 0, base.VietnamTimezone)
	evening := time.Date(2026, 4, 2, 20, 0, 0, 0, base.VietnamTimezone)

	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(morning), State: logic.Active})
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(afternoon), State: logic.Active})
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(evening), State: logic.Idle})

	// WHEN querying only the afternoon range
	start := time.Date(2026, 4, 2, 12, 0, 0, 0, base.VietnamTimezone)
	end := time.Date(2026, 4, 2, 16, 0, 0, 0, base.VietnamTimezone)
	entries, err := store.ReadRange(start, &end)
	if err != nil {
		t.Fatalf("ReadRange error: %v", err)
	}

	// THEN only the afternoon entry is returned
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestReadRange_NoEndReturnsAllFromStart(t *testing.T) {
	// GIVEN a file store with entries
	dir := t.TempDir()
	store := NewFileStore(dir)
	defer store.Close()

	t1 := time.Date(2026, 4, 2, 8, 0, 0, 0, base.VietnamTimezone)
	t2 := time.Date(2026, 4, 2, 14, 0, 0, 0, base.VietnamTimezone)

	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(t1), State: logic.Active})
	store.WriteEntry(logic.HistoryEntry{Time: logic.FormatTime(t2), State: logic.Idle})

	// WHEN querying with no end time
	start := time.Date(2026, 4, 2, 10, 0, 0, 0, base.VietnamTimezone)
	entries, err := store.ReadRange(start, nil)
	if err != nil {
		t.Fatalf("ReadRange error: %v", err)
	}

	// THEN entries from start onwards are returned
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (afternoon only), got %d", len(entries))
	}
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	return count
}
