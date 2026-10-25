// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

func Test_memo_resized(t *testing.T) {
	t.Parallel()

	t.Run("when the Battle Info log grows wider, should wrap its lines again and move the scroll bar to its edge", func(t *testing.T) {
		t.Parallel()

		mem := newTestApp(t).info.memo
		mem.setLines([]string{
			strings.Repeat("Charmander used Scratch! ", 4),
		}, nil)

		require.Len(t, mem.rows, 2, "the line before")

		rect := mem.rect
		rect.Max.X += 189
		mem.setBounds(rect)

		assert.Len(t, mem.rows, 1, "the line after")
		assert.Equal(t, rect.Dx()-2, mem.scroll.rect.Max.X, "the scroll bar's right edge")
	})
}

// memoWindow shows a window with a grey 200×60 memo, with ScrollBars
// scrollBars, or the VCL's default, none, when it is empty.
func memoWindow(ta *testApp, scrollBars string) *memo {
	props := map[string]string{
		"Color": "clBtnFace",
	}

	if scrollBars != "" {
		props["ScrollBars"] = scrollBars
	}

	wdw := ta.showWindow(&layout.Component{
		Class: "TForm",
		Name:  "MemoForm",
		Props: map[string]string{
			"ClientHeight": "60",
			"ClientWidth":  "200",
			"Position":     "poScreenCenter",
		},
		Children: []layout.Component{
			{
				Class:  "TMemo",
				Name:   "Memo1",
				Bounds: layout.Rect{X: 0, Y: 0, W: 200, H: 60},
				Props:  props,
			},
		},
	})

	return get[*memo](wdw, "Memo1")
}

// paintMemo paints memo mem on an image of its size.
func paintMemo(mem *memo) *image.RGBA {
	img := image.NewRGBA(mem.local())
	mem.paint(newCanvas(img))

	return img
}

// inksIn returns the colours of img in rect that aren't bg, each once.
func inksIn(img *image.RGBA, rect image.Rectangle, bg color.RGBA) []color.RGBA {
	var inks []color.RGBA

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			clr := img.RGBAAt(x, y)
			if clr != bg && !slices.Contains(inks, clr) {
				inks = append(inks, clr)
			}
		}
	}

	return inks
}

// rowInks paints memo mem and returns the colours of row i's text.
func rowInks(mem *memo, i int) []color.RGBA {
	tr := mem.textRect()
	hgt := mem.font.height()
	top := tr.Min.Y + 1 + (i-mem.scroll.pos)*hgt
	row := image.Rect(tr.Min.X, top, tr.Max.X, top+hgt)

	return inksIn(paintMemo(mem), row, mem.color)
}

func Test_memo_paint(t *testing.T) {
	t.Parallel()

	green := lineStyle{
		color: rgb(0x006400),
	}

	t.Run("when a line has a style, should draw it in its colour and the others in the font's", func(t *testing.T) {
		t.Parallel()

		mem := memoWindow(newTestApp(t), "ssVertical")
		mem.setLines([]string{"Charmander did 9 points of damage.", "Critical hit!"}, []lineStyle{green})

		assert.Equal(t, []color.RGBA{rgb(0x006400)}, rowInks(mem, 0), "the styled line")
		assert.Equal(t, []color.RGBA{colText}, rowInks(mem, 1), "the line without a style")
	})

	t.Run("when a styled line wraps, should draw every row of it in its colour", func(t *testing.T) {
		t.Parallel()

		mem := memoWindow(newTestApp(t), "ssVertical")
		mem.setLines([]string{strings.Repeat("Charmander used Scratch! ", 2)}, []lineStyle{green})

		require.Len(t, mem.rows, 2, "the line didn't wrap")
		assert.Equal(t, []color.RGBA{rgb(0x006400)}, rowInks(mem, 0), "the first row")
		assert.Equal(t, []color.RGBA{rgb(0x006400)}, rowInks(mem, 1), "the second row")
	})

	t.Run("when ScrollBars is ssVertical, should draw the scroll bar at the right", func(t *testing.T) {
		t.Parallel()

		mem := memoWindow(newTestApp(t), "ssVertical")

		assert.Equal(t, image.Rect(2, 2, 182, 58), mem.textRect())
		assert.Equal(t, image.Rect(182, 2, 198, 58), mem.barRect())
		assert.NotEmpty(t, inksIn(paintMemo(mem), mem.barRect(), mem.color), "no scroll bar is drawn")
	})

	t.Run("when ScrollBars is the default, should leave the bar's room to the text", func(t *testing.T) {
		t.Parallel()

		mem := memoWindow(newTestApp(t), "")

		assert.Equal(t, image.Rect(2, 2, 198, 58), mem.textRect())
		assert.True(t, mem.barRect().Empty(), "the memo has a scroll bar")
		// With no text, nothing but the background is inside the border.
		assert.Empty(t, inksIn(paintMemo(mem), mem.local().Inset(2), mem.color), "a scroll bar is drawn")
	})
}

func Test_memo_setLines(t *testing.T) {
	t.Parallel()

	t.Run("when a line becomes bold, should wrap it again at the bold face's widths", func(t *testing.T) {
		t.Parallel()

		mem := memoWindow(newTestApp(t), "ssVertical")
		wid := mem.textRect().Dx() - 2
		txt := "Charmander"

		for mem.font.width(txt+" used") <= wid {
			txt += " used"
		}

		require.Greater(t, boldFace(mem.font).width(txt), wid, "the bold line fits")

		mem.setLines([]string{txt}, nil)
		assert.Len(t, mem.rows, 1, "the plain line")

		bold := lineStyle{
			color: colText,
			bold:  true,
		}
		mem.setLines([]string{txt}, []lineStyle{bold})
		assert.Len(t, mem.rows, 2, "the bold line")
	})
}
