// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"strconv"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
)

// The side screens: POKéDEX, Type Comparisons, the Mart, the Game Corner,
// and the Statistics window's figures.

// PokedexWindow is the POKéDEX (Form8).
type PokedexWindow struct {
	Visible  bool
	Index    int  // ComboBox1.ItemIndex: the species shown − 1, or -1
	Selected int  // the species selected, each one animated as it shows
	Shiny    bool // the "Shiny" check box, which the original doesn't have

	// Whether UpdateDisplay showed the species' attacks rather than "???".
	// It runs only when the species is selected, so registering the
	// species shown, by an evolution, changes nothing until then.
	MovesShown bool
}

// OpenPokedex is Actions → View POKéDEX (it keeps its selection).
func (g *Game) OpenPokedex() {
	g.Pokedex.Visible = true
	g.show(PokedexForm)
}

// ShowPokedex opens the POKéDEX on species k+1, as a new species does.
func (g *Game) ShowPokedex(k int) {
	g.Pokedex.Visible = true
	g.show(PokedexForm)
	g.SelectPokedex(k)
}

// SelectPokedex shows species k+1 (ComboBox1's OnChange, UpdateDisplay).
func (g *Game) SelectPokedex(k int) {
	if k >= -1 && k < len(g.Data.Species) {
		g.Pokedex.Index = k
		g.Pokedex.MovesShown = g.pokedexMovesShown(k + 1)
	}

	if k >= 0 && k < len(g.Data.Species) {
		g.Pokedex.Selected++
	}
}

// PokedexShiny is the POKéDEX's "Shiny" check box: the species in its
// shiny colours, or in its normal ones. A change animates the species shown
// again, as selecting it does.
func (g *Game) PokedexShiny(on bool) {
	if on == g.Pokedex.Shiny {
		return
	}

	g.Pokedex.Shiny = on
	if g.Pokedex.Index >= 0 {
		g.Pokedex.Selected++
	}
}

// ClosePokedex is "Close POKéDEX".
func (g *Game) ClosePokedex() {
	g.Pokedex.Visible = false
}

// pokedexMovesShown reports whether UpdateDisplay shows species s's
// attacks: once it is registered, or always with the Debug option.
func (g *Game) pokedexMovesShown(s int) bool {
	registered := s >= 1 && s < len(g.InPokedex) && g.InPokedex[s]

	return registered || g.menu["AlwaysshowattacksinPOKDEX1"].Checked
}

// TypesWindow is Type Comparisons (FormA).
type TypesWindow struct {
	Visible bool
	Species int // ComboBox1.ItemIndex: species − 1, or -1
	Attack  int // ComboBox2.ItemIndex: move − 1, or -1
}

// OpenTypes is Actions → Type Comparisons (both lists keep their
// selection).
func (g *Game) OpenTypes() {
	g.Types.Visible = true
	g.show(TypesForm)
}

// SelectTypes selects a species and an attack (-1 for none) to compare.
func (g *Game) SelectTypes(species, attack int) {
	if species >= -1 && species < len(g.Data.Species) {
		g.Types.Species = species
	}

	if attack >= -1 && attack < len(g.Data.Moves) {
		g.Types.Attack = attack
	}
}

// TypeMultipliers returns the selected attack's multipliers against the
// selected species' first type, second type (1 for none) and both, as the
// window prints them; ok is false until both lists have a selection.
func (g *Game) TypeMultipliers() (first, second, total string, ok bool) {
	if g.Types.Species < 0 || g.Types.Attack < 0 {
		return "", "", "", false
	}

	spc, mov := g.species(g.Types.Species+1), g.move(g.Types.Attack+1)
	a := float32(g.Data.Effect(mov.Type, spc.Types[0]))
	b := float32(g.Data.Effect(mov.Type, spc.Types[1]))

	return floatToStr(float64(a)), floatToStr(float64(b)), floatToStr(float64(a * b)), true
}

// floatToStr prints x as the original's FloatToStr does, with "." for the
// decimal separator where the original followed the Windows locale. Both
// write 15 significant digits, in fixed notation for 0 and from 0.0001 up
// to 10¹⁵, where every number the game prints falls: the type multipliers
// (0 to 4) and the win percentage (0, or 0.01 and up, below 10¹² even when
// Wins + Losses wraps). Elsewhere they differ ("1E-05" for "0.00001").
func floatToStr(x float64) string {
	return strconv.FormatFloat(x, 'G', 15, 64)
}

// CloseTypes is "Close".
func (g *Game) CloseTypes() {
	g.Types.Visible = false
}

