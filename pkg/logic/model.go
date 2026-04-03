package logic

import (
	"time"
)

// ActivityState is the user's input activity state.
type ActivityState string

// ActivityState enum values.
const (
	Active ActivityState = "ACTIVE"
	Idle   ActivityState = "IDLE"
)

type Config struct {
	ContinuousActiveLimit       time.Duration
	IdleDurationToConsiderBreak time.Duration
	NotificationInitialBackoff  time.Duration
	WebUIPort                   int
}

func DefaultConfig() Config {
	return Config{
		ContinuousActiveLimit:       DefaultContinuousActiveLimit,
		IdleDurationToConsiderBreak: DefaultIdleDurationToConsiderBreak,
		NotificationInitialBackoff:  DefaultNotificationInitialBackoff,
		WebUIPort:                   DefaultWebUIPort,
	}
}

type HistoryEntry struct {
	Time           string        `json:"Time"`
	State          ActivityState `json:"State"`
	IsCompact      bool          `json:"IsCompact,omitempty"`
	TimeCompactEnd string        `json:"TimeCompactEnd,omitempty"`
}

// TimeFormat used in log files
const TimeFormat = "2006-01-02T15:04:05Z07:00"

func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeFormat, s)
}
