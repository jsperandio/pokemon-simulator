// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"slices"
	"strconv"
	"time"

	"github.com/jsperandio/pokemon-simulator/internal/save"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// statsForm is the Statistics window (TForm5): the counters, the items
// and the party.
type statsForm struct {
	a      *App
	w      *window
	party  *listView
	items  [6]*button
	counts map[string]*label
	items0 []listItem // the party as last shown
}

func newStatsForm(a *App) *statsForm {
	w := newWindow(a.desk, &layout.Form5)
	f := &statsForm{a: a, w: w, counts: map[string]*label{}}
	f.party = get[*listView](w, "ListView1")

	for i := range f.items {
		k := i
		f.items[i] = get[*button](w, "Button"+itoa(i+1))
		f.items[i].onClick = func() {
			a.call(func() {
				a.g.UseItem(k)
			})
		}
	}

	for _, n := range []string{
		"PlayerNameLbl",
		"MoneyLbl",
		"FightsLbl",
		"WinsLbl",
		"CapturesLbl",
		"LossesLbl",
		"Percent",
		"OwnedPKMNLbl",
		"PokeBallsLbl",
		"GreatBallsLbl",
		"UltraBallsLbl",
		"MasterBallsLbl",
		"PotionsLbl",
		"SuperPotionsLbl",
	} {
		f.counts[n] = get[*label](w, n)
	}

	f.party.onDblClick = func() {
		if k := f.party.selected; k >= 0 {
			a.call(func() {
				a.g.ClickParty(k)
			})
		}
	}
	// FormClick: a click on the window gives the focus to the game window.
	w.onClick = func() {
		if a.main.w.visible {
			a.desk.activate(a.main.w)
		}
	}
	a.desk.add(w)

	return f
}

func (f *statsForm) sync() {
	g := f.a.g
	f.counts["PlayerNameLbl"].setCaption(g.Stats.Name)

	party := g.Party()
	if len(party) != len(f.items0) || !slices.EqualFunc(party, f.items0, func(m save.Slot, it listItem) bool {
		return it.image == int(m.Species)-1
	}) {
		items := make([]listItem, len(party))
		for i, m := range party {
			items[i] = listItem{f.a.speciesName(int(m.Species)), int(m.Species) - 1}
		}

		f.items0 = items
		f.party.setItems(items)
	}

	if f.party.enabled != g.Stats.PartyEnabled {
		f.party.setEnabled(g.Stats.PartyEnabled)

		f.party.color = colWindow
		if !g.Stats.PartyEnabled {
			f.party.color = colFace
		}
	}

	for i, b := range f.items {
		b.setVisible(g.Stats.Items)
		b.setEnabled(g.Stats.ItemEnabled[i])
	}
}

// freshen is TForm5.Freshen, which the original runs on every tick while
// the window shows: the counters, and the focus back to the game window
// when this one has it without a control to hold it.
func (f *statsForm) freshen() {
	if !f.w.visible {
		return
	}

	g := f.a.g
	for n, v := range map[string]uint32{
		"PokeBallsLbl":    g.PokeBalls,
		"GreatBallsLbl":   g.GreatBalls,
		"UltraBallsLbl":   g.UltraBalls,
		"MasterBallsLbl":  g.MasterBalls,
		"PotionsLbl":      g.Potions,
		"SuperPotionsLbl": g.SuperPotions,
		"WinsLbl":         g.Wins,
		"LossesLbl":       g.Losses,
		"FightsLbl":       g.Fights,
		"CapturesLbl":     g.Catches,
		"MoneyLbl":        g.Money,
	} {
		f.counts[n].setCaption(strconv.FormatUint(uint64(v), 10))
	}

	if p, ok := g.WinPercent(); ok {
		f.counts["Percent"].setCaption(p)
	}

	f.counts["OwnedPKMNLbl"].setCaption(itoa(g.DexCount()))

	if f.w.active && f.w.focus == nil && !f.w.noFocus && f.a.main.w.visible && f.a.desk.topModal() == nil {
		f.a.desk.activate(f.a.main.w)
	}
}

// infoForm is Battle Info (TForm7).
type infoForm struct {
	a    *App
	w    *window
	memo *memo
}

func newInfoForm(a *App) *infoForm {
	f := &infoForm{a: a, w: newWindow(a.desk, &layout.Form7)}
	f.memo = get[*memo](f.w, "Memo1")
	a.desk.add(f.w)

	return f
}

func (f *infoForm) sync() {
	f.memo.setLines(f.a.g.Info.Lines)
}

// dexForm is the POKéDEX (TForm8).
type dexForm struct {
	a     *App
	w     *window
	combo *comboBox
	panel *panel
	image *listView
}

