// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIME stands in for Ebitengine's text input: a session starts at the
// first Update with the keyboard open, as the text area has the focus, and
// a test plays the IME's part through the keyboard's callbacks.
type fakeIME struct {
	kbd       *keyboard
	live      bool // a session runs
	sessions  int  // the sessions that started
	confirms  int
	digitsOn  bool   // what the keyboard last asked for
	handled   bool   // what the next Update reports
	next      func() // what the IME does in the next Update
	composing string // the word being composed, which Confirm types

	// The last session's caret and text around it.
	caret  image.Rectangle
	before string
	after  string
}

func (fi *fakeIME) Update() (bool, error) {
	if !fi.live {
		fi.caret, fi.before, fi.after, fi.live = fi.kbd.session()
		if fi.live {
			fi.sessions++
		}
	}

	if fi.next != nil {
		act := fi.next
		fi.next = nil

		act()
	}

	handled := fi.handled
	fi.handled = false

	return handled, nil
}

func (fi *fakeIME) Confirm() {
	if fi.live && fi.composing != "" {
		fi.kbd.commit(fi.composing, false, "", "")
		fi.kbd.compose("", 0)
	}

	fi.composing = ""
	fi.confirms++
	fi.live = false
}

// compose composes word, as the IME does while the player types it.
func (fi *fakeIME) compose(word string) {
	fi.composing = word
	fi.kbd.compose(word, len(word))
}

func (fi *fakeIME) digits(on bool) {
	fi.digitsOn = on
}

// withKeyboard gives ta a touch screen's keyboard on a fake IME.
func withKeyboard(ta *testApp) *fakeIME {
	fi := &fakeIME{}
	ta.keyboard = &keyboard{
		ime: fi,
	}
	fi.kbd = ta.keyboard

	return fi
}

// newSaveAs shows Save As, its file name focused, on a touch screen.
func newSaveAs(t *testing.T) (*testApp, *fakeIME) {
	t.Helper()

	ta := newTestApp(t)
	fi := withKeyboard(ta)
	ta.wait(80)
	ta.clickOn(ta.start.wdw, "NewButton")

	require.NotNil(t, ta.fileDialog, "no Save As")

	return ta, fi
}

// onText returns a point on a field's text, at its start: a press there
// puts the caret before the first character.
func onText(fld textField) image.Point {
	_, rect, _ := fld.textArea()

	return widgetOrigin(fld).Add(rect.Min).Add(image.Pt(1, rect.Dy()/2))
}

