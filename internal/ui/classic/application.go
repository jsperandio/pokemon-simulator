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

// storage is the game's storage: what the file dialogs browse, and what the
// game reads and writes, from the folder the dialogs show first.
type storage interface {
	navigator
	writer
	Home() string
}

// transfers moves files between the game's storage and the player's
// computer.
type transfers interface {
	// upload copies the files the player picks into dir; their names
	// come on the channel once they are there.
	upload(dir string) <-chan []string
	download(name string, data []byte) error
}

// Application is the classic user interface: an ebiten.Game that shows the
// desktop, and the game's game.UI.
type Application struct {
	game    *game.Game
	res     *resources
	desk    *desktop
	files   storage
	args    []string // the command line's arguments, as the original's ParamStr
	lastDir string   // the file dialogs' folder: the last file chosen's

	frame  *image.RGBA
	img    *ebiten.Image
	screen *ebiten.Image // the screen Draw drew on last
	shown  ebiten.CursorShapeType
	wheel  float64 // the wheel's movement not yet a whole notch
	finger finger  // a touch screen's left button
	pace   *pacer

	settings    settingsStore
	preferences preferences // the player's, as the settings keep them

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
	prefsForm  *preferencesForm

	helpWindow *helpWindow // the manual, once opened
	music      musicPlayer
	xfer       transfers   // the browser's uploads and downloads, on the web
	keyboard   *keyboard   // the keyboard a touch screen shows, on the web
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

// NewApplication makes the user interface for a new run of the game, at its loading
// screen, with the player's preferences from settings. args are the
// command line's arguments: a save file among them is loaded instead of
// showing the start screen.
func NewApplication(data *gamedata.Data, files storage, settings settingsStore, args []string) (*Application, error) {
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
		settings: settings,
		keyboard: newKeyboard(),
	}

	// Settings that can't be read leave the defaults.
	dat, _ := settings.Read()
	app.preferences = loadPreferences(dat)

	if platform.Transfers {
		app.xfer = browser{files}
	}

	app.desk.onMinimize = ebiten.MinimizeWindow
	app.game = game.New(data, app)
	app.game.RollShiny = game.RandomShiny
	app.buildForms()

	app.game.OnShow = app.show
	app.game.OnDraws = app.showDraws
	app.load.begin()

	return app, nil
}

// Game returns the game the user interface plays.
func (app *Application) Game() *game.Game {
	return app.game
}

// help is the start screen's "Help!" and Help → Contents (F1): the manual,
// at its contents.
func (app *Application) help() {
	if app.helpWindow == nil {
		app.helpWindow = newHelpWindow(app)
	}

	app.helpWindow.jump(help.Contents)
	app.desk.show(app.helpWindow.wdw)
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

	app.prefsForm = newPreferencesForm(app)
}

// placeWindows puts the game, Statistics and Battle Info windows as the
// preferences lay them out: filling the screen, or side by side around
// the screen's centre with their own sizes, as a new game (dy 236) or a
// loaded one (dy 246) does. The original's left edge, Screen.Width div 2
// − 305, is half the screen less half the two windows' width.
func (app *Application) placeWindows(dy int) {
	if app.preferences.fillScreen {
		app.fillScreen()

		return
	}

	for _, wdw := range []*window{app.main.wdw, app.stats.wdw, app.info.wdw} {
		wdw.pinned = false
		wdw.resize(wdw.design)
	}

	mainSize, statsSize := app.main.wdw.size(), app.stats.wdw.size()
	x, y := screenWidth/2-(mainSize.X+statsSize.X)/2, screenHeight/2-dy
	app.main.wdw.pos = image.Pt(x, y)
	app.stats.wdw.pos = image.Pt(x+mainSize.X, y)
	app.info.wdw.pos = image.Pt(x, y+mainSize.Y)
	app.desk.dirty = true
}

