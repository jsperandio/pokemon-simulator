// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// The program's start: the loading screen, the start screen and
// the name dialog.

// loadSteps are the loader's steps (TForm3.R), with the items each reads.
var loadSteps = []struct {
	n       int
	caption string
}{
	{306, "Declaring type relations"},
	{386, "Declaring POKéMON names"},
	{24, "Declaring badge names"},
	{772, "Declaring POKéMON Types"},
	{386, "Declaring evolution levels"},
	{386, "Declaring evolutions"},
	{119, "Declaring attack names"},
	{119, "Declaring attack powers"},
	{119, "Declaring attack types"},
	{17, "Declaring type names"},
	{1544, "Declaring POKéMON attacks"},
	{18, "Initializing tournament prizes"},
	{64, "Initializing tournament trainers"},
}

// loadPerTick is how many items the loading screen shows read per tick:
// the 4260 items take about a second.
const loadPerTick int = 71

// loadForm is the loading screen (TForm3), the program's main form. The
// port has read the data already; the screen plays the original's
// progress.
type loadForm struct {
	app     *Application
	wdw     *window
	step    int // the step under way
	done    int // its items read
	total   int // all items read
	running bool
}

func newLoadForm(app *Application) *loadForm {
	lf := &loadForm{
		app: app,
		wdw: newWindow(app.desk, &layout.Form3),
	}
	app.desk.add(lf.wdw)

	return lf
}

// begin shows the screen (Timer1 starts TimerEventLoad).
func (lf *loadForm) begin() {
	lf.running = true
	lf.startStep(0)
	lf.app.desk.show(lf.wdw)
}

// startStep is TForm3.R: a step's caption, and Gauge1 at 0 of its items.
func (lf *loadForm) startStep(i int) {
	lf.step, lf.done = i, 0
	get[*label](lf.wdw, "Label2").setCaption(loadSteps[i].caption)
	get[*gauge](lf.wdw, "Gauge1").set(loadSteps[i].n, 0, colSelection)
}

// tick reads the next items (TForm3.Incre for each).
func (lf *loadForm) tick() {
	if !lf.running {
		return
	}

	for range loadPerTick {
		lf.done++

		lf.total++

		if lf.done == loadSteps[lf.step].n {
			if lf.step+1 == len(loadSteps) {
				lf.finish()

				return
			}

			lf.startStep(lf.step + 1)
		}
	}

	g1 := get[*gauge](lf.wdw, "Gauge1")
	g1.set(g1.max, lf.done, g1.fore)

	g2 := get[*gauge](lf.wdw, "Gauge2")
	g2.set(g2.max, lf.total, g2.fore)
}

// finish hides the screen (TForm3.FormHide): a save file on the command
// line is loaded, or the start screen shows.
func (lf *loadForm) finish() {
	lf.running = false
	app := lf.app
	app.desk.hide(lf.wdw)

	file := ""

	for _, arg := range app.args {
		if st, err := app.files.Stat(arg); err == nil && !st.Dir {
			file = arg // the last existing one wins
		}
	}

	if file != "" {
		app.call(func() {
			if !app.loadGame(file) {
				app.desk.show(app.start.wdw)
			}
		})

		return
	}

	app.desk.show(app.start.wdw)
}

// startForm is the start screen (TForm1).
type startForm struct {
	app *Application
	wdw *window
}

func newStartForm(app *Application) *startForm {
	sf := &startForm{
		app: app,
		wdw: newWindow(app.desk, &layout.Form1),
	}
	wdw := sf.wdw
	get[*button](wdw, "NewButton").onClick = sf.newButtonClick
	get[*button](wdw, "LoadButton").onClick = sf.loadButtonClick
	get[*button](wdw, "HelpButton").onClick = app.help
	get[*button](wdw, "AboutButton").onClick = app.showAbout
	get[*button](wdw, "ExitButton").onClick = app.exit
	wdw.onClose = app.exit // FormClose closes the main form
	wdw.onDeactivate = func() {
		if wdw.visible && app.main.wdw.visible {
			app.desk.hide(wdw)
		}
	}
	app.desk.add(wdw)

	return sf
}

// saveFilter is the save dialogs' filter.
const saveFilter string = "Saved Games (*.PGS)|*.pgs"

// newButtonClick is "Start a new game!": the save file first, then the
// name, then the starter.
func (sf *startForm) newButtonClick() {
	app := sf.app
	app.call(func() {
		file, ok := app.openFile(fileOptions{
			save:       true,
			filter:     saveFilter,
			defaultExt: "pgs",
			overwrite:  true,
		})
		if !ok {
			return
		}

		app.desk.hide(sf.wdw)

		name, ok := app.name.ask()
		app.game.NewGame(file, name, ok)
		app.sync()
		app.placeWindows(236)
		app.main.centreForStarters()
	})
}

// loadButtonClick is "Load a saved game...".
func (sf *startForm) loadButtonClick() {
	app := sf.app
	app.call(func() {
		file, ok := app.openFile(fileOptions{
			filter:     saveFilter,
			defaultExt: "pgs",
			mustExist:  true,
		})
		if ok {
			app.loadGame(file)
		}
	})
}

// nameDialog is "Enter your name:" (TOKRightDlg).
type nameDialog struct {
	app  *Application
	wdw  *window
	edit *edit
}

func newNameDialog(app *Application) *nameDialog {
	nd := &nameDialog{
		app: app,
		wdw: newWindow(app.desk, &layout.OKRightDlg),
	}
	nd.edit = get[*edit](nd.wdw, "Edit1")
	app.desk.add(nd.wdw)

	return nd
}

// ask shows the dialog and returns the name typed, and whether it closed
// with OK (OKBtnClick) rather than Alt+F4; it runs on the handler's
// goroutine.
func (nd *nameDialog) ask() (name string, ok bool) {
	rst := nd.app.modal(nd.wdw)

	return nd.edit.text(), rst == mrOk
}
