package logic

import (
	"math"
	"time"

	"github.com/daominah/reminderd/pkg/base"
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

// TimeFormat used in log files and this service logic
const TimeFormat = "2006-01-02T15:04:05Z07:00"

func FormatTime(t time.Time) string {
	return t.In(base.VietnamTimezone).Format(TimeFormat)
}

func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeFormat, s)
}

func DiffTimeString(next, prev string) time.Duration {
	t1, err1 := ParseTime(next)
	t2, err2 := ParseTime(prev)
	if err1 != nil || err2 != nil {
		// unparseable timestamps are treated as a gap
		return time.Duration(math.MaxInt64)
	}
	d := t1.Sub(t2)
	if d < 0 {
		return -d
	}
	return d
}
