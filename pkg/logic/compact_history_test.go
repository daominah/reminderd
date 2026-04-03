package logic

import "testing"

func TestCompactHistory_SingleEntry(t *testing.T) {
	// WHEN compacting a single entry
	entries := []HistoryEntry{
		{Time: "2026-04-03T09:00:00+07:00", State: Active},
	}
	result := CompactHistory(entries)

	// THEN it is kept as-is
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
}

func TestCompactHistory_TwoEntriesSameState(t *testing.T) {
	// WHEN compacting two entries with the same state
	entries := []HistoryEntry{
		{Time: "2026-04-03T09:00:00+07:00", State: Active},
		{Time: "2026-04-03T09:00:10+07:00", State: Active},
	}
	result := CompactHistory(entries)

	// THEN both are kept (first and last of the run)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].Time != "2026-04-03T09:00:00+07:00" {
		t.Errorf("expected first entry time 09:00:00, got %s", result[0].Time)
	}
	if result[1].Time != "2026-04-03T09:00:10+07:00" {
		t.Errorf("expected last entry time 09:00:10, got %s", result[1].Time)
	}
}

func TestCompactHistory_LongRunKeepsOnlyFirstAndLast(t *testing.T) {
	// GIVEN 6 consecutive ACTIVE entries (10s apart)
	entries := []HistoryEntry{
		{Time: "2026-04-03T09:00:00+07:00", State: Active},
		{Time: "2026-04-03T09:00:10+07:00", State: Active},
		{Time: "2026-04-03T09:00:20+07:00", State: Active},
		{Time: "2026-04-03T09:00:30+07:00", State: Active},
		{Time: "2026-04-03T09:00:40+07:00", State: Active},
		{Time: "2026-04-03T09:00:50+07:00", State: Active},
	}

	// WHEN compacting
	result := CompactHistory(entries)

	// THEN only the first and last are kept
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].Time != "2026-04-03T09:00:00+07:00" {
		t.Errorf("first: expected 09:00:00, got %s", result[0].Time)
	}
	if result[1].Time != "2026-04-03T09:00:50+07:00" {
		t.Errorf("last: expected 09:00:50, got %s", result[1].Time)
	}
}

func TestCompactHistory_AlternatingStates(t *testing.T) {
	// GIVEN entries that alternate every entry
	entries := []HistoryEntry{
		{Time: "2026-04-03T09:00:00+07:00", State: Active},
		{Time: "2026-04-03T09:00:10+07:00", State: Idle},
		{Time: "2026-04-03T09:00:20+07:00", State: Active},
	}

	// WHEN compacting
	result := CompactHistory(entries)

	// THEN nothing is removed (each run has length 1)
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
}

func TestCompactHistory_MultipleRuns(t *testing.T) {
	// GIVEN 5 ACTIVE, 3 IDLE, 2 ACTIVE entries
	entries := []HistoryEntry{
		{Time: "2026-04-03T09:00:00+07:00", State: Active},
		{Time: "2026-04-03T09:00:10+07:00", State: Active},
		{Time: "2026-04-03T09:00:20+07:00", State: Active},
		{Time: "2026-04-03T09:00:30+07:00", State: Active},
		{Time: "2026-04-03T09:00:40+07:00", State: Active},
		{Time: "2026-04-03T09:00:50+07:00", State: Idle},
		{Time: "2026-04-03T09:01:00+07:00", State: Idle},
		{Time: "2026-04-03T09:01:10+07:00", State: Idle},
		{Time: "2026-04-03T09:01:20+07:00", State: Active},
		{Time: "2026-04-03T09:01:30+07:00", State: Active},
	}

	// WHEN compacting
	result := CompactHistory(entries)

	// THEN each run keeps first and last: 2 + 2 + 2 = 6 entries
	if len(result) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(result))
	}
	// First ACTIVE run: 09:00:00, 09:00:40
	if result[0].Time != "2026-04-03T09:00:00+07:00" || result[0].State != Active {
		t.Errorf("entry 0: got %s %s", result[0].Time, result[0].State)
	}
	if result[1].Time != "2026-04-03T09:00:40+07:00" || result[1].State != Active {
		t.Errorf("entry 1: got %s %s", result[1].Time, result[1].State)
	}
	// IDLE run: 09:00:50, 09:01:10
	if result[2].Time != "2026-04-03T09:00:50+07:00" || result[2].State != Idle {
		t.Errorf("entry 2: got %s %s", result[2].Time, result[2].State)
	}
	if result[3].Time != "2026-04-03T09:01:10+07:00" || result[3].State != Idle {
		t.Errorf("entry 3: got %s %s", result[3].Time, result[3].State)
	}
	// Second ACTIVE run: 09:01:20, 09:01:30
	if result[4].Time != "2026-04-03T09:01:20+07:00" || result[4].State != Active {
		t.Errorf("entry 4: got %s %s", result[4].Time, result[4].State)
	}
	if result[5].Time != "2026-04-03T09:01:30+07:00" || result[5].State != Active {
		t.Errorf("entry 5: got %s %s", result[5].Time, result[5].State)
	}
}

