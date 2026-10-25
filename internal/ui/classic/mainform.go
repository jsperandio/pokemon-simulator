// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"errors"
	"image"
	"image/color"
	"slices"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/audio"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// mainForm is the game window (TForm4): the menu, the starter choice,
// the battle arena, the evolution choice and the Mewtwo-power picker.
type mainForm struct {
	app *Application
	wdw *window

	starterLabel *label
	starters     *listView
	starterSpot  image.Point // where the starter choice centred the window
	gameSpot     image.Point // where the window goes once a starter is chosen

	evolveChoice *radioGroup
	evolveButton *button

	picker      []widget // Label10, Label11, ComboBox1, SpinEdit1, Button2–4
	pickCombo   *comboBox
	pickLevel   *spinEdit
	pickerShown bool

	arena          *groupBox
	ownLabel       *label // Label2
	enemyLabel     *label // Label3
	ownHP, enemyHP *label // Label8, Label9
	ownGauge       *gauge // Gauge1
	enemyGauge     *gauge // Gauge2
	pokeball       *picture
	attacks        *radioGroup
	enemyAttacks   *groupBox
	enemyMoves     [4]*label // Label4–7
	goButton       *button
	chosen         int       // the fighters sent out at the last sync: a new one takes Go!'s focus
	ownImage       *listView // ListView2
	enemyImage     *listView // ListView3
	endButton      *button
	searchButton   *button    // SearchAgainButton
	enemyAppear    appearance // the wild POKéMON's, over ListView3
	ownAppear      appearance // the player's fighter's, over ListView2
	enemySparkle   sparkle    // the wild POKéMON's stars when it is shiny
	ownSparkle     sparkle    // the player's fighter's stars when it is shiny
}

// searchAgainButton is the Battle Arena's Search Again button, which the
// original doesn't have: once a battle a search found is over, it shows in
// Go!'s place, as high as Go!, and ends where Go! ends.
var searchAgainButton = layout.Component{
	Class:   "TButton",
	Name:    "SearchAgainButton",
	Bounds:  layout.Rect{X: 214, Y: 13, W: 75, H: 22},
	Caption: "Search Again",
	Hidden:  true,
	Props: map[string]string{
		"TabOrder": "6",
	},
}

// searchShortcut is Search for a POKéMON's shortcut, which the original
// doesn't have: F3, as a TShortCut.
const searchShortcut string = "114"

// goBounds is Go!'s place in the Battle Arena, in the room arenaRise and
// arenaDrop make: 22 pixels high, 6 below the arena's frame and 5 above
// the gauges, and ending with the gauges and the attack boxes. The
// original's Go! was 15 high, with no room above or below it, and ended 2
// pixels further right.
var goBounds = layout.Rect{X: 246, Y: 13, W: 43, H: 22}

// arenaRise and arenaDrop are how much further up and down the game window
// the Battle Arena's frame reaches than the original's, into the window's
// margins, so the window keeps its size. The arena's contents move down by
// both, which leaves room above the gauges for goBounds.
const (
	arenaRise int = 10
	arenaDrop int = 6
)

// mainLayout is the original's game window with the port's changes. The
// Battle Arena's frame reaches arenaRise further up and arenaDrop further
// down, and its contents move down by both. Go! is at goBounds, and is the
// default button, so Enter presses it even when an attack has the focus.
// The arena has searchAgainButton, and Search for a POKéMON has
// searchShortcut.
func mainLayout() *layout.Component {
	frm := layout.Form4.Clone()
	arena := frm.Find("GroupBox1")
	arena.Bounds.Y -= arenaRise
	arena.Bounds.H += arenaRise + arenaDrop

	for i := range arena.Children {
		arena.Children[i].Bounds.Y += arenaRise + arenaDrop
	}

	arena.Children = append(arena.Children, searchAgainButton)
	frm.Find("FindPKMN1").Props = map[string]string{
		"ShortCut": searchShortcut,
	}

	goButton := frm.Find("Button1")
	goButton.Bounds = goBounds
	goButton.Props["Default"] = "True"

	return &frm
}

