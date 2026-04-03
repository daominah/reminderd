package notify

import (
	"os/exec"
	"strings"
)

// OSNotifier sends notifications via macOS osascript.
type OSNotifier struct{}

func (n *OSNotifier) Notify(title, message string) error {
	title = escapeAppleScript(title)
	message = escapeAppleScript(message)
	script := `display notification "` + message + `" with title "` + title + `"`
	return exec.Command("osascript", "-e", script).Run()
}

func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
