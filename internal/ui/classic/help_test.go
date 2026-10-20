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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
)

// The manual opens from the start screen's Help button and from F1 in the
// game window; every page lays out, and shows once scrolled to its end.
func Test_helpWindow(t *testing.T) {
	t.Parallel()

	// The help's flow is one subtest: Tab chooses the hotspot after the
	// links were followed, and the screenshots show that state.
	t.Run("when the help opens from the start screen, should show the contents, follow the links, open the port note in the browser, and maximize and close", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.wait(80)
		ta.clickOn(ta.start.wdw, "HelpButton")
		ta.wait(1)

		hw := ta.helpWindow
		require.NotNil(t, hw, "no help window")
		require.True(t, hw.wdw.visible, "the help window isn't shown")
		require.Same(t, hw.wdw, ta.desk.active, "the help window isn't active")

		var opened []string
		hw.openURL = func(url string) error {
			opened = append(opened, url)

			return nil
		}

		ta.shot("30-help")

		assert.Equal(t, "index.md", hw.view.name, "the help's page")
		assert.False(t, hw.back.enabled, "Back at the contents")
		assert.Equal(t, "POKéMON Simulator", hw.wdw.caption, "the caption")
		assert.Equal(t, image.Rect(39, 4, 39+722, 4+563), hw.wdw.rect(), "the help window's rectangle")

		followHelpLinks(t, ta)

		// Tab chooses the first hotspot, the port note's link, and Enter
		// follows it out to the browser.
		ta.key(ebiten.KeyTab)
		ta.shot("32-help-tab")
		ta.key(ebiten.KeyEnter)

		require.Len(t, opened, 1, "opened %q", opened)
		assert.True(t, strings.HasSuffix(opened[0], "/pokemon-simulator/issues"), "opened %q", opened)

		ta.click(onCaptionButton(hw.wdw, capMax))
		ta.wait(1)
		ta.shot("34-help-maximized")

		assert.True(t, hw.wdw.maximized, "the help window is maximized")
		assert.Equal(t, hw.view.rect.Size(), hw.view.laid, "the topic laid out for its size")
		assert.Equal(t, screenWidth, hw.view.rect.Dx(), "the topic's width")

		ta.click(onCaptionButton(hw.wdw, capClose))
		ta.wait(1)

		assert.False(t, hw.wdw.visible, "the help window didn't close")
		assert.Empty(t, hw.history, "the help's history after it closed")
	})

	// F1 in the game window opens the help.
	t.Run("when F1 is pressed in the game window, should open the help at its contents", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.key(ebiten.KeyF1)
		ta.wait(1)

		hw := ta.helpWindow
		require.NotNil(t, hw, "F1 shows no help")
		require.True(t, hw.wdw.visible, "F1 shows no help")
		require.Equal(t, "index.md", hw.view.name, "F1 shows no help")

		ta.shot("33-help-game")
	})

	// Every page lays out, its links have boxes in the topic, and it shows
	// once scrolled to its end.
	t.Run("when each page is shown, should lay out its links with boxes and show its end", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.wait(80)
		ta.clickOn(ta.start.wdw, "HelpButton")
		ta.wait(1)

		hv := ta.helpWindow.view

		names, err := fs.Glob(assets.FS, "help/*.md")
		require.NoError(t, err)

		for _, pth := range names {
			nam := path.Base(pth)
			hv.open(nam, 0)

			if !assert.True(t, hv.name == nam && hv.doc != nil, "%s doesn't show", nam) {
				continue
			}

			for _, hsp := range hv.spots {
				assert.NotEmpty(t, hsp.boxes, "%s: the link to %s has no box", nam, hsp.target)
			}

			ta.desk.dirty = true
			ta.shot("help-" + strings.TrimSuffix(nam, ".md"))

			if hv.scroll.enabled() {
				hv.scroll.set(hv.scroll.limit())
				ta.shot("help-" + strings.TrimSuffix(nam, ".md") + "-end")
			}
		}
	})
}

// onCaptionButton is a point inside caption button cbt of wdw.
func onCaptionButton(wdw *window, cbt int) image.Point {
	return wdw.captionButton(cbt).Min.Add(image.Pt(4, 4))
}

// followHelpLinks follows two links in the manual from its contents, one
// after the wheel scrolled, then goes Back to the contents.
func followHelpLinks(t *testing.T, ta *testApp) {
	t.Helper()

	hw := ta.helpWindow
	hv := hw.view

	ta.click(ta.spotPoint("playing.md"))
	ta.wait(1)

	require.Equal(t, "playing.md", hv.name, "the link to playing.md")
	require.True(t, hw.back.enabled, "Back after the link")

	ta.shot("31-help-playing")
	ta.frame(input{
		wheel: -2,
	})

	pos := hv.scroll.pos
	assert.Equal(t, 6*hv.scroll.line, pos, "scrolled to %d by the wheel", pos)

	pnt := ta.spotPoint("statistics.md") // scrolling to it
	pos = hv.scroll.pos

	ta.click(pnt)
	ta.wait(1)

	require.Equal(t, "statistics.md", hv.name, "the link to statistics.md")
	require.Zero(t, hv.scroll.pos, "the top of statistics.md")

	ta.clickOn(hw.wdw, "Back")
	ta.wait(1)

	assert.Equal(t, "playing.md", hv.name, "the page Back goes to")
	assert.Equal(t, pos, hv.scroll.pos, "the scroll Back goes to")
	assert.Same(t, hv, ta.desk.active.focus, "the buttons kept the focus")

	ta.clickOn(hw.wdw, "Back")
	ta.wait(1)

	assert.Equal(t, "index.md", hv.name, "the page the second Back goes to")
	assert.False(t, hw.back.enabled, "Back at the contents")
}
