// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js && !noanim

package classic

import (
	"image"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets/anim"
	"github.com/jsperandio/pokemon-simulator/internal/game"
)

// lugia's animation passes the slot's four sides.
const lugia int = 249

// appearingApp returns a game with the animations on, at the game window,
// once species has appeared, wild, behind its message box.
func appearingApp(t *testing.T, species int) *testApp {
	t.Helper()

	ta := newTestApp(t)
	ta.preferences.animatedSprites = true
	ta.startGame()
	ta.call(func() {
		ta.game.StartWildBattle(species, 50, false)
	})
	ta.wait(1)

	return ta
}

// chooseFighter chooses the first party member, startGame's Charmander,
// to fight.
func chooseFighter(ta *testApp) {
	ta.t.Helper()

	rect := ta.stats.party.itemRect(0)
	ta.doubleClick(widgetOrigin(ta.stats.party).Add(rect.Min).Add(image.Pt(10, 10)))
	ta.answer("# 004 Charmander")
	ta.answer("Choose Charmander?")
}

// framesOf returns how many of the test's frames, at 60 a second, the
// animation lasts.
func framesOf(spr *anim.Sprite) int {
	return int(spr.Duration()/(time.Second/60)) + 1
}

func Test_appearance(t *testing.T) {
	t.Parallel()

	t.Run("when a wild POKéMON appears, should wait for the message box to close", func(t *testing.T) {
		t.Parallel()

		ta := appearingApp(t, lugia)
		ta.wait(30)

		require.NotNil(t, ta.desk.topModal(), "no message box")
		assert.NotNil(t, ta.main.enemyAppear.sprite, "no animation armed")
		assert.False(t, ta.main.enemyAppear.playing())
		assert.Equal(t, []listItem{{"", lugia - 1, false}}, ta.main.enemyImage.items)
	})

	t.Run("when the message box closes, should play over the arena, then show the still sprite", func(t *testing.T) {
		t.Parallel()

		ta := appearingApp(t, lugia)
		ta.answer("Select a POKéMON")

		apr := &ta.main.enemyAppear
		require.True(t, apr.playing(), "not playing once the box closed")
		assert.Equal(t, []listItem{{"", -1, false}}, ta.main.enemyImage.items)

		spr := apr.sprite

		ta.wait(30)

		frm, on := spr.At(ta.now.Sub(apr.start))
		require.True(t, on)
		assert.Equal(t, frm, apr.frame)
		assertArenaFrame(t, ta, ta.main.enemyImage, spr, frm)

		ta.wait(framesOf(spr))
		assert.False(t, apr.playing())
		assert.Nil(t, apr.sprite)
		assert.Equal(t, []listItem{{"", lugia - 1, false}}, ta.main.enemyImage.items)

		ta.shot("appearance-after")
		ta.wait(1)
		assert.False(t, ta.desk.dirty, "still repainting after the animation")
	})

	t.Run("when a shiny POKéMON appears, should play its shiny animation", func(t *testing.T) {
		t.Parallel()

		ta := shinyWildApp(t, true)
		ta.answer("Select a POKéMON")

		shiny, err := anim.Load(25, true)
		require.NoError(t, err)

		apr := &ta.main.enemyAppear
		require.True(t, apr.playing(), "not playing once the box closed")
		assert.Equal(t, shiny.Sheet.Pix, apr.sprite.Sheet.Pix, "not the shiny colours")
	})

	t.Run("when another POKéMON appears, should play its animation", func(t *testing.T) {
		t.Parallel()

		ta := appearingApp(t, 25)
		ta.answer("Select a POKéMON")
		ta.wait(framesOf(ta.main.enemyAppear.sprite))
		require.False(t, ta.main.enemyAppear.playing())

		ta.game.EndFight()
		ta.call(func() {
			ta.game.StartWildBattle(25, 50, false)
		})
		ta.wait(1)
		ta.answer("Select a POKéMON")

		assert.True(t, ta.main.enemyAppear.playing(), "the same species again isn't played")
	})

	t.Run("when the arena hides while it plays, should stop", func(t *testing.T) {
		t.Parallel()

		ta := appearingApp(t, lugia)
		ta.answer("Select a POKéMON")
		require.True(t, ta.main.enemyAppear.playing())

		ta.game.Main.Battle = false
		ta.sync()
		ta.wait(1)

		assert.False(t, ta.main.enemyAppear.playing())
		assert.Nil(t, ta.main.enemyAppear.sprite)
	})

	t.Run("when the choose prompt is hidden, should play as soon as the POKéMON appears", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.preferences.animatedSprites = true
		ta.preferences.choosePrompt = false
		ta.startGame()
		ta.call(func() {
			ta.game.StartWildBattle(lugia, 50, false)
		})
		ta.wait(1)

		assert.Nil(t, ta.desk.topModal(), "the prompt shows")
		assert.True(t, ta.main.enemyAppear.playing(), "not playing with nothing over the arena")
		ta.shot("appearance-no-prompt")
	})

	t.Run("when the animations are off, should show the still sprite only", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()
		ta.call(func() {
			ta.game.StartWildBattle(lugia, 50, false)
		})
		ta.wait(1)
		ta.answer("Select a POKéMON")
		ta.wait(1)

		assert.False(t, ta.main.enemyAppear.playing())
		assert.Equal(t, []listItem{{"", lugia - 1, false}}, ta.main.enemyImage.items)
	})

	t.Run("when the animations are turned off while one plays, should stop it", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.preferences.animatedSprites = true
		ta.startGame()
		ta.call(func() {
			ta.game.StartWildBattle(lugia, 50, false)
		})
		ta.wait(1)
		ta.answer("Select a POKéMON")
		require.True(t, ta.main.enemyAppear.playing())

		ta.menu("File1", "Preferences1")
		ta.wait(1)

		pf := ta.prefsForm
		ta.clickOn(pf.wdw, "AnimatedSprites")
		require.True(t, ta.main.enemyAppear.playing(), "it ended before the choice")

		ta.clickOn(pf.wdw, "OKButton")

		assert.False(t, ta.main.enemyAppear.playing())
		assert.Equal(t, []listItem{{"", lugia - 1, false}}, ta.main.enemyImage.items)
	})

	t.Run("when the player chooses a fighter, should play its entry, then show its still sprite", func(t *testing.T) {
		t.Parallel()

		ta := appearingApp(t, 25)
		ta.answer("Select a POKéMON")
		ta.wait(framesOf(ta.main.enemyAppear.sprite))
		chooseFighter(ta)

		apr := &ta.main.ownAppear
		require.True(t, apr.playing(), "not playing once the fighter was chosen")
		assert.Equal(t, []listItem{{"", -1, false}}, ta.main.ownImage.items)

		spr := apr.sprite

		ta.wait(10)

		frm, on := spr.At(ta.now.Sub(apr.start))
		require.True(t, on)
		assert.Equal(t, frm, apr.frame)
		assertArenaFrame(t, ta, ta.main.ownImage, spr, frm)

		ta.wait(framesOf(spr))
		assert.False(t, apr.playing())
		assert.Equal(t, []listItem{{"", 3, false}}, ta.main.ownImage.items)
	})

	t.Run("when the battle ends while the fighter's entry plays, should stop and show no fighter", func(t *testing.T) {
		t.Parallel()

		ta := appearingApp(t, 25)
		ta.answer("Select a POKéMON")
		chooseFighter(ta)
		require.True(t, ta.main.ownAppear.playing())

		ta.game.Main.Battle = false
		ta.game.Main.Own = game.Shown{}
		ta.sync()
		ta.wait(1)

		assert.False(t, ta.main.ownAppear.playing())
		assert.Equal(t, []listItem{{"", -1, false}}, ta.main.ownImage.items)
	})
}

