// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"fmt"
	"maps"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

func loadData(t *testing.T) *gamedata.Data {
	t.Helper()

	raw, err := assets.FS.ReadFile("pokegame.dat")
	require.NoError(t, err)

	d, err := gamedata.Load(raw)
	require.NoError(t, err)

	return d
}

// silent answers every dialog with OK or Yes.
type silent struct {
	messages []string
}

func (s *silent) Message(kind DialogKind, yesNo bool, text string) bool {
	s.messages = append(s.messages, text)

	return true
}

func newGame(t *testing.T) (*Game, *silent) {
	t.Helper()

	ui := &silent{}

	return New(loadData(t), ui), ui
}

// The win percentage is computed in x87 Extended; the Statistics window
// showed 77.99 for 39 wins and 11 losses, and 33.33 for 1 and 2.
func Test_Game_WinPercent(t *testing.T) {
	t.Parallel()

	t.Run("when the percentage has decimals, should truncate it to two", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)

		for _, cas := range []struct {
			wins, losses uint32
			want         string
		}{
			{39, 11, "77.99"},
			{1, 2, "33.33"},
			{2, 1, "66.66"},
		} {
			gam.Wins, gam.Losses = cas.wins, cas.losses

			percent, ok := gam.WinPercent()
			assert.True(t, ok, "%d wins, %d losses", cas.wins, cas.losses)
			assert.Equal(t, cas.want, percent, "%d wins, %d losses", cas.wins, cas.losses)
		}
	})

	t.Run("when every fight is won, should give 100", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.Wins, gam.Losses = 1, 0

		percent, ok := gam.WinPercent()
		assert.True(t, ok)
		assert.Equal(t, "100", percent)
	})

	t.Run("when no fight is won, should give 0", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.Wins, gam.Losses = 0, 3

		percent, ok := gam.WinPercent()
		assert.True(t, ok)
		assert.Equal(t, "0", percent)
	})

	t.Run("when there are no fights, should give no percentage", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)

		_, ok := gam.WinPercent()
		assert.False(t, ok)
	})
}

// Type Comparisons printed "0.5" and "0.25" for Razor Leaf on Charizard.
func Test_Game_TypeMultipliers(t *testing.T) {
	t.Parallel()

	t.Run("when nothing is selected, should show no multipliers", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.OpenTypes()

		_, _, _, ok := gam.TypeMultipliers()
		assert.False(t, ok)
	})

	t.Run("when Razor Leaf is compared with Charizard, should give 0.5 for each type and 0.25 in all", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.OpenTypes()
		gam.SelectTypes(5, 3) // Charizard (Fire/Flying), Razor Leaf (Grass)

		first, second, total, ok := gam.TypeMultipliers()
		assert.True(t, ok)
		assert.Equal(t, "0.5", first)
		assert.Equal(t, "0.5", second)
		assert.Equal(t, "0.25", total)
	})
}

// The POKéDEX count in the Statistics window counts species 1–251 only.
func Test_Game_PokedexCount(t *testing.T) {
	t.Parallel()

	t.Run("when registered species go past 251, should count only 1 to 251", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		for _, s := range []int{1, 150, 251, 252, 386} {
			gam.InPokedex[s] = true
		}

		assert.Equal(t, 3, gam.PokedexCount())
	})
}

