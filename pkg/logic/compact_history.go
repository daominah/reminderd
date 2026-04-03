package logic

// CompactHistory keeps only the first and last entry of each consecutive state run.
func CompactHistory(entries []HistoryEntry) []HistoryEntry {
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