func Test_pokedexForm_appear(t *testing.T) {
	t.Parallel()

	t.Run("when a POKéMON is chosen, should play its animation, then show the still sprite", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		ta.preferences.animatedSprites = true
		pickCharmander(t, ta)

		dex := ta.pokedex
		apr := &dex.appear
		require.True(t, apr.playing(), "not playing once chosen")
		assert.Equal(t, []listItem{{"", -1, false}}, dex.image.items)

		spr := apr.sprite

		ta.wait(10)

		frm, on := spr.At(ta.now.Sub(apr.start))
		require.True(t, on)
		assert.Equal(t, frm, apr.frame)
		assertFrameDrawn(t, ta, dex.panel, []image.Rectangle{dex.panel.inside()}, dex.image, spr, frm)
		ta.shot("dex-playing")

		ta.wait(framesOf(spr))
		assert.False(t, apr.playing())
		assert.Equal(t, []listItem{{"", 3, false}}, dex.image.items)
	})

	t.Run("when another POKéMON is chosen while one plays, should play the new one from the start", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		ta.preferences.animatedSprites = true
		pickCharmander(t, ta)
		ta.wait(10)

		apr := &ta.pokedex.appear
		first := apr.start

		ta.key(ebiten.KeyDown) // Charmeleon
		ta.wait(1)

		require.Equal(t, 4, ta.game.Pokedex.Index)
		assert.True(t, apr.playing(), "not playing once chosen")
		assert.True(t, apr.start.After(first), "the animation didn't start again")
	})

	t.Run("when the POKéDEX closes while it plays, should stop", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		ta.preferences.animatedSprites = true
		pickCharmander(t, ta)
		require.True(t, ta.pokedex.appear.playing())

		ta.clickOn(ta.pokedex.wdw, "Button1")
		ta.wait(1)

		assert.False(t, ta.pokedex.appear.playing())
		assert.Nil(t, ta.pokedex.appear.sprite)
	})

	t.Run("when the animations are turned off while one plays, should stop it", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		ta.preferences.animatedSprites = true
		pickCharmander(t, ta)
		require.True(t, ta.pokedex.appear.playing())

		ta.setPreferences(defaultPreferences())
		ta.wait(1)
		require.True(t, ta.pokedex.appear.playing(), "stopped with the animations on")

		ta.setPreferences(preferences{
			animatedSprites: false,
			choosePrompt:    true,
		})
		ta.wait(1)

		assert.False(t, ta.pokedex.appear.playing())
		assert.Equal(t, []listItem{{"", 3, false}}, ta.pokedex.image.items)
	})

	t.Run("when the animations are off, should show the still sprite only", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		pickCharmander(t, ta)

		assert.False(t, ta.pokedex.appear.playing())
		assert.Equal(t, []listItem{{"", 3, false}}, ta.pokedex.image.items)
	})
}

