package logic

import "testing"

func TestCompactHistory_SingleEntry(t *testing.T) {
	// WHEN compacting a single entry
	entries := []HistoryEntry{
		{Time: "2026-04-03T09:00:00+07:00", State: Active},
	}
	result := CompactHistory(entries, PollInterval)

	// THEN it is kept as-is, not marked as compact (no range to collapse)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result[0].IsCompact {
		t.Error("single entry should not be marked as compact")
	}
}

func TestCompactHistory_TwoEntriesSameState(t *testing.T) {
	// WHEN compacting two entries with the same state
	entries := []HistoryEntry{
		{Time: "2026-04-03T09:00:00+07:00", State: Active},
		{Time: "2026-04-03T09:00:10+07:00", State: Active},
	}
	result := CompactHistory(entries, PollInterval)

	// THEN they collapse into 1 compact entry with a time range
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if !result[0].IsCompact {
		t.Error("expected IsCompact=true")
	}
	if result[0].Time != "2026-04-03T09:00:00+07:00" {
		t.Errorf("expected Time 09:00:00, got %s", result[0].Time)
	}
	if result[0].TimeCompactEnd != "2026-04-03T09:00:10+07:00" {
		t.Errorf("expected TimeCompactEnd 09:00:10, got %s", result[0].TimeCompactEnd)
	}
}

func TestCompactHistory_LongRunKeepsOneCompactEntry(t *testing.T) {
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
	result := CompactHistory(entries, PollInterval)

	// THEN 1 compact entry spanning the full run
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if !result[0].IsCompact {
		t.Error("expected IsCompact=true")
	}
	if result[0].Time != "2026-04-03T09:00:00+07:00" {
		t.Errorf("Time: expected 09:00:00, got %s", result[0].Time)
	}
	if result[0].TimeCompactEnd != "2026-04-03T09:00:50+07:00" {
		t.Errorf("TimeCompactEnd: expected 09:00:50, got %s", result[0].TimeCompactEnd)
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
	result := CompactHistory(entries, PollInterval)

	// THEN nothing is removed (each run is length 1, not compact)
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	for i, e := range result {
		if e.IsCompact {
			t.Errorf("entry %d: should not be compact", i)
		}
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
	result := CompactHistory(entries, PollInterval)

	// THEN 3 compact entries (one per run)
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	// First ACTIVE run: 09:00:00 to 09:00:40
	if result[0].State != Active || !result[0].IsCompact {
		t.Errorf("entry 0: expected ACTIVE compact, got %s IsCompact=%v", result[0].State, result[0].IsCompact)
	}
	if result[0].Time != "2026-04-03T09:00:00+07:00" || result[0].TimeCompactEnd != "2026-04-03T09:00:40+07:00" {
		t.Errorf("entry 0: got %s to %s", result[0].Time, result[0].TimeCompactEnd)
	}

	// IDLE run: 09:00:50 to 09:01:10
	if result[1].State != Idle || !result[1].IsCompact {
		t.Errorf("entry 1: expected IDLE compact, got %s IsCompact=%v", result[1].State, result[1].IsCompact)
	}
	if result[1].Time != "2026-04-03T09:00:50+07:00" || result[1].TimeCompactEnd != "2026-04-03T09:01:10+07:00" {
		t.Errorf("entry 1: got %s to %s", result[1].Time, result[1].TimeCompactEnd)
	}

	// Second ACTIVE run: 09:01:20 to 09:01:30
	if result[2].State != Active || !result[2].IsCompact {
		t.Errorf("entry 2: expected ACTIVE compact, got %s IsCompact=%v", result[2].State, result[2].IsCompact)
	}
	if result[2].Time != "2026-04-03T09:01:20+07:00" || result[2].TimeCompactEnd != "2026-04-03T09:01:30+07:00" {
		t.Errorf("entry 2: got %s to %s", result[2].Time, result[2].TimeCompactEnd)
	}
}

func TestCompactHistory_GapWithIdleEntry(t *testing.T) {
	// GIVEN ACTIVE entries with a 1h35m gap, and an IDLE entry on resume
	// (bug_reminderd.txt scenario)
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
	result := CompactHistory(entries, PollInterval)

	// THEN 3 entries: ACTIVE(compact), IDLE(single), ACTIVE(compact)
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	// First ACTIVE run: 18:40:46 to 18:41:06 (only 20s of real activity)
	if !result[0].IsCompact || result[0].State != Active {
		t.Errorf("entry 0: expected ACTIVE compact")
	}
	if result[0].Time != "2026-04-03T18:40:46+07:00" || result[0].TimeCompactEnd != "2026-04-03T18:41:06+07:00" {
		t.Errorf("entry 0: got %s to %s", result[0].Time, result[0].TimeCompactEnd)
	}

	// IDLE: single entry, not compact
	if result[1].IsCompact || result[1].State != Idle {
		t.Errorf("entry 1: expected IDLE non-compact")
	}
	if result[1].Time != "2026-04-03T20:15:50+07:00" {
		t.Errorf("entry 1: got %s", result[1].Time)
	}

	// Second ACTIVE run: 20:15:56 to 20:16:16
	if !result[2].IsCompact || result[2].State != Active {
		t.Errorf("entry 2: expected ACTIVE compact")
	}
	if result[2].Time != "2026-04-03T20:15:56+07:00" || result[2].TimeCompactEnd != "2026-04-03T20:16:16+07:00" {
		t.Errorf("entry 2: got %s to %s", result[2].Time, result[2].TimeCompactEnd)
	}
}

func TestCompactHistory_GapWithinSameState(t *testing.T) {
	// GIVEN ACTIVE entries with a 1h35m gap (bug_reminderd2.txt scenario)
	entries := []HistoryEntry{
		{Time: "2026-04-03T18:40:46+07:00", State: Active},
		{Time: "2026-04-03T18:40:56+07:00", State: Active},
		{Time: "2026-04-03T18:41:06+07:00", State: Active},
		{Time: "2026-04-03T20:15:56+07:00", State: Active},
		{Time: "2026-04-03T20:16:06+07:00", State: Active},
		{Time: "2026-04-03T20:16:16+07:00", State: Active},
	}

	// WHEN compacting
	result := CompactHistory(entries, PollInterval)

	// THEN 2 compact entries: the 1h35m gap splits the run
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	// First session: 18:40:46 to 18:41:06
	if !result[0].IsCompact || result[0].State != Active {
		t.Errorf("entry 0: expected ACTIVE compact")
	}
	if result[0].Time != "2026-04-03T18:40:46+07:00" || result[0].TimeCompactEnd != "2026-04-03T18:41:06+07:00" {
		t.Errorf("entry 0: got %s to %s", result[0].Time, result[0].TimeCompactEnd)
	}
	// Second session: 20:15:56 to 20:16:16
	if !result[1].IsCompact || result[1].State != Active {
		t.Errorf("entry 1: expected ACTIVE compact")
	}
	if result[1].Time != "2026-04-03T20:15:56+07:00" || result[1].TimeCompactEnd != "2026-04-03T20:16:16+07:00" {
		t.Errorf("entry 1: got %s to %s", result[1].Time, result[1].TimeCompactEnd)
	}
}
