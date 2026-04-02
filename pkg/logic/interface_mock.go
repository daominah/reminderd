package logic

// MockIdleDetector returns a configurable idle duration.
type MockIdleDetector struct {
	Seconds float64
	Err     error
}

func (m *MockIdleDetector) IdleSeconds() (float64, error) {
	return m.Seconds, m.Err
}

// MockNotifier records notifications instead of sending them.
type MockNotifier struct {
	Calls []MockNotifyCall
}

type MockNotifyCall struct {
	Title   string
	Message string
}

func (m *MockNotifier) Notify(title, message string) error {
	m.Calls = append(m.Calls, MockNotifyCall{Title: title, Message: message})
	return nil
}
