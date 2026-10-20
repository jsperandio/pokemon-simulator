// SPDX-License-Identifier: AGPL-3.0-only

package gamedata

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
)

// embeddedFile returns the POKéGAME.dat embedded in the port.
func embeddedFile(t *testing.T) []byte {
	t.Helper()

	raw, err := assets.FS.ReadFile("pokegame.dat")
	require.NoError(t, err)

	return raw
}

// loadEmbedded loads the POKéGAME.dat embedded in the port.
func loadEmbedded(t *testing.T) *Data {
	t.Helper()

	gdt, err := Load(embeddedFile(t))
	require.NoError(t, err)

	return gdt
}

func Test_Load(t *testing.T) {
	t.Parallel()

	raw := embeddedFile(t)
	gdt, err := Load(raw)
	require.NoError(t, err)

	lines := strings.Split(string(raw), "\r\n")

	t.Run("when the embedded file is loaded, should fill every table in full", func(t *testing.T) {
		t.Parallel()

		assert.Len(t, gdt.Species, NumSpecies, "species")
		assert.Len(t, gdt.Badges, NumBadges, "badges")
		assert.Len(t, gdt.Moves, NumMoves, "moves")
		assert.Len(t, gdt.Prizes, NumPrizes, "prizes")
		assert.Len(t, gdt.TrainerNames, NumTrainerNames, "trainer names")
	})

	t.Run("when the embedded file is loaded, should give its texts", func(t *testing.T) {
		t.Parallel()

		require.Len(t, gdt.Badges, NumBadges)
		require.Len(t, gdt.Prizes, NumPrizes)
		require.Len(t, gdt.TrainerNames, NumTrainerNames)

		assert.Equal(t, "BoulderBadge", gdt.Badges[0])
		assert.Equal(t, "RainBadge", gdt.Badges[NumBadges-1])
		assert.Equal(t, "3 POKéBALLs", gdt.Prizes[0])
		assert.Equal(t, "2500¥", gdt.Prizes[2])
		assert.Equal(t, "a random POKéMON at a random level", gdt.Prizes[NumPrizes-1])
		assert.Equal(t, "Tracey", gdt.TrainerNames[0])
		assert.Equal(t, "Julia", gdt.TrainerNames[NumTrainerNames-1])
	})

	t.Run("when the file is cut short, should fail naming line 1001", func(t *testing.T) {
		t.Parallel()

		_, err := Load([]byte(strings.Join(lines[:1000], "\r\n")))
		assert.ErrorContains(t, err, "line 1001")
	})

	t.Run("when a number's line holds a word, should fail naming line 700", func(t *testing.T) {
		t.Parallel()

		bad := append([]string(nil), lines...)
		bad[699] = "four" // first type of Bulbasaur

		_, err := Load([]byte(strings.Join(bad, "\r\n")))
		assert.ErrorContains(t, err, "line 700")
	})

	t.Run("when an effectiveness line holds a word, should fail naming line 1", func(t *testing.T) {
		t.Parallel()

		bad := append([]string(nil), lines...)
		bad[0] = "half" // Normal against Normal

		_, err := Load([]byte(strings.Join(bad, "\r\n")))
		assert.ErrorContains(t, err, "line 1: want a number")
	})

	t.Run("when a reference points at nothing, should fail naming its entry", func(t *testing.T) {
		t.Parallel()

		bad := append([]string(nil), lines...)
		bad[699] = "18" // first type of Bulbasaur, past Steel

		_, err := Load([]byte(strings.Join(bad, "\r\n")))
		assert.ErrorContains(t, err, "Bulbasaur has invalid types")
	})

	t.Run("when the line endings are LF, should still load", func(t *testing.T) {
		t.Parallel()

		_, err := Load([]byte(strings.ReplaceAll(string(raw), "\r\n", "\n")))
		assert.NoError(t, err)
	})
}

