package logic

import (
	"context"
	"fmt"
	"log"
	"time"
)

const (
	// ContinuousActiveLimit is the maximum duration a user should sit
	// at the computer without a break. After this, a reminder is sent.
	ContinuousActiveLimit = 20 * time.Minute

	// IdleThreshold is how long the user must be idle
	// for it to count as a break (resets the active timer).
	IdleThreshold = 2 * time.Minute

	// InitialBackoff is the delay before the second reminder.
	// Subsequent reminders double: 5m, 10m, 20m, ...
	InitialBackoff = 5 * time.Minute

	// PollInterval is how often we check the OS for idle time.
	PollInterval = 30 * time.Second
)

// UserInputTracker monitors user input and sends break reminders.
type UserInputTracker struct {
	IdleDetector IdleDetector
	Notifier     Notifier
	TimeNow      func() time.Time

	activeStart      time.Time
	isReminded       bool
	lastReminderTime time.Time
	reminderCount    int
}

func NewUserInputTracker(idle IdleDetector, notifier Notifier) *UserInputTracker {
	return &UserInputTracker{
		IdleDetector: idle,
		Notifier:     notifier,
	}
}

// Run polls the idle detector on an interval until the context is cancelled.
func (t *UserInputTracker) Run(ctx context.Context) error {
	log.Println("reminderd started")
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
	idle, err := t.IdleDetector.IdleSeconds()
	if err != nil {
		log.Printf("error IdleDetector.IdleSeconds: %v", err)
		return
	}

	now := t.timeNow()

	if idle >= IdleThreshold.Seconds() {
		if !t.activeStart.IsZero() {
			log.Printf("break detected (idle %.0fs), resetting timer", idle)
			t.activeStart = time.Time{}
			t.isReminded = false
			t.reminderCount = 0
		}
		return
	}

	// User is active.
	if t.activeStart.IsZero() {
		t.activeStart = now
		return
	}

	activeDuration := now.Sub(t.activeStart)
	if activeDuration < ContinuousActiveLimit {
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
	backoff := InitialBackoff * (1 << (t.reminderCount - 1))
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