func newDexForm(a *App) *dexForm {
	f := &dexForm{a: a, w: newWindow(a.desk, &layout.Form8)}
	f.combo = get[*comboBox](f.w, "ComboBox1")
	f.panel = get[*panel](f.w, "Panel1")
	f.image = get[*listView](f.w, "ListView1")
	f.combo.items = a.dexItems() // FormShow
	f.w.onShow = f.combo.reselect
	f.combo.onSelect = func(i int) {
		a.g.SelectDex(i)
	}
	get[*button](f.w, "Button1").onClick = func() {
		a.g.CloseDex()
	}
	a.desk.add(f.w)

	return f
}

// sync is TForm8.UpdateDisplay: name, picture, types and evolution level
// always; the attacks once the species is registered.
func (f *dexForm) sync() {
	g := f.a.g
	f.combo.setIndex(g.Dex.Index)

	if g.Dex.Index < 0 {
		return
	}

	s := g.Dex.Index + 1
	sp := g.Data.SpeciesByID(s)
	w := f.w
	l2, l3 := get[*label](w, "Label2"), get[*label](w, "Label3")
	l2.setCaption(g.Data.TypeName(sp.Types[0]))
	l3.setVisible(sp.Types[1] != 0)

	r3 := l3.r // the original puts Label3 at Label2's left plus its width
	l3.setBounds(image.Rect(l2.r.Max.X, r3.Min.Y, l2.r.Max.X+r3.Dx(), r3.Max.Y))

	if sp.Types[1] != 0 {
		l3.setCaption("/" + g.Data.TypeName(sp.Types[1]))
	}

	f.image.setItems([]listItem{{"", s - 1}})
	get[*label](w, "Label1").setCaption(f.combo.items[s-1])

	shown := g.DexMovesShown(s)

	for i, n := range []string{"Label6", "Label10", "Label11", "Label12"} {
		name, typ := "???", "???"
		if mv := g.Data.MoveByID(sp.Moves[i]); shown && mv != nil {
			name, typ = mv.Name, g.Data.TypeName(mv.Type)
		}

		get[*label](w, n).setCaption(name)
		get[*label](w, "Label"+itoa(13+i)).setCaption(typ)
	}

	evolve := "Does not evolve."
	if sp.EvolveLevel != 0 {
		evolve = "Can evolve at level " + itoa(sp.EvolveLevel) + "."
	}

	get[*label](w, "Label4").setCaption(evolve)
	f.panel.setVisible(true)
}

// martForm is the POKéMON Mart (TForm9).
type martForm struct {
	a     *App
	w     *window
	spin  *spinEdit
	radio *radioGroup
	buy   *button
	total *edit
}

func newMartForm(a *App) *martForm {
	f := &martForm{a: a, w: newWindow(a.desk, &layout.Form9)}
	f.spin = get[*spinEdit](f.w, "SpinEdit1")
	f.radio = get[*radioGroup](f.w, "RadioGroup1")
	f.buy = get[*button](f.w, "Button1")
	f.total = get[*edit](f.w, "Edit1")
	g := a.g
	f.spin.onChangeValue = func(v int) {
		g.SetMartQuantity(v)
	}
	f.radio.onClick = func() {
		g.SelectMartItem(f.radio.index)
	}
	f.buy.onClick = func() {
		a.call(g.Buy)
	}
	get[*button](f.w, "Button2").onClick = func() {
		g.LeaveMart()
	}

	f.w.onShow = func() {
		f.spin.tb.set(itoa(g.Mart.Quantity)) // FormShow, before the focus comes in
	}
	a.desk.add(f.w)

	return f
}

func (f *martForm) sync() {
	g := f.a.g
	f.radio.setIndex(g.Mart.Item)
	f.total.tb.set(g.Mart.Total)
	f.buy.setEnabled(g.Mart.BuyEnabled)

	a, b := g.MartDescription()
	get[*label](f.w, "Label2").setCaption(a)
	get[*label](f.w, "Label3").setCaption(b)
}

// typesForm is Type Comparisons (TFormA).
type typesForm struct {
	a       *App
	w       *window
	species *comboBox
	attack  *comboBox
}

