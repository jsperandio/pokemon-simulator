// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// shinyGame loads a game whose party is slots, with the author beaten (so
// Mewtwo can be fought), POKéBalls and a Master Ball.
func shinyGame(t *testing.T, slots ...save.Slot) (*Game, *silent) {
	t.Helper()

	gam, ui := newGame(t)
	sav := save.File{
		PlayerName:  "Probe",
		GymProgress: 24,
		BeatLance:   true,
		BeatAsh:     true,
		BeatLink:    true,
		PokeBalls:   3,
		MasterBalls: 1,
	}
	copy(sav.Owned[:], slots)

	for _, slt := range slots {
		sav.InPokedex[slt.Species] = true
	}

	require.NoError(t, gam.Load(`C:\Temp\probe-shiny.pgs`, sav.Bytes()))

	return gam, ui
}

// always rolls a shiny POKéMON every time.
func always() bool {
	return true
}

func Test_RandomShiny(t *testing.T) {
	t.Parallel()

	t.Run("when rolled a million times, should come up about once in ShinyOdds", func(t *testing.T) {
		t.Parallel()

		hits := 0

		for range 1000000 {
			if RandomShiny() {
				hits++
			}
		}

		// 244 expected; the bounds are six standard deviations away.
		assert.InDelta(t, 1000000/ShinyOdds, hits, 94)
	})
}

func Test_Game_FindPKMN(t *testing.T) {
	t.Parallel()

	slot := save.Slot{
		Species: 4,
		Level:   50,
		XP:      125000,
	}

	t.Run("when the roll is shiny, should meet a shiny POKéMON and say so after it appears", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, slot)
		gam.RollShiny = always

		gam.FindPKMN()

		assert.True(t, gam.EnemyShiny)
		assert.True(t, gam.Main.Enemy.Shiny)

		lines := gam.Info.Lines
		require.Len(t, lines, 2)
		assert.Equal(t, "Wild "+gam.name(gam.EnemySpecies)+" appeared!", lines[0])
		assert.Equal(t, "It's a shiny POKéMON!", lines[1])
	})

	t.Run("when no roll is set, should meet a POKéMON that isn't shiny", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, slot)

		gam.FindPKMN()

		assert.False(t, gam.EnemyShiny)
		assert.Len(t, gam.Info.Lines, 1)
	})

	t.Run("when it rolls, should leave the original's draws as they were", func(t *testing.T) {
		t.Parallel()

		plain, _ := shinyGame(t, slot)
		shiny, _ := shinyGame(t, slot)
		shiny.RollShiny = always
		plain.Rand.Seed, shiny.Rand.Seed = 7, 7

		plain.FindPKMN()
		shiny.FindPKMN()

		assert.Equal(t, plain.EnemySpecies, shiny.EnemySpecies)
		assert.Equal(t, plain.EnemyLevel, shiny.EnemyLevel)
		assert.Equal(t, plain.Rand.Seed, shiny.Rand.Seed)
	})

	t.Run("when the party has a Mewtwo, should never make the picker's POKéMON shiny", func(t *testing.T) {
		t.Parallel()

		mewtwo := save.Slot{
			Species: 150,
			Level:   100,
			XP:      1000000,
		}

		gam, _ := shinyGame(t, mewtwo)
		gam.RollShiny = always

		gam.FindPKMN()
		gam.PickSpecies(129) // Gyarados
		gam.PickGo()
		assert.False(t, gam.EnemyShiny, "Go")

		gam.EndFight()
		gam.FindPKMN()
		gam.PickRandom()
		assert.False(t, gam.EnemyShiny, "Random")
	})
}

