// SPDX-License-Identifier: AGPL-3.0-only

// Package anim embeds the animations POKéMON Emerald plays when a POKéMON
// appears, one per species, and tells which frame shows when. A build with
// the noanim tag leaves them out, and Load then finds none. The embedded
// files are not covered by the AGPL; see README.md in the assets
// directory.
package anim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"sync"
	"time"
)

// Tick is a frame of the Game Boy Advance, the unit of an animation's
// steps: 280,896 cycles of its 16.78 MHz clock, about 1/59.73 s.
const Tick time.Duration = 280896 * time.Second / (1 << 24)

// Step shows a frame for a number of ticks.
type Step struct {
	Frame int
	Ticks int
}

// Sprite is a species' appearance animation.
type Sprite struct {
	Sheet  *image.NRGBA // the distinct frames, side by side
	Size   image.Point  // a frame's size
	Offset image.Point  // a frame's top-left corner, from the still sprite's
	Steps  []Step
}

// entry is a species' line of front.json.
type entry struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Left   int     `json:"left"`
	Top    int     `json:"top"`
	Frames [][]int `json:"frames"` // [frame, ticks] pairs
}

// table reads front.json once.
var table = sync.OnceValues(func() ([]entry, error) {
	dat, err := files.ReadFile("front.json")
	if err != nil {
		return nil, err
	}

	var ents []entry
	if err := json.Unmarshal(dat, &ents); err != nil {
		return nil, fmt.Errorf("front.json: %w", err)
	}

	return ents, nil
})

// Load returns the animation of species (1–386), decoding its frames.
func Load(species int) (*Sprite, error) {
	ents, err := table()
	if err != nil {
		return nil, err
	}

	if species < 1 || species > len(ents) {
		return nil, fmt.Errorf("no animation for species %d", species)
	}

	pth := fmt.Sprintf("front/%03d.png", species)

	dat, err := files.ReadFile(pth)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(dat))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", pth, err)
	}

	sheet := image.NewNRGBA(img.Bounds())
	draw.Draw(sheet, sheet.Rect, img, img.Bounds().Min, draw.Src)

	ent := ents[species-1]
	spr := &Sprite{
		Sheet:  sheet,
		Size:   image.Pt(ent.Width, ent.Height),
		Offset: image.Pt(ent.Left, ent.Top),
		Steps:  make([]Step, 0, len(ent.Frames)),
	}

	for _, pair := range ent.Frames {
		if len(pair) != 2 {
			return nil, fmt.Errorf("front.json: species %d: a step of %d numbers", species, len(pair))
		}

		spr.Steps = append(spr.Steps, Step{
			Frame: pair[0],
			Ticks: pair[1],
		})
	}

	return spr, nil
}

// Frame is frame i's rectangle in the sheet.
func (s *Sprite) Frame(i int) image.Rectangle {
	return image.Rect(i*s.Size.X, 0, (i+1)*s.Size.X, s.Size.Y)
}

// Duration is how long the animation runs.
func (s *Sprite) Duration() time.Duration {
	ticks := 0
	for _, stp := range s.Steps {
		ticks += stp.Ticks
	}

	return time.Duration(ticks) * Tick
}

// At returns the frame that shows elapsed after the start, and false
// once the animation is over.
func (s *Sprite) At(elapsed time.Duration) (int, bool) {
	tick := int(elapsed / Tick)
	for _, stp := range s.Steps {
		if tick < stp.Ticks {
			return stp.Frame, true
		}

		tick -= stp.Ticks
	}

	return 0, false
}
