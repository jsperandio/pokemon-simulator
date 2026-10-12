// SPDX-License-Identifier: AGPL-3.0-only

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

// menuEffects is what a menu command can change that the player sees.
type menuEffects struct {
	modal   string // the caption of the modal window up
	windows string // the windows shown
	battle  [3]bool
	checked string // the items checked
	quit    bool
}

func (ta *testApp) menuEffects() menuEffects {
	e := menuEffects{quit: ta.quit}
	if w := ta.desk.topModal(); w != nil {
		e.modal = w.caption
	}
	for _, w := range ta.desk.windows {
		if w.visible {
			e.windows += w.name + " "
		}
	}
	g := ta.g
	e.battle = [3]bool{g.InWildBattle, g.InTrainerBattle, g.InTournament}
	var walk func(items []*game.MenuItem)
	walk = func(items []*game.MenuItem) {
		for _, m := range items {
			if m.Checked {
				e.checked += m.Name + " "
			}
			walk(m.Items)
		}
	}
	walk(g.Menu)
	return e
}

// TestEveryMenuCommand clicks each command of the game window's menu, in a
// game where every trainer is unlocked, and checks that it does something
// the player sees: a battle, a window, a dialog, a check mark, or the
// end. The Debug menu stays hidden, as in the original.
func TestEveryMenuCommand(t *testing.T) {
	f := save.File{PlayerName: "Ash", GymProgress: 24, BeatLance: true, BeatAsh: true, BeatLink: true, Money: 50000, PokeBalls: 3}
	f.Owned[0] = save.Slot{Species: 6, Level: 100, XP: 1000000}   // Charizard
	f.Owned[1] = save.Slot{Species: 150, Level: 100, XP: 1000000} // Mewtwo
	f.InDex[6], f.InDex[150] = true, true
	b := f.Bytes()

	var commands [][]string
	var walk func(items []*game.MenuItem, path []string)
	walk = func(items []*game.MenuItem, path []string) {
		for _, m := range items {
			if !m.Visible {
				continue
			}
			p := append(slices.Clone(path), m.Name)
			if len(m.Items) > 0 {
				walk(m.Items, p)
				continue
			}
			commands = append(commands, p)
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
			if err := os.WriteFile(file, b, 0o644); err != nil {
				t.Fatal(err)
			}
			ta := newTestAppIn(t, dir, []string{file})
			ta.music = &fakeMusic{stopErr: audio.ErrNotOpen}
			ta.wait(80)
			if !ta.main.w.visible {
				t.Fatal("the game didn't load")
			}
			if ta.main.w.menu.entries[len(ta.main.w.menu.entries)-1].item.Visible {
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
