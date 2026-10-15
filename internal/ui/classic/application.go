// SPDX-License-Identifier: AGPL-3.0-only

// Package classic is the classic user interface: the original's Windows 98
// windows, drawn in software on an 800×600 desktop that Ebitengine scales
// to the screen.
package classic

import (
	"fmt"
	"image"
	"io/fs"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/jsperandio/pokemon-simulator/internal/audio"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/help"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// Keyboard auto-repeat, in ticks at 60 per second: Windows' default delay
// of half a second, then about 30 repeats a second.
const (
	repeatDelay    int = 30
	repeatInterval int = 2
)

// musicPlayer is what the Music menu plays music with: an audio.Music.
type musicPlayer interface {
	Open(name string, data []byte) error
	Stop() error
	Update(repeat bool)
	Playing() bool
}

// Application is the classic user interface: an ebiten.Game that shows the
// desktop, and the game's game.UI.
type Application struct {
	g       *game.Game
	res     *resources
	desk    *desktop
	files   platform.Files
	args    []string // the command line's arguments, as the original's ParamStr
	lastDir string   // the file dialogs' folder: the last file chosen's

	frame  *image.RGBA
	img    *ebiten.Image
	screen *ebiten.Image // the screen Draw drew on last
	shown  ebiten.CursorShapeType
	wheel  float64 // the wheel's movement not yet a whole notch
	pace   *pacer

	start      *startForm
	load       *loadForm
	main       *mainForm
	stats      *statsForm
	info       *infoForm
	pokedex    *pokedexForm
	pokemart   *pokemartForm
	types      *typesForm
	gameCorner *gameCornerForm
	name       *nameDialog
	about      *window

	helpWindow *helpWindow // the manual, once opened
	music      musicPlayer
	xfer       transfers   // the browser's uploads and downloads, on the web
	fileDialog *fileDialog // the file dialog that shows, if one does
	draws      *drawShow   // the Game Corner's draws, while they show

	// A game handler runs on its own goroutine, taking turns with the
	// user interface: while it runs, the user interface waits, and while
	// it waits for a modal window, the user interface runs.
	running  bool
	done     chan struct{}
	requests chan func()

	quit bool
}

// ErrQuit is returned by Update when the game ends.
var ErrQuit = ebiten.Termination

// NewApplication makes the user interface for a new run of the game, at its loading
// screen. args are the command line's arguments: a save file among them
// is loaded instead of showing the start screen.
func NewApplication(data *gamedata.Data, files platform.Files, args []string) (*Application, error) {
	res, err := loadResources()
	if err != nil {
		return nil, err
	}

	app := &Application{
		res:      res,
		desk:     newDesktop(res),
		files:    files,
		args:     args,
		frame:    image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight)),
		requests: make(chan func()),
		music:    &audio.Music{},
		pace:     newPacer(),
	}
	if platform.Transfers {
		app.xfer = browser{files}
	}

	app.desk.onMinimize = ebiten.MinimizeWindow
	app.g = game.New(data, app)
	app.buildForms()

	app.g.OnShow = app.show
	app.g.OnDraws = app.showDraws
	app.load.begin()

	return app, nil
}

// Game returns the game the user interface plays.
func (app *Application) Game() *game.Game {
	return app.g
}

