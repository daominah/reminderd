package logic

import (
	"testing"
	"time"
)

// ticksFor returns how many poll ticks fit in the given duration.
func ticksFor(d time.Duration) int {
	return int(d / PollInterval)
}

func TestTick_NoReminderBeforeThreshold(t *testing.T) {
	// GIVEN an activity tracker with a user active for less than the limit
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	notifier := &MockNotifier{}
	tracker := &UserInputTracker{
		IdleDetector: idle,
		Notifier:     notifier,
		TimeNow:      func() time.Time { return now },
	}

	// WHEN we tick for just under the limit
	for range ticksFor(ContinuousActiveLimit) - 2 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// THEN no notification is sent
	if len(notifier.Calls) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(notifier.Calls))
	}
}

func TestTick_ReminderAtThreshold(t *testing.T) {
	// GIVEN an activity tracker
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	notifier := &MockNotifier{}
	tracker := &UserInputTracker{
		IdleDetector: idle,
		Notifier:     notifier,
		TimeNow:      func() time.Time { return now },
	}

	// WHEN the user is active for exactly the limit
	for range ticksFor(ContinuousActiveLimit) {
		tracker.Tick()
		now = now.Add(PollInterval)
	}
	tracker.Tick() // this tick crosses the threshold

	// THEN exactly one notification is sent
	if len(notifier.Calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.Calls))
	}
	if notifier.Calls[0].Title != "Break Reminder" {
		t.Errorf("expected title 'Break Reminder', got %q", notifier.Calls[0].Title)
	}
}

func TestTick_ExponentialBackoff(t *testing.T) {
	// GIVEN a tracker that already sent the first reminder
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	notifier := &MockNotifier{}
	tracker := &UserInputTracker{
		IdleDetector: idle,
		Notifier:     notifier,
		TimeNow:      func() time.Time { return now },
	}

	// Tick past the limit to trigger the first reminder.
	for range ticksFor(ContinuousActiveLimit) + 1 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}
	if len(notifier.Calls) != 1 {
		t.Fatalf("expected 1 notification after limit, got %d", len(notifier.Calls))
	}

	// WHEN the first backoff interval passes
	for range ticksFor(InitialBackoff) {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// THEN second reminder is sent
	if len(notifier.Calls) != 2 {
		t.Fatalf("expected 2 notifications after first backoff, got %d", len(notifier.Calls))
	}

	// WHEN the second backoff interval passes (2x initial)
	for range ticksFor(InitialBackoff * 2) {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// THEN third reminder is sent
	if len(notifier.Calls) != 3 {
		t.Fatalf("expected 3 notifications after second backoff, got %d", len(notifier.Calls))
	}
}

func TestTick_IdleResetsTimer(t *testing.T) {
	// GIVEN a tracker where user has been active for most of the limit
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	notifier := &MockNotifier{}
	tracker := &UserInputTracker{
		IdleDetector: idle,
		Notifier:     notifier,
		TimeNow:      func() time.Time { return now },
	}

	almostLimit := ticksFor(ContinuousActiveLimit) - 20
	for range almostLimit {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// WHEN the user goes idle for the idle threshold
	idle.Seconds = IdleThreshold.Seconds()
	tracker.Tick()
	now = now.Add(PollInterval)

	// AND becomes active again for the same duration
	idle.Seconds = 0
	for range almostLimit {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// THEN no notification is sent (timer was reset)
	if len(notifier.Calls) != 0 {
		t.Errorf("expected 0 notifications after idle reset, got %d", len(notifier.Calls))
	}
}

func TestTick_IdleResetsAfterReminder(t *testing.T) {
	// GIVEN a tracker that already sent a reminder
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	notifier := &MockNotifier{}
	tracker := &UserInputTracker{
		IdleDetector: idle,
		Notifier:     notifier,
		TimeNow:      func() time.Time { return now },
	}

	for range ticksFor(ContinuousActiveLimit) + 1 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}
	if len(notifier.Calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.Calls))
	}

	// WHEN the user takes a break
	idle.Seconds = IdleThreshold.Seconds()
	tracker.Tick()
	now = now.Add(PollInterval)

	// AND works for another full session
	idle.Seconds = 0
	for range ticksFor(ContinuousActiveLimit) + 1 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// THEN a new reminder is sent (fresh session)
	if len(notifier.Calls) != 2 {
		t.Errorf("expected 2 total notifications, got %d", len(notifier.Calls))
	}
}
