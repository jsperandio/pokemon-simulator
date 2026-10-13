// SPDX-License-Identifier: AGPL-3.0-only

package winfont

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

func sansSerif(t *testing.T) *Font {
	t.Helper()

	b, err := assets.FS.ReadFile("wine/sserife.fon")
	if err != nil {
		t.Fatal(err)
	}

	fonts, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}

	if len(fonts) != 3 {
		t.Fatalf("%d sizes, want 3 (8, 10 and 12 pt)", len(fonts))
	}

	return fonts[0]
}

func TestParse(t *testing.T) {
	f := sansSerif(t)
	if f.Name != "MS Sans Serif" || f.Points != 8 || f.Height != 13 || f.Ascent != 11 {
		t.Errorf("got %q %d pt, height %d, ascent %d", f.Name, f.Points, f.Height, f.Ascent)
	}
}

// Delphi stores an autosized label's width, measured with the real MS Sans
// Serif when the author saved the form: Wine's font must measure the same.
func TestLabelWidths(t *testing.T) {
	f := sansSerif(t)
	n := 0

	for _, form := range layout.All {
		form.Walk(func(c *layout.Component) {
			if c.Class != "TLabel" || c.Caption == "" || c.Props["AutoSize"] == "False" || c.Props["Font.Name"] != "" {
				return
			}

			n++

			if w := f.Width(c.Caption, false); w != c.Bounds.W {
				t.Errorf("%s.%s %q: %d pixels wide, the form says %d", form.Name, c.Name, c.Caption, w, c.Bounds.W)
			}
		})
	}

	if n < 70 {
		t.Errorf("only %d labels checked", n)
	}
}

// "Name:" as Wine draws it in the Statistics window
// (captures/screens/screens-game-stats.png).
func TestDraw(t *testing.T) {
	f := sansSerif(t)
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
	dst := image.NewRGBA(image.Rect(0, 0, 31, 13))

	if x := f.Draw(dst, dst.Rect, 0, 0, "Name:", color.RGBA{A: 255}, false); x != 31 {
		t.Errorf("Draw ends at %d, want 31", x)
	}

	for y, row := range want {
		var got strings.Builder

		for x := range row {
			if dst.RGBAAt(x, y).A != 0 {
				got.WriteByte('#')
			} else {
				got.WriteByte('.')
			}
		}

		if got.String() != row {
			t.Errorf("row %d: %s, want %s", y, got.String(), row)
		}
	}

	if w := f.Width("Name:", true); w != 36 {
		t.Errorf("bold width %d, want 36", w)
	}
}