// help is the start screen's "Help!" and Help → Contents (F1): the manual,
// at its contents.
func (app *Application) help() {
	if app.helpWindow == nil {
		app.helpWindow = newHelpWindow(app)
	}

	app.helpWindow.jump(help.Contents)
	app.desk.show(app.helpWindow.w)
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

// call runs a game handler, or any code that waits for modal windows, on
// its own goroutine; it returns once the handler ends or waits for a
// window. Nothing runs while another handler waits.
func (app *Application) call(f func()) {
	if app.running {
		return
	}

	app.running = true
	app.done = make(chan struct{})

	go func() {
		defer close(app.done)

		f()
	}()

	app.wait()
}

// wait lets the handler run until it ends or asks for a modal window.
func (app *Application) wait() {
	select {
	case <-app.done:
		app.running = false
		app.sync()
	case show := <-app.requests:
		app.sync()
		show()
	}
}

// modal shows w as a modal window and returns its modal result once it
// closes. It runs on the handler's goroutine.
func (app *Application) modal(w *window) int {
	resp := make(chan int)

	app.requests <- func() {
		w.onModal = func(result int) {
			w.onModal = nil

			resp <- result

			app.wait()
		}
		app.desk.showModal(w)
	}

	return <-resp
}

// appTitle is the program's title (Application.Title).
const appTitle string = "POKéMON Simulator"

// showException shows an error as Application.ShowException does: its
// message, ending in a full stop, in a box captioned with the program's
// title. It runs on the handler's goroutine.
func (app *Application) showException(msg string) {
	if msg != "" && msg[len(msg)-1] > '.' {
		msg += "."
	}

	w := messageDialog(app.desk, game.Error, false, msg)
	w.caption = appTitle
	app.modal(w)
	app.desk.remove(w)
}

// Message shows a message dialog and waits for the player (game.UI).
func (app *Application) Message(kind game.DialogKind, yesNo bool, text string) bool {
	w := messageDialog(app.desk, kind, yesNo, text)
	r := app.modal(w)
	app.desk.remove(w)

	return r == mrOk || r == mrYes
}

// show shows a window the game shows, bringing it to the front, as Show
// does; it runs on the handler's goroutine.
func (app *Application) show(w game.Window) {
	app.syncForms() // the controls as the game left them, before the focus comes in
	app.desk.show(map[game.Window]*window{
		game.MainForm:   app.main.w,
		game.StatsForm:  app.stats.wdw,
		game.InfoForm:   app.info.wdw,
		game.DexForm:    app.pokedex.wdw,
		game.MartForm:   app.pokemart.wdw,
		game.TypesForm:  app.types.wdw,
		game.CornerForm: app.gameCorner.wdw,
	}[w])
}

// showDraws shows a play's draws in "Lucky number", one after another, as
// the original's loop did, before the game tells the result. It runs on
// the handler's goroutine, which waits until the last draw shows; the
// user interface takes no input meanwhile, as the original's didn't.
func (app *Application) showDraws(draws []int) {
	done := make(chan struct{})

	app.requests <- func() {
		app.stats.freshen() // Form5.Freshen: the bet paid

		app.draws = &drawShow{
			draws: draws,
			start: app.desk.now,
			done: func() {
				close(done)
				app.wait()
			},
		}
	}

	<-done
}

// sync brings the windows up to date with the game's state.
func (app *Application) sync() {
	g := app.g
	// The windows the game shows, in the order the original shows them
	// when several appear at once.
	for _, f := range []struct {
		w       *window
		visible bool
	}{
		{app.info.wdw, g.Info.Visible},
		{app.pokedex.wdw, g.Dex.Visible},
		{app.stats.wdw, g.Stats.Visible},
		{app.pokemart.wdw, g.Mart.Visible},
		{app.types.wdw, g.Types.Visible},
		{app.gameCorner.wdw, g.Corner.Visible},
		{app.main.w, g.Main.Visible},
	} {
		switch {
		case f.visible && !f.w.visible:
			app.syncForms()
			app.desk.show(f.w)
		case !f.visible && f.w.visible:
			app.desk.hide(f.w)
		}
	}

	app.syncForms()
}

// syncForms brings the forms' controls up to date with the game's state.
func (app *Application) syncForms() {
	app.main.sync()
	app.stats.sync()
	app.info.sync()
	app.pokedex.sync()
	app.pokemart.sync()
	app.types.sync()
	app.gameCorner.sync()
}

// Update handles a frame's input (ebiten.Game).
func (app *Application) Update() error {
	if app.quit {
		platform.WarnOnLeave(false)

		return ErrQuit
	}

	app.pace.wait()

	if ebiten.IsWindowBeingClosed() && app.desk.topModal() == nil {
		app.closeProgram()
	}

	in, now := app.input(), time.Now()
	app.step(in, now)
	// On the web, leaving the page doesn't save the game, as quitting does.
	platform.WarnOnLeave(app.main.w.visible && !app.quit)

	if app.desk.cursor != app.shown {
		app.shown = app.desk.cursor
		ebiten.SetCursorShape(app.shown)
	}

	if app.quit {
		platform.WarnOnLeave(false)

		return ErrQuit
	}

	app.pace.next(&in, app.desk.dirty, app.music.Playing(), now)

	return nil
}

// step runs a frame: the input, then the timers.
func (app *Application) step(in input, now time.Time) {
	if app.draws != nil {
		app.draws.tick(app, now) // and no input

		return
	}

	app.desk.update(&in, now)
	app.load.tick()
	app.music.Update(app.g.Item("RepeatMusic1").Checked) // TForm4.Timer1Timer

	if app.xfer != nil && in.dropped != nil {
		app.importDropped(in.dropped)
	}

	if app.fileDialog != nil {
		app.fileDialog.poll()
	}

	if !app.running {
		app.sync()
		app.stats.freshen()
	}
}

// keyFires reports whether a key held for d frames acts in this one: at
// once, then every repeatInterval frames after repeatDelay.
func keyFires(d int) bool {
	return d == 1 || (d >= repeatDelay && (d-repeatDelay)%repeatInterval == 0)
}

// input reads this frame's input from Ebitengine.
func (app *Application) input() input {
	var in input
	x, y := ebiten.CursorPosition()
	in.mouse = image.Pt(x, y)
	in.down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in.pressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	in.released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	_, dy := ebiten.Wheel()
	app.wheel += dy

	if n := math.Trunc(app.wheel); n != 0 {
		in.wheel = int(n)
		app.wheel -= n
	}

	in.mods = mods{
		shift: ebiten.IsKeyPressed(ebiten.KeyShift),
		ctrl:  ebiten.IsKeyPressed(ebiten.KeyControl),
		alt:   ebiten.IsKeyPressed(ebiten.KeyAlt),
	}

	for _, k := range inpututil.AppendPressedKeys(nil) {
		d := inpututil.KeyPressDuration(k)
		if keyFires(d) {
			in.keys = append(in.keys, k)
		}
	}

	in.keysUp = inpututil.AppendJustReleasedKeys(nil)
	in.chars = ebiten.AppendInputChars(nil)
	in.dropped = ebiten.DroppedFiles()

	return in
}

// importDropped uploads the files dropped on the game into the folder the
// file dialog shows, or else the home folder.
func (app *Application) importDropped(fsys fs.FS) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return
	}

	dir := app.files.Home()
	if app.fileDialog != nil {
		dir = app.fileDialog.dir
	}

	var names []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		b, err := fs.ReadFile(fsys, e.Name())
		if err == nil {
			err = app.files.WriteFile(app.files.Join(dir, e.Name()), b)
		}

		if err == nil {
			names = append(names, e.Name())
		}
	}

	if app.fileDialog != nil && len(names) > 0 {
		app.fileDialog.uploaded(names)
	}
}

