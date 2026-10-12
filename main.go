// SPDX-License-Identifier: AGPL-3.0-only

// Command pokemon-simulator is a port of POKéMON Simulator V4.5
// (BirdieSoft, 2002–2004) to Go and Ebitengine.
//
// A save file given on the command line is loaded at start-up, as the
// original loaded one opened from Explorer.
package main

import (
	"errors"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic"
)

func main() {
	raw, err := assets.FS.ReadFile("pokegame.dat")
	if err != nil {
		fail(err)
	}
	data, err := gamedata.Load(raw)
	if err != nil {
		fail(err)
	}
	app, err := classic.New(data, platform.Default(), platform.Args())
	if err != nil {
		fail(err)
	}
	ebiten.SetWindowSize(classic.WindowSize())
	ebiten.SetWindowTitle(game.Title)
	ebiten.SetWindowIcon([]image.Image{app.Icon()})
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ebiten.Termination) {
		fail(err)
	}
}

// fail tells the player why the game stopped, also where there is no
// terminal to read it in, as when it was started from an icon, and ends it.
func fail(err error) {
	platform.Alert(game.Title, "The game stopped because of an error:\n\n"+err.Error())
	log.Fatal(err)
}