func Test_groupBox_inside(t *testing.T) {
	t.Parallel()

	t.Run("when the arena's inside is asked for, should leave out its frame and caption", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		arena := ta.main.arena
		rect := arena.local()
		fnt := arena.font
		caption := image.Rect(8, 0, 8+fnt.width(arena.caption), fnt.height())
		inner := image.Rect(2, fnt.height()/2+1, rect.Max.X-2, rect.Max.Y-2)

		for _, part := range arena.inside() {
			assert.False(t, part.Overlaps(caption), "%v covers the caption %v", part, caption)
			assert.True(t, part.In(inner), "%v is outside the frame's inside %v", part, inner)
		}

		assert.True(t, inArena(arena, image.Pt(4, 20)), "below the caption")
		assert.True(t, inArena(arena, image.Pt(rect.Max.X-4, 8)), "beside the caption")
		assert.False(t, inArena(arena, image.Pt(20, 10)), "on the caption")
	})
}

// assertArenaFrame checks that the screen shows frame frm of spr where it
// stands in for view's sprite, inside the arena's frame.
func assertArenaFrame(t *testing.T, ta *testApp, view *listView, spr *anim.Sprite, frm int) {
	t.Helper()

	arena := ta.main.arena
	assertFrameDrawn(t, ta, arena, arena.inside(), view, spr, frm)
}

// assertFrameDrawn checks that the screen shows frame frm of spr where it
// stands in for view's sprite, inside host's rectangles inside, which are
// in host's coordinates.
func assertFrameDrawn(t *testing.T, ta *testApp, host widget, inside []image.Rectangle, view *listView, spr *anim.Sprite, frm int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	ta.desk.paint(img)

	corner := widgetOrigin(view).Add(spr.Offset).Sub(widgetOrigin(host))
	src := spr.Frame(frm)
	drawn := 0

	for y := range src.Dy() {
		for x := range src.Dx() {
			pix := spr.Sheet.NRGBAAt(src.Min.X+x, src.Min.Y+y)
			pnt := corner.Add(image.Pt(x, y)) // in host

			if pix.A == 0 || !slices.ContainsFunc(inside, pnt.In) {
				continue
			}

			scr := pnt.Add(widgetOrigin(host))

			got := img.RGBAAt(scr.X, scr.Y)
			if !assert.Equal(t, [3]uint8{pix.R, pix.G, pix.B}, [3]uint8{got.R, got.G, got.B}, "pixel %v", pnt) {
				return
			}

			drawn++
		}
	}

	assert.Positive(t, drawn, "no pixel of the frame is inside %s", host.base().name)
}

// inArena reports whether pnt, in the arena's coordinates, is inside it.
func inArena(arena *groupBox, pnt image.Point) bool {
	return slices.ContainsFunc(arena.inside(), pnt.In)
}
