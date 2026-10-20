// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_gnomeFrames(t *testing.T) {
	t.Parallel()

	t.Run("when the desktop is GNOME and the helper is installed, should be true", func(t *testing.T) {
		t.Parallel()

		assert.True(t, framesOn("ubuntu:GNOME", "/usr/libexec/mutter-x11-frames"))
		assert.True(t, framesOn("GNOME", "/usr/lib/mutter-x11-frames"))
		assert.True(t, framesOn("GNOME-Classic:GNOME", "/usr/libexec/mutter-x11-frames"))
		assert.True(t, framesOn("GNOME", "/usr/local/libexec/mutter-x11-frames"))
	})

	// GNOME 43 or earlier: Mutter draws the bars itself.
	t.Run("when GNOME has no helper, should be false", func(t *testing.T) {
		t.Parallel()

		assert.False(t, framesOn("ubuntu:GNOME", ""))
	})

	t.Run("when the desktop isn't GNOME, should be false", func(t *testing.T) {
		t.Parallel()

		assert.False(t, framesOn("KDE", "/usr/libexec/mutter-x11-frames"))
		assert.False(t, framesOn("X-Cinnamon", ""))
		assert.False(t, framesOn("", ""))
	})
}

// framesOn runs gnomeFrames on desktop, with the helper installed at
// helper, or none for "".
func framesOn(desktop, helper string) bool {
	getenv := func(key string) string {
		if key == "XDG_CURRENT_DESKTOP" {
			return desktop
		}

		return ""
	}
	exists := func(path string) bool {
		return path == helper
	}

	return gnomeFrames(getenv, exists)
}
