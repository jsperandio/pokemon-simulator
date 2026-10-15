// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestActive checks what the player does that runs frames: the mouse
// moving on the desktop, or onto or off it, but not elsewhere on the
// screen; the button, the wheel and the keyboard anywhere.
func TestActive(t *testing.T) {
	off := image.Pt(-40, 700) // off the desktop
	for _, cas := range []struct {
		name string
		in   input
		prev image.Point
		want bool
	}{
		{"still", input{mouse: image.Pt(10, 10)}, image.Pt(10, 10), false},
		{"moved on the desktop", input{mouse: image.Pt(11, 10)}, image.Pt(10, 10), true},
		{"moved onto it", input{mouse: image.Pt(0, 599)}, off, true},
		{"moved off it", input{mouse: off}, image.Pt(799, 0), true},
		{"moved elsewhere", input{mouse: image.Pt(900, 10)}, off, false},
		{"button held", input{mouse: off, down: true}, off, true},
		{"wheel", input{mouse: off, wheel: -1}, off, true},
		{"key", input{mouse: off, keys: []ebiten.Key{ebiten.KeyTab}}, off, true},
		{"key up", input{mouse: off, keysUp: []ebiten.Key{ebiten.KeyTab}}, off, true},
		{"typed", input{mouse: off, chars: []rune("a")}, off, true},
	} {
		if got := cas.in.active(cas.prev); got != cas.want {
			t.Errorf("%s: active is %v, want %v", cas.name, got, cas.want)
		}
	}
}

// TestPace checks the pace the pacer keeps: a frame every frameTime for
// busyFor after the player did something or the desktop changed, and
// otherwise one every idleFrame, or every musicFrame while music plays.
func TestPace(t *testing.T) {
	var pcr pacer
	now := time.Now()
	pcr.next(&input{mouse: image.Pt(10, 10)}, false, false, now)

	if !pcr.busyUntil.Equal(now.Add(busyFor)) || time.Duration(pcr.idle.Load()) != idleFrame {
		t.Errorf("mouse moved: busy for %v, idle frames every %v", pcr.busyUntil.Sub(now), time.Duration(pcr.idle.Load()))
	}

	later := now.Add(busyFor + time.Second)
	pcr.next(&input{mouse: image.Pt(10, 10)}, false, true, later)

	if !pcr.busyUntil.Equal(now.Add(busyFor)) || time.Duration(pcr.idle.Load()) != musicFrame {
		t.Errorf("music playing: busy until %v, idle frames every %v", pcr.busyUntil.Sub(now), time.Duration(pcr.idle.Load()))
	}

	pcr.next(&input{mouse: image.Pt(10, 10)}, true, false, later)

	if !pcr.busyUntil.Equal(later.Add(busyFor)) || time.Duration(pcr.idle.Load()) != idleFrame {
		t.Errorf("desktop changed: busy for %v, idle frames every %v", pcr.busyUntil.Sub(later), time.Duration(pcr.idle.Load()))
	}
}
