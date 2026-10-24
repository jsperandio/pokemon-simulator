// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/audio"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
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
// the command line's arguments args, and its settings turning the
// animations off: the tests of the animations turn them on.
func newTestAppIn(t *testing.T, dir string, args []string) *testApp {
	t.Helper()

	off := &memorySettings{
		data: []byte("[Graphics]\nAnimatedSprites=0\n"),
	}

	return newTestAppWith(t, dir, args, off)
}

// newTestAppWith makes the user interface as newTestAppIn does, with the
// given settings.
func newTestAppWith(t *testing.T, dir string, args []string, settings *memorySettings) *testApp {
	t.Helper()

	app, err := NewApplication(testData(t), platform.NewDisk(dir), settings, args)
	require.NoError(t, err)

	app.game.Rand.Seed = 1
	// The shinies are the port's: none comes, and the POKéDEX's "Shiny"
	// box hides, so the windows look as the original's captures. The
	// tests of the shinies bring them back.
	app.game.RollShiny = nil
	app.pokedex.shiny.setVisible(false)

	return &testApp{
		Application: app,
		t:           t,
		now:         time.Date(2004, 1, 1, 12, 0, 0, 0, time.UTC),
		dir:         dir,
	}
}

// memorySettings keeps the settings in memory, as the tests' settings
// file; err is what writing it returns.
type memorySettings struct {
	data []byte
	err  error
}

func (ms *memorySettings) Read() ([]byte, error) {
	if ms.data == nil {
		return nil, platform.ErrNotExist
	}

	return ms.data, nil
}

func (ms *memorySettings) Write(data []byte) error {
	if ms.err != nil {
		return ms.err
	}

	ms.data = slices.Clone(data)

	return nil
}

// testData returns the game's data.
func testData(t *testing.T) *gamedata.Data {
	t.Helper()

	raw, err := assets.FS.ReadFile("pokegame.dat")
	require.NoError(t, err)

	data, err := gamedata.Load(raw)
	require.NoError(t, err)

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
	ta.frame(input{
		released: true,
	})
}

func (ta *testApp) doubleClick(pnt image.Point) {
	ta.click(pnt)
	ta.click(pnt)
}

// clickOn clicks the middle of a window's control.
func (ta *testApp) clickOn(wdw *window, name string) {
	ta.t.Helper()

	wgt := wdw.find(name)
	require.NotNil(ta.t, wgt, "%s has no %s", wdw.name, name)

	ctl := wgt.base()
	ta.click(widgetOrigin(wgt).Add(ctl.rect.Size().Div(2)))
}

// showWindow shows a window of comp's controls, over the loading screen.
func (ta *testApp) showWindow(comp *layout.Component) *window {
	wdw := newWindow(ta.desk, comp)
	ta.desk.add(wdw)
	ta.desk.show(wdw)
	ta.wait(1)

	return wdw
}

// key presses and releases a key.
func (ta *testApp) key(key ebiten.Key) {
	ta.frame(input{
		keys: []ebiten.Key{key},
	})
	ta.frame(input{
		keysUp: []ebiten.Key{key},
	})
}

// altF4 presses Alt+F4.
func (ta *testApp) altF4() {
	ta.frame(input{
		keys: []ebiten.Key{ebiten.KeyAltLeft},
		mods: mods{
			alt: true,
		},
	})
	ta.frame(input{
		keys: []ebiten.Key{ebiten.KeyF4},
		mods: mods{
			alt: true,
		},
	})
	ta.frame(input{
		keysUp: []ebiten.Key{ebiten.KeyF4, ebiten.KeyAltLeft},
	})
}

// typeText types characters.
func (ta *testApp) typeText(txt string) {
	ta.frame(input{
		chars: []rune(txt),
	})
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

		require.True(ta.t, found, "menu item %s not open", nam)
	}
}

// answer closes the modal message dialog with its default button.
func (ta *testApp) answer(want string) {
	ta.t.Helper()

	wdw := ta.desk.topModal()
	require.NotNil(ta.t, wdw, "no dialog; want %q", want)

	msg := wdw.find("Message")
	require.NotNil(ta.t, msg, "dialog %q has no message; want %q", wdw.caption, want)
	require.Contains(ta.t, msg.base().caption, want, "dialog %q", wdw.caption)

	ta.key(ebiten.KeyEnter)
	ta.wait(1)
}

