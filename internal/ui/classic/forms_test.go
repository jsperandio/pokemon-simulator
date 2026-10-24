// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// The party list's hint shows after half a second on it, under the cursor
// and kept on the screen, and hides after two and a half seconds, when the
// mouse leaves the list, or at a click.
func Test_statsForm(t *testing.T) {
	t.Parallel()

	t.Run("when the mouse rests on the party list, should show the hint under the cursor after half a second", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()

		lv := ta.stats.party
		ta.mouse = widgetOrigin(lv).Add(image.Pt(60, 100))
		ta.wait(29) // under half a second

		require.False(t, ta.hintShown(), "the hint shows before HintPause")

		ta.wait(4)

		hnt := ta.desk.hint
		ta.shot("50-stats-hint")

		require.Equal(t, "Doubleclick a POKéMON to see its statistics.", hnt.text)

		// Under the cursor, and moved left to end at the screen's edge.
		size := image.Pt(ta.res.sans.width(hnt.text)+6, ta.res.sans.height()+6)
		pnt := image.Pt(screenWidth-size.X, ta.mouse.Y+hintCursorMargin)

		want := image.Rectangle{
			Min: pnt,
			Max: pnt.Add(size),
		}
		assert.Equal(t, want, hnt.rect, "the hint's rectangle")
	})

	t.Run("when the hint has shown for two and a half seconds, should hide it, and show it again only once the mouse has left and come back", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()

		lv := ta.stats.party
		ta.mouse = widgetOrigin(lv).Add(image.Pt(60, 100))
		ta.wait(33) // half a second

		require.True(t, ta.hintShown(), "the hint doesn't show after half a second")

		ta.wait(140) // under three seconds on the list

		require.True(t, ta.hintShown(), "the hint hid before HintHidePause")

		ta.wait(10)

		require.False(t, ta.hintShown(), "the hint stays after HintHidePause")

		ta.wait(60)

		require.False(t, ta.hintShown(), "the hint came back on the same control")

		// Off the list and back: it shows again.
		ta.mouse = image.Pt(10, 10)
		ta.wait(1)

		ta.mouse = widgetOrigin(lv).Add(image.Pt(60, 100))
		ta.wait(33)

		require.True(t, ta.hintShown(), "no hint on coming back")
	})

	t.Run("when a click lands while the hint shows, should hide it, and keep it hidden once the mouse is off the list", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()

		lv := ta.stats.party
		ta.mouse = widgetOrigin(lv).Add(image.Pt(60, 100))
		ta.wait(33)

		require.True(t, ta.hintShown(), "no hint on the list")

		ta.frame(input{
			down:    true,
			pressed: true,
		})
		ta.frame(input{
			released: true,
		})

		require.False(t, ta.hintShown(), "a click leaves the hint")

		ta.mouse = image.Pt(10, 10)
		ta.wait(40)

		require.False(t, ta.hintShown(), "a hint off the list")
	})
}

