// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"fmt"

	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
)

// The battle. Both sides have one POKéMON with level × 10 HP and four fixed
// attacks that always hit; the enemy picks its attack at random.

const choosePrompt string = "Select a POKéMON from the Statistics Window. You'll be asked to confirm your choice."

// StartTrainerBattle starts a battle against a trainer's POKéMON. The
// caller sets BattleType afterwards.
func (g *Game) StartTrainerBattle(species, level int, trainer string) {
	g.enable(false, "Action1")

	g.EnemyLevel, g.EnemySpecies = uint8(level), species //nolint:gosec // a Byte in the original
	g.Main.Battle = true
	g.InTrainerBattle = true
	g.EnemyHP = level * 10
	g.showInfo()
	g.log(trainer + " wants to fight!")
	g.log(trainer + " sent out " + g.name(species) + "!")

	g.Main.Enemy = Shown{species, level, level * 10}
	g.Main.EnemyImage = species - 1
	g.Main.Attacks = false
	g.message(Information, choosePrompt)

	g.Main.EnemyMoves = g.Main.Enemy
}

// StartWildBattle starts a battle against a wild POKéMON. BattleType is
// left as the last battle set it: the callers that want 2 set it first.
func (g *Game) StartWildBattle(species, level int) {
	win := &g.Main
	win.Picker = false
	win.Battle = true
	g.InWildBattle = true
	g.showInfo()
	g.enable(false, "Action1")

	g.EnemyHP = level * 10
	g.EnemyLevel = uint8(level) //nolint:gosec // a Byte in the original
	g.log("Wild " + g.name(species) + " appeared!")

	win.Enemy = Shown{species, level, level * 10}
	win.EnemyImage = species - 1

	g.message(Information, choosePrompt)

	win.Attacks = false
	win.EnemyMoves = win.Enemy
	g.EnemySpecies = species
	win.Pokeball = g.InPokedex[species]
}

// ClickParty is a double click on party member k (0-based) in the
// Statistics window (TForm5.ListView1Click): its details, then, depending
// on the mode, choosing it to fight, evolving it or releasing it.
func (g *Game) ClickParty(k int) {
	party := g.Party()
	if k < 0 || k >= len(party) || !g.Stats.PartyEnabled {
		return
	}

	i := k + 1
	slt := g.member(i)
	s := int(slt.Species)
	name := g.name(s)
	toNext := 0

	if slt.Level < 100 {
		n := int(slt.Level) + 1
		toNext = n*n*n - int(slt.XP)
	}

	g.message(Information, fmt.Sprintf("# %03d %s, L.%d, XP:%d, XP to next level:%d", s, name, slt.Level, slt.XP, toNext))

	if g.InWildBattle && g.confirm("Choose "+name+"?") {
		g.choose(i)
	}

	inLevelRange := g.MinLevel <= slt.Level && slt.Level <= g.MaxLevel
	if g.InTrainerBattle && inLevelRange && g.confirm("Choose "+name+"?") {
		g.choose(i)
	}

	if g.EvolveMode {
		g.evolve(i)
	}

	if g.ReleaseMode {
		g.release(i, len(party))
	}

	g.menu["ReleasePKMN1"].Checked = false
	g.ReleaseMode = false
	g.enable(true, "Tournaments1", "Evolve1", "FightTrainer1", "Buy1", "FightWild1")
}

// choose sends party member i into the battle.
func (g *Game) choose(i int) {
	m := g.member(i)
	g.showItems()
	g.log(g.name(int(m.Species)) + ", I choose you!")

	g.Stats.PartyEnabled = false
	g.Main.Go = true
	g.Main.Attacks = true
	g.enable(false, "File1")

	g.OwnHP = int(m.Level) * 10
	g.Main.Own = Shown{int(m.Species), int(m.Level), g.OwnHP}
	g.OwnIndex = i
	g.Fights++
}

// Go is "Go!" (TForm4.Button1Click): the selected attack, then the
// enemy's if it's still standing, or "Run!".
func (g *Game) Go() {
	if !g.Main.Go {
		return
	}

	switch k := g.Main.AttackIndex; {
	case 0 <= k && k <= 3:
		g.ownAttack(k)

		if g.EnemyHP > 0 {
			g.oppAttack()
		}
	case k == 4:
		if g.InTrainerBattle && !g.InTournament {
			g.log("No! There's no running from a trainer battle!")
		}

		if g.InWildBattle {
			g.log("Got away safely!")

			g.BattleType = 0
			g.battleOver()
		}

		if g.InTournament {
			g.quitTournament()
		}
	}
}

