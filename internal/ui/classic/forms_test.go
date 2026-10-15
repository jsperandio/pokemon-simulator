// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// The party list's hint shows after half a second on it, under the
// cursor and kept on the screen, and hides after two and a half seconds,
// when the mouse leaves the list, or at a click.
func TestStatsHint(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()

	lv := ta.stats.party
	ta.mouse = widgetOrigin(lv).Add(image.Pt(60, 100))
	ta.wait(29) // under half a second

	if ta.hintShown() {
		t.Fatal("the hint shows before HintPause")
	}

	ta.wait(4)

	hnt := ta.desk.hint
	ta.shot("50-stats-hint")

	if hnt.text != "Doubleclick a POKéMON to see its statistics." {
		t.Fatalf("hint %q", hnt.text)
	}

	// Under the cursor, and moved left to end at the screen's edge.
	size := image.Pt(ta.res.sans.width(hnt.text)+6, ta.res.sans.height()+6)
	pnt := image.Pt(screenWidth-size.X, ta.mouse.Y+hintCursorMargin)

	if want := (image.Rectangle{Min: pnt, Max: pnt.Add(size)}); hnt.rect != want {
		t.Errorf("hint at %v, want %v", hnt.rect, want)
	}

	ta.wait(140) // under three seconds on the list

	if !ta.hintShown() {
		t.Fatal("the hint hid before HintHidePause")
	}

	ta.wait(10)

	if ta.hintShown() {
		t.Fatal("the hint stays after HintHidePause")
	}

	ta.wait(60)

	if ta.hintShown() {
		t.Fatal("the hint came back on the same control")
	}

	// Off the list and back: it shows again, and a click hides it.
	ta.mouse = image.Pt(10, 10)
	ta.wait(1)

	ta.mouse = widgetOrigin(lv).Add(image.Pt(60, 100))
	ta.wait(33)

	if !ta.hintShown() {
		t.Fatal("no hint on coming back")
	}

	ta.frame(input{down: true, pressed: true})
	ta.frame(input{released: true})

	if ta.hintShown() {
		t.Fatal("a click leaves the hint")
	}

	ta.mouse = image.Pt(10, 10)
	ta.wait(40)

	if ta.hintShown() {
		t.Fatal("a hint off the list")
	}
}

// The Game Corner's bet goes up and down by its Increment, 50, with the
// spin buttons and the arrow keys.
func TestSpinIncrement(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()

	ta.gam.Money = 1000
	ta.menu("Action1", "GameCorner1")
	ta.wait(1)

	se := get[*spinEdit](ta.gameCorner.wdw, "SpinEdit1")
	up, down := se.buttons()
	ta.click(widgetOrigin(se).Add(up.Min).Add(up.Size().Div(2)))
	ta.click(widgetOrigin(se).Add(up.Min).Add(up.Size().Div(2)))

	if ta.gam.Corner.Bet != 100 {
		t.Errorf("bet %d after two clicks up, want 100", ta.gam.Corner.Bet)
	}

	ta.click(widgetOrigin(se).Add(down.Min).Add(down.Size().Div(2)))
	ta.key(ebiten.KeyUp)
	ta.key(ebiten.KeyUp)

	if ta.gam.Corner.Bet != 150 || se.tb.String() != "150" {
		t.Errorf("bet %d, text %q, want 150", ta.gam.Corner.Bet, se.tb.String())
	}
}

// The start screen's minimize button minimizes the program's window.
func TestMinimize(t *testing.T) {
	ta := newTestApp(t)
	n := 0
	ta.desk.onMinimize = func() {
		n++
	}
	ta.wait(80)

	wdw := ta.start.wdw
	if !wdw.minBox {
		t.Fatal("the start screen has no minimize button")
	}

	ta.click(wdw.captionButton(capMin).Min.Add(image.Pt(4, 4)))

	if n != 1 || !wdw.visible {
		t.Errorf("%d minimizes, start screen shown: %t", n, wdw.visible)
	}
}

