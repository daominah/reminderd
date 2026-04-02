package userinput

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getLastInputInfo = user32.NewProc("GetLastInputInfo")
	getTickCount     = kernel32.NewProc("GetTickCount")
)

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

// Detector reads idle time from Windows GetLastInputInfo.
type IdleDetector struct{}

func (d *IdleDetector) IdleSeconds() (float64, error) {
	var info lastInputInfo
	info.cbSize = uint32(unsafe.Sizeof(info))
	r, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0, fmt.Errorf("GetLastInputInfo: %w", err)
	}
	tick, _, _ := getTickCount.Call()
	idleMs := uint32(tick) - info.dwTime
	return float64(idleMs) / 1000.0, nil
}
