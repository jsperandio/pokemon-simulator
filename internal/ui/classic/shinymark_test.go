// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_markByKey(t *testing.T) {
	t.Parallel()

	t.Run("when the key names a mark, should find it whatever its case", func(t *testing.T) {
		t.Parallel()

		sm, ok := markByKey("outlinedtwinkles")
		assert.True(t, ok)
		assert.Equal(t, outlinedTwinkles, sm)
	})

	t.Run("when the key names none, should give the default", func(t *testing.T) {
		t.Parallel()

		sm, ok := markByKey("Rainbow")
		assert.False(t, ok)
		assert.Equal(t, goldSparkles, sm)
	})

	t.Run("when each mark's key is given, should find that mark", func(t *testing.T) {
		t.Parallel()

		for i, mi := range shinyMarks {
			sm, ok := markByKey(mi.key)
			assert.True(t, ok, mi.key)
			assert.Equal(t, shinyMark(i), sm, mi.key)
		}
	})
}

func Test_markNames(t *testing.T) {
	t.Parallel()

	t.Run("when asked for, should list the seven marks, the default first and None last", func(t *testing.T) {
		t.Parallel()

		names := markNames()
		require.Len(t, names, 7)
		assert.Equal(t, "Gold sparkles", names[0])
		assert.Equal(t, "None", names[6])
	})
}

func Test_shinyMark_draw(t *testing.T) {
	t.Parallel()

	// drawn draws sm over a 32×32 icon at (32, 32) and returns the
	// rectangle of what it drew.
	drawn := func(sm shinyMark) image.Rectangle {
		img := image.NewRGBA(image.Rect(0, 0, 96, 96))
		sm.draw(newCanvas(img), 32, 32)

		var box image.Rectangle

		for y := range 96 {
			for x := range 96 {
				if img.RGBAAt(x, y).A != 0 {
					box = box.Union(image.Rect(x, y, x+1, y+1))
				}
			}
		}

		return box
	}

	t.Run("when a mark is drawn, should stay on the icon's item", func(t *testing.T) {
		t.Parallel()

		icon := image.Rect(32, 32, 64, 65) // the item is a row taller

		for i := range int(noMark) {
			sm := shinyMark(i)
			box := drawn(sm)
			assert.False(t, box.Empty(), "%s draws nothing", shinyMarks[sm].name)
			assert.True(t, box.In(icon), "%s draws at %v, off the icon", shinyMarks[sm].name, box)
		}
	})

	t.Run("when the mark is None or unknown, should draw nothing", func(t *testing.T) {
		t.Parallel()

		assert.True(t, drawn(noMark).Empty())
		assert.True(t, drawn(shinyMark(len(shinyMarks))).Empty())
		assert.True(t, drawn(shinyMark(-1)).Empty())
	})
}

// iconPixel returns the screen's colour at off from the corner of the
// first item's icon in lv.
func iconPixel(ta *testApp, lv *listView, off image.Point) color.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	ta.desk.paint(img)

	pnt := widgetOrigin(lv).Add(lv.itemRect(0).Min).Add(off)

	return img.RGBAAt(pnt.X, pnt.Y)
}

func Test_listView_paintIcon(t *testing.T) {
	t.Parallel()

	// paintIcon paints li's icon from the image list named list, with a
	// mark that records where it was drawn.
	paintIcon := func(t *testing.T, list string, li listItem) (*image.RGBA, []image.Point) {
		t.Helper()

		ta := newTestApp(t)

		var marks []image.Point

		lv := &listView{
			images: ta.res.imageList(list),
			mark: func(_ canvas, x, y int) {
				marks = append(marks, image.Pt(x, y))
			},
		}
		img := image.NewRGBA(image.Rect(0, 0, 64, 64))
		lv.paintIcon(newCanvas(img), li, image.Point{}, false)

		return img, marks
	}

	t.Run("when the item is shiny and the list has shiny frames, should draw the shiny frame and the mark over it", func(t *testing.T) {
		t.Parallel()

		li := listItem{
			image: 129,
			shiny: true,
		}
		img, marks := paintIcon(t, "PKMNImages", li)
		shiny := newTestApp(t).res.imageList("PKMNImages").shiny
		frm := shiny.frame(129)

		for y := range 64 {
			for x := range 64 {
				want := shiny.img.NRGBAAt(frm.Min.X+x, frm.Min.Y+y)
				if want.A == 0xff {
					require.Equal(t, color.RGBA{want.R, want.G, want.B, 0xff}, img.RGBAAt(x, y), "at (%d, %d)", x, y)
				}
			}
		}

		assert.Equal(t, []image.Point{{}}, marks)
	})

	t.Run("when the item is shiny and the list has no shiny frames, should draw the mark over the frame", func(t *testing.T) {
		t.Parallel()

		li := listItem{
			image: 129,
			shiny: true,
		}
		_, marks := paintIcon(t, "SmallPKMNImg", li)

		assert.Equal(t, []image.Point{{}}, marks)
	})

	t.Run("when the item isn't shiny, should draw the frame alone", func(t *testing.T) {
		t.Parallel()

		li := listItem{
			image: 129,
		}
		_, marks := paintIcon(t, "SmallPKMNImg", li)

		assert.Empty(t, marks)
	})
}

