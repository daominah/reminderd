package notify

import "os/exec"

// OSNotifier sends notifications via Windows PowerShell balloon tip.
type OSNotifier struct{}

func (n *OSNotifier) Notify(title, message string) error {
	script := `
Add-Type -AssemblyName System.Windows.Forms
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.SystemIcons]::Information
$n.BalloonTipTitle = '` + title + `'
$n.BalloonTipText = '` + message + `'
$n.Visible = $True
$n.ShowBalloonTip(10000)
Start-Sleep -Seconds 1
$n.Dispose()
`
	return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
}
