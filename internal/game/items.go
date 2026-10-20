// SPDX-License-Identifier: AGPL-3.0-only

package game

// Items in battle: the Statistics window's six item buttons, shown once a
// fighter is chosen.

// The item buttons, in the Statistics window's order.
const (
	PokeBall int = iota
	GreatBall
	UltraBall
	MasterBall
	Potion
	SuperPotion
)

// ItemUsable reports whether item button k (0-based: PokeBall …
// SuperPotion) is shown and enabled.
func (g *Game) ItemUsable(k int) bool {
	return k >= 0 && k < len(g.Stats.ItemEnabled) && g.Stats.Items && g.Stats.ItemEnabled[k]
}

// UseItem is item button k (0-based: PokeBall … SuperPotion), if it is
// shown and enabled.
func (g *Game) UseItem(k int) {
	if !g.ItemUsable(k) {
		return
	}

	switch k {
	case PokeBall:
		g.throwBall("a POKéBall", &g.PokeBalls, 50)
	case GreatBall:
		g.throwBall("a Great Ball", &g.GreatBalls, 100)
	case UltraBall:
		g.throwBall("an Ultra Ball", &g.UltraBalls, 250)
	case MasterBall:
		g.log("You threw a Master Ball.")
		g.log("You caught " + g.name(g.EnemySpecies) + "!")
		g.AddPKMN(g.EnemySpecies, g.name(g.EnemySpecies), g.EnemyLevel)

		g.MasterBalls--
		g.BattleType = 1
		g.Main.EndFight = true
		// no EnableButtons: the buttons are hidden anyway
	case Potion:
		g.potion(&g.Potions, 50)
	case SuperPotion:
		g.potion(&g.SuperPotions, 100)
	}
}

// throwBall throws a ball that catches when Random(bound) ≥ the enemy's
// HP + 2; a miss gives the enemy a free attack.
func (g *Game) throwBall(ball string, count *uint32, bound int) {
	g.log("You threw " + ball + ".")

	*count--
	// Int(r − (EnemyHP + 0.5)) > 0 in Extended: r ≥ EnemyHP + 2.
	if r := g.Rand.Random(bound); r >= g.EnemyHP+2 {
		g.log("You caught " + g.name(g.EnemySpecies) + "!")
		g.AddPKMN(g.EnemySpecies, g.name(g.EnemySpecies), g.EnemyLevel)

		g.BattleType = 1
		g.Main.EndFight = true
	}

	if !g.Main.EndFight {
		g.log("Sorry, " + g.name(g.EnemySpecies) + " wasn't caught.")
		g.oppAttack()
	}

	g.enableItems()
}

// potion heals the fighter, which costs the turn; at full HP it does
// nothing.
func (g *Game) potion(count *uint32, heal int) {
	before := g.OwnHP
	full := int(g.member(g.OwnIndex).Level) * 10

	if g.OwnHP == full {
		g.log("It won't have any effect.")
	}

	if g.OwnHP < full {
		*count--
		g.OwnHP += heal
		g.enableItems()

		g.OwnHP = min(g.OwnHP, full)
		g.logf("%s recovered %d HP!", g.name(int(g.member(g.OwnIndex).Species)), g.OwnHP-before)
		g.oppAttack()
	}
}

// showItems shows the item buttons (TForm5.ShowButtons).
func (g *Game) showItems() {
	g.enableItems()

	g.Stats.Items = true
}

// hideItems hides them (TForm5.HideButtons).
func (g *Game) hideItems() {
	g.Stats.Items = false
}

// counter returns how many of item k (PokeBall … SuperPotion) the player has.
func (g *Game) counter(k int) *uint32 {
	return [6]*uint32{
		&g.PokeBalls,
		&g.GreatBalls,
		&g.UltraBalls,
		&g.MasterBalls,
		&g.Potions,
		&g.SuperPotions,
	}[k]
}

// enableItems enables the buttons of the items the player has, as the
// battle allows (TForm5.EnableButtons).
func (g *Game) enableItems() {
	enabled := &g.Stats.ItemEnabled
	for k := range enabled {
		enabled[k] = *g.counter(k) != 0
	}

	noBalls := func() {
		enabled[PokeBall], enabled[GreatBall], enabled[UltraBall], enabled[MasterBall] = false, false, false, false
	}
	if g.partyHas(150) && g.Main.EnemyImage == 149 {
		noBalls() // one Mewtwo at a time
	}

	if g.InTrainerBattle || (g.InWildBattle && len(g.Party()) == len(g.Owned)) {
		noBalls()
	}

	if g.InTournament {
		enabled[Potion], enabled[SuperPotion] = false, false
	}
}
