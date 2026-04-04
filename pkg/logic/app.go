package logic

import (
	"context"
	"fmt"
	"log"
	"time"
)

// PollInterval is how often the app checks for keyboard/mouse activity.
// This is a fixed constant, not user-configurable, because log compaction
// uses it to detect gaps between entries.
const PollInterval = 10 * time.Second

// Default values used when no ConfigStore is provided.
const (
	DefaultContinuousActiveLimit       = 45 * time.Minute
	DefaultIdleDurationToConsiderBreak = 2 * time.Minute
	DefaultNotificationInitialBackoff  = 5 * time.Minute
	DefaultWebUIPort                   = 20902
)

// UserInputTracker monitors user input and sends break reminders.
type UserInputTracker struct {
	IdleDetector  IdleDetector
	Notifier      Notifier
	ConfigStore   ConfigStore
	HistoryWriter HistoryWriter
	HistoryReader HistoryReader

	// TimeNow overrides time.Now() in tests to simulate time progression.
	// Leave nil in production.
	TimeNow func() time.Time

	config           Config
	activeStart      time.Time
	isReminded       bool
	lastReminderTime time.Time
	reminderCount    int
	lastStatusLog    time.Time
}

func NewUserInputTracker(idle IdleDetector, notifier Notifier) *UserInputTracker {
	return &UserInputTracker{
		IdleDetector: idle,
		Notifier:     notifier,
	}
}

func (t *UserInputTracker) activeLimit() time.Duration {
	if t.config.ContinuousActiveLimit > 0 {
		return t.config.ContinuousActiveLimit
	}
	return DefaultContinuousActiveLimit
}

func (t *UserInputTracker) idleThreshold() time.Duration {
	if t.config.IdleDurationToConsiderBreak > 0 {
		return t.config.IdleDurationToConsiderBreak
	}
	return DefaultIdleDurationToConsiderBreak
}

func (t *UserInputTracker) initialBackoff() time.Duration {
	if t.config.NotificationInitialBackoff > 0 {
		return t.config.NotificationInitialBackoff
	}
	return DefaultNotificationInitialBackoff
}

// Run polls the idle detector on an interval until the context is cancelled.
func (t *UserInputTracker) Run(ctx context.Context) error {
	t.loadConfig()

	if t.HistoryWriter != nil {
		if err := t.HistoryWriter.CompactPrevious(); err != nil {
			log.Printf("error HistoryWriter.CompactPrevious: %v", err)
		}
	}

	t.restoreActiveStart()

	log.Printf("reminderd started (activeLimit=%s, idleThreshold=%s, poll=%s)",
		t.activeLimit(), t.idleThreshold(), PollInterval)
	if idle, err := t.IdleDetector.IdleSeconds(); err == nil {
		lastInput := t.timeNow().Add(-time.Duration(idle * float64(time.Second)))
		log.Printf("last input: %s (idle %.0fs)", lastInput.Format("15:04:05"), idle)
	}

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			t.Tick()
		}
	}
}

// Tick performs a single poll-and-check cycle.
func (t *UserInputTracker) Tick() {
	t.reloadConfigIfChanged()

	idle, err := t.IdleDetector.IdleSeconds()
	if err != nil {
		log.Printf("error IdleDetector.IdleSeconds: %v", err)
		return
	}

	now := t.timeNow()

	if idle >= t.idleThreshold().Seconds() {
		if !t.activeStart.IsZero() {
			log.Printf("break detected (idle %.0fs), resetting timer", idle)
			t.writeHistory(now, Idle)
			t.activeStart = time.Time{}
			t.isReminded = false
			t.reminderCount = 0
		}
		return
	}

	// Log last input time periodically.
	lastInput := now.Add(-time.Duration(idle * float64(time.Second)))
	if now.Sub(t.lastStatusLog) >= t.idleThreshold() {
		log.Printf("last input: %s (idle %.0fs)", lastInput.Format("15:04:05"), idle)
		t.lastStatusLog = now
	}

	// User is active: write history entry.
	t.writeHistory(now, Active)

	if t.activeStart.IsZero() {
		t.activeStart = now
		return
	}

	activeDuration := now.Sub(t.activeStart)
	if activeDuration < t.activeLimit() {
		return
	}

	// Active for >= threshold, check if we should remind.
	if !t.isReminded {
		if !t.SendReminder(activeDuration) {
			return
		}
		t.isReminded = true
		t.lastReminderTime = now
		t.reminderCount = 1
		return
	}

	// Exponential backoff: 5m, 10m, 20m, ...
	backoff := t.initialBackoff() * (1 << (t.reminderCount - 1))
	if now.Sub(t.lastReminderTime) < backoff {
		return
	}

	if !t.SendReminder(activeDuration) {
		return
	}
	t.lastReminderTime = now
	t.reminderCount++
}

