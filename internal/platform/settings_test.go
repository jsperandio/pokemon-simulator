// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package platform

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Settings(t *testing.T) {
	t.Parallel()

	t.Run("when nothing was written, should fail with ErrNotExist", func(t *testing.T) {
		t.Parallel()

		set := NewSettings(filepath.Join(t.TempDir(), "settings.ini"))

		_, err := set.Read()
		assert.ErrorIs(t, err, ErrNotExist)
	})

	t.Run("when written, should make the folder and read the bytes back", func(t *testing.T) {
		t.Parallel()

		set := NewSettings(filepath.Join(t.TempDir(), "pokemon-simulator", "settings.ini"))
		require.NoError(t, set.Write([]byte("[Graphics]\nAnimatedSprites=0\n")))

		dat, err := set.Read()
		require.NoError(t, err)
		assert.Equal(t, "[Graphics]\nAnimatedSprites=0\n", string(dat))
	})

	t.Run("when the system has no settings folder, should read nothing and fail to write", func(t *testing.T) {
		t.Parallel()

		set := Settings{}

		_, err := set.Read()
		assert.ErrorIs(t, err, ErrNotExist)
		assert.Error(t, set.Write([]byte("x")))
	})

	t.Run("when the user's settings are asked for, should name the game's file", func(t *testing.T) {
		t.Parallel()

		set := UserSettings()
		if set.path == "" {
			t.Skip("no folder for programs' settings here")
		}

		assert.Equal(t, filepath.Join("pokemon-simulator", "settings.ini"), filepath.Join(filepath.Base(filepath.Dir(set.path)), filepath.Base(set.path)))
	})
}