func newMainForm(app *Application) *mainForm {
	lay := mainLayout()
	wdw := newWindow(app.desk, lay)
	mf := &mainForm{
		app: app,
		wdw: wdw,
	}
	mf.starterLabel = get[*label](wdw, "Label1")
	mf.starters = get[*listView](wdw, "ListView1")
	mf.evolveChoice = get[*radioGroup](wdw, "EvolveChoice")
	mf.evolveButton = get[*button](wdw, "EvolveButton")
	mf.pickCombo = get[*comboBox](wdw, "ComboBox1")
	mf.pickLevel = get[*spinEdit](wdw, "SpinEdit1")

	pickerNames := []string{
		"Label10",
		"Label11",
		"ComboBox1",
		"SpinEdit1",
		"Button2",
		"Button3",
		"Button4",
	}

	for _, nam := range pickerNames {
		mf.picker = append(mf.picker, wdw.find(nam))
	}

	mf.arena = get[*groupBox](wdw, "GroupBox1")
	mf.ownLabel = get[*label](wdw, "Label2")
	mf.enemyLabel = get[*label](wdw, "Label3")
	mf.ownHP = get[*label](wdw, "Label8")
	mf.enemyHP = get[*label](wdw, "Label9")
	mf.ownGauge = get[*gauge](wdw, "Gauge1")
	mf.enemyGauge = get[*gauge](wdw, "Gauge2")
	mf.pokeball = get[*picture](wdw, "Image1")
	mf.attacks = get[*radioGroup](wdw, "RadioGroup1")
	mf.enemyAttacks = get[*groupBox](wdw, "GroupBox2")

	for i := range mf.enemyMoves {
		mf.enemyMoves[i] = get[*label](wdw, "Label"+strconv.Itoa(4+i))
	}

	mf.goButton = get[*button](wdw, "Button1")
	mf.ownImage = get[*listView](wdw, "ListView2")
	mf.enemyImage = get[*listView](wdw, "ListView3")
	// The arena's sprites only show. The original's list views select a
	// clicked sprite, which Windows draws tinted blue, with a focus
	// rectangle beside it.
	mf.ownImage.showOnly = true
	mf.enemyImage.showOnly = true
	mf.endButton = get[*button](wdw, "EndButton")
	mf.searchButton = get[*button](wdw, "SearchAgainButton")
	mf.widenArena()
	mf.placeGauges()
	mf.anchorForFill()

	mf.arena.over = func(cv canvas) {
		enemy, own := mf.enemyImage.rect.Min, mf.ownImage.rect.Min
		mf.enemyAppear.paint(cv, enemy)
		mf.ownAppear.paint(cv, own)
		mf.enemySparkle.paint(cv, enemy.Add(spriteCentre))
		mf.ownSparkle.paint(cv, own.Add(spriteCentre))
	}

	gam := app.game
	mf.starters.onClick = mf.starterClick
	mf.starters.mark = func(cv canvas, x, y int) {
		app.preferences.shinyMark.draw(cv, x, y)
	}
	mf.evolveChoice.onClick = func() {
		gam.ChooseEvolution(mf.evolveChoice.index)
	}
	mf.evolveButton.onClick = func() {
		app.call(gam.Evolve)
	}

	mf.pickCombo.items = app.pokedexItems() // FormShow
	mf.pickCombo.onSelect = func(i int) {
		gam.PickSpecies(i)
	}
	mf.pickLevel.onChangeValue = func(val int) {
		gam.PickLevel(val)
	}
	get[*button](wdw, "Button2").onClick = func() {
		app.call(gam.PickGo)
	}
	get[*button](wdw, "Button3").onClick = func() {
		app.call(gam.PickCancel)
	}
	get[*button](wdw, "Button4").onClick = func() {
		app.call(gam.PickRandom)
	}
	mf.attacks.onClick = func() {
		gam.Main.AttackIndex = mf.attacks.index
	}
	mf.goButton.onClick = func() {
		app.call(gam.Go)
	}
	mf.endButton.onClick = func() {
		app.call(gam.EndFight)
	}
	mf.searchButton.onClick = func() {
		app.call(gam.SearchAgain)
	}
	wdw.onClose = mf.formClose
	wdw.onKey = mf.key

	mb := newMenuBar(wdw, gam.Menu, lay.Find("MainMenu1"), map[string]*game.MenuItem{
		"Quit1": {
			Name:    "Preferences1",
			Caption: "Preferences...",
			Enabled: true,
			Visible: true,
		},
	})
	mb.onCommand = mf.menuCommand

	app.desk.add(wdw)

	return mf
}

// starterClick is a click on the starter list (TForm4.ListView1Click).
func (mf *mainForm) starterClick() {
	idx := mf.starters.selected
	if idx < 0 {
		return
	}

	app := mf.app
	app.call(func() {
		app.game.ChooseStarter(idx)
		mf.leaveCentre()
	})
}

// anchorForFill sets the anchors that fill the screen with the window: the
// Battle Arena's frame stretches, and its contents, the starter list and
// its caption keep their size, centred. The other controls stay at the
// top left. The layout's anchors are the original's defaults, which the
// separate windows keep, as they never resize.
func (mf *mainForm) anchorForFill() {
	mf.arena.setAnchors(akLeft | akTop | akRight | akBottom)

	for _, wgt := range mf.arena.kids {
		wgt.base().setAnchors(0)
	}

	mf.starterLabel.setAnchors(0)
	mf.starters.setAnchors(0)
}