// A play's draws show one after another in "Lucky number" for a second,
// the bet already paid in Statistics, with no input taken, and the last
// one stays: the one the game counts.
func TestCornerDraws(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()

	ta.gam.Money = 1000
	ta.menu("Action1", "GameCorner1")
	ta.wait(1)

	gcf := ta.gameCorner
	radio := func(rg *radioGroup, idx int) {
		rect := rg.buttonRect(idx)
		ta.click(widgetOrigin(rg).Add(rect.Min).Add(image.Pt(6, rect.Dy()/2)))
	}
	radio(gcf.prize, 0)  // money
	radio(gcf.number, 3) // on 3
	ta.click(widgetOrigin(gcf.bet).Add(image.Pt(5, 5)))
	ta.key(ebiten.KeyHome)
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyEnd}, mods: mods{shift: true}})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyEnd}})
	ta.typeText("100")
	ta.clickOn(gcf.wdw, "Button1") // Draw number!

	if ta.draws == nil {
		t.Fatal("no draws show")
	}

	if got := ta.stats.counts["MoneyLbl"].caption; got != "900" {
		t.Errorf("Statistics shows %s¥ during the draws, want the bet paid", got)
	}

	seen := map[int]bool{}
	frames := 0

	for ; ta.draws != nil && frames < 120; frames++ {
		seen[gcf.drawn.index] = true

		if frames == 20 {
			ta.shot("51-corner-draws")
			ta.clickOn(gcf.wdw, "Button2") // Close: no input meanwhile

			frames += 2
		}

		ta.wait(1)
	}

	if frames < 57 || frames > 63 {
		t.Errorf("the draws showed for %d frames, want about a second", frames)
	}

	if len(seen) < 8 {
		t.Errorf("%d numbers shown", len(seen))
	}

	if !gcf.wdw.visible {
		t.Error("Close worked during the draws")
	}

	ta.wait(1)

	if gcf.drawn.index != ta.gam.Corner.Drawn {
		t.Errorf("%d shows, the game drew %d", gcf.drawn.index, ta.gam.Corner.Drawn)
	}
}

// The loading screen is AutoSize: it grows to its title, a pixel wider in
// Arial than the form, after Position centred it, and the labels anchored
// to its right move with it.
func TestLoadingFits(t *testing.T) {
	wdw := newTestApp(t).load.wdw
	if wdw.client != image.Pt(511, 157) || wdw.pos.X != (screenWidth-510)/2 {
		t.Errorf("client %v at %v, want 511×157 centred as 510 wide", wdw.client, wdw.pos)
	}

	for name, x := range map[string]int{
		"Label1": 216,
		"Label2": 265,
		"Label3": 12,
		"Label4": 29,
	} {
		rect := wdw.find(name).base().rect
		if rect.Min.X != x {
			t.Errorf("%s at x=%d, want %d", name, rect.Min.X, x)
		}
	}
}

// The POKéDEX is AutoSize: with no species selected, its panel hidden, it
// is its list and its button, and it grows to the panel with one. Shown
// again, its FormShow selects the species again, which leaves the text
// unselected: the focus is on the button that closed it.
func TestDexFits(t *testing.T) {
	sav := save.File{PlayerName: "Gary", PokeBalls: 3}
	sav.Owned[0] = save.Slot{Species: 4, Level: 5, XP: 125}
	sav.InDex[4] = true
	dir := t.TempDir()
	file := filepath.Join(dir, "dex.pgs")

	if err := os.WriteFile(file, sav.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	ta := newTestAppIn(t, dir, []string{file})
	ta.wait(80)
	ta.menu("Action1", "ViewDEX1")
	ta.wait(1)

	dex := ta.pokedex
	if dex.wdw.client != image.Pt(241, 25) {
		t.Errorf("client %v with no selection, want 241×25", dex.wdw.client)
	}

	cb := dex.combo
	rect := cb.button()
	ta.click(widgetOrigin(cb).Add(rect.Min).Add(rect.Size().Div(2)))

	drop := cb.dropped.rect
	ta.click(drop.Min.Add(image.Pt(5, 1+3*cb.itemHeight+cb.itemHeight/2)))
	ta.wait(1)

	if dex.wdw.client != image.Pt(241, 161) || cb.index != 3 {
		t.Errorf("client %v with species %d, want 241×161 with Charmander", dex.wdw.client, cb.index+1)
	}

	ta.clickOn(dex.wdw, "Button1")
	ta.menu("Action1", "ViewDEX1")
	ta.wait(1)

	if from, to := cb.tb.sel(); from != to || cb.tb.String() != "Charmander (#004)" {
		t.Errorf("text %q, selected from %d to %d; want Charmander's, unselected", cb.tb.String(), from, to)
	}

	if dex.wdw.focus != dex.wdw.find("Button1") {
		t.Error("the focus left the button")
	}
}
