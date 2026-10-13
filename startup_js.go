// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/ui/classic"
)

// watchWindow returns the game as it is: a page has no window to watch.
func watchWindow(app *classic.App) ebiten.Game {
	return app
}
