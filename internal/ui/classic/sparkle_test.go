// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets/anim"
)

func Test_movingStars(t *testing.T) {
	t.Parallel()

	t.Run("when they start, should be the big star at the circle's bottom", func(t *testing.T) {
		t.Parallel()

		want := []star{
			{image.Pt(0, 24), 5},
		}
		assert.Equal(t, want, movingStars(0))
	})

	t.Run("when 4 frames have gone, should have the big one circling and crossing, and the next one starting", func(t *testing.T) {
		t.Parallel()

		want := []star{
			{image.Pt(22, 9), 5},
			{image.Pt(-27, 27), 5},
			{image.Pt(0, 24), 3},
		}
		assert.Equal(t, want, movingStars(4))
	})

	t.Run("when the small one is on its last frame round, should be it alone", func(t *testing.T) {
		t.Parallel()

		stars := movingStars(sparkleRest - sparkleDelay - 1)
		require.Len(t, stars, 1)
		assert.Equal(t, 2, stars[0].size)
	})

	t.Run("when they come to rest, should be none", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, movingStars(sparkleRest-sparkleDelay))
	})
}

// sparkleStart is when the stars of the tests below start.
var sparkleStart = time.Date(2004, 1, 1, 12, 0, 0, 0, time.UTC)

func Test_sparkle_update(t *testing.T) {
	t.Parallel()

	t.Run("when armed to play, should wait for the sprite to be free, show nothing for 60 frames, move, then rest", func(t *testing.T) {
		t.Parallel()

		var spk sparkle

		spk.arm(1, true, true)

		assert.False(t, spk.update(sparkleStart, true, false))
		assert.True(t, spk.start.IsZero(), "started while covered")

		assert.False(t, spk.update(sparkleStart, true, true), "the start changed what shows")
		assert.Equal(t, sparkleStart, spk.start)

		assert.False(t, spk.update(sparkleStart.Add(59*anim.Tick), true, true), "a star before frame 60")
		assert.True(t, spk.update(sparkleStart.Add(60*anim.Tick), true, true), "no star at frame 60")
		assert.Equal(t, 60, spk.step)

		assert.True(t, spk.update(sparkleStart.Add(time.Hour), true, true))
		assert.Equal(t, sparkleRest, spk.step)
		assert.False(t, spk.update(sparkleStart.Add(2*time.Hour), true, true), "moved at rest")
	})

	t.Run("when armed not to play, should be at rest at once", func(t *testing.T) {
		t.Parallel()

		var spk sparkle

		spk.arm(1, true, false)

		assert.Equal(t, sparkleRest, spk.step)
		assert.False(t, spk.update(sparkleStart, true, true))
	})

	t.Run("when the POKéMON isn't shiny, should never start", func(t *testing.T) {
		t.Parallel()

		var spk sparkle

		spk.arm(1, false, true)

		assert.False(t, spk.update(sparkleStart, true, true))
		assert.True(t, spk.start.IsZero())
	})

	t.Run("when the sprite doesn't show, should wait", func(t *testing.T) {
		t.Parallel()

		var spk sparkle

		spk.arm(1, true, true)

		assert.False(t, spk.update(sparkleStart, false, true))
		assert.True(t, spk.start.IsZero())
	})

	t.Run("when armed again for the same entry, should go on; for another, should start again", func(t *testing.T) {
		t.Parallel()

		var spk sparkle

		spk.arm(1, true, true)
		spk.update(sparkleStart, true, true)
		spk.update(sparkleStart.Add(70*anim.Tick), true, true)

		spk.arm(1, true, true)
		assert.Equal(t, sparkleStart, spk.start)
		assert.Equal(t, 70, spk.step)

		spk.arm(2, true, true)
		assert.True(t, spk.start.IsZero())
		assert.Equal(t, 0, spk.step)
	})
}

