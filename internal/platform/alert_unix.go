// SPDX-License-Identifier: AGPL-3.0-only

//go:build unix

package platform

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Alert tells the player about a problem in a notification or a window of
// the system's, for when the game's own window can't show it or there is
// no terminal to read it in. It waits for a window to close, not for a
// notification.
func Alert(title, text string) {
	if runtime.GOOS == "darwin" {
		script := fmt.Sprintf("display alert %s message %s as critical", appleScriptString(title), appleScriptString(text))
		exec.Command("osascript", "-e", script).Run() //nolint:errcheck,gosec // the script quotes the text, and there is nothing else to tell it with

		return
	}

	// The notification's body is markup in GNOME's and KDE's.
	markup := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
	for _, cmd := range [][]string{
		{"notify-send", "--app-name=" + title, "--urgency=critical", title, markup},
		{"zenity", "--error", "--no-markup", "--title=" + title, "--text=" + text},
		{"kdialog", "--title", title, "--error", text},
	} {
		if exec.Command(cmd[0], cmd[1:]...).Run() == nil { //nolint:gosec // the programs above, with the text as an argument
			return
		}
	}
}

// appleScriptString quotes s for AppleScript.
func appleScriptString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
