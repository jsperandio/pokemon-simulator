// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

// Test_input_active checks what the player does that runs frames: the mouse
// moving on the desktop, or onto or off it, but not elsewhere on the
// screen; the button, the wheel and the keyboard anywhere.
func Test_input_active(t *testing.T) {
	t.Parallel()

	pnt := image.Pt(10, 10)
	off := image.Pt(-40, 700) // off the desktop

	for _, cas := range []struct {
		name string
		in   input
		prev image.Point
		want bool
	}{
		{"when the mouse stays still, should not be active", input{mouse: pnt}, pnt, false},
		{
			"when the mouse moves on the desktop, should be active",
			input{mouse: image.Pt(11, 10)},
			pnt,
			true,
		},
		{
			"when the mouse moves onto the desktop, should be active",
			input{mouse: image.Pt(0, 599)},
			off,
			true,
		},
		{
			"when the mouse moves off the desktop, should be active",
			input{mouse: off},
			image.Pt(799, 0),
			true,
		},
		{
			"when the mouse moves elsewhere on the screen, should not be active",
			input{mouse: image.Pt(900, 10)},
			off,
			false,
		},
		{"when a button is held, should be active", input{mouse: off, down: true}, off, true},
		{"when the wheel turns, should be active", input{mouse: off, wheel: -1}, off, true},
		{
			"when a key goes down, should be active",
			input{mouse: off, keys: []ebiten.Key{ebiten.KeyTab}},
			off,
			true,
		},
		{
			"when a key comes up, should be active",
			input{mouse: off, keysUp: []ebiten.Key{ebiten.KeyTab}},
			off,
			true,
		},
		{
			"when a character is typed, should be active",
			input{mouse: off, chars: []rune("a")},
			off,
			true,
		},
	} {
		t.Run(cas.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, cas.want, cas.in.active(cas.prev))
		})
	}
}

// Test_pacer_next checks the pace the pacer keeps: a frame every frameTime
// for busyFor after the player did something or the desktop changed, and
// otherwise one every idleFrame, or every musicFrame while music plays.
func Test_pacer_next(t *testing.T) {
	t.Parallel()

	t.Run("when the mouse moves, should run frames at once for busyFor, then idle", func(t *testing.T) {
		t.Parallel()

		var pcr pacer
		now := time.Now()
		pcr.next(&input{
			mouse: image.Pt(10, 10),
		}, false, false, now)

		assert.Equal(t, now.Add(busyFor), pcr.busyUntil, "busy until")
		assert.Equal(t, idleFrame, time.Duration(pcr.idle.Load()), "idle frames every")
	})

	t.Run("when music plays once the busy time is over, should not extend it and tick every musicFrame", func(t *testing.T) {
		t.Parallel()

		var pcr pacer
		now := time.Now()
		pcr.next(&input{
			mouse: image.Pt(10, 10),
		}, false, false, now)

		later := now.Add(busyFor + time.Second)
		pcr.next(&input{
			mouse: image.Pt(10, 10),
		}, false, true, later)

		assert.Equal(t, now.Add(busyFor), pcr.busyUntil, "busy until")
		assert.Equal(t, musicFrame, time.Duration(pcr.idle.Load()), "idle frames every")
	})

	// Music plays, then the desktop changes: the change makes the game busy
	// again, and the idle frames go back to idleFrame.
	t.Run("when the desktop changed, should run frames at once for busyFor from then, then idle", func(t *testing.T) {
		t.Parallel()

		var pcr pacer
		now := time.Now()
		pcr.next(&input{
			mouse: image.Pt(10, 10),
		}, false, true, now)

		later := now.Add(busyFor + time.Second)
		pcr.next(&input{
			mouse: image.Pt(10, 10),
		}, true, false, later)

		assert.Equal(t, later.Add(busyFor), pcr.busyUntil, "busy until")
		assert.Equal(t, idleFrame, time.Duration(pcr.idle.Load()), "idle frames every")
	})
}