func Test_sparkle_paint(t *testing.T) {
	t.Parallel()

	centre := image.Pt(64, 64)
	paint := func(spk *sparkle) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 128, 128))
		spk.paint(newCanvas(img), centre)

		return img
	}

	t.Run("when at rest, should draw the eight stars around the centre", func(t *testing.T) {
		t.Parallel()

		spk := sparkle{
			shiny: true,
			step:  sparkleRest,
		}
		img := paint(&spk)

		for _, str := range restingStars {
			pnt := centre.Add(str.at)
			assert.Equal(t, sparkleCentre, img.RGBAAt(pnt.X, pnt.Y), "no centre at %v", str.at)
			assert.Equal(t, sparkleColor, img.RGBAAt(pnt.X+str.size, pnt.Y), "no arm at %v", str.at)
			assert.Equal(t, sparkleColor, img.RGBAAt(pnt.X, pnt.Y-str.size), "no arm at %v", str.at)
		}
	})

	t.Run("when moving, should draw the stars of that frame", func(t *testing.T) {
		t.Parallel()

		spk := sparkle{
			shiny: true,
			step:  sparkleDelay + 4,
		}
		img := paint(&spk)

		for _, str := range movingStars(4) {
			pnt := centre.Add(str.at)
			assert.Equal(t, sparkleCentre, img.RGBAAt(pnt.X, pnt.Y), "no centre at %v", str.at)
		}
	})

	t.Run("when brief, should draw the moving stars but none at rest", func(t *testing.T) {
		t.Parallel()

		blank := image.NewRGBA(image.Rect(0, 0, 128, 128))
		moving := sparkle{
			brief: true,
			shiny: true,
			step:  sparkleDelay,
		}
		resting := sparkle{
			brief: true,
			shiny: true,
			step:  sparkleRest,
		}

		assert.NotEqual(t, blank.Pix, paint(&moving).Pix, "no moving stars")
		assert.Equal(t, blank.Pix, paint(&resting).Pix, "stars at rest")
	})

	t.Run("when waiting or not shiny, should draw nothing", func(t *testing.T) {
		t.Parallel()

		blank := image.NewRGBA(image.Rect(0, 0, 128, 128))
		waiting := sparkle{
			shiny: true,
			step:  sparkleDelay - 1,
		}
		plain := sparkle{
			step: sparkleRest,
		}

		assert.Equal(t, blank.Pix, paint(&waiting).Pix)
		assert.Equal(t, blank.Pix, paint(&plain).Pix)
	})
}

// shinyWildApp starts a game, at the game window, whose wild Pikachu is
// shiny, behind its message box.
func shinyWildApp(t *testing.T, animated bool) *testApp {
	t.Helper()

	ta := newTestApp(t)
	ta.preferences.animatedSprites = animated
	ta.startGame()
	ta.call(func() {
		ta.game.StartWildBattle(25, 50, true)
	})
	ta.wait(1)

	return ta
}

// screenAt returns the screen's colour at pnt.
func screenAt(ta *testApp, pnt image.Point) any {
	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	ta.desk.paint(img)

	return img.RGBAAt(pnt.X, pnt.Y)
}

func Test_mainForm_sparkle(t *testing.T) {
	t.Parallel()

	t.Run("when a shiny wild POKéMON appears with the animations off, should show it shiny with its stars at rest", func(t *testing.T) {
		t.Parallel()

		ta := shinyWildApp(t, false)
		ta.answer("Select a POKéMON")

		spk := &ta.main.enemySparkle
		big := widgetOrigin(ta.main.enemyImage).Add(spriteCentre).Add(restingStars[0].at)

		assert.Equal(t, []listItem{{"", 24, true}}, ta.main.enemyImage.items)
		assert.True(t, spk.shiny)
		assert.Equal(t, sparkleRest, spk.step)
		assert.Equal(t, sparkleCentre, screenAt(ta, big), "no star on the screen")
		assert.Contains(t, ta.game.Info.Lines, "It's a shiny POKéMON!")
		ta.shot("arena-shiny")
	})

	t.Run("when a shiny wild POKéMON appears with the animations on, should start its stars once the box closes", func(t *testing.T) {
		t.Parallel()

		ta := shinyWildApp(t, true)
		spk := &ta.main.enemySparkle
		ta.wait(30)

		assert.True(t, spk.start.IsZero(), "started behind the message box")

		ta.answer("Select a POKéMON")
		require.False(t, spk.start.IsZero(), "not started once the box closed")

		ta.wait(62)
		assert.Equal(t, int(ta.now.Sub(spk.start)/anim.Tick), spk.step)
		assert.GreaterOrEqual(t, spk.step, sparkleDelay)
		ta.shot("arena-shiny-moving")

		ta.wait(600)
		assert.Equal(t, sparkleRest, spk.step)

		ta.shot("arena-shiny-rest")
		ta.wait(1)
		assert.False(t, ta.desk.dirty, "still repainting at rest")
	})

	t.Run("when the animations are turned off while the stars move, should put them at rest", func(t *testing.T) {
		t.Parallel()

		ta := shinyWildApp(t, true)
		ta.answer("Select a POKéMON")
		ta.wait(70)
		require.Less(t, ta.main.enemySparkle.step, sparkleRest)

		ta.setPreferences(preferences{
			choosePrompt: true,
		})
		ta.wait(1)

		assert.Equal(t, sparkleRest, ta.main.enemySparkle.step)
	})

	t.Run("when the player chooses a shiny fighter, should show it shiny with its stars", func(t *testing.T) {
		t.Parallel()

		ta := newTestApp(t)
		ta.startGame()

		ta.game.Owned[0].Shiny = true
		ta.call(func() {
			ta.game.StartWildBattle(25, 50, false)
		})
		ta.wait(1)
		ta.answer("Select a POKéMON")

		rect := ta.stats.party.itemRect(0)
		ta.doubleClick(widgetOrigin(ta.stats.party).Add(rect.Min).Add(image.Pt(10, 10)))
		ta.answer("# 004 Charmander (shiny), L.5")
		ta.answer("Choose Charmander?")

		assert.Equal(t, []listItem{{"", 3, true}}, ta.main.ownImage.items)
		assert.True(t, ta.main.ownSparkle.shiny)
		assert.Equal(t, sparkleRest, ta.main.ownSparkle.step)
		assert.False(t, ta.main.enemySparkle.shiny, "the wild Pikachu has stars")
		ta.shot("arena-shiny-fighter")
	})
}

