// SPDX-License-Identifier: AGPL-3.0-only

package gamedata

import (
	"strings"
	"testing"

	"github.com/jsperandio/pokemon-simulator/assets"
)

func loadEmbedded(t *testing.T) *Data {
	t.Helper()

	raw, err := assets.FS.ReadFile("pokegame.dat")
	if err != nil {
		t.Fatal(err)
	}

	d, err := Load(raw)
	if err != nil {
		t.Fatal(err)
	}

	return d
}

func TestLoadCounts(t *testing.T) {
	d := loadEmbedded(t)
	counts := []struct {
		name      string
		got, want int
	}{
		{"species", len(d.Species), NumSpecies},
		{"badges", len(d.Badges), NumBadges},
		{"moves", len(d.Moves), NumMoves},
		{"prizes", len(d.Prizes), NumPrizes},
		{"trainer names", len(d.TrainerNames), NumTrainerNames},
	}

	for _, c := range counts {
		if c.got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestTypes(t *testing.T) {
	d := loadEmbedded(t)
	if d.TypeName(Normal) != "Normal" || d.TypeName(Steel) != "Steel" || d.TypeName(Grass) != "Grass" {
		t.Errorf("type names: %q", d.TypeNames)
	}

	effects := []struct {
		atk, def Type
		want     float64
	}{
		{Normal, Ghost, 0},
		{Normal, Rock, 0.5},
		{Fire, Grass, 2},
		{Fire, Water, 0.5},
		{Fire, Steel, 2},
		{Water, Fire, 2},
		{Electric, Ground, 0},
		{Fire, NoType, 1},
	}
	for _, e := range effects {
		if got := d.Effect(e.atk, e.def); got != e.want {
			t.Errorf("Effect(%s, %s) = %v, want %v", d.TypeName(e.atk), d.TypeName(e.def), got, e.want)
		}
	}
}

func TestSpecies(t *testing.T) {
	d := loadEmbedded(t)
	tests := []Species{
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
		// The author's values differ from the official games (16 and 36).
		{ID: 7, Name: "Squirtle", Types: [2]Type{Water, NoType}, EvolveLevel: 18, EvolveTo: 8},
		{ID: 25, Name: "Pikachu", Types: [2]Type{Electric, NoType}, EvolveTo: 26},
		{ID: 29, Name: "Nidoran[F]"},
		// Branching evolution: a level but no single target.
		{ID: 133, Name: "Eevee", Types: [2]Type{Normal, NoType}, EvolveLevel: 30},
		{ID: 252, Name: "Treecko"},
		{ID: 386, Name: "Deoxys"},
	}

	for _, want := range tests {
		got := d.SpeciesByID(want.ID)
		if got == nil || got.Name != want.Name {
			t.Errorf("species %d: got %+v, want name %q", want.ID, got, want.Name)

			continue
		}

		if want.Types != [2]Type{} && got.Types != want.Types {
			t.Errorf("%s types: got %v, want %v", want.Name, got.Types, want.Types)
		}

		if want.EvolveLevel != 0 && got.EvolveLevel != want.EvolveLevel {
			t.Errorf("%s evolve level: got %d, want %d", want.Name, got.EvolveLevel, want.EvolveLevel)
		}

		if want.EvolveTo != 0 && got.EvolveTo != want.EvolveTo {
			t.Errorf("%s evolves to: got %d, want %d", want.Name, got.EvolveTo, want.EvolveTo)
		}

		if want.Moves != [4]int{} && got.Moves != want.Moves {
			t.Errorf("%s moves: got %v, want %v", want.Name, got.Moves, want.Moves)
		}
	}

	if eevee := d.SpeciesByID(133); eevee.EvolveTo != 0 {
		t.Errorf("Eevee evolves to %d, want 0 (chosen by the game)", eevee.EvolveTo)
	}

	if d.SpeciesByID(0) != nil || d.SpeciesByID(387) != nil {
		t.Error("SpeciesByID accepts out-of-range IDs")
	}
}

// moveMatches reports whether got is the move want; a Power of 0 in want
// isn't checked.
func moveMatches(got *Move, want Move) bool {
	if got == nil {
		return false
	}

	return got.Name == want.Name && got.Type == want.Type && (want.Power == 0 || got.Power == want.Power)
}

func TestMoves(t *testing.T) {
	d := loadEmbedded(t)
	tests := []Move{
		{ID: 1, Name: "Tackle", Power: 70, Type: Normal},
		{ID: 3, Name: "Leech Seed", Power: 110, Type: Grass},
		{ID: 8, Name: "Ember", Type: Fire},
	}

	for _, want := range tests {
		got := d.MoveByID(want.ID)
		if !moveMatches(got, want) {
			t.Errorf("move %d: got %+v, want %+v", want.ID, got, want)
		}
	}

	for _, s := range d.Species {
		for _, m := range s.Moves {
			if d.MoveByID(m) == nil {
				t.Errorf("%s knows unknown move %d", s.Name, m)
			}
		}
	}
}

func TestTexts(t *testing.T) {
	d := loadEmbedded(t)
	checks := []struct{ got, want string }{
		{d.Badges[0], "BoulderBadge"},
		{d.Badges[NumBadges-1], "RainBadge"},
		{d.Prizes[0], "3 POKéBALLs"},
		{d.Prizes[2], "2500¥"},
		{d.Prizes[NumPrizes-1], "a random POKéMON at a random level"},
		{d.TrainerNames[0], "Tracey"},
		{d.TrainerNames[NumTrainerNames-1], "Julia"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

func TestLoadErrors(t *testing.T) {
	raw, err := assets.FS.ReadFile("pokegame.dat")
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(string(raw), "\r\n")

	truncated := strings.Join(lines[:1000], "\r\n")
	if _, err := Load([]byte(truncated)); err == nil || !strings.Contains(err.Error(), "line 1001") {
		t.Errorf("truncated file: got error %v, want one naming line 1001", err)
	}

	bad := append([]string(nil), lines...)
	bad[699] = "four" // first type of Bulbasaur

	if _, err := Load([]byte(strings.Join(bad, "\r\n"))); err == nil || !strings.Contains(err.Error(), "line 700") {
		t.Errorf("non-numeric value: got error %v, want one naming line 700", err)
	}

	if _, err := Load([]byte(strings.ReplaceAll(string(raw), "\r\n", "\n"))); err != nil {
		t.Errorf("LF line endings: %v", err)
	}
}
