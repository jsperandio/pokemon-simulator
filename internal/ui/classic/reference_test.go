// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/game/scenario"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// refShots are the screenshots the original's were taken at, after an
// action of a scenario: the windows then, and the dialogs the next
// actions show, in order.
type refShots struct {
	windows []string // "Form4 idle"
	dialogs []string
}

// referenceShots are the points of the scenarios where the original's
// reference screenshots were taken, by the action before them (its second
// time in the scenario with " #2" …).
var referenceShots = map[string]map[string]refShots{
	"screens-dex": {
		"> menu Actions > View POKéDEX":    {windows: []string{"Form8 empty"}},
		"> combo Form8.ComboBox1 24":       {windows: []string{"Form8 selected"}},
		"> menu Actions > View POKéDEX #2": {windows: []string{"Form8 reopened"}},
	},
	"screens-start": {
		"> start":                   {windows: []string{"Form1 start"}, dialogs: []string{"about"}},
		"> click Form1.AboutButton": {dialogs: []string{"name"}},
		"> new Gary":                {windows: []string{"Form4 starters"}},
		"> starter 1":               {windows: []string{"Form8 pokedex-new", "Form5 stats-new"}},
	},
	"screens-game": {
		"> seed 30": {windows: []string{"Form4 idle", "Form5 stats"}, dialogs: []string{"dialog-information"}},
		"> menu Actions > Fight a Trainer > Trainers > Beginner > Red": {windows: []string{"Form4 battle-start"}},
		"> answer yes":                 {dialogs: []string{"dialog-pokemon", "dialog-confirmation"}},
		"> choose 1":                   {windows: []string{"Form4 battle", "Form5 battle-items", "Form7 battle-info"}},
		"> move 1":                     {windows: []string{"Form4 battle-turn", "Form7 battle-info-turn"}},
		"> fightbest":                  {windows: []string{"Form4 battle-over"}},
		"> type Form9.SpinEdit1 1":     {windows: []string{"Form9 mart"}},
		"> menu Actions > Game Corner": {windows: []string{"FormB gamecorner"}, dialogs: []string{"dialog-warning"}},
		"> combo FormA.ComboBox2 3":    {windows: []string{"FormA typecomparisons"}},
		"> combo Form8.ComboBox1 132":  {windows: []string{"Form8 pokedex"}},
		"> click Form8.Button1":        {dialogs: []string{"dialog-mewtwo"}},
		"> menu Actions > Fight a wild POKéMON > Search for a POKéMON": {windows: []string{"Form4 picker"}},
		"> choose 3": {windows: []string{"Form4 evolve-choice"}},
	},
	"stats-percent": {
		"> start":        {windows: []string{"Form5 percent-39-11"}},
		"> set Losses 2": {windows: []string{"Form5 percent-1-2"}},
	},
}

// originalClients are the original's client areas in its screenshots, by
// window, or by screenshot where the window had another.
var originalClients = map[string]image.Point{
	"Form1":             {536, 366},
	"Form3":             {511, 157},
	"Form4":             {422, 270},
	"Form5":             {177, 473},
	"Form7":             {422, 159},
	"Form8":             {241, 161},
	"Form9":             {471, 65},
	"FormA":             {377, 41},
	"FormB":             {385, 153},
	"AboutBox":          {241, 98},
	"OKRightDlg":        {153, 25},
	"screens-dex-empty": {241, 25}, // the POKéDEX's AutoSize, without its panel
}

// Test_ReferenceScreens plays the scenarios the original's reference
// screenshots were taken in, through the classic interface, and takes the
// port's at the same points, of the same areas: Wine's X11 window of each
// window, which is its client area and menu bar (a tool window's whole),
// and the menus open. The windows must have the original's client areas.
// With $CLASSIC_SHOTS set, the screenshots are saved there with the
// original's names (screens-game-idle.png …), to compare with its.
func Test_ReferenceScreens(t *testing.T) {
	t.Parallel()

	data := testData(t)
	dir := filepath.Join("..", "..", "game", "testdata", "scenarios")

	for name, shots := range referenceShots {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			shootScenario(t, data, dir, name, shots)
		})
	}

	// The loading screen, which the original shows for about a second: its
	// screenshot is a frame of smoke/loading.sh's, at the badge names.
	t.Run("screens-start-loading", func(t *testing.T) {
		t.Parallel()

		lf := newTestApp(t).load
		lf.startStep(2)

		gge := get[*gauge](lf.wdw, "Gauge2")
		gge.set(gge.max, 700, gge.fore) // 23%
		shootWindow(t, "screens-start-loading", lf.wdw)
	})

	t.Run("screens-menus", func(t *testing.T) {
		t.Parallel()

		shootMenus(t)
	})
}

