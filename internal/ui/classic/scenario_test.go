// SPDX-License-Identifier: AGPL-3.0-only

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
	ui := func(t *testing.T) scenario.Driver { return &uiDriver{t: t} }
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

func (r dialogs) Message(kind game.DialogKind, yesNo bool, text string) bool {
	yes := r.UI.Message(kind, yesNo, text)
	text = strings.ReplaceAll(text, r.d.ta.dir+string(filepath.Separator), `C:\Temp\`)
	r.d.p.Dialog(kind, text, yesNo, yes)

	return yes
}

// winBase is a Windows path's file name.
func winBase(path string) string { return path[strings.LastIndex(path, `\`)+1:] }

func (d *uiDriver) Launch(p *scenario.Player, file string, save []byte, hook func(*game.Game)) error {
	d.p = p
	dir := d.t.TempDir()

	var args []string

	if save != nil {
		// The probe started the original with the save on its command line.
		path := filepath.Join(dir, winBase(file))
		if err := os.WriteFile(path, save, 0o644); err != nil {
			return err
		}

		args = []string{path}
	}

	d.ta = newTestAppIn(d.t, dir, args)
	d.files = &writes{Files: d.ta.files}
	d.ta.files = d.files
	d.ta.g.UI = dialogs{d.ta.g.UI, d}
	hook(d.ta.g)
	d.ta.wait(80) // the loading screen
	d.settle()

	if save != nil && !d.ta.main.w.visible {
		return errors.New("the save didn't load")
	}

	return nil
}

// seen shows the driver's modal hook a modal window about to be answered.
func (d *uiDriver) seen(w *window) {
	if d.modal != nil {
		d.modal(w)
	}
}

// settle runs frames until the game waits for the player, with no
// handler running but one waiting for a window. The game's message
// dialogs are answered on the way, as the scenario says.
func (d *uiDriver) settle() {
	ta := d.ta
	for range 1000 {
		ta.wait(2)

		top := ta.desk.topModal()
		if top != nil && top.name == "MessageForm" {
			d.seen(top)

			text := top.find("Message").base().caption
			b := "Button1" // OK or Yes

			if !d.p.Answer(top.find("Button2") != nil, text) {
				b = "Button2"
			}

			d.press(top, top.find(b))

			continue
		}

		if top != nil || !ta.running {
			return
		}
	}

	d.t.Fatal("the game is still busy")
}

// front brings w to the front when another window covers p, by clicking
// a part of its caption that the others leave uncovered, as the player
// would.
func (d *uiDriver) front(w *window, p image.Point) {
	desk := d.ta.desk
	if desk.windowAt(p) == w {
		return
	}

	c := w.captionRect()
	y := c.Min.Y + c.Dy()/2

	for x := c.Min.X + 20; x < c.Max.X-3*captionButtonW-8; x++ { // past the icon, short of the buttons
		if q := image.Pt(x, y); desk.windowAt(q) == w {
			d.ta.click(q)

			break
		}
	}

	if desk.windowAt(p) != w {
		d.t.Fatalf("%s stays behind another window", w.name)
	}
}

// clickAt clicks at p in w, brought to the front.
func (d *uiDriver) clickAt(w *window, p image.Point) {
	d.front(w, p)
	d.ta.click(p)
}

// press clicks the middle of control k in w.
func (d *uiDriver) press(w *window, k widget) {
	d.clickAt(w, widgetOrigin(k).Add(k.base().r.Size().Div(2)))
}

// form returns a form's window by its name, or nil.
func (d *uiDriver) form(name string) *window {
	ta := d.ta

	return map[string]*window{
		"Form1": ta.start.w, "Form4": ta.main.w, "Form5": ta.stats.w, "Form7": ta.info.w,
		"Form8": ta.dex.w, "Form9": ta.mart.w, "FormA": ta.types.w, "FormB": ta.corner.w,
	}[name]
}

// control returns a control by form and name: "Form9.Button1".
func (d *uiDriver) control(path string) (*window, widget, error) {
	form, name, _ := strings.Cut(path, ".")
	w := d.form(form)

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

func (d *uiDriver) Menu(path []*game.MenuItem) error {
	ta := d.ta
	m := ta.main.w.menu

	for i, e := range m.entries {
		if e.name() == path[0].Name {
			r := m.barRects()[i]
			d.front(ta.main.w, r.Min.Add(r.Size().Div(2)))
		}
	}

	names := make([]string, len(path))
	for i, it := range path {
		names[i] = it.Name
	}

	ta.menu(names...)
	d.settle()

	return nil
}

func (d *uiDriver) Choose(i int) error {
	ta := d.ta
	l := ta.stats.party
	r := l.itemRect(i)
	p := widgetOrigin(l).Add(r.Min).Add(r.Size().Div(2))
	d.front(ta.stats.w, p)
	ta.doubleClick(p)
	d.settle()

	return nil
}

// radio clicks radio button k of g.
func (d *uiDriver) radio(w *window, g *radioGroup, k int) {
	r := g.buttonRect(k)
	d.clickAt(w, widgetOrigin(g).Add(r.Min).Add(image.Pt(6, r.Dy()/2)))
}

func (d *uiDriver) Attack(k int) error {
	m := d.ta.main
	d.radio(m.w, m.attacks, k)
	d.press(m.w, m.goButton)
	d.settle()

	return nil
}

func (d *uiDriver) Item(k int) error {
	s := d.ta.stats
	d.press(s.w, s.items[k])
	d.settle()

	return nil
}

func (d *uiDriver) End() error {
	m := d.ta.main
	d.press(m.w, m.endButton)
	d.settle()

	return nil
}

func (d *uiDriver) Click(path string) error {
	w, k, err := d.control(path)
	if err != nil {
		return err
	}

	d.press(w, k)
	d.settle()

	if path == "Form1.AboutButton" {
		about := d.ta.desk.topModal()
		if about != d.ta.about {
			return errors.New("no About box")
		}

		d.seen(about)
		d.press(about, about.find("OKButton"))
		d.settle()
	}

	return nil
}

func (d *uiDriver) Radio(path string, k int) error {
	w, c, err := d.control(path)
	if err != nil {
		return err
	}

	g, ok := c.(*radioGroup)
	if !ok {
		return fmt.Errorf("%s isn't a radio group", path)
	}

	d.radio(w, g, k)
	d.settle()

	return nil
}

func (d *uiDriver) Combo(path string, i int) error {
	w, k, err := d.control(path)
	if err != nil {
		return err
	}

	c, ok := k.(*comboBox)
	if !ok {
		return fmt.Errorf("%s isn't a combo box", path)
	}

	b := c.button()
	d.clickAt(w, widgetOrigin(c).Add(b.Min).Add(b.Size().Div(2)))

	l := c.dropped
	if l == nil {
		return errors.New("the list didn't drop down")
	}

	for i < l.top || i >= l.top+l.rows {
		dy := -1 // down
		if i < l.top {
			dy = 1
		}

		d.ta.frame(input{wheel: dy})
	}

	d.ta.click(l.r.Min.Add(image.Pt(5, 1+(i-l.top)*c.itemHeight+c.itemHeight/2)))
	d.settle()

	if c.index != i {
		return fmt.Errorf("%s chose %d", path, c.index)
	}

	return nil
}

func (d *uiDriver) Type(path string, n int) error {
	w, k, err := d.control(path)
	if err != nil {
		return err
	}

	s, ok := k.(*spinEdit)
	if !ok {
		return fmt.Errorf("%s isn't a spin edit", path)
	}

	ta := d.ta
	d.clickAt(w, widgetOrigin(s).Add(image.Pt(5, s.r.Dy()/2)))
	ta.key(ebiten.KeyHome)
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyEnd}, mods: mods{shift: true}})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyEnd}})
	ta.typeText(strconv.Itoa(n))
	d.settle()

	return nil
}

func (d *uiDriver) Text(path string) (string, error) {
	switch path {
	case "Form4":
		return d.ta.main.w.caption, nil
	case "Form9.Edit1":
		return d.ta.mart.total.text(), nil
	}

	return "", fmt.Errorf("no text for %s", path)
}

// fileDialog fills in the file dialog that shows with name.
func (d *uiDriver) fileDialog(caption, name string) error {
	ta := d.ta
	if top := ta.desk.topModal(); top == nil || top.caption != caption {
		return fmt.Errorf("no %s dialog: %v", caption, top)
	}

	ta.typeText(name)
	ta.key(ebiten.KeyEnter)
	d.settle()

	return nil
}

func (d *uiDriver) New(file, name string) error {
	ta := d.ta
	d.press(ta.start.w, ta.start.w.find("NewButton"))
	d.settle()

	if err := d.fileDialog("Save As", winBase(file)); err != nil {
		return err
	}

	if ta.desk.topModal() != ta.name.w {
		return errors.New("no name dialog")
	}

	d.seen(ta.name.w)
	ta.typeText(name) // over "Ash", all selected
	ta.key(ebiten.KeyEnter)
	d.settle()

	return nil
}

func (d *uiDriver) Load(file string, save []byte) error {
	ta := d.ta
	if err := os.WriteFile(filepath.Join(ta.dir, winBase(file)), save, 0o644); err != nil {
		return err
	}

	d.press(ta.start.w, ta.start.w.find("LoadButton"))
	d.settle()

	return d.fileDialog("Open", winBase(file))
}

func (d *uiDriver) Starter(k int) error {
	m := d.ta.main
	r := m.starters.itemRect(k)
	d.clickAt(m.w, widgetOrigin(m.starters).Add(r.Min).Add(r.Size().Div(2)))
	d.settle()

	return nil
}

func (d *uiDriver) Export(file string) error {
	if err := d.Menu([]*game.MenuItem{d.ta.g.Item("File1"), d.ta.g.Item("ExportPOKMONinformation1")}); err != nil {
		return err
	}

	return d.fileDialog("Save As", winBase(file))
}

func (d *uiDriver) Quit() ([]byte, error) {
	ta := d.ta
	d.files.path, d.files.data = "", nil
	b := ta.main.w.captionButton(capClose)
	d.clickAt(ta.main.w, b.Min.Add(b.Size().Div(2)))
	d.settle()

	if !ta.quit {
		return nil, errors.New("the game didn't end")
	}

	if d.files.path != ta.g.SaveFile {
		return nil, nil
	}

	return d.files.data, nil
}
