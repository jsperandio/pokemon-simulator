// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// get returns a window's control by name; the layouts are fixed, so a
// missing one is a bug.
func get[T widget](w *window, name string) T {
	k, ok := w.find(name).(T)
	if !ok {
		panic(fmt.Sprintf("classic: %s has no %s of that kind", w.name, name))
	}

	return k
}

// buildForms makes every form, hidden, as the original creates them all
// at start-up.
func (a *App) buildForms() {
	a.load = newLoadForm(a)
	a.start = newStartForm(a)
	a.main = newMainForm(a)
	a.stats = newStatsForm(a)
	a.info = newInfoForm(a)
	a.dex = newDexForm(a)
	a.mart = newMartForm(a)
	a.types = newTypesForm(a)
	a.corner = newCornerForm(a)
	a.name = newNameDialog(a)
	a.about = newWindow(a.desk, &layout.AboutBox)
	a.desk.add(a.about)
}

// placeWindows puts the game, Statistics and Battle Info windows side by
// side around the screen's centre, as a new game (dy 236) or a loaded one
// (dy 246) does.
func (a *App) placeWindows(dy int) {
	x, y := screenWidth/2-305, screenHeight/2-dy
	a.main.w.pos.X, a.main.w.pos.Y = x, y
	a.stats.w.pos.X, a.stats.w.pos.Y = x+428, y
	a.info.w.pos.X, a.info.w.pos.Y = x, y+314
	a.desk.dirty = true
}

// showAbout is the About box (TAboutBox.ShowModal).
func (a *App) showAbout() {
	a.call(func() { a.modal(a.about) })
}

// exit ends the program, as closing the loading screen, the main form,
// does.
func (a *App) exit() { a.quit = true }

// loadGame loads a save (TForm1.LoadGame); it reports whether it could.
// It runs on the handler's goroutine.
func (a *App) loadGame(file string) bool {
	b, err := a.files.ReadFile(file)
	if err == nil {
		err = a.g.Load(file, b)
	}

	if err != nil {
		a.Message(game.Error, false, err.Error())

		return false
	}

	a.sync()
	a.placeWindows(246)
	a.desk.hide(a.start.w)

	return true
}

// speciesName is a species' name, or "" for none.
func (a *App) speciesName(s int) string {
	if sp := a.g.Data.SpeciesByID(s); sp != nil {
		return sp.Name
	}

	return ""
}

// dexItems are the species lists' items: "Bulbasaur (#001)".
func (a *App) dexItems() []string {
	out := make([]string, 0, len(a.g.Data.Species))
	for _, sp := range a.g.Data.Species {
		out = append(out, fmt.Sprintf("%s (#%03d)", sp.Name, sp.ID))
	}

	return out
}

// gaugeColors are the HP gauges' colours.
var gaugeColors = map[game.GaugeColor]color.RGBA{
	game.Green: vclColors["clGreen"], game.Yellow: vclColors["clYellow"], game.Red: vclColors["clRed"],
}

// setItems replaces a radio group's items if they changed.
func setRadioItems(g *radioGroup, items []string) {
	if !slices.Equal(g.items, items) {
		g.setItems(append([]string(nil), items...))
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
