// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searched is a new game with Charmander, L.5, and Pikachu, L.50, in a
// battle Search for a POKéMON found, with Charmander out and its first
// attack selected.
func searched(t *testing.T) (*Game, *silent) {
	t.Helper()

	gam, ui := newGame(t)
	gam.NewGame("ash.pgs", "Ash", true)
	gam.ChooseStarter(1) // Charmander, L.5
	gam.AddPKMN(25, "Pikachu", 50, false)

	gam.MasterBalls = 1
	gam.Rand.Seed = 1 // the same wild POKéMON every time: the battles must be winnable with Scratch

	gam.FindPKMN()
	gam.ClickParty(0)

	gam.Main.AttackIndex = 0

	return gam, ui
}

// withMewtwo is a new game with Charmander, L.5, and Mewtwo, L.70, before
// the Mewtwo-power picker.
func withMewtwo(t *testing.T) (*Game, *silent) {
	t.Helper()

	gam, ui := newGame(t)
	gam.NewGame("ash.pgs", "Ash", true)
	gam.ChooseStarter(1) // Charmander, L.5
	gam.AddPKMN(150, "Mewtwo", 70, false)

	gam.Rand.Seed = 1 // the same draws every time, as searched's

	gam.FindPKMN()
	require.True(t, gam.Main.Picker)

	return gam, ui
}

// winBattle wins the battle with the attack selected: the enemy has 1 HP
// left.
func winBattle(t *testing.T, gam *Game) {
	t.Helper()

	gam.EnemyHP = 1

	gam.Go()
	require.True(t, gam.Main.EndFight, "the battle goes on")
}

// loseBattle loses it: the fighter has 1 HP left, and the enemy more than
// an attack takes.
func loseBattle(t *testing.T, gam *Game) {
	t.Helper()

	gam.EnemyHP, gam.OwnHP = 10000, 1

	gam.Go()
	require.Zero(t, gam.BattleType, "the fighter didn't faint")
}

// flee runs from it.
func flee(t *testing.T, gam *Game) {
	t.Helper()

	gam.Main.AttackIndex = 4

	gam.Go()
	require.True(t, gam.Main.EndFight, "the fighter didn't run")
}

// catchWithBall catches the enemy, down to 1 HP, with a POKéBall, at the
// first seed whose draw catches it.
func catchWithBall(t *testing.T, gam *Game) {
	t.Helper()

	gam.EnemyHP = 1

	for gam.Rand.Seed = 0; ; gam.Rand.Seed++ {
		seed := gam.Rand.Seed
		if gam.Rand.Random(50) >= gam.EnemyHP+2 {
			gam.Rand.Seed = seed

			break
		}
	}

	gam.UseItem(PokeBall)
	require.Equal(t, uint8(1), gam.BattleType, "the POKéBall missed")
}

// catchWithMasterBall catches it with the Master Ball.
func catchWithMasterBall(t *testing.T, gam *Game) {
	t.Helper()

	gam.UseItem(MasterBall)
	require.Equal(t, uint8(1), gam.BattleType, "the Master Ball wasn't thrown")
}

func Test_Game_CanSearchAgain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		end  func(t *testing.T, gam *Game)
	}{
		{
			name: "when the battle was won, should show",
			end:  winBattle,
		},
		{
			name: "when the fighter ran, should show",
			end:  flee,
		},
		{
			name: "when the battle was lost, should show",
			end:  loseBattle,
		},
		{
			name: "when a POKéBall caught the POKéMON, should show",
			end:  catchWithBall,
		},
		{
			name: "when the Master Ball caught it, should show",
			end:  catchWithMasterBall,
		},
	}

	for _, cas := range cases {
		t.Run(cas.name, func(t *testing.T) {
			t.Parallel()

			gam, _ := searched(t)
			cas.end(t, gam)

			assert.True(t, gam.CanSearchAgain())
		})
	}

	t.Run("when the Mewtwo-power picker chose the POKéMON, or drew it, should show", func(t *testing.T) {
		t.Parallel()

		gam, _ := withMewtwo(t)
		gam.PickSpecies(24) // Pikachu
		gam.PickGo()
		gam.ClickParty(0)

		gam.Main.AttackIndex = 0

		winBattle(t, gam)
		assert.True(t, gam.CanSearchAgain())

		gam, _ = withMewtwo(t)
		gam.PickRandom()
		gam.ClickParty(0)

		gam.Main.AttackIndex = 0

		winBattle(t, gam)
		assert.True(t, gam.CanSearchAgain())
	})

	t.Run("when the battle goes on, should hide", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)

		assert.False(t, gam.CanSearchAgain())
	})

	t.Run("when End Fight was clicked, should hide", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		winBattle(t, gam)
		gam.EndFight()

		assert.False(t, gam.CanSearchAgain())
	})

	t.Run("when Mewtwo was fought from the menu, should hide", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		winBattle(t, gam)
		gam.EndFight()

		gam.menu["Mewtwo1"].Enabled = true

		require.True(t, gam.ClickMenu("Mewtwo1"))
		gam.ClickParty(0)

		gam.Main.AttackIndex = 0

		winBattle(t, gam)
		assert.False(t, gam.CanSearchAgain())
	})

	t.Run("when the battle was a trainer's, should hide", func(t *testing.T) {
		t.Parallel()

		gam, _ := newGame(t)
		gam.NewGame("ash.pgs", "Ash", true)
		gam.ChooseStarter(1)
		gam.StartTrainerBattle(74, 12, "Brock")
		gam.ClickParty(0)

		gam.Main.AttackIndex = 0

		winBattle(t, gam)
		assert.False(t, gam.CanSearchAgain())
	})
}

