// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The game starts again without a title bar only when its window isn't
// up in time where the desktop may be holding it back.
func Test_watch(t *testing.T) {
	t.Parallel()

	for _, cas := range []struct {
		name          string
		shown, frames bool
		want          bool
	}{
		{"when the window shows on GNOME, should not start again", true, true, false},
		{"when GNOME holds the window back, should start again", false, true, true},
		{"when the window is slow elsewhere, should not start again", false, false, false},
	} {
		t.Run(cas.name, func(t *testing.T) {
			t.Parallel()

			shown := make(chan struct{})
			if cas.shown {
				close(shown)
			}

			restarted := false

			watch(shown, 10*time.Millisecond, cas.frames, func() {
				restarted = true
			})

			assert.Equal(t, cas.want, restarted)
		})
	}
}
