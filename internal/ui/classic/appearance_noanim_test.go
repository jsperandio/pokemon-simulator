// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js && noanim

package classic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_appearance(t *testing.T) {
	t.Parallel()

	t.Run("when built without the animations, should show the still sprite only", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.preferences.animatedSprites = true
		ta.startGame()
		ta.call(func() {
			ta.game.StartWildBattle(25, 50, false)
		})
		ta.wait(1)
		ta.answer("Select a POKéMON")
		ta.wait(1)

		assert.False(t, ta.main.enemyAppear.playing())
		assert.Equal(t, []listItem{{"", 24}}, ta.main.enemyImage.items)
	})
}
