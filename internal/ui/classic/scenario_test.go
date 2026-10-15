// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/game/scenario"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
)

// TestScenarios plays the traces of the original in
// internal/game/testdata/scenarios on the classic user interface: each
// action with the mouse and the keyboard, as a player would, and each of
// the game's dialogs answered with its buttons. The traces must come out
// as the original's, as they do when the game's handlers are called
// directly.
func TestScenarios(t *testing.T) {
	ui := func(t *testing.T) scenario.Driver {
		return &uiDriver{t: t}
	}
	scenario.Run(t, filepath.Join("..", "..", "game", "testdata", "scenarios"), testData(t), ui, "")
}

// uiDriver does a scenario's actions with the classic user interface's
// controls.
type uiDriver struct {
	t     *testing.T
	ta    *testApp
	p     *scenario.Player
	files *writes
	modal func(w *window) // if set, sees each of the game's modal windows before it is answered
}

// writes keeps the last file written: the save, on quitting.
type writes struct {
	platform.Files
	path string
	data []byte
}

func (w *writes) WriteFile(path string, data []byte) error {
	w.path, w.data = path, data

	return w.Files.WriteFile(path, data)
}

// dialogs records the game's dialogs once the player has answered them,
// with the original's temporary folder for the game's.
type dialogs struct {
	game.UI
	d *uiDriver
}