// battleOver shows "End Fight" in place of "Go!" and the item buttons.
func (g *Game) battleOver() {
	g.hideItems()

	g.Main.EndFight = true
	g.Main.Go = false
}

// ownAttack is the fighter's attack k (TForm4.DoOwnAttack), and the
// victory when the enemy faints.
func (g *Game) ownAttack(k int) {
	crit := g.Rand.Random(10)+1 == 10
	roll := g.Rand.Random(20)
	m := g.member(g.OwnIndex)
	own := g.species(int(m.Species))
	enemy := g.species(g.EnemySpecies)
	mov := g.move(own.Moves[k])
	g.log(own.Name + " used " + mov.Name + "!")

	if crit {
		g.log("Critical hit!")
	}

	d := g.hit(roll, mov, int(m.Level), crit, enemy)
	g.logf("%s did %d points of damage to enemy %s.", own.Name, d, enemy.Name)

	g.EnemyHP -= d
	g.Main.EnemyColor = gaugeColor(g.EnemyHP, int(g.EnemyLevel))

	if g.EnemyHP < 0 {
		g.EnemyHP = 0
	}

	g.Main.Enemy.HP = g.EnemyHP
	if g.EnemyHP < 1 {
		g.victory()
	}
}

// oppAttack is the enemy's attack (TForm4.DoOppAttack), and the defeat
// when the fighter faints.
func (g *Game) oppAttack() {
	k := g.Rand.Random(4)
	crit := g.Rand.Random(10)+1 == 10
	roll := g.Rand.Random(20)
	slt := g.member(g.OwnIndex)
	own := g.species(int(slt.Species))
	enemy := g.species(g.EnemySpecies)
	mov := g.move(enemy.Moves[k])
	g.log("Enemy " + enemy.Name + " used " + mov.Name + "!")

	if crit {
		g.log("Critical hit!")
	}

	d := g.hit(roll, mov, int(g.EnemyLevel), crit, own)
	g.logf("Enemy %s did %d points of damage to %s.", enemy.Name, d, own.Name)

	g.OwnHP -= d
	g.Main.OwnColor = gaugeColor(g.OwnHP, int(slt.Level))

	if g.OwnHP < 0 {
		g.OwnHP = 0
	}

	g.Main.Own.HP = g.OwnHP
	if g.OwnHP < 1 {
		g.BattleType = 0
		g.battleOver()

		g.Fights++

		g.Losses++
		g.log(own.Name + " fainted!")
		g.logf("You lost %d¥!", g.Money-g.Money/2)

		g.Money /= 2
	}
}

// move returns move id, or a placeholder for one the tables don't have.
func (g *Game) move(id int) *gamedata.Move {
	if mov := g.Data.MoveByID(id); mov != nil {
		return mov
	}

	return &gamedata.Move{ID: id}
}

// hit logs the type messages of an attack on def and returns its damage.
func (g *Game) hit(r int, mov *gamedata.Move, level int, crit bool, def *gamedata.Species) int {
	t1 := g.Data.Effect(mov.Type, def.Types[0])
	t2 := g.Data.Effect(mov.Type, def.Types[1])
	damage := Damage(r, mov.Power, level, crit, t1, t2)

	if superEffective(t1, t2) {
		g.log("It's super effective!")
	}

	if notVeryEffective(t1, t2) {
		g.log("It's not very effective...")
	}

	if t1 == 0 || t2 == 0 {
		g.log("It doesn't affect " + def.Name + "!")

		damage = 0
	}

	return damage
}

// superEffective reports whether the multipliers t1 and t2 of an attack
// against a defender's two types show "It's super effective!". The
// original tests these cases only:
//
//	t1   t2
//	2    1 or 2
//	1    2
//
// so a 2 against a type that resists on the other shows nothing.
func superEffective(t1, t2 float64) bool {
	return (t1 == 2 && t2 >= 1) || (t2 == 2 && t1 == 1)
}

