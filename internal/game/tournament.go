// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"fmt"
	"slices"
)

// Tournaments (notes/tournaments.md): eight random trainers in a row, an
// entry fee, a level range, and a prize for winning all eight.

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
	ok := false

	for _, m := range g.Party() {
		if t.lo <= m.Level && m.Level <= t.hi {
			ok = true
		}
	}

	if !ok {
		g.message(Warning, "None of your POKéMON meet the level requirements.")
	}

	if g.Money < t.fee && ok {
		g.message(Information, "You don't have enough money!")
	} else if ok {
		g.setUpTournament(tier)

		g.Money -= t.fee
	}
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

	for {
		ok := true

		for i := range g.TournamentTrainers {
			g.TournamentTrainers[i] = uint8(g.Rand.Random(64) + 1)
		}

		for i, a := range g.TournamentTrainers {
			for _, b := range g.TournamentTrainers[i+1:] {
				if a == b {
					ok = false
				}
			}
		}

		if ok {
			for i := range g.TournamentSpecies {
				s := g.Rand.Random(386) + 4 // so 1–3 never come, and 387–389 can
				g.TournamentSpecies[i] = s

				if slices.Contains(notInTournaments, s) {
					ok = false
				}
			}
		}

		if ok {
			lo, hi := int(g.MinLevel), int(g.MaxLevel)
			for i := range g.TournamentLevels {
				l := g.Rand.Random(hi-lo+5) + lo // hi comes 5 times as often
				g.TournamentLevels[i] = uint8(min(max(l, lo), hi))
			}
		}

		if ok && tier < 3 { // unevolved species only
			for _, s := range g.TournamentSpecies {
				if g.isEvolved(s) || slices.Contains(notInEasyTiers, s) {
					ok = false
				}
			}
		}

		// Species 387–389 don't exist, and a set the original accepts with
		// one crashes it (tournament-387). The port draws again then, and
		// only then, so the draws stay the original's.
		for _, s := range g.TournamentSpecies {
			if ok && s > len(g.Data.Species) {
				ok = false
			}
		}

		if ok {
			break
		}
	}

	g.BattleType = uint8(tier*10 + 1)
	g.StartTrainerBattle(g.TournamentSpecies[0], int(g.TournamentLevels[0]), g.trainerName(0))

	g.InTournament = true
}

// isEvolved reports whether some species evolves into s.
func (g *Game) isEvolved(s int) bool {
	for _, sp := range g.Data.Species {
		if sp.EvolveTo == s {
			return true
		}
	}

	return false
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
	r := int(g.BattleType % 10) // the round won
	if r < 8 {
		g.message(Information, fmt.Sprintf("Congratulations! You beat %s! Next up is trainer #%d, %s with a L.%d %s.",
			g.trainerName(r-1), r+1, g.trainerName(r), g.TournamentLevels[r], g.name(g.TournamentSpecies[r])))

		g.BattleType++
		g.StartTrainerBattle(g.TournamentSpecies[r], int(g.TournamentLevels[r]), g.trainerName(r))

		g.InTournament = true
	}

	if r != 8 {
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

	switch (tier-1)*3 + roll {
	case 1:
		g.PokeBalls += 3
	case 2:
		g.Potions += 2
	case 3:
		g.Money += 2500
	case 4:
		g.Potions += 3
	case 5:
		g.PokeBalls += 5
	case 6:
		g.Money += 4000
	case 7:
		g.GreatBalls += 3
	case 8:
		g.UltraBalls++
	case 9:
		g.SuperPotions++
	case 10:
		g.SuperPotions += 3
	case 11:
		g.GreatBalls += 5
	case 12:
		g.Money += 10000
	case 13:
		g.UltraBalls += 3
	case 14:
		g.Money += 15000
	case 15:
		g.GreatBalls += 10
	case 16:
		g.MasterBalls++
	case 17:
		g.Money += 30000
	case 18:
		s := g.Rand.Random(386) + 1
		l := uint8(g.Rand.Random(99) + 2)
		g.AddPKMN(s, g.name(s), l)
		g.message(Information, fmt.Sprintf("You won a L.%d %s!", l, g.name(s)))
	}
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
