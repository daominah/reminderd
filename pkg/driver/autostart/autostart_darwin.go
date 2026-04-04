package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const plistName = "com.daominah.reminderd.plist"

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error os.UserHomeDir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", plistName), nil
}

// Register creates a Launch Agent plist so reminderd starts at login.
func Register() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error os.Executable: %w", err)
	}
	p, err := plistPath()
	if err != nil {
		return err
	}
	content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.daominah.reminderd</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + exePath + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
</dict>
</plist>
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		return fmt.Errorf("error os.WriteFile %s: %w", p, err)
	}
	return nil
}

// Unregister removes the Launch Agent plist.
func Unregister() error {
	p, err := plistPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error os.Remove %s: %w", p, err)
	}
	return nil
}

// IsRegistered checks whether the Launch Agent plist exists.
func IsRegistered() bool {
	p, err := plistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}
