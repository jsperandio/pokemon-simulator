// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package platform

import "testing"

func TestGNOMEFrames(t *testing.T) {
	for _, c := range []struct {
		desktop string
		helper  string // the installed helper, or ""
		want    bool
	}{
		{"ubuntu:GNOME", "/usr/libexec/mutter-x11-frames", true},
		{"GNOME", "/usr/lib/mutter-x11-frames", true},
		{"GNOME-Classic:GNOME", "/usr/libexec/mutter-x11-frames", true},
		{"GNOME", "/usr/local/libexec/mutter-x11-frames", true},
		{"ubuntu:GNOME", "", false}, // GNOME 43 or earlier: Mutter draws the bars itself
		{"KDE", "/usr/libexec/mutter-x11-frames", false},
		{"X-Cinnamon", "", false},
		{"", "", false},
	} {
		getenv := func(key string) string {
			if key == "XDG_CURRENT_DESKTOP" {
				return c.desktop
			}
			return ""
		}
		exists := func(path string) bool { return path == c.helper }
		if got := gnomeFrames(getenv, exists); got != c.want {
			t.Errorf("desktop %q, helper %q: %t, want %t", c.desktop, c.helper, got, c.want)
		}
	}
}
