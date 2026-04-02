package userinput

/*
#cgo LDFLAGS: -lXss -lX11
#include <X11/extensions/scrnsaver.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Detector reads idle time from X11 XScreenSaver extension.
type IdleDetector struct{}

func (d *IdleDetector) IdleSeconds() (float64, error) {
	dpy := C.XOpenDisplay(nil)
	if dpy == nil {
		return 0, fmt.Errorf("XOpenDisplay failed")
	}
	defer C.XCloseDisplay(dpy)

	info := C.XScreenSaverAllocInfo()
	if info == nil {
		return 0, fmt.Errorf("XScreenSaverAllocInfo failed")
	}
	defer C.XFree(unsafe.Pointer(info))

	C.XScreenSaverQueryInfo(dpy, C.XDefaultRootWindow(dpy), info)
	return float64(info.idle) / 1000.0, nil
}