func (t *UserInputTracker) SendReminder(activeDuration time.Duration) bool {
	msg := fmt.Sprintf(
		"You've been sitting at the computer for %s. Walk away, make a coffee!",
		formatDuration(activeDuration),
	)
	if err := t.Notifier.Notify("Sat Too Long, Take a Break", msg); err != nil {
		log.Printf("error Notifier.Notify: %v", err)
		return false
	}
	log.Printf("reminder sent (active %s)", formatDuration(activeDuration))
	return true
}

// ActiveDuration returns how long the user has been continuously active
// in the current session. Returns 0 if the user is not currently active.
func (t *UserInputTracker) ActiveDuration() time.Duration {
	if t.activeStart.IsZero() {
		return 0
	}
	return t.timeNow().Sub(t.activeStart)
}

// restoreActiveStart reads recent history on startup to pick up an
// active session that was in progress before the process restarted.
func (t *UserInputTracker) restoreActiveStart() {
	if t.HistoryReader == nil {
		return
	}
	now := t.timeNow()
	startOfDay := now.Add(-24 * time.Hour)
	entries, err := t.HistoryReader.ReadRange(startOfDay, nil)
	if err != nil || len(entries) == 0 {
		return
	}

	// Walk backwards log entries one by one
	standupPivot := FormatTime(now)
	activeStartStr := ""
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		// Check if the gap between standupPivot and this entry
		// is long enough to count as a standup break.
		var entryEnd string
		if entry.IsCompact {
			entryEnd = entry.TimeCompactEnd
		} else {
			entryEnd = entry.Time
		}
		if DiffTimeString(standupPivot, entryEnd) >= t.idleThreshold() {
			break
		}

		// No standup break: if this entry is ACTIVE, update activeStart
		// and advance the pivot. IDLE entries don't move the pivot,
		// so idle duration accumulates across consecutive IDLE entries.
		if entry.State == Active {
			activeStartStr = entry.Time
			standupPivot = entry.Time
		}
	}

	if activeStartStr == "" {
		return
	}
	parsed, err := ParseTime(activeStartStr)
	if err != nil {
		return
	}
	t.activeStart = parsed
	log.Printf("restored active session from %s (active %s)",
		parsed.Format("15:04:05"), now.Sub(parsed).Round(time.Second))
}

func (t *UserInputTracker) loadConfig() {
	if t.ConfigStore == nil {
		return
	}
	cfg, err := t.ConfigStore.Load()
	if err != nil {
		log.Printf("error ConfigStore.Load: %v", err)
		return
	}
	t.config = cfg
}

func (t *UserInputTracker) reloadConfigIfChanged() {
	if t.ConfigStore == nil {
		return
	}
	cfg, changed, err := t.ConfigStore.LoadIfChanged()
	if err != nil {
		log.Printf("error ConfigStore.LoadIfChanged: %v", err)
		return
	}
	if changed {
		log.Printf("config reloaded (activeLimit=%s, idleThreshold=%s)",
			cfg.ContinuousActiveLimit, cfg.IdleDurationToConsiderBreak)
		t.config = cfg
	}
}

func (t *UserInputTracker) writeHistory(ts time.Time, state ActivityState) {
	if t.HistoryWriter == nil {
		return
	}
	entry := HistoryEntry{Time: FormatTime(ts), State: state}
	if err := t.HistoryWriter.WriteEntry(entry); err != nil {
		log.Printf("error HistoryWriter.WriteEntry: %v", err)
	}
}

func (t *UserInputTracker) timeNow() time.Time {
	if t.TimeNow != nil {
		return t.TimeNow()
	}
	return time.Now()
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
