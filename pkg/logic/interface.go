package logic

// IdleDetector returns how long the user has been idle.
type IdleDetector interface {
	IdleSeconds() (float64, error)
}

// Notifier sends a desktop notification.
type Notifier interface {
	Notify(title, message string) error
}
