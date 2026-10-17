// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jsperandio/pokemon-simulator/internal/audio"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// TestEveryMenuCommand clicks each command of the game window's menu, in a
// game where every trainer is unlocked, and checks that it does something
// the player sees: a battle, a window, a dialog, a check mark, or the
// end. The Debug menu stays hidden, as in the original.
func TestEveryMenuCommand(t *testing.T) {
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
		walk     func(items []*game.MenuItem, path []string)
	)
	walk = func(items []*game.MenuItem, path []string) {
		for _, mi := range items {
			if !mi.Visible {
				continue
			}

			pth := append(slices.Clone(path), mi.Name)
			if len(mi.Items) > 0 {
				walk(mi.Items, pth)

				continue
			}

			commands = append(commands, pth)
		}
	}
	walk(game.New(nil, nil).Menu, nil)

	// 45 trainers, 2 wild battles, 6 tournaments, 6 other actions, and
	// File's 2, Music's 3 and Help's 2.
	if len(commands) != 66 {
		t.Errorf("%d commands, want the 66 outside the Debug menu", len(commands))
	}

	for _, path := range commands {
		t.Run(path[len(path)-1], func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "all.pgs")

			if err := os.WriteFile(file, dat, 0o600); err != nil {
				t.Fatal(err)
			}

			ta := newTestAppIn(t, dir, []string{file})
			ta.music = &fakeMusic{stopErr: audio.ErrNotOpen}
			ta.wait(80)

			if !ta.main.wdw.visible {
				t.Fatal("the game didn't load")
			}

			bar := ta.main.wdw.menu
			if bar.entries[len(bar.entries)-1].item.Visible {
				t.Error("the Debug menu shows")
			}

			before := ta.menuEffects()
			ta.menu(path...)
			ta.wait(2)

			if after := ta.menuEffects(); after == before {
				t.Errorf("%s did nothing", strings.Join(path, " > "))
			}
		})
	}
}