// The POKéDEX shows what TForm8.UpdateDisplay showed, which runs only when
// a species is selected or added.
func Test_Game_SelectPokedex(t *testing.T) {
	t.Parallel()

	t.Run("when a new species is added, should show its attacks", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(0) // Bulbasaur

		assert.Equal(t, 0, gam.Pokedex.Index)
		assert.True(t, gam.Pokedex.MovesShown)
	})

	t.Run("when an evolution registers the species shown, should keep its attacks hidden until it is selected again", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		sav := save.File{
			PlayerName: "Probe",
			PokeBalls:  3,
		}
		sav.Owned[0] = save.Slot{
			Species: 1,
			Level:   16,
			XP:      4096,
		}
		sav.InPokedex[1] = true
		require.NoError(t, gam.Load(`C:\Temp\probe-dex.pgs`, sav.Bytes()))

		gam.OpenPokedex()
		gam.SelectPokedex(1) // Ivysaur
		require.False(t, gam.Pokedex.MovesShown)

		gam.ClickMenu("Evolve1")
		gam.ClickParty(0) // Bulbasaur evolves into Ivysaur
		require.True(t, gam.InPokedex[2])
		assert.False(t, gam.Pokedex.MovesShown)

		gam.SelectPokedex(0)
		gam.SelectPokedex(1)
		assert.True(t, gam.Pokedex.MovesShown)
	})

	t.Run("when species are selected, should count them, but not the POKéDEX opening", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(0) // the POKéDEX shows Bulbasaur
		require.Equal(t, 1, gam.Pokedex.Selected)

		gam.OpenPokedex()
		assert.Equal(t, 1, gam.Pokedex.Selected)

		gam.SelectPokedex(24)
		assert.Equal(t, 2, gam.Pokedex.Selected)

		gam.SelectPokedex(-1)
		gam.SelectPokedex(386)
		assert.Equal(t, 2, gam.Pokedex.Selected)
	})
}

// The PokéMart describes the item chosen in two lines, and nothing until one
// is chosen, as TForm9's FormShow and RadioGroup1Click do.
func Test_Game_PokemartDescription(t *testing.T) {
	t.Parallel()

	t.Run("when the Mart opens, should describe nothing", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.OpenPokemart()

		line1, line2 := gam.PokemartDescription()
		assert.Empty(t, line1)
		assert.Empty(t, line2)
	})

	t.Run("when an item is chosen, should describe it in the original's two lines", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.OpenPokemart()

		for idx, want := range [][2]string{
			{"A standard POKéBALL.", "Used for catching POKéMON."},
			{"This Ball has good chances", "of capturing a POKéMON."},
			{"Has a very high chance of", "capturing a POKéMON."},
			{"Restores 50HP in battle.", ""},
			{"Restores 100HP in battle.", ""},
		} {
			gam.SelectPokemartItem(idx)

			line1, line2 := gam.PokemartDescription()
			assert.Equal(t, want, [2]string{line1, line2}, "item %d", idx)
		}
	})
}

func Test_Damage(t *testing.T) {
	t.Parallel()

	for _, cas := range []struct {
		name               string
		roll, power, level int
		crit               bool
		first, second      float64
		want               int
	}{
		// The original's trace wild-tackle-1 (testdata/scenarios).
		{"when Bulbasaur's Tackle on Weedle does 9.5, should deal 9", 16, 70, 5, false, 1, 1, 9},
		// -9.6 truncates to -9; the minimum of 1 then replaces it.
		{"when the damage is -9.6, should truncate to -9, then deal 1", 0, 20, 2, false, 1, 1, 1},
		// 109 × 2 × 4: a critical hit doubles it; two super effective types quadruple it.
		{"when both types are super effective, should deal 872", 19, 100, 100, true, 2, 2, 872},
		{"when one type halves it, should truncate 2.5 to 2", 10, 50, 10, false, 0.5, 1, 2},
		// Immunity zeroes it later, after the minimum.
		{"when the attack is immune, should give the minimum 1 first", 10, 50, 10, false, 0, 1, 1},
		// 129 × 2 × 0.5: one type doubles it, the other halves it.
		{"when the types double and halve it, should deal 129", 19, 120, 100, false, 2, 0.5, 129},
	} {
		t.Run(cas.name, func(t *testing.T) {
			t.Parallel()

			got := Damage(cas.roll, cas.power, cas.level, cas.crit, cas.first, cas.second)
			assert.Equal(t, cas.want, got)
		})
	}
}