// startGame plays up to the game window with a Charmander.
func (ta *testApp) startGame() {
	ta.chooseStarters()
	ta.pickStarter(1)
}

// chooseStarters plays up to the starter choice.
func (ta *testApp) chooseStarters() {
	ta.wait(80)
	ta.clickOn(ta.start.wdw, "NewButton")
	ta.typeText("ash")
	ta.key(ebiten.KeyEnter)
	ta.typeText("Gary")
	ta.key(ebiten.KeyEnter)
	ta.wait(1)
}

// pickStarter clicks starter idx.
func (ta *testApp) pickStarter(idx int) {
	rect := ta.main.starters.itemRect(idx)
	ta.click(widgetOrigin(ta.main.starters).Add(rect.Min).Add(image.Pt(20, 20)))
	ta.wait(1)
}

// openSong opens song.ogg from the Music menu's Open file dialog, once
// writeSong has put it in the folder.
func (ta *testApp) openSong() {
	ta.t.Helper()

	ta.menu("SelectMusic1", "Openfile1")
	ta.wait(1)
	ta.typeText("song.ogg")
	ta.key(ebiten.KeyEnter)
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

	require.Fail(ta.t, "the page has no such link", "%s has no link to %s", hv.name, target)

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
	eff := menuEffects{
		quit: ta.quit,
	}
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

// Test_NewGame plays the start of a game: loading, the start screen, the
// save file, the name and the starter.
func Test_NewGame(t *testing.T) {
	t.Parallel()

	t.Run("when the player starts a new game and chooses a starter, should open the game with it", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.wait(20)
		ta.shot("01-loading")

		require.True(t, ta.load.wdw.visible, "no loading screen")

		ta.wait(60)

		require.False(t, ta.load.wdw.visible, "the loading screen is still up")
		require.True(t, ta.start.wdw.visible, "the start screen didn't follow the loading screen")

		ta.shot("02-start")
		ta.clickOn(ta.start.wdw, "NewButton")
		ta.wait(1)
		ta.shot("03-save-dialog")

		top := ta.desk.topModal()
		require.NotNil(t, top, "no save dialog")
		require.Equal(t, "Save As", top.caption, "the top dialog")

		ta.typeText("ash")
		ta.key(ebiten.KeyEnter)
		ta.wait(1)
		ta.shot("04-name")

		require.Same(t, ta.name.wdw, ta.desk.topModal(), "no name dialog")

		ta.typeText("Gary")
		ta.key(ebiten.KeyEnter)
		ta.wait(1)
		ta.shot("05-starters")

		assert.Equal(t, "POKéMON Simulator V4.5 - Gary [ash.pgs]", ta.main.wdw.caption)
		require.True(t, ta.main.starters.shown(), "no starters")

		// Charmander, the second starter.
		rect := ta.main.starters.itemRect(1)
		ta.click(widgetOrigin(ta.main.starters).Add(rect.Min).Add(image.Pt(20, 20)))
		ta.wait(1)
		ta.shot("06-chosen")

		got := ta.game.Party()
		require.Len(t, got, 1)
		require.Equal(t, int32(4), got[0].Species)

		assert.Same(t, ta.main.wdw, ta.desk.active, "the active window")
		assert.True(t, ta.stats.wdw.visible, "the Statistics window doesn't show")
		assert.True(t, ta.pokedex.wdw.visible, "the POKéDEX window doesn't show")

		ta.mouse = image.Pt(0, 0)
		ta.wait(1)
		ta.shot("07-game")

		assert.True(t, strings.HasSuffix(ta.game.SaveFile, "ash.pgs"), "save file %q", ta.game.SaveFile)
	})
}

