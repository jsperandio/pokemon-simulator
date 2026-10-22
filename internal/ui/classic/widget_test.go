// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// checkBoxWindow shows a window with two check boxes, the second disabled.
func checkBoxWindow(ta *testApp) (wdw *window, first, second *checkBox) {
	wdw = ta.showWindow(&layout.Component{
		Class: "TForm",
		Name:  "CheckForm",
		Props: map[string]string{
			"ClientHeight": "60",
			"ClientWidth":  "160",
			"Position":     "poScreenCenter",
		},
		Children: []layout.Component{
			{
				Class:   "TCheckBox",
				Name:    "First",
				Bounds:  layout.Rect{X: 8, Y: 8, W: 140, H: 17},
				Caption: "&First",
				Props: map[string]string{
					"TabOrder": "0",
				},
			},
			{
				Class:    "TCheckBox",
				Name:     "Second",
				Bounds:   layout.Rect{X: 8, Y: 32, W: 140, H: 17},
				Caption:  "&Second",
				Disabled: true,
				Props: map[string]string{
					"Checked":  "True",
					"TabOrder": "1",
				},
			},
		},
	})

	return wdw, get[*checkBox](wdw, "First"), get[*checkBox](wdw, "Second")
}

func Test_checkBox(t *testing.T) {
	t.Parallel()

	t.Run("when clicked, should toggle and tell", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		wdw, first, _ := checkBoxWindow(ta)

		clicks := 0
		first.onClick = func() {
			clicks++
		}

		ta.clickOn(wdw, "First")
		assert.True(t, first.checked)

		ta.clickOn(wdw, "First")
		assert.False(t, first.checked)
		assert.Equal(t, 2, clicks)
	})

	t.Run("when Space goes down and up, should toggle once, on the way up", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		wdw, first, _ := checkBoxWindow(ta)
		wdw.setFocus(first, false)

		ta.frame(input{
			keys: []ebiten.Key{ebiten.KeySpace},
		})
		assert.False(t, first.checked, "toggled on the way down")
		assert.True(t, first.keyDown, "not drawn pushed")

		ta.frame(input{
			keysUp: []ebiten.Key{ebiten.KeySpace},
		})
		assert.True(t, first.checked)
	})

	t.Run("when disabled, should keep its state", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		wdw, _, second := checkBoxWindow(ta)
		ta.shot("checkbox")

		ta.clickOn(wdw, "Second")
		ta.typeText("s")

		assert.True(t, second.checked)
	})

	t.Run("when its accelerator is typed, should focus and toggle it", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		wdw, first, _ := checkBoxWindow(ta)
		ta.typeText("f")

		assert.True(t, first.checked)
		assert.Same(t, first, wdw.focus)
	})
}

func Test_panel_inside(t *testing.T) {
	t.Parallel()

	t.Run("when the panel has its bevel, should leave it out", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		pickCharmander(t, ta)

		pnl := ta.pokedex.panel
		require.True(t, pnl.beveled(), "the POKéDEX's panel has no bevel")
		assert.Equal(t, image.Rect(1, 1, 240, 136), pnl.inside())
	})
}
