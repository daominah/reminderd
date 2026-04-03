package notify

import "testing"

func TestNotify_Success(t *testing.T) {
	// WHEN a notification is sent with a normal message
	n := &OSNotifier{}
	err := n.Notify("Test Title", "Test message body")

	// THEN osascript executes without error
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNotify_SpecialCharacters(t *testing.T) {
	// WHEN the message contains quotes and backslashes
	n := &OSNotifier{}
	err := n.Notify(`He said "hello"`, `path: C:\Users\test`)

	// THEN osascript executes without error
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNotify_EmptyStrings(t *testing.T) {
	// WHEN both title and message are empty
	n := &OSNotifier{}
	err := n.Notify("", "")

	// THEN osascript still executes without error
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
