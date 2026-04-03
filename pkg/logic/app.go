package logic

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/daominah/reminderd/pkg/model"
)

// Default values used when no ConfigStore is provided.
const (
	ContinuousActiveLimit = 60 * time.Minute
	IdleThreshold         = 2 * time.Minute
	InitialBackoff        = 5 * time.Minute
	PollInterval          = 10 * time.Second
)

// UserInputTracker monitors user input and sends break reminders.
type UserInputTracker struct {
	IdleDetector  IdleDetector
	Notifier      Notifier
	ConfigStore   ConfigStore
	HistoryWriter HistoryWriter
	TimeNow       func() time.Time

	config           model.Config
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
	return ContinuousActiveLimit
}

func (t *UserInputTracker) idleThreshold() time.Duration {
	if t.config.IdleDurationToConsiderBreak > 0 {
		return t.config.IdleDurationToConsiderBreak
	}
	return IdleThreshold
}

func (t *UserInputTracker) initialBackoff() time.Duration {
	if t.config.NotificationInitialBackoff > 0 {
		return t.config.NotificationInitialBackoff
	}
	return InitialBackoff
}

func (t *UserInputTracker) pollInterval() time.Duration {
	if t.config.KeyboardMouseInputPollInterval > 0 {
		return t.config.KeyboardMouseInputPollInterval
	}
	return PollInterval
}

// Run polls the idle detector on an interval until the context is cancelled.
func (t *UserInputTracker) Run(ctx context.Context) error {
	t.loadConfig()

	if t.HistoryWriter != nil {
		if err := t.HistoryWriter.CompactPrevious(); err != nil {
			log.Printf("error HistoryWriter.CompactPrevious: %v", err)
		}
	}

	log.Printf("reminderd started (activeLimit=%s, idleThreshold=%s, poll=%s)",
		t.activeLimit(), t.idleThreshold(), t.pollInterval())
	if idle, err := t.IdleDetector.IdleSeconds(); err == nil {
		lastInput := t.timeNow().Add(-time.Duration(idle * float64(time.Second)))
		log.Printf("last input: %s (idle %.0fs)", lastInput.Format("15:04:05"), idle)
	}

	ticker := time.NewTicker(t.pollInterval())
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
			t.writeHistory(now, model.Idle)
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
	t.writeHistory(now, model.Active)

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
		msg := fmt.Sprintf(
			"You have been active for %s. Take a break!",
			formatDuration(activeDuration),
		)
		if err := t.Notifier.Notify("Break Reminder", msg); err != nil {
			log.Printf("error Notifier.Notify: %v", err)
			return
		}
		log.Printf("reminder sent (active %s)", formatDuration(activeDuration))
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

	msg := fmt.Sprintf(
		"You have been active for %s. Take a break!",
		formatDuration(activeDuration),
	)
	if err := t.Notifier.Notify("Break Reminder", msg); err != nil {
		log.Printf("error Notifier.Notify: %v", err)
		return
	}
	log.Printf("reminder sent (active %s, backoff %s)",
		formatDuration(activeDuration), backoff)
	t.lastReminderTime = now
	t.reminderCount++
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
		log.Printf("config reloaded (activeLimit=%s, idleThreshold=%s, poll=%s)",
			cfg.ContinuousActiveLimit, cfg.IdleDurationToConsiderBreak,
			cfg.KeyboardMouseInputPollInterval)
		t.config = cfg
	}
}

func (t *UserInputTracker) writeHistory(ts time.Time, state model.ActivityState) {
	if t.HistoryWriter == nil {
		return
	}
	entry := model.HistoryEntry{Time: model.FormatTime(ts), State: state}
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
