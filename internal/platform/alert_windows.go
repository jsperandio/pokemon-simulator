// SPDX-License-Identifier: AGPL-3.0-only

package platform

import "golang.org/x/sys/windows"

// Alert tells the player about a problem in a message box, for when the
// game's own window can't show it: the Windows build has no console.
func Alert(title, text string) {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}

	m, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return
	}

	windows.MessageBox(0, m, t, windows.MB_OK|windows.MB_ICONERROR) //nolint:errcheck // there is nothing else to tell it with
}
