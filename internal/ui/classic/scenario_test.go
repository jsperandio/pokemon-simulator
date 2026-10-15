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
	ply   *scenario.Player
	files *writes
	modal func(wdw *window) // if set, sees each of the game's modal windows before it is answered
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
	drv *uiDriver
}

func (d dialogs) Message(kind game.DialogKind, yesNo bool, text string) bool {
	yes := d.UI.Message(kind, yesNo, text)
	text = strings.ReplaceAll(text, d.drv.ta.dir+string(filepath.Separator), `C:\Temp\`)
	d.drv.ply.Dialog(kind, text, yesNo, yes)

	return yes
}

// winBase is a Windows path's file name.
func winBase(path string) string {
	return path[strings.LastIndex(path, `\`)+1:]
}

func (ud *uiDriver) Launch(ply *scenario.Player, file string, save []byte, hook func(*game.Game)) error {
	ud.ply = ply
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
	ta.gam.UI = dialogs{ta.gam.UI, ud}
	hook(ta.gam)
	ta.wait(80) // the loading screen
	ud.settle()

	if save != nil && !ta.main.wdw.visible {
		return errors.New("the save didn't load")
	}

	return nil
}

// seen shows the driver's modal hook a modal window about to be answered.
func (ud *uiDriver) seen(wdw *window) {
	if ud.modal != nil {
		ud.modal(wdw)
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
			nam := "Button1" // OK or Yes

			if !ud.ply.Answer(top.find("Button2") != nil, text) {
				nam = "Button2"
			}

			ud.press(top, top.find(nam))

			continue
		}

		if top != nil || !ta.running {
			return
		}
	}

	ud.t.Fatal("the game is still busy")
}

// front brings wdw to the front when another window covers pnt, by clicking
// a part of its caption that the others leave uncovered, as the player
// would.
func (ud *uiDriver) front(wdw *window, pnt image.Point) {
	desk := ud.ta.desk
	if desk.windowAt(pnt) == wdw {
		return
	}

	rect := wdw.captionRect()
	y := rect.Min.Y + rect.Dy()/2

	for x := rect.Min.X + 20; x < rect.Max.X-3*captionButtonW-8; x++ { // past the icon, short of the buttons
		if cur := image.Pt(x, y); desk.windowAt(cur) == wdw {
			ud.ta.click(cur)

			break
		}
	}

	if desk.windowAt(pnt) != wdw {
		ud.t.Fatalf("%s stays behind another window", wdw.name)
	}
}

// clickAt clicks at pnt in wdw, brought to the front.
func (ud *uiDriver) clickAt(wdw *window, pnt image.Point) {
	ud.front(wdw, pnt)
	ud.ta.click(pnt)
}

// press clicks the middle of control wgt in wdw.
func (ud *uiDriver) press(wdw *window, wgt widget) {
	size := wgt.base().rect.Size()
	ud.clickAt(wdw, widgetOrigin(wgt).Add(size.Div(2)))
}

// form returns a form's window by its name, or nil.
func (ud *uiDriver) form(name string) *window {
	ta := ud.ta

	return map[string]*window{
		"Form1": ta.start.wdw,
		"Form4": ta.main.wdw,
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
	wdw := ud.form(form)

	if wdw == nil {
		return nil, nil, fmt.Errorf("unknown form %s", form)
	}

	wgt := wdw.find(name)
	if wgt == nil {
		return nil, nil, fmt.Errorf("%s has no %s", form, name)
	}

	if !wdw.visible || !wgt.base().shown() {
		return nil, nil, fmt.Errorf("%s isn't shown", path)
	}

	return wdw, wgt, nil
}

func (ud *uiDriver) Menu(path []*game.MenuItem) error {
	ta := ud.ta
	mb := ta.main.wdw.menu

	for i, me := range mb.entries {
		if me.name() == path[0].Name {
			rect := mb.barRects()[i]
			ud.front(ta.main.wdw, rect.Min.Add(rect.Size().Div(2)))
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
	lv := ta.stats.party
	rect := lv.itemRect(i)
	pnt := widgetOrigin(lv).Add(rect.Min).Add(rect.Size().Div(2))
	ud.front(ta.stats.wdw, pnt)
	ta.doubleClick(pnt)
	ud.settle()

	return nil
}

// radio clicks radio button idx of rg.
func (ud *uiDriver) radio(wdw *window, rg *radioGroup, idx int) {
	rect := rg.buttonRect(idx)
	ud.clickAt(wdw, widgetOrigin(rg).Add(rect.Min).Add(image.Pt(6, rect.Dy()/2)))
}

func (ud *uiDriver) Attack(idx int) error {
	mf := ud.ta.main
	ud.radio(mf.wdw, mf.attacks, idx)
	ud.press(mf.wdw, mf.goButton)
	ud.settle()

	return nil
}

func (ud *uiDriver) Item(idx int) error {
	stf := ud.ta.stats
	ud.press(stf.wdw, stf.items[idx])
	ud.settle()

	return nil
}

func (ud *uiDriver) End() error {
	mf := ud.ta.main
	ud.press(mf.wdw, mf.endButton)
	ud.settle()

	return nil
}

func (ud *uiDriver) Click(path string) error {
	wdw, wgt, err := ud.control(path)
	if err != nil {
		return err
	}

	ud.press(wdw, wgt)
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

func (ud *uiDriver) Radio(path string, idx int) error {
	wdw, wgt, err := ud.control(path)
	if err != nil {
		return err
	}

	rg, ok := wgt.(*radioGroup)
	if !ok {
		return fmt.Errorf("%s isn't a radio group", path)
	}

	ud.radio(wdw, rg, idx)
	ud.settle()

	return nil
}

func (ud *uiDriver) Combo(path string, i int) error {
	wdw, wgt, err := ud.control(path)
	if err != nil {
		return err
	}

	cb, ok := wgt.(*comboBox)
	if !ok {
		return fmt.Errorf("%s isn't a combo box", path)
	}

	rect := cb.button()
	ud.clickAt(wdw, widgetOrigin(cb).Add(rect.Min).Add(rect.Size().Div(2)))

	dl := cb.dropped
	if dl == nil {
		return errors.New("the list didn't drop down")
	}

	for i < dl.top || i >= dl.top+dl.rows {
		dy := -1 // down
		if i < dl.top {
			dy = 1
		}

		ud.ta.frame(input{wheel: dy})
	}

	ud.ta.click(dl.rect.Min.Add(image.Pt(5, 1+(i-dl.top)*cb.itemHeight+cb.itemHeight/2)))
	ud.settle()

	if cb.index != i {
		return fmt.Errorf("%s chose %d", path, cb.index)
	}

	return nil
}

func (ud *uiDriver) Type(path string, n int) error {
	wdw, wgt, err := ud.control(path)
	if err != nil {
		return err
	}

	se, ok := wgt.(*spinEdit)
	if !ok {
		return fmt.Errorf("%s isn't a spin edit", path)
	}

	ta := ud.ta
	ud.clickAt(wdw, widgetOrigin(se).Add(image.Pt(5, se.rect.Dy()/2)))
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
		return ta.main.wdw.caption, nil
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
	ud.press(ta.start.wdw, ta.start.wdw.find("NewButton"))
	ud.settle()

	if err := ud.fileDialog("Save As", winBase(file)); err != nil {
		return err
	}

	if ta.desk.topModal() != ta.name.wdw {
		return errors.New("no name dialog")
	}

	ud.seen(ta.name.wdw)
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

	ud.press(ta.start.wdw, ta.start.wdw.find("LoadButton"))
	ud.settle()

	return ud.fileDialog("Open", winBase(file))
}

func (ud *uiDriver) Starter(idx int) error {
	mf := ud.ta.main
	rect := mf.starters.itemRect(idx)
	ud.clickAt(mf.wdw, widgetOrigin(mf.starters).Add(rect.Min).Add(rect.Size().Div(2)))
	ud.settle()

	return nil
}

func (ud *uiDriver) Export(file string) error {
	if err := ud.Menu([]*game.MenuItem{
		ud.ta.gam.Item("File1"),
		ud.ta.gam.Item("ExportPOKMONinformation1"),
	}); err != nil {
		return err
	}

	return ud.fileDialog("Save As", winBase(file))
}

func (ud *uiDriver) Quit() ([]byte, error) {
	ta := ud.ta
	ud.files.path, ud.files.data = "", nil
	rect := ta.main.wdw.captionButton(capClose)
	ud.clickAt(ta.main.wdw, rect.Min.Add(rect.Size().Div(2)))
	ud.settle()

	if !ta.quit {
		return nil, errors.New("the game didn't end")
	}

	if ud.files.path != ta.gam.SaveFile {
		return nil, nil
	}

	return ud.files.data, nil
}