// centreForStarters puts the window in the middle of the screen for the
// starter choice, when it is the only one shown; the spot it held beside
// the other windows waits for the choice. Filling the screen, the window
// already has its place.
func (mf *mainForm) centreForStarters() {
	mf.gameSpot = mf.wdw.pos

	if !mf.app.preferences.fillScreen {
		mf.wdw.center()
	}

	mf.starterSpot = mf.wdw.pos
	mf.app.desk.dirty = true
}

// leaveCentre moves the window from the middle of the screen to its place
// beside the Statistics and Battle Info windows, once a starter is chosen,
// unless the player has moved it.
func (mf *mainForm) leaveCentre() {
	mgw := &mf.app.game.Main
	if mgw.Starters || mf.wdw.pos != mf.starterSpot {
		return
	}

	mf.wdw.pos = mf.gameSpot
	mf.app.desk.dirty = true
}

// key is the game window's own key: F3 is Search Again once a battle a
// search found is over, as the menu that has it for Search for a POKéMON
// is disabled then.
func (mf *mainForm) key(key ebiten.Key, mds mods) bool {
	if key != ebiten.KeyF3 || mds != (mods{}) {
		return false
	}

	mf.searchButton.click()

	return true
}

// menuCommand runs a menu item: the user interface's own, or the game's.
func (mf *mainForm) menuCommand(me *menuEntry) {
	app := mf.app

	switch me.name() {
	case "ExportPOKMONinformation1":
		app.call(mf.export)
	case "Quit1":
		mf.formClose()
	case "Preferences1":
		app.showPreferences()
	case "Openfile1":
		app.call(mf.openMusic)
	case "StopMusic1":
		mf.stopMusic()
	case "Contents1":
		app.help()
	case "About1":
		app.showAbout()
	default:
		app.call(func() {
			app.game.ClickMenu(me.name())
		})
	}
}

// formClose is closing the game window (TForm4.FormClose): a tournament's
// penalty, then the save, then the end of the program.
func (mf *mainForm) formClose() {
	app := mf.app
	app.call(func() {
		if dat := app.game.Close(); dat != nil {
			if err := app.files.WriteFile(app.game.SaveFile, dat); err != nil {
				app.Message(game.Error, false, err.Error())
			}
		}

		app.exit()
	})
}

// export is File → Export POKéMON information...: the page, written where
// the player chooses, then shown if wanted.
func (mf *mainForm) export() {
	app := mf.app
	file, ok := app.openFile(fileOptions{
		save:       true,
		filter:     "HTML Files (*.htm)|*.htm",
		defaultExt: "htm",
		overwrite:  true,
	})

	if !ok {
		return
	}

	page := app.game.ExportPage(time.Now())
	if err := app.files.WriteFile(file, page); err != nil {
		app.Message(game.Error, false, err.Error())

		return
	}

	if app.game.Exported(file) {
		if err := platform.OpenPage(file, page); err != nil {
			app.Message(game.Error, false, err.Error())
		}
	}
}

// musicFilter is Music → Open file...'s. The original's offered what
// Windows Media Player played; these are what the port plays.
const musicFilter string = "All supported formats (*.mp3;*.ogg;*.wav)|*.mp3;*.ogg;*.wav|" +
	"MPEG Audio Files (*.mp3)|*.mp3|Ogg Vorbis Files (*.ogg)|*.ogg|Wave Files (*.wav)|*.wav|All files (*.*)|*.*"

// openMusic is Music → Open file... (TForm4.Openfile1Click): the music
// chosen plays at once.
func (mf *mainForm) openMusic() {
	app := mf.app
	file, ok := app.openFile(fileOptions{
		filter:    musicFilter,
		mustExist: true,
	})

	if !ok {
		return
	}

	dat, err := app.files.ReadFile(file)
	if err == nil {
		err = app.music.Open(app.files.Base(file), dat)
	}

	switch {
	case errors.Is(err, audio.ErrFormat):
		app.showException("Cannot play " + app.files.Base(file) + ". Only MP3, Ogg Vorbis and WAV files can be played")
	case err != nil:
		app.showException(err.Error())
	}
}

// stopMusic is Music → Stop Music (TForm4.StopMusic1Click), which with no
// music open raised the media player's error.
func (mf *mainForm) stopMusic() {
	app := mf.app
	if err := app.music.Stop(); err != nil {
		msg := err.Error()
		if errors.Is(err, audio.ErrNotOpen) {
			msg = "No MCI device open"
		}

		app.call(func() {
			app.showException(msg)
		})
	}
}