// The prizes' effects are hard-coded; their names come from
// POKéGAME.dat. Each effect must be what its name says.
func Test_Game_handleTournament(t *testing.T) {
	t.Parallel()

	for tier := 1; tier <= 6; tier++ {
		for roll := 1; roll <= 3; roll++ {
			t.Run(fmt.Sprintf("when tier %d rolls %d, should give what its prize says", tier, roll), func(t *testing.T) {
				t.Parallel()

				gam, ui := newGame(t)
				gam.Owned[0].Species, gam.Owned[0].Level, gam.Owned[0].XP = 25, 50, 125000
				// The tier's eighth round, won.
				gam.BattleType = uint8(tier*10 + 8) //nolint:gosec // tiers 1–6 make 18–68
				before := gam.File

				seedForRoll(gam, roll)
				gam.handleTournament()

				prize := gam.prize(tier, roll)
				assert.Contains(t, ui.messages, "You won "+prize+"!")

				deltas := map[string]int{
					"¥":             int(gam.Money) - int(before.Money),
					" POKéBALL":     int(gam.PokeBalls) - int(before.PokeBalls),
					" Great Ball":   int(gam.GreatBalls) - int(before.GreatBalls),
					" Ultra Ball":   int(gam.UltraBalls) - int(before.UltraBalls),
					" Master Ball":  int(gam.MasterBalls) - int(before.MasterBalls),
					" Potion":       int(gam.Potions) - int(before.Potions),
					" Super Potion": int(gam.SuperPotions) - int(before.SuperPotions),
					" POKéMON":      len(gam.Party()) - len(before.Party()),
				}
				maps.DeleteFunc(deltas, func(_ string, change int) bool {
					return change == 0
				})

				assert.Equal(t, wantChanges(t, tier, roll, prize), deltas, "what %q changes", prize)
			})
		}
	}
}

// seedForRoll sets gam's seed to one whose first Random(3) is roll.
func seedForRoll(gam *Game, roll int) {
	for gam.Rand.Seed = 0; ; gam.Rand.Seed++ {
		seed := gam.Rand.Seed
		if gam.Rand.Random(3)+1 == roll {
			gam.Rand.Seed = seed

			return
		}
	}
}

// wantChanges is what a prize changes, by what it gives: the amount.
func wantChanges(t *testing.T, tier, roll int, prize string) map[string]int {
	t.Helper()

	if prize == "a random POKéMON at a random level" {
		return map[string]int{" POKéMON": 1}
	}

	n, unit, _ := strings.Cut(strings.TrimSuffix(strings.TrimSuffix(prize, "s"), "¥"), " ")
	if amount, ok := strings.CutSuffix(prize, "¥"); ok {
		n, unit = amount, ""
	}

	v, err := strconv.Atoi(n)
	require.NoError(t, err, "tier %d roll %d: prize %q", tier, roll, prize)

	key := " " + unit
	if unit == "" {
		key = "¥"
	}

	return map[string]int{key: v}
}

func Test_levelFor(t *testing.T) {
	t.Parallel()

	t.Run("when the XP is below the next level's, should keep the level", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, uint8(5), levelFor(5, 125))
		assert.Equal(t, uint8(5), levelFor(5, 215))
	})

	t.Run("when the XP reaches the next level's, should level up", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, uint8(6), levelFor(5, 216))
	})

	t.Run("when the XP reaches several levels, should level up through all of them", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, uint8(10), levelFor(5, 1000))
		assert.Equal(t, uint8(34), levelFor(30, 42392))
	})

	t.Run("when the level is already above what the XP gives, should not drop", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, uint8(50), levelFor(50, 0))
	})

	t.Run("when the XP would pass level 255, should stop at 255", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, uint8(255), levelFor(1, 4294967295))
	})
}

func Test_New(t *testing.T) {
	t.Parallel()

	t.Run("when the game is made, should have the original's 86 menu items", func(t *testing.T) {
		t.Parallel()

		gam := New(nil, nil)
		total := 0

		var count func(items []*MenuItem)
		count = func(items []*MenuItem) {
			for _, m := range items {
				total++

				count(m.Items)
			}
		}
		count(gam.Menu)

		assert.Equal(t, 86, total, "menu items, the original's count")
	})
}

