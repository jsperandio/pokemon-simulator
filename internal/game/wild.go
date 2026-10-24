// SPDX-License-Identifier: AGPL-3.0-only

package game

import "slices"

// Wild POKéMON: a random encounter that depends on the badges, or, with a
// Mewtwo in the party, any POKéMON at any level.

// Species that never appear in the wild: Vaporeon, Jolteon, Flareon,
// Mewtwo, Mew, Espeon, Umbreon, Celebi, Jirachi, Deoxys.
var neverWild = []int{134, 135, 136, 150, 151, 196, 197, 251, 385, 386}

// Branched evolutions, which EvolveTo doesn't list: Vileplume, Poliwrath,
// Slowbro, Bellossom, Politoed, Slowking, Silcoon, Cascoon, Shedinja,
// Huntail, Gorebyss.
var branchedEvolutions = []int{45, 62, 80, 182, 186, 199, 266, 268, 292, 367, 368}

// heldBack are the species that don't appear in the wild up to a badge
// count.
var heldBack = []struct {
	badges  uint8
	species []int
}{
	{4, []int{106, 107, 237}},  // the Hitmons
	{8, []int{143}},            // Snorlax
	{10, []int{243, 244, 245}}, // Raikou, Entei, Suicune
	{12, []int{144, 145, 146}}, // Articuno, Zapdos, Moltres
	{13, []int{249, 250}},      // Lugia, Ho-Oh
	{19, []int{382, 383, 384}}, // Kyogre, Groudon, Rayquaza
	{21, []int{380, 381}},      // Latias, Latios
}

// wildAllowed reports whether species s can be met at the current badge
// count.
func (g *Game) wildAllowed(s int) bool {
	gp := g.GymProgress
	tooEarly := gp <= 3 && (g.isEvolved(s) || slices.Contains(branchedEvolutions, s))

	if slices.Contains(neverWild, s) || tooEarly {
		return false
	}

	for _, h := range heldBack {
		if gp <= h.badges && slices.Contains(h.species, s) {
			return false
		}
	}

	return true
}

// FindPKMN is Actions → Fight a wild POKéMON → Search for a POKéMON
// (TForm4.FindPKMN1Click).
func (g *Game) FindPKMN() {
	if g.partyHas(150) {
		g.enable(false, "Action1")

		g.Main.Picker = true
		g.Main.PickSpecies = -1
		g.Main.PickLevel = 100
		g.message(Information, "Using Mewtwo's psychic powers, you can fight any POKéMON you want to, and at any level. Please select which POKéMON to fight against, or click Random to face a random POKéMON.")

		return
	}

	species := g.Rand.Random(386) + 1
	for !g.wildAllowed(species) {
		species = g.Rand.Random(386) + 1
	}

	// Each rule that applies draws a level; the last one counts.
	level := g.Rand.Random(30) + 71
	if !g.InPokedex[150] {
		level = g.Rand.Random(30) + 61
	}

	if !g.BeatAsh {
		level = g.Rand.Random(20) + 61
	}

	gp := g.GymProgress
	for _, rule := range wildLevels {
		if gp < rule.below {
			level = g.Rand.Random(rule.n) + rule.base
		}
	}

	g.BattleType = 2
	g.StartWildBattle(species, level, g.rollShiny())
}

// wildLevels are the level rules by badge count, in the order they draw.
var wildLevels = []struct {
	below   uint8
	n, base int
}{
	{15, 30, 40}, // 40–69
	{13, 20, 40}, // 40–59
	{11, 20, 30}, // 30–49
	{9, 10, 30},  // 30–39
	{7, 10, 20},  // 20–29
	{5, 19, 2},   // 2–20
	{3, 9, 2},    // 2–10
}

// PickSpecies selects species k+1 in the Mewtwo-power picker (ComboBox1).
func (g *Game) PickSpecies(k int) {
	if k >= -1 && k < len(g.Data.Species) {
		g.Main.PickSpecies = k
	}
}

// PickLevel sets the picker's level (SpinEdit1).
func (g *Game) PickLevel(level int) {
	g.Main.PickLevel = level
}

// PickGo is the picker's "Go!" (TForm4.Button2Click). BattleType stays as
// the last battle left it. The original fights species 0, a nameless
// POKéMON, when none is selected; the port waits for a selection.
func (g *Game) PickGo() {
	if !g.Main.Picker || g.Main.PickSpecies < 0 {
		return
	}

	g.StartWildBattle(g.Main.PickSpecies+1, g.Main.PickLevel, false)
}

// PickCancel is the picker's "Cancel" (TForm4.Button3Click).
func (g *Game) PickCancel() {
	if !g.Main.Picker {
		return
	}

	g.enable(true, "Action1")

	g.Main.Picker = false
}

// PickRandom is the picker's "Random" (TForm4.Button4Click): any species,
// legendaries included, at level 90–100.
func (g *Game) PickRandom() {
	if !g.Main.Picker {
		return
	}

	g.enable(true, "Action1")

	s := g.Rand.Random(386) + 1
	l := g.Rand.Random(11) + 90
	g.BattleType = 2
	g.StartWildBattle(s, l, false)
}