func Test_Game_ClickMenu_mewtwo(t *testing.T) {
	t.Parallel()

	mewtwo := save.Slot{
		Species: 150,
		Level:   100,
		XP:      1000000,
	}
	shinyMewtwo := mewtwo
	shinyMewtwo.Shiny = true

	t.Run("when the Mewtwo fight rolls shiny, should meet a shiny Mewtwo", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, save.Slot{
			Species: 6,
			Level:   100,
			XP:      1000000,
		})
		gam.RollShiny = always

		require.True(t, gam.ClickMenu("Mewtwo1"))
		assert.True(t, gam.EnemyShiny)
		assert.Equal(t, 150, gam.EnemySpecies)
	})

	t.Run("when the party has a normal Mewtwo, should let the shiny one be caught and keep both", func(t *testing.T) {
		t.Parallel()

		gam, ui := shinyGame(t, mewtwo)
		gam.RollShiny = always

		gam.ClickMenu("Mewtwo1")
		gam.ClickParty(0)
		require.True(t, gam.ItemUsable(MasterBall), "the Master Ball is disabled")

		gam.UseItem(MasterBall)

		party := gam.Party()
		require.Len(t, party, 2)
		assert.False(t, party[0].Shiny)
		assert.True(t, party[1].Shiny)
		assert.Contains(t, ui.messages, "NOTE: You can only catch one shiny Mewtwo at a time. You must release the one(s) you already have if you want to catch another.")
	})

	t.Run("when the party has a shiny Mewtwo, should not let a second shiny one be caught", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, shinyMewtwo)
		gam.RollShiny = always

		gam.ClickMenu("Mewtwo1")
		gam.ClickParty(0)

		assert.False(t, gam.ItemUsable(MasterBall))
		assert.False(t, gam.ItemUsable(PokeBall))
	})

	t.Run("when the party has a normal Mewtwo, should not let a second normal one be caught", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, mewtwo, shinyMewtwo)

		gam.ClickMenu("Mewtwo1")
		gam.ClickParty(0)

		assert.False(t, gam.ItemUsable(MasterBall))
	})
}

func Test_Game_UseItem_shiny(t *testing.T) {
	t.Parallel()

	t.Run("when a shiny POKéMON is caught, should join the party shiny", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, save.Slot{
			Species: 4,
			Level:   50,
			XP:      125000,
		})
		gam.RollShiny = always

		gam.FindPKMN()
		gam.ClickParty(0)
		gam.UseItem(MasterBall)

		party := gam.Party()
		require.Len(t, party, 2)
		assert.Equal(t, gam.EnemySpecies, int(party[1].Species))
		assert.True(t, party[1].Shiny)
	})

	t.Run("when a trainer battle follows a shiny wild one, should not be shiny", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, save.Slot{
			Species: 4,
			Level:   50,
			XP:      125000,
		})
		gam.RollShiny = always

		gam.FindPKMN()
		gam.EndFight()
		gam.StartTrainerBattle(74, 12, "Brock")

		assert.False(t, gam.EnemyShiny)
		assert.False(t, gam.Main.Enemy.Shiny)
	})
}

func Test_Game_ClickParty_shiny(t *testing.T) {
	t.Parallel()

	t.Run("when a shiny member is double clicked, should say so in its details", func(t *testing.T) {
		t.Parallel()

		gam, ui := shinyGame(t, save.Slot{
			Species: 130,
			Level:   50,
			XP:      125000,
			Shiny:   true,
		})

		gam.ClickParty(0)

		assert.Contains(t, ui.messages, "# 130 Gyarados (shiny), L.50, XP:125000, XP to next level:7651")
	})

	t.Run("when a shiny member fights, should show it shiny in the arena", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, save.Slot{
			Species: 130,
			Level:   50,
			XP:      125000,
			Shiny:   true,
		})

		gam.FindPKMN()
		gam.ClickParty(0)

		assert.True(t, gam.Main.Own.Shiny)
	})

	t.Run("when a shiny member evolves, should stay shiny", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, save.Slot{
			Species: 129,
			Level:   20,
			XP:      8000,
			Shiny:   true,
		})

		gam.ClickMenu("Evolve1")
		gam.ClickParty(0)

		party := gam.Party()
		require.Len(t, party, 1)
		assert.Equal(t, int32(130), party[0].Species)
		assert.True(t, party[0].Shiny)
	})

	t.Run("when a shiny Nincada evolves with a Shedinja, should give a shiny Shedinja", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t, save.Slot{
			Species: 290,
			Level:   20,
			XP:      8000,
			Shiny:   true,
		})
		gam.Rand.Seed = shedinjaSeed()

		gam.ClickMenu("Evolve1")
		gam.ClickParty(0)

		party := gam.Party()
		require.Len(t, party, 2)
		assert.Equal(t, int32(291), party[0].Species)
		assert.Equal(t, int32(292), party[1].Species)
		assert.True(t, party[1].Shiny)
	})

	t.Run("when a member before a shiny one is released, should keep the shiny one shiny", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t,
			save.Slot{
				Species: 16,
				Level:   5,
				XP:      125,
			},
			save.Slot{
				Species: 130,
				Level:   50,
				XP:      125000,
				Shiny:   true,
			},
			save.Slot{
				Species: 19,
				Level:   5,
				XP:      125,
			},
		)

		gam.ClickMenu("ReleasePKMN1")
		gam.ClickParty(0)

		party := gam.Party()
		require.Len(t, party, 2)
		assert.True(t, party[0].Shiny, "Gyarados")
		assert.False(t, party[1].Shiny, "Rattata")
	})
}