// fillScreen docks the game, Statistics and Battle Info windows edge to
// edge over the screen, where they can't be dragged: Statistics down the
// right edge, with its own width, Battle Info along the bottom of the
// rest, with its own height, and the game window above it. A hidden one
// leaves its place empty.
func (app *Application) fillScreen() {
	mainWdw, statsWdw, infoWdw := app.main.wdw, app.stats.wdw, app.info.wdw
	side := statsWdw.design.X + statsWdw.nonClient().X
	bottom := infoWdw.design.Y + infoWdw.nonClient().Y

	statsWdw.resize(image.Pt(statsWdw.design.X, screenHeight-statsWdw.nonClient().Y))
	infoWdw.resize(image.Pt(screenWidth-side-infoWdw.nonClient().X, infoWdw.design.Y))
	mainWdw.resize(image.Pt(screenWidth-side, screenHeight-bottom).Sub(mainWdw.nonClient()))

	statsWdw.pos = image.Pt(screenWidth-side, 0)
	infoWdw.pos = image.Pt(0, screenHeight-bottom)
	mainWdw.pos = image.Point{}

	for _, wdw := range []*window{mainWdw, statsWdw, infoWdw} {
		wdw.pinned = true
	}

	app.desk.dirty = true
}

// showAbout is the About box (TAboutBox.ShowModal).
func (app *Application) showAbout() {
	app.call(func() {
		app.modal(app.about)
	})
}

// showPreferences is File → Preferences..., the Preferences window.
func (app *Application) showPreferences() {
	app.call(app.prefsForm.open)
}

// setPreferences makes prf the player's preferences, which take effect at
// once: animations turned off stop, the lists with shiny POKéMON show the
// mark chosen, and the game's windows take the layout chosen, side by side
// as a loaded game has them, or filling the screen.
func (app *Application) setPreferences(prf preferences) {
	relayout := prf.fillScreen != app.preferences.fillScreen
	app.preferences = prf

	if relayout && app.main.wdw.visible {
		app.placeWindows(246)
	}

	app.stats.party.invalidate()
	app.main.starters.invalidate()
	app.info.sync()

	if !prf.animatedSprites {
		app.main.stopAnimations()
		app.pokedex.stopAnimation()
	}
}

// keepPreferences makes prf the player's preferences, and keeps them in
// the settings for the next runs.
func (app *Application) keepPreferences(prf preferences) error {
	app.setPreferences(prf)

	dat, _ := app.settings.Read()

	return app.settings.Write(prf.store(dat))
}

// exit ends the program, as closing the loading screen, the main form,
// does.
func (app *Application) exit() {
	app.quit = true
}

// loadGame loads a save (TForm1.LoadGame); it reports whether it could.
// It runs on the handler's goroutine.
func (app *Application) loadGame(file string) bool {
	dat, err := app.files.ReadFile(file)
	if err == nil {
		err = app.game.Load(file, dat)
	}

	if err != nil {
		app.Message(game.Error, false, err.Error())

		return false
	}

	app.sync()
	app.placeWindows(246)
	app.desk.hide(app.start.wdw)

	return true
}

// speciesName is a species' name, or "" for none.
func (app *Application) speciesName(sid int) string {
	if sp := app.game.Data.SpeciesByID(sid); sp != nil {
		return sp.Name
	}

	return ""
}

// pokedexItems are the species lists' items: "Bulbasaur (#001)".
func (app *Application) pokedexItems() []string {
	out := make([]string, 0, len(app.game.Data.Species))
	for _, sp := range app.game.Data.Species {
		out = append(out, fmt.Sprintf("%s (#%03d)", sp.Name, sp.ID))
	}

	return out
}

