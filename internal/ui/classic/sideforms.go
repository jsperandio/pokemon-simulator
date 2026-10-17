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
		idx := i
		sf.items[i] = get[*button](wdw, "Button"+strconv.Itoa(i+1))
		sf.items[i].onClick = func() {
			app.call(func() {
				app.game.UseItem(idx)
			})
		}
	}

	for _, nam := range []string{
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
		sf.counts[nam] = get[*label](wdw, nam)
	}

	sf.party.onDblClick = func() {
		if idx := sf.party.selected; idx >= 0 {
			app.call(func() {
				app.game.ClickParty(idx)
			})
		}
	}
	// FormClick: a click on the window gives the focus to the game window.
	wdw.onClick = func() {
		if app.main.wdw.visible {
			app.desk.activate(app.main.wdw)
		}
	}
	app.desk.add(wdw)

	return sf
}

func (stf *statsForm) sync() {
	gam := stf.app.game
	stf.counts["PlayerNameLbl"].setCaption(gam.Stats.Name)

	party := gam.Party()
	if len(party) != len(stf.items0) || !slices.EqualFunc(party, stf.items0, func(slt save.Slot, it listItem) bool {
		return it.image == int(slt.Species)-1
	}) {
		items := make([]listItem, len(party))
		for i, slt := range party {
			items[i] = listItem{stf.app.speciesName(int(slt.Species)), int(slt.Species) - 1}
		}

		stf.items0 = items
		stf.party.setItems(items)
	}

	if stf.party.enabled != gam.Stats.PartyEnabled {
		stf.party.setEnabled(gam.Stats.PartyEnabled)

		stf.party.color = colWindow
		if !gam.Stats.PartyEnabled {
			stf.party.color = colFace
		}
	}

	for i, btn := range stf.items {
		btn.setVisible(gam.Stats.Items)
		btn.setEnabled(gam.Stats.ItemEnabled[i])
	}
}

// freshen is TForm5.Freshen, which the original runs on every tick while
// the window shows: the counters, and the focus back to the game window
// when this one has it without a control to hold it.
func (stf *statsForm) freshen() {
	if !stf.wdw.visible {
		return
	}

	gam := stf.app.game
	for nam, val := range map[string]uint32{
		"PokeBallsLbl":    gam.PokeBalls,
		"GreatBallsLbl":   gam.GreatBalls,
		"UltraBallsLbl":   gam.UltraBalls,
		"MasterBallsLbl":  gam.MasterBalls,
		"PotionsLbl":      gam.Potions,
		"SuperPotionsLbl": gam.SuperPotions,
		"WinsLbl":         gam.Wins,
		"LossesLbl":       gam.Losses,
		"FightsLbl":       gam.Fights,
		"CapturesLbl":     gam.Catches,
		"MoneyLbl":        gam.Money,
	} {
		stf.counts[nam].setCaption(strconv.FormatUint(uint64(val), 10))
	}

	if pct, ok := gam.WinPercent(); ok {
		stf.counts["Percent"].setCaption(pct)
	}

	stf.counts["OwnedPKMNLbl"].setCaption(strconv.Itoa(gam.PokedexCount()))

	if stf.wantsFocusBack() {
		stf.app.desk.activate(stf.app.main.wdw)
	}
}

// wantsFocusBack reports whether the window is active but nothing in it has
// the focus, while the game window shows and no modal window is up: the
// activation goes back to the game window.
func (stf *statsForm) wantsFocusBack() bool {
	app, wdw := stf.app, stf.wdw
	nothingFocused := wdw.active && wdw.focus == nil && !wdw.noFocus

	return nothingFocused && app.main.wdw.visible && app.desk.topModal() == nil
}

// infoForm is Battle Info (TForm7).
type infoForm struct {
	app  *Application
	wdw  *window
	memo *memo
}

func newInfoForm(app *Application) *infoForm {
	inf := &infoForm{
		app: app,
		wdw: newWindow(app.desk, &layout.Form7),
	}
	inf.memo = get[*memo](inf.wdw, "Memo1")
	app.desk.add(inf.wdw)

	return inf
}

func (inf *infoForm) sync() {
	gam := inf.app.game
	inf.memo.setLines(gam.Info.Lines)
}

// pokedexForm is the POKéDEX (TForm8).
type pokedexForm struct {
	app   *Application
	wdw   *window
	combo *comboBox
	panel *panel
	image *listView
}

