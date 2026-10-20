// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"fmt"
	"slices"

	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// Tournaments: eight random trainers in a row, an entry fee, a level range,
// and a prize for winning all eight.

// tiers are the tournaments, by tier (1–6).
var tiers = [...]struct {
	item   string
	lo, hi uint8
	fee    uint32
}{
	1: {"VeryEasyLevels2101", 2, 10, 1000},
	2: {"EasyLevels10301", 10, 30, 2000},
	3: {"IntermediateLevels30501", 30, 50, 3000},
	4: {"HardLevels50701", 50, 70, 5000},
	5: {"VeryHardLevels70991", 70, 99, 7000},
	6: {"PrimeCupLevel1001", 100, 100, 10000},
}

// enterTournament is a Tournaments menu item: a party member in the level
// range and the fee are needed.
func (g *Game) enterTournament(tier int) {
	t := tiers[tier]
	ok := slices.ContainsFunc(g.Party(), func(m save.Slot) bool {
		return t.lo <= m.Level && m.Level <= t.hi
	})

	if !ok {
		g.message(Warning, "None of your POKéMON meet the level requirements.")

		return
	}

	if g.Money < t.fee {
		g.message(Information, "You don't have enough money!")

		return
	}

	g.setUpTournament(tier)

	g.Money -= t.fee
}

// Species no tournament trainer has: Magikarp, Ditto, Unown, Smeargle,
// Feebas.
var notInTournaments = []int{129, 132, 201, 235, 349}

// Species the two easiest tiers leave out besides evolved ones: the
// Eeveelutions, Vileplume, Bellossom, Politoed, Poliwrath, Slowbro,
// Slowking, Hitmontop, Hitmonlee and Hitmonchan.
var notInEasyTiers = []int{134, 135, 136, 196, 197, 45, 182, 186, 62, 80, 199, 237, 106, 107}

// setUpTournament draws the eight opponents and starts the first battle
// (TForm4.SetUpTournament). Everything is drawn again until the trainers
// are distinct and the species allowed.
func (g *Game) setUpTournament(tier int) {
	g.MinLevel, g.MaxLevel = tiers[tier].lo, tiers[tier].hi

	for !g.drawTournament(tier) {
	}

	g.BattleType = uint8(tier*10 + 1) //nolint:gosec // tiers 1–6 make 11–61
	g.StartTrainerBattle(g.TournamentSpecies[0], int(g.TournamentLevels[0]), g.trainerName(0))

	g.InTournament = true
}

// drawTournament draws a set of opponents and reports whether it is
// allowed. Each part is drawn only while the parts before it are allowed.
func (g *Game) drawTournament(tier int) bool {
	if !g.drawTrainers() || !g.drawSpecies() {
		return false
	}

	g.drawLevels()

	if tier < 3 { // unevolved species only
		for _, s := range g.TournamentSpecies {
			if g.isEvolved(s) || slices.Contains(notInEasyTiers, s) {
				return false
			}
		}
	}

	// Species 387–389 don't exist, and a set the original accepts with
	// one crashes it (tournament-387). The port draws again then, and
	// only then, so the draws stay the original's.
	for _, s := range g.TournamentSpecies {
		if s > len(g.Data.Species) {
			return false
		}
	}

	return true
}

// drawTrainers draws the eight trainers and reports whether they are
// distinct.
func (g *Game) drawTrainers() bool {
	for i := range g.TournamentTrainers {
		g.TournamentTrainers[i] = uint8(g.Rand.Random(64) + 1) //nolint:gosec // 1–64
	}

	for i, a := range g.TournamentTrainers {
		if slices.Contains(g.TournamentTrainers[i+1:], a) {
			return false
		}
	}

	return true
}

// drawSpecies draws the trainers' species and reports whether every
// trainer may have its own; all eight are drawn either way.
func (g *Game) drawSpecies() bool {
	ok := true

	for i := range g.TournamentSpecies {
		s := g.Rand.Random(386) + 4 // so 1–3 never come, and 387–389 can
		g.TournamentSpecies[i] = s

		if slices.Contains(notInTournaments, s) {
			ok = false
		}
	}

	return ok
}

