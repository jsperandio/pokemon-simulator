// SPDX-License-Identifier: AGPL-3.0-only

package winfont

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// sansSerifSizes parses the embedded MS Sans Serif, the font of every window
// and dialog, in all its sizes.
func sansSerifSizes(t *testing.T) []*Font {
	t.Helper()

	dat, err := assets.FS.ReadFile("wine/sserife.fon")
	require.NoError(t, err)

	fonts, err := Parse(dat)
	require.NoError(t, err)

	return fonts
}

// sansSerif returns the 8 pt size of the embedded MS Sans Serif.
func sansSerif(t *testing.T) *Font {
	t.Helper()

	fonts := sansSerifSizes(t)
	require.Len(t, fonts, 3, "8, 10 and 12 pt")

	return fonts[0]
}

func Test_Parse(t *testing.T) {
	t.Parallel()

	t.Run("when the file is parsed, should give three sizes", func(t *testing.T) {
		t.Parallel()

		assert.Len(t, sansSerifSizes(t), 3, "8, 10 and 12 pt")
	})

	t.Run("when the 8 pt size is read, should give its name and metrics", func(t *testing.T) {
		t.Parallel()

		fnt := sansSerif(t)

		assert.Equal(t, "MS Sans Serif", fnt.Name)
		assert.Equal(t, 8, fnt.Points)
		assert.Equal(t, 13, fnt.Height)
		assert.Equal(t, 11, fnt.Ascent)
	})
}

// autoSized reports whether cmp is a label with a caption, whose width Delphi
// stored as measured in the form's font.
func autoSized(cmp *layout.Component) bool {
	isLabel := cmp.Class == "TLabel" && cmp.Caption != ""
	autoSize := cmp.Props["AutoSize"] != "False" && cmp.Props["Font.Name"] == ""

	return isLabel && autoSize
}

// originalCaptions are the captions Delphi measured of the labels that
// show the port's version instead.
var originalCaptions = map[string]string{
	"AboutBox.Version": "Version 4.5 (32-bit)",
	"Form3.Label5":     "Version 4.5, Copyright© BirdieSoft 2002-2004",
}

func Test_Font_Width(t *testing.T) {
	t.Parallel()

	fnt := sansSerif(t)

	t.Run("when a label is autosized, should be as wide as Delphi stored it", func(t *testing.T) {
		t.Parallel()

		// Delphi stores an autosized label's width, measured with the real MS Sans
		// Serif when the author saved the form: Wine's font must measure the same.
		num := 0

		for _, frm := range layout.All {
			frm.Walk(func(cmp *layout.Component) {
				if !autoSized(cmp) {
					return
				}

				num++

				txt := cmp.Caption
				if org, ok := originalCaptions[frm.Name+"."+cmp.Name]; ok {
					txt = org
				}

				assert.Equal(t, cmp.Bounds.W, fnt.Width(txt, false),
					"%s.%s %q", frm.Name, cmp.Name, txt)
			})
		}

		assert.GreaterOrEqual(t, num, 70, "labels checked")
	})

	t.Run("when Name: is bold, should be 36 pixels wide", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 36, fnt.Width("Name:", true))
	})
}

// Test_Font_Draw checks "Name:" as Wine draws it, and where Draw ends.
func Test_Font_Draw(t *testing.T) {
	t.Parallel()

	fnt := sansSerif(t)
	ink := color.RGBA{
		A: 255,
	}
	want := []string{
		"...............................",
		"...............................",
		".#....#........................",
		".#....#........................",
		".##...#........................",
		".#.#..#...###..###.##...###..#.",
		".#.#..#......#.#..#..#.#...#...",
		".#..#.#...####.#..#..#.#####...",
		".#...##..#...#.#..#..#.#.......",
		".#....#..#...#.#..#..#.#...#...",
		".#....#...####.#..#..#..###..#.",
		"...............................",
		"...............................",
	}

	t.Run("when Name: is drawn, should set the pixels Wine draws", func(t *testing.T) {
		t.Parallel()

		// "Name:" as Wine draws it in the Statistics window.
		dst := image.NewRGBA(image.Rect(0, 0, 31, 13))

		fnt.Draw(dst, dst.Rect, 0, 0, "Name:", ink, false)

		assert.Equal(t, want, pixelRows(dst))
	})

	t.Run("when Name: is drawn, should end at x 31", func(t *testing.T) {
		t.Parallel()

		dst := image.NewRGBA(image.Rect(0, 0, 31, 13))

		assert.Equal(t, 31, fnt.Draw(dst, dst.Rect, 0, 0, "Name:", ink, false))
	})
}

// pixelRows returns the rows of dst, with '#' where a pixel is drawn and '.'
// where it is not.
func pixelRows(dst *image.RGBA) []string {
	rows := make([]string, dst.Rect.Dy())

	for y := range rows {
		var row strings.Builder

		for x := range dst.Rect.Dx() {
			pixel := byte('.')
			if dst.RGBAAt(x, y).A != 0 {
				pixel = '#'
			}

			row.WriteByte(pixel)
		}

		rows[y] = row.String()
	}

	return rows
}