func newPokedexForm(app *Application) *pokedexForm {
	pf := &pokedexForm{
		app: app,
		wdw: newWindow(app.desk, &layout.Form8),
	}
	pf.combo = get[*comboBox](pf.wdw, "ComboBox1")
	pf.panel = get[*panel](pf.wdw, "Panel1")
	pf.image = get[*listView](pf.wdw, "ListView1")
	pf.combo.items = app.pokedexItems() // FormShow
	pf.wdw.onShow = pf.combo.reselect
	pf.combo.onSelect = func(i int) {
		app.game.SelectPokedex(i)
	}
	get[*button](pf.wdw, "Button1").onClick = func() {
		app.game.ClosePokedex()
	}
	app.desk.add(pf.wdw)

	return pf
}

// sync is TForm8.UpdateDisplay: name, picture, types and evolution level
// always; the attacks once the species is registered.
func (pf *pokedexForm) sync() {
	gam := pf.app.game
	pf.combo.setIndex(gam.Pokedex.Index)

	if gam.Pokedex.Index < 0 {
		return
	}

	sid := gam.Pokedex.Index + 1
	sp := gam.Data.SpeciesByID(sid)
	wdw := pf.wdw
	l2, l3 := get[*label](wdw, "Label2"), get[*label](wdw, "Label3")
	l2.setCaption(gam.Data.TypeName(sp.Types[0]))
	l3.setVisible(sp.Types[1] != 0)

	r3 := l3.rect // the original puts Label3 at Label2's left plus its width
	l3.setBounds(image.Rect(l2.rect.Max.X, r3.Min.Y, l2.rect.Max.X+r3.Dx(), r3.Max.Y))

	if sp.Types[1] != 0 {
		l3.setCaption("/" + gam.Data.TypeName(sp.Types[1]))
	}

	pf.image.setItems([]listItem{{"", sid - 1}})
	get[*label](wdw, "Label1").setCaption(pf.combo.items[sid-1])

	shown := gam.PokedexMovesShown(sid)

	for i, nam := range []string{"Label6", "Label10", "Label11", "Label12"} {
		name, typ := "???", "???"
		if mv := gam.Data.MoveByID(sp.Moves[i]); shown && mv != nil {
			name, typ = mv.Name, gam.Data.TypeName(mv.Type)
		}

		get[*label](wdw, nam).setCaption(name)
		get[*label](wdw, "Label"+strconv.Itoa(13+i)).setCaption(typ)
	}

	evolve := "Does not evolve."
	if sp.EvolveLevel != 0 {
		evolve = "Can evolve at level " + strconv.Itoa(sp.EvolveLevel) + "."
	}

	get[*label](wdw, "Label4").setCaption(evolve)
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

func newPokemartForm(app *Application) *pokemartForm {
	pmf := &pokemartForm{
		app: app,
		wdw: newWindow(app.desk, &layout.Form9),
	}
	pmf.spin = get[*spinEdit](pmf.wdw, "SpinEdit1")
	pmf.radio = get[*radioGroup](pmf.wdw, "RadioGroup1")
	pmf.buy = get[*button](pmf.wdw, "Button1")
	pmf.total = get[*edit](pmf.wdw, "Edit1")
	gam := app.game
	pmf.spin.onChangeValue = func(val int) {
		gam.SetPokemartQuantity(val)
	}
	pmf.radio.onClick = func() {
		gam.SelectPokemartItem(pmf.radio.index)
	}
	pmf.buy.onClick = func() {
		app.call(gam.Buy)
	}
	get[*button](pmf.wdw, "Button2").onClick = func() {
		gam.LeavePokemart()
	}

	pmf.wdw.onShow = func() {
		pmf.spin.tb.set(strconv.Itoa(gam.Pokemart.Quantity)) // FormShow, before the focus comes in
	}
	app.desk.add(pmf.wdw)

	return pmf
}

func (pmf *pokemartForm) sync() {
	gam := pmf.app.game
	pmf.radio.setIndex(gam.Pokemart.Item)
	pmf.total.tb.set(gam.Pokemart.Total)
	pmf.buy.setEnabled(gam.Pokemart.BuyEnabled)

	fst, snd := gam.PokemartDescription()
	get[*label](pmf.wdw, "Label2").setCaption(fst)
	get[*label](pmf.wdw, "Label3").setCaption(snd)
}

// typesForm is Type Comparisons (TFormA).
type typesForm struct {
	app     *Application
	wdw     *window
	species *comboBox
	attack  *comboBox
}

func newTypesForm(app *Application) *typesForm {
	tf := &typesForm{
		app: app,
		wdw: newWindow(app.desk, &layout.FormA),
	}
	tf.species = get[*comboBox](tf.wdw, "ComboBox1")
	tf.attack = get[*comboBox](tf.wdw, "ComboBox2")
	gam := app.game

	for _, sp := range gam.Data.Species { // FormShow
		item := sp.Name + ", " + gam.Data.TypeName(sp.Types[0])
		if sp.Types[1] != 0 {
			item += "/" + gam.Data.TypeName(sp.Types[1])
		}

		tf.species.items = append(tf.species.items, item)
	}

	for _, mv := range gam.Data.Moves {
		tf.attack.items = append(tf.attack.items, mv.Name+", "+gam.Data.TypeName(mv.Type))
	}

	tf.wdw.onShow = func() {
		tf.species.reselect()
		tf.attack.reselect()
	}
	tf.species.onSelect = func(i int) {
		gam.SelectTypes(i, gam.Types.Attack)
	}
	tf.attack.onSelect = func(i int) {
		gam.SelectTypes(gam.Types.Species, i)
	}
	get[*button](tf.wdw, "Button1").onClick = func() {
		gam.CloseTypes()
	}
	app.desk.add(tf.wdw)

	return tf
}

// sync is TFormA.CalculateMultipliers.
func (tf *typesForm) sync() {
	gam := tf.app.game
	tf.species.setIndex(gam.Types.Species)
	tf.attack.setIndex(gam.Types.Attack)

	first, second, total, ok := gam.TypeMultipliers()
	wdw := tf.wdw

	if !ok {
		return
	}

	two := gam.Data.SpeciesByID(gam.Types.Species + 1).Types[1] != 0

	get[*label](wdw, "Label1").setVisible(true)
	get[*label](wdw, "Label2").setVisible(two)
	get[*label](wdw, "Label3").setVisible(true)
	get[*label](wdw, "Label4").setVisible(two)
	get[*label](wdw, "Label5").setVisible(true)
	get[*label](wdw, "Label6").setVisible(true)
	get[*label](wdw, "Label3").setCaption(first)
	get[*label](wdw, "Label4").setCaption(second)
	get[*label](wdw, "Label6").setCaption(total)
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

func newGameCornerForm(app *Application) *gameCornerForm {
	gcf := &gameCornerForm{
		app: app,
		wdw: newWindow(app.desk, &layout.FormB),
	}
	gcf.bet = get[*spinEdit](gcf.wdw, "SpinEdit1")
	gcf.prize = get[*radioGroup](gcf.wdw, "RadioGroup1")
	gcf.number = get[*radioGroup](gcf.wdw, "RadioGroup2")
	gcf.drawn = get[*radioGroup](gcf.wdw, "RadioGroup3")
	gam := app.game
	gcf.bet.onChangeValue = func(val int) {
		gam.SetBet(val)
	}
	gcf.prize.onClick = func() {
		gam.SelectPrize(gcf.prize.index)
	}
	gcf.number.onClick = func() {
		gam.SelectNumber(gcf.number.index)
	}
	get[*button](gcf.wdw, "Button1").onClick = func() {
		app.call(gam.Draw)
	}
	get[*button](gcf.wdw, "Button2").onClick = func() {
		gam.CloseGameCorner()
	}

	gcf.wdw.onShow = func() { // FormShow, before the focus comes in and selects the bet
		gcf.bet.maxValue = gam.GameCorner.MaxBet
		gcf.bet.tb.set(strconv.Itoa(gam.GameCorner.Bet))
	}
	app.desk.add(gcf.wdw)

	return gcf
}

func (gcf *gameCornerForm) sync() {
	gam := gcf.app.game
	gcf.prize.setIndex(gam.GameCorner.Prize)
	gcf.number.setIndex(gam.GameCorner.Number)
	gcf.drawn.setIndex(gam.GameCorner.Drawn)
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
func (ds *drawShow) tick(app *Application, now time.Time) {
	dur := now.Sub(ds.start)
	i := len(ds.draws) - 1

	if dur < drawTime {
		i = int(int64(len(ds.draws)) * int64(dur) / int64(drawTime))
	}

	app.gameCorner.drawn.setIndex(ds.draws[i])

	app.desk.dirty = true
	if dur >= drawTime {
		app.draws = nil

		ds.done()
	}
}