// notVeryEffective reports whether the multipliers show "It's not very
// effective...": the cases of the original are
//
//	t1   t2
//	0.5  1 or 0.5
//	1    0.5
func notVeryEffective(t1, t2 float64) bool {
	return (t1 == 0.5 && t2 == 1) || (t1 == 0.5 && t2 == 0.5) || (t1 == 1 && t2 == 0.5)
}

// Damage is an attack's damage, before immunity: (r − 10) + power/100 ×
// level, where r is Random(20), doubled by a critical hit, times the type
// multipliers t1 and t2, truncated, and at least 1.
//
// The original computes it in x87 Extended and Double. Integer arithmetic
// gives the same result: the exact value has denominator 400 at most, and
// the floating-point error is far too small to cross an integer.
func Damage(r, power, level int, crit bool, t1, t2 float64) int {
	c := 1
	if crit {
		c = 2
	}

	m := int(4 * t1 * t2) // 0, 1, 2, 4, 8 or 16
	d := ((r-10)*100 + power*level) * c * m / 400

	return max(d, 1)
}

// victory ends a won battle: XP, maybe levels, and prize money.
func (g *Game) victory() {
	g.battleOver()

	slt := g.member(g.OwnIndex)
	own := g.name(int(slt.Species))
	g.log("Enemy " + g.name(g.EnemySpecies) + " fainted!")

	e := uint32(g.EnemyLevel)
	gain := e*e*e - (e-1)*(e-1)*(e-1)
	g.logf("%s gained %d experience points!", own, gain)

	slt.XP += gain
	old := slt.Level
	// No levels mid-tournament or in the R/B/Y and Ru/Sa Elite Four:
	// they catch up at the next win, loss or tournament end.
	if g.BattleType < 11 || g.BattleType%10 > 7 {
		slt.Level = levelFor(slt.Level, slt.XP)
	}

	if slt.XP > maxXP {
		slt.Level, slt.XP = 100, maxXP
	}

	if slt.Level > old {
		g.logf("%s grew to level %d!", own, slt.Level)
	}

	r := g.Rand.Random(3) + 1
	if r == 3 {
		r = 4
	}

	money := r * 250
	gp := g.GymProgress

	if 3 < gp && gp < 8 {
		money *= 2
	}

	if 7 < gp && gp < 12 {
		money *= 4
	}

	if gp > 11 && !g.InPokedex[150] {
		money *= 6
	}

	if g.InPokedex[150] {
		money *= 8
	}

	g.logf("You got %d¥!", money)

	g.Money += uint32(money)
}

// quitTournament is fleeing a tournament, or quitting the game in one: a
// twentieth of the fighter's XP, and the levels with it.
func (g *Game) quitTournament() {
	if g.OwnIndex >= 1 { // the original reads out of its tables if no one fought yet
		m := g.member(g.OwnIndex)
		own := g.name(int(m.Species))
		lost := (m.XP + 10) / 20 // rounded, as the original's Int(XP/20 + 0.5) is
		g.logf("Quit the tournament! %s lost %d experience points!", own, lost)

		m.XP -= lost
		for l := uint32(m.Level); l*l*l > m.XP; l = uint32(m.Level) {
			m.Level--

			g.log(own + " lost a level!")
		}
	}

	g.BattleType = 0
	g.InTournament = false
	g.battleOver()
}

// EndFight is "End Fight" (TForm4.EndBattle): the counters, the rewards,
// and the next battle of an Elite Four or a tournament.
func (g *Game) EndFight() {
	if !g.Main.EndFight {
		return
	}

	g.hideItems()

	if g.BattleType == 0 { // lost or fled
		g.recomputeLevels()

		if g.InTournament {
			g.Losses++
		}

		g.MinLevel, g.MaxLevel = 2, 100
	}

	if g.BattleType == 1 {
		g.Catches++
	}

	if g.BattleType > 1 {
		g.Wins++
	}

	g.rewards()

	g.Main.EndFight = false
	g.Main.AttackIndex = -1
	g.InTrainerBattle, g.InTournament, g.InWildBattle = false, false, false
	g.enable(true, "File1", "Action1")

	g.Info.Visible = false
	g.Main.Battle = false
	g.Stats.PartyEnabled = true
	g.Main.EnemyColor, g.Main.OwnColor = Green, Green
	g.Main.Own = Shown{}
	g.Main.Attacks = false
	g.eliteFour()

	if 10 < g.BattleType && g.BattleType < 70 {
		g.handleTournament()
	}
}
