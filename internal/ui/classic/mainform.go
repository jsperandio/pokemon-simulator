// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"errors"
	"image"
	"time"

	"github.com/jsperandio/pokemon-simulator/internal/audio"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// mainForm is the game window (TForm4): the menu, the starter choice,
// the battle arena, the evolution choice and the Mewtwo-power picker.
type mainForm struct {
	a *App
	w *window

	starterLabel *label
	starters     *listView

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
	ownImage       *listView // ListView2
	enemyImage     *listView // ListView3
	endButton      *button
}

func newMainForm(a *App) *mainForm {
	w := newWindow(a.desk, &layout.Form4)
	f := &mainForm{a: a, w: w}
	f.starterLabel = get[*label](w, "Label1")
	f.starters = get[*listView](w, "ListView1")
	f.evolveChoice = get[*radioGroup](w, "EvolveChoice")
	f.evolveButton = get[*button](w, "EvolveButton")
	f.pickCombo = get[*comboBox](w, "ComboBox1")
	f.pickLevel = get[*spinEdit](w, "SpinEdit1")

	for _, n := range []string{"Label10", "Label11", "ComboBox1", "SpinEdit1", "Button2", "Button3", "Button4"} {
		f.picker = append(f.picker, w.find(n))
	}

	f.arena = get[*groupBox](w, "GroupBox1")
	f.ownLabel = get[*label](w, "Label2")
	f.enemyLabel = get[*label](w, "Label3")
	f.ownHP = get[*label](w, "Label8")
	f.enemyHP = get[*label](w, "Label9")
	f.ownGauge = get[*gauge](w, "Gauge1")
	f.enemyGauge = get[*gauge](w, "Gauge2")
	f.pokeball = get[*picture](w, "Image1")
	f.attacks = get[*radioGroup](w, "RadioGroup1")
	f.enemyAttacks = get[*groupBox](w, "GroupBox2")

	for i := range f.enemyMoves {
		f.enemyMoves[i] = get[*label](w, "Label"+itoa(4+i))
	}

	f.goButton = get[*button](w, "Button1")
	f.ownImage = get[*listView](w, "ListView2")
	f.enemyImage = get[*listView](w, "ListView3")
	f.endButton = get[*button](w, "EndButton")

	g := a.g
	f.starters.onClick = f.starterClick
	f.evolveChoice.onClick = func() {
		g.ChooseEvolution(f.evolveChoice.index)
	}
	f.evolveButton.onClick = func() {
		a.call(g.Evolve)
	}

	f.pickCombo.items = a.dexItems() // FormShow
	f.pickCombo.onSelect = func(i int) {
		g.PickSpecies(i)
	}
	f.pickLevel.onChangeValue = func(v int) {
		g.PickLevel(v)
	}
	get[*button](w, "Button2").onClick = func() {
		a.call(g.PickGo)
	}
	get[*button](w, "Button3").onClick = func() {
		a.call(g.PickCancel)
	}
	get[*button](w, "Button4").onClick = func() {
		a.call(g.PickRandom)
	}
	f.attacks.onClick = func() {
		g.Main.AttackIndex = f.attacks.index
	}
	f.goButton.onClick = func() {
		a.call(g.Go)
	}
	f.endButton.onClick = func() {
		a.call(g.EndFight)
	}
	w.onClose = f.formClose

	m := newMenuBar(w, g.Menu, layout.Form4.Find("MainMenu1"))
	m.onCommand = f.menuCommand

	a.desk.add(w)

	return f
}

// starterClick is a click on the starter list (TForm4.ListView1Click).
func (f *mainForm) starterClick() {
	k := f.starters.selected
	if k < 0 {
		return
	}

	a := f.a
	a.call(func() {
		a.g.ChooseStarter(k)
	})
}

// menuCommand runs a menu item: the user interface's own, or the game's.
func (f *mainForm) menuCommand(e *menuEntry) {
	a := f.a

	switch e.name() {
	case "ExportPOKMONinformation1":
		a.call(f.export)
	case "Quit1":
		f.formClose()
	case "Openfile1":
		a.call(f.openMusic)
	case "StopMusic1":
		f.stopMusic()
	case "Contents1":
		a.help()
	case "About1":
		a.showAbout()
	default:
		a.call(func() {
			a.g.ClickMenu(e.name())
		})
	}
}

// formClose is closing the game window (TForm4.FormClose): a tournament's
// penalty, then the save, then the end of the program.
func (f *mainForm) formClose() {
	a := f.a
	a.call(func() {
		if b := a.g.Close(); b != nil {
			if err := a.files.WriteFile(a.g.SaveFile, b); err != nil {
				a.Message(game.Error, false, err.Error())
			}
		}

		a.exit()
	})
}

// export is File → Export POKéMON information...: the page, written where
// the player chooses, then shown if wanted.
func (f *mainForm) export() {
	a := f.a
	file, ok := a.openFile(fileOptions{save: true, filter: "HTML Files (*.htm)|*.htm", defExt: "htm", overwrite: true})

	if !ok {
		return
	}

	page := a.g.ExportPage(time.Now())
	if err := a.files.WriteFile(file, page); err != nil {
		a.Message(game.Error, false, err.Error())

		return
	}

	if a.g.Exported(file) {
		if err := platform.OpenPage(file, page); err != nil {
			a.Message(game.Error, false, err.Error())
		}
	}
}

