// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"fmt"

	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// The party: new members, evolution and release (notes/lifecycle.md,
// notes/evolution.md).

// AddPKMN adds a POKéMON at the given level, with level³ XP, and registers
// its species (TForm5.AddPKMN). The original doesn't check for a full
// party and writes past its arrays; the port drops the POKéMON instead.
func (g *Game) AddPKMN(species int, name string, level uint8) {
	if species == 150 {
		g.message(Information, "NOTE: You can only catch one Mewtwo at a time. You must release the one(s) you already have if you want to catch another.")
	}

	g.hideItems()

	g.Main.Go = false
	i := 1 // the starter takes the list's placeholder item

	if !g.Main.Starters {
		i = len(g.Party()) + 1
	}

	if !g.InDex[species] {
		g.log(name + " has been added to the POKéDEX!")
		g.ShowDex(species - 1)

		g.InDex[species] = true
	}

	if i > len(g.Owned) {
		return
	}

	l := uint32(level)
	*g.member(i) = save.Slot{
		Species: int32(species), //nolint:gosec // a species number
		Level:   level,
		XP:      l * l * l,
	}
}

// toggleEvolveMode is Actions → Evolve a POKéMON: the next double click
// on the party evolves.
func (g *Game) toggleEvolveMode() {
	modes := []string{
		"ReleasePKMN1",
		"Tournaments1",
		"FightTrainer1",
		"Buy1",
		"FightWild1",
	}

	if !g.EvolveMode {
		g.menu["Evolve1"].Checked = true
		g.EvolveMode = true
		g.enable(false, modes...)
		g.message(Information, "Doubleclick a POKéMON in the Statistics window to evolve it. Once you have doubleclicked on a POKéMON, Evolution Mode will be disabled.")

		return
	}

	g.menu["Evolve1"].Checked = false
	g.EvolveMode = false
	g.enable(true, modes...)
	g.message(Information, "Evolution Mode cancelled.")
}

// toggleReleaseMode is Actions → Release a POKéMON: the next double click
// on the party releases.
func (g *Game) toggleReleaseMode() {
	if len(g.Party()) < 2 {
		g.message(Warning, "You can't release your last POKéMON!")

		return
	}

	modes := []string{
		"Evolve1",
		"Tournaments1",
		"FightTrainer1",
		"Buy1",
		"FightWild1",
	}

	if !g.ReleaseMode {
		g.menu["ReleasePKMN1"].Checked = true
		g.ReleaseMode = true
		g.enable(false, modes...)
		g.message(Information, "Doubleclick a POKéMON in the Statistics window to release it. Once you have doubleclicked on a POKéMON, Release Mode will be disabled.")

		return
	}

	g.menu["ReleasePKMN1"].Checked = false
	g.ReleaseMode = false
	g.enable(true, modes...)
	g.message(Information, "Release Mode deactivated. Statistics window will now function normally.")
}

// branches are the evolutions the player chooses, by species: the name the
// question uses (sic) and the targets.
var branches = map[int]struct {
	from    string
	targets []string
}{
	236: {"Tyrouge", []string{"Hitmonlee", "Hitmonchan", "Hitmontop"}},
	44:  {"Gloom", []string{"Vileplume", "Bellossom"}},
	61:  {"Poliwhirl", []string{"Poliwrath", "Politoed"}},
	79:  {"Slowpoke", []string{"Slowbro", "Slowking"}},
	133: {"Eevee", []string{"Vaporeon", "Jolteon", "Flareon", "Espeon", "Umbreon"}},
	265: {"Wurmple", []string{"Silcoon", "Cascoon"}},
	366: {"Clamperl", []string{"Huntail", "Gorebyss"}},
}