func Test_Game_available(t *testing.T) {
	t.Parallel()

	t.Run("when an item and every item above it are enabled and visible, should be available", func(t *testing.T) {
		t.Parallel()

		gam := New(nil, nil)

		assert.True(t, gam.available("Brock1"))
		assert.True(t, gam.available("Quit1"))
	})

	t.Run("when an item is disabled, should not be available", func(t *testing.T) {
		t.Parallel()

		gam := New(nil, nil)

		assert.False(t, gam.available("Misty1"))
	})

	t.Run("when an item above it is disabled, should not be available", func(t *testing.T) {
		t.Parallel()

		gam := New(nil, nil)

		assert.False(t, gam.available("Falkner1")) // JohtoGymLeaders1 is disabled
	})

	t.Run("when an item above it is hidden, should not be available", func(t *testing.T) {
		t.Parallel()

		gam := New(nil, nil)

		assert.False(t, gam.available("MakeVisible1")) // Debug1 is hidden
	})
}

func Test_Game_ClickMenu(t *testing.T) {
	t.Parallel()

	t.Run("when the item is disabled, should not run it", func(t *testing.T) {
		t.Parallel()

		gam := New(nil, nil)

		assert.False(t, gam.ClickMenu("Misty1"))
	})

	t.Run("when the item is unknown, should not run it", func(t *testing.T) {
		t.Parallel()

		gam := New(nil, nil)

		assert.False(t, gam.ClickMenu("NoSuchItem"))
	})

	t.Run("when the item is a user interface command, should not run it", func(t *testing.T) {
		t.Parallel()

		gam := New(nil, nil)

		assert.False(t, gam.ClickMenu("Quit1"))
	})
}

// The export scenario's page, as the original wrote it.
func Test_Game_ExportPage(t *testing.T) {
	t.Parallel()

	t.Run("when the party and the POKéDEX are exported, should write the original's page", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		sav := save.File{
			PlayerName: "Probe",
			PokeBalls:  3,
		}
		sav.Owned[0] = save.Slot{
			Species: 1,
			Level:   5,
			XP:      125,
		}
		sav.Owned[1] = save.Slot{
			Species: 150,
			Level:   100,
			XP:      1000000,
		}
		sav.Owned[2] = save.Slot{
			Species: 252,
			Level:   50,
			XP:      130000,
		}

		for _, s := range []int{1, 7, 150, 252} {
			sav.InPokedex[s] = true
		}

		require.NoError(t, gam.Load(`C:\Temp\probe-export.pgs`, sav.Bytes()))

		want, err := os.ReadFile("testdata/export.htm")
		require.NoError(t, err)

		got := gam.ExportPage(time.Date(2026, 10, 11, 17, 32, 51, 0, time.Local))

		assert.Equal(t, string(want), string(got), "the page differs from the original's")
	})

	t.Run("when a level lags its XP, should print the next level's XP unsigned, as the original", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		sav := save.File{
			PlayerName: "Probe",
			PokeBalls:  3,
		}
		sav.Owned[0] = save.Slot{
			Species: 6,
			Level:   30,
			XP:      50000, // past level 31's 29791
		}
		sav.Owned[1] = save.Slot{
			Species: 1,
			Level:   5,
			XP:      125,
		}
		sav.Owned[2] = save.Slot{
			Species: 150,
			Level:   100,
			XP:      1000000,
		}
		sav.Owned[3] = save.Slot{
			Species: 7,
			Level:   99,
			XP:      1000000,
		}

		for _, s := range []int{1, 6, 7, 150} {
			sav.InPokedex[s] = true
		}

		require.NoError(t, gam.Load(`C:\Temp\probe-audit-export-lagging.pgs`, sav.Bytes()))

		want, err := os.ReadFile("testdata/export-lagging.htm")
		require.NoError(t, err)

		got := gam.ExportPage(time.Date(2026, 10, 20, 8, 13, 16, 0, time.Local))

		assert.Equal(t, string(want), string(got), "the page differs from the original's")
	})
}

// uiFunc answers dialogs with a function.
type uiFunc func(kind DialogKind, yesNo bool, text string) bool

func (uf uiFunc) Message(kind DialogKind, yesNo bool, text string) bool {
	return uf(kind, yesNo, text)
}

