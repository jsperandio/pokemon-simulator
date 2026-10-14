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
	app    *Application
	wdw    *window
	party  *listView
	items  [6]*button
	counts map[string]*label
	items0 []listItem // the party as last shown
}

func newStatsForm(app *Application) *statsForm {
	wdw := newWindow(app.desk, &layout.Form5)

	sf := &statsForm{
		app:    app,
		wdw:    wdw,
		counts: map[string]*label{},
	}
	sf.party = get[*listView](wdw, "ListView1")

	for i := range sf.items {
		k := i
		sf.items[i] = get[*button](wdw, "Button"+itoa(i+1))
		sf.items[i].onClick = func() {
			app.call(func() {
				app.g.UseItem(k)
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
		sf.counts[n] = get[*label](wdw, n)
	}

	sf.party.onDblClick = func() {
		if k := sf.party.selected; k >= 0 {
			app.call(func() {
				app.g.ClickParty(k)
			})
		}
	}
	// FormClick: a click on the window gives the focus to the game window.
	wdw.onClick = func() {
		if app.main.w.visible {
			app.desk.activate(app.main.w)
		}
	}
	app.desk.add(wdw)

	return sf
}

func (stf *statsForm) sync() {
	g := stf.app.g
	stf.counts["PlayerNameLbl"].setCaption(g.Stats.Name)

	party := g.Party()
	if len(party) != len(stf.items0) || !slices.EqualFunc(party, stf.items0, func(m save.Slot, it listItem) bool {
		return it.image == int(m.Species)-1
	}) {
		items := make([]listItem, len(party))
		for i, m := range party {
			items[i] = listItem{stf.app.speciesName(int(m.Species)), int(m.Species) - 1}
		}

		stf.items0 = items
		stf.party.setItems(items)
	}

	if stf.party.enabled != g.Stats.PartyEnabled {
		stf.party.setEnabled(g.Stats.PartyEnabled)

		stf.party.color = colWindow
		if !g.Stats.PartyEnabled {
			stf.party.color = colFace
		}
	}

	for i, b := range stf.items {
		b.setVisible(g.Stats.Items)
		b.setEnabled(g.Stats.ItemEnabled[i])
	}
}

// freshen is TForm5.Freshen, which the original runs on every tick while
// the window shows: the counters, and the focus back to the game window
// when this one has it without a control to hold it.
func (stf *statsForm) freshen() {
	if !stf.wdw.visible {
		return
	}

	g := stf.app.g
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
		stf.counts[n].setCaption(strconv.FormatUint(uint64(v), 10))
	}

	if p, ok := g.WinPercent(); ok {
		stf.counts["Percent"].setCaption(p)
	}

	stf.counts["OwnedPKMNLbl"].setCaption(itoa(g.DexCount()))

	if stf.wantsFocusBack() {
		stf.app.desk.activate(stf.app.main.w)
	}
}

// wantsFocusBack reports whether the window is active but nothing in it has
// the focus, while the game window shows and no modal window is up: the
// activation goes back to the game window.
func (stf *statsForm) wantsFocusBack() bool {
	a, w := stf.app, stf.wdw
	nothingFocused := w.active && w.focus == nil && !w.noFocus

	return nothingFocused && a.main.w.visible && a.desk.topModal() == nil
}

// infoForm is Battle Info (TForm7).
type infoForm struct {
	app  *Application
	wdw  *window
	memo *memo
}

func newInfoForm(app *Application) *infoForm {
	f := &infoForm{
		app: app,
		wdw: newWindow(app.desk, &layout.Form7),
	}
	f.memo = get[*memo](f.wdw, "Memo1")
	app.desk.add(f.wdw)

	return f
}

func (inf *infoForm) sync() {
	g := inf.app.g
	inf.memo.setLines(g.Info.Lines)
}

// pokedexForm is the POKéDEX (TForm8).
type pokedexForm struct {
	app   *Application
	wdw   *window
	combo *comboBox
	panel *panel
	image *listView
}

func newDexForm(a *Application) *pokedexForm {
	f := &pokedexForm{
		app: a,
		wdw: newWindow(a.desk, &layout.Form8),
	}
	f.combo = get[*comboBox](f.wdw, "ComboBox1")
	f.panel = get[*panel](f.wdw, "Panel1")
	f.image = get[*listView](f.wdw, "ListView1")
	f.combo.items = a.dexItems() // FormShow
	f.wdw.onShow = f.combo.reselect
	f.combo.onSelect = func(i int) {
		a.g.SelectDex(i)
	}
	get[*button](f.wdw, "Button1").onClick = func() {
		a.g.CloseDex()
	}
	a.desk.add(f.wdw)

	return f
}

// sync is TForm8.UpdateDisplay: name, picture, types and evolution level
// always; the attacks once the species is registered.
func (pf *pokedexForm) sync() {
	g := pf.app.g
	pf.combo.setIndex(g.Dex.Index)

	if g.Dex.Index < 0 {
		return
	}

	s := g.Dex.Index + 1
	sp := g.Data.SpeciesByID(s)
	w := pf.wdw
	l2, l3 := get[*label](w, "Label2"), get[*label](w, "Label3")
	l2.setCaption(g.Data.TypeName(sp.Types[0]))
	l3.setVisible(sp.Types[1] != 0)

	r3 := l3.rect // the original puts Label3 at Label2's left plus its width
	l3.setBounds(image.Rect(l2.rect.Max.X, r3.Min.Y, l2.rect.Max.X+r3.Dx(), r3.Max.Y))

	if sp.Types[1] != 0 {
		l3.setCaption("/" + g.Data.TypeName(sp.Types[1]))
	}

	pf.image.setItems([]listItem{{"", s - 1}})
	get[*label](w, "Label1").setCaption(pf.combo.items[s-1])

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
	pf.panel.setVisible(true)
}

// pokemartForm is the POKéMON Mart (TForm9).
type pokemartForm struct {
	app   *Application
	wdw   *window
	spin  *spinEdit
	radio *radioGroup
	buy   *button
	total *edit
}

func newMartForm(a *Application) *pokemartForm {
	f := &pokemartForm{
		app: a,
		wdw: newWindow(a.desk, &layout.Form9),
	}
	f.spin = get[*spinEdit](f.wdw, "SpinEdit1")
	f.radio = get[*radioGroup](f.wdw, "RadioGroup1")
	f.buy = get[*button](f.wdw, "Button1")
	f.total = get[*edit](f.wdw, "Edit1")
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
	get[*button](f.wdw, "Button2").onClick = func() {
		g.LeaveMart()
	}

	f.wdw.onShow = func() {
		f.spin.tb.set(itoa(g.Mart.Quantity)) // FormShow, before the focus comes in
	}
	a.desk.add(f.wdw)

	return f
}

func (pmf *pokemartForm) sync() {
	g := pmf.app.g
	pmf.radio.setIndex(g.Mart.Item)
	pmf.total.tb.set(g.Mart.Total)
	pmf.buy.setEnabled(g.Mart.BuyEnabled)

	a, b := g.MartDescription()
	get[*label](pmf.wdw, "Label2").setCaption(a)
	get[*label](pmf.wdw, "Label3").setCaption(b)
}

// typesForm is Type Comparisons (TFormA).
type typesForm struct {
	app     *Application
	wdw     *window
	species *comboBox
	attack  *comboBox
}

func newTypesForm(a *Application) *typesForm {
	f := &typesForm{
		app: a,
		wdw: newWindow(a.desk, &layout.FormA),
	}
	f.species = get[*comboBox](f.wdw, "ComboBox1")
	f.attack = get[*comboBox](f.wdw, "ComboBox2")
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

	f.wdw.onShow = func() {
		f.species.reselect()
		f.attack.reselect()
	}
	f.species.onSelect = func(i int) {
		g.SelectTypes(i, g.Types.Attack)
	}
	f.attack.onSelect = func(i int) {
		g.SelectTypes(g.Types.Species, i)
	}
	get[*button](f.wdw, "Button1").onClick = func() {
		g.CloseTypes()
	}
	a.desk.add(f.wdw)

	return f
}

// sync is TFormA.CalculateMultipliers.
func (tf *typesForm) sync() {
	g := tf.app.g
	tf.species.setIndex(g.Types.Species)
	tf.attack.setIndex(g.Types.Attack)

	first, second, total, ok := g.TypeMultipliers()
	w := tf.wdw

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

// gameCornerForm is the Game Corner (TFormB).
type gameCornerForm struct {
	app    *Application
	wdw    *window
	bet    *spinEdit
	prize  *radioGroup
	number *radioGroup
	drawn  *radioGroup
}

func newCornerForm(app *Application) *gameCornerForm {
	f := &gameCornerForm{
		app: app,
		wdw: newWindow(app.desk, &layout.FormB),
	}
	f.bet = get[*spinEdit](f.wdw, "SpinEdit1")
	f.prize = get[*radioGroup](f.wdw, "RadioGroup1")
	f.number = get[*radioGroup](f.wdw, "RadioGroup2")
	f.drawn = get[*radioGroup](f.wdw, "RadioGroup3")
	g := app.g
	f.bet.onChangeValue = func(v int) {
		g.SetBet(v)
	}
	f.prize.onClick = func() {
		g.SelectPrize(f.prize.index)
	}
	f.number.onClick = func() {
		g.SelectNumber(f.number.index)
	}
	get[*button](f.wdw, "Button1").onClick = func() {
		app.call(g.Draw)
	}
	get[*button](f.wdw, "Button2").onClick = func() {
		g.CloseCorner()
	}

	f.wdw.onShow = func() { // FormShow, before the focus comes in and selects the bet
		f.bet.maxValue = g.Corner.MaxBet
		f.bet.tb.set(itoa(g.Corner.Bet))
	}
	app.desk.add(f.wdw)

	return f
}

func (gcf *gameCornerForm) sync() {
	g := gcf.app.g
	gcf.prize.setIndex(g.Corner.Prize)
	gcf.number.setIndex(g.Corner.Number)
	gcf.drawn.setIndex(g.Corner.Drawn)
}

// drawTime is how long a play's draws show. The original repainted the
// window for each of its 10,000 draws, which took a while; the port shows
// them in a second.
const drawTime time.Duration = time.Second

// drawShow is a play's draws showing.
type drawShow struct {
	draws []int
	start time.Time
	done  func()
}

// tick shows the draw due at now; the last one ends the show.
func (ds *drawShow) tick(a *Application, now time.Time) {
	t := now.Sub(ds.start)
	i := len(ds.draws) - 1

	if t < drawTime {
		i = int(int64(len(ds.draws)) * int64(t) / int64(drawTime))
	}

	a.corner.drawn.setIndex(ds.draws[i])

	a.desk.dirty = true
	if t >= drawTime {
		a.draws = nil

		ds.done()
	}
}
