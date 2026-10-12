// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"testing"
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
