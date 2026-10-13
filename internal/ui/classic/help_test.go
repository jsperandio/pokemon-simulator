// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/assets"
)

// spotPoint returns a point on the screen inside a link to target, after
// scrolling it into view.
func (ta *testApp) spotPoint(target string) image.Point {
	ta.t.Helper()

	v := ta.helpWin.view
	for _, s := range v.spots {
		if s.target != target || len(s.boxes) == 0 {
			continue
		}

		b := s.boxes[0]
		if b.Min.Y < v.scroll.pos || b.Max.Y > v.scroll.pos+v.scroll.page {
			v.scroll.set(b.Min.Y)
		}

		p := b.Min.Add(image.Pt(2, v.head.Rect.Dy()-v.scroll.pos+2))

		return widgetOrigin(v).Add(p)
	}

	ta.t.Fatalf("%s has no link to %s", v.name, target)

	return image.Point{}
}

func TestHelp(t *testing.T) {
	ta := newTestApp(t)
	ta.wait(80)
	ta.clickOn(ta.start.w, "HelpButton")
	ta.wait(1)

	h := ta.helpWin
	if h == nil || !h.w.visible || ta.desk.active != h.w {
		t.Fatal("no help window")
	}

	var opened []string
	h.openURL = func(url string) error {
		opened = append(opened, url)

		return nil
	}

	ta.shot("30-help")

	if h.view.name != "index.md" || h.back.enabled || h.w.caption != "POKéMON Simulator" {
		t.Errorf("help at %s, Back %v, caption %q", h.view.name, h.back.enabled, h.w.caption)
	}

	if got := h.w.rect(); got != image.Rect(39, 4, 39+722, 4+563) {
		t.Errorf("help window at %v", got)
	}

	followHelpLinks(t, ta)

	// Tab chooses the first hotspot, the port note's link, and Enter
	// follows it out to the browser.
	ta.key(ebiten.KeyTab)
	ta.shot("32-help-tab")
	ta.key(ebiten.KeyEnter)

	if len(opened) != 1 || !strings.HasSuffix(opened[0], "/pokemon-simulator/issues") {
		t.Errorf("opened %q", opened)
	}

	ta.click(h.w.captionButton(capMax).Min.Add(image.Pt(4, 4)))
	ta.wait(1)
	ta.shot("34-help-maximized")

	if !h.w.maximized || h.view.laid != h.view.r.Size() || h.view.r.Dx() != screenWidth {
		t.Errorf("maximized: %v, the topic %v laid out for %v", h.w.maximized, h.view.r, h.view.laid)
	}

	ta.click(h.w.captionButton(capClose).Min.Add(image.Pt(4, 4)))
	ta.wait(1)

	if h.w.visible || len(h.history) != 0 {
		t.Error("the help window didn't close")
	}
}

// followHelpLinks follows two links in the manual from its contents, one
// after the wheel scrolled, then goes Back to the contents.
func followHelpLinks(t *testing.T, ta *testApp) {
	h := ta.helpWin
	ta.click(ta.spotPoint("playing.md"))
	ta.wait(1)

	if h.view.name != "playing.md" || !h.back.enabled {
		t.Fatalf("help at %s after the link, Back %v", h.view.name, h.back.enabled)
	}

	ta.shot("31-help-playing")
	ta.frame(input{wheel: -2})

	pos := h.view.scroll.pos
	if pos != 6*h.view.scroll.line {
		t.Errorf("scrolled to %d by the wheel", pos)
	}

	p := ta.spotPoint("statistics.md") // scrolling to it
	pos = h.view.scroll.pos

	ta.click(p)
	ta.wait(1)

	if h.view.name != "statistics.md" || h.view.scroll.pos != 0 {
		t.Fatalf("help at %s, %d", h.view.name, h.view.scroll.pos)
	}

	ta.clickOn(h.w, "Back")
	ta.wait(1)

	if h.view.name != "playing.md" || h.view.scroll.pos != pos {
		t.Errorf("Back went to %s at %d", h.view.name, h.view.scroll.pos)
	}

	if ta.desk.active.focus != h.view {
		t.Error("the buttons kept the focus")
	}

	ta.clickOn(h.w, "Back")
	ta.wait(1)

	if h.view.name != "index.md" || h.back.enabled {
		t.Errorf("Back went to %s, Back %v", h.view.name, h.back.enabled)
	}
}

// F1 in the game window opens the help.
func TestHelpF1(t *testing.T) {
	tb := newTestApp(t)
	tb.startGame()
	tb.key(ebiten.KeyF1)
	tb.wait(1)

	if tb.helpWin == nil || !tb.helpWin.w.visible || tb.helpWin.view.name != "index.md" {
		t.Fatal("F1 shows no help")
	}

	tb.shot("33-help-game")
}

// Every page lays out, its links have boxes in the topic, and it shows
// once scrolled to its end.
func TestHelpPages(t *testing.T) {
	ta := newTestApp(t)
	ta.wait(80)
	ta.clickOn(ta.start.w, "HelpButton")
	ta.wait(1)

	v := ta.helpWin.view

	names, err := fs.Glob(assets.FS, "help/*.md")
	if err != nil {
		t.Fatal(err)
	}

	for _, n := range names {
		name := path.Base(n)
		v.open(name, 0)

		if v.name != name || v.doc == nil {
			t.Errorf("%s doesn't show", name)

			continue
		}

		for _, s := range v.spots {
			if len(s.boxes) == 0 {
				t.Errorf("%s: the link to %s has no box", name, s.target)
			}
		}

		ta.desk.dirty = true
		ta.shot("help-" + strings.TrimSuffix(name, ".md"))

		if v.scroll.enabled() {
			v.scroll.set(v.scroll.limit())
			ta.shot("help-" + strings.TrimSuffix(name, ".md") + "-end")
		}
	}
}