// The arena shows what the original's labels and gauges showed, which
// lags the battle.
func Test_MainWindow(t *testing.T) {
	t.Parallel()

	t.Run("when a wild battle is fought, should show what the original's labels and gauges showed", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(1) // Charmander, L.5

		win := &gam.Main

		var during MainWindow
		gam.UI = uiFunc(func(kind DialogKind, yesNo bool, text string) bool {
			if text == ChoosePrompt {
				during = *win
			}

			return true
		})
		gam.StartWildBattle(25, 7, false) // Pikachu, L.7

		// While the choose prompt shows: the enemy's level and HP, but no attacks.
		assert.Equal(t, Shown{
			Species: 25,
			Level:   7,
			HP:      70,
		}, during.Enemy)
		assert.Equal(t, Shown{}, during.EnemyMoves)

		// After it: the enemy's attacks are listed, and no POKéBall shows.
		assert.Equal(t, win.Enemy, win.EnemyMoves)
		assert.False(t, win.Pokeball)

		gam.ClickParty(0)
		assert.Equal(t, Shown{
			Species: 4,
			Level:   5,
			HP:      50,
		}, win.Own)

		win.AttackIndex = 0

		gam.Go()
		assert.Equal(t, gam.EnemyHP, win.Enemy.HP)
		assert.Equal(t, gam.OwnHP, win.Own.HP)
		assert.NotEqual(t, 70, gam.EnemyHP)

		win.AttackIndex = 4 // Run!

		gam.Go()
		gam.EndFight()
		assert.Equal(t, Shown{}, win.Own)
		assert.Equal(t, 25, win.Enemy.Species)
	})

	t.Run("when a battle has ended, should turn both gauges green for the next one", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(1) // Charmander, L.5

		win := &gam.Main

		gam.StartWildBattle(25, 100, false) // Pikachu, L.100
		gam.ClickParty(0)

		gam.EnemyHP, gam.OwnHP = 200, 1 // Pikachu takes the attack and faints Charmander

		win.AttackIndex = 0

		gam.Go()
		require.Equal(t, Red, win.EnemyColor)
		require.Equal(t, Red, win.OwnColor)

		gam.EndFight()
		assert.Equal(t, Green, win.EnemyColor)
		assert.Equal(t, Green, win.OwnColor)
	})

	t.Run("when the wild POKéMON is in the POKéDEX, should show the POKéBall", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(1) // Charmander is registered

		gam.StartWildBattle(4, 5, false)

		assert.True(t, gam.Main.Pokeball)
	})

	t.Run("when POKéMON appear, should count the wild ones before the choose prompt", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(1)

		win := &gam.Main

		var during int
		gam.UI = uiFunc(func(kind DialogKind, yesNo bool, text string) bool {
			if text == ChoosePrompt {
				during = win.Appeared
			}

			return true
		})

		gam.StartWildBattle(25, 7, false)
		assert.Equal(t, 1, during)

		gam.StartWildBattle(25, 7, false)
		assert.Equal(t, 2, win.Appeared)

		gam.StartTrainerBattle(74, 12, "Brock")
		assert.Equal(t, 2, win.Appeared)
	})

	t.Run("when the player chooses a fighter, should count it", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(1)

		win := &gam.Main

		gam.StartWildBattle(25, 7, false)
		assert.Equal(t, 0, win.Chosen)

		gam.ClickParty(0)
		assert.Equal(t, 1, win.Chosen)
		assert.Equal(t, 4, win.Own.Species)
	})
}

func Test_gaugeColor(t *testing.T) {
	t.Parallel()

	t.Run("when more than half the HP is left, should be green", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Green, gaugeColor(50, 5))
		assert.Equal(t, Green, gaugeColor(26, 5))
	})

	t.Run("when half or less is left, should be yellow", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Yellow, gaugeColor(25, 5))
		assert.Equal(t, Yellow, gaugeColor(13, 5))
	})

	t.Run("when a quarter or less is left, should be red", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Red, gaugeColor(12, 5))
		assert.Equal(t, Red, gaugeColor(0, 5))
		assert.Equal(t, Red, gaugeColor(-3, 5))
	})
}