// PokemartWindow is the POKéMON Mart (Form9).
type PokemartWindow struct {
	Visible    bool
	Item       int    // RadioGroup1.ItemIndex: PokemartPrices' index, or -1
	Quantity   int    // SpinEdit1.Value
	Total      string // Edit1: the price, "" until one is written
	BuyEnabled bool
}

// PokemartPrices are the Mart's items' prices: POKéBalls, Great Balls, Ultra
// Balls, Potions, Super Potions.
var PokemartPrices = [5]uint32{1000, 3000, 6000, 2000, 5000}

// pokemartItems are the items PokemartPrices prices; the Mart has no Master Ball.
var pokemartItems = [5]int{
	PokeBall,
	GreatBall,
	UltraBall,
	Potion,
	SuperPotion,
}

// OpenPokemart is Actions → Buy Items (Form9.Show). FormShow runs only
// when the Mart wasn't showing: an open one just comes to the front, as
// it is.
func (g *Game) OpenPokemart() {
	if g.Pokemart.Visible {
		g.show(PokemartForm)

		return
	}

	// Only RadioGroup1Click and SpinEdit1Change write the price, and
	// FormShow's SpinEdit1.Value := 0 runs SpinEdit1Change only when the
	// quantity wasn't "0". So the box stays empty until a price is
	// written, and shows "0¥" from then on.
	total := g.Pokemart.Total
	if total != "" {
		total = "0¥"
	}

	g.Pokemart = PokemartWindow{
		Visible: true,
		Item:    -1,
		Total:   total,
	}
	g.show(PokemartForm)
}

// pokemartTotal is the price of the order, wrapping at 2³² as the original's
// Cardinal arithmetic does.
func (g *Game) pokemartTotal() uint32 {
	if g.Pokemart.Item < 0 {
		return 0
	}

	return uint32(g.Pokemart.Quantity) * PokemartPrices[g.Pokemart.Item] //nolint:gosec // the original's Cardinal arithmetic
}

// SelectPokemartItem is a click on item k (TForm9.RadioGroup1Click).
func (g *Game) SelectPokemartItem(k int) {
	if k < 0 || k >= len(PokemartPrices) {
		return
	}

	g.Pokemart.Item = k
	g.updatePokemart()
}

// SetPokemartQuantity is typing a quantity (TForm9.SpinEdit1Change).
func (g *Game) SetPokemartQuantity(n int) {
	g.Pokemart.Quantity = n
	g.updatePokemart()
}

func (g *Game) updatePokemart() {
	t := g.pokemartTotal()
	g.Pokemart.Total = strconv.FormatUint(uint64(t), 10) + "¥"
	g.Pokemart.BuyEnabled = t != 0 && t <= g.Money
}

// PokemartDescription is the selected item's two lines of description.
func (g *Game) PokemartDescription() (line1, line2 string) {
	switch g.Pokemart.Item {
	case 0:
		return "A standard POKéBALL.", "Used for catching POKéMON."
	case 1:
		return "This Ball has good chances", "of capturing a POKéMON."
	case 2:
		return "Has a very high chance of", "capturing a POKéMON."
	case 3:
		return "Restores 50HP in battle.", ""
	case 4:
		return "Restores 100HP in battle.", ""
	}

	return "", ""
}

// Buy is the Mart's "Buy" (TForm9.Button1Click).
func (g *Game) Buy() {
	if !g.Pokemart.Visible || !g.Pokemart.BuyEnabled || g.Pokemart.Item < 0 {
		return
	}

	t := g.pokemartTotal()
	g.Money -= t
	*g.counter(pokemartItems[g.Pokemart.Item]) += t / PokemartPrices[g.Pokemart.Item]
	g.Pokemart.Visible = false
}

// LeavePokemart is the Mart's "Leave".
func (g *Game) LeavePokemart() {
	g.Pokemart.Visible = false
}

// GameCornerWindow is the Game Corner (FormB).
type GameCornerWindow struct {
	Visible bool
	Prize   int // RadioGroup1.ItemIndex: GameCornerPrizes' index, or -1
	Number  int // RadioGroup2.ItemIndex: the number bet on, or -1
	Drawn   int // RadioGroup3.ItemIndex: the number drawn, or -1
	Bet     int // SpinEdit1.Value
	MaxBet  int // SpinEdit1.MaxValue: the money when the window opened
}