func newTypesForm(a *App) *typesForm {
	f := &typesForm{a: a, w: newWindow(a.desk, &layout.FormA)}
	f.species = get[*comboBox](f.w, "ComboBox1")
	f.attack = get[*comboBox](f.w, "ComboBox2")
	g := a.g

	for _, sp := range g.Data.Species { // FormShow
		item := sp.Name + ", " + g.Data.TypeName(sp.Types[0])
		if sp.Types[1] != 0 {
			item += "/" + g.Data.TypeName(sp.Types[1])
		}

		f.species.items = append(f.species.items, item)
	}

	for _, mv := range g.Data.Moves {
		f.attack.items = append(f.attack.items, mv.Name+", "+g.Data.TypeName(mv.Type))
	}

	f.w.onShow = func() {
		f.species.reselect()
		f.attack.reselect()
	}
	f.species.onSelect = func(i int) {
		g.SelectTypes(i, g.Types.Attack)
	}
	f.attack.onSelect = func(i int) {
		g.SelectTypes(g.Types.Species, i)
	}
	get[*button](f.w, "Button1").onClick = func() {
		g.CloseTypes()
	}
	a.desk.add(f.w)

	return f
}

// sync is TFormA.CalculateMultipliers.
func (f *typesForm) sync() {
	g := f.a.g
	f.species.setIndex(g.Types.Species)
	f.attack.setIndex(g.Types.Attack)

	first, second, total, ok := g.TypeMultipliers()
	w := f.w

	if !ok {
		return
	}

	two := g.Data.SpeciesByID(g.Types.Species + 1).Types[1] != 0

	get[*label](w, "Label1").setVisible(true)
	get[*label](w, "Label2").setVisible(two)
	get[*label](w, "Label3").setVisible(true)
	get[*label](w, "Label4").setVisible(two)
	get[*label](w, "Label5").setVisible(true)
	get[*label](w, "Label6").setVisible(true)
	get[*label](w, "Label3").setCaption(first)
	get[*label](w, "Label4").setCaption(second)
	get[*label](w, "Label6").setCaption(total)
}

// cornerForm is the Game Corner (TFormB).
type cornerForm struct {
	a      *App
	w      *window
	bet    *spinEdit
	prize  *radioGroup
	number *radioGroup
	drawn  *radioGroup
}

func newCornerForm(a *App) *cornerForm {
	f := &cornerForm{a: a, w: newWindow(a.desk, &layout.FormB)}
	f.bet = get[*spinEdit](f.w, "SpinEdit1")
	f.prize = get[*radioGroup](f.w, "RadioGroup1")
	f.number = get[*radioGroup](f.w, "RadioGroup2")
	f.drawn = get[*radioGroup](f.w, "RadioGroup3")
	g := a.g
	f.bet.onChangeValue = func(v int) {
		g.SetBet(v)
	}
	f.prize.onClick = func() {
		g.SelectPrize(f.prize.index)
	}
	f.number.onClick = func() {
		g.SelectNumber(f.number.index)
	}
	get[*button](f.w, "Button1").onClick = func() {
		a.call(g.Draw)
	}
	get[*button](f.w, "Button2").onClick = func() {
		g.CloseCorner()
	}

	f.w.onShow = func() { // FormShow, before the focus comes in and selects the bet
		f.bet.maxValue = g.Corner.MaxBet
		f.bet.tb.set(itoa(g.Corner.Bet))
	}
	a.desk.add(f.w)

	return f
}

func (f *cornerForm) sync() {
	g := f.a.g
	f.prize.setIndex(g.Corner.Prize)
	f.number.setIndex(g.Corner.Number)
	f.drawn.setIndex(g.Corner.Drawn)
}

// drawTime is how long a play's draws show. The original repainted the
// window for each of its 10,000 draws, which took a while; the port shows
// them in a second.
const drawTime time.Duration = time.Second

// showDraws shows a play's draws in "Lucky number", one after another, as
// the original's loop did, before the game tells the result. It runs on
// the handler's goroutine, which waits until the last draw shows; the
// user interface takes no input meanwhile, as the original's didn't.
func (a *App) showDraws(draws []int) {
	done := make(chan struct{})

	a.requests <- func() {
		a.stats.freshen() // Form5.Freshen: the bet paid

		a.draws = &drawShow{
			draws: draws,
			start: a.desk.now,
			done: func() {
				close(done)
				a.wait()
			},
		}
	}

	<-done
}

// drawShow is a play's draws showing.
type drawShow struct {
	draws []int
	start time.Time
	done  func()
}

// tick shows the draw due at now; the last one ends the show.
func (s *drawShow) tick(a *App, now time.Time) {
	t := now.Sub(s.start)
	i := len(s.draws) - 1

	if t < drawTime {
		i = int(int64(len(s.draws)) * int64(t) / int64(drawTime))
	}

	a.corner.drawn.setIndex(s.draws[i])

	a.desk.dirty = true
	if t >= drawTime {
		a.draws = nil

		s.done()
	}
}
