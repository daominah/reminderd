package autostart

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const registryKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const appName = "Reminderd"

// Register adds the current executable to Windows startup (HKCU\...\Run).
// The entry appears in Task Manager's Startup tab.
func Register() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error os.Executable: %w", err)
	}
	key, _, err := registry.CreateKey(
		registry.CURRENT_USER, registryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("error registry.CreateKey: %w", err)
	}
	defer key.Close()
	if err := key.SetStringValue(appName, exePath); err != nil {
		return fmt.Errorf("error key.SetStringValue: %w", err)
	}
	return nil
}

// Unregister removes the startup entry.
func Unregister() error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER, registryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("error registry.OpenKey: %w", err)
	}
	defer key.Close()
	if err := key.DeleteValue(appName); err != nil {
		return fmt.Errorf("error key.DeleteValue: %w", err)
	}
	return nil
}

// IsRegistered checks whether the startup entry exists.
func IsRegistered() bool {
	key, err := registry.OpenKey(
		registry.CURRENT_USER, registryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(appName)
	return err == nil
}
