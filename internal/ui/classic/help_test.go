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

func TestHelp(t *testing.T) {
	ta := newTestApp(t)
	ta.wait(80)
	ta.clickOn(ta.start.wdw, "HelpButton")
	ta.wait(1)

	hw := ta.helpWindow
	if hw == nil || !hw.wdw.visible || ta.desk.active != hw.wdw {
		t.Fatal("no help window")
	}

	var opened []string
	hw.openURL = func(url string) error {
		opened = append(opened, url)

		return nil
	}

	ta.shot("30-help")

	if hw.view.name != "index.md" || hw.back.enabled || hw.wdw.caption != "POKéMON Simulator" {
		t.Errorf("help at %s, Back %v, caption %q", hw.view.name, hw.back.enabled, hw.wdw.caption)
	}

	if got := hw.wdw.rect(); got != image.Rect(39, 4, 39+722, 4+563) {
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

	ta.click(onCaptionButton(hw.wdw, capMax))
	ta.wait(1)
	ta.shot("34-help-maximized")

	if !hw.wdw.maximized || hw.view.laid != hw.view.rect.Size() || hw.view.rect.Dx() != screenWidth {
		t.Errorf("maximized: %v, the topic %v laid out for %v", hw.wdw.maximized, hw.view.rect, hw.view.laid)
	}

	ta.click(onCaptionButton(hw.wdw, capClose))
	ta.wait(1)

	if hw.wdw.visible || len(hw.history) != 0 {
		t.Error("the help window didn't close")
	}
}

// onCaptionButton is a point inside caption button cbt of wdw.
func onCaptionButton(wdw *window, cbt int) image.Point {
	return wdw.captionButton(cbt).Min.Add(image.Pt(4, 4))
}

// followHelpLinks follows two links in the manual from its contents, one
// after the wheel scrolled, then goes Back to the contents.
func followHelpLinks(t *testing.T, ta *testApp) {
	hw := ta.helpWindow
	hv := hw.view

	ta.click(ta.spotPoint("playing.md"))
	ta.wait(1)

	if hv.name != "playing.md" || !hw.back.enabled {
		t.Fatalf("help at %s after the link, Back %v", hv.name, hw.back.enabled)
	}

	ta.shot("31-help-playing")
	ta.frame(input{wheel: -2})

	pos := hv.scroll.pos
	if pos != 6*hv.scroll.line {
		t.Errorf("scrolled to %d by the wheel", pos)
	}

	pnt := ta.spotPoint("statistics.md") // scrolling to it
	pos = hv.scroll.pos

	ta.click(pnt)
	ta.wait(1)

	if hv.name != "statistics.md" || hv.scroll.pos != 0 {
		t.Fatalf("help at %s, %d", hv.name, hv.scroll.pos)
	}

	ta.clickOn(hw.wdw, "Back")
	ta.wait(1)

	if hv.name != "playing.md" || hv.scroll.pos != pos {
		t.Errorf("Back went to %s at %d", hv.name, hv.scroll.pos)
	}

	if ta.desk.active.focus != hv {
		t.Error("the buttons kept the focus")
	}

	ta.clickOn(hw.wdw, "Back")
	ta.wait(1)

	if hv.name != "index.md" || hw.back.enabled {
		t.Errorf("Back went to %s, Back %v", hv.name, hw.back.enabled)
	}
}

// F1 in the game window opens the help.
func TestHelpF1(t *testing.T) {
	tb := newTestApp(t)
	tb.startGame()
	tb.key(ebiten.KeyF1)
	tb.wait(1)

	if tb.helpWindow == nil || !tb.helpWindow.wdw.visible || tb.helpWindow.view.name != "index.md" {
		t.Fatal("F1 shows no help")
	}

	tb.shot("33-help-game")
}

// Every page lays out, its links have boxes in the topic, and it shows
// once scrolled to its end.
func TestHelpPages(t *testing.T) {
	ta := newTestApp(t)
	ta.wait(80)
	ta.clickOn(ta.start.wdw, "HelpButton")
	ta.wait(1)

	hv := ta.helpWindow.view

	names, err := fs.Glob(assets.FS, "help/*.md")
	if err != nil {
		t.Fatal(err)
	}

	for _, nam := range names {
		name := path.Base(nam)
		hv.open(name, 0)

		if hv.name != name || hv.doc == nil {
			t.Errorf("%s doesn't show", name)

			continue
		}

		for _, hsp := range hv.spots {
			if len(hsp.boxes) == 0 {
				t.Errorf("%s: the link to %s has no box", name, hsp.target)
			}
		}

		ta.desk.dirty = true
		ta.shot("help-" + strings.TrimSuffix(name, ".md"))

		if hv.scroll.enabled() {
			hv.scroll.set(hv.scroll.limit())
			ta.shot("help-" + strings.TrimSuffix(name, ".md") + "-end")
		}
	}
}
