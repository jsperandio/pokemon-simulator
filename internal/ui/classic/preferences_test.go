// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"errors"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_loadPreferences(t *testing.T) {
	t.Parallel()

	t.Run("when there is no file, should animate the sprites and show the choose prompt", func(t *testing.T) {
		t.Parallel()

		prf := loadPreferences(nil)
		assert.True(t, prf.animatedSprites)
		assert.True(t, prf.choosePrompt)
		assert.Equal(t, goldSparkles, prf.shinyMark)
		assert.False(t, prf.fillScreen, "the screen is filled")
		assert.True(t, prf.colourMessages, "Battle Info isn't coloured")
		assert.False(t, prf.whiteBackground, "Battle Info is white")
	})

	t.Run("when the file fills the screen, should fill it only", func(t *testing.T) {
		t.Parallel()

		prf := loadPreferences([]byte("[Layout]\nFillScreen=1\n"))
		assert.True(t, prf.fillScreen)
		assert.True(t, prf.animatedSprites)
	})

	t.Run("when the file turns them off, should not animate the sprites", func(t *testing.T) {
		t.Parallel()

		assert.False(t, loadPreferences([]byte("[Graphics]\r\nAnimatedSprites=0\r\n")).animatedSprites)
	})

	t.Run("when the section and key differ in case, should still read them", func(t *testing.T) {
		t.Parallel()

		assert.False(t, loadPreferences([]byte("[graphics]\n animatedsprites = 0\n")).animatedSprites)
	})

	t.Run("when the value isn't a number, should keep the default", func(t *testing.T) {
		t.Parallel()

		assert.True(t, loadPreferences([]byte("[Graphics]\nAnimatedSprites=no\n")).animatedSprites)
	})

	t.Run("when the value is a number other than 0, should animate the sprites", func(t *testing.T) {
		t.Parallel()

		assert.True(t, loadPreferences([]byte("[Graphics]\nAnimatedSprites=-1\n")).animatedSprites)
	})

	t.Run("when the key is in another section, should keep the default", func(t *testing.T) {
		t.Parallel()

		assert.True(t, loadPreferences([]byte("[Sound]\nAnimatedSprites=0\n")).animatedSprites)
	})

	t.Run("when the file hides the choose prompt, should hide it only", func(t *testing.T) {
		t.Parallel()

		prf := loadPreferences([]byte("[Messages]\nSelectPokemon=0\n"))
		assert.False(t, prf.choosePrompt)
		assert.True(t, prf.animatedSprites)
	})

	t.Run("when the file names a shiny mark, should read it whatever its case", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, thinTwinkles, loadPreferences([]byte("[Graphics]\nshinymark = thintwinkles\n")).shinyMark)
	})

	t.Run("when the shiny mark isn't known, should keep the default", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, goldSparkles, loadPreferences([]byte("[Graphics]\nShinyMark=Rainbow\n")).shinyMark)
	})

	t.Run("when the file turns Battle Info's colours off and whitens it, should read both", func(t *testing.T) {
		t.Parallel()

		prf := loadPreferences([]byte("[BattleInfo]\nColourMessages=0\nWhiteBackground=1\n"))
		assert.False(t, prf.colourMessages)
		assert.True(t, prf.whiteBackground)
		assert.True(t, prf.animatedSprites)
	})
}