// Each step replays the ones before it on its own game, then records only
// the windows its own step shows.
func Test_Game_OnShow(t *testing.T) {
	t.Parallel()

	t.Run("when a new game starts, should show the main window", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)

		var shown []Window
		gam.OnShow = func(w Window) {
			shown = append(shown, w)
		}
		gam.NewGame("ash.pgs", "Ash", true)

		assert.Equal(t, []Window{MainForm}, shown)
	})

	t.Run("when a starter is chosen, should show the POKéDEX, Statistics and then the main window", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)

		var shown []Window
		gam.OnShow = func(w Window) {
			shown = append(shown, w)
		}
		gam.ChooseStarter(0)

		assert.Equal(t, []Window{PokedexForm, StatsForm, MainForm}, shown)
	})

	t.Run("when a wild battle starts, should show Battle Info", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(0)

		var shown []Window
		gam.OnShow = func(w Window) {
			shown = append(shown, w)
		}
		gam.StartWildBattle(25, 7, false)

		assert.Equal(t, []Window{InfoForm}, shown)
	})

	t.Run("when the POKéDEX is opened, should show it", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(0)
		gam.StartWildBattle(25, 7, false)

		var shown []Window
		gam.OnShow = func(w Window) {
			shown = append(shown, w)
		}
		gam.OpenPokedex()

		assert.Equal(t, []Window{PokedexForm}, shown)
	})
}

// kindsSince returns the kinds of Battle Info's lines from the from-th on,
// once it has checked that every line has one.
func kindsSince(t *testing.T, gam *Game, from int) []LineKind {
	t.Helper()

	require.Len(t, gam.Info.Kinds, len(gam.Info.Lines), "a line without its kind")

	return gam.Info.Kinds[from:]
}

