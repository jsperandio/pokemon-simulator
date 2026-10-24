// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searchedAgain is searched's game after a win and Search Again:
// Charmander is out again, and hasn't acted.
func searchedAgain(t *testing.T) (*Game, *silent) {
	t.Helper()

	gam, ui := searched(t)
	winBattle(t, gam)
	gam.SearchAgain()
	require.True(t, gam.swapping())

	return gam, ui
}

func Test_Game_ClickParty(t *testing.T) {
	t.Parallel()

	t.Run("when the fighter Search Again sent hasn't acted, should send the one double-clicked in its place", func(t *testing.T) {
		t.Parallel()

		gam, ui := searchedAgain(t)
		fights, chosen, shown := gam.Fights, gam.Main.Chosen, len(ui.messages)

		gam.ClickParty(1) // Pikachu
		assert.Equal(t, []string{
			"# 025 Pikachu, L.50, XP:125000, XP to next level:7651",
			"Choose Pikachu?",
		}, ui.messages[shown:])
		assert.Equal(t, 2, gam.OwnIndex)
		assert.Equal(t, 500, gam.OwnHP)
		assert.Equal(t, 25, gam.Main.Own.Species)
		assert.Equal(t, -1, gam.Main.AttackIndex)
		assert.Equal(t, chosen+1, gam.Main.Chosen)
		assert.Equal(t, fights, gam.Fights, "the fight counted again")
		assert.Equal(t, "Pikachu, I choose you!", gam.Info.Lines[len(gam.Info.Lines)-1])
		assert.True(t, gam.swapping())
	})

	t.Run("when the fighter has changed, should send the new one at the next Search Again", func(t *testing.T) {
		t.Parallel()

		gam, _ := searchedAgain(t)
		gam.ClickParty(1) // Pikachu

		gam.Main.AttackIndex = 1

		winBattle(t, gam)
		gam.SearchAgain()
		assert.Equal(t, 2, gam.OwnIndex)
		assert.Equal(t, 25, gam.Main.Own.Species)
		assert.Equal(t, 1, gam.Main.AttackIndex)
	})

	t.Run("when the fighter out is double-clicked, should only show its details", func(t *testing.T) {
		t.Parallel()

		gam, ui := searchedAgain(t)
		chosen, shown := gam.Main.Chosen, len(ui.messages)

		gam.ClickParty(0) // Charmander
		assert.Len(t, ui.messages[shown:], 1)
		assert.Equal(t, chosen, gam.Main.Chosen)
		assert.True(t, gam.swapping())
	})

	t.Run("when a potion has no effect, should still let the fighter change", func(t *testing.T) {
		t.Parallel()

		gam, _ := searchedAgain(t)

		gam.Potions = 1

		gam.enableItems()
		gam.UseItem(Potion)
		require.Equal(t, "It won't have any effect.", gam.Info.Lines[len(gam.Info.Lines)-1])

		gam.ClickParty(1) // Pikachu
		assert.Equal(t, 2, gam.OwnIndex)
	})

	cases := []struct {
		name string
		act  func(gam *Game)
	}{
		{
			name: "when the fighter has attacked, should keep it",
			act: func(gam *Game) {
				gam.Go()
			},
		},
		{
			name: "when it has run, should keep it",
			act: func(gam *Game) {
				gam.Main.AttackIndex = 4

				gam.Go()
			},
		},
		{
			name: "when a ball has missed, should keep it",
			act: func(gam *Game) {
				gam.UseItem(PokeBall)
			},
		},
		{
			name: "when the Master Ball has caught the POKéMON, should keep it",
			act: func(gam *Game) {
				gam.UseItem(MasterBall)
			},
		},
	}

	for _, cas := range cases {
		t.Run(cas.name, func(t *testing.T) {
			t.Parallel()

			gam, _ := searchedAgain(t)

			gam.EnemyHP, gam.OwnHP = 10000, 10000 // no one faints, and no ball but the Master Ball catches

			cas.act(gam)

			chosen := gam.Main.Chosen

			gam.ClickParty(1) // Pikachu
			assert.False(t, gam.Stats.PartyEnabled)
			assert.Equal(t, chosen, gam.Main.Chosen)
			assert.Equal(t, 1, gam.OwnIndex)
		})
	}

	t.Run("when the player chose the fighter, should keep it", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		chosen := gam.Main.Chosen

		gam.ClickParty(1) // Pikachu
		assert.False(t, gam.Stats.PartyEnabled)
		assert.Equal(t, chosen, gam.Main.Chosen)
		assert.Equal(t, 1, gam.OwnIndex)
	})
}