func Test_keyboard(t *testing.T) {
	t.Parallel()

	t.Run("when a finger presses the file name, should open the keyboard on it, with letters", func(t *testing.T) {
		t.Parallel()

		ta, fi := newSaveAs(t)
		name := ta.fileDialog.name
		ta.tap(onText(name))

		assert.Same(t, name, ta.keyboard.field, "the field the keyboard types in")
		assert.Equal(t, 1, fi.sessions, "the sessions")
		assert.False(t, fi.digitsOn, "digits")
	})

	t.Run("when the mouse presses the file name, should leave the keyboard closed", func(t *testing.T) {
		t.Parallel()

		ta, fi := newSaveAs(t)
		ta.click(onText(ta.fileDialog.name))

		assert.Nil(t, ta.keyboard.field, "the field the keyboard types in")
		assert.Zero(t, fi.sessions, "the sessions")
	})

	t.Run("when the IME commits a word with the Enter that ends it, should type the word before Enter saves", func(t *testing.T) {
		t.Parallel()

		ta, fi := newSaveAs(t)
		ta.tap(onText(ta.fileDialog.name))

		fi.next = func() {
			ta.keyboard.commit("web", false, "", "")
		}

		ta.key(ebiten.KeyEnter)

		assert.Nil(t, ta.fileDialog, "Save As still shows")
		assert.True(t, ta.name.wdw.visible, "the name dialog doesn't show")

		ta.key(ebiten.KeyEnter)
		ta.wait(1)

		assert.True(t, strings.HasSuffix(ta.game.SaveFile, "web.pgs"), "the save file %q", ta.game.SaveFile)
	})

	t.Run("when the IME deletes before the caret, should take the text out and keep what follows", func(t *testing.T) {
		t.Parallel()

		ta, _ := newSaveAs(t)
		name := ta.fileDialog.name
		ta.tap(onText(name))
		ta.keyboard.commit("wweb", false, "", "")

		name.tb.anchor, name.tb.caret = 2, 2

		ta.keyboard.commit("", true, "w", "eb")

		assert.Equal(t, "web", name.text(), "the file name")
		assert.Equal(t, 1, name.tb.caret, "the caret")
	})

	t.Run("when the IME composes a word, should show it underlined without typing it", func(t *testing.T) {
		t.Parallel()

		ta, _ := newSaveAs(t)
		name := ta.fileDialog.name
		ta.tap(onText(name))
		ta.keyboard.compose("we", 2)

		assert.Empty(t, name.text(), "the file name")

		img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
		ta.desk.paint(img)

		_, rect, _ := name.textArea()
		org := widgetOrigin(name).Add(rect.Min)
		line := org.Y + name.font.ascent() + 1

		for x := org.X; x < org.X+name.font.width("we"); x++ {
			assert.Equal(t, color.RGBA{A: 255}, img.RGBAAt(x, line), "the underline at x %d", x)
		}

		ta.keyboard.compose("", 0)

		assert.Empty(t, name.tb.composing, "the word after it ended")
	})

	t.Run("when the focus leaves the field, should type the word being composed and put the keyboard away", func(t *testing.T) {
		t.Parallel()

		ta, fi := newSaveAs(t)
		name := ta.fileDialog.name
		ta.tap(onText(name))
		fi.compose("web")

		view := ta.fileDialog.view
		ta.tap(widgetOrigin(view).Add(image.Pt(20, 20)))

		assert.Nil(t, ta.keyboard.field, "the field the keyboard types in")
		assert.Equal(t, "web", name.text(), "the file name")
		assert.Empty(t, name.tb.composing, "the word being composed")
	})

	t.Run("when a finger presses the field while a word is composed, should type the word where it was composed before the caret moves", func(t *testing.T) {
		t.Parallel()

		ta, fi := newSaveAs(t)
		name := ta.fileDialog.name
		ta.tap(onText(name))
		ta.keyboard.commit("ash", false, "", "")
		ta.wait(40) // past a double click's time
		fi.compose("ketchum")

		ta.tap(onText(name))

		assert.Equal(t, "ashketchum", name.text(), "the file name")
		assert.Zero(t, name.tb.caret, "the caret")
		assert.Empty(t, fi.before, "the new session's text before the caret")
		assert.Equal(t, "ashketchum", fi.after, "the new session's text after the caret")
	})

	t.Run("when the player puts the keyboard away, should stay closed until a finger presses a field", func(t *testing.T) {
		t.Parallel()

		ta, fi := newSaveAs(t)
		name := ta.fileDialog.name
		ta.tap(onText(name))
		ta.keyboard.close()
		ta.wait(40) // past a double click's time

		assert.Nil(t, ta.keyboard.field, "the field after the keyboard was put away")
		assert.Equal(t, 1, fi.sessions, "the sessions")

		ta.tap(onText(name))

		assert.Same(t, name, ta.keyboard.field, "the field after another press")
		assert.Equal(t, 2, fi.sessions, "the sessions")
	})

	t.Run("when the keyboard is open, should drop the characters the text area forwards", func(t *testing.T) {
		t.Parallel()

		ta, _ := newSaveAs(t)
		name := ta.fileDialog.name
		ta.tap(onText(name))
		ta.typeText("x")

		assert.Empty(t, name.text(), "the file name")
	})

	t.Run("when the IME took the frame's keys, should drop them", func(t *testing.T) {
		t.Parallel()

		ta, fi := newSaveAs(t)
		ta.tap(onText(ta.fileDialog.name))
		ta.keyboard.commit("web", false, "", "")

		fi.handled = true

		ta.key(ebiten.KeyEnter)

		assert.NotNil(t, ta.fileDialog, "Save As closed")
	})

	t.Run("when a session starts, should give the IME the text around the selection and the caret on the screen", func(t *testing.T) {
		t.Parallel()

		ta, fi := newSaveAs(t)
		name := ta.fileDialog.name
		ta.tap(onText(name))
		ta.keyboard.commit("pokemon", false, "", "")

		name.tb.anchor, name.tb.caret = 3, 5

		ta.wait(1)

		_, rect, _ := name.textArea()
		org := widgetOrigin(name).Add(rect.Min)
		x := org.X + name.font.width("pok")

		assert.Equal(t, "pok", fi.before, "the text before the caret")
		assert.Equal(t, "on", fi.after, "the text after the caret")
		assert.Equal(t, image.Rect(x, org.Y, x+1, org.Y+name.font.height()), fi.caret, "the caret")
	})

	t.Run("when a finger presses a spin edit, should open on its text, with digits, and not on its buttons", func(t *testing.T) {
		t.Parallel()

		ta := newGameCorner(t)
		fi := withKeyboard(ta)
		bet := get[*spinEdit](ta.gameCorner.wdw, "SpinEdit1")
		up, _ := bet.buttons()
		ta.tap(widgetOrigin(bet).Add(up.Min).Add(up.Size().Div(2)))

		assert.Nil(t, ta.keyboard.field, "the field after a press on the up button")

		ta.tap(onText(bet))

		assert.Same(t, bet, ta.keyboard.field, "the field after a press on the text")
		assert.True(t, fi.digitsOn, "digits")

		bet.tb.set("")
		ta.keyboard.commit("12a3", false, "", "")

		assert.Equal(t, "123", bet.tb.String(), "the bet's text")
	})

	t.Run("when the IME commits part of a species in the POKéDEX, should complete it as typing does", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		withKeyboard(ta)

		cb := ta.pokedex.combo
		ta.tap(onText(cb))

		require.Same(t, cb, ta.keyboard.field, "the field the keyboard types in")

		cb.tb.set("")
		ta.keyboard.commit("Char", false, "", "")

		assert.Equal(t, "Charmander (#004)", cb.tb.String(), "the species' text")
		assert.Equal(t, 3, cb.index, "the species chosen")
	})
}

