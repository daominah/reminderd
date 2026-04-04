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
	for range ticksFor(DefaultContinuousActiveLimit) - 2 {
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
	for range ticksFor(DefaultContinuousActiveLimit) {
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
	for range ticksFor(DefaultContinuousActiveLimit) + 1 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}
	if len(notifier.Calls) != 1 {
		t.Fatalf("expected 1 notification after limit, got %d", len(notifier.Calls))
	}

	// WHEN the first backoff interval passes
	for range ticksFor(DefaultNotificationInitialBackoff) {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// THEN second reminder is sent
	if len(notifier.Calls) != 2 {
		t.Fatalf("expected 2 notifications after first backoff, got %d", len(notifier.Calls))
	}

	// WHEN the second backoff interval passes (2x initial)
	for range ticksFor(DefaultNotificationInitialBackoff * 2) {
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

	almostLimit := ticksFor(DefaultContinuousActiveLimit) - 20
	for range almostLimit {
		tracker.Tick()
		now = now.Add(PollInterval)
	}

	// WHEN the user goes idle for the idle threshold
	idle.Seconds = DefaultIdleDurationToConsiderBreak.Seconds()
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

	for range ticksFor(DefaultContinuousActiveLimit) + 1 {
		tracker.Tick()
		now = now.Add(PollInterval)
	}
	if len(notifier.Calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.Calls))
	}

	// WHEN the user takes a break
	idle.Seconds = DefaultIdleDurationToConsiderBreak.Seconds()
	tracker.Tick()
	now = now.Add(PollInterval)

	// AND works for another full session
	idle.Seconds = 0
	for range ticksFor(DefaultContinuousActiveLimit) + 1 {
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
		if e.State != Active {
			t.Errorf("entry %d: expected state %q, got %q", i, Active, e.State)
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
	idle.Seconds = DefaultIdleDurationToConsiderBreak.Seconds()
	tracker.Tick()
	now = now.Add(PollInterval)

	// THEN the last entry is an idle entry
	if len(history.Entries) < 3 {
		t.Fatalf("expected at least 3 history entries, got %d", len(history.Entries))
	}
	last := history.Entries[len(history.Entries)-1]
	if last.State != Idle {
		t.Errorf("expected last entry state %q, got %q", Idle, last.State)
	}
}

func TestTick_UsesConfigForThresholds(t *testing.T) {
	// GIVEN a tracker with a short active limit from config
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idle := &MockIdleDetector{Seconds: 0}
	notifier := &MockNotifier{}
	shortLimit := 1 * time.Minute
	configStore := &MockConfigStore{
		Cfg: Config{
			ContinuousActiveLimit:       shortLimit,
			IdleDurationToConsiderBreak: DefaultIdleDurationToConsiderBreak,
			NotificationInitialBackoff:  DefaultNotificationInitialBackoff,
			WebUIPort:                   DefaultWebUIPort,
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
		Cfg: Config{
			ContinuousActiveLimit:       DefaultContinuousActiveLimit,
			IdleDurationToConsiderBreak: DefaultIdleDurationToConsiderBreak,
			NotificationInitialBackoff:  DefaultNotificationInitialBackoff,
			WebUIPort:                   DefaultWebUIPort,
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

func TestRestoreActiveStart_ResumesFromRawEntries(t *testing.T) {
	// GIVEN a history with raw ACTIVE entries every 10s,
	// an IDLE break before them, and now is within idleThreshold of the last entry
	now := time.Date(2026, 1, 1, 8, 1, 0, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []HistoryEntry{
			{Time: FormatTime(time.Date(2026, 1, 1, 7, 0, 0, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 7, 30, 0, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 10, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 20, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 30, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 40, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 50, 0, time.UTC)), State: Active},
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

	// THEN activeStart is restored to 08:00:00 (start of the active run)
	expected := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	if !tracker.activeStart.Equal(expected) {
		t.Errorf("expected activeStart %v, got %v", expected, tracker.activeStart)
	}

	// AND ActiveDuration reflects the time since 08:00:00
	d := tracker.ActiveDuration()
	if d != 1*time.Minute {
		t.Errorf("expected ActiveDuration 1m, got %v", d)
	}
}

func TestRestoreActiveStart_ResumesFromCompactEntry(t *testing.T) {
	// GIVEN a compacted history with a compact ACTIVE entry,
	// and now is within idleThreshold of the last entry
	now := time.Date(2026, 1, 1, 9, 1, 0, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []HistoryEntry{
			{Time: FormatTime(time.Date(2026, 1, 1, 7, 0, 0, 0, time.UTC)), State: Idle, IsCompact: true, TimeCompactEnd: FormatTime(time.Date(2026, 1, 1, 7, 59, 50, 0, time.UTC))},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)), State: Active, IsCompact: true, TimeCompactEnd: FormatTime(time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC))},
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

	// THEN activeStart is restored to 08:00:00 (the compact entry's Time)
	expected := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	if !tracker.activeStart.Equal(expected) {
		t.Errorf("expected activeStart %v, got %v", expected, tracker.activeStart)
	}
}

func TestRestoreActiveStart_NoRestoreWhenGapTooLarge(t *testing.T) {
	// GIVEN a history with ACTIVE entries, but now is 10 minutes after the last entry
	// (exceeds idleThreshold of 2m)
	now := time.Date(2026, 1, 1, 9, 10, 0, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []HistoryEntry{
			{Time: FormatTime(time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)), State: Active},
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

	// THEN activeStart is not set (gap exceeds idleThreshold)
	if !tracker.activeStart.IsZero() {
		t.Errorf("expected zero activeStart, got %v", tracker.activeStart)
	}
}

func TestRestoreActiveStart_ShortIdleDoesNotResetSession(t *testing.T) {
	// GIVEN a history with ACTIVE, short IDLE (30s < idleThreshold 2m), then ACTIVE again
	now := time.Date(2026, 1, 1, 8, 1, 30, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []HistoryEntry{
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 10, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 20, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 30, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 40, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 50, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 1, 0, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 1, 10, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 1, 20, 0, time.UTC)), State: Active},
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

	// THEN activeStart is 08:00:00 (short idle didn't break the session)
	expected := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	if !tracker.activeStart.Equal(expected) {
		t.Errorf("expected activeStart %v, got %v", expected, tracker.activeStart)
	}
}

func TestRestoreActiveStart_ShortGapDoesNotResetSession(t *testing.T) {
	// GIVEN a history with ACTIVE entries, a 30s data gap (< idleThreshold 2m),
	// then ACTIVE again
	now := time.Date(2026, 1, 1, 8, 1, 30, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []HistoryEntry{
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 10, 0, time.UTC)), State: Active},
			// 30s gap here (no entries from 08:00:20 to 08:00:50)
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 50, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 1, 0, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 1, 10, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 1, 20, 0, time.UTC)), State: Active},
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

	// THEN activeStart is 08:00:00 (short gap didn't break the session)
	expected := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	if !tracker.activeStart.Equal(expected) {
		t.Errorf("expected activeStart %v, got %v", expected, tracker.activeStart)
	}
}

func TestRestoreActiveStart_LastEntryIdleButWithinThreshold(t *testing.T) {
	// GIVEN a history where the last entry is IDLE but within idleThreshold of now,
	// and ACTIVE entries before it
	now := time.Date(2026, 1, 1, 8, 0, 40, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []HistoryEntry{
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 10, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 20, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 30, 0, time.UTC)), State: Idle},
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

	// THEN activeStart is 08:00:00 (short idle doesn't break session)
	expected := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	if !tracker.activeStart.Equal(expected) {
		t.Errorf("expected activeStart %v, got %v", expected, tracker.activeStart)
	}
}

func TestRestoreActiveStart_IdleLongerThanThreshold(t *testing.T) {
	// GIVEN a history with a gap/idle duration longer than standup break threshold
	now := time.Date(2026, 1, 1, 8, 1, 0, 0, time.UTC)
	historyReader := &MockHistoryReader{
		Entries: []HistoryEntry{
			{Time: FormatTime(time.Date(2026, 1, 1, 7, 57, 00, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 7, 57, 30, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 7, 58, 00, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 7, 59, 30, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 7, 59, 50, 0, time.UTC)), State: Idle},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 10, 0, time.UTC)), State: Active},
			{Time: FormatTime(time.Date(2026, 1, 1, 8, 0, 40, 0, time.UTC)), State: Active},
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

	// THEN activeStart is set to the first Active after the standup break
	expected := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	if !tracker.activeStart.Equal(expected) {
		t.Errorf("expected activeStart %v, got %v", expected, tracker.activeStart)
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