// validate rejects a POKéGAME.dat whose references point at nothing, so that
// a modified file fails when it loads rather than in a battle.
func Test_Data_validate(t *testing.T) {
	t.Parallel()

	t.Run("when a species' first type is none or past Steel, should fail", func(t *testing.T) {
		t.Parallel()

		gdt := loadEmbedded(t)

		gdt.Species[0].Types[0] = NoType
		assert.ErrorContains(t, gdt.validate(), "Bulbasaur has invalid types")

		gdt.Species[0].Types[0] = Steel + 1
		assert.ErrorContains(t, gdt.validate(), "Bulbasaur has invalid types")
	})

	t.Run("when a species' second type is past Steel, should fail", func(t *testing.T) {
		t.Parallel()

		gdt := loadEmbedded(t)
		gdt.Species[0].Types[1] = Steel + 1

		assert.ErrorContains(t, gdt.validate(), "Bulbasaur has invalid types")
	})

	t.Run("when a species evolves into one that doesn't exist, should fail", func(t *testing.T) {
		t.Parallel()

		gdt := loadEmbedded(t)

		gdt.Species[0].EvolveTo = -1
		assert.ErrorContains(t, gdt.validate(), "Bulbasaur evolves into unknown species -1")

		gdt.Species[0].EvolveTo = NumSpecies + 1
		assert.ErrorContains(t, gdt.validate(), "Bulbasaur evolves into unknown species 387")
	})

	t.Run("when a species knows a move that doesn't exist, should fail", func(t *testing.T) {
		t.Parallel()

		gdt := loadEmbedded(t)

		gdt.Species[0].Moves[0] = 0
		assert.ErrorContains(t, gdt.validate(), "Bulbasaur knows unknown move 0")

		gdt.Species[0].Moves[0] = NumMoves + 1
		assert.ErrorContains(t, gdt.validate(), "Bulbasaur knows unknown move 120")
	})

	t.Run("when a move's type is none or past Steel, should fail", func(t *testing.T) {
		t.Parallel()

		gdt := loadEmbedded(t)

		gdt.Moves[0].Type = NoType
		assert.ErrorContains(t, gdt.validate(), "move Tackle has invalid type 0")

		gdt.Moves[0].Type = Steel + 1
		assert.ErrorContains(t, gdt.validate(), "move Tackle has invalid type 18")
	})
}

func Test_Data_TypeName(t *testing.T) {
	t.Parallel()

	gdt := loadEmbedded(t)

	t.Run("when the type is in the chart, should give its name", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "Normal", gdt.TypeName(Normal))
		assert.Equal(t, "Steel", gdt.TypeName(Steel))
		assert.Equal(t, "Grass", gdt.TypeName(Grass))
	})

	t.Run("when the type is NoType, should give no name", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, gdt.TypeName(NoType))
	})
}

func Test_Data_Effect(t *testing.T) {
	t.Parallel()

	gdt := loadEmbedded(t)

	t.Run("when the defender is immune, should take no damage", func(t *testing.T) {
		t.Parallel()

		assertMultiplier(t, gdt, Normal, Ghost, 0)
		assertMultiplier(t, gdt, Electric, Ground, 0)
	})

	t.Run("when the defender resists, should halve the damage", func(t *testing.T) {
		t.Parallel()

		assertMultiplier(t, gdt, Normal, Rock, 0.5)
		assertMultiplier(t, gdt, Fire, Water, 0.5)
	})

	t.Run("when the defender is weak, should double the damage", func(t *testing.T) {
		t.Parallel()

		assertMultiplier(t, gdt, Fire, Grass, 2)
		assertMultiplier(t, gdt, Fire, Steel, 2)
		assertMultiplier(t, gdt, Water, Fire, 2)
	})

	t.Run("when the defender has no second type, should take normal damage", func(t *testing.T) {
		t.Parallel()

		assertMultiplier(t, gdt, Fire, NoType, 1)
	})
}

// assertMultiplier asserts that an attack of type atk does want times the
// damage to a defender of type def.
func assertMultiplier(t *testing.T, gdt *Data, atk, def Type, want float64) {
	t.Helper()

	assert.Equal(t, want, gdt.Effect(atk, def), //nolint:testifylint // exact: 0, 0.5, 1 and 2 are binary fractions
		"Effect(%s, %s)", gdt.TypeName(atk), gdt.TypeName(def))
}

