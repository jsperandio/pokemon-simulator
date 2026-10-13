// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package main

import (
	"testing"
	"time"
)

// The game starts again without a title bar only when its window isn't
// up in time where the desktop may be holding it back.
func TestWatch(t *testing.T) {
	for _, c := range []struct {
		name          string
		shown, frames bool
		want          bool
	}{
		{"shown on GNOME", true, true, false},
		{"held back on GNOME", false, true, true},
		{"slow elsewhere", false, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			shown := make(chan struct{})
			if c.shown {
				close(shown)
			}

			restarted := false

			watch(shown, 10*time.Millisecond, c.frames, func() { restarted = true })

			if restarted != c.want {
				t.Errorf("started again: %t, want %t", restarted, c.want)
			}
		})
	}
}
