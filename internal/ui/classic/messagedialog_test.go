// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/game"
)

// promptText is a message as long as the battle's choose prompt.
const promptText string = "Select a POKéMON from the Statistics Window. You'll be asked to confirm your choice."

// showMessage shows a message dialog of the user interface's own, which
// keeps its result in result once it closes.
func showMessage(ta *testApp, check string, result *int) *window {
	ta.t.Helper()

	dsk := ta.desk
	wdw := messageDialog(dsk, game.Information, false, promptText, check)
	wdw.onModal = func(rst int) {
		*result = rst
	}
	dsk.showModal(wdw)
	ta.wait(1)

	return wdw
}

func Test_messageDialog(t *testing.T) {
	t.Parallel()

	t.Run("when it has no check box, should keep the original's layout", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)

		var rst int

		wdw := showMessage(ta, "", &rst)

		assert.Nil(t, wdw.find("DontShowAgain"))
		assert.Equal(t, image.Pt(444, 97), wdw.client)
	})

	t.Run("when it has a check box, should put it below the text, lined up with it, and keep the focus on OK", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)

		var rst int

		wdw := showMessage(ta, dontShowAgain, &rst)
		ta.shot("message-check")

		chb := get[*checkBox](wdw, "DontShowAgain")
		msg := wdw.find("Message").base()
		btn := get[*button](wdw, "Button1")

		assert.False(t, chb.checked)
		assert.Equal(t, msg.rect.Min.X, chb.rect.Min.X, "not lined up with the text")
		assert.GreaterOrEqual(t, chb.rect.Min.Y, msg.rect.Max.Y, "over the text")
		assert.GreaterOrEqual(t, btn.rect.Min.Y, chb.rect.Max.Y, "under the button")
		assert.True(t, chb.rect.In(image.Rectangle{Max: wdw.client}), "%v is outside the box", chb.rect)
		assert.Same(t, btn, wdw.focus, "OK hasn't the focus")
	})

	t.Run("when its accelerator is typed and Enter pressed, should close with OK and the box checked", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)

		var rst int

		wdw := showMessage(ta, dontShowAgain, &rst)
		ta.typeText("d")
		ta.key(ebiten.KeyEnter)

		require.Nil(t, ta.desk.topModal(), "the box is still up")
		assert.Equal(t, mrOk, rst)
		assert.True(t, get[*checkBox](wdw, "DontShowAgain").checked)
	})
}