func Test_preferences_store(t *testing.T) {
	t.Parallel()

	off := preferences{
		animatedSprites: false,
		shinyMark:       noMark,
		choosePrompt:    false,
	}

	t.Run("when there is no file, should write a new one", func(t *testing.T) {
		t.Parallel()

		want := "[Graphics]\nAnimatedSprites=0\nShinyMark=None\n\n[Messages]\nSelectPokemon=0\n\n[Layout]\nFillScreen=0\n\n[BattleInfo]\nColourMessages=0\nWhiteBackground=0\n"
		assert.Equal(t, want, string(off.store(nil)))
	})

	t.Run("when the keys are there, should change their lines only", func(t *testing.T) {
		t.Parallel()

		dat := "; comment\n[Graphics]\nShinyMark=OneTwinkle\nTheme=classic\nAnimatedSprites=1\n\n[Messages]\nSelectPokemon=1\n\n[Sound]\nVolume=7\n"
		want := "; comment\n[Graphics]\nShinyMark=None\nTheme=classic\nAnimatedSprites=0\n\n[Messages]\nSelectPokemon=0\n\n[Sound]\nVolume=7\n\n[Layout]\nFillScreen=0\n\n[BattleInfo]\nColourMessages=0\nWhiteBackground=0\n"
		assert.Equal(t, want, string(off.store([]byte(dat))))
	})

	t.Run("when the section has other keys, should add the key after them", func(t *testing.T) {
		t.Parallel()

		dat := "[Graphics]\nTheme=classic\n\n[Messages]\nSelectPokemon=1\n"
		want := "[Graphics]\nTheme=classic\nAnimatedSprites=0\nShinyMark=None\n\n[Messages]\nSelectPokemon=0\n\n[Layout]\nFillScreen=0\n\n[BattleInfo]\nColourMessages=0\nWhiteBackground=0\n"
		assert.Equal(t, want, string(off.store([]byte(dat))))
	})

	t.Run("when the sections are missing, should add them at the end", func(t *testing.T) {
		t.Parallel()

		want := "[Sound]\nVolume=7\n\n[Graphics]\nAnimatedSprites=0\nShinyMark=None\n\n[Messages]\nSelectPokemon=0\n\n[Layout]\nFillScreen=0\n\n[BattleInfo]\nColourMessages=0\nWhiteBackground=0\n"
		assert.Equal(t, want, string(off.store([]byte("[Sound]\r\nVolume=7\r\n"))))
	})

	t.Run("when stored and loaded again, should give the same preferences", func(t *testing.T) {
		t.Parallel()

		mixed := preferences{
			animatedSprites: false,
			shinyMark:       outlinedTwinkles,
			choosePrompt:    true,
			fillScreen:      true,
			colourMessages:  true,
			whiteBackground: true,
		}

		assert.Equal(t, off, loadPreferences(off.store(nil)))
		assert.Equal(t, mixed, loadPreferences(mixed.store(nil)))
		assert.Equal(t, defaultPreferences(), loadPreferences(defaultPreferences().store(nil)))
	})
}

// keptOn is what the test app's settings hold once the window keeps the
// animations on, with the default shiny mark, the choose prompt shown and
// Battle Info's colours still off.
const keptOn string = "[Graphics]\nAnimatedSprites=1\nShinyMark=GoldSparkles\n\n[BattleInfo]\nColourMessages=0\nWhiteBackground=0\n\n[Messages]\nSelectPokemon=1\n\n[Layout]\nFillScreen=0\n"

// openPreferences plays up to the game window, the first time, and opens
// File → Preferences....
func openPreferences(ta *testApp) *preferencesForm {
	ta.t.Helper()

	if !ta.main.wdw.visible {
		ta.startGame()
	}

	ta.menu("File1", "Preferences1")
	ta.wait(1)

	pf := ta.prefsForm
	require.Same(ta.t, pf.wdw, ta.desk.topModal(), "the Preferences window isn't up")

	return pf
}

// chooseCategory clicks the window's category idx.
func chooseCategory(ta *testApp, pf *preferencesForm, idx int) {
	ta.t.Helper()

	lbx := pf.categories
	ta.click(widgetOrigin(lbx).Add(lbx.itemRect(idx).Min).Add(image.Pt(4, 4)))
	ta.wait(1)
}

