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

	gapThreshold := 2 * pollInterval

	var result []HistoryEntry
	runStart := 0
	for i := 1; i <= len(entries); i++ {
		sameRun := i < len(entries) &&
			entries[i].State == entries[i-1].State &&
			!isGap(entries[i-1].Time, entries[i].Time, gapThreshold)

		if !sameRun {
			// Emit the run from runStart to i-1.
			if runStart == i-1 {
				// Single entry run: keep as-is.
				result = append(result, entries[runStart])
			} else {
				// Multi-entry run: collapse into one compact entry.
				result = append(result, HistoryEntry{
					Time:           entries[runStart].Time,
					State:          entries[runStart].State,
					IsCompact:      true,
					TimeCompactEnd: entries[i-1].Time,
				})
			}
			runStart = i
		}
	}
	return result
}

func isGap(prev, next string, threshold time.Duration) bool {
	t1, err1 := ParseTime(prev)
	t2, err2 := ParseTime(next)
	if err1 != nil || err2 != nil {
		return false
	}
	return t2.Sub(t1) > threshold
}