// shootScenario plays scenario name, in dir, through the classic
// interface, and takes the port's screenshots after its actions in shots.
func shootScenario(t *testing.T, data *gamedata.Data, dir, name string, shots map[string]refShots) {
	src, err := os.ReadFile(filepath.Join(dir, name+".txt")) //nolint:gosec // a scenario in testdata
	require.NoError(t, err)

	var (
		ud      *uiDriver
		dialogs []string
	)
	ply := &scenario.Player{
		Name: name,
		Dir:  dir,
		Data: data,
	}
	ply.NewDriver = func() scenario.Driver {
		ud = &uiDriver{
			t: t,
		}
		ud.modal = func(wdw *window) {
			if len(dialogs) > 0 {
				shootWindow(t, name+"-"+dialogs[0], wdw)

				dialogs = dialogs[1:]
			}
		}

		return ud
	}
	taken, times := map[string]bool{}, map[string]int{}
	ply.After = func(action string) {
		times[action]++
		if times[action] > 1 {
			action += " #" + strconv.Itoa(times[action])
		}

		shs, ok := shots[action]
		if !ok {
			return
		}

		taken[action] = true

		ud.settle()

		for _, spec := range shs.windows {
			form, label, _ := strings.Cut(spec, " ")
			wdw := ud.form(form)

			if !assert.True(t, wdw.visible, "%s: %s isn't shown", label, form) {
				continue
			}

			shootWindow(t, name+"-"+label, wdw)
		}

		assert.Empty(t, dialogs, "after %s, no dialog showed", action)

		dialogs = shs.dialogs
	}

	_, err = ply.Play(string(src))
	require.NoError(t, err, "%s", strings.Join(ply.Trace(), "\n"))

	for action := range shots {
		assert.True(t, taken[action], "no %q in the scenario", action)
	}
}

// shootMenus takes the menus' screenshots, opened with the keyboard as the
// probe did (Alt and the accelerator, then the arrows), with the first item
// selected.
func shootMenus(t *testing.T) {
	sav := save.File{
		PlayerName: "Probe",
		PokeBalls:  3,
	}
	sav.Owned[0] = save.Slot{
		Species: 25,
		Level:   5,
		XP:      125,
	}
	sav.InPokedex[25] = true
	dir := t.TempDir()
	file := filepath.Join(dir, "probe-screens-menus.pgs")

	err := os.WriteFile(file, sav.Bytes(), 0o600)
	require.NoError(t, err)

	ta := newTestAppIn(t, dir, []string{file})
	ta.wait(80)

	arrows := map[string]ebiten.Key{
		"left":  ebiten.KeyLeft,
		"right": ebiten.KeyRight,
		"up":    ebiten.KeyUp,
		"down":  ebiten.KeyDown,
	}

	for _, mnu := range []string{
		"f menu-file",
		"a menu-actions",
		"a right menu-fight",
		"a right right menu-trainers",
		"a right right right menu-beginner",
		"a right down down down right menu-johto",
		"a down menu-wild",
		"m menu-music",
		"h menu-help",
	} {
		keys := strings.Fields(mnu)
		label := keys[len(keys)-1]
		letter := ebiten.KeyA + ebiten.Key(keys[0][0]-'a')
		ta.frame(input{
			keys: []ebiten.Key{ebiten.KeyAltLeft},
			mods: mods{
				alt: true,
			},
		})
		ta.frame(input{
			keys:  []ebiten.Key{letter},
			chars: []rune{rune(keys[0][0])},
			mods: mods{
				alt: true,
			},
		})
		ta.frame(input{
			keysUp: []ebiten.Key{letter, ebiten.KeyAltLeft},
		})

		for _, key := range keys[1 : len(keys)-1] {
			ta.key(arrows[key])
		}

		bar := ta.main.wdw.menu
		require.True(t, bar.active(), "%s: no menu open", label)
		require.NotEmpty(t, bar.open, "%s: no menu open", label)

		shootWindow(t, "screens-menus-"+label, ta.main.wdw)

		for range len(bar.open) + 2 {
			ta.key(ebiten.KeyEscape)
		}

		require.False(t, bar.active(), "%s: the menu stays open", label)
	}
}

// shotArea is the part of wdw on the desktop that Wine's X11 window for it
// holds.
func shotArea(wdw *window) image.Rectangle {
	if wdw.style == bsToolWindow {
		return wdw.rect()
	}

	rect := wdw.clientRect()
	if wdw.menu != nil {
		rect.Min.Y = wdw.menuRect().Min.Y
	}

	return rect
}

// shootWindow paints wdw alone and checks its client area against the
// original's. With $CLASSIC_SHOTS set, it saves wdw's area there as
// name.png, and its open menus as name-1.png, name-2.png … from the left.
func shootWindow(t *testing.T, name string, wdw *window) {
	t.Helper()

	want, ok := originalClients[name]
	if !ok {
		want, ok = originalClients[wdw.name]
	}

	if ok {
		assert.Equal(t, want, wdw.client, "%s: %s's client area differs from the original's", name, wdw.name)
	}

	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	cv := newCanvas(img)
	cv.fill(img.Rect, colDesktop)
	wdw.paint(cv)
	savePNG(t, name, crop(img, shotArea(wdw)))

	if mb := wdw.menu; mb != nil && mb.active() {
		mb.paintPopups(cv)

		for i, mp := range mb.open {
			savePNG(t, name+"-"+string(rune('1'+i)), crop(img, mp.rect))
		}
	}
}

// crop copies a part of img.
func crop(img *image.RGBA, rect image.Rectangle) *image.RGBA {
	out := image.NewRGBA(image.Rectangle{
		Max: rect.Size(),
	})
	draw.Draw(out, out.Rect, img, rect.Min, draw.Src)

	return out
}

// savePNG saves img as name.png in $CLASSIC_SHOTS, if set.
func savePNG(t *testing.T, name string, img image.Image) {
	t.Helper()

	dir := os.Getenv("CLASSIC_SHOTS")
	if dir == "" {
		return
	}

	var buf bytes.Buffer

	err := png.Encode(&buf, img)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, name+".png"), buf.Bytes(), 0o600) //nolint:gosec // the folder $CLASSIC_SHOTS names
	require.NoError(t, err)
}
