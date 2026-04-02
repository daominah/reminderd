package userinput

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>
*/
import "C"

// Detector reads idle time from macOS Core Graphics.
type IdleDetector struct{}

func (d *IdleDetector) IdleSeconds() (float64, error) {
	seconds := C.CGEventSourceSecondsSinceLastEventType(
		C.kCGEventSourceStateCombinedSessionState,
		C.kCGAnyInputEventType,
	)
	return float64(seconds), nil
}
