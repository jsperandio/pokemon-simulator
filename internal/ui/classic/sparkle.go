// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"math"
	"time"

	"github.com/jsperandio/pokemon-simulator/assets/anim"
)

// sparkle is a shiny POKéMON's stars, which the original doesn't have:
// Emerald's, which circle and cross the sprite a second after it comes in,
// then the same kind of stars at rest around it while it stays, unless
// brief. With the animations off, the stars at rest show from the start.
// It goes with an appearance, armed by the same count of entries, and
// starts with it, once nothing covers the sprite.
type sparkle struct {
	brief bool      // no stars at rest: they show only while they move
	seen  int       // the game's count of such entries, as last seen
	shiny bool      // the POKéMON shown is shiny
	start time.Time // when the entry started; zero while it waits
	step  int       // GBA frames since the start, up to sparkleRest
}

const (
	// sparkleDelay is how many frames Emerald waits before the stars.
	sparkleDelay int = 60
	// sparkleRest is the frame the stars come to rest on: the last of the
	// five circling ones starts 16 frames after the first and goes round
	// in 22.
	sparkleRest int = sparkleDelay + 16 + 22
)

// spriteCentre is a 64×64 sprite's centre, from its corner.
var spriteCentre = image.Pt(32, 32)

// The stars' colours: pale gold, with white centres.
var (
	sparkleColor  = rgb(0xFFE65A)
	sparkleCentre = rgb(0xFFFFFF)
)

// star is a twinkle: its centre's place from the sprite's centre, and the
// length of its arms.
type star struct {
	at   image.Point
	size int
}

// restingStars are the stars around a shiny POKéMON at rest: five on the
// circle the moving ones go round, and three on their diagonal.
var restingStars = []star{
	{image.Pt(24, -5), 5},
	{image.Pt(5, -24), 3},
	{image.Pt(-20, -13), 3},
	{image.Pt(-20, 13), 3},
	{image.Pt(5, 23), 2},
	{image.Pt(-2, 2), 5},
	{image.Pt(18, -18), 3},
	{image.Pt(38, -38), 3},
}

// movingStars are the stars t frames after they start (Task_ShinyStars in
// Emerald): one every 4 frames, a big one, three medium ones and a small
// one, each circling the sprite 24 px out, a turn in about 21 frames, and
// another crossing it from the bottom left at 5 px a frame, 4 frames
// behind.
func movingStars(t int) []star {
	var stars []star

	for k, size := range []int{5, 3, 3, 3, 2} {
		age := t - 4*k
		if age < 0 {
			continue
		}

		if 12*age <= 255 {
			ang := 2 * math.Pi * float64(12*age) / 256
			pnt := image.Pt(int(math.Round(24*math.Sin(ang))), int(math.Round(24*math.Cos(ang))))
			stars = append(stars, star{
				at:   pnt,
				size: size,
			})
		}

		if age >= 4 && 5*(age-3) <= 64 {
			off := -32 + 5*(age-3)
			stars = append(stars, star{
				at:   image.Pt(off, -off),
				size: size,
			})
		}
	}

	return stars
}

// arm follows the POKéMON shown, shiny or not, and readies its stars when
// entries, the game's count of the entries they play for, has changed: to
// move once it starts if play, or else at rest at once.
func (s *sparkle) arm(entries int, shiny, play bool) {
	s.shiny = shiny

	if entries == s.seen {
		return
	}

	s.seen = entries
	s.start = time.Time{}
	s.step = 0

	if !play {
		s.rest()
	}
}

// rest puts the stars at rest at once, as the animations off show them.
func (s *sparkle) rest() {
	s.step = sparkleRest
}

// update moves the stars on at now while the sprite shows and nothing
// covers it (free), starting them then; it reports whether what shows
// changed.
func (s *sparkle) update(now time.Time, shown, free bool) bool {
	if !s.shiny || !shown || s.step >= sparkleRest {
		return false
	}

	if s.start.IsZero() {
		if free {
			s.start = now
		}

		return false
	}

	step := min(int(now.Sub(s.start)/anim.Tick), sparkleRest)
	if step == s.step {
		return false
	}

	before := s.step
	s.step = step

	return before >= sparkleDelay || step >= sparkleDelay
}

// paint draws the stars around centre, the sprite's.
func (s *sparkle) paint(cv canvas, centre image.Point) {
	if !s.shiny {
		return
	}

	stars := s.resting()

	if s.step < sparkleRest {
		if s.step < sparkleDelay {
			return
		}

		stars = movingStars(s.step - sparkleDelay)
	}

	for _, str := range stars {
		paintTwinkle(cv, centre.Add(str.at), str.size)
	}
}

// resting returns the stars at rest: none if brief.
func (s *sparkle) resting() []star {
	if s.brief {
		return nil
	}

	return restingStars
}

// paintTwinkle draws a twinkle at pnt: arms of size pixels and a white
// centre, with a small cross for the bigger ones.
func paintTwinkle(cv canvas, pnt image.Point, size int) {
	for off := -size; off <= size; off++ {
		cv.pix(pnt.X+off, pnt.Y, sparkleColor)
		cv.pix(pnt.X, pnt.Y+off, sparkleColor)
	}

	if size > 2 {
		for _, off := range []int{-1, 1} {
			cv.pix(pnt.X+off, pnt.Y+off, sparkleColor)
			cv.pix(pnt.X+off, pnt.Y-off, sparkleColor)
		}
	}

	cv.pix(pnt.X, pnt.Y, sparkleCentre)
}