// Each line of Battle Info has a kind, which the port colours it by.
func Test_Game_log(t *testing.T) {
	t.Parallel()

	t.Run("when the game is made, should give Memo1's design-time line the plain kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)

		assert.Equal(t, []string{""}, gam.Info.Lines)
		assert.Equal(t, []LineKind{Plain}, kindsSince(t, gam, 0))
	})

	t.Run("when a battle starts, should clear the kinds with the lines and log who comes out as plain", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)

		assert.Equal(t, []string{"Wild Weedle appeared!", "Charmander, I choose you!"}, gam.Info.Lines)
		assert.Equal(t, []LineKind{Plain, Plain}, kindsSince(t, gam, 0))
	})

	t.Run("when the fighter wins, should log the damage dealt, the faint, the experience, the level and the money by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		from := len(gam.Info.Lines)
		gam.member(1).XP = 6*6*6 - 1 // a level up at any win

		winBattle(t, gam)

		assert.Equal(t, []string{
			"Charmander used Scratch!",
			"Charmander did 9 points of damage to enemy Weedle.",
			"Enemy Weedle fainted!",
			"Charmander gained 91 experience points!",
			"Charmander grew to level 6!",
			"You got 250¥!",
		}, gam.Info.Lines[from:])
		assert.Equal(t, []LineKind{
			Plain,
			DamageDealt,
			EnemyFainted,
			ExperienceGained,
			LevelUp,
			PrizeMoney,
		}, kindsSince(t, gam, from))
	})

	t.Run("when the fighter loses, should log the damage taken, its faint and the money lost by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		from := len(gam.Info.Lines)

		loseBattle(t, gam)

		assert.Equal(t, []string{
			"Charmander used Scratch!",
			"Charmander did 9 points of damage to enemy Weedle.",
			"Enemy Weedle used Tackle!",
			"Enemy Weedle did 12 points of damage to Charmander.",
			"Charmander fainted!",
			"You lost 0¥!",
		}, gam.Info.Lines[from:])
		assert.Equal(t, []LineKind{
			Plain,
			DamageDealt,
			Plain,
			DamageTaken,
			OwnFainted,
			MoneyLost,
		}, kindsSince(t, gam, from))
	})

	t.Run("when the fighter lands a critical hit, should log it by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		from := len(gam.Info.Lines)
		gam.Rand.Seed = 29 // Scratch's first draw is a critical hit

		winBattle(t, gam)

		assert.Equal(t, []string{
			"Charmander used Scratch!",
			"Critical hit!",
			"Charmander did 1 points of damage to enemy Weedle.",
		}, gam.Info.Lines[from:from+3])
		assert.Equal(t, []LineKind{
			Plain,
			CriticalDealt,
			DamageDealt,
		}, kindsSince(t, gam, from)[:3])
	})

	t.Run("when the enemy lands a critical hit, should log it by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		from := len(gam.Info.Lines)
		gam.Rand.Seed = 10 // the enemy's attack is a critical hit, and Scratch isn't

		loseBattle(t, gam)

		assert.Equal(t, []string{
			"Charmander used Scratch!",
			"Charmander did 1 points of damage to enemy Weedle.",
			"Enemy Weedle used String Shot!",
			"Critical hit!",
			"It's not very effective...",
			"Enemy Weedle did 1 points of damage to Charmander.",
			"Charmander fainted!",
			"You lost 0¥!",
		}, gam.Info.Lines[from:])
		assert.Equal(t, []LineKind{
			Plain,
			DamageDealt,
			Plain,
			CriticalTaken,
			Plain,
			DamageTaken,
			OwnFainted,
			MoneyLost,
		}, kindsSince(t, gam, from))
	})

	t.Run("when a POKéBall catches the POKéMON, should log the catch and the new POKéDEX entry by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		from := len(gam.Info.Lines)

		catchWithBall(t, gam)

		assert.Equal(t, []string{
			"You threw a POKéBall.",
			"You caught Weedle!",
			"Weedle has been added to the POKéDEX!",
		}, gam.Info.Lines[from:])
		assert.Equal(t, []LineKind{Plain, Caught, Registered}, kindsSince(t, gam, from))
	})

	t.Run("when the Master Ball catches the POKéMON, should log the catch by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		from := len(gam.Info.Lines)

		catchWithMasterBall(t, gam)

		assert.Equal(t, "You caught Weedle!", gam.Info.Lines[from+1])
		assert.Equal(t, []LineKind{Plain, Caught, Registered}, kindsSince(t, gam, from))
	})

	t.Run("when a Potion heals the fighter, should log its HP by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		gam.OwnHP, gam.Potions = 10, 1
		gam.enableItems()

		from := len(gam.Info.Lines)

		gam.UseItem(Potion)

		assert.Equal(t, "Charmander recovered 40 HP!", gam.Info.Lines[from])
		assert.Equal(t, []LineKind{Healed, Plain, DamageTaken}, kindsSince(t, gam, from))
	})

	t.Run("when the player quits a tournament, should log the experience and the level lost by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		fighter := gam.member(1)
		fighter.Level, fighter.XP = 10, 1000 // a twentieth takes a level

		from := len(gam.Info.Lines)

		gam.quitTournament()

		assert.Equal(t, []string{
			"Quit the tournament! Charmander lost 50 experience points!",
			"Charmander lost a level!",
		}, gam.Info.Lines[from:])
		assert.Equal(t, []LineKind{ExperienceLost, LevelLost}, kindsSince(t, gam, from))
	})

	t.Run("when a shiny POKéMON appears, should log it by kind", func(t *testing.T) {
		t.Parallel()

		slot := save.Slot{
			Species: 4,
			Level:   50,
			XP:      125000,
		}
		gam, _ := shinyGame(t, slot)
		gam.RollShiny = always

		gam.FindPKMN()

		assert.Equal(t, "It's a shiny POKéMON!", gam.Info.Lines[1])
		assert.Equal(t, []LineKind{Plain, Shiny}, kindsSince(t, gam, 0))
	})

	t.Run("when the starter chosen is shiny, should log it by kind", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.RollShiny = always
		gam.NewGame("ash.pgs", "Ash", true)

		from := len(gam.Info.Lines)

		gam.ChooseStarter(1)

		assert.Equal(t, []string{
			"Charmander has been added to the POKéDEX!",
			"It's a shiny POKéMON!",
		}, gam.Info.Lines[from:])
		assert.Equal(t, []LineKind{Registered, Shiny}, kindsSince(t, gam, from))
	})
}