func (d dialogs) Message(kind game.DialogKind, yesNo bool, text string) bool {
	yes := d.UI.Message(kind, yesNo, text)
	text = strings.ReplaceAll(text, d.d.ta.dir+string(filepath.Separator), `C:\Temp\`)
	d.d.p.Dialog(kind, text, yesNo, yes)

	return yes
}

// winBase is a Windows path's file name.
func winBase(path string) string {
	return path[strings.LastIndex(path, `\`)+1:]
}

func (ud *uiDriver) Launch(p *scenario.Player, file string, save []byte, hook func(*game.Game)) error {
	ud.p = p
	dir := ud.t.TempDir()

	var args []string

	if save != nil {
		// The probe started the original with the save on its command line.
		path := filepath.Join(dir, winBase(file))
		if err := os.WriteFile(path, save, 0o600); err != nil {
			return err
		}

		args = []string{path}
	}

	ta := newTestAppIn(ud.t, dir, args)
	ud.ta = ta
	ud.files = &writes{Files: ta.files}
	ta.files = ud.files
	ta.g.UI = dialogs{ta.g.UI, ud}
	hook(ta.g)
	ta.wait(80) // the loading screen
	ud.settle()

	if save != nil && !ta.main.w.visible {
		return errors.New("the save didn't load")
	}

	return nil
}

// seen shows the driver's modal hook a modal window about to be answered.
func (ud *uiDriver) seen(w *window) {
	if ud.modal != nil {
		ud.modal(w)
	}
}

// settle runs frames until the game waits for the player, with no
// handler running but one waiting for a window. The game's message
// dialogs are answered on the way, as the scenario says.
func (ud *uiDriver) settle() {
	ta := ud.ta
	for range 1000 {
		ta.wait(2)

		top := ta.desk.topModal()
		if top != nil && top.name == "MessageForm" {
			ud.seen(top)

			text := top.find("Message").base().caption
			b := "Button1" // OK or Yes

			if !ud.p.Answer(top.find("Button2") != nil, text) {
				b = "Button2"
			}

			ud.press(top, top.find(b))

			continue
		}

		if top != nil || !ta.running {
			return
		}
	}

	ud.t.Fatal("the game is still busy")
}

// front brings w to the front when another window covers p, by clicking
// a part of its caption that the others leave uncovered, as the player
// would.
func (ud *uiDriver) front(w *window, p image.Point) {
	desk := ud.ta.desk
	if desk.windowAt(p) == w {
		return
	}

	c := w.captionRect()
	y := c.Min.Y + c.Dy()/2

	for x := c.Min.X + 20; x < c.Max.X-3*captionButtonW-8; x++ { // past the icon, short of the buttons
		if q := image.Pt(x, y); desk.windowAt(q) == w {
			ud.ta.click(q)

			break
		}
	}

	if desk.windowAt(p) != w {
		ud.t.Fatalf("%s stays behind another window", w.name)
	}
}

// clickAt clicks at p in w, brought to the front.
func (ud *uiDriver) clickAt(w *window, p image.Point) {
	ud.front(w, p)
	ud.ta.click(p)
}

// press clicks the middle of control k in w.
func (ud *uiDriver) press(w *window, k widget) {
	size := k.base().rect.Size()
	ud.clickAt(w, widgetOrigin(k).Add(size.Div(2)))
}

// form returns a form's window by its name, or nil.
func (ud *uiDriver) form(name string) *window {
	ta := ud.ta

	return map[string]*window{
		"Form1": ta.start.w,
		"Form4": ta.main.w,
		"Form5": ta.stats.wdw,
		"Form7": ta.info.wdw,
		"Form8": ta.pokedex.wdw,
		"Form9": ta.pokemart.wdw,
		"FormA": ta.types.wdw,
		"FormB": ta.gameCorner.wdw,
	}[name]
}

// control returns a control by form and name: "Form9.Button1".
func (ud *uiDriver) control(path string) (*window, widget, error) {
	form, name, _ := strings.Cut(path, ".")
	w := ud.form(form)

	if w == nil {
		return nil, nil, fmt.Errorf("unknown form %s", form)
	}

	k := w.find(name)
	if k == nil {
		return nil, nil, fmt.Errorf("%s has no %s", form, name)
	}

	if !w.visible || !k.base().shown() {
		return nil, nil, fmt.Errorf("%s isn't shown", path)
	}

	return w, k, nil
}

func (ud *uiDriver) Menu(path []*game.MenuItem) error {
	ta := ud.ta
	m := ta.main.w.menu

	for i, e := range m.entries {
		if e.name() == path[0].Name {
			r := m.barRects()[i]
			ud.front(ta.main.w, r.Min.Add(r.Size().Div(2)))
		}
	}

	names := make([]string, len(path))
	for i, it := range path {
		names[i] = it.Name
	}

	ta.menu(names...)
	ud.settle()

	return nil
}

func (ud *uiDriver) Choose(i int) error {
	ta := ud.ta
	l := ta.stats.party
	r := l.itemRect(i)
	p := widgetOrigin(l).Add(r.Min).Add(r.Size().Div(2))
	ud.front(ta.stats.wdw, p)
	ta.doubleClick(p)
	ud.settle()

	return nil
}

// radio clicks radio button k of g.
func (ud *uiDriver) radio(w *window, g *radioGroup, k int) {
	r := g.buttonRect(k)
	ud.clickAt(w, widgetOrigin(g).Add(r.Min).Add(image.Pt(6, r.Dy()/2)))
}

func (ud *uiDriver) Attack(k int) error {
	m := ud.ta.main
	ud.radio(m.w, m.attacks, k)
	ud.press(m.w, m.goButton)
	ud.settle()

	return nil
}

func (ud *uiDriver) Item(k int) error {
	s := ud.ta.stats
	ud.press(s.wdw, s.items[k])
	ud.settle()

	return nil
}

func (ud *uiDriver) End() error {
	m := ud.ta.main
	ud.press(m.w, m.endButton)
	ud.settle()

	return nil
}

func (ud *uiDriver) Click(path string) error {
	w, k, err := ud.control(path)
	if err != nil {
		return err
	}

	ud.press(w, k)
	ud.settle()

	if path == "Form1.AboutButton" {
		about := ud.ta.desk.topModal()
		if about != ud.ta.about {
			return errors.New("no About box")
		}

		ud.seen(about)
		ud.press(about, about.find("OKButton"))
		ud.settle()
	}

	return nil
}

func (ud *uiDriver) Radio(path string, k int) error {
	w, c, err := ud.control(path)
	if err != nil {
		return err
	}

	g, ok := c.(*radioGroup)
	if !ok {
		return fmt.Errorf("%s isn't a radio group", path)
	}

	ud.radio(w, g, k)
	ud.settle()

	return nil
}

func (ud *uiDriver) Combo(path string, i int) error {
	w, k, err := ud.control(path)
	if err != nil {
		return err
	}

	c, ok := k.(*comboBox)
	if !ok {
		return fmt.Errorf("%s isn't a combo box", path)
	}

	b := c.button()
	ud.clickAt(w, widgetOrigin(c).Add(b.Min).Add(b.Size().Div(2)))

	l := c.dropped
	if l == nil {
		return errors.New("the list didn't drop down")
	}

	for i < l.top || i >= l.top+l.rows {
		dy := -1 // down
		if i < l.top {
			dy = 1
		}

		ud.ta.frame(input{wheel: dy})
	}

	ud.ta.click(l.rect.Min.Add(image.Pt(5, 1+(i-l.top)*c.itemHeight+c.itemHeight/2)))
	ud.settle()

	if c.index != i {
		return fmt.Errorf("%s chose %d", path, c.index)
	}

	return nil
}

func (ud *uiDriver) Type(path string, n int) error {
	w, k, err := ud.control(path)
	if err != nil {
		return err
	}

	s, ok := k.(*spinEdit)
	if !ok {
		return fmt.Errorf("%s isn't a spin edit", path)
	}

	ta := ud.ta
	ud.clickAt(w, widgetOrigin(s).Add(image.Pt(5, s.rect.Dy()/2)))
	ta.key(ebiten.KeyHome)
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyEnd}, mods: mods{shift: true}})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyEnd}})
	ta.typeText(strconv.Itoa(n))
	ud.settle()

	return nil
}

func (ud *uiDriver) Text(path string) (string, error) {
	ta := ud.ta

	switch path {
	case "Form4":
		return ta.main.w.caption, nil
	case "Form9.Edit1":
		return ta.pokemart.total.text(), nil
	}

	return "", fmt.Errorf("no text for %s", path)
}

// fileDialog fills in the file dialog that shows with name.
func (ud *uiDriver) fileDialog(caption, name string) error {
	ta := ud.ta
	if top := ta.desk.topModal(); top == nil || top.caption != caption {
		return fmt.Errorf("no %s dialog: %v", caption, top)
	}

	ta.typeText(name)
	ta.key(ebiten.KeyEnter)
	ud.settle()

	return nil
}

func (ud *uiDriver) New(file, name string) error {
	ta := ud.ta
	ud.press(ta.start.w, ta.start.w.find("NewButton"))
	ud.settle()

	if err := ud.fileDialog("Save As", winBase(file)); err != nil {
		return err
	}

	if ta.desk.topModal() != ta.name.w {
		return errors.New("no name dialog")
	}

	ud.seen(ta.name.w)
	ta.typeText(name) // over "Ash", all selected
	ta.key(ebiten.KeyEnter)
	ud.settle()

	return nil
}

func (ud *uiDriver) Load(file string, save []byte) error {
	ta := ud.ta
	if err := os.WriteFile(filepath.Join(ta.dir, winBase(file)), save, 0o600); err != nil {
		return err
	}

	ud.press(ta.start.w, ta.start.w.find("LoadButton"))
	ud.settle()

	return ud.fileDialog("Open", winBase(file))
}

func (ud *uiDriver) Starter(k int) error {
	m := ud.ta.main
	r := m.starters.itemRect(k)
	ud.clickAt(m.w, widgetOrigin(m.starters).Add(r.Min).Add(r.Size().Div(2)))
	ud.settle()

	return nil
}

func (ud *uiDriver) Export(file string) error {
	if err := ud.Menu([]*game.MenuItem{
		ud.ta.g.Item("File1"),
		ud.ta.g.Item("ExportPOKMONinformation1"),
	}); err != nil {
		return err
	}

	return ud.fileDialog("Save As", winBase(file))
}

func (ud *uiDriver) Quit() ([]byte, error) {
	ta := ud.ta
	ud.files.path, ud.files.data = "", nil
	b := ta.main.w.captionButton(capClose)
	ud.clickAt(ta.main.w, b.Min.Add(b.Size().Div(2)))
	ud.settle()

	if !ta.quit {
		return nil, errors.New("the game didn't end")
	}

	if ud.files.path != ta.g.SaveFile {
		return nil, nil
	}

	return ud.files.data, nil
}