func Test_WildBattle(t *testing.T) {
	t.Parallel()

	t.Run("when the player fights a wild POKéMON, should log the battle and close its dialogs", func(t *testing.T) {
		t.Parallel()

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

		require.True(t, ta.main.goButton.shown(), "no Go! button")

		attacks := ta.main.attacks
		ta.click(widgetOrigin(attacks).Add(attacks.buttonRect(0).Min).Add(image.Pt(20, 5)))
		ta.clickOn(ta.main.wdw, "Button1")
		ta.wait(1)
		ta.shot("16-turn")

		assert.GreaterOrEqual(t, len(ta.game.Info.Lines), 4, "Battle Info: %q", ta.game.Info.Lines)

		ta.clickOn(ta.stats.wdw, "Button1") // a POKéBall
		ta.wait(1)
		ta.shot("17-ball")

		for ta.desk.topModal() != nil {
			ta.key(ebiten.KeyEnter)
			ta.wait(1)
		}

		ta.shot("18-after")
		t.Logf("log: %q", ta.game.Info.Lines)
	})

	t.Run("when the arena's sprites are clicked, should neither select nor focus them", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.menu("Action1", "FightWild1", "FindPKMN1")
		ta.answer("Select a POKéMON from the Statistics Window")

		rect := ta.stats.party.itemRect(0)
		ta.doubleClick(widgetOrigin(ta.stats.party).Add(rect.Min).Add(image.Pt(10, 10)))
		ta.answer("# 004 Charmander")
		ta.answer("Choose Charmander?")

		for _, view := range []*listView{ta.main.ownImage, ta.main.enemyImage} {
			ta.click(widgetOrigin(view).Add(view.itemRect(0).Min).Add(image.Pt(12, 12)))
			ta.wait(1)

			assert.Equal(t, -1, view.selected, "%s is selected", view.name)
			assert.NotSame(t, view, ta.main.wdw.focus, "%s has the focus", view.name)
		}

		ta.shot("arena-clicked")
	})
}

// fightWild plays from the game window to a battle Search for a POKéMON
// found, with Charmander out.
func (ta *testApp) fightWild() {
	ta.t.Helper()

	ta.menu("Action1", "FightWild1", "FindPKMN1")
	ta.answer("Select a POKéMON from the Statistics Window")
	ta.doubleClickParty(0)
	ta.answer("# 004 Charmander")
	ta.answer("Choose Charmander?")
}

// doubleClickParty double-clicks party member idx in the Statistics
// window.
func (ta *testApp) doubleClickParty(idx int) {
	rect := ta.stats.party.itemRect(idx)
	ta.doubleClick(widgetOrigin(ta.stats.party).Add(rect.Min).Add(image.Pt(10, 10)))
}

// clickAttack clicks attack idx in the Battle Arena.
func (ta *testApp) clickAttack(idx int) {
	attacks := ta.main.attacks
	ta.click(widgetOrigin(attacks).Add(attacks.buttonRect(idx).Min).Add(image.Pt(20, 5)))
}

// winBattle wins the battle with the attack selected: the enemy has 1 HP
// left, and Go! is clicked.
func (ta *testApp) winBattle() {
	ta.t.Helper()

	ta.game.EnemyHP = 1

	ta.clickOn(ta.main.wdw, "Button1")
	ta.wait(1)
	require.True(ta.t, ta.game.Main.EndFight, "the battle goes on")
}