// The Game Corner's bet goes up and down by its Increment, 50, with the
// spin buttons and the arrow keys.
//
// A play's draws show one after another in "Lucky number" for a second, the
// bet already paid in Statistics, with no input taken, and the last one
// stays: the one the game counts.
func Test_gameCornerForm(t *testing.T) {
	t.Parallel()

	t.Run("when the spin's buttons and the arrow keys are used, should move the bet by its increment of 50", func(t *testing.T) {
		t.Parallel()

		ta := newGameCorner(t)

		se := get[*spinEdit](ta.gameCorner.wdw, "SpinEdit1")
		up, down := se.buttons()
		ta.click(widgetOrigin(se).Add(up.Min).Add(up.Size().Div(2)))
		ta.click(widgetOrigin(se).Add(up.Min).Add(up.Size().Div(2)))

		assert.Equal(t, 100, ta.game.GameCorner.Bet, "the bet after two clicks up")

		ta.click(widgetOrigin(se).Add(down.Min).Add(down.Size().Div(2)))
		ta.key(ebiten.KeyUp)
		ta.key(ebiten.KeyUp)

		assert.Equal(t, 150, ta.game.GameCorner.Bet, "the bet after a click down and two keys up")
		assert.Equal(t, "150", se.tb.String(), "the bet's text")
	})

	t.Run("when a play is drawn, should show the draws one after another for about a second, with the bet already paid", func(t *testing.T) {
		t.Parallel()

		ta := startPlay(t)
		gcf := ta.gameCorner

		assert.Equal(t, "900", ta.stats.counts["MoneyLbl"].caption, "the money Statistics shows during the draws")

		seen := map[int]bool{}
		frames := 0

		for ; ta.draws != nil && frames < 120; frames++ {
			seen[gcf.drawn.index] = true

			if frames == 20 {
				ta.shot("51-corner-draws")
			}

			ta.wait(1)
		}

		assert.GreaterOrEqual(t, frames, 57, "the draws showed for %d frames, want about a second", frames)
		assert.LessOrEqual(t, frames, 63, "the draws showed for %d frames, want about a second", frames)
		assert.GreaterOrEqual(t, len(seen), 8, "the numbers shown during the draws")

		ta.wait(1)

		assert.Equal(t, ta.game.GameCorner.Drawn, gcf.drawn.index, "the number shown after the draws, the one the game drew")
	})

	t.Run("when Close is clicked while the draws show, should not close the Game Corner", func(t *testing.T) {
		t.Parallel()

		ta := startPlay(t)
		gcf := ta.gameCorner

		ta.wait(20)
		ta.clickOn(gcf.wdw, "Button2") // Close: no input meanwhile

		assert.NotNil(t, ta.draws, "the draws stopped at Close")

		ta.wait(60) // past the draws' end

		require.Nil(t, ta.draws, "the draws still show")
		assert.True(t, gcf.wdw.visible, "Close worked during the draws")
	})
}

// newGameCorner starts a game with money to bet, and opens the Game Corner.
func newGameCorner(t *testing.T) *testApp {
	t.Helper()

	ta := newTestApp(t)
	ta.startGame()

	ta.game.Money = 1000
	ta.menu("Action1", "GameCorner1")
	ta.wait(1)

	return ta
}

// startPlay bets 100 on 3 for money in the Game Corner, and clicks Draw: the
// draws show.
func startPlay(t *testing.T) *testApp {
	t.Helper()

	ta := newGameCorner(t)
	gcf := ta.gameCorner

	clickRadio(ta, gcf.prize, 0)  // money
	clickRadio(ta, gcf.number, 3) // on 3
	ta.click(widgetOrigin(gcf.bet).Add(image.Pt(5, 5)))
	ta.key(ebiten.KeyHome)
	ta.frame(input{
		keys: []ebiten.Key{ebiten.KeyEnd},
		mods: mods{
			shift: true,
		},
	})
	ta.frame(input{
		keysUp: []ebiten.Key{ebiten.KeyEnd},
	})
	ta.typeText("100")
	ta.clickOn(gcf.wdw, "Button1") // Draw number!

	require.NotNil(t, ta.draws, "no draws show")

	return ta
}

// clickRadio clicks button idx of rg, near its left edge.
func clickRadio(ta *testApp, rg *radioGroup, idx int) {
	rect := rg.buttonRect(idx)
	ta.click(widgetOrigin(rg).Add(rect.Min).Add(image.Pt(6, rect.Dy()/2)))
}

// The start screen's minimize button minimizes the program's window, and
// Alt+F4 closes its About box, which has no system menu, as OK does.
func Test_startForm(t *testing.T) {
	t.Parallel()

	t.Run("when its minimize button is clicked, should minimize the program's window and keep the start screen shown", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		n := 0
		ta.desk.onMinimize = func() {
			n++
		}
		ta.wait(80)

		wdw := ta.start.wdw
		require.True(t, wdw.minBox, "the start screen has no minimize button")

		ta.click(wdw.captionButton(capMin).Min.Add(image.Pt(4, 4)))

		assert.Equal(t, 1, n, "the minimizes of the program's window")
		assert.True(t, wdw.visible, "the start screen hid")
	})

	t.Run("when Alt+F4 is pressed on the About box, should close it and keep the start screen shown", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.wait(80)
		ta.clickOn(ta.start.wdw, "AboutButton")
		ta.wait(1)

		require.Same(t, ta.about, ta.desk.topModal(), "no About box")
		require.False(t, ta.about.sysMenu, "the About box has a system menu")

		ta.altF4()
		ta.wait(1)

		assert.Nil(t, ta.desk.topModal(), "the About box stays open")
		assert.False(t, ta.about.visible, "the About box shows")
		assert.True(t, ta.start.wdw.visible, "the start screen hid")
	})
}