func Test_Data_SpeciesByID(t *testing.T) {
	t.Parallel()

	gdt := loadEmbedded(t)

	t.Run("when the ID is a species, should give the fields its row gives", func(t *testing.T) {
		t.Parallel()

		// A row's zero fields are not checked.
		for _, cas := range []Species{
			{
				ID:          1,
				Name:        "Bulbasaur",
				Types:       [2]Type{Grass, Poison},
				EvolveLevel: 16,
				EvolveTo:    2,
				Moves:       [4]int{1, 2, 3, 4},
			},
			{ID: 3, Name: "Venusaur", Types: [2]Type{Grass, Poison}},
			{ID: 6, Name: "Charizard", Types: [2]Type{Fire, Flying}},
			{ID: 25, Name: "Pikachu", Types: [2]Type{Electric, NoType}, EvolveTo: 26},
			{ID: 29, Name: "Nidoran[F]"},
			{ID: 252, Name: "Treecko"},
			{ID: 386, Name: "Deoxys"},
		} {
			spc := gdt.SpeciesByID(cas.ID)
			if !assert.NotNil(t, spc, "ID %d", cas.ID) {
				continue
			}

			assert.Equal(t, cas.Name, spc.Name, "ID %d", cas.ID)

			if cas.Types != [2]Type{} {
				assert.Equal(t, cas.Types, spc.Types, "%s types", cas.Name)
			}

			if cas.EvolveLevel != 0 {
				assert.Equal(t, cas.EvolveLevel, spc.EvolveLevel, "%s evolve level", cas.Name)
			}

			if cas.EvolveTo != 0 {
				assert.Equal(t, cas.EvolveTo, spc.EvolveTo, "%s evolves to", cas.Name)
			}

			if cas.Moves != [4]int{} {
				assert.Equal(t, cas.Moves, spc.Moves, "%s moves", cas.Name)
			}
		}
	})

	// The author's values differ from the official games (16 and 36).
	t.Run("when Squirtle is read, should keep the author's evolution level and target", func(t *testing.T) {
		t.Parallel()

		spc := gdt.SpeciesByID(7)
		require.NotNil(t, spc)

		assert.Equal(t, "Squirtle", spc.Name)
		assert.Equal(t, [2]Type{Water, NoType}, spc.Types)
		assert.Equal(t, 18, spc.EvolveLevel)
		assert.Equal(t, 8, spc.EvolveTo)
	})

	// Branching evolution: a level but no single target.
	t.Run("when Eevee is read, should evolve at a level with no single target", func(t *testing.T) {
		t.Parallel()

		eevee := gdt.SpeciesByID(133)
		require.NotNil(t, eevee)

		assert.Equal(t, "Eevee", eevee.Name)
		assert.Equal(t, [2]Type{Normal, NoType}, eevee.Types)
		assert.Equal(t, 30, eevee.EvolveLevel)
		assert.Zero(t, eevee.EvolveTo, "Eevee evolves to a target, want none (chosen by the game)")
	})

	t.Run("when the ID is out of range, should give nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, gdt.SpeciesByID(0))
		assert.Nil(t, gdt.SpeciesByID(387))
	})
}

func Test_Data_MoveByID(t *testing.T) {
	t.Parallel()

	gdt := loadEmbedded(t)

	t.Run("when the ID is a move, should give its name, power and type", func(t *testing.T) {
		t.Parallel()

		// A row's zero power is not checked.
		for _, cas := range []Move{
			{ID: 1, Name: "Tackle", Power: 70, Type: Normal},
			{ID: 3, Name: "Leech Seed", Power: 110, Type: Grass},
			{ID: 8, Name: "Ember", Type: Fire},
		} {
			mov := gdt.MoveByID(cas.ID)
			if !assert.NotNil(t, mov, "move %d", cas.ID) {
				continue
			}

			assert.Equal(t, cas.Name, mov.Name, "move %d", cas.ID)
			assert.Equal(t, cas.Type, mov.Type, "%s type", cas.Name)

			if cas.Power != 0 {
				assert.Equal(t, cas.Power, mov.Power, "%s power", cas.Name)
			}
		}
	})

	t.Run("when a species knows a move, should find that move", func(t *testing.T) {
		t.Parallel()

		for _, spc := range gdt.Species {
			for _, mid := range spc.Moves {
				assert.NotNil(t, gdt.MoveByID(mid), "%s knows unknown move %d", spc.Name, mid)
			}
		}
	})

	t.Run("when the ID is out of range, should give nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, gdt.MoveByID(0))
		assert.Nil(t, gdt.MoveByID(120))
	})
}
