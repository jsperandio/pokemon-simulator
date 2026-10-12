// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/audio"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
)

// testApp drives the user interface headlessly, a frame at a time, on a
// clock of its own.
type testApp struct {
	*App
	t     *testing.T
	now   time.Time
	dir   string // the file dialogs' folder
	mouse image.Point
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	raw, err := assets.FS.ReadFile("pokegame.dat")
	if err != nil {
		t.Fatal(err)
	}
	data, err := gamedata.Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	a, err := New(data, platform.Disk(dir), nil)
	if err != nil {
		t.Fatal(err)
	}
	a.g.Rand.Seed = 1
	return &testApp{App: a, t: t, now: time.Date(2004, 1, 1, 12, 0, 0, 0, time.UTC), dir: dir}
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

// click clicks at p on the screen; double clicks twice.
func (ta *testApp) click(p image.Point) {
	ta.mouse = p
	ta.frame(input{})
	ta.frame(input{down: true, pressed: true})
	ta.frame(input{released: true})
}

func (ta *testApp) doubleClick(p image.Point) {
	ta.click(p)
	ta.click(p)
}

// clickOn clicks the middle of a window's control.
func (ta *testApp) clickOn(w *window, name string) {
	ta.t.Helper()
	k := w.find(name)
	if k == nil {
		ta.t.Fatalf("%s has no %s", w.name, name)
	}
	c := k.base()
	ta.click(widgetOrigin(k).Add(c.r.Size().Div(2)))
}

// key presses and releases a key.
func (ta *testApp) key(k ebiten.Key) {
	ta.frame(input{keys: []ebiten.Key{k}})
	ta.frame(input{keysUp: []ebiten.Key{k}})
}

// typeText types characters.
func (ta *testApp) typeText(s string) {
	ta.frame(input{chars: []rune(s)})
}

// shot paints the screen and, when $CLASSIC_SHOTS names a folder, saves
// it there as name.png.
func (ta *testApp) shot(name string) *image.RGBA {
	ta.t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	ta.desk.paint(img)
	dir := os.Getenv("CLASSIC_SHOTS")
	if dir == "" {
		return img
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		ta.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".png"), b.Bytes(), 0o644); err != nil {
		ta.t.Fatal(err)
	}
	return img
}

// TestNewGame plays the start of a game: loading, the start screen, the
// save file, the name and the starter.
func TestNewGame(t *testing.T) {
	ta := newTestApp(t)
	ta.wait(20)
	ta.shot("01-loading")
	if !ta.load.w.visible {
		t.Fatal("no loading screen")
	}
	ta.wait(60)
	if ta.load.w.visible || !ta.start.w.visible {
		t.Fatal("the start screen didn't follow the loading screen")
	}
	ta.shot("02-start")
	ta.clickOn(ta.start.w, "NewButton")
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
	if top := ta.desk.topModal(); top != ta.name.w {
		t.Fatalf("no name dialog: %v", top)
	}
	ta.typeText("Gary")
	ta.key(ebiten.KeyEnter)
	ta.wait(1)
	ta.shot("05-starters")
	if want := "POKéMON Simulator V4.5 - Gary [ash.pgs]"; ta.main.w.caption != want {
		t.Errorf("caption %q, want %q", ta.main.w.caption, want)
	}
	if !ta.main.starters.shown() {
		t.Fatal("no starters")
	}
	// Charmander, the second starter.
	r := ta.main.starters.itemRect(1)
	ta.click(widgetOrigin(ta.main.starters).Add(r.Min).Add(image.Pt(20, 20)))
	ta.wait(1)
	ta.shot("06-chosen")
	if got := ta.g.Party(); len(got) != 1 || got[0].Species != 4 {
		t.Fatalf("party %v", got)
	}
	if ta.desk.active != ta.main.w || !ta.stats.w.visible || !ta.dex.w.visible {
		t.Errorf("windows: active %v, Statistics %v, POKéDEX %v", ta.desk.active, ta.stats.w.visible, ta.dex.w.visible)
	}
	ta.mouse = image.Pt(0, 0)
	ta.wait(1)
	ta.shot("07-game")
	if !strings.HasSuffix(ta.g.SaveFile, "ash.pgs") {
		t.Errorf("save file %q", ta.g.SaveFile)
	}
}

// menu clicks through the game window's menu: a bar item, then submenu
// items, by name.
func (ta *testApp) menu(names ...string) {
	ta.t.Helper()
	m := ta.main.w.menu
	for i, e := range m.entries {
		if e.name() == names[0] && (!m.active() || m.sel != i || len(m.open) == 0) {
			r := m.barRects()[i]
			ta.click(r.Min.Add(r.Size().Div(2)))
		}
	}
	for _, n := range names[1:] {
		found := false
		for _, p := range m.open {
			for i, e := range p.items {
				if e.name() == n {
					r := p.itemRect(i)
					ta.click(r.Min.Add(r.Size().Div(2)))
					found = true
				}
			}
		}
		if !found {
			ta.t.Fatalf("menu item %s not open", n)
		}
	}
}