// Search Again, which the original doesn't have, ends a battle a search
// found and searches again, with the same fighter, which another party
// member can replace until it acts. F3 is Search for a POKéMON's
// shortcut, and Search Again's once the battle is over. Go! has the focus
// whenever a fighter comes out, and Enter presses it.
func Test_SearchAgain(t *testing.T) {
	t.Parallel()

	t.Run("when a fighter comes out, should give Go! the focus, with the game window active", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.fightWild()

		assert.Same(t, ta.main.wdw, ta.desk.active, "the active window")
		assert.Same(t, ta.main.goButton, ta.main.wdw.focus, "the focus")
	})

	t.Run("when an attack has the focus, should attack with Enter", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.fightWild()
		ta.clickAttack(0)
		require.Same(t, ta.main.attacks, ta.main.wdw.focus, "the attacks don't have the focus")

		lines := len(ta.game.Info.Lines)

		ta.key(ebiten.KeyEnter)
		ta.wait(1)
		require.Greater(t, len(ta.game.Info.Lines), lines, "Enter didn't attack")
		assert.Equal(t, "Charmander used Scratch!", ta.game.Info.Lines[lines])
		ta.shot("go-default")
	})

	t.Run("when a battle a search found is over, should show Search Again where Go! was, and start the next battle at a click with the same fighter and attack", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.fightWild()
		ta.clickAttack(1)
		ta.winBattle()

		require.True(t, ta.main.searchButton.shown(), "no Search Again")
		assert.False(t, ta.main.goButton.shown(), "Go! shows")
		assert.True(t, ta.main.endButton.shown(), "no End Fight")
		ta.shot("search-again")

		mgw := &ta.game.Main
		appeared := mgw.Appeared

		ta.clickOn(ta.main.wdw, "SearchAgainButton")
		ta.wait(1)
		assert.Nil(t, ta.desk.topModal(), "a dialog shows")
		assert.Equal(t, appeared+1, mgw.Appeared, "no wild POKéMON appeared")
		assert.Equal(t, 4, mgw.Own.Species, "the fighter")
		assert.Equal(t, 1, ta.main.attacks.index, "the attack selected")
		assert.Same(t, ta.main.goButton, ta.main.wdw.focus, "the focus")
		assert.False(t, ta.main.searchButton.shown(), "Search Again shows in the battle")
	})

	t.Run("when F3 is pressed with the battle over, should search again", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.fightWild()
		ta.clickAttack(0)
		ta.winBattle()

		appeared := ta.game.Main.Appeared

		ta.key(ebiten.KeyF3)
		ta.wait(1)
		assert.Nil(t, ta.desk.topModal(), "a dialog shows")
		assert.Equal(t, appeared+1, ta.game.Main.Appeared, "no wild POKéMON appeared")
		assert.True(t, ta.main.goButton.shown(), "no Go!")
	})

	t.Run("when Ctrl+F3 is pressed with the battle over, should do nothing", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.fightWild()
		ta.clickAttack(0)
		ta.winBattle()

		appeared := ta.game.Main.Appeared

		ta.frame(input{
			keys: []ebiten.Key{ebiten.KeyF3},
			mods: mods{
				ctrl: true,
			},
		})
		ta.frame(input{
			keysUp: []ebiten.Key{ebiten.KeyF3},
		})
		ta.wait(1)
		assert.Equal(t, appeared, ta.game.Main.Appeared, "a wild POKéMON appeared")
		assert.True(t, ta.main.searchButton.shown(), "no Search Again")
	})

	t.Run("when F3 is pressed in the game window with no battle, should search for a POKéMON", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.desk.activate(ta.main.wdw)
		ta.key(ebiten.KeyF3)
		ta.wait(1)
		ta.answer("Select a POKéMON from the Statistics Window")

		assert.True(t, ta.game.InWildBattle)
	})

	t.Run("when F3 is pressed during a battle, should do nothing", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.fightWild()

		appeared := ta.game.Main.Appeared

		ta.key(ebiten.KeyF3)
		ta.wait(1)
		assert.Nil(t, ta.desk.topModal(), "a dialog shows")
		assert.Equal(t, appeared, ta.game.Main.Appeared, "a wild POKéMON appeared")
		assert.True(t, ta.main.goButton.shown(), "the battle is over")
	})

	t.Run("when Search Again has sent the fighter out, should send another party member double-clicked before it acts", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()

		ta.game.InPokedex[25] = true // so no POKéDEX opens over the party

		ta.game.AddPKMN(25, "Pikachu", 50, false)
		ta.wait(1)
		ta.fightWild()
		ta.clickAttack(0)
		ta.winBattle()
		ta.clickOn(ta.main.wdw, "SearchAgainButton")
		ta.wait(1)
		ta.doubleClickParty(1)
		ta.answer("# 025 Pikachu")
		ta.answer("Choose Pikachu?")

		assert.Equal(t, 25, ta.game.Main.Own.Species, "the fighter")
		assert.Equal(t, -1, ta.main.attacks.index, "the attack selected")
		assert.Same(t, ta.main.wdw, ta.desk.active, "the active window")
		assert.Same(t, ta.main.goButton, ta.main.wdw.focus, "the focus")
		ta.shot("search-again-swapped")
	})
}

