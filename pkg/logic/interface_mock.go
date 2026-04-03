package logic

import (
	"time"
)

// MockIdleDetector returns a configurable idle duration.
type MockIdleDetector struct {
	Seconds float64
	Err     error
}

func (m *MockIdleDetector) IdleSeconds() (float64, error) {
	return m.Seconds, m.Err
}

// MockNotifier records notifications instead of sending them.
type MockNotifier struct {
	Calls []MockNotifyCall
}

type MockNotifyCall struct {
	Title   string
	Message string
}

func (m *MockNotifier) Notify(title, message string) error {
	m.Calls = append(m.Calls, MockNotifyCall{Title: title, Message: message})
	return nil
}

// MockConfigStore returns a fixed config.
type MockConfigStore struct {
	Cfg     Config
	Changed bool
	Err     error
}

func (m *MockConfigStore) Load() (Config, error) {
	return m.Cfg, m.Err
}

func (m *MockConfigStore) LoadIfChanged() (Config, bool, error) {
	return m.Cfg, m.Changed, m.Err
}

func (m *MockConfigStore) Save(cfg Config) error {
	m.Cfg = cfg
	return m.Err
}

// MockHistoryWriter records entries instead of writing to files.
type MockHistoryWriter struct {
	Entries         []HistoryEntry
	CompactedCalled bool
}

func (m *MockHistoryWriter) WriteEntry(e HistoryEntry) error {
	m.Entries = append(m.Entries, e)
	return nil
}

func (m *MockHistoryWriter) CompactPrevious() error {
	m.CompactedCalled = true
	return nil
}

// MockHistoryReader returns a fixed set of entries.
type MockHistoryReader struct {
	Entries []HistoryEntry
	Err     error
}

func (m *MockHistoryReader) ReadRange(start time.Time, end *time.Time) ([]HistoryEntry, error) {
	return m.Entries, m.Err
}