// gaugeColors are the HP gauges' colours.
var gaugeColors = map[game.GaugeColor]color.RGBA{
	game.Green:  vclColors["clGreen"],
	game.Yellow: vclColors["clYellow"],
	game.Red:    vclColors["clRed"],
}

// setItems replaces a radio group's items if they changed.
func setRadioItems(rg *radioGroup, items []string) {
	if !slices.Equal(rg.items, items) {
		rg.setItems(slices.Clone(items))
	}
}

// sync shows the game window's state.
func (mf *mainForm) sync() {
	app, gam := mf.app, mf.app.game
	mgw := &gam.Main

	if mf.wdw.caption != mgw.Caption {
		mf.wdw.caption = mgw.Caption
		app.desk.dirty = true
	}

	mf.starterLabel.setVisible(mgw.Starters)
	mf.starters.setVisible(mgw.Starters)
	mf.showStarters(mgw.StarterShiny)

	evolving := len(mgw.Choices) > 0
	mf.evolveChoice.setVisible(evolving)
	mf.evolveButton.setVisible(evolving)

	if evolving {
		setRadioItems(mf.evolveChoice, mgw.Choices)
		mf.evolveChoice.setIndex(mgw.Choice)
	}

	for _, wgt := range mf.picker {
		wgt.base().setVisible(mgw.Picker)
	}

	if mgw.Picker && !mf.pickerShown {
		mf.pickLevel.tb.set(strconv.Itoa(mgw.PickLevel))
	}

	mf.pickerShown = mgw.Picker
	mf.pickCombo.setIndex(mgw.PickSpecies)

	mf.arena.setVisible(mgw.Battle)
	mf.goButton.setVisible(mgw.Go)
	mf.endButton.setVisible(mgw.EndFight)
	mf.searchButton.setVisible(gam.CanSearchAgain())
	mf.attacks.setVisible(mgw.Attacks)
	mf.attacks.setIndex(mgw.AttackIndex)
	mf.pokeball.setVisible(mgw.Pokeball)

	if mgw.Chosen != mf.chosen && mgw.Go {
		// A fighter has come out: Enter attacks from the first turn.
		mf.wdw.setFocus(mf.goButton, false)
		app.desk.activate(mf.wdw)
	}

	mf.chosen = mgw.Chosen

	play := app.preferences.animatedSprites
	mf.enemyAppear.arm(mgw.Appeared, mgw.EnemyImage+1, mgw.Enemy.Shiny, play)
	mf.ownAppear.arm(mgw.Chosen, mgw.Own.Species, mgw.Own.Shiny, play)
	mf.enemySparkle.arm(mgw.Appeared, mgw.Enemy.Shiny, play)
	mf.ownSparkle.arm(mgw.Chosen, mgw.Own.Shiny, play)
	mf.showEnemyImage()
	mf.showOwnImage()

	if enm := mgw.Enemy; enm.Species != 0 {
		name := app.speciesName(enm.Species)
		mf.enemyLabel.setCaption("Enemy " + name + " L." + strconv.Itoa(enm.Level))
		mf.enemyHP.setCaption(strconv.Itoa(enm.HP) + "/" + strconv.Itoa(enm.Level) + "0")
		mf.enemyAttacks.setCaption("Enemy " + name + "'s Attacks:")
		mf.enemyGauge.set(enm.Level*10, enm.HP, gaugeColors[mgw.EnemyColor])
	}

	if enm := mgw.EnemyMoves; enm.Species != 0 {
		sp := gam.Data.SpeciesByID(enm.Species)
		for i, lbl := range mf.enemyMoves {
			if mv := gam.Data.MoveByID(sp.Moves[i]); mv != nil {
				lbl.setCaption(mv.Name)
			}
		}
	}

	own := mgw.Own
	shown := own.Species != 0
	mf.ownLabel.setVisible(shown)
	mf.ownHP.setVisible(shown)
	mf.ownGauge.setVisible(shown)

	if !shown {
		return
	}

	sp := gam.Data.SpeciesByID(own.Species)
	mf.ownLabel.setCaption(sp.Name + " L." + strconv.Itoa(own.Level))
	mf.ownHP.setCaption(strconv.Itoa(own.HP) + "/" + strconv.Itoa(own.Level) + "0")
	mf.ownGauge.set(own.Level*10, own.HP, gaugeColors[mgw.OwnColor])
	mf.attacks.setCaption(sp.Name + "'s attacks:")

	items := make([]string, 0, 5)

	for _, id := range sp.Moves {
		if mv := gam.Data.MoveByID(id); mv != nil {
			items = append(items, mv.Name+" ("+gam.Data.TypeName(mv.Type)+")")
		}
	}

	setRadioItems(mf.attacks, append(items, "Run!"))
}