// The loading screen is AutoSize: it grows to its title, a pixel wider in
// Arial than the form, after Position centred it, and the labels anchored
// to its right move with it.
func Test_loadForm(t *testing.T) {
	t.Parallel()

	t.Run("when the loading screen fits its title, should grow to 511×157 and stay centred as 510 wide", func(t *testing.T) {
		t.Parallel()

		wdw := newTestApp(t).load.wdw

		assert.Equal(t, image.Pt(511, 157), wdw.client, "the client area")
		assert.Equal(t, (screenWidth-510)/2, wdw.pos.X, "the x position")
	})

	t.Run("when the loading screen fits its title, should move the labels anchored to its right", func(t *testing.T) {
		t.Parallel()

		wdw := newTestApp(t).load.wdw

		for name, x := range map[string]int{
			"Label1": 216,
			"Label2": 265,
			"Label3": 12,
			"Label4": 29,
		} {
			rect := wdw.find(name).base().rect
			assert.Equal(t, x, rect.Min.X, "%s", name)
		}
	})
}

// The POKéDEX is AutoSize: with no species selected, its panel hidden, it
// is its list and its button, and it grows to the panel with one. Shown
// again, its FormShow selects the species again, which leaves the text
// unselected: the focus is on the button that closed it.
func Test_pokedexForm(t *testing.T) {
	t.Parallel()

	t.Run("when no species is selected, should be only its list and its button", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)

		assert.Equal(t, image.Pt(241, 25), ta.pokedex.wdw.client, "the client area with no selection")
	})

	t.Run("when a species is chosen, should grow to show its panel", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		pickCharmander(t, ta)

		assert.Equal(t, image.Pt(241, 161), ta.pokedex.wdw.client, "the client area with Charmander")
		assert.Equal(t, 3, ta.pokedex.combo.index, "the species chosen")
	})

	t.Run("when it is shown again, should select the species with its text unselected and the focus on the button that closed it", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		pickCharmander(t, ta)

		dex := ta.pokedex
		ta.clickOn(dex.wdw, "Button1")
		ta.menu("Action1", "ViewDEX1")
		ta.wait(1)

		cb := dex.combo
		from, to := cb.tb.sel()
		assert.Equal(t, from, to, "the text should be unselected, but is selected from %d to %d", from, to)
		assert.Equal(t, "Charmander (#004)", cb.tb.String(), "the text")
		assert.Same(t, dex.wdw.find("Button1"), dex.wdw.focus, "the focus left the button")
	})

	t.Run("when its sprite is clicked, should neither select nor focus it", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		pickCharmander(t, ta)

		img := ta.pokedex.image
		ta.click(widgetOrigin(img).Add(img.itemRect(0).Min).Add(image.Pt(12, 12)))
		ta.wait(1)

		assert.Equal(t, -1, img.selected, "the sprite is selected")
		assert.NotSame(t, img, ta.pokedex.wdw.focus, "the sprite has the focus")
	})
}

// newDexApp starts a game whose player has a Charmander and has it in the
// POKéDEX, and opens the POKéDEX.
func newDexApp(t *testing.T) *testApp {
	t.Helper()

	sav := save.File{
		PlayerName: "Gary",
		PokeBalls:  3,
	}
	sav.Owned[0] = save.Slot{
		Species: 4,
		Level:   5,
		XP:      125,
	}
	sav.InPokedex[4] = true
	dir := t.TempDir()
	file := filepath.Join(dir, "dex.pgs")

	require.NoError(t, os.WriteFile(file, sav.Bytes(), 0o600))

	ta := newTestAppIn(t, dir, []string{file})
	ta.wait(80)
	ta.menu("Action1", "ViewDEX1")
	ta.wait(1)

	return ta
}

// pickCharmander opens the POKéDEX's list and chooses Charmander, the
// fourth item, which closes the list again.
func pickCharmander(t *testing.T, ta *testApp) {
	t.Helper()

	cb := ta.pokedex.combo
	rect := cb.button()
	ta.click(widgetOrigin(cb).Add(rect.Min).Add(rect.Size().Div(2)))

	require.NotNil(t, cb.dropped, "the list didn't drop down")

	drop := cb.dropped.rect
	ta.click(drop.Min.Add(image.Pt(5, 1+3*cb.itemHeight+cb.itemHeight/2)))
	ta.wait(1)
}
