// SPDX-License-Identifier: AGPL-3.0-only

// Package classic is the classic user interface: the original's Windows 98
// windows, drawn in software on an 800×600 desktop that Ebitengine scales
// to the screen.
package classic

import (
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
)

// Keyboard auto-repeat, in ticks at 60 per second: Windows' default delay
// of half a second, then about 30 repeats a second.
const (
	repeatDelay    = 30
	repeatInterval = 2
)

// App is the classic user interface: an ebiten.Game that shows the
// desktop, and the game's game.UI.
type App struct {
	g       *game.Game
	res     *resources
	desk    *desktop
	files   platform.Files
	args    []string // the command line's arguments, as the original's ParamStr
	lastDir string   // the file dialogs' folder: the last file chosen's

	frame *image.RGBA
	img   *ebiten.Image
	shown ebiten.CursorShapeType
	wheel float64 // the wheel's movement not yet a whole notch

	start  *startForm
	load   *loadForm
	main   *mainForm
	stats  *statsForm
	info   *infoForm
	dex    *dexForm
	mart   *martForm
	types  *typesForm
	corner *cornerForm
	name   *nameDialog
	about  *window

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

// New makes the user interface for a new run of the game, at its loading
// screen. args are the command line's arguments: a save file among them
// is loaded instead of showing the start screen.
func New(data *gamedata.Data, files platform.Files, args []string) (*App, error) {
	res, err := loadResources()
	if err != nil {
		return nil, err
	}
	a := &App{
		res:      res,
		desk:     newDesktop(res),
		files:    files,
		args:     args,
		frame:    image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight)),
		requests: make(chan func()),
	}
	a.g = game.New(data, a)
	a.buildForms()
	a.g.OnShow = a.show
	a.load.begin()
	return a, nil
}

// Game returns the game the user interface plays.
func (a *App) Game() *game.Game { return a.g }

// call runs a game handler, or any code that waits for modal windows, on
// its own goroutine; it returns once the handler ends or waits for a
// window. Nothing runs while another handler waits.
func (a *App) call(f func()) {
	if a.running {
		return
	}
	a.running = true
	a.done = make(chan struct{})
	go func() {
		defer close(a.done)
		f()
	}()
	a.wait()
}

// wait lets the handler run until it ends or asks for a modal window.
func (a *App) wait() {
	select {
	case <-a.done:
		a.running = false
		a.sync()
	case show := <-a.requests:
		a.sync()
		show()
	}
}

// modal shows w as a modal window and returns its modal result once it
// closes. It runs on the handler's goroutine.
func (a *App) modal(w *window) int {
	result := make(chan int)
	a.requests <- func() {
		w.onModal = func(r int) {
			w.onModal = nil
			result <- r
			a.wait()
		}
		a.desk.showModal(w)
	}
	return <-result
}

// Message shows a message dialog and waits for the player (game.UI).
func (a *App) Message(kind game.DialogKind, yesNo bool, text string) bool {
	w := messageDialog(a.desk, kind, yesNo, text)
	r := a.modal(w)
	a.desk.remove(w)
	return r == mrOk || r == mrYes
}

// show shows a window the game shows, bringing it to the front, as Show
// does; it runs on the handler's goroutine.
func (a *App) show(w game.Window) {
	a.desk.show(map[game.Window]*window{
		game.MainForm: a.main.w, game.StatsForm: a.stats.w, game.InfoForm: a.info.w, game.DexForm: a.dex.w,
		game.MartForm: a.mart.w, game.TypesForm: a.types.w, game.CornerForm: a.corner.w,
	}[w])
}

// sync brings the windows up to date with the game's state.
func (a *App) sync() {
	g := a.g
	// The windows the game shows, in the order the original shows them
	// when several appear at once.
	for _, f := range []struct {
		w       *window
		visible bool
	}{
		{a.info.w, g.Info.Visible},
		{a.dex.w, g.Dex.Visible},
		{a.stats.w, g.Stats.Visible},
		{a.mart.w, g.Mart.Visible},
		{a.types.w, g.Types.Visible},
		{a.corner.w, g.Corner.Visible},
		{a.main.w, g.Main.Visible},
	} {
		switch {
		case f.visible && !f.w.visible:
			a.desk.show(f.w)
		case !f.visible && f.w.visible:
			a.desk.hide(f.w)
		}
	}
	a.main.sync()
	a.stats.sync()
	a.info.sync()
	a.dex.sync()
	a.mart.sync()
	a.types.sync()
	a.corner.sync()
}

// Update handles a frame's input (ebiten.Game).
func (a *App) Update() error {
	if a.quit {
		return ErrQuit
	}
	if ebiten.IsWindowBeingClosed() && a.desk.topModal() == nil {
		a.closeProgram()
	}
	a.step(a.input(), time.Now())
	if a.desk.cursor != a.shown {
		a.shown = a.desk.cursor
		ebiten.SetCursorShape(a.shown)
	}
	if a.quit {
		return ErrQuit
	}
	return nil
}

// step runs a frame: the input, then the timers.
func (a *App) step(in input, now time.Time) {
	a.desk.update(&in, now)
	a.load.tick()
	if !a.running {
		a.sync()
		a.stats.freshen()
	}
}

// input reads this frame's input from Ebitengine.
func (a *App) input() input {
	var in input
	x, y := ebiten.CursorPosition()
	in.mouse = image.Pt(x, y)
	in.down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in.pressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	in.released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	_, dy := ebiten.Wheel()
	a.wheel += dy
	if n := math.Trunc(a.wheel); n != 0 {
		in.wheel = int(n)
		a.wheel -= n
	}
	in.mods = mods{
		shift: ebiten.IsKeyPressed(ebiten.KeyShift),
		ctrl:  ebiten.IsKeyPressed(ebiten.KeyControl),
		alt:   ebiten.IsKeyPressed(ebiten.KeyAlt),
	}
	for _, k := range inpututil.AppendPressedKeys(nil) {
		d := inpututil.KeyPressDuration(k)
		if d == 1 || d >= repeatDelay && (d-repeatDelay)%repeatInterval == 0 {
			in.keys = append(in.keys, k)
		}
	}
	in.keysUp = inpututil.AppendJustReleasedKeys(nil)
	in.chars = ebiten.AppendInputChars(nil)
	return in
}

// Draw draws the desktop (ebiten.Game).
func (a *App) Draw(screen *ebiten.Image) {
	if a.img == nil {
		a.img = ebiten.NewImage(screenWidth, screenHeight)
		a.desk.dirty = true
	}
	if a.desk.dirty {
		a.desk.paint(a.frame)
		a.img.WritePixels(a.frame.Pix)
	}
	screen.DrawImage(a.img, nil)
}

// Layout returns the logical screen's size (ebiten.Game).
func (a *App) Layout(int, int) (int, int) { return screenWidth, screenHeight }

// closeProgram ends the program as closing the main form does.
func (a *App) closeProgram() {
	if a.main.w.visible {
		a.main.formClose()
		return
	}
	a.quit = true
}

// WindowSize returns the window's default size: the desktop at the
// largest scale, in steps of a half, that fits 90 % of the monitor.
func WindowSize() (int, int) {
	mw, mh := ebiten.Monitor().Size()
	s := math.Floor(min(0.9*float64(mw)/screenWidth, 0.9*float64(mh)/screenHeight)*2) / 2
	s = max(s, 1)
	return int(screenWidth * s), int(screenHeight * s)
}

// Icon returns the program's icon, for the window.
func (a *App) Icon() image.Image { return a.res.picture("sprites/icon.png") }