// musicPlayer is what the Music menu plays music with: an audio.Music.
type musicPlayer interface {
	Open(name string, data []byte) error
	Stop() error
	Update(repeat bool)
	Playing() bool
}

// musicFilter is Music → Open file...'s. The original's offered what
// Windows Media Player played; these are what the port plays.
const musicFilter string = "All supported formats (*.mp3;*.ogg;*.wav)|*.mp3;*.ogg;*.wav|" +
	"MPEG Audio Files (*.mp3)|*.mp3|Ogg Vorbis Files (*.ogg)|*.ogg|Wave Files (*.wav)|*.wav|All files (*.*)|*.*"

// openMusic is Music → Open file... (TForm4.Openfile1Click): the music
// chosen plays at once.
func (f *mainForm) openMusic() {
	a := f.a
	file, ok := a.openFile(fileOptions{filter: musicFilter, mustExist: true})

	if !ok {
		return
	}

	b, err := a.files.ReadFile(file)
	if err == nil {
		err = a.music.Open(a.files.Base(file), b)
	}

	switch {
	case errors.Is(err, audio.ErrFormat):
		a.showException("Cannot play " + a.files.Base(file) + ". Only MP3, Ogg Vorbis and WAV files can be played")
	case err != nil:
		a.showException(err.Error())
	}
}

// stopMusic is Music → Stop Music (TForm4.StopMusic1Click), which with no
// music open raised the media player's error.
func (f *mainForm) stopMusic() {
	a := f.a
	if err := a.music.Stop(); err != nil {
		msg := err.Error()
		if errors.Is(err, audio.ErrNotOpen) {
			msg = "No MCI device open"
		}

		a.call(func() {
			a.showException(msg)
		})
	}
}

// sync shows the game window's state.
func (f *mainForm) sync() {
	a, g := f.a, f.a.g
	m := &g.Main

	if f.w.caption != m.Caption {
		f.w.caption = m.Caption
		a.desk.dirty = true
	}

	f.starterLabel.setVisible(m.Starters)
	f.starters.setVisible(m.Starters)

	evolving := len(m.Choices) > 0
	f.evolveChoice.setVisible(evolving)
	f.evolveButton.setVisible(evolving)

	if evolving {
		setRadioItems(f.evolveChoice, m.Choices)
		f.evolveChoice.setIndex(m.Choice)
	}

	for _, k := range f.picker {
		k.base().setVisible(m.Picker)
	}

	if m.Picker && !f.pickerShown {
		f.pickLevel.tb.set(itoa(m.PickLevel))
	}

	f.pickerShown = m.Picker
	f.pickCombo.setIndex(m.PickSpecies)

	f.arena.setVisible(m.Battle)
	f.goButton.setVisible(m.Go)
	f.endButton.setVisible(m.EndFight)
	f.attacks.setVisible(m.Attacks)
	f.attacks.setIndex(m.AttackIndex)
	f.pokeball.setVisible(m.Pokeball)
	f.enemyImage.setItems([]listItem{{"", m.EnemyImage}})

	if e := m.Enemy; e.Species != 0 {
		name := a.speciesName(e.Species)
		f.enemyLabel.setCaption("Enemy " + name + " L." + itoa(e.Level))
		f.enemyHP.setCaption(itoa(e.HP) + "/" + itoa(e.Level) + "0")
		f.enemyAttacks.setCaption("Enemy " + name + "'s Attacks:")
		f.enemyGauge.set(e.Level*10, e.HP, gaugeColors[m.EnemyColor])
	}

	if e := m.EnemyMoves; e.Species != 0 {
		sp := g.Data.SpeciesByID(e.Species)
		for i, l := range f.enemyMoves {
			if mv := g.Data.MoveByID(sp.Moves[i]); mv != nil {
				l.setCaption(mv.Name)
			}
		}

		// Gauge2 sits after Label3, as that fighter's caption sized it.
		f.placeGauge(f.enemyGauge, "Enemy "+sp.Name+" L."+itoa(e.Level))
	}

	o := m.Own
	shown := o.Species != 0
	f.ownLabel.setVisible(shown)
	f.ownHP.setVisible(shown)
	f.ownGauge.setVisible(shown)

	if !shown {
		f.ownImage.setItems([]listItem{{"", -1}})

		return
	}

	sp := g.Data.SpeciesByID(o.Species)
	f.ownImage.setItems([]listItem{{"", o.Species - 1}})
	f.ownLabel.setCaption(sp.Name + " L." + itoa(o.Level))
	f.ownHP.setCaption(itoa(o.HP) + "/" + itoa(o.Level) + "0")
	f.ownGauge.set(o.Level*10, o.HP, gaugeColors[m.OwnColor])
	f.placeGauge(f.ownGauge, sp.Name+" L."+itoa(o.Level))
	f.attacks.setCaption(sp.Name + "'s attacks:")

	items := make([]string, 0, 5)

	for _, id := range sp.Moves {
		if mv := g.Data.MoveByID(id); mv != nil {
			items = append(items, mv.Name+" ("+g.Data.TypeName(mv.Type)+")")
		}
	}

	setRadioItems(f.attacks, append(items, "Run!"))
}

// placeGauge puts a gauge after its label's caption: Left = the label's
// width + 80, and Width = 289 − Left.
func (f *mainForm) placeGauge(g *gauge, caption string) {
	left := f.enemyLabel.font.width(caption) + 80
	r := g.r
	g.setBounds(image.Rect(left, r.Min.Y, 289, r.Max.Y))
}