func Test_statsForm_shiny(t *testing.T) {
	t.Parallel()

	// shinyStatsApp starts a game whose Charmander is shiny.
	shinyStatsApp := func(t *testing.T) *testApp {
		t.Helper()

		ta := newTestApp(t)
		ta.startGame()

		ta.game.Owned[0].Shiny = true
		ta.sync()
		ta.wait(1)

		return ta
	}

	t.Run("when a party member is shiny, should name it so and mark its icon", func(t *testing.T) {
		t.Parallel()

		ta := shinyStatsApp(t)

		assert.Equal(t, []listItem{{"Charmander (shiny)", 3, true}}, ta.stats.party.items)
		assert.Equal(t, markGold, iconPixel(ta, ta.stats.party, image.Pt(12, 2)), "no gold sparkle")
		ta.shot("stats-shiny")
	})

	t.Run("when the mark is None, should only name it", func(t *testing.T) {
		t.Parallel()

		ta := shinyStatsApp(t)
		ta.setPreferences(preferences{
			shinyMark: noMark,
		})
		ta.wait(1)

		assert.Equal(t, "Charmander (shiny)", ta.stats.party.items[0].caption)
		assert.NotEqual(t, markGold, iconPixel(ta, ta.stats.party, image.Pt(12, 2)), "a gold sparkle")
	})

	t.Run("when the member stops being shiny, should show it as usual", func(t *testing.T) {
		t.Parallel()

		ta := shinyStatsApp(t)
		ta.game.Owned[0].Shiny = false
		ta.sync()
		ta.wait(1)

		assert.Equal(t, []listItem{{"Charmander", 3, false}}, ta.stats.party.items)
	})
}

func Test_mainForm_showStarters(t *testing.T) {
	t.Parallel()

	// rolledApp plays up to the starter choice, whose starters the game
	// rolls shiny as shiny says, in order.
	rolledApp := func(t *testing.T, shiny ...bool) *testApp {
		t.Helper()

		ta := newTestApp(t)
		ta.game.RollShiny = func() bool {
			res := shiny[0]
			shiny = shiny[1:]

			return res
		}
		ta.wait(80)
		ta.clickOn(ta.start.wdw, "NewButton")
		ta.typeText("Gary")
		ta.key(ebiten.KeyEnter)
		ta.typeText("ash")
		ta.key(ebiten.KeyEnter)
		ta.wait(1)

		return ta
	}

	t.Run("when the game rolls shiny starters, should show them shiny with the mark", func(t *testing.T) {
		t.Parallel()

		ta := rolledApp(t, true, false, true)
		lv := ta.main.starters
		shown := make([]bool, len(lv.items))

		for i, li := range lv.items {
			shown[i] = li.shiny
		}

		assert.Equal(t, []bool{true, false, true}, shown)
		assert.Equal(t, "Bulbasaur", lv.items[0].caption)
		assert.Equal(t, markGold, iconPixel(ta, lv, image.Pt(12, 2)), "no gold sparkle on Bulbasaur")
		ta.shot("starters-shiny")
	})

	t.Run("when a shiny starter is chosen, should join the party shiny", func(t *testing.T) {
		t.Parallel()

		ta := rolledApp(t, true, false, false)
		lv := ta.main.starters
		rect := lv.itemRect(0)
		ta.click(widgetOrigin(lv).Add(rect.Min).Add(image.Pt(20, 20)))
		ta.wait(1)

		assert.Equal(t, []listItem{{"Bulbasaur (shiny)", 0, true}}, ta.stats.party.items)
		assert.Contains(t, ta.game.Info.Lines, "It's a shiny POKéMON!")
	})

	t.Run("when the game rolls none, should show the starters as the original", func(t *testing.T) {
		t.Parallel()

		ta := rolledApp(t, false, false, false)

		for _, li := range ta.main.starters.items {
			assert.False(t, li.shiny, li.caption)
		}
	})
}
