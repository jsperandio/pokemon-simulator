// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package platform

import (
	"os"
	"slices"
	"strings"
)

// GNOMEFrames reports whether the desktop is GNOME 44 or later, which draws
// the title bars of X11 windows, the game's among them, with a helper
// program, mutter-x11-frames. GNOME shows such a window only once the
// helper has given it a bar, so while the helper isn't running it shows
// none; Mutter starts it again only after a crash.
func GNOMEFrames() bool {
	return gnomeFrames(os.Getenv, func(path string) bool {
		_, err := os.Stat(path)

		return err == nil
	})
}

// framesHelpers are where the systems install mutter-x11-frames.
var framesHelpers = []string{
	"/usr/libexec/mutter-x11-frames",
	"/usr/lib/mutter-x11-frames",
	"/usr/local/libexec/mutter-x11-frames", // the BSDs, and Mutter built from source
}

func gnomeFrames(getenv func(string) string, exists func(string) bool) bool {
	desktops := strings.Split(strings.ToUpper(getenv("XDG_CURRENT_DESKTOP")), ":")

	return slices.Contains(desktops, "GNOME") && slices.ContainsFunc(framesHelpers, exists)
}