// Draw draws the desktop when it has changed, or on a new screen
// (ebiten.Game). The screen keeps the last frame otherwise, as main turns
// its clearing off, and Ebitengine stops presenting frames that change
// nothing.
func (app *Application) Draw(screen *ebiten.Image) {
	if app.img == nil {
		app.img = ebiten.NewImage(screenWidth, screenHeight)
		app.desk.dirty = true
	}

	switch {
	case app.desk.dirty:
		app.desk.paint(app.frame)
		app.img.WritePixels(app.frame.Pix)
	case screen == app.screen:
		return
	}

	app.screen = screen
	screen.DrawImage(app.img, nil)
}

// Layout returns the logical screen's size (ebiten.Game).
func (app *Application) Layout(int, int) (w, h int) {
	return screenWidth, screenHeight
}

// openFile shows a file dialog and waits for the player; it returns the
// chosen file's path. It runs on the handler's goroutine.
func (app *Application) openFile(opts fileOptions) (string, bool) {
	f := newFileDialog(app, opts)
	app.fileDialog = f
	r := app.modal(f.window)
	app.fileDialog = nil
	app.desk.remove(f.window)

	return f.result, r == mrOk
}

// closeProgram ends the program as closing the main form does.
func (app *Application) closeProgram() {
	if app.main.w.visible {
		app.main.formClose()

		return
	}

	app.quit = true
}

// WindowSize returns the window's default size: the desktop at the
// largest scale, in steps of a half, that fits 90 % of the monitor. With
// no monitor, it is the desktop's size; ebiten.RunGame then says why.
func WindowSize() (w, h int) {
	m := ebiten.Monitor()
	if m == nil {
		return screenWidth, screenHeight
	}

	mw, mh := m.Size()
	deskW, deskH := float64(screenWidth), float64(screenHeight)
	s := math.Floor(min(0.9*float64(mw)/deskW, 0.9*float64(mh)/deskH)*2) / 2
	s = max(s, 1)

	return int(deskW * s), int(deskH * s)
}

// Icon returns the program's icon, for the window.
func (app *Application) Icon() image.Image {
	return app.res.picture("sprites/icon.png")
}
