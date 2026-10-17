// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"strconv"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
)

// The side screens (notes/side-screens.md): POKéDEX, Type Comparisons,
// the Mart, the Game Corner, and the Statistics window's figures.

// DexWindow is the POKéDEX (Form8).
type DexWindow struct {
	Visible bool
	Index   int // ComboBox1.ItemIndex: the species shown − 1, or -1
}

// OpenDex is Actions → View POKéDEX (it keeps its selection).
func (g *Game) OpenDex() {
	g.Dex.Visible = true
	g.show(DexForm)
}

// ShowDex opens the POKéDEX on species k+1, as a new species does.
func (g *Game) ShowDex(k int) {
	g.Dex.Visible = true
	g.show(DexForm)
	g.SelectDex(k)
}

// SelectDex shows species k+1 (ComboBox1's OnChange).
func (g *Game) SelectDex(k int) {
	if k >= -1 && k < len(g.Data.Species) {
		g.Dex.Index = k
	}
}

// CloseDex is "Close POKéDEX".
func (g *Game) CloseDex() {
	g.Dex.Visible = false
}

// DexMovesShown reports whether the POKéDEX shows species s's attacks:
// once it is registered, or always with the Debug option.
func (g *Game) DexMovesShown(s int) bool {
	registered := s >= 1 && s < len(g.InDex) && g.InDex[s]

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

	return delphi.FloatToStr(float64(a)), delphi.FloatToStr(float64(b)), delphi.FloatToStr(float64(a * b)), true
}

// CloseTypes is "Close".
func (g *Game) CloseTypes() {
	g.Types.Visible = false
}

// MartWindow is the POKéMON Mart (Form9).
type MartWindow struct {
	Visible    bool
	Item       int // RadioGroup1.ItemIndex: MartPrices' index, or -1
	Quantity   int // SpinEdit1.Value
	Total      string
	BuyEnabled bool
}

// MartPrices are the Mart's items' prices: POKéBalls, Great Balls, Ultra
// Balls, Potions, Super Potions.
var MartPrices = [5]uint32{1000, 3000, 6000, 2000, 5000}

// martItems are the items MartPrices prices; the Mart has no Master Ball.
var martItems = [5]int{
	PokeBall,
	GreatBall,
	UltraBall,
	Potion,
	SuperPotion,
}

// OpenMart is Actions → Buy Items.
func (g *Game) OpenMart() {
	g.Mart = MartWindow{
		Visible: true,
		Item:    -1,
		Total:   "0¥",
	}
	g.show(MartForm)
}

// martTotal is the price of the order, wrapping at 2³² as the original's
// Cardinal arithmetic does.
func (g *Game) martTotal() uint32 {
	if g.Mart.Item < 0 {
		return 0
	}

	return uint32(g.Mart.Quantity) * MartPrices[g.Mart.Item] //nolint:gosec // the original's Cardinal arithmetic
}

// SelectMartItem is a click on item k (TForm9.RadioGroup1Click).
func (g *Game) SelectMartItem(k int) {
	if k < 0 || k >= len(MartPrices) {
		return
	}

	g.Mart.Item = k
	g.updateMart()
}

// SetMartQuantity is typing a quantity (TForm9.SpinEdit1Change).
func (g *Game) SetMartQuantity(n int) {
	g.Mart.Quantity = n
	g.updateMart()
}

func (g *Game) updateMart() {
	t := g.martTotal()
	g.Mart.Total = strconv.FormatUint(uint64(t), 10) + "¥"
	g.Mart.BuyEnabled = t != 0 && t <= g.Money
}

// MartDescription is the selected item's two lines of description.
func (g *Game) MartDescription() (line1, line2 string) {
	switch g.Mart.Item {
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
	if !g.Mart.Visible || !g.Mart.BuyEnabled || g.Mart.Item < 0 {
		return
	}

	t := g.martTotal()
	g.Money -= t
	*g.counter(martItems[g.Mart.Item]) += t / MartPrices[g.Mart.Item]
	g.Mart.Visible = false
}

// LeaveMart is the Mart's "Leave".
func (g *Game) LeaveMart() {
	g.Mart.Visible = false
}

// CornerWindow is the Game Corner (FormB).
type CornerWindow struct {
	Visible bool
	Prize   int // RadioGroup1.ItemIndex: CornerPrizes' index, or -1
	Number  int // RadioGroup2.ItemIndex: the number bet on, or -1
	Drawn   int // RadioGroup3.ItemIndex: the number drawn, or -1
	Bet     int // SpinEdit1.Value
	MaxBet  int // SpinEdit1.MaxValue: the money when the window opened
}

// CornerPrizes are what the Game Corner pays in, after money: each item's
// price there and its name.
var CornerPrizes = [6]struct {
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

// OpenCorner is Actions → Game Corner.
func (g *Game) OpenCorner() {
	g.Corner.Visible = true
	g.Corner.Bet = 0
	g.Corner.Prize = -1
	g.Corner.MaxBet = int(g.Money)
	g.show(CornerForm)
}

// SelectPrize selects what to play for (0 money, then CornerPrizes).
func (g *Game) SelectPrize(k int) {
	if k >= 0 && k <= len(CornerPrizes) {
		g.Corner.Prize = k
	}
}

// SelectNumber selects the number to bet on.
func (g *Game) SelectNumber(n int) {
	if n >= 0 && n <= 9 {
		g.Corner.Number = n
	}
}

// SetBet is typing the bet.
func (g *Game) SetBet(n int) {
	g.Corner.Bet = n
}

// Draw is "Draw number!" (TFormB.Button1Click): one number in ten wins ten
// times the bet, in money or in items.
func (g *Game) Draw() {
	gcw := &g.Corner

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
	if g.Corner.Prize == 0 { // "Money (Pays out 10:1¥)"
		won = bet * 10

		return won, strconv.FormatUint(uint64(won), 10) + "¥!"
	}

	p := CornerPrizes[g.Corner.Prize-1]
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
	*g.counter(g.Corner.Prize - 1) += n

	return won, msg
}

// CloseCorner is the Game Corner's "Close".
func (g *Game) CloseCorner() {
	g.Corner.Visible = false
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

	return delphi.FloatToStr(x.Quo(delphi.NewExtended(100)).Float64()), true
}

// DexCount is the Statistics window's "In POKéDEX" figure, which counts
// species 1–251 only.
func (g *Game) DexCount() int {
	n := 0

	for s := 1; s <= 251; s++ {
		if g.InDex[s] {
			n++
		}
	}

	return n
}
