// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTouches is a touch screen whose fingers a test sets for each frame.
type fakeTouches struct {
	down []ebiten.TouchID               // the fingers that land in the frame
	up   []ebiten.TouchID               // ... and that lift
	at   map[ebiten.TouchID]image.Point // where the fingers are
}

func (ft *fakeTouches) landed() []ebiten.TouchID {
	return ft.down
}

func (ft *fakeTouches) lifted(id ebiten.TouchID) bool {
	return slices.Contains(ft.up, id)
}

func (ft *fakeTouches) position(id ebiten.TouchID) image.Point {
	return ft.at[id]
}

// readFrame reads one frame of touches ft into in, whose mouse fields hold
// the mouse's, and returns it.
func readFrame(f *finger, in input, ft *fakeTouches) input {
	f.read(&in, ft)

	return in
}

func Test_finger_read(t *testing.T) {
	t.Parallel()

	mouse := image.Pt(-249, 0) // where a phone's cursor stays: off the desktop
	first, second := ebiten.TouchID(7), ebiten.TouchID(8)
	pntA, pntB, pntC := image.Pt(120, 80), image.Pt(130, 86), image.Pt(400, 300)

	// lands puts the first finger down at pntA, after a frame with no
	// finger.
	lands := func(f *finger) input {
		readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{})

		return readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{
			down: []ebiten.TouchID{first},
			at: map[ebiten.TouchID]image.Point{
				first: pntA,
			},
		})
	}

	t.Run("when a finger lands, should press the left button where it lands", func(t *testing.T) {
		t.Parallel()

		in := lands(&finger{})

		assert.True(t, in.pressed, "pressed")
		assert.True(t, in.down, "down")
		assert.False(t, in.released, "released")
		assert.True(t, in.touch, "touch")
		assert.Equal(t, pntA, in.mouse, "the pointer")
	})

	t.Run("when the finger moves, should hold the button and follow it", func(t *testing.T) {
		t.Parallel()

		f := &finger{}
		lands(f)

		in := readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{
			at: map[ebiten.TouchID]image.Point{
				first: pntB,
			},
		})

		assert.False(t, in.pressed, "pressed")
		assert.True(t, in.down, "down")
		assert.True(t, in.touch, "touch")
		assert.Equal(t, pntB, in.mouse, "the pointer")
	})

	t.Run("when the finger lifts, should release the button where it last was", func(t *testing.T) {
		t.Parallel()

		f := &finger{}
		lands(f)
		readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{
			at: map[ebiten.TouchID]image.Point{
				first: pntB,
			},
		})

		in := readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{
			up: []ebiten.TouchID{first},
		})

		assert.True(t, in.released, "released")
		assert.False(t, in.down, "down")
		assert.True(t, in.touch, "touch")
		assert.Equal(t, pntB, in.mouse, "the pointer")
	})

	t.Run("when the finger has lifted, should keep the pointer there while the mouse stays", func(t *testing.T) {
		t.Parallel()

		f := &finger{}
		lands(f)
		readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{
			up: []ebiten.TouchID{first},
		})

		in := readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{})

		assert.Equal(t, input{
			mouse: pntA,
		}, in, "the frame after the tap")
	})

	t.Run("when the mouse moves after a tap, should follow the mouse", func(t *testing.T) {
		t.Parallel()

		f := &finger{}
		lands(f)
		readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{
			up: []ebiten.TouchID{first},
		})

		in := readFrame(f, input{
			mouse: pntC,
		}, &fakeTouches{})

		assert.Equal(t, pntC, in.mouse, "the pointer")
		assert.False(t, in.touch, "touch")
	})

	t.Run("when a second finger lands while the first rests, should ignore it", func(t *testing.T) {
		t.Parallel()

		f := &finger{}
		lands(f)

		in := readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{
			down: []ebiten.TouchID{second},
			at: map[ebiten.TouchID]image.Point{
				first:  pntA,
				second: pntC,
			},
		})

		assert.False(t, in.pressed, "pressed by the second finger")
		assert.Equal(t, pntA, in.mouse, "the pointer")

		in = readFrame(f, input{
			mouse: mouse,
		}, &fakeTouches{
			up: []ebiten.TouchID{second},
			at: map[ebiten.TouchID]image.Point{
				first: pntA,
			},
		})

		assert.False(t, in.released, "released by the second finger")
		assert.True(t, in.down, "down")
	})

	t.Run("when no finger touches, should leave the mouse's input as it is", func(t *testing.T) {
		t.Parallel()

		f := &finger{}
		in := readFrame(f, input{
			mouse:   pntC,
			down:    true,
			pressed: true,
		}, &fakeTouches{})

		assert.Equal(t, input{
			mouse:   pntC,
			down:    true,
			pressed: true,
		}, in, "the mouse's press")
	})
}

func Test_DoubleTap(t *testing.T) {
	t.Parallel()

	// selecting runs a wild battle up to choosing the fighter, and returns
	// a point on the first party member.
	selecting := func(t *testing.T) (*testApp, image.Point) {
		t.Helper()

		ta := newTestApp(t)
		ta.startGame()
		ta.menu("Action1", "FightWild1", "FindPKMN1")
		ta.answer("Select a POKéMON")

		rect := ta.stats.party.itemRect(0)

		return ta, widgetOrigin(ta.stats.party).Add(rect.Min).Add(image.Pt(10, 10))
	}

	t.Run("when a finger taps a party member twice, 10 px apart, should act as a double click", func(t *testing.T) {
		t.Parallel()

		ta, pnt := selecting(t)
		ta.tap(pnt)
		ta.tap(pnt.Add(image.Pt(10, 4)))

		ta.answer("# 004 Charmander")
	})

	t.Run("when the second tap lands 30 px away, should not act", func(t *testing.T) {
		t.Parallel()

		ta, pnt := selecting(t)
		ta.tap(pnt)
		ta.tap(pnt.Add(image.Pt(30, 0)))

		assert.Nil(t, ta.desk.topModal(), "a dialog showed")
	})

	t.Run("when the second tap lands 600 ms after the first, should not act", func(t *testing.T) {
		t.Parallel()

		ta, pnt := selecting(t)
		ta.tap(pnt)
		ta.wait(34) // with the taps' own frames, 36 frames of 1/60 s
		ta.tap(pnt)

		assert.Nil(t, ta.desk.topModal(), "a dialog showed")
	})

	t.Run("when the mouse clicks twice 3 px apart, should not act, as it moved between the clicks", func(t *testing.T) {
		t.Parallel()

		ta, pnt := selecting(t)
		ta.click(pnt)
		ta.click(pnt.Add(image.Pt(3, 0)))

		assert.Nil(t, ta.desk.topModal(), "a dialog showed")
	})

	t.Run("when the mouse clicks twice on one spot, should still act", func(t *testing.T) {
		t.Parallel()

		ta, pnt := selecting(t)
		ta.doubleClick(pnt)

		require.NotNil(t, ta.desk.topModal(), "no dialog")
		ta.answer("# 004 Charmander")
	})
}
