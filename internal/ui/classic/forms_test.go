// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The party list's hint shows after half a second on it, under the
// cursor and kept on the screen, and hides after two and a half seconds,
// when the mouse leaves the list, or at a click.
func TestStatsHint(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()
	l := ta.stats.party
	ta.mouse = widgetOrigin(l).Add(image.Pt(60, 100))
	ta.wait(29) // under half a second
	if !ta.desk.hint.r.Empty() {
		t.Fatal("the hint shows before HintPause")
	}
	ta.wait(4)
	h := ta.desk.hint
	ta.shot("50-stats-hint")
	if h.text != "Doubleclick a POKéMON to see its statistics." {
		t.Fatalf("hint %q", h.text)
	}
	// Under the cursor, and moved left to end at the screen's edge.
	size := image.Pt(ta.res.sans.width(h.text)+6, ta.res.sans.height()+6)
	o := image.Pt(screenWidth-size.X, ta.mouse.Y+hintCursorMargin)
	if want := (image.Rectangle{Min: o, Max: o.Add(size)}); h.r != want {
		t.Errorf("hint at %v, want %v", h.r, want)
	}
	ta.wait(140) // under three seconds on the list
	if ta.desk.hint.r.Empty() {
		t.Fatal("the hint hid before HintHidePause")
	}
	ta.wait(10)
	if !ta.desk.hint.r.Empty() {
		t.Fatal("the hint stays after HintHidePause")
	}
	ta.wait(60)
	if !ta.desk.hint.r.Empty() {
		t.Fatal("the hint came back on the same control")
	}

	// Off the list and back: it shows again, and a click hides it.
	ta.mouse = image.Pt(10, 10)
	ta.wait(1)
	ta.mouse = widgetOrigin(l).Add(image.Pt(60, 100))
	ta.wait(33)
	if ta.desk.hint.r.Empty() {
		t.Fatal("no hint on coming back")
	}
	ta.frame(input{down: true, pressed: true})
	ta.frame(input{released: true})
	if !ta.desk.hint.r.Empty() {
		t.Fatal("a click leaves the hint")
	}
	ta.mouse = image.Pt(10, 10)
	ta.wait(40)
	if !ta.desk.hint.r.Empty() {
		t.Fatal("a hint off the list")
	}
}

// The Game Corner's bet goes up and down by its Increment, 50, with the
// spin buttons and the arrow keys.
func TestSpinIncrement(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()
	ta.g.Money = 1000
	ta.menu("Action1", "GameCorner1")
	ta.wait(1)
	s := get[*spinEdit](ta.corner.w, "SpinEdit1")
	up, down := s.buttons()
	ta.click(widgetOrigin(s).Add(up.Min).Add(up.Size().Div(2)))
	ta.click(widgetOrigin(s).Add(up.Min).Add(up.Size().Div(2)))
	if ta.g.Corner.Bet != 100 {
		t.Errorf("bet %d after two clicks up, want 100", ta.g.Corner.Bet)
	}
	ta.click(widgetOrigin(s).Add(down.Min).Add(down.Size().Div(2)))
	ta.key(ebiten.KeyUp)
	ta.key(ebiten.KeyUp)
	if ta.g.Corner.Bet != 150 || s.tb.String() != "150" {
		t.Errorf("bet %d, text %q, want 150", ta.g.Corner.Bet, s.tb.String())
	}
}

// The start screen's minimize button minimizes the program's window.
func TestMinimize(t *testing.T) {
	ta := newTestApp(t)
	n := 0
	ta.desk.onMinimize = func() { n++ }
	ta.wait(80)
	w := ta.start.w
	if !w.minBox {
		t.Fatal("the start screen has no minimize button")
	}
	ta.click(w.captionButton(capMin).Min.Add(image.Pt(4, 4)))
	if n != 1 || !w.visible {
		t.Errorf("%d minimizes, start screen shown: %t", n, w.visible)
	}
}

// A play's draws show one after another in "Lucky number" for a second,
// the bet already paid in Statistics, with no input taken, and the last
// one stays: the one the game counts.
func TestCornerDraws(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()
	ta.g.Money = 1000
	ta.menu("Action1", "GameCorner1")
	ta.wait(1)
	c := ta.corner
	radio := func(g *radioGroup, k int) {
		r := g.buttonRect(k)
		ta.click(widgetOrigin(g).Add(r.Min).Add(image.Pt(6, r.Dy()/2)))
	}
	radio(c.prize, 0)  // money
	radio(c.number, 3) // on 3
	ta.click(widgetOrigin(c.bet).Add(image.Pt(5, 5)))
	ta.key(ebiten.KeyHome)
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyEnd}, mods: mods{shift: true}})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyEnd}})
	ta.typeText("100")
	ta.clickOn(c.w, "Button1") // Draw number!
	if ta.draws == nil {
		t.Fatal("no draws show")
	}
	if got := ta.stats.counts["MoneyLbl"].caption; got != "900" {
		t.Errorf("Statistics shows %s¥ during the draws, want the bet paid", got)
	}
	seen := map[int]bool{}
	frames := 0
	for ; ta.draws != nil && frames < 120; frames++ {
		seen[c.drawn.index] = true
		if frames == 20 {
			ta.shot("51-corner-draws")
			ta.clickOn(c.w, "Button2") // Close: no input meanwhile
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
	if !c.w.visible {
		t.Error("Close worked during the draws")
	}
	ta.wait(1)
	if c.drawn.index != ta.g.Corner.Drawn {
		t.Errorf("%d shows, the game drew %d", c.drawn.index, ta.g.Corner.Drawn)
	}
}