// shedinjaSeed is a RandSeed whose next Random(4) isn't 0: Nincada then
// gives a Shedinja too.
func shedinjaSeed() uint32 {
	seed := uint32(1)
	for {
		rnd := delphi.Rand{Seed: seed}
		if rnd.Random(4) != 0 {
			return seed
		}

		seed++
	}
}

func Test_Game_ExportPage_shiny(t *testing.T) {
	t.Parallel()

	t.Run("when a member is shiny, should say so after its name", func(t *testing.T) {
		t.Parallel()

		gam, _ := shinyGame(t,
			save.Slot{
				Species: 130,
				Level:   50,
				XP:      125000,
				Shiny:   true,
			},
			save.Slot{
				Species: 130,
				Level:   50,
				XP:      125000,
			},
		)

		page := string(gam.ExportPage(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)))

		assert.Contains(t, page, "<td>1</td><td># 130 Gyarados (shiny)</td>")
		assert.Contains(t, page, "<td>2</td><td># 130 Gyarados</td>")
	})
}

func Test_Game_PokedexShiny(t *testing.T) {
	t.Parallel()

	t.Run("when it is ticked with a species shown, should animate the species again", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.SelectPokedex(129)

		before := gam.Pokedex.Selected

		gam.PokedexShiny(true)

		assert.True(t, gam.Pokedex.Shiny)
		assert.Equal(t, before+1, gam.Pokedex.Selected)
	})

	t.Run("when it is ticked again, should change nothing", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.SelectPokedex(129)
		gam.PokedexShiny(true)

		before := gam.Pokedex.Selected

		gam.PokedexShiny(true)

		assert.Equal(t, before, gam.Pokedex.Selected)
	})

	t.Run("when it is ticked with no species shown, should animate nothing", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)

		gam.PokedexShiny(true)

		assert.True(t, gam.Pokedex.Shiny)
		assert.Zero(t, gam.Pokedex.Selected)
	})
}

// rolls returns a roll that gives each of results in turn.
func rolls(results ...bool) func() bool {
	return func() bool {
		res := results[0]
		results = results[1:]

		return res
	}
}

func Test_Game_NewGame(t *testing.T) {
	t.Parallel()

	t.Run("when a new game starts, should roll each starter on its own", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.RollShiny = rolls(true, false, true)
		gam.NewGame("ash.pgs", "Ash", true)

		assert.Equal(t, [3]bool{true, false, true}, gam.Main.StarterShiny)
	})

	t.Run("when there is no roll, should make no starter shiny", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)

		assert.Equal(t, [3]bool{}, gam.Main.StarterShiny)
	})
}

func Test_Game_ChooseStarter(t *testing.T) {
	t.Parallel()

	t.Run("when the starter chosen is shiny, should add it shiny and say so", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.RollShiny = always
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(1)

		want := []string{
			"Charmander has been added to the POKéDEX!",
			"It's a shiny POKéMON!",
		}
		party := gam.Party()

		require.Len(t, party, 1)
		assert.Equal(t, int32(4), party[0].Species)
		assert.True(t, party[0].Shiny)
		assert.Equal(t, want, gam.Info.Lines[len(gam.Info.Lines)-2:])
	})

	t.Run("when only another starter is shiny, should add the one chosen as it is", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.RollShiny = rolls(true, false, true)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(1)

		party := gam.Party()

		require.Len(t, party, 1)
		assert.False(t, party[0].Shiny)
		assert.NotContains(t, gam.Info.Lines, "It's a shiny POKéMON!")
	})
}