// GameCornerPrizes are what the Game Corner pays in, after money: each item's
// price there and its name.
var GameCornerPrizes = [6]struct {
	Price uint32
	Name  string
}{
	{100, "POKéBall"},
	{300, "Great Ball"},
	{600, "Ultra Ball"},
	{2500, "Master Ball"},
	{200, "Potion"},
	{500, "Super Potion"},
}

// OpenGameCorner is Actions → Game Corner (FormB.Show). FormShow runs
// only when the Game Corner wasn't showing: an open one just comes to the
// front, as it is.
func (g *Game) OpenGameCorner() {
	if g.GameCorner.Visible {
		g.show(GameCornerForm)

		return
	}

	g.GameCorner.Visible = true
	g.GameCorner.Bet = 0
	g.GameCorner.Prize = -1
	g.GameCorner.MaxBet = int(g.Money)
	g.show(GameCornerForm)
}

// SelectPrize selects what to play for (0 money, then GameCornerPrizes).
func (g *Game) SelectPrize(k int) {
	if k >= 0 && k <= len(GameCornerPrizes) {
		g.GameCorner.Prize = k
	}
}

// SelectNumber selects the number to bet on.
func (g *Game) SelectNumber(n int) {
	if n >= 0 && n <= 9 {
		g.GameCorner.Number = n
	}
}

// SetBet is typing the bet.
func (g *Game) SetBet(n int) {
	g.GameCorner.Bet = n
}

// Draw is "Draw number!" (TFormB.Button1Click): one number in ten wins ten
// times the bet, in money or in items.
func (g *Game) Draw() {
	gcw := &g.GameCorner

	if !gcw.Visible {
		return
	}

	if gcw.Bet < 0 || gcw.Bet > int(g.Money) {
		return // silently
	}

	bet := uint32(gcw.Bet) //nolint:gosec // between 0 and Money, checked above

	if gcw.Prize == -1 {
		g.message(Warning, "Please select something to play for.")

		return
	}

	if gcw.Number == -1 {
		g.message(Warning, "Please select a number to bet on.")

		return
	}

	g.Money -= bet

	var draws []int

	if g.OnDraws != nil {
		draws = make([]int, 0, 10000)
	}

	for range 10000 { // the animation: the last draw counts
		gcw.Drawn = g.Rand.Random(10)
		if draws != nil {
			draws = append(draws, gcw.Drawn)
		}
	}

	if g.OnDraws != nil {
		g.OnDraws(draws)
	}

	if gcw.Number != gcw.Drawn {
		return
	}

	g.message(Information, "Congratulations! You won! And your prize is...")

	won, msg := g.payOut(bet)
	g.Money += won
	g.message(Information, msg)
}

// payOut pays a winning bet's prize: the items go to the player, and it
// returns the money that goes with them and the message that announces
// the prize.
func (g *Game) payOut(bet uint32) (won uint32, msg string) {
	if g.GameCorner.Prize == 0 { // "Money (Pays out 10:1¥)"
		won = bet * 10

		return won, strconv.FormatUint(uint64(won), 10) + "¥!"
	}

	p := GameCornerPrizes[g.GameCorner.Prize-1]
	n := bet / p.Price
	won = bet % p.Price * 10
	msg = strconv.FormatUint(uint64(n), 10) + " " + p.Name

	if n > 1 {
		msg += "s"
	}

	if won != 0 {
		msg += " and " + strconv.FormatUint(uint64(won), 10) + "¥"
	}

	msg += "!"
	*g.counter(g.GameCorner.Prize - 1) += n

	return won, msg
}

// CloseGameCorner is the Game Corner's "Close".
func (g *Game) CloseGameCorner() {
	g.GameCorner.Visible = false
}

// WinPercent is the Statistics window's win percentage, truncated to two
// decimals: Int(Wins / (Wins + Losses) × 10000) / 100, computed in x87
// Extended, which prints 77.99 where float64 would give 78 (39 wins, 11
// losses). ok is false with no wins or losses, when the window keeps what
// it showed.
func (g *Game) WinPercent() (percent string, ok bool) {
	n := g.Wins + g.Losses
	if n == 0 {
		return "", false
	}

	w := delphi.NewExtended(float64(g.Wins))
	x := w.Quo(delphi.NewExtended(float64(n))).Mul(delphi.NewExtended(10000)).Int()

	return floatToStr(x.Quo(delphi.NewExtended(100)).Float64()), true
}

// PokedexCount is the Statistics window's "In POKéDEX" figure, which counts
// species 1–251 only.
func (g *Game) PokedexCount() int {
	n := 0

	for s := 1; s <= 251; s++ {
		if g.InPokedex[s] {
			n++
		}
	}

	return n
}
