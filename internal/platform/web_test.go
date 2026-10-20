// SPDX-License-Identifier: AGPL-3.0-only

//go:build js

package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_memory_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("when the folder is missing, should fail with ErrNotExist", func(t *testing.T) {
		t.Parallel()

		mem := newMemory()

		assert.ErrorIs(t, mem.WriteFile("/saves/ash.pgs", []byte("x")), ErrNotExist)
	})

	t.Run("when the folder was made, should keep the file", func(t *testing.T) {
		t.Parallel()

		mem := newMemory()
		require.NoError(t, mem.Mkdir("/saves"))

		require.NoError(t, mem.WriteFile("/saves/ash.pgs", []byte("save")))

		dat, err := mem.ReadFile("/saves/ash.pgs")
		require.NoError(t, err)
		assert.Equal(t, "save", string(dat))
	})
}

func Test_memory_ReadFile(t *testing.T) {
	t.Parallel()

	t.Run("when the path goes up and back down, should read the file it leads to", func(t *testing.T) {
		t.Parallel()

		mem := newSaves(t)

		dat, err := mem.ReadFile("/saves/../saves/ash.pgs")
		require.NoError(t, err)
		assert.Equal(t, "save", string(dat))
	})
}

func Test_memory_Stat(t *testing.T) {
	t.Parallel()

	t.Run("when the path is a folder, should say so", func(t *testing.T) {
		t.Parallel()

		mem := newSaves(t)

		ent, err := mem.Stat("/saves")
		require.NoError(t, err)
		assert.True(t, ent.Dir)
	})
}

func Test_memory_List(t *testing.T) {
	t.Parallel()

	t.Run("when the folder holds a folder and a file, should list both", func(t *testing.T) {
		t.Parallel()

		mem := newSaves(t)

		entries, err := mem.List("/")
		require.NoError(t, err)

		names := make([]string, 0, len(entries))
		for _, ent := range entries {
			names = append(names, ent.Name)
		}

		assert.ElementsMatch(t, []string{"saves", "song.ogg"}, names)
	})
}

// The database's records come back with the folders they are in.
func Test_memory_restore(t *testing.T) {
	t.Parallel()

	t.Run("when a record is in folders, should bring back the folders too", func(t *testing.T) {
		t.Parallel()

		mem := newMemory()
		mem.restore("/a/b/c.pgs", false, []byte("c"))
		mem.restore("/empty", true, nil)

		for _, pth := range []string{"/a", "/a/b", "/empty"} {
			ent, err := mem.Stat(pth)
			assert.NoError(t, err, "stat %s", pth)
			assert.True(t, ent.Dir, "%s isn't a folder", pth)
		}

		ent, err := mem.Stat("/a/b/c.pgs")
		require.NoError(t, err)
		assert.False(t, ent.Dir)
	})

	t.Run("when a record's path is relative, should skip it", func(t *testing.T) {
		t.Parallel()

		mem := newMemory()
		mem.restore("relative", false, []byte("?"))

		assert.Empty(t, mem.files)
	})
}

// newSaves returns a memory with ash.pgs in the folder /saves and song.ogg
// at the root.
func newSaves(t *testing.T) *memory {
	t.Helper()

	mem := newMemory()
	require.NoError(t, mem.Mkdir("/saves"))
	require.NoError(t, mem.WriteFile("/saves/ash.pgs", []byte("save")))
	require.NoError(t, mem.WriteFile("/song.ogg", []byte("OggS")))

	return mem
}
