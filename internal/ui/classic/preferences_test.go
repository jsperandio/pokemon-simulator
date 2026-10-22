// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