// call runs a game handler, or any code that waits for modal windows, on
// its own goroutine; it returns once the handler ends or waits for a
// window. Nothing runs while another handler waits.
func (app *Application) call(fnc func()) {
	if app.running {
		return
	}

	app.running = true
	app.done = make(chan struct{})

	go func() {
		defer close(app.done)

		fnc()
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

// modal shows wdw as a modal window and returns its modal result once it
// closes. It runs on the handler's goroutine.
func (app *Application) modal(wdw *window) int {
	resp := make(chan int)

	app.requests <- func() {
		wdw.onModal = func(result int) {
			wdw.onModal = nil

			resp <- result

			app.wait()
		}
		app.desk.showModal(wdw)
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

	wdw := messageDialog(app.desk, game.Error, false, msg, "")
	wdw.caption = appTitle
	app.modal(wdw)
	app.desk.remove(wdw)
}

// Message shows a message dialog and waits for the player (game.UI).
// The prompt every battle starts with, game.ChoosePrompt, has "Don't show
// this message again", which the original doesn't have: once the player
// closes it checked, it is skipped as if they had clicked OK.
func (app *Application) Message(kind game.DialogKind, yesNo bool, text string) bool {
	prompt := text == game.ChoosePrompt
	if prompt && !app.preferences.choosePrompt {
		return true
	}

	check := ""
	if prompt {
		check = dontShowAgain
	}

	wdw := messageDialog(app.desk, kind, yesNo, text, check)
	rst := app.modal(wdw)
	app.desk.remove(wdw)

	// Any way of closing the box counts, as each one is its OK. If the
	// choice can't be kept, it holds for the run.
	if chb, ok := wdw.find("DontShowAgain").(*checkBox); ok && chb.checked {
		prf := app.preferences
		prf.choosePrompt = false
		_ = app.keepPreferences(prf)
	}

	return rst == mrOk || rst == mrYes
}

// show shows a window the game shows, bringing it to the front, as Show
// does; it runs on the handler's goroutine.
func (app *Application) show(gw game.Window) {
	app.syncForms() // the controls as the game left them, before the focus comes in
	app.desk.show(map[game.Window]*window{
		game.MainForm:       app.main.wdw,
		game.StatsForm:      app.stats.wdw,
		game.InfoForm:       app.info.wdw,
		game.PokedexForm:    app.pokedex.wdw,
		game.PokemartForm:   app.pokemart.wdw,
		game.TypesForm:      app.types.wdw,
		game.GameCornerForm: app.gameCorner.wdw,
	}[gw])
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
	gam := app.game
	// The windows the game shows, in the order the original shows them
	// when several appear at once.
	for _, frm := range []struct {
		w       *window
		visible bool
	}{
		{app.info.wdw, gam.Info.Visible},
		{app.pokedex.wdw, gam.Pokedex.Visible},
		{app.stats.wdw, gam.Stats.Visible},
		{app.pokemart.wdw, gam.Pokemart.Visible},
		{app.types.wdw, gam.Types.Visible},
		{app.gameCorner.wdw, gam.GameCorner.Visible},
		{app.main.wdw, gam.Main.Visible},
	} {
		switch {
		case frm.visible && !frm.w.visible:
			app.syncForms()
			app.desk.show(frm.w)
		case !frm.visible && frm.w.visible:
			app.desk.hide(frm.w)
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

// Update handles a frame's input (ebiten.Game). It returns
// ebiten.Termination when the game ends.
func (app *Application) Update() error {
	if app.quit {
		platform.WarnOnLeave(false)

		return ebiten.Termination
	}

	app.pace.wait()

	in, now := app.input(), time.Now()
	if ebiten.IsWindowBeingClosed() {
		app.closeRequested(&in)
	}

	app.step(in, now)
	// On the web, leaving the page doesn't save the game, as quitting does.
	platform.WarnOnLeave(app.main.wdw.visible && !app.quit)

	if app.desk.cursor != app.shown {
		app.shown = app.desk.cursor
		ebiten.SetCursorShape(app.shown)
	}

	if app.quit {
		platform.WarnOnLeave(false)

		return ebiten.Termination
	}

	// The keyboard's text area wakes no frames, so frames run while it is up.
	typing := app.keyboard != nil && app.keyboard.isOpen()
	app.pace.next(&in, app.desk.dirty || typing, app.music.Playing(), now)

	return nil
}

// step runs a frame: the input, then the timers.
func (app *Application) step(in input, now time.Time) {
	if app.draws != nil {
		app.draws.tick(app, now) // and no input

		return
	}

	if app.keyboard != nil {
		app.keyboard.commits(&in)
	}

	app.desk.update(&in, now)

	if app.keyboard != nil {
		app.keyboard.follow(app.desk, &in)
	}

	app.main.animate(now)
	app.pokedex.animate(now)
	app.load.tick()
	app.music.Update(app.game.Item("RepeatMusic1").Checked) // TForm4.Timer1Timer

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

// keyFires reports whether a key held for dur frames acts in this one: at
// once, then every repeatInterval frames after repeatDelay.
func keyFires(dur int) bool {
	return dur == 1 || (dur >= repeatDelay && (dur-repeatDelay)%repeatInterval == 0)
}

// input reads this frame's input from Ebitengine.
func (app *Application) input() input {
	var in input
	x, y := ebiten.CursorPosition()
	in.mouse = image.Pt(x, y)
	in.down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in.pressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	in.released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	app.finger.read(&in, ebitenTouches{})

	_, dy := ebiten.Wheel()
	app.wheel += dy

	if nch := math.Trunc(app.wheel); nch != 0 {
		in.wheel = int(nch)
		app.wheel -= nch
	}

	in.mods = mods{
		shift: ebiten.IsKeyPressed(ebiten.KeyShift),
		ctrl:  ebiten.IsKeyPressed(ebiten.KeyControl),
		alt:   ebiten.IsKeyPressed(ebiten.KeyAlt),
	}

	for _, key := range inpututil.AppendPressedKeys(nil) {
		dur := inpututil.KeyPressDuration(key)
		if keyFires(dur) {
			in.keys = append(in.keys, key)
		}
	}

	in.keysUp = inpututil.AppendJustReleasedKeys(nil)
	in.chars = ebiten.AppendInputChars(nil)
	in.dropped = ebiten.DroppedFiles()

	return in
}

// importDropped uploads the files dropped on the game into the folder the
// file dialog shows, or else the home folder; a dialog that takes no
// uploads ignores them.
func (app *Application) importDropped(fsys fs.FS) {
	fd := app.fileDialog
	if fd != nil && fd.opts.noUpload {
		return
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return
	}

	dir := app.files.Home()
	if fd != nil {
		dir = fd.dir
	}

	var names []string

	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}

		dat, err := fs.ReadFile(fsys, ent.Name())
		if err == nil {
			err = app.files.WriteFile(app.files.Join(dir, ent.Name()), dat)
		}

		if err == nil {
			names = append(names, ent.Name())
		}
	}

	if fd != nil && len(names) > 0 {
		fd.uploaded(names)
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
func (app *Application) Layout(int, int) (wid, hgt int) {
	return screenWidth, screenHeight
}

// openFile shows a file dialog and waits for the player; it returns the
// chosen file's path. It runs on the handler's goroutine.
func (app *Application) openFile(opts fileOptions) (string, bool) {
	fd := newFileDialog(app, opts)
	app.fileDialog = fd
	rst := app.modal(fd.wdw)
	app.fileDialog = nil
	app.desk.remove(fd.wdw)

	return fd.result, rst == mrOk
}

// closeRequested is the system's request to close the program's window,
// in the frame of input in: its close button, or Alt+F4 where the system
// turns the key into a close as well, as Windows does. The program closes
// as the game window does, unless a modal window shows, or the request
// comes with the Alt+F4 that the desktop gives its active window: each of
// the original's windows was the system's, so Alt+F4 closed only the one
// that had the focus.
func (app *Application) closeRequested(in *input) {
	if app.desk.topModal() != nil || app.desk.takesAltF4(in) {
		return
	}

	app.closeProgram()
}

// closeProgram ends the program as closing the main form does.
func (app *Application) closeProgram() {
	if app.main.wdw.visible {
		app.main.formClose()

		return
	}

	app.quit = true
}

// WindowSize returns the window's default size: the desktop at the
// largest scale, in steps of a half, that fits 90 % of the monitor. With
// no monitor, it is the desktop's size; ebiten.RunGame then says why.
func WindowSize() (wid, hgt int) {
	mon := ebiten.Monitor()
	if mon == nil {
		return screenWidth, screenHeight
	}

	mw, mh := mon.Size()
	deskW, deskH := float64(screenWidth), float64(screenHeight)
	scl := math.Floor(min(0.9*float64(mw)/deskW, 0.9*float64(mh)/deskH)*2) / 2
	scl = max(scl, 1)

	return int(deskW * scl), int(deskH * scl)
}

// Icon returns the program's icon, for the window.
func (app *Application) Icon() image.Image {
	return app.res.picture("sprites/icon.png")
}
