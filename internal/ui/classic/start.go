// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// The program's start (notes/loader.md, notes/lifecycle.md): the loading
// screen, the start screen and the name dialog.

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
	a       *Application
	w       *window
	step    int // the step under way
	done    int // its items read
	total   int // all items read
	running bool
}

func newLoadForm(a *Application) *loadForm {
	f := &loadForm{a: a, w: newWindow(a.desk, &layout.Form3)}
	a.desk.add(f.w)

	return f
}

// begin shows the screen (Timer1 starts TimerEventLoad).
func (f *loadForm) begin() {
	f.running = true
	f.startStep(0)
	f.a.desk.show(f.w)
}

// startStep is TForm3.R: a step's caption, and Gauge1 at 0 of its items.
func (f *loadForm) startStep(i int) {
	f.step, f.done = i, 0
	get[*label](f.w, "Label2").setCaption(loadSteps[i].caption)
	get[*gauge](f.w, "Gauge1").set(loadSteps[i].n, 0, colSelection)
}

// tick reads the next items (TForm3.Incre for each).
func (f *loadForm) tick() {
	if !f.running {
		return
	}

	for range loadPerTick {
		f.done++

		f.total++

		if f.done == loadSteps[f.step].n {
			if f.step+1 == len(loadSteps) {
				f.finish()

				return
			}

			f.startStep(f.step + 1)
		}
	}

	g1 := get[*gauge](f.w, "Gauge1")
	g1.set(g1.max, f.done, g1.fore)

	g2 := get[*gauge](f.w, "Gauge2")
	g2.set(g2.max, f.total, g2.fore)
}

// finish hides the screen (TForm3.FormHide): a save file on the command
// line is loaded, or the start screen shows.
func (f *loadForm) finish() {
	f.running = false
	a := f.a
	a.desk.hide(f.w)

	file := ""

	for _, arg := range a.args {
		if st, err := a.files.Stat(arg); err == nil && !st.Dir {
			file = arg // the last existing one wins
		}
	}

	if file != "" {
		a.call(func() {
			if !a.loadGame(file) {
				a.desk.show(a.start.w)
			}
		})

		return
	}

	a.desk.show(a.start.w)
}

// startForm is the start screen (TForm1).
type startForm struct {
	a *Application
	w *window
}

func newStartForm(a *Application) *startForm {
	f := &startForm{a: a, w: newWindow(a.desk, &layout.Form1)}
	w := f.w
	get[*button](w, "NewButton").onClick = f.newButtonClick
	get[*button](w, "LoadButton").onClick = f.loadButtonClick
	get[*button](w, "HelpButton").onClick = a.help
	get[*button](w, "AboutButton").onClick = a.showAbout
	get[*button](w, "ExitButton").onClick = a.exit
	w.onClose = a.exit // FormClose closes the main form
	w.onDeactivate = func() {
		if w.visible && a.main.w.visible {
			a.desk.hide(w)
		}
	}
	a.desk.add(w)

	return f
}

// saveFilter is the save dialogs' filter.
const saveFilter string = "Saved Games (*.PGS)|*.pgs"

// newButtonClick is "Start a new game!": the save file first, then the
// name, then the starter.
func (f *startForm) newButtonClick() {
	a := f.a
	a.call(func() {
		file, ok := a.openFile(fileOptions{
			save:       true,
			filter:     saveFilter,
			defaultExt: "pgs",
			overwrite:  true,
		})
		if !ok {
			return
		}

		a.desk.hide(f.w)

		name := a.name.ask()
		a.g.NewGame(file, name)
		a.sync()
		a.placeWindows(236)
	})
}

// loadButtonClick is "Load a saved game...".
func (f *startForm) loadButtonClick() {
	a := f.a
	a.call(func() {
		file, ok := a.openFile(fileOptions{filter: saveFilter, defaultExt: "pgs", mustExist: true})
		if ok {
			a.loadGame(file)
		}
	})
}

// nameDialog is "Enter your name:" (TOKRightDlg).
type nameDialog struct {
	a    *Application
	w    *window
	edit *edit
}

func newNameDialog(a *Application) *nameDialog {
	d := &nameDialog{a: a, w: newWindow(a.desk, &layout.OKRightDlg)}
	d.edit = get[*edit](d.w, "Edit1")
	a.desk.add(d.w)

	return d
}

// ask shows the dialog and returns the name typed (OKBtnClick); it runs on
// the handler's goroutine.
func (d *nameDialog) ask() string {
	d.a.modal(d.w)

	return d.edit.text()
}
