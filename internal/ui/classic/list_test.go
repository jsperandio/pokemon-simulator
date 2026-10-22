// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// listBoxWindow shows a window with a list box of n items, 11 lines high.
func listBoxWindow(ta *testApp, n int) (*window, *listBox) {
	items := make([]string, n)
	for i := range items {
		items[i] = "Item " + strconv.Itoa(i)
	}

	wdw := ta.showWindow(&layout.Component{
		Class: "TForm",
		Name:  "ListForm",
		Props: map[string]string{
			"ClientHeight": "160",
			"ClientWidth":  "120",
			"Position":     "poScreenCenter",
		},
		Children: []layout.Component{
			{
				Class:  "TListBox",
				Name:   "List",
				Bounds: layout.Rect{X: 0, Y: 0, W: 100, H: 147},
				Items:  items,
			},
		},
	})

	return wdw, get[*listBox](wdw, "List")
}

func Test_listBox(t *testing.T) {
	t.Parallel()

	t.Run("when an item is clicked, should select it and tell", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		_, lbx := listBoxWindow(ta, 3)

		selects := 0
		lbx.onSelect = func() {
			selects++
		}

		rect := lbx.itemRect(2)
		ta.click(widgetOrigin(lbx).Add(rect.Min).Add(image.Pt(5, 5)))

		assert.Equal(t, 2, lbx.selected)
		assert.Equal(t, 1, selects)
		assert.Same(t, lbx, lbx.wdw.focus, "the click didn't focus the list")
	})

	t.Run("when the items fit, should show no scroll bar", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		_, lbx := listBoxWindow(ta, 3)

		assert.Equal(t, image.Rectangle{}, lbx.scroll.rect)
	})

	t.Run("when the keys move the selection, should keep it in view", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		wdw, lbx := listBoxWindow(ta, 30)
		wdw.setFocus(lbx, false)
		ta.shot("listbox")

		require.NotEqual(t, image.Rectangle{}, lbx.scroll.rect, "no scroll bar for 30 items")
		assert.Equal(t, 11, lbx.rows())

		ta.key(ebiten.KeyDown)
		assert.Equal(t, 0, lbx.selected, "Down with none selected should select the first")

		ta.key(ebiten.KeyEnd)
		assert.Equal(t, 29, lbx.selected)
		assert.Equal(t, 19, lbx.scroll.pos, "the last item isn't in view")

		ta.key(ebiten.KeyPageUp)
		assert.Equal(t, 19, lbx.selected)

		ta.key(ebiten.KeyHome)
		assert.Equal(t, 0, lbx.selected)
		assert.Equal(t, 0, lbx.scroll.pos)

		ta.key(ebiten.KeyUp)
		assert.Equal(t, 0, lbx.selected, "Up past the first item")
	})

	t.Run("when the wheel turns, should scroll without selecting", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		_, lbx := listBoxWindow(ta, 30)
		ta.mouse = widgetOrigin(lbx).Add(image.Pt(20, 20))
		ta.frame(input{
			wheel: -3,
		})

		assert.Equal(t, 3, lbx.scroll.pos)
		assert.Equal(t, -1, lbx.selected)
	})
}
