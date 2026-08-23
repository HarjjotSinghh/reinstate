package daemon

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// OSNotifier shows a desktop notification with the platform's own tool:
// osascript on macOS, notify-send on Linux, a PowerShell toast on Windows.
// None of them is required; a missing tool is an error the loop logs.
type OSNotifier struct {
	GOOS string
	Run  Runner
}

// Notify implements Notifier.
func (n OSNotifier) Notify(title, body string) error {
	run := n.Run
	if run == nil {
		run = ExecRunner
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	switch n.GOOS {
	case "darwin":
		script := fmt.Sprintf("display notification %s with title %s", appleScriptString(body), appleScriptString(title))
		_, err := run(ctx, "osascript", "-e", script)
		return err
	case "linux":
		_, err := run(ctx, "notify-send", "--app-name=Reinstate", title, body)
		return err
	case "windows":
		_, err := run(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", windowsToast(title, body))
		return err
	}
	return fmt.Errorf("no notifier for %s", n.GOOS)
}

func appleScriptString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// windowsToast builds a PowerShell command that shows a toast through the
// WinRT notification API, falling back to a balloon tip on hosts where the
// toast API is not reachable from powershell.exe.
func windowsToast(title, body string) string {
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	return strings.Join([]string{
		"$t = " + q(title) + "; $b = " + q(body) + ";",
		"try {",
		"[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null;",
		"[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null;",
		"$x = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02);",
		"$n = $x.GetElementsByTagName('text'); $n.Item(0).AppendChild($x.CreateTextNode($t)) | Out-Null; $n.Item(1).AppendChild($x.CreateTextNode($b)) | Out-Null;",
		"[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Reinstate').Show([Windows.UI.Notifications.ToastNotification]::new($x));",
		"} catch {",
		"Add-Type -AssemblyName System.Windows.Forms;",
		"$i = New-Object System.Windows.Forms.NotifyIcon; $i.Icon = [System.Drawing.SystemIcons]::Information; $i.Visible = $true;",
		"$i.ShowBalloonTip(10000, $t, $b, [System.Windows.Forms.ToolTipIcon]::Info); Start-Sleep -Seconds 3; $i.Dispose();",
		"}",
	}, " ")
}
