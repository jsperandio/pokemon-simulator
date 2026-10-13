// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The game runs a frame only when it may change, as the original painted
// only when Windows asked it to, so it takes next to no processor while
// nobody plays. main runs Ebitengine in FPSModeVsyncOffMinimum, where a
// frame runs on input or when the game asks for one, and the pacer asks:
// a frame every frameTime while the player is at it or the desktop
// changes, and for busyFor after; otherwise one every idleFrame, for the
// clocks, or every musicFrame while music plays, so it repeats on time.
const (
	frameTime  = time.Second / 60       // the most frames a second, as with vsync
	busyFor    = 4 * time.Second        // past a hint's 3 seconds
	idleFrame  = 500 * time.Millisecond // between frames otherwise
	musicFrame = 100 * time.Millisecond // ... while music plays
)

// pacer asks Ebitengine for the frames the game needs.
type pacer struct {
	last      time.Time    // when the last frame started
	mouse     image.Point  // where the mouse was then
	busyUntil time.Time    // frames run every frameTime until then
	idle      atomic.Int64 // the time between frames otherwise
}

func newPacer() *pacer {
	p := &pacer{}
	p.idle.Store(int64(idleFrame))
	go p.beat()
	return p
}

// beat asks for a frame every p.idle, for good.
func (p *pacer) beat() {
	for {
		time.Sleep(time.Duration(p.idle.Load()))
		ebiten.ScheduleFrame() //nolint:staticcheck // the frames the game asks for (see above)
	}
}

// wait holds a frame back until frameTime after the last one, as input
// runs frames too and vsync is off. Browsers pace frames themselves.
func (p *pacer) wait() {
	if runtime.GOOS != "js" {
		if d := frameTime - time.Since(p.last); d > 0 {
			time.Sleep(d)
		}
	}
	p.last = time.Now()
}

// next asks for the next frame at once while the game is busy: for busyFor
// after the player used the mouse or the keyboard, or the desktop changed.
func (p *pacer) next(in *input, changed, music bool, now time.Time) {
	if changed || in.active(p.mouse) {
		p.busyUntil = now.Add(busyFor)
	}
	p.mouse = in.mouse
	if now.Before(p.busyUntil) {
		ebiten.ScheduleFrame() //nolint:staticcheck // the frames the game asks for (see above)
	}
	idle := idleFrame
	if music {
		idle = musicFrame
	}
	p.idle.Store(int64(idle))
}

// active reports whether the player did anything in the frame: moved the
// mouse on the desktop, from or to prev, or used it or the keyboard. The
// mouse moving elsewhere on the screen doesn't count, as the original
// heard of it only over its windows.
func (in *input) active(prev image.Point) bool {
	desktop := image.Rect(0, 0, screenWidth, screenHeight)
	moved := in.mouse != prev && (in.mouse.In(desktop) || prev.In(desktop))
	return moved || in.down || in.pressed || in.released || in.wheel != 0 ||
		len(in.keys) > 0 || len(in.keysUp) > 0 || len(in.chars) > 0 || in.dropped != nil
}