func Test_textBox_splice(t *testing.T) {
	t.Parallel()

	t.Run("when the IME replaces the text around the caret, should keep before, the text and after, with the caret after the text", func(t *testing.T) {
		t.Parallel()

		changes := 0
		tbx := &textBox{
			onChange: func() {
				changes++
			},
		}
		tbx.set("Pikac is")
		tbx.splice("", "Pikachu", " is")

		assert.Equal(t, "Pikachu is", tbx.String(), "the text")
		assert.Equal(t, 7, tbx.caret, "the caret")
		assert.Equal(t, 7, tbx.anchor, "the anchor")
		assert.Equal(t, 1, changes, "the changes told")
	})

	t.Run("when the text has characters the box doesn't take, should drop them", func(t *testing.T) {
		t.Parallel()

		tbx := &textBox{
			valid: isDigit,
		}
		tbx.splice("1", "2x3日", "")

		assert.Equal(t, "123", tbx.String(), "the text")
	})

	t.Run("when cp1252 can't hold a character, should drop it", func(t *testing.T) {
		t.Parallel()

		tbx := &textBox{}
		tbx.splice("", "Poké日", "")

		assert.Equal(t, "Poké", tbx.String(), "the text")
	})

	t.Run("when the box has a length limit, should cut the text to fit", func(t *testing.T) {
		t.Parallel()

		tbx := &textBox{
			maxLen: 5,
		}
		tbx.splice("ab", "cdefg", "z")

		assert.Equal(t, "abcdz", tbx.String(), "the text")
		assert.Equal(t, 4, tbx.caret, "the caret")
	})

	t.Run("when the box is read-only, should change nothing", func(t *testing.T) {
		t.Parallel()

		tbx := &textBox{
			readOnly: true,
		}
		tbx.set("Ash")
		tbx.splice("", "Gary", "")

		assert.Equal(t, "Ash", tbx.String(), "the text")
	})
}
