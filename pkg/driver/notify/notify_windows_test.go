package notify

import (
	"testing"
)

// These tests spawn real PowerShell processes and show actual desktop
// notifications. Run with -v to see notification output.
// Skip with -short to suppress during regular test runs.

func TestBalloonNotifier_Notify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping notification test in short mode")
	}

	// WHEN a normal message is sent via balloon tip
	err := BalloonNotifier{}.Notify("reminderd test", "Balloon tip: normal message")

	// THEN no error is returned
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestBalloonNotifier_Notify_SingleQuoteInMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping notification test in short mode")
	}

	// WHEN a message containing a single quote is sent
	// (apostrophes break unescaped PowerShell single-quoted strings)
	err := BalloonNotifier{}.Notify("reminderd test", "You've been active for 1h")

	// THEN the single quote is escaped and no error is returned
	if err != nil {
		t.Errorf("expected no error for message with apostrophe, got: %v", err)
	}
}

func TestToastNotifier_Notify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping notification test in short mode")
	}

	// WHEN a normal message is sent as a Windows toast
	err := ToastNotifier{}.Notify("reminderd test", "Toast: normal message")

	// THEN no error is returned and notification persists in Action Center
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestToastNotifier_Notify_XMLSpecialChars(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping notification test in short mode")
	}

	// WHEN a message containing XML special characters is sent
	// (unescaped &, <, >, ", ' break the toast XML payload)
	err := ToastNotifier{}.Notify(
		"reminderd test: <Alert>",
		`You've used >50% of focus time & "deep work" quota`,
	)

	// THEN special characters are escaped and no error is returned
	if err != nil {
		t.Errorf("expected no error for message with XML special chars, got: %v", err)
	}
}