func Test_pokedexForm_shiny(t *testing.T) {
	t.Parallel()

	// shinyDexApp opens the POKéDEX on Charmander, with its "Shiny" box,
	// which the test app hides, showing.
	shinyDexApp := func(t *testing.T) *testApp {
		t.Helper()

		ta := newDexApp(t)
		ta.pokedex.shiny.setVisible(true)
		pickCharmander(t, ta)

		return ta
	}

	// starless reports whether the POKéDEX's sparkle draws nothing.
	starless := func(dex *pokedexForm) bool {
		blank := image.NewRGBA(image.Rect(0, 0, 128, 128))
		img := image.NewRGBA(blank.Rect)
		dex.sparkle.paint(newCanvas(img), image.Pt(64, 64))

		return string(img.Pix) == string(blank.Pix)
	}

	t.Run("when Shiny is checked with the animations off, should show the species shiny without stars", func(t *testing.T) {
		t.Parallel()

		ta := shinyDexApp(t)
		dex := ta.pokedex
		ta.clickOn(dex.wdw, "ShinyBox")

		assert.True(t, ta.game.Pokedex.Shiny)
		assert.True(t, dex.shiny.checked)
		assert.Equal(t, []listItem{{"", 3, true}}, dex.image.items)
		assert.True(t, dex.sparkle.shiny)
		assert.True(t, starless(dex), "stars on the still sprite")
		ta.shot("dex-shiny")
	})

	t.Run("when Shiny is checked with the animations on, should play the stars, then show the still sprite without them", func(t *testing.T) {
		t.Parallel()

		ta := shinyDexApp(t)
		ta.preferences.animatedSprites = true

		dex := ta.pokedex
		spk := &dex.sparkle
		ta.clickOn(dex.wdw, "ShinyBox")
		ta.wait(1)

		require.False(t, spk.start.IsZero(), "the stars didn't start")

		ta.wait(70)
		assert.GreaterOrEqual(t, spk.step, sparkleDelay)
		assert.Less(t, spk.step, sparkleRest)
		assert.False(t, starless(dex), "no stars while they move")
		ta.shot("dex-shiny-moving")

		ta.wait(600)
		assert.Equal(t, sparkleRest, spk.step)
		assert.True(t, starless(dex), "stars on the still sprite")
		ta.shot("dex-shiny-rest")
	})

	t.Run("when another species is chosen, should show it shiny too", func(t *testing.T) {
		t.Parallel()

		ta := shinyDexApp(t)
		ta.clickOn(ta.pokedex.wdw, "ShinyBox")
		ta.pokedex.combo.pick(4) // Charmeleon
		ta.wait(1)

		assert.Equal(t, []listItem{{"", 4, true}}, ta.pokedex.image.items)
		assert.True(t, ta.pokedex.shiny.checked)
	})

	t.Run("when Shiny is cleared, should show the species as usual", func(t *testing.T) {
		t.Parallel()

		ta := shinyDexApp(t)
		dex := ta.pokedex
		ta.clickOn(dex.wdw, "ShinyBox")
		ta.clickOn(dex.wdw, "ShinyBox")

		assert.False(t, ta.game.Pokedex.Shiny)
		assert.Equal(t, []listItem{{"", 3, false}}, dex.image.items)
		assert.False(t, dex.sparkle.shiny)
	})

	t.Run("when the test app opens it, should hide the box", func(t *testing.T) {
		t.Parallel()

		ta := newDexApp(t)
		pickCharmander(t, ta)

		assert.False(t, ta.pokedex.shiny.shown())
	})
}
