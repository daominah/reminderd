package logic

import (
	"time"

	"github.com/daominah/reminderd/pkg/model"
)

// IdleDetector returns how long the user has been idle.
type IdleDetector interface {
	IdleSeconds() (float64, error)
}

// Notifier sends a desktop notification.
type Notifier interface {
	Notify(title, message string) error
}

// ConfigStore loads and saves application configuration.
type ConfigStore interface {
	Load() (model.Config, error)
	LoadIfChanged() (model.Config, bool, error)
	Save(model.Config) error
}

// HistoryWriter appends activity entries to persistent storage.
type HistoryWriter interface {
	WriteEntry(model.HistoryEntry) error
	CompactPrevious() error
}

// HistoryReader reads activity history from persistent storage.
type HistoryReader interface {
	ReadRange(start time.Time, end *time.Time) ([]model.HistoryEntry, error)
}