// wildBattle starts a wild battle, at the game window, as a game handler.
func wildBattle(ta *testApp) {
	ta.call(func() {
		ta.game.StartWildBattle(25, 50, false)
	})
	ta.wait(1)
}

func Test_Application_Message(t *testing.T) {
	t.Parallel()

	t.Run("when the choose prompt closes with its box checked, should skip it from then on and keep the choice", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		wildBattle(ta)
		ta.clickOn(ta.desk.topModal(), "DontShowAgain")
		ta.shot("prompt-checked")
		ta.answer(game.ChoosePrompt)

		assert.False(t, ta.preferences.choosePrompt)
		assert.Contains(t, string(ta.settings.(*memorySettings).data), "[Messages]\nSelectPokemon=0\n")

		ta.game.EndFight()
		wildBattle(ta)
		assert.Nil(t, ta.desk.topModal(), "a wild battle shows the prompt")
		assert.True(t, ta.game.Main.Battle, "the wild battle didn't start")

		ta.game.EndFight()
		ta.call(func() {
			ta.game.StartTrainerBattle(74, 12, "Brock")
		})
		ta.wait(1)
		assert.Nil(t, ta.desk.topModal(), "a trainer battle shows the prompt")
	})

	t.Run("when the choose prompt closes with its box cleared, should show it the next time", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		wildBattle(ta)
		ta.answer(game.ChoosePrompt)
		ta.game.EndFight()
		wildBattle(ta)

		assert.True(t, ta.preferences.choosePrompt)
		ta.answer(game.ChoosePrompt)
	})

	t.Run("when the choose prompt closes with Escape, its box checked, should still skip it", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		wildBattle(ta)
		ta.typeText("d")
		ta.key(ebiten.KeyEscape)
		ta.wait(1)

		assert.Nil(t, ta.desk.topModal(), "Escape didn't close the prompt")
		assert.False(t, ta.preferences.choosePrompt)
	})

	t.Run("when the choice can't be kept, should still skip the prompt for the run", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.settings.(*memorySettings).err = errors.New("disk full")
		ta.startGame()
		wildBattle(ta)
		ta.clickOn(ta.desk.topModal(), "DontShowAgain")
		ta.answer(game.ChoosePrompt)

		assert.Nil(t, ta.desk.topModal(), "an error shows")
		assert.False(t, ta.preferences.choosePrompt)
	})

	t.Run("when another message shows, should have no check box", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.call(func() {
			ta.Message(game.Information, false, "Gotcha!")
		})
		ta.wait(1)

		require.NotNil(t, ta.desk.topModal())
		assert.Nil(t, ta.desk.topModal().find("DontShowAgain"))
		ta.answer("Gotcha!")
	})
}

// Test_CatchSaveLoad catches a POKéMON, quits, which saves, and loads the
// save in a new run.
func Test_CatchSaveLoad(t *testing.T) {
	t.Parallel()

	t.Run("when a POKéMON is caught and the game quit, should load the save in a new run", func(t *testing.T) {
		t.Parallel()

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

		require.True(t, ta.main.endButton.shown(), "no End Fight; log %q", ta.game.Info.Lines)

		ta.clickOn(ta.main.wdw, "EndButton")
		ta.wait(1)
		ta.shot("21-ended")

		require.Len(t, ta.game.Party(), 2)

		ta.menu("File1", "Quit1")
		ta.wait(1)

		require.True(t, ta.quit, "still running after Quit")

		saved := filepath.Join(ta.dir, "ash.pgs")
		require.FileExists(t, saved)

		tb := newTestApp(t)
		tb.files = platform.NewDisk(ta.dir)
		tb.wait(80)
		tb.clickOn(tb.start.wdw, "LoadButton")
		tb.wait(1)
		tb.shot("22-open")

		top := tb.desk.topModal()
		require.NotNil(t, top, "no open dialog")
		require.Equal(t, "Open", top.caption, "the top dialog")

		tb.typeText("ash")
		tb.key(ebiten.KeyEnter)
		tb.wait(1)
		tb.shot("23-loaded")

		require.Len(t, tb.game.Party(), 2)
		require.Equal(t, "Gary", tb.game.PlayerName)

		assert.False(t, tb.start.wdw.visible, "the start screen still shows")
		assert.True(t, tb.main.wdw.visible, "the game window doesn't show")
		assert.True(t, tb.stats.wdw.visible, "the Statistics window doesn't show")
		assert.Equal(t, image.Pt(74, 54), tb.main.wdw.pos, "the game window's position")
	})
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

// writeSong puts song.ogg in dir, the folder the Open dialog starts in.
func writeSong(t *testing.T, dir string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, "song.ogg"), []byte("OggS"), 0o600)
	require.NoError(t, err)
}

