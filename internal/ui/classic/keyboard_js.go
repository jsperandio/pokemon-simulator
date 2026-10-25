// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"

	"github.com/jsperandio/pokemon-simulator/internal/platform"
)

// webIME is the browser's IME through Ebitengine's text input. Its hidden
// text area takes the browser's focus when the finger that opened the
// keyboard lifts, which is what brings a phone's keyboard up.
type webIME struct {
	*textinput.Composer
}

func (webIME) digits(on bool) {
	platform.KeyboardDigits(on)
}

// newKeyboard returns the keyboard a touch screen shows, closed, or nil
// where the main pointer isn't a finger: there the player has a keyboard
// of their own.
func newKeyboard() *keyboard {
	if !platform.TouchScreen() {
		return nil
	}

	kbd := &keyboard{}
	kbd.ime = webIME{
		Composer: &textinput.Composer{
			OnNewSession: func() *textinput.SessionOptions {
				caret, before, after, ok := kbd.session()
				if !ok {
					return nil
				}

				return &textinput.SessionOptions{
					CaretBounds:     caret,
					TextBeforeCaret: before,
					TextAfterCaret:  after,
				}
			},
			OnComposition: func(cmp *textinput.Composition) {
				beg, _ := cmp.SelectionRangeInBytes()
				kbd.compose(cmp.Text(), beg)
			},
			OnCommit: func(cmt *textinput.Commit) {
				headGone, tailGone := cmt.IsSurroundingTextReplaced()
				head, tail := cmt.SurroundingText()
				kbd.commit(cmt.Text(), headGone || tailGone, head, tail)
			},
			OnEndByUser: kbd.close,
		},
	}

	return kbd
}
