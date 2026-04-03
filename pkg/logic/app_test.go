package logic

import (
	"testing"
	"time"

	"github.com/daominah/reminderd/pkg/model"
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
	if notifier.Calls[0].Title != "Sat Too Long, Take a Break" {
		t.Errorf("expected title 'Sat Too Long, Take a Break', got %q", notifier.Calls[0].Title)
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

func TestTick_WritesActiveHistoryEntry(t *testing.T) {
	// GIVEN a tracker with a history writer
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	history := &MockHistoryWriter{}
	tracker := &UserInputTracker{
		IdleDetector:  idle,
		Notifier:      &MockNotifier{},
		HistoryWriter: history,
		TimeNow:       func() time.Time { return now },
	}

	// WHEN the user is active for 3 ticks
	for range 3 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// THEN 3 active entries are written
	if len(history.Entries) != 3 {
		t.Fatalf("expected 3 history entries, got %d", len(history.Entries))
	}
	for i, e := range history.Entries {
		if e.State != model.Active {
			t.Errorf("entry %d: expected state %q, got %q", i, model.Active, e.State)
		}
	}
}

func TestTick_WritesIdleHistoryEntryOnBreak(t *testing.T) {
	// GIVEN a tracker with a history writer and active user
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	history := &MockHistoryWriter{}
	tracker := &UserInputTracker{
		IdleDetector:  idle,
		Notifier:      &MockNotifier{},
		HistoryWriter: history,
		TimeNow:       func() time.Time { return now },
	}

	// WHEN the user is active for 2 ticks
	for range 2 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// AND then goes idle past the threshold
	idle.Seconds = IdleThreshold.Seconds()
	tracker.Tick()
	now = now.Add(PollInterval)

	// THEN the last entry is an idle entry
	if len(history.Entries) < 3 {
		t.Fatalf("expected at least 3 history entries, got %d", len(history.Entries))
	}
	last := history.Entries[len(history.Entries)-1]
	if last.State != model.Idle {
		t.Errorf("expected last entry state %q, got %q", model.Idle, last.State)
	}
}

func TestTick_UsesConfigForThresholds(t *testing.T) {
	// GIVEN a tracker with a short active limit from config
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	notifier := &MockNotifier{}
	shortLimit := 1 * time.Minute
	configStore := &MockConfigStore{
		Cfg: model.Config{
			ContinuousActiveLimit:          shortLimit,
			IdleDurationToConsiderBreak:    2 * time.Minute,
			KeyboardMouseInputPollInterval: PollInterval,
			NotificationInitialBackoff:     5 * time.Minute,
			WebUIPort:                      20902,
		},
		Changed: true,
	}
	tracker := &UserInputTracker{
		IdleDetector:  idle,
		Notifier:      notifier,
		ConfigStore:   configStore,
		HistoryWriter: &MockHistoryWriter{},
		TimeNow:       func() time.Time { return now },
	}

	// WHEN the user is active past the short limit (1 minute)
	for range ticksFor(shortLimit) + 1 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// THEN a reminder is sent (using config limit, not the constant)
	if len(notifier.Calls) != 1 {
		t.Fatalf("expected 1 notification from config-based limit, got %d", len(notifier.Calls))
	}
}

func TestTick_ConfigHotReload(t *testing.T) {
	// GIVEN a tracker using the default active limit
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	notifier := &MockNotifier{}
	configStore := &MockConfigStore{
		Cfg: model.Config{
			ContinuousActiveLimit:          ContinuousActiveLimit,
			IdleDurationToConsiderBreak:    IdleThreshold,
			KeyboardMouseInputPollInterval: PollInterval,
			NotificationInitialBackoff:     InitialBackoff,
			WebUIPort:                      20902,
		},
		Changed: false,
	}
	tracker := &UserInputTracker{
		IdleDetector:  idle,
		Notifier:      notifier,
		ConfigStore:   configStore,
		HistoryWriter: &MockHistoryWriter{},
		TimeNow:       func() time.Time { return now },
	}

	// WHEN the user is active for 2 minutes (well under default 60m limit)
	for range ticksFor(2 * time.Minute) {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// AND then the config changes to a 2-minute limit
	configStore.Cfg.ContinuousActiveLimit = 2 * time.Minute
	configStore.Changed = true

	// AND one more tick happens
	tracker.Tick()
	now = now.Add(PollInterval)

	// THEN a reminder is sent (the new shorter limit is now exceeded)
	if len(notifier.Calls) != 1 {
		t.Fatalf("expected 1 notification after config hot-reload, got %d", len(notifier.Calls))
	}
}

func TestRestoreActiveStart_ResumesFromHistory(t *testing.T) {
	// GIVEN a history with an active session that started at 08:00
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []model.HistoryEntry{
			{Time: model.FormatTime(time.Date(2026, 1, 1, 7, 0, 0, 0, time.UTC)), State: model.Active},
			{Time: model.FormatTime(time.Date(2026, 1, 1, 7, 30, 0, 0, time.UTC)), State: model.Idle},
			{Time: model.FormatTime(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)), State: model.Active},
			{Time: model.FormatTime(time.Date(2026, 1, 1, 8, 50, 0, 0, time.UTC)), State: model.Active},
		},
	}
	tracker := &UserInputTracker{
		IdleDetector:  &MockIdleDetector{Seconds: 0},
		Notifier:      &MockNotifier{},
		HistoryReader: historyReader,
		HistoryWriter: &MockHistoryWriter{},
		TimeNow:       func() time.Time { return now },
	}

	// WHEN the tracker starts (simulating a restart)
	tracker.restoreActiveStart()

	// THEN activeStart is restored to 08:00 (start of the last active run)
	expected := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	if !tracker.activeStart.Equal(expected) {
		t.Errorf("expected activeStart %v, got %v", expected, tracker.activeStart)
	}

	// AND ActiveDuration reflects the time since 08:00
	d := tracker.ActiveDuration()
	if d != 1*time.Hour {
		t.Errorf("expected ActiveDuration 1h, got %v", d)
	}
}

func TestRestoreActiveStart_NoRestoreWhenLastEntryIsIdle(t *testing.T) {
	// GIVEN a history where the last entry is idle
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []model.HistoryEntry{
			{Time: model.FormatTime(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)), State: model.Active},
			{Time: model.FormatTime(time.Date(2026, 1, 1, 8, 30, 0, 0, time.UTC)), State: model.Idle},
		},
	}
	tracker := &UserInputTracker{
		IdleDetector:  &MockIdleDetector{Seconds: 0},
		Notifier:      &MockNotifier{},
		HistoryReader: historyReader,
		HistoryWriter: &MockHistoryWriter{},
		TimeNow:       func() time.Time { return now },
	}

	// WHEN the tracker starts
	tracker.restoreActiveStart()

	// THEN activeStart is not set (user was idle when process stopped)
	if !tracker.activeStart.IsZero() {
		t.Errorf("expected zero activeStart, got %v", tracker.activeStart)
	}
	if tracker.ActiveDuration() != 0 {
		t.Errorf("expected ActiveDuration 0, got %v", tracker.ActiveDuration())
	}
}

func TestRestoreActiveStart_NoHistoryReader(t *testing.T) {
	// GIVEN a tracker with no history reader
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	tracker := &UserInputTracker{
		IdleDetector: &MockIdleDetector{Seconds: 0},
		Notifier:     &MockNotifier{},
		TimeNow:      func() time.Time { return now },
	}

	// WHEN the tracker starts
	tracker.restoreActiveStart()

	// THEN activeStart stays zero (no crash, no restore)
	if !tracker.activeStart.IsZero() {
		t.Errorf("expected zero activeStart, got %v", tracker.activeStart)
	}
}
