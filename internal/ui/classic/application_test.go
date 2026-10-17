// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/audio"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
)

// testApp drives the user interface headlessly, a frame at a time, on a
// clock of its own.
type testApp struct {
	*Application
	t     *testing.T
	now   time.Time
	dir   string // the file dialogs' folder
	mouse image.Point
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	return newTestAppIn(t, t.TempDir(), nil)
}

// newTestAppIn makes the user interface with the file dialogs in dir and
// the command line's arguments args.
func newTestAppIn(t *testing.T, dir string, args []string) *testApp {
	t.Helper()

	app, err := NewApplication(testData(t), platform.NewDisk(dir), args)
	if err != nil {
		t.Fatal(err)
	}

	app.game.Rand.Seed = 1

	return &testApp{
		Application: app,
		t:           t,
		now:         time.Date(2004, 1, 1, 12, 0, 0, 0, time.UTC),
		dir:         dir,
	}
}

// testData returns the game's data.
func testData(t *testing.T) *gamedata.Data {
	t.Helper()

	raw, err := assets.FS.ReadFile("pokegame.dat")
	if err != nil {
		t.Fatal(err)
	}

	data, err := gamedata.Load(raw)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

// frame runs one frame with the given input.
func (ta *testApp) frame(in input) {
	ta.now = ta.now.Add(time.Second / 60)
	in.mouse = ta.mouse
	ta.step(in, ta.now)
}

// wait runs frames with no input.
func (ta *testApp) wait(n int) {
	for range n {
		ta.frame(input{})
	}
}

// click clicks at pnt on the screen; double clicks twice.
func (ta *testApp) click(pnt image.Point) {
	ta.mouse = pnt
	ta.frame(input{})
	ta.frame(input{
		down:    true,
		pressed: true,
	})
	ta.frame(input{released: true})
}

func (ta *testApp) doubleClick(pnt image.Point) {
	ta.click(pnt)
	ta.click(pnt)
}

// clickOn clicks the middle of a window's control.
func (ta *testApp) clickOn(wdw *window, name string) {
	ta.t.Helper()

	wgt := wdw.find(name)
	if wgt == nil {
		ta.t.Fatalf("%s has no %s", wdw.name, name)
	}

	ctl := wgt.base()
	ta.click(widgetOrigin(wgt).Add(ctl.rect.Size().Div(2)))
}

// key presses and releases a key.
func (ta *testApp) key(key ebiten.Key) {
	ta.frame(input{keys: []ebiten.Key{key}})
	ta.frame(input{keysUp: []ebiten.Key{key}})
}

// typeText types characters.
func (ta *testApp) typeText(txt string) {
	ta.frame(input{chars: []rune(txt)})
}

// shot paints the screen and, when $CLASSIC_SHOTS names a folder, saves
// it there as name.png.
func (ta *testApp) shot(name string) {
	ta.t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	ta.desk.paint(img)
	savePNG(ta.t, name, img)
}

// menu clicks through the game window's menu: a bar item, then submenu
// items, by name.
func (ta *testApp) menu(names ...string) {
	ta.t.Helper()

	mb := ta.main.wdw.menu
	for i, me := range mb.entries {
		closed := !mb.active() || mb.sel != i || len(mb.open) == 0
		if me.name() == names[0] && closed {
			rect := mb.barRects()[i]
			ta.click(rect.Min.Add(rect.Size().Div(2)))
		}
	}

	for _, nam := range names[1:] {
		found := false

		for _, mp := range mb.open {
			for i, me := range mp.items {
				if me.name() == nam {
					rect := mp.itemRect(i)
					ta.click(rect.Min.Add(rect.Size().Div(2)))

					found = true
				}
			}
		}

		if !found {
			ta.t.Fatalf("menu item %s not open", nam)
		}
	}
}

// answer closes the modal message dialog with its default button.
func (ta *testApp) answer(want string) {
	ta.t.Helper()

	wdw := ta.desk.topModal()
	if wdw == nil {
		ta.t.Fatalf("no dialog; want %q", want)
	}

	if msg := wdw.find("Message"); msg == nil || !strings.Contains(msg.base().caption, want) {
		ta.t.Fatalf("dialog %q, want %q", msg.base().caption, want)
	}

	ta.key(ebiten.KeyEnter)
	ta.wait(1)
}

// startGame plays up to the game window with a Charmander.
func (ta *testApp) startGame() {
	ta.wait(80)
	ta.clickOn(ta.start.wdw, "NewButton")
	ta.typeText("ash")
	ta.key(ebiten.KeyEnter)
	ta.typeText("Gary")
	ta.key(ebiten.KeyEnter)

	rect := ta.main.starters.itemRect(1)
	ta.click(widgetOrigin(ta.main.starters).Add(rect.Min).Add(image.Pt(20, 20)))
	ta.wait(1)
}

// hintShown reports whether the hint window shows.
func (ta *testApp) hintShown() bool {
	hint := ta.desk.hint

	return !hint.rect.Empty()
}

// spotPoint returns a point on the screen inside a link to target, after
// scrolling it into view.
func (ta *testApp) spotPoint(target string) image.Point {
	ta.t.Helper()

	hv := ta.helpWindow.view
	for _, hsp := range hv.spots {
		if hsp.target != target || len(hsp.boxes) == 0 {
			continue
		}

		rect := hsp.boxes[0]
		if rect.Min.Y < hv.scroll.pos || rect.Max.Y > hv.scroll.pos+hv.scroll.page {
			hv.scroll.set(rect.Min.Y)
		}

		pnt := rect.Min.Add(image.Pt(2, hv.head.Rect.Dy()-hv.scroll.pos+2))

		return widgetOrigin(hv).Add(pnt)
	}

	ta.t.Fatalf("%s has no link to %s", hv.name, target)

	return image.Point{}
}

// menuEffects is what a menu command can change that the player sees.
type menuEffects struct {
	modal   string // the caption of the modal window up
	windows string // the windows shown
	battle  [3]bool
	checked string // the items checked
	quit    bool
}

func (ta *testApp) menuEffects() menuEffects {
	eff := menuEffects{quit: ta.quit}
	if wdw := ta.desk.topModal(); wdw != nil {
		eff.modal = wdw.caption
	}

	for _, wdw := range ta.desk.windows {
		if wdw.visible {
			eff.windows += wdw.name + " "
		}
	}

	gam := ta.game
	eff.battle = [3]bool{gam.InWildBattle, gam.InTrainerBattle, gam.InTournament}

	var walk func(items []*game.MenuItem)
	walk = func(items []*game.MenuItem) {
		for _, mi := range items {
			if mi.Checked {
				eff.checked += mi.Name + " "
			}

			walk(mi.Items)
		}
	}
	walk(gam.Menu)

	return eff
}

// TestNewGame plays the start of a game: loading, the start screen, the
// save file, the name and the starter.
func TestNewGame(t *testing.T) {
	ta := newTestApp(t)
	ta.wait(20)
	ta.shot("01-loading")

	if !ta.load.wdw.visible {
		t.Fatal("no loading screen")
	}

	ta.wait(60)

	if ta.load.wdw.visible || !ta.start.wdw.visible {
		t.Fatal("the start screen didn't follow the loading screen")
	}

	ta.shot("02-start")
	ta.clickOn(ta.start.wdw, "NewButton")
	ta.wait(1)
	ta.shot("03-save-dialog")

	top := ta.desk.topModal()
	if top == nil || top.caption != "Save As" {
		t.Fatalf("no save dialog: %v", top)
	}

	ta.typeText("ash")
	ta.key(ebiten.KeyEnter)
	ta.wait(1)
	ta.shot("04-name")

	if top := ta.desk.topModal(); top != ta.name.wdw {
		t.Fatalf("no name dialog: %v", top)
	}

	ta.typeText("Gary")
	ta.key(ebiten.KeyEnter)
	ta.wait(1)
	ta.shot("05-starters")

	if want := "POKéMON Simulator V4.5 - Gary [ash.pgs]"; ta.main.wdw.caption != want {
		t.Errorf("caption %q, want %q", ta.main.wdw.caption, want)
	}

	if !ta.main.starters.shown() {
		t.Fatal("no starters")
	}

	// Charmander, the second starter.
	rect := ta.main.starters.itemRect(1)
	ta.click(widgetOrigin(ta.main.starters).Add(rect.Min).Add(image.Pt(20, 20)))
	ta.wait(1)
	ta.shot("06-chosen")

	if got := ta.game.Party(); len(got) != 1 || got[0].Species != 4 {
		t.Fatalf("party %v", got)
	}

	if ta.desk.active != ta.main.wdw || !ta.stats.wdw.visible || !ta.pokedex.wdw.visible {
		t.Errorf("windows: active %v, Statistics %v, POKéDEX %v", ta.desk.active, ta.stats.wdw.visible, ta.pokedex.wdw.visible)
	}

	ta.mouse = image.Pt(0, 0)
	ta.wait(1)
	ta.shot("07-game")

	if !strings.HasSuffix(ta.game.SaveFile, "ash.pgs") {
		t.Errorf("save file %q", ta.game.SaveFile)
	}
}

func TestWildBattle(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()
	ta.menu("Action1")
	ta.shot("10-menu")
	ta.menu("Action1", "FightWild1", "FindPKMN1")
	ta.shot("11-wild")
	ta.answer("Select a POKéMON from the Statistics Window")
	ta.shot("12-arena")

	rect := ta.stats.party.itemRect(0)
	ta.doubleClick(widgetOrigin(ta.stats.party).Add(rect.Min).Add(image.Pt(10, 10)))
	ta.shot("13-info")
	ta.answer("# 004 Charmander, L.5, XP:125, XP to next level:91")
	ta.shot("14-confirm")
	ta.answer("Choose Charmander?")
	ta.shot("15-fighter")

	if !ta.main.goButton.shown() {
		t.Fatal("no Go! button")
	}

	attacks := ta.main.attacks
	ta.click(widgetOrigin(attacks).Add(attacks.buttonRect(0).Min).Add(image.Pt(20, 5)))
	ta.clickOn(ta.main.wdw, "Button1")
	ta.wait(1)
	ta.shot("16-turn")

	if len(ta.game.Info.Lines) < 4 {
		t.Errorf("Battle Info: %q", ta.game.Info.Lines)
	}

	ta.clickOn(ta.stats.wdw, "Button1") // a POKéBall
	ta.wait(1)
	ta.shot("17-ball")

	for ta.desk.topModal() != nil {
		ta.key(ebiten.KeyEnter)
		ta.wait(1)
	}

	ta.shot("18-after")
	t.Logf("log: %q", ta.game.Info.Lines)
}

// TestCatchSaveLoad catches a POKéMON, quits, which saves, and loads the
// save in a new run.
func TestCatchSaveLoad(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()

	ta.game.MasterBalls = 1
	ta.menu("Action1", "FightWild1", "FindPKMN1")
	ta.answer("Select a POKéMON")

	rect := ta.stats.party.itemRect(0)
	ta.doubleClick(widgetOrigin(ta.stats.party).Add(rect.Min).Add(image.Pt(10, 10)))
	ta.answer("# 004 Charmander")
	ta.answer("Choose Charmander?")
	ta.clickOn(ta.stats.wdw, "Button4") // the Master Ball
	ta.wait(1)
	ta.shot("20-caught")

	if !ta.main.endButton.shown() {
		t.Fatalf("no End Fight; log %q", ta.game.Info.Lines)
	}

	ta.clickOn(ta.main.wdw, "EndButton")
	ta.wait(1)
	ta.shot("21-ended")

	if n := len(ta.game.Party()); n != 2 {
		t.Fatalf("party of %d", n)
	}

	ta.menu("File1", "Quit1")
	ta.wait(1)

	if !ta.quit {
		t.Fatal("still running after Quit")
	}

	saved := filepath.Join(ta.dir, "ash.pgs")
	if _, err := os.Stat(saved); err != nil {
		t.Fatal(err)
	}

	tb := newTestApp(t)
	tb.files = platform.NewDisk(ta.dir)
	tb.wait(80)
	tb.clickOn(tb.start.wdw, "LoadButton")
	tb.wait(1)
	tb.shot("22-open")

	if top := tb.desk.topModal(); top == nil || top.caption != "Open" {
		t.Fatalf("no open dialog: %v", top)
	}

	tb.typeText("ash")
	tb.key(ebiten.KeyEnter)
	tb.wait(1)
	tb.shot("23-loaded")

	if n := len(tb.game.Party()); n != 2 || tb.game.PlayerName != "Gary" {
		t.Fatalf("loaded %q with a party of %d", tb.game.PlayerName, n)
	}

	if tb.start.wdw.visible || !tb.main.wdw.visible || !tb.stats.wdw.visible {
		t.Error("the game windows don't show")
	}

	if got := tb.main.wdw.pos; got != image.Pt(95, 54) {
		t.Errorf("game window at %v", got)
	}
}

// fakeMusic records what the Music menu asks of the player.
type fakeMusic struct {
	opened  []string
	stopped int
	repeat  bool
	err     error
	stopErr error
}

func (fm *fakeMusic) Open(name string, data []byte) error {
	fm.opened = append(fm.opened, name+":"+string(data))

	return fm.err
}

func (fm *fakeMusic) Stop() error {
	fm.stopped++

	return fm.stopErr
}

func (fm *fakeMusic) Update(repeat bool) {
	fm.repeat = repeat
}

func (fm *fakeMusic) Playing() bool {
	return false
}

// TestMusicMenu opens music from the Music menu, stops it, and turns
// Repeat Music off; the player's errors show as the original's exceptions.
func TestMusicMenu(t *testing.T) {
	ta := newTestApp(t)
	fm := &fakeMusic{stopErr: audio.ErrNotOpen}
	ta.music = fm
	ta.startGame()
	ta.wait(1)

	if !fm.repeat {
		t.Error("Repeat Music is off at the start")
	}

	ta.menu("SelectMusic1", "StopMusic1")
	ta.wait(1)
	ta.shot("40-stop-music")

	if top := ta.desk.topModal(); top == nil || top.caption != "POKéMON Simulator" {
		t.Fatalf("no error box: %v", top)
	}

	ta.answer("No MCI device open.")

	if err := os.WriteFile(filepath.Join(ta.dir, "song.ogg"), []byte("OggS"), 0o600); err != nil {
		t.Fatal(err)
	}

	ta.menu("SelectMusic1", "Openfile1")
	ta.wait(1)
	ta.shot("41-open-music")
	ta.typeText("song.ogg")
	ta.key(ebiten.KeyEnter)
	ta.wait(1)

	if len(fm.opened) != 1 || fm.opened[0] != "song.ogg:OggS" {
		t.Fatalf("opened %q", fm.opened)
	}

	fm.stopErr = nil

	ta.menu("SelectMusic1", "StopMusic1")
	ta.wait(1)

	if fm.stopped != 2 || ta.desk.topModal() != nil {
		t.Errorf("Stop Music: %d stops, dialog %v", fm.stopped, ta.desk.topModal())
	}

	ta.menu("SelectMusic1", "RepeatMusic1")
	ta.wait(1)

	if fm.repeat {
		t.Error("Repeat Music is still on")
	}

	fm.err = audio.ErrFormat

	ta.menu("SelectMusic1", "Openfile1")
	ta.wait(1)
	ta.typeText("song.ogg")
	ta.key(ebiten.KeyEnter)
	ta.wait(1)
	ta.shot("42-cannot-play")
	ta.answer("Cannot play song.ogg. Only MP3, Ogg Vorbis and WAV files can be played.")
}

// A menu's keys type nothing in the dialog they open: Alt+M, then O,
// whose character comes in the same frame as the key.
func TestMenuKeysTypeNothing(t *testing.T) {
	ta := newTestApp(t)
	ta.music = &fakeMusic{}
	ta.startGame()
	ta.frame(input{
		keys: []ebiten.Key{ebiten.KeyAltLeft},
		mods: mods{alt: true},
	})
	ta.frame(input{
		keys:  []ebiten.Key{ebiten.KeyM},
		chars: []rune{'m'},
		mods:  mods{alt: true},
	})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft}})
	ta.frame(input{
		keys:  []ebiten.Key{ebiten.KeyO},
		chars: []rune{'o'},
	})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyO}})

	top := ta.desk.topModal()
	if top == nil || top.caption != "Open" {
		t.Fatalf("no Open dialog: %v", top)
	}

	if got := top.find("FileName").(*edit).text(); got != "" {
		t.Errorf("the file name box has %q", got)
	}
}

// Alt and a letter that come down in the same frame open the menu, which
// releasing Alt leaves open. Ebitengine lists them letter first.
func TestAltAndLetterInOneFrame(t *testing.T) {
	ta := newTestApp(t)
	ta.music = &fakeMusic{}
	ta.startGame()
	ta.frame(input{
		keys:  []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft},
		chars: []rune{'m'},
		mods:  mods{alt: true},
	})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft}})

	if ta.desk.menu == nil {
		t.Fatal("releasing Alt closed the menu")
	}

	ta.frame(input{
		keys:  []ebiten.Key{ebiten.KeyO},
		chars: []rune{'o'},
	})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyO}})

	if top := ta.desk.topModal(); top == nil || top.caption != "Open" {
		t.Fatalf("no Open dialog: %v", top)
	}
}
