package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopFileName = "reminderd.desktop"

func desktopFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("error os.UserConfigDir: %w", err)
	}
	dir := filepath.Join(configDir, "autostart")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("error os.MkdirAll %s: %w", dir, err)
	}
	return filepath.Join(dir, desktopFileName), nil
}

// Register creates an XDG autostart .desktop entry so reminderd starts at login.
func Register() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error os.Executable: %w", err)
	}
	p, err := desktopFilePath()
	if err != nil {
		return err
	}
	content := "[Desktop Entry]\nType=Application\nName=Reminderd\nExec=" + exePath + "\nX-GNOME-Autostart-enabled=true\n"
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		return fmt.Errorf("error os.WriteFile %s: %w", p, err)
	}
	return nil
}

// Unregister removes the XDG autostart .desktop entry.
func Unregister() error {
	p, err := desktopFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error os.Remove %s: %w", p, err)
	}
	return nil
}

// IsRegistered checks whether the XDG autostart .desktop entry exists.
func IsRegistered() bool {
	p, err := desktopFilePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}
