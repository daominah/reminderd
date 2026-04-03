package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/daominah/reminderd/pkg/base"
	"github.com/daominah/reminderd/pkg/logic"
)

// FileStore reads and writes history entries to daily JSONL files.
type FileStore struct {
	Dir         string
	currentDate string
	currentFile *os.File
}

func NewFileStore(dir string) *FileStore {
	return &FileStore{Dir: dir}
}

// Close closes the currently open file handle, if any.
func (s *FileStore) Close() error {
	if s.currentFile != nil {
		err := s.currentFile.Close()
		s.currentFile = nil
		return err
	}
	return nil
}

func dateKey(t time.Time) string {
	return t.In(base.VietnamTimezone).Format("2006-01-02")
}

func dateKeyFromString(s string) string {
	t, err := logic.ParseTime(s)
	if err != nil {
		return ""
	}
	return dateKey(t)
}

func filename(date string) string {
	return "history-" + date + ".jsonl"
}

// WriteEntry appends a history entry to the appropriate daily file.
func (s *FileStore) WriteEntry(e logic.HistoryEntry) error {
	date := dateKeyFromString(e.Time)
	// Rotate at midnight +07:00: when the entry's date differs
	// from the current file's date, close the old file.
	if date != s.currentDate {
		if s.currentFile != nil {
			s.currentFile.Close()
			s.currentFile = nil
		}
		s.currentDate = date
	}

	if s.currentFile == nil {
		path := filepath.Join(s.Dir, filename(date))
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("error os.OpenFile %s: %w", path, err)
		}
		s.currentFile = f
	}

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error json.Marshal: %w", err)
	}
	data = append(data, '\n')
	if _, err := s.currentFile.Write(data); err != nil {
		return fmt.Errorf("error writing entry: %w", err)
	}
	return nil
}

// CompactPrevious compacts the most recent history file before today.
// It keeps only the first and last entry of each consecutive state run.
func (s *FileStore) CompactPrevious() error {
	today := dateKey(time.Now().In(base.VietnamTimezone))
	files, err := filepath.Glob(filepath.Join(s.Dir, "history-*.jsonl"))
	if err != nil {
		return fmt.Errorf("error filepath.Glob: %w", err)
	}
	sort.Strings(files)

	// Find the most recent file that is not today.
	var target string
	for i := len(files) - 1; i >= 0; i-- {
		base := filepath.Base(files[i])
		date := strings.TrimPrefix(base, "history-")
		date = strings.TrimSuffix(date, ".jsonl")
		if date != today {
			target = files[i]
			break
		}
	}
	if target == "" {
		return nil
	}

	entries, err := readFile(target)
	if err != nil {
		return err
	}
	if len(entries) <= 2 {
		return nil
	}

	compacted := logic.CompactHistory(entries, logic.PollInterval)
	return writeFile(target, compacted)
}

// ReadRange returns history entries within the given time range.
// If end is nil, all entries from start onwards are returned.
func (s *FileStore) ReadRange(start time.Time, end *time.Time) ([]logic.HistoryEntry, error) {
	startDate := start.In(base.VietnamTimezone)
	var endDate time.Time
	if end != nil {
		endDate = end.In(base.VietnamTimezone)
	} else {
		endDate = time.Now().In(base.VietnamTimezone)
	}

	var result []logic.HistoryEntry
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		path := filepath.Join(s.Dir, filename(date))
		entries, err := readFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			t, err := logic.ParseTime(e.Time)
			if err != nil {
				continue
			}
			if t.Before(start) {
				continue
			}
			if end != nil && t.After(*end) {
				continue
			}
			result = append(result, e)
		}
	}
	return result, nil
}

func readFile(path string) ([]logic.HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []logic.HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e logic.HistoryEntry
		if err := json.Unmarshal(line, &e); err != nil {
			// Skip malformed lines.
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

func writeFile(path string, entries []logic.HistoryEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error os.Create %s: %w", path, err)
	}
	defer f.Close()

	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("error json.Marshal: %w", err)
		}
		data = append(data, '\n')
		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("error writing entry: %w", err)
		}
	}
	return nil
}
