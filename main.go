// SPDX-License-Identifier: AGPL-3.0-only

// Command pokemon-simulator is a port of POKéMON Simulator V4.5
// (BirdieSoft, 2002–2004) to Go and Ebitengine.
package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// The classic theme recreates the original's 800×600-era window
// arrangement on a logical desktop of this size.
const (
	screenWidth  = 640
	screenHeight = 512
)

// desktopTeal is the default Windows 98 desktop colour.
var desktopTeal = color.RGBA{R: 0x00, G: 0x80, B: 0x80, A: 0xff}

type game struct{}

func (g *game) Update() error { return nil }

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(desktopTeal)
	ebitenutil.DebugPrint(screen, "POKEMON Simulator port - work in progress")
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// windowScale returns the largest integer scale at which the logical
// desktop still fits comfortably on the current monitor.
func windowScale() int {
	m := ebiten.Monitor()
	if m == nil {
		return 1
	}
	w, h := m.Size()
	return max(min(w*9/10/screenWidth, h*9/10/screenHeight), 1)
}

func main() {
	s := windowScale()
	ebiten.SetWindowSize(screenWidth*s, screenHeight*s)
	ebiten.SetWindowTitle("POKéMON Simulator")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