func Test_Game_SearchAgain(t *testing.T) {
	t.Parallel()

	t.Run("when a battle a search found is over, should send the same fighter out with no prompt, details or confirmation", func(t *testing.T) {
		t.Parallel()

		gam, ui := searched(t)
		winBattle(t, gam)

		fights, shown, wins := gam.Fights, len(ui.messages), gam.Wins

		gam.SearchAgain()
		assert.True(t, gam.InWildBattle)
		assert.True(t, gam.Main.Go)
		assert.Empty(t, ui.messages[shown:])
		assert.Equal(t, 1, gam.OwnIndex)
		assert.Equal(t, 4, gam.Main.Own.Species)
		assert.Equal(t, gam.Main.Own.Level*10, gam.OwnHP)
		assert.Equal(t, fights+1, gam.Fights)
		assert.Equal(t, wins+1, gam.Wins, "End Fight didn't count the win")
		assert.Contains(t, gam.Info.Lines, "Charmander, I choose you!")
	})

	t.Run("when the fighter attacked, should keep its attack selected", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)

		gam.Main.AttackIndex = 2

		winBattle(t, gam)
		gam.SearchAgain()

		assert.Equal(t, 2, gam.Main.AttackIndex)
	})

	t.Run("when the fighter ran, should select no attack", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		flee(t, gam)
		gam.SearchAgain()

		assert.Equal(t, -1, gam.Main.AttackIndex)
	})

	t.Run("when the wild POKéMON was caught, should count the catch and send the same fighter", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		catchWithBall(t, gam)

		catches := gam.Catches

		gam.SearchAgain()
		assert.Equal(t, catches+1, gam.Catches)
		assert.Len(t, gam.Party(), 3)
		assert.Equal(t, 1, gam.OwnIndex)
		assert.Equal(t, 4, gam.Main.Own.Species)
	})

	t.Run("when the Mewtwo-power picker chose the POKéMON, should fight it again as an ordinary wild battle", func(t *testing.T) {
		t.Parallel()

		gam, ui := withMewtwo(t)
		gam.PickSpecies(24) // Pikachu
		gam.PickLevel(42)

		gam.BattleType = 3 // a Gym Leader beaten: the picker's Go! keeps it

		gam.PickGo()
		gam.ClickParty(0)

		gam.Main.AttackIndex = 0

		winBattle(t, gam)
		gam.SearchAgain() // End Fight gives the badge, as in the original

		badges, shown := gam.GymProgress, len(ui.messages)

		assert.Equal(t, 25, gam.Main.Enemy.Species)
		assert.Equal(t, 42, gam.Main.Enemy.Level)
		assert.Equal(t, uint8(2), gam.BattleType)

		winBattle(t, gam)
		gam.SearchAgain()
		assert.Equal(t, badges, gam.GymProgress)
		assert.Empty(t, ui.messages[shown:])
	})

	t.Run("when the picker drew the POKéMON, should draw again", func(t *testing.T) {
		t.Parallel()

		gam, ui := withMewtwo(t)
		gam.PickRandom()
		gam.ClickParty(0)

		gam.Main.AttackIndex = 0

		winBattle(t, gam)

		rnd, shown := *gam.Rand, len(ui.messages)
		species, level := rnd.Random(386)+1, rnd.Random(11)+90

		gam.SearchAgain()
		assert.Equal(t, species, gam.Main.Enemy.Species)
		assert.Equal(t, level, gam.Main.Enemy.Level)
		assert.Equal(t, uint8(2), gam.BattleType)
		assert.Empty(t, ui.messages[shown:])
	})

	t.Run("when the battle goes on, should do nothing", func(t *testing.T) {
		t.Parallel()

		gam, _ := searched(t)
		fights, appeared := gam.Fights, gam.Main.Appeared

		gam.SearchAgain()
		assert.Equal(t, fights, gam.Fights)
		assert.Equal(t, appeared, gam.Main.Appeared)
	})
}
