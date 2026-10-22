// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_loadPreferences(t *testing.T) {
	t.Parallel()

	t.Run("when there is no file, should animate the sprites", func(t *testing.T) {
		t.Parallel()

		assert.True(t, loadPreferences(nil).animatedSprites)
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
}

func Test_preferences_store(t *testing.T) {
	t.Parallel()

	off := preferences{
		animatedSprites: false,
	}

	t.Run("when there is no file, should write a new one", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "[Graphics]\nAnimatedSprites=0\n", string(off.store(nil)))
	})

	t.Run("when the key is there, should change its line only", func(t *testing.T) {
		t.Parallel()

		dat := "; comment\n[Graphics]\nTheme=classic\nAnimatedSprites=1\n\n[Sound]\nVolume=7\n"
		want := "; comment\n[Graphics]\nTheme=classic\nAnimatedSprites=0\n\n[Sound]\nVolume=7\n"
		assert.Equal(t, want, string(off.store([]byte(dat))))
	})

	t.Run("when the section has other keys, should add the key after them", func(t *testing.T) {
		t.Parallel()

		dat := "[Graphics]\nTheme=classic\n\n[Sound]\nVolume=7\n"
		want := "[Graphics]\nTheme=classic\nAnimatedSprites=0\n\n[Sound]\nVolume=7\n"
		assert.Equal(t, want, string(off.store([]byte(dat))))
	})

	t.Run("when the section is missing, should add it at the end", func(t *testing.T) {
		t.Parallel()

		want := "[Sound]\nVolume=7\n\n[Graphics]\nAnimatedSprites=0\n"
		assert.Equal(t, want, string(off.store([]byte("[Sound]\r\nVolume=7\r\n"))))
	})

	t.Run("when stored and loaded again, should give the same preferences", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, off, loadPreferences(off.store(nil)))
		assert.Equal(t, defaultPreferences(), loadPreferences(defaultPreferences().store(nil)))
	})
}

// openPreferences plays up to the game window and opens File →
// Preferences....
func openPreferences(ta *testApp) *preferencesForm {
	ta.t.Helper()

	ta.startGame()
	ta.menu("File1", "Preferences1")
	ta.wait(1)

	pf := ta.prefsForm
	require.Same(ta.t, pf.wdw, ta.desk.topModal(), "the Preferences window isn't up")

	return pf
}

func Test_preferencesForm(t *testing.T) {
	t.Parallel()

	t.Run("when File → Preferences... is chosen, should show Graphics with the player's choice and nothing to apply", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		ta.shot("prefs-open")

		assert.Equal(t, "Preferences", pf.wdw.caption)
		assert.Equal(t, []string{"Graphics"}, pf.categories.items)
		assert.Equal(t, 0, pf.categories.selected)
		assert.True(t, pf.pages[0].shown(), "the Graphics page is hidden")
		assert.False(t, pf.animated.checked, "the test app's settings turn the animations off")
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
		assert.Equal(t, "[Graphics]\nAnimatedSprites=1\n", string(ta.settings.(*memorySettings).data))
	})

	t.Run("when Cancel is clicked, should keep the old choice", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		ta.clickOn(pf.wdw, "AnimatedSprites")
		ta.clickOn(pf.wdw, "CancelButton")

		assert.Nil(t, ta.desk.topModal(), "the window is still up")
		assert.False(t, ta.preferences.animatedSprites)
		assert.Equal(t, "[Graphics]\nAnimatedSprites=0\n", string(ta.settings.(*memorySettings).data))
	})

	t.Run("when Apply is clicked, should keep the choice and stay open with nothing to apply", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		pf := openPreferences(ta)
		ta.clickOn(pf.wdw, "AnimatedSprites")
		ta.clickOn(pf.wdw, "ApplyButton")

		assert.Same(t, pf.wdw, ta.desk.topModal(), "the window closed")
		assert.True(t, ta.preferences.animatedSprites)
		assert.Equal(t, "[Graphics]\nAnimatedSprites=1\n", string(ta.settings.(*memorySettings).data))
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
		assert.Equal(t, "[Graphics]\nAnimatedSprites=1\n", string(ta.settings.(*memorySettings).data))
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
}