// answer closes the modal message dialog with its default button.
func (ta *testApp) answer(want string) {
	ta.t.Helper()
	w := ta.desk.topModal()
	if w == nil {
		ta.t.Fatalf("no dialog; want %q", want)
	}
	if msg := w.find("Message"); msg == nil || !strings.Contains(msg.base().caption, want) {
		ta.t.Fatalf("dialog %q, want %q", msg.base().caption, want)
	}
	ta.key(ebiten.KeyEnter)
	ta.wait(1)
}

// startGame plays up to the game window with a Charmander.
func (ta *testApp) startGame() {
	ta.wait(80)
	ta.clickOn(ta.start.w, "NewButton")
	ta.typeText("ash")
	ta.key(ebiten.KeyEnter)
	ta.typeText("Gary")
	ta.key(ebiten.KeyEnter)
	r := ta.main.starters.itemRect(1)
	ta.click(widgetOrigin(ta.main.starters).Add(r.Min).Add(image.Pt(20, 20)))
	ta.wait(1)
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
	r := ta.stats.party.itemRect(0)
	ta.doubleClick(widgetOrigin(ta.stats.party).Add(r.Min).Add(image.Pt(10, 10)))
	ta.shot("13-info")
	ta.answer("# 004 Charmander, L.5, XP:125, XP to next level:91")
	ta.shot("14-confirm")
	ta.answer("Choose Charmander?")
	ta.shot("15-fighter")
	if !ta.main.goButton.shown() {
		t.Fatal("no Go! button")
	}
	ta.click(widgetOrigin(ta.main.attacks).Add(ta.main.attacks.buttonRect(0).Min).Add(image.Pt(20, 5)))
	ta.clickOn(ta.main.w, "Button1")
	ta.wait(1)
	ta.shot("16-turn")
	if len(ta.g.Info.Lines) < 4 {
		t.Errorf("Battle Info: %q", ta.g.Info.Lines)
	}
	ta.clickOn(ta.stats.w, "Button1") // a POKéBall
	ta.wait(1)
	ta.shot("17-ball")
	for ta.desk.topModal() != nil {
		ta.key(ebiten.KeyEnter)
		ta.wait(1)
	}
	ta.shot("18-after")
	t.Logf("log: %q", ta.g.Info.Lines)
}

// TestCatchSaveLoad catches a POKéMON, quits, which saves, and loads the
// save in a new run.
func TestCatchSaveLoad(t *testing.T) {
	ta := newTestApp(t)
	ta.startGame()
	ta.g.MasterBalls = 1
	ta.menu("Action1", "FightWild1", "FindPKMN1")
	ta.answer("Select a POKéMON")
	r := ta.stats.party.itemRect(0)
	ta.doubleClick(widgetOrigin(ta.stats.party).Add(r.Min).Add(image.Pt(10, 10)))
	ta.answer("# 004 Charmander")
	ta.answer("Choose Charmander?")
	ta.clickOn(ta.stats.w, "Button4") // the Master Ball
	ta.wait(1)
	ta.shot("20-caught")
	if !ta.main.endButton.shown() {
		t.Fatalf("no End Fight; log %q", ta.g.Info.Lines)
	}
	ta.clickOn(ta.main.w, "EndButton")
	ta.wait(1)
	ta.shot("21-ended")
	if n := len(ta.g.Party()); n != 2 {
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
	tb.files = platform.Disk(ta.dir)
	tb.wait(80)
	tb.clickOn(tb.start.w, "LoadButton")
	tb.wait(1)
	tb.shot("22-open")
	if top := tb.desk.topModal(); top == nil || top.caption != "Open" {
		t.Fatalf("no open dialog: %v", top)
	}
	tb.typeText("ash")
	tb.key(ebiten.KeyEnter)
	tb.wait(1)
	tb.shot("23-loaded")
	if n := len(tb.g.Party()); n != 2 || tb.g.PlayerName != "Gary" {
		t.Fatalf("loaded %q with a party of %d", tb.g.PlayerName, n)
	}
	if tb.start.w.visible || !tb.main.w.visible || !tb.stats.w.visible {
		t.Error("the game windows don't show")
	}
	if got := tb.main.w.pos; got != image.Pt(95, 54) {
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

func (m *fakeMusic) Open(name string, data []byte) error {
	m.opened = append(m.opened, name+":"+string(data))
	return m.err
}

func (m *fakeMusic) Stop() error {
	m.stopped++
	return m.stopErr
}

func (m *fakeMusic) Update(repeat bool) { m.repeat = repeat }

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

	if err := os.WriteFile(filepath.Join(ta.dir, "song.ogg"), []byte("OggS"), 0o644); err != nil {
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
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyAltLeft}, mods: mods{alt: true}})
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyM}, chars: []rune{'m'}, mods: mods{alt: true}})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft}})
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyO}, chars: []rune{'o'}})
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
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft}, chars: []rune{'m'}, mods: mods{alt: true}})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft}})
	if ta.desk.menu == nil {
		t.Fatal("releasing Alt closed the menu")
	}
	ta.frame(input{keys: []ebiten.Key{ebiten.KeyO}, chars: []rune{'o'}})
	ta.frame(input{keysUp: []ebiten.Key{ebiten.KeyO}})
	if top := ta.desk.topModal(); top == nil || top.caption != "Open" {
		t.Fatalf("no Open dialog: %v", top)
	}
}
