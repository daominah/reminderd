package notify

import "os/exec"

// OSNotifier sends notifications via macOS osascript.
type OSNotifier struct{}

func (n *OSNotifier) Notify(title, message string) error {
	script := `display notification "` + message + `" with title "` + title + `"`
	return exec.Command("osascript", "-e", script).Run()
}
