package notify

import (
	"os/exec"
	"strings"
)

// ToastNotifier sends persistent Windows 10/11 toast notifications
// that remain in the Action Center until the user dismisses them.
// Windows silently drops toasts from unregistered app IDs, so we borrow
// the Windows Explorer AUMID which is always registered on Windows 10/11.
type ToastNotifier struct{}

func (n ToastNotifier) Notify(title, message string) error {
	// Escape XML special characters for the toast XML payload.
	title = xmlEscape(title)
	message = xmlEscape(message)
	script := `
[Windows.UI.Notifications.ToastNotificationManager,Windows.UI.Notifications,ContentType=WindowsRuntime]|Out-Null
[Windows.Data.Xml.Dom.XmlDocument,Windows.Data.Xml.Dom.XmlDocument,ContentType=WindowsRuntime]|Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml('<toast><visual><binding template="ToastGeneric"><text>` + title + `</text><text>` + message + `</text></binding></visual></toast>')
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('microsoft.windows.explorer').Show($toast)
`
	return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// BalloonNotifier sends notifications via Windows PowerShell balloon tip.
type BalloonNotifier struct{}

// Notify blocks for ~10 seconds while the balloon tip is displayed.
// Windows does not save balloon tips in the notification history.
func (n BalloonNotifier) Notify(title, message string) error {
	// Escape single quotes for PowerShell single-quoted strings.
	title = strings.ReplaceAll(title, "'", "''")
	message = strings.ReplaceAll(message, "'", "''")
	script := `
Add-Type -AssemblyName System.Windows.Forms
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.SystemIcons]::Information
$n.BalloonTipTitle = '` + title + `'
$n.BalloonTipText = '` + message + `'
$n.Visible = $True
$n.ShowBalloonTip(10000)
Start-Sleep -Seconds 10
$n.Dispose()
`
	return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
}
