// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"time"

	"github.com/jsperandio/pokemon-simulator/assets/anim"
)

// appearance is a POKéMON's entry animation in the arena, Emerald's,
// which the original doesn't have: the wild POKéMON's when it appears, and
// the player's fighter's when it is chosen. It is armed when the POKéMON
// comes in, and it starts once nothing covers the arena: a wild POKéMON's
// "Select a POKéMON" message box does at first. While it plays, it is drawn
// over the arena's controls instead of the still sprite of its list view.
type appearance struct {
	seen   int          // the game's count of such entries, as last seen
	sprite *anim.Sprite // the animation armed or playing, or nil
	start  time.Time    // when it started; zero while it is armed
	frame  int          // the frame showing
}

// arm readies species' animation, if play, when entries, the game's count
// of the entries this animation plays for, has changed since it was last
// seen.
func (a *appearance) arm(entries, species int, play bool) {
	if entries == a.seen {
		return
	}

	a.seen = entries
	a.stop()

	if !play {
		return
	}

	if spr, err := anim.Load(species, false); err == nil {
		a.sprite = spr
	}
}

// playing reports whether the animation is showing.
func (a *appearance) playing() bool {
	return a.sprite != nil && !a.start.IsZero()
}

// stop drops the animation, armed or playing.
func (a *appearance) stop() {
	a.sprite = nil
	a.start = time.Time{}
}

// update moves the animation on at now while the arena shows, or stops it
// when the arena hides; free tells whether nothing covers the arena. It
// reports whether what shows changed.
func (a *appearance) update(now time.Time, shown, free bool) bool {
	if shown {
		return a.tick(now, free)
	}

	if a.sprite == nil {
		return false
	}

	a.stop()

	return true
}

// tick starts the armed animation at now if the arena is free, or moves
// the playing one on; it reports whether what shows changed.
func (a *appearance) tick(now time.Time, free bool) bool {
	if a.sprite == nil {
		return false
	}

	if a.start.IsZero() {
		if !free {
			return false
		}

		a.start = now
		a.frame, _ = a.sprite.At(0)

		return true
	}

	frm, on := a.sprite.At(now.Sub(a.start))
	if !on {
		a.stop()

		return true
	}

	if frm == a.frame {
		return false
	}

	a.frame = frm

	return true
}

// paint draws the frame showing, placed from slot, the corner of the still
// sprite it stands in for.
func (a *appearance) paint(cv canvas, slot image.Point) {
	if !a.playing() {
		return
	}

	spr := a.sprite
	pnt := slot.Add(spr.Offset)
	cv.draw(pnt.X, pnt.Y, spr.Sheet, spr.Frame(a.frame))
}
