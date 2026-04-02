package notify

import "os/exec"

// OSNotifier sends notifications via notify-send.
type OSNotifier struct{}

func (n *OSNotifier) Notify(title, message string) error {
	return exec.Command("notify-send", title, message).Run()
}