// The system's request to close the program's window closes the game, as
// the game window's close does, unless it comes with the Alt+F4 that the
// desktop gives its active window: Windows sends one with the other.
func Test_Application_closeRequested(t *testing.T) {
	t.Parallel()

	t.Run("when the request comes with Alt+F4, should close only the active window and keep the game", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.menu("Action1", "Buy1")
		ta.wait(1)

		require.Same(t, ta.pokemart.wdw, ta.desk.active, "the active window")

		in := input{
			keys: []ebiten.Key{ebiten.KeyF4},
			mods: mods{
				alt: true,
			},
		}
		ta.closeRequested(&in)
		ta.frame(in)

		assert.False(t, ta.quit, "the game quit")
		assert.False(t, ta.pokemart.wdw.visible, "the Mart shows")
		assert.True(t, ta.main.wdw.visible, "the game window hid")
	})

	t.Run("when the request comes without Alt+F4, should save the game and quit", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.menu("Action1", "Buy1")
		ta.wait(1)

		ta.closeRequested(&input{})
		ta.wait(1)

		assert.True(t, ta.quit, "the game runs")
		assert.FileExists(t, filepath.Join(ta.dir, "ash.pgs"))
	})
}

// Test_MusicMenu opens music from the Music menu, stops it, and turns
// Repeat Music off; the player's errors show as the original's exceptions.
func Test_MusicMenu(t *testing.T) {
	t.Parallel()

	t.Run("when Stop Music is chosen with no music open, should show the media player's error", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.music = &fakeMusic{
			stopErr: audio.ErrNotOpen,
		}
		ta.startGame()
		ta.wait(1)

		ta.menu("SelectMusic1", "StopMusic1")
		ta.wait(1)
		ta.shot("40-stop-music")

		top := ta.desk.topModal()
		require.NotNil(t, top, "no error box")
		require.Equal(t, "POKéMON Simulator", top.caption, "the top dialog")

		ta.answer("No MCI device open.")
	})

	t.Run("when a song is opened from the menu, should open it", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		fm := &fakeMusic{}
		ta.music = fm
		ta.startGame()
		ta.wait(1)
		writeSong(t, ta.dir)

		ta.menu("SelectMusic1", "Openfile1")
		ta.wait(1)
		ta.shot("41-open-music")
		ta.typeText("song.ogg")
		ta.key(ebiten.KeyEnter)
		ta.wait(1)

		require.Equal(t, []string{"song.ogg:OggS"}, fm.opened)
	})

	t.Run("when Stop Music is chosen once a song plays, should stop it with no error", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		fm := &fakeMusic{}
		ta.music = fm
		ta.startGame()
		ta.wait(1)
		writeSong(t, ta.dir)
		ta.openSong()

		ta.menu("SelectMusic1", "StopMusic1")
		ta.wait(1)

		assert.Equal(t, 1, fm.stopped, "Stop Music's stops")
		assert.Nil(t, ta.desk.topModal(), "Stop Music's dialog")
	})

	t.Run("when Repeat Music is on, should turn it off", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		fm := &fakeMusic{}
		ta.music = fm
		ta.startGame()
		ta.wait(1)

		assert.True(t, fm.repeat, "Repeat Music is off at the start")

		ta.menu("SelectMusic1", "RepeatMusic1")
		ta.wait(1)

		assert.False(t, fm.repeat, "Repeat Music is still on")
	})

	t.Run("when a file that can't play is opened, should say it can't be played", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.music = &fakeMusic{
			err: audio.ErrFormat,
		}
		ta.startGame()
		ta.wait(1)
		writeSong(t, ta.dir)

		ta.openSong()
		ta.shot("42-cannot-play")
		ta.answer("Cannot play song.ogg. Only MP3, Ogg Vorbis and WAV files can be played.")
	})
}

