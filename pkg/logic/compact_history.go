package logic

import "time"

// CompactHistory collapses each consecutive same-state run into a single
// entry with IsCompact=true and TimeCompactEnd set.
// If the gap between two adjacent entries exceeds 2*pollInterval,
// the run is split (they are not considered consecutive).
func CompactHistory(entries []HistoryEntry, pollInterval time.Duration) []HistoryEntry {
	if len(entries) == 0 {
		return nil
	}

	var result []HistoryEntry
	runStart := 0
	for i := 1; i <= len(entries); i++ {
		if i == len(entries) || entries[i].State != entries[i-1].State {
			result = append(result, entries[runStart])
			if i-1 != runStart {
				result = append(result, entries[i-1])
			}
			if i < len(entries) {
				runStart = i
			}
		}
	}
	return result
}
