package notify

import (
	"os/exec"
	"strings"
)

// New returns the platform notifier for Linux.
func New() *OSNotifier { return &OSNotifier{} }

// OSNotifier sends notifications via notify-send.
type OSNotifier struct{}

func (n *OSNotifier) Notify(title, message string) error {
	// notify-send interprets Pango markup by default,
	// so escape angle brackets to prevent unintended formatting.
	title = escapeMarkup(title)
	message = escapeMarkup(message)
	return exec.Command("notify-send", title, message).Run()
}

func escapeMarkup(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
