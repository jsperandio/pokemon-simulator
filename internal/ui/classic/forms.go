// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"fmt"
	"image"
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
func (app *Application) buildForms() {
	app.load = newLoadForm(app)
	app.start = newStartForm(app)
	app.main = newMainForm(app)
	app.stats = newStatsForm(app)
	app.info = newInfoForm(app)
	app.pokedex = newPokedexForm(app)
	app.pokemart = newPokemartForm(app)
	app.types = newTypesForm(app)
	app.gameCorner = newGameCornerForm(app)
	app.name = newNameDialog(app)
	app.about = newWindow(app.desk, &layout.AboutBox)
	app.desk.add(app.about)
}

// placeWindows puts the game, Statistics and Battle Info windows side by
// side around the screen's centre, as a new game (dy 236) or a loaded one
// (dy 246) does.
func (app *Application) placeWindows(dy int) {
	x, y := screenWidth/2-305, screenHeight/2-dy
	app.main.w.pos = image.Pt(x, y)
	app.stats.wdw.pos = image.Pt(x+428, y)
	app.info.wdw.pos = image.Pt(x, y+314)
	app.desk.dirty = true
}

// showAbout is the About box (TAboutBox.ShowModal).
func (app *Application) showAbout() {
	app.call(func() {
		app.modal(app.about)
	})
}

// exit ends the program, as closing the loading screen, the main form,
// does.
func (app *Application) exit() {
	app.quit = true
}

// loadGame loads a save (TForm1.LoadGame); it reports whether it could.
// It runs on the handler's goroutine.
func (app *Application) loadGame(file string) bool {
	b, err := app.files.ReadFile(file)
	if err == nil {
		err = app.g.Load(file, b)
	}

	if err != nil {
		app.Message(game.Error, false, err.Error())

		return false
	}

	app.sync()
	app.placeWindows(246)
	app.desk.hide(app.start.w)

	return true
}

// speciesName is a species' name, or "" for none.
func (app *Application) speciesName(s int) string {
	if sp := app.g.Data.SpeciesByID(s); sp != nil {
		return sp.Name
	}

	return ""
}

// dexItems are the species lists' items: "Bulbasaur (#001)".
func (app *Application) dexItems() []string {
	out := make([]string, 0, len(app.g.Data.Species))
	for _, sp := range app.g.Data.Species {
		out = append(out, fmt.Sprintf("%s (#%03d)", sp.Name, sp.ID))
	}

	return out
}

// gaugeColors are the HP gauges' colours.
var gaugeColors = map[game.GaugeColor]color.RGBA{
	game.Green:  vclColors["clGreen"],
	game.Yellow: vclColors["clYellow"],
	game.Red:    vclColors["clRed"],
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
