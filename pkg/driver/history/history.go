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

	"github.com/daominah/reminderd/pkg/model"
)

var vnTimezone = time.FixedZone("ICT", 7*60*60)

// FileStore reads and writes history entries to daily JSONL files.
type FileStore struct {
	Dir         string
	currentDate string
	currentFile *os.File
}

func NewFileStore(dir string) *FileStore {
	return &FileStore{Dir: dir}
}

func dateKey(t time.Time) string {
	return t.In(vnTimezone).Format("2006-01-02")
}

func dateKeyFromString(s string) string {
	t, err := model.ParseTime(s)
	if err != nil {
		return ""
	}
	return dateKey(t)
}

func filename(date string) string {
	return "history-" + date + ".jsonl"
}

// WriteEntry appends a history entry to the appropriate daily file.
// If the date has changed, the previous file is closed and a new one is opened.
func (s *FileStore) WriteEntry(e model.HistoryEntry) error {
	date := dateKeyFromString(e.Time)
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
	today := dateKey(time.Now().In(vnTimezone))
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

	compacted := compact(entries)
	return writeFile(target, compacted)
}

// ReadRange returns history entries within the given time range.
// If end is nil, all entries from start onwards are returned.
func (s *FileStore) ReadRange(start time.Time, end *time.Time) ([]model.HistoryEntry, error) {
	startDate := start.In(vnTimezone)
	var endDate time.Time
	if end != nil {
		endDate = end.In(vnTimezone)
	} else {
		endDate = time.Now().In(vnTimezone)
	}

	var result []model.HistoryEntry
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
			t, err := model.ParseTime(e.Time)
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

func readFile(path string) ([]model.HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []model.HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e model.HistoryEntry
		if err := json.Unmarshal(line, &e); err != nil {
			// Skip malformed lines.
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

func writeFile(path string, entries []model.HistoryEntry) error {
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

// compact keeps only the first and last entry of each consecutive state run.
func compact(entries []model.HistoryEntry) []model.HistoryEntry {
	if len(entries) == 0 {
		return nil
	}

	var result []model.HistoryEntry
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
