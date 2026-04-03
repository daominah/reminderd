package model

import "time"

// ActivityState is the user's input activity state.
type ActivityState string

// ActivityState enum values.
const (
	Active ActivityState = "ACTIVE"
	Idle   ActivityState = "IDLE"
)

type Config struct {
	ContinuousActiveLimit          time.Duration
	IdleDurationToConsiderBreak    time.Duration
	KeyboardMouseInputPollInterval time.Duration
	NotificationInitialBackoff     time.Duration
	WebUIPort                      int
}

func DefaultConfig() Config {
	return Config{
		ContinuousActiveLimit:          60 * time.Minute,
		IdleDurationToConsiderBreak:    2 * time.Minute,
		KeyboardMouseInputPollInterval: 10 * time.Second,
		NotificationInitialBackoff:     5 * time.Minute,
		WebUIPort:                      20902,
	}
}

type HistoryEntry struct {
	Time  time.Time `json:"Time"`
	State ActivityState `json:"State"`
}