func Test_preferencesForm(t *testing.T) {
	t.Parallel()

	t.Run("when File → Preferences... is chosen, should show Graphics with the player's choice and nothing to apply", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		ta.shot("prefs-open")

		assert.Equal(t, "Preferences", pf.wdw.caption)
		assert.Equal(t, []string{"Graphics", "Messages", "Layout", "Battle Info"}, pf.categories.items)
		assert.Equal(t, 0, pf.categories.selected)
		assert.True(t, pf.pages[0].shown(), "the Graphics page is hidden")
		assert.False(t, pf.pages[1].shown(), "the Messages page shows")
		assert.False(t, pf.animated.checked, "the test app's settings turn the animations off")
		assert.True(t, pf.prompt.checked, "the choose prompt is hidden")
		assert.False(t, pf.apply.enabled, "Apply is enabled with nothing to apply")
		assert.Same(t, pf.categories, pf.wdw.focus, "the category list hasn't the focus")
	})

	t.Run("when the box is checked and OK clicked, should animate and keep the choice", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		ta.clickOn(pf.wdw, "AnimatedSprites")

		assert.True(t, pf.animated.checked)
		assert.True(t, pf.apply.enabled, "Apply is greyed with a change to apply")

		ta.shot("prefs-checked")
		ta.clickOn(pf.wdw, "OKButton")

		assert.Nil(t, ta.desk.topModal(), "the window is still up")
		assert.True(t, ta.preferences.animatedSprites)
		assert.Equal(t, keptOn, string(ta.settings.(*memorySettings).data))
	})

	t.Run("when Cancel is clicked, should keep the old choice", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		ta.clickOn(pf.wdw, "AnimatedSprites")
		ta.clickOn(pf.wdw, "CancelButton")

		assert.Nil(t, ta.desk.topModal(), "the window is still up")
		assert.False(t, ta.preferences.animatedSprites)
		assert.Equal(t, offSettings, string(ta.settings.(*memorySettings).data))
	})

	t.Run("when Apply is clicked, should keep the choice and stay open with nothing to apply", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		ta.clickOn(pf.wdw, "AnimatedSprites")
		ta.clickOn(pf.wdw, "ApplyButton")

		assert.Same(t, pf.wdw, ta.desk.topModal(), "the window closed")
		assert.True(t, ta.preferences.animatedSprites)
		assert.Equal(t, keptOn, string(ta.settings.(*memorySettings).data))
		assert.False(t, pf.apply.enabled, "Apply is enabled after applying")
	})

	t.Run("when the keyboard is used, should toggle with the accelerator and cancel with Escape", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		ta.typeText("e")

		assert.True(t, pf.animated.checked, "E didn't toggle the box")
		assert.Same(t, pf.animated, pf.wdw.focus, "the box hasn't the focus")

		ta.key(ebiten.KeySpace)
		assert.False(t, pf.animated.checked, "Space didn't toggle the box")

		ta.key(ebiten.KeySpace)
		ta.key(ebiten.KeyEscape)

		assert.Nil(t, ta.desk.topModal(), "Escape didn't close the window")
		assert.False(t, ta.preferences.animatedSprites)
	})

	t.Run("when the choice can't be kept, should say so and stay open", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.settings.(*memorySettings).err = errors.New("disk full")
		pf := openPreferences(ta)
		ta.clickOn(pf.wdw, "AnimatedSprites")
		ta.clickOn(pf.wdw, "OKButton")
		ta.shot("prefs-failed")

		ta.answer("The preferences couldn't be saved: disk full")
		assert.Same(t, pf.wdw, ta.desk.topModal(), "the window closed")
		assert.True(t, ta.preferences.animatedSprites, "the choice doesn't hold for the run")
		assert.True(t, pf.apply.enabled, "Apply is greyed, so the player can't try again")

		ta.settings.(*memorySettings).err = nil
		ta.clickOn(pf.wdw, "ApplyButton")
		assert.Equal(t, keptOn, string(ta.settings.(*memorySettings).data))
	})

	t.Run("when Messages is chosen and its box cleared, should hide the choose prompt", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		chooseCategory(ta, pf, 1)

		assert.False(t, pf.pages[0].shown(), "the Graphics page shows")
		assert.True(t, pf.pages[1].shown(), "the Messages page is hidden")

		ta.clickOn(pf.wdw, "ChoosePrompt")
		ta.shot("prefs-messages")

		assert.False(t, pf.prompt.checked)
		assert.True(t, pf.apply.enabled, "Apply is greyed with a change to apply")

		ta.clickOn(pf.wdw, "OKButton")

		want := "[Graphics]\nAnimatedSprites=0\nShinyMark=GoldSparkles\n\n[BattleInfo]\nColourMessages=0\nWhiteBackground=0\n\n[Messages]\nSelectPokemon=0\n\n[Layout]\nFillScreen=0\n"

		assert.False(t, ta.preferences.choosePrompt)
		assert.Equal(t, want, string(ta.settings.(*memorySettings).data))
	})

	t.Run("when Restore Defaults is clicked, should reset every page for OK or Apply to keep", func(t *testing.T) {
		t.Parallel()

		settings := &memorySettings{
			data: []byte("[Graphics]\nAnimatedSprites=0\nShinyMark=None\n\n[Messages]\nSelectPokemon=0\n\n[Layout]\nFillScreen=1\n\n[BattleInfo]\nColourMessages=0\nWhiteBackground=1\n"),
		}
		ta := newTestAppWith(t, t.TempDir(), nil, settings)
		pf := openPreferences(ta)
		require.False(t, pf.animated.checked)
		require.False(t, pf.prompt.checked)
		require.Equal(t, int(noMark), pf.mark.index)
		require.Equal(t, windowsFill, pf.windows.index)
		require.False(t, pf.colours.checked)
		require.Equal(t, backgroundWhite, pf.background.index)

		ta.clickOn(pf.wdw, "RestoreButton")

		assert.True(t, pf.animated.checked, "the animations are still off")
		assert.True(t, pf.prompt.checked, "the choose prompt is still hidden")
		assert.Equal(t, int(goldSparkles), pf.mark.index, "the mark is still None")
		assert.Equal(t, windowsSeparate, pf.windows.index, "the screen is still filled")
		assert.True(t, pf.colours.checked, "Battle Info's colours are still off")
		assert.Equal(t, backgroundGrey, pf.background.index, "Battle Info is still white")
		assert.True(t, pf.apply.enabled, "Apply is greyed with a change to apply")
		assert.False(t, ta.preferences.choosePrompt, "restored before OK or Apply")

		ta.clickOn(pf.wdw, "CancelButton")

		kept := preferences{
			shinyMark:       noMark,
			fillScreen:      true,
			whiteBackground: true,
		}
		assert.Equal(t, kept, ta.preferences, "Cancel kept the defaults")

		pf = openPreferences(ta)
		ta.clickOn(pf.wdw, "RestoreButton")
		ta.clickOn(pf.wdw, "OKButton")

		want := "[Graphics]\nAnimatedSprites=1\nShinyMark=GoldSparkles\n\n[Messages]\nSelectPokemon=1\n\n[Layout]\nFillScreen=0\n\n[BattleInfo]\nColourMessages=1\nWhiteBackground=0\n"

		assert.Equal(t, defaultPreferences(), ta.preferences)
		assert.Equal(t, want, string(settings.data))
		assert.Equal(t, image.Pt(74, 54), ta.main.wdw.pos, "the game window isn't back beside the others")
	})

	t.Run("when a shiny mark is chosen and OK clicked, should keep it and mark the preview with it", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)

		assert.Equal(t, markNames(), pf.mark.items)
		assert.Equal(t, int(goldSparkles), pf.mark.index)

		ta.clickOn(pf.wdw, "ShinyMark")
		ta.shot("prefs-shiny-marks")
		ta.key(ebiten.KeyEscape)
		ta.key(ebiten.KeyDown)
		ta.shot("prefs-shiny-mark")

		assert.Equal(t, int(goldSparklesOutlined), pf.mark.index)
		assert.True(t, pf.apply.enabled, "Apply is greyed with a change to apply")
		assert.Equal(t, markBrown, iconPixel(ta, pf.preview, image.Pt(11, 1)), "the preview hasn't the outlined mark")

		ta.clickOn(pf.wdw, "OKButton")

		want := "[Graphics]\nAnimatedSprites=0\nShinyMark=GoldSparklesOutlined\n\n[BattleInfo]\nColourMessages=0\nWhiteBackground=0\n\n[Messages]\nSelectPokemon=1\n\n[Layout]\nFillScreen=0\n"

		assert.Nil(t, ta.desk.topModal(), "the window is still up")
		assert.Equal(t, goldSparklesOutlined, ta.preferences.shinyMark)
		assert.Equal(t, want, string(ta.settings.(*memorySettings).data))
	})

	t.Run("when the game starts again, should read the kept choice", func(t *testing.T) {
		t.Parallel()

		settings := &memorySettings{}
		ta := newTestAppWith(t, t.TempDir(), nil, settings)
		require.True(t, ta.preferences.animatedSprites, "no settings should animate")

		pf := openPreferences(ta)
		ta.clickOn(pf.wdw, "AnimatedSprites")
		ta.clickOn(pf.wdw, "OKButton")

		tb := newTestAppWith(t, t.TempDir(), nil, settings)
		assert.False(t, tb.preferences.animatedSprites)
	})

	t.Run("when Layout is chosen, should show the Windows choice at separate windows", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		chooseCategory(ta, pf, 2)
		ta.shot("prefs-layout")

		assert.True(t, pf.pages[2].shown(), "the Layout page is hidden")
		assert.False(t, pf.pages[0].shown(), "the Graphics page shows")
		assert.Equal(t, windowsSeparate, pf.windows.index)
	})

	t.Run("when Fill the screen is chosen and OK clicked, should dock the windows at once and keep the choice", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		chooseCategory(ta, pf, 2)
		clickRadio(ta, pf.windows, windowsFill)

		assert.True(t, pf.apply.enabled, "Apply is greyed with a change to apply")

		ta.clickOn(pf.wdw, "OKButton")

		assert.True(t, ta.preferences.fillScreen)
		assert.Equal(t, image.Rect(0, 0, 617, 416), ta.main.wdw.rect(), "the game window")
		assert.Equal(t, image.Rect(617, 0, 800, 600), ta.stats.wdw.rect(), "Statistics")
		assert.Contains(t, string(ta.settings.(*memorySettings).data), "[Layout]\nFillScreen=1\n")
	})

	t.Run("when separate windows are chosen again with Apply, should put the windows side by side with their own sizes", func(t *testing.T) {
		t.Parallel()

		settings := &memorySettings{
			data: []byte("[Graphics]\nAnimatedSprites=0\n\n[Layout]\nFillScreen=1\n"),
		}
		ta := newTestAppWith(t, t.TempDir(), nil, settings)
		pf := openPreferences(ta)

		require.Equal(t, image.Point{}, ta.main.wdw.pos, "the screen isn't filled to begin with")

		chooseCategory(ta, pf, 2)
		clickRadio(ta, pf.windows, windowsSeparate)
		ta.clickOn(pf.wdw, "ApplyButton")

		assert.False(t, ta.preferences.fillScreen)
		assert.Equal(t, image.Pt(74, 54), ta.main.wdw.pos, "the game window's place")
		assert.Equal(t, image.Pt(464, 270), ta.main.wdw.client, "the game window's size")
		assert.Equal(t, image.Pt(177, 473), ta.stats.wdw.client, "Statistics' size")
	})

	t.Run("when Battle Info is chosen, should show the player's colours and background, and the preview as they have it", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		chooseCategory(ta, pf, 3)
		ta.shot("prefs-battle-info")

		assert.True(t, pf.pages[3].shown(), "the Battle Info page is hidden")
		assert.False(t, pf.pages[2].shown(), "the Layout page shows")
		assert.False(t, pf.colours.checked, "the test app's settings turn the colours off")
		assert.Equal(t, backgroundGrey, pf.background.index)
		assert.Equal(t, previewLines, pf.logPreview.lines)
		assert.Nil(t, pf.logPreview.styles, "the preview is coloured")
		assert.Equal(t, colFace, pf.logPreview.color, "the preview isn't grey")
		assert.False(t, pf.logPreview.bar, "the preview has a scroll bar")
	})

	t.Run("when the colours are checked, should colour the preview at once and Battle Info once applied", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		chooseCategory(ta, pf, 3)
		ta.clickOn(pf.wdw, "ColourMessages")
		ta.shot("prefs-battle-info-grey")

		want := []lineStyle{
			{
				color: inkGreen.grey,
			},
			{
				color: inkRed.grey,
			},
			{
				color: inkBlue.grey,
			},
			{
				color: inkBlue.grey,
				bold:  true,
			},
		}

		assert.Equal(t, want, pf.logPreview.styles)
		assert.True(t, pf.apply.enabled, "Apply is greyed with a change to apply")
		assert.Nil(t, ta.info.memo.styles, "Battle Info is coloured before Apply")

		ta.clickOn(pf.wdw, "ApplyButton")

		// The log has its design-time line and the starter's POKéDEX entry.
		coloured := []lineStyle{
			{
				color: colText,
			},
			{
				color: inkPurple.grey,
			},
		}

		assert.True(t, ta.preferences.colourMessages)
		assert.Equal(t, coloured, ta.info.memo.styles, "Battle Info isn't coloured")
		assert.Contains(t, string(ta.settings.(*memorySettings).data), "[BattleInfo]\nColourMessages=1\nWhiteBackground=0\n")
	})

	t.Run("when White is chosen and OK clicked, should whiten the preview, then Battle Info, and keep the choice", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		chooseCategory(ta, pf, 3)
		ta.clickOn(pf.wdw, "ColourMessages")
		clickRadio(ta, pf.background, backgroundWhite)
		ta.shot("prefs-battle-info-white")

		assert.Equal(t, colWindow, pf.logPreview.color, "the preview isn't white")
		assert.Equal(t, inkGreen.white, pf.logPreview.styles[0].color, "the preview's first line")
		assert.Equal(t, colFace, ta.info.memo.color, "Battle Info is white before OK")

		ta.clickOn(pf.wdw, "OKButton")

		assert.True(t, ta.preferences.whiteBackground)
		assert.Equal(t, colWindow, ta.info.memo.color, "Battle Info isn't white")
		assert.Contains(t, string(ta.settings.(*memorySettings).data), "[BattleInfo]\nColourMessages=1\nWhiteBackground=1\n")
	})
}