// evolve is Evolution Mode's double click on party member i.
func (g *Game) evolve(i int) {
	slt := g.member(i)
	species := int(slt.Species)
	spc := g.species(species)
	name := spc.Name
	level := uint8(spc.EvolveLevel) //nolint:gosec // a Byte in the original
	b, branched := branches[species]

	switch {
	case level == 0:
		g.message(Information, name+" can't evolve.")
	case slt.Level < level:
		g.message(Information, fmt.Sprintf("Your %s needs to grow %d more level(s) before it can evolve.", name, level-slt.Level))
	case branched:
		g.message(Information, "Please select what you would like to evolve your "+b.from+" to.")

		g.Main.Choices = append([]string{"Don't evolve"}, b.targets...)
		g.Main.Choice = 0
		g.EvolveIndex = i
		g.Stats.PartyEnabled = false
	case species == 290:
		g.evolveNincada(slt, spc)
	default:
		g.evolveTo(slt, spc)
	}

	g.menu["Evolve1"].Checked = false
	g.EvolveMode = false
	g.enable(true, "Tournaments1", "ReleasePKMN1", "FightTrainer1", "Buy1", "FightWild1")
}

// evolveNincada asks to evolve Nincada into Ninjask, which gives a Shedinja
// too 3 of 4 times. After a No it asks as for any other species.
func (g *Game) evolveNincada(slt *save.Slot, spc *gamedata.Species) {
	ninjask := g.species(290).EvolveTo
	if !g.confirm("Your " + g.name(290) + " can evolve into " + g.name(ninjask) + ". Do you want to evolve it? NOTE: Nincada will evolve into Ninjask AND Shedinja every 3 of 4 times.") {
		g.evolveTo(slt, spc)

		return
	}

	slt.Species = 291
	g.InDex[291] = true
	g.message(Information, g.name(290)+" evolved into "+g.name(291)+"!")

	if g.Rand.Random(4) == 0 {
		g.message(Information, "Shedinja didn't appear.")
	} else {
		g.message(Information, "Nincada also evolved into Shedinja!")
		g.AddPKMN(292, "Shedinja", slt.Level)
	}
}

// evolveTo asks to evolve m into what its species evolves to.
func (g *Game) evolveTo(m *save.Slot, spc *gamedata.Species) {
	t := spc.EvolveTo
	if g.confirm("Your " + spc.Name + " can evolve into " + g.name(t) + ". Do you want to evolve it?") {
		m.Species = int32(t) //nolint:gosec // a species number
		g.InDex[t] = true
		g.message(Information, spc.Name+" evolved into "+g.name(t)+"!")
	}
}

// ChooseEvolution selects item k of the evolution choice.
func (g *Game) ChooseEvolution(k int) {
	if k >= 0 && k < len(g.Main.Choices) {
		g.Main.Choice = k
	}
}

// Evolve is the evolution choice's "Evolve" (TForm4.EvolveButtonClick).
func (g *Game) Evolve() {
	if len(g.Main.Choices) == 0 {
		return
	}

	g.Stats.PartyEnabled = true
	if g.Main.Choice > 0 {
		from := "" // Wurmple and Clamperl have none: " evolved into Silcoon!"

		switch g.Main.Choices[1] {
		case "Vileplume":
			from = "Gloom"
		case "Poliwrath":
			from = "Poliwhirl"
		case "Slowbro":
			from = "Slowpoke"
		case "Vaporeon":
			from = "Eevee"
		case "Hitmonlee":
			from = "Tyrouge"
		}

		t := 0 // the last species of that name

		for s := 1; s <= len(g.Data.Species); s++ {
			if g.name(s) == g.Main.Choices[g.Main.Choice] {
				t = s
			}
		}

		g.member(g.EvolveIndex).Species = int32(t)
		g.InDex[t] = true
		g.message(Information, from+" evolved into "+g.name(t)+"!")
	}

	if g.Main.Choice == 0 {
		g.message(Information, "Evolution cancelled.")
	}

	g.Main.Choices = nil
}

// release is Release Mode's double click on party member i, of n.
func (g *Game) release(i, n int) {
	name := g.name(int(g.member(i).Species))
	if !g.confirm("Once released, " + name + " will not come back. Release " + name + "?") {
		return
	}

	k := i - 1
	g.Owned[k] = save.Slot{}
	// Compact the party: the members after it, and the empty slot after
	// them, move up one.
	if k+1 < len(g.Owned) && g.Owned[k+1].Species > 0 {
		for j := k + 1; j <= n; j++ {
			if j < len(g.Owned) {
				g.Owned[j-1] = g.Owned[j]
			} else {
				g.Owned[j-1] = save.Slot{}
			}
		}
	}

	g.message(Information, name+" was released. Bye, "+name+"!")
}