func TestCompactHistory_GapWithIdleEntry_KeepsIdleBoundary(t *testing.T) {
	// GIVEN ACTIVE entries with a 1h35m gap, and an IDLE entry on resume
	// (bug_reminderd.txt scenario: process stopped, on resume the OS
	// reported idle time, then user became active again)
	entries := []HistoryEntry{
		{Time: "2026-04-03T18:40:46+07:00", State: Active},
		{Time: "2026-04-03T18:40:56+07:00", State: Active},
		{Time: "2026-04-03T18:41:06+07:00", State: Active},
		{Time: "2026-04-03T20:15:50+07:00", State: Idle},
		{Time: "2026-04-03T20:15:56+07:00", State: Active},
		{Time: "2026-04-03T20:16:06+07:00", State: Active},
		{Time: "2026-04-03T20:16:16+07:00", State: Active},
	}

	// WHEN compacting
	result := CompactHistory(entries)

	// THEN 3 runs are preserved: ACTIVE(first,last), IDLE(single), ACTIVE(first,last)
	// = 2 + 1 + 2 = 5 entries.
	// The IDLE entry survives, but the 1h35m gap before it is still
	// counted as ACTIVE by the state-boundary model.
	if len(result) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(result))
	}
	if result[0].Time != "2026-04-03T18:40:46+07:00" || result[0].State != Active {
		t.Errorf("entry 0: got %s %s", result[0].Time, result[0].State)
	}
	if result[1].Time != "2026-04-03T18:41:06+07:00" || result[1].State != Active {
		t.Errorf("entry 1: got %s %s", result[1].Time, result[1].State)
	}
	if result[2].Time != "2026-04-03T20:15:50+07:00" || result[2].State != Idle {
		t.Errorf("entry 2: got %s %s", result[2].Time, result[2].State)
	}
	if result[3].Time != "2026-04-03T20:15:56+07:00" || result[3].State != Active {
		t.Errorf("entry 3: got %s %s", result[3].Time, result[3].State)
	}
	if result[4].Time != "2026-04-03T20:16:16+07:00" || result[4].State != Active {
		t.Errorf("entry 4: got %s %s", result[4].Time, result[4].State)
	}
}

func TestCompactHistory_GapWithinSameState_MergesIntoOneRun(t *testing.T) {
	// GIVEN ACTIVE entries with a 1h35m gap (process was stopped),
	// but compact only looks at State, not timestamps
	entries := []HistoryEntry{
		{Time: "2026-04-03T18:40:46+07:00", State: Active},
		{Time: "2026-04-03T18:40:56+07:00", State: Active},
		{Time: "2026-04-03T18:41:06+07:00", State: Active},
		{Time: "2026-04-03T20:15:56+07:00", State: Active},
		{Time: "2026-04-03T20:16:06+07:00", State: Active},
		{Time: "2026-04-03T20:16:16+07:00", State: Active},
	}

	// WHEN compacting
	result := CompactHistory(entries)

	// THEN compact treats this as one ACTIVE run, keeping only first and last.
	// The 1h35m gap is lost: indistinguishable from genuinely continuous activity.
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].Time != "2026-04-03T18:40:46+07:00" {
		t.Errorf("first: got %s", result[0].Time)
	}
	if result[1].Time != "2026-04-03T20:16:16+07:00" {
		t.Errorf("last: got %s", result[1].Time)
	}
}
