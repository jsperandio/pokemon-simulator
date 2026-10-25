// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"unicode/utf8"
)

// textField is a control the player types text in: an edit, a spin edit,
// or a combo box with an edit.
type textField interface {
	widget
	keyer
	// textArea returns the field's text box and where its text shows, in
	// the control; ok is false when the field takes no typing.
	textArea() (tbx *textBox, rect image.Rectangle, ok bool)
	// splice sets the text to before, txt and after, with the caret after
	// txt, as the keyboard types over the text around the caret.
	splice(before, txt, after string)
}

// ime is the keyboard's IME: Ebitengine's text input on the web, whose
// sessions type in the field, and the kind of keys the keyboard shows.
type ime interface {
	// Update runs the session for the frame, starting one when there is
	// none; it reports whether the IME took the frame's keys.
	Update() (handled bool, err error)
	// Confirm ends the session, typing the word being composed.
	Confirm()
	// digits asks the keyboard for digits, or for letters again.
	digits(on bool)
}

// keyboard is the keyboard a touch screen shows, typing in the focused text
// field through the IME. It opens when a finger presses a field's text,
// and closes when the field loses the focus or the player puts it away.
// What the player types comes as the IME's commits, and the word being
// typed as its composition.
type keyboard struct {
	ime    ime
	field  textField // where the keyboard types; nil while it is closed
	before string    // the field's text before its selection, as the session began
	after  string    // ... and after it
}

// isOpen reports whether the keyboard is up, typing in a field.
func (kbd *keyboard) isOpen() bool {
	return kbd.field != nil
}

// commits types the IME's commits in the field before the desktop gets the
// frame's input, so that a word lands before the Enter that came with it.
// The keyboard's text comes only as commits: the characters the text area
// also forwards are dropped, and so are the keys of a frame the IME took.
// A press or a key of the player's own confirms the word being typed
// first, as it may move the caret or change the field.
func (kbd *keyboard) commits(in *input) {
	if kbd.field == nil {
		return
	}

	in.chars = nil

	if kbd.update() {
		in.keys = nil

		return
	}

	if in.pressed || len(in.keys) > 0 {
		kbd.ime.Confirm()
	}
}

// follow opens the keyboard when a finger presses the text of the focused
// field, and closes it when the field loses the focus, once the desktop
// had the frame's input. While it is open, it keeps the IME's session in
// step with the field: a field that changed under the session ends it,
// and the next one starts from the field's text.
func (kbd *keyboard) follow(dsk *desktop, in *input) {
	fld := focusedField(dsk)
	if kbd.field != nil && fld != kbd.field {
		kbd.close()
	}

	fingerOnText := in.pressed && in.touch && fld != nil && pressedText(fld)
	if kbd.field == nil && fingerOnText {
		kbd.open(fld)
	}

	if kbd.field == nil {
		return
	}

	if before, after := aroundSelection(kbd.field); before != kbd.before || after != kbd.after {
		kbd.ime.Confirm()
	}

	kbd.update()
}

// open opens the keyboard on field fld, with digits for a spin edit.
func (kbd *keyboard) open(fld textField) {
	_, digits := fld.(*spinEdit)
	kbd.ime.digits(digits)

	kbd.field = fld
}

// close puts the keyboard away, typing the word being composed first.
func (kbd *keyboard) close() {
	kbd.ime.Confirm()

	kbd.field = nil
}

// update runs the IME's session for the frame, and reports whether the IME
// took the frame's keys. A session that fails closes the keyboard.
func (kbd *keyboard) update() bool {
	handled, err := kbd.ime.Update()
	if err != nil {
		kbd.field = nil

		return false
	}

	return handled
}

// session returns what the IME needs as a session begins: where the caret
// shows on the screen, and the field's text before and after its
// selection, which typing replaces. ok is false while the keyboard is
// closed.
func (kbd *keyboard) session() (caret image.Rectangle, before, after string, ok bool) {
	if kbd.field == nil {
		return image.Rectangle{}, "", "", false
	}

	tbx, rect, _ := kbd.field.textArea()
	kbd.before, kbd.after = aroundSelection(kbd.field)

	fnt := kbd.field.base().font
	org := widgetOrigin(kbd.field).Add(rect.Min)
	x := org.X + fnt.width(kbd.before) - tbx.scroll

	return image.Rect(x, org.Y, x+1, org.Y+fnt.height()), kbd.before, kbd.after, true
}

// compose shows txt, the word the IME is composing, in the field, with the
// caret caret bytes into it; an empty txt ends the word.
func (kbd *keyboard) compose(txt string, caret int) {
	if kbd.field == nil {
		return
	}

	tbx, _, _ := kbd.field.textArea()
	tbx.compose(txt, utf8.RuneCountInString(txt[:max(min(caret, len(txt)), 0)]))
	kbd.field.base().invalidate()
}

// commit types txt, the text the IME committed, in the field. It goes in
// as typed characters, so that a combo box completes them and a spin edit
// keeps its digits; when the IME replaced text around the caret, as a
// keyboard's backspace does, the field becomes before, txt and after.
func (kbd *keyboard) commit(txt string, replaced bool, before, after string) {
	if kbd.field == nil {
		return
	}

	if replaced {
		kbd.field.splice(before, txt, after)

		return
	}

	for _, chr := range txt {
		kbd.field.char(chr)
	}
}

// focusedField returns the active window's focused control when it is a
// text field that takes typing, or nil.
func focusedField(dsk *desktop) textField {
	if dsk.active == nil {
		return nil
	}

	fld, ok := dsk.active.focus.(textField)
	if !ok {
		return nil
	}

	if _, _, typing := fld.textArea(); !typing {
		return nil
	}

	return fld
}

// pressedText reports whether the mouse went down on a field's text, not
// on its buttons, and is still down.
func pressedText(fld textField) bool {
	tbx, _, _ := fld.textArea()

	return tbx.dragging
}

// aroundSelection returns a field's text before and after its selection.
func aroundSelection(fld textField) (before, after string) {
	tbx, _, _ := fld.textArea()
	beg, end := tbx.sel()

	return string(tbx.text[:beg]), string(tbx.text[end:])
}