// drawLevels draws the trainers' levels in the tournament's range.
func (g *Game) drawLevels() {
	lo, hi := int(g.MinLevel), int(g.MaxLevel)
	for i := range g.TournamentLevels {
		l := g.Rand.Random(hi-lo+5) + lo                   // hi comes 5 times as often
		g.TournamentLevels[i] = uint8(min(max(l, lo), hi)) //nolint:gosec // between two Bytes
	}
}

// isEvolved reports whether some species evolves into s.
func (g *Game) isEvolved(s int) bool {
	return slices.ContainsFunc(g.Data.Species, func(spc gamedata.Species) bool {
		return spc.EvolveTo == s
	})
}

// trainerName is TrainerNames[TournamentTrainers[i]].
func (g *Game) trainerName(i int) string {
	if i < 0 || i >= len(g.TournamentTrainers) {
		return ""
	}

	n := int(g.TournamentTrainers[i])
	if n < 1 || n > len(g.Data.TrainerNames) {
		return ""
	}

	return g.Data.TrainerNames[n-1]
}

// handleTournament follows a tournament win (TForm4.HandleTournament): the
// next round, or the prize after the eighth.
func (g *Game) handleTournament() {
	round := int(g.BattleType % 10) // the round won
	if round < 8 {
		g.message(Information, fmt.Sprintf("Congratulations! You beat %s! Next up is trainer #%d, %s with a L.%d %s.",
			g.trainerName(round-1), round+1, g.trainerName(round), g.TournamentLevels[round], g.name(g.TournamentSpecies[round])))

		g.BattleType++
		g.StartTrainerBattle(g.TournamentSpecies[round], int(g.TournamentLevels[round]), g.trainerName(round))

		g.InTournament = true
	}

	if round != 8 {
		return
	}

	g.message(Information, "Congratulations! You beat "+g.trainerName(7)+"! You won the tournament! Now for the unveiling of your prize...")
	g.recomputeLevels()

	g.MinLevel, g.MaxLevel = 2, 100
	g.InTournament = false
	roll := g.Rand.Random(3) + 1
	tier := int(g.BattleType / 10)

	if len(g.Party()) == len(g.Owned) && tier == 6 && roll == 3 {
		roll = g.Rand.Random(2) + 1 // no room for a POKéMON
	}

	g.message(Information, "You won "+g.prize(tier, roll)+"!")

	if k := (tier-1)*3 + roll; k >= 1 && k < len(tournamentPrizes) {
		tournamentPrizes[k](g)
	}
}

// tournamentPrizes give the prizes, by (tier−1)×3 + roll.
var tournamentPrizes = [...]func(g *Game){
	1: func(g *Game) {
		g.PokeBalls += 3
	},
	2: func(g *Game) {
		g.Potions += 2
	},
	3: func(g *Game) {
		g.Money += 2500
	},
	4: func(g *Game) {
		g.Potions += 3
	},
	5: func(g *Game) {
		g.PokeBalls += 5
	},
	6: func(g *Game) {
		g.Money += 4000
	},
	7: func(g *Game) {
		g.GreatBalls += 3
	},
	8: func(g *Game) {
		g.UltraBalls++
	},
	9: func(g *Game) {
		g.SuperPotions++
	},
	10: func(g *Game) {
		g.SuperPotions += 3
	},
	11: func(g *Game) {
		g.GreatBalls += 5
	},
	12: func(g *Game) {
		g.Money += 10000
	},
	13: func(g *Game) {
		g.UltraBalls += 3
	},
	14: func(g *Game) {
		g.Money += 15000
	},
	15: func(g *Game) {
		g.GreatBalls += 10
	},
	16: func(g *Game) {
		g.MasterBalls++
	},
	17: func(g *Game) {
		g.Money += 30000
	},
	18: (*Game).winPokemon,
}

// winPokemon is the Prime Cup's last prize: a random POKéMON at a random
// level.
func (g *Game) winPokemon() {
	s := g.Rand.Random(386) + 1
	l := uint8(g.Rand.Random(99) + 2) //nolint:gosec // 2–100
	g.AddPKMN(s, g.name(s), l)
	g.message(Information, fmt.Sprintf("You won a L.%d %s!", l, g.name(s)))
}

// prize is TournamentPrizes[tier][roll], the prize's name; its effect is
// hard-coded in handleTournament.
func (g *Game) prize(tier, roll int) string {
	i := (tier-1)*3 + roll - 1
	if i < 0 || i >= len(g.Data.Prizes) {
		return ""
	}

	return g.Data.Prizes[i]
}
