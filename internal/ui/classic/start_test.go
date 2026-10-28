// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_saveName(t *testing.T) {
	t.Parallel()

	t.Run("when the player is named, should suggest the name with the save's extension", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "Gary.pgs", saveName("Gary", true))
	})

	t.Run("when the name is empty, should suggest Ash's, as the game names the player", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "Ash.pgs", saveName("", true))
	})

	t.Run("when the name has characters a file name can't have, should leave them out", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "Mr. Mime.pgs", saveName(` Mr. <Mime>/\:*?"|`, true))
		assert.Equal(t, "Red.pgs", saveName("..Red..", true))
	})

	t.Run("when nothing is left of the name, should suggest none", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, saveName("?/. *", true))
	})

	t.Run("when the name dialog closed with Alt+F4, should suggest none", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, saveName("Gary", false))
	})
}