// showStarters shows each starter shiny or not, as the game rolled it.
func (mf *mainForm) showStarters(shiny [3]bool) {
	for i, shn := range shiny {
		li := &mf.starters.items[i]
		if li.shiny != shn {
			li.shiny = shn

			mf.starters.invalidate()
		}
	}
}

// showEnemyImage shows the enemy's still sprite in ListView3, or none while
// its appearance animation plays there.
func (mf *mainForm) showEnemyImage() {
	mgw := &mf.app.game.Main

	img := mgw.EnemyImage
	if mf.enemyAppear.playing() {
		img = -1
	}

	li := listItem{
		image: img,
		shiny: mgw.Enemy.Shiny,
	}
	mf.enemyImage.setItems([]listItem{li})
}

// showOwnImage shows the player's fighter's still sprite in ListView2, or
// none while there is no fighter or its entry animation plays there.
func (mf *mainForm) showOwnImage() {
	mgw := &mf.app.game.Main

	img := mgw.Own.Species - 1
	if mf.ownAppear.playing() {
		img = -1
	}

	li := listItem{
		image: img,
		shiny: mgw.Own.Shiny,
	}
	mf.ownImage.setItems([]listItem{li})
}

// stopAnimations stops the entry animations, showing the still sprites,
// and puts the stars at rest.
func (mf *mainForm) stopAnimations() {
	mf.enemyAppear.stop()
	mf.ownAppear.stop()
	mf.enemySparkle.rest()
	mf.ownSparkle.rest()
	mf.showEnemyImage()
	mf.showOwnImage()
	mf.arena.invalidate()
}

// animate moves the entry animations on. Each starts once the arena shows
// with no modal window open, and stops if the arena hides first.
func (mf *mainForm) animate(now time.Time) {
	shown := mf.wdw.visible && mf.arena.shown()
	free := mf.app.desk.topModal() == nil

	if mf.enemyAppear.update(now, shown, free) {
		mf.showEnemyImage()
		mf.arena.invalidate()
	}

	if mf.ownAppear.update(now, shown, free) {
		mf.showOwnImage()
		mf.arena.invalidate()
	}

	if mf.enemySparkle.update(now, shown, free) {
		mf.arena.invalidate()
	}

	if mf.ownSparkle.update(now, shown, free) {
		mf.arena.invalidate()
	}
}

// arenaExtra is the width the port adds to the game window, the Battle
// Arena and Battle Info, for HP gauges half as long again as the 84 pixels
// the widest caption leaves in the original's arena.
const arenaExtra int = 42

// widenArena makes the window arenaExtra wider than the original's: the
// Battle Arena, its attack boxes and its gauges grow to the right, the Go!
// and Search Again buttons move with them, and the starter choice stays
// centred. The layout's anchors are still the defaults, so the resize
// moves nothing.
func (mf *mainForm) widenArena() {
	mf.wdw.widen(arenaExtra)

	for _, ctl := range []*control{&mf.arena.control, &mf.attacks.control, &mf.enemyAttacks.control} {
		rect := ctl.rect
		rect.Max.X += arenaExtra
		ctl.setBounds(rect)
	}

	for _, btn := range []*button{mf.goButton, mf.searchButton} {
		btn.setBounds(btn.rect.Add(image.Pt(arenaExtra, 0)))
	}

	for _, ctl := range []*control{&mf.starterLabel.control, &mf.starters.control} {
		ctl.setBounds(ctl.rect.Add(image.Pt(arenaExtra/2, 0)))
	}
}

// placeGauges puts both HP gauges after the widest caption a fighter can
// have, "Enemy " + the longest species name + " L.100": Left = its width
// + 80, and the gauges end with the attack boxes, as the original's do at
// 289. The original put each gauge after its own fighter's caption, so the
// two were seldom the same width.
func (mf *mainForm) placeGauges() {
	gam := mf.app.game

	wid := 0
	for _, sp := range gam.Data.Species {
		wid = max(wid, mf.enemyLabel.font.width("Enemy "+sp.Name+" L.100"))
	}

	box := mf.attacks.rect
	left, right := wid+80, box.Max.X

	for _, gge := range []*gauge{mf.ownGauge, mf.enemyGauge} {
		rect := gge.rect
		gge.setBounds(image.Rect(left, rect.Min.Y, right, rect.Max.Y))
	}
}
