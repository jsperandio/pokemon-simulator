// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// touchScreen is where a frame's fingers are read from, in screen
// coordinates.
type touchScreen interface {
	// landed returns the fingers that came down in the frame.
	landed() []ebiten.TouchID
	// lifted reports whether finger id went up in the frame.
	lifted(id ebiten.TouchID) bool
	// position returns where finger id is.
	position(id ebiten.TouchID) image.Point
}

// ebitenTouches is the touch screen as Ebitengine reports it.
type ebitenTouches struct{}

func (ebitenTouches) landed() []ebiten.TouchID {
	return inpututil.AppendJustPressedTouchIDs(nil)
}

func (ebitenTouches) lifted(id ebiten.TouchID) bool {
	return inpututil.IsTouchJustReleased(id)
}

func (ebitenTouches) position(id ebiten.TouchID) image.Point {
	x, y := ebiten.TouchPosition(id)

	return image.Pt(x, y)
}

// finger makes the first finger on a touch screen the left mouse button,
// as browsers make mouse events of taps: Ebitengine reports touches apart
// from the mouse, and keeps the browser from making those events.
type finger struct {
	id     ebiten.TouchID
	down   bool        // the finger is on the screen
	at     image.Point // the pointer: where the finger is or lifted, or the mouse
	cursor image.Point // the mouse, as the last frame had it
}

// read turns the frame's fingers into the left button of in, whose mouse
// fields hold the mouse's. The first finger presses the button where it
// lands, holds it while it moves, and releases it where it lifts; other
// fingers do nothing. The pointer stays where the finger lifted until the
// mouse moves.
func (f *finger) read(in *input, ts touchScreen) {
	moved := in.mouse != f.cursor
	f.cursor = in.mouse

	if moved && !f.down {
		f.at = in.mouse
	}

	switch {
	case f.down && ts.lifted(f.id):
		f.down = false
		in.released = true
		in.touch = true
	case f.down:
		f.at = ts.position(f.id)
		in.down = true
		in.touch = true
	default:
		f.land(in, ts)
	}

	in.mouse = f.at
}

// land presses the button with the first finger that came down in the
// frame, if one did.
func (f *finger) land(in *input, ts touchScreen) {
	ids := ts.landed()
	if len(ids) == 0 {
		return
	}

	f.id, f.down = ids[0], true
	f.at = ts.position(f.id)
	in.pressed = true
	in.down = true
	in.touch = true
}
