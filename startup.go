// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package main

import (
	"log"
	"os"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic"
)

// A GNOME desktop shows the game's window only once its helper for the
// title bars of X11 windows has given it one (platform.GNOMEFrames). When
// the helper has stopped, GNOME waits for the bar and Ebitengine for the
// window, both for good, with no window and a busy processor. So the game
// watches for its window, and when it isn't up in time there, starts again
// with a window without a title bar, which GNOME shows at once.

// noFrameEnv is set for the game started again without a title bar.
const noFrameEnv = "POKEMON_SIMULATOR_NO_FRAME"

// showTimeout is how long the window may take to show.
const showTimeout = 5 * time.Second

// noFrameText tells the player why the window has no title bar.
const noFrameText = "GNOME didn't show the game's window, as happens when its helper for the title bars " +
	"of X11 windows, mutter-x11-frames, isn't working. The game now runs without a title bar: move it " +
	"with the Super key and the mouse, and close it with Alt+F4 or File → Quit. If other programs have " +
	"lost their title bars too, logging out and in again brings them back."

// watchedApp is the game, telling when its window is up: a frame can come
// before, if the window manager holds the window back and Ebitengine stops
// waiting for it.
type watchedApp struct {
	*classic.App
	up    bool
	shown chan struct{}
}

func (w *watchedApp) Update() error {
	if !w.up && ebiten.IsWindowVisible() {
		w.up = true
		close(w.shown)
	}
	return w.App.Update()
}

// watchWindow sets the window up, and watches it come.
func watchWindow(app *classic.App) ebiten.Game {
	w := &watchedApp{App: app, shown: make(chan struct{})}
	restarted := os.Getenv(noFrameEnv) != ""
	if restarted {
		ebiten.SetWindowDecorated(false)
		go platform.Alert(game.Title, noFrameText)
	}
	go watch(w.shown, showTimeout, !restarted && platform.GNOMEFrames(), restart)
	return w
}

// watch waits for the window. If it isn't up in time where the desktop
// may be holding it back for its title bar (frames), it calls restart.
func watch(shown <-chan struct{}, timeout time.Duration, frames bool, restart func()) {
	select {
	case <-shown:
		return
	case <-time.After(timeout):
	}
	if frames {
		log.Printf("The window hasn't shown after %v: starting again without a title bar.", timeout)
		restart()
		return
	}
	log.Printf("The window hasn't shown after %v.", timeout)
}

// restart starts the game again in this process, without a title bar.
func restart() {
	exe, err := os.Executable()
	if err == nil {
		err = syscall.Exec(exe, os.Args, append(os.Environ(), noFrameEnv+"=1"))
	}
	log.Printf("Starting again failed: %v", err)
}