// Test_MenuKeys checks the keys that open the menus.
func Test_MenuKeys(t *testing.T) {
	t.Parallel()

	// A menu's keys type nothing in the dialog they open: Alt+M, then O,
	// whose character comes in the same frame as the key.
	t.Run("when Alt+M and then O open the Open dialog, should type nothing into its file name", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.music = &fakeMusic{}
		ta.startGame()
		ta.frame(input{
			keys: []ebiten.Key{ebiten.KeyAltLeft},
			mods: mods{
				alt: true,
			},
		})
		ta.frame(input{
			keys:  []ebiten.Key{ebiten.KeyM},
			chars: []rune{'m'},
			mods: mods{
				alt: true,
			},
		})
		ta.frame(input{
			keysUp: []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft},
		})
		ta.frame(input{
			keys:  []ebiten.Key{ebiten.KeyO},
			chars: []rune{'o'},
		})
		ta.frame(input{
			keysUp: []ebiten.Key{ebiten.KeyO},
		})

		top := ta.desk.topModal()
		require.NotNil(t, top, "no Open dialog")
		require.Equal(t, "Open", top.caption, "the top dialog")

		got := top.find("FileName").(*edit).text()
		assert.Empty(t, got, "the file name box")
	})

	// Alt and a letter that come down in the same frame open the menu, which
	// releasing Alt leaves open. Ebitengine lists them letter first.
	t.Run("when Alt and M come down in one frame, should keep the menu open after Alt is released", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.music = &fakeMusic{}
		ta.startGame()
		ta.frame(input{
			keys:  []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft},
			chars: []rune{'m'},
			mods: mods{
				alt: true,
			},
		})
		ta.frame(input{
			keysUp: []ebiten.Key{ebiten.KeyM, ebiten.KeyAltLeft},
		})

		require.NotNil(t, ta.desk.menu, "releasing Alt closed the menu")

		ta.frame(input{
			keys:  []ebiten.Key{ebiten.KeyO},
			chars: []rune{'o'},
		})
		ta.frame(input{
			keysUp: []ebiten.Key{ebiten.KeyO},
		})

		top := ta.desk.topModal()
		require.NotNil(t, top, "no Open dialog")
		require.Equal(t, "Open", top.caption, "the top dialog")
	})
}

// drag drags with the mouse from one point of the screen to another.
func (ta *testApp) drag(from, to image.Point) {
	ta.mouse = from
	ta.frame(input{
		down:    true,
		pressed: true,
	})

	ta.mouse = to
	ta.frame(input{
		down: true,
	})
	ta.frame(input{
		released: true,
	})
}

// fillApp is a test app whose preferences fill the screen.
func fillApp(t *testing.T) *testApp {
	t.Helper()

	ta := newTestApp(t)
	ta.preferences.fillScreen = true

	return ta
}

