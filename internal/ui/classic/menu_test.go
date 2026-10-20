// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/audio"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// Test_EveryMenuCommand clicks each command of the game window's menu, in a
// game where every trainer is unlocked, and checks that it does something
// the player sees: a battle, a window, a dialog, a check mark, or the
// end. The Debug menu stays hidden, as in the original.
func Test_EveryMenuCommand(t *testing.T) {
	t.Parallel()

	sav := save.File{
		PlayerName:  "Ash",
		GymProgress: 24,
		BeatLance:   true,
		BeatAsh:     true,
		BeatLink:    true,
		Money:       50000,
		PokeBalls:   3,
	}
	// Charizard.
	sav.Owned[0] = save.Slot{
		Species: 6,
		Level:   100,
		XP:      1000000,
	}
	// Mewtwo.
	sav.Owned[1] = save.Slot{
		Species: 150,
		Level:   100,
		XP:      1000000,
	}
	sav.InPokedex[6], sav.InPokedex[150] = true, true
	dat := sav.Bytes()

	var (
		commands [][]string
		walk     func(items []*game.MenuItem, pth []string)
	)
	walk = func(items []*game.MenuItem, pth []string) {
		for _, mi := range items {
			if !mi.Visible {
				continue
			}

			cmd := append(slices.Clone(pth), mi.Name)
			if len(mi.Items) > 0 {
				walk(mi.Items, cmd)

				continue
			}

			commands = append(commands, cmd)
		}
	}
	walk(game.New(nil, nil).Menu, nil)

	t.Run("when the menu is walked, should hold the 66 commands outside the Debug menu", func(t *testing.T) {
		t.Parallel()

		// 45 trainers, 2 wild battles, 6 tournaments, 6 other actions, and
		// File's 2, Music's 3 and Help's 2.
		assert.Len(t, commands, 66, "the commands outside the Debug menu")
	})

	t.Run("when the game is loaded, should keep the Debug menu hidden", func(t *testing.T) {
		t.Parallel()

		ta := newAllApp(t, dat)
		ta.wait(80)

		require.True(t, ta.main.wdw.visible, "the game didn't load")

		bar := ta.main.wdw.menu
		require.NotEmpty(t, bar.entries, "the menu bar")
		assert.False(t, bar.entries[len(bar.entries)-1].item.Visible, "the Debug menu shows")
	})

	for _, cmd := range commands {
		t.Run(cmd[len(cmd)-1], func(t *testing.T) {
			t.Parallel()

			ta := newAllApp(t, dat)
			ta.music = &fakeMusic{
				stopErr: audio.ErrNotOpen,
			}
			ta.wait(80)

			require.True(t, ta.main.wdw.visible, "the game didn't load")

			before := ta.menuEffects()
			ta.menu(cmd...)
			ta.wait(2)

			after := ta.menuEffects()
			assert.NotEqual(t, before, after, "%s did nothing", strings.Join(cmd, " > "))
		})
	}
}

// newAllApp starts the program with the save dat in a folder of its own, the
// game the menu test plays.
func newAllApp(t *testing.T, dat []byte) *testApp {
	t.Helper()

	dir := t.TempDir()
	file := filepath.Join(dir, "all.pgs")

	require.NoError(t, os.WriteFile(file, dat, 0o600))

	return newTestAppIn(t, dir, []string{file})
}