// Filling the screen docks the game, Statistics and Battle Info windows
// edge to edge over the 800×600 screen; the separate windows have their
// own sizes, side by side.
func Test_Application_placeWindows(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		window func(ta *testApp) *window
		want   image.Rectangle
	}{
		{
			name: "when the screen is filled, should dock Statistics down the right edge",
			window: func(ta *testApp) *window {
				return ta.stats.wdw
			},
			want: image.Rect(617, 0, 800, 600),
		},
		{
			name: "when the screen is filled, should dock Battle Info along the bottom, against Statistics",
			window: func(ta *testApp) *window {
				return ta.info.wdw
			},
			want: image.Rect(0, 416, 617, 600),
		},
		{
			name: "when the screen is filled, should give the game window the rest",
			window: func(ta *testApp) *window {
				return ta.main.wdw
			},
			want: image.Rect(0, 0, 617, 416),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ta := fillApp(t)
			ta.startGame()

			assert.Equal(t, tc.want, tc.window(ta).rect())
		})
	}

	t.Run("when the screen is filled, should stretch the arena's frame and keep its contents at their size, centred", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()

		arena := ta.main.arena.rect
		before := map[string]image.Rectangle{}

		for _, wgt := range ta.main.arena.kids {
			before[wgt.base().name] = wgt.base().rect
		}

		ta.preferences.fillScreen = true
		ta.placeWindows(246)

		assert.Equal(t, image.Rect(120, 6, 606, 365), ta.main.arena.rect, "the arena's frame, from %v", arena)

		for _, wgt := range ta.main.arena.kids {
			ctl := wgt.base()
			assert.Equal(t, before[ctl.name].Add(image.Pt(73, 51)), ctl.rect, ctl.name)
		}
	})

	t.Run("when the screen is filled, should grow the party list and the battle log with their windows", func(t *testing.T) {
		t.Parallel()

		ta := fillApp(t)
		ta.startGame()

		assert.Equal(t, image.Rect(0, 248, 177, 575), ta.stats.party.rect, "the party list")
		assert.Equal(t, image.Rect(0, 0, 611, 159), ta.info.memo.rect, "the battle log")
	})

	t.Run("when the screen is filled, should keep a window in place when its caption is dragged", func(t *testing.T) {
		t.Parallel()

		ta := fillApp(t)
		ta.startGame()

		from := ta.main.wdw.captionRect().Min.Add(image.Pt(100, 5))
		ta.drag(from, from.Add(image.Pt(50, 40)))

		assert.Equal(t, image.Point{}, ta.main.wdw.pos, "the pinned window")

		ta.main.wdw.pinned = false
		ta.drag(from, from.Add(image.Pt(50, 40)))

		assert.Equal(t, image.Pt(50, 40), ta.main.wdw.pos, "the same drag on the window unpinned")
	})

	t.Run("when the starter choice shows with the screen filled, should keep the game window in its place and centre the list", func(t *testing.T) {
		t.Parallel()

		ta := fillApp(t)
		ta.chooseStarters()

		assert.Equal(t, image.Rect(0, 0, 617, 416), ta.main.wdw.rect(), "the game window")
		assert.Equal(t, image.Rect(241, 83, 374, 300), ta.main.starters.rect, "the starter list")
		ta.shot("fill-starters")

		ta.pickStarter(0)

		assert.Equal(t, image.Point{}, ta.main.wdw.pos, "the game window after the choice")
	})

	t.Run("when the layout goes back to separate windows, should give the windows their own sizes and places", func(t *testing.T) {
		t.Parallel()

		ta := fillApp(t)
		ta.startGame()

		ta.preferences.fillScreen = false
		ta.placeWindows(246)

		assert.Equal(t, image.Pt(74, 54), ta.main.wdw.pos, "the game window's place")
		assert.Equal(t, image.Pt(464, 270), ta.main.wdw.client, "the game window's size")
		assert.Equal(t, image.Pt(177, 473), ta.stats.wdw.client, "Statistics' size")
		assert.Equal(t, image.Pt(464, 159), ta.info.wdw.client, "Battle Info's size")
		assert.Equal(t, image.Rect(0, 248, 177, 473), ta.stats.party.rect, "the party list")
		assert.Equal(t, image.Rect(120, 6, 459, 263), ta.main.arena.rect, "the arena's frame")
		assert.False(t, ta.main.wdw.pinned, "the game window is still pinned")
	})

	t.Run("when a battle shows with the screen filled, should look docked", func(t *testing.T) {
		t.Parallel()

		ta := fillApp(t)
		ta.startGame()
		ta.menu("Action1", "FightWild1", "FindPKMN1")
		ta.answer("Select a POKéMON from the Statistics Window")

		rect := ta.stats.party.itemRect(0)
		ta.doubleClick(widgetOrigin(ta.stats.party).Add(rect.Min).Add(image.Pt(10, 10)))
		ta.answer("# 004 Charmander")
		ta.answer("Choose Charmander?")

		assert.True(t, ta.info.wdw.visible, "Battle Info doesn't show")
		ta.shot("fill-battle")
	})
}
