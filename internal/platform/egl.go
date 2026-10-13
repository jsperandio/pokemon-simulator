// SPDX-License-Identifier: AGPL-3.0-only

package platform

import "os"

// EGLForX11 forgets a window system other than X11 named in EGL_PLATFORM,
// before the game's window opens. On Linux and the BSDs, Ebitengine draws
// on an X11 window through EGL, and gets EGL's display for its X11
// connection with eglGetDisplay, which takes EGL_PLATFORM's word for what
// the connection is. VS Code's terminal, on a Wayland desktop, passes on
// the "wayland" of its Electron: EGL then reads the X11 connection as a
// Wayland one, and the game crashes before it can say why. Without the
// variable, EGL tells the connection's kind itself. Elsewhere, nothing
// reads it.
func EGLForX11() {
	if v, ok := os.LookupEnv("EGL_PLATFORM"); ok && v != "x11" {
		os.Unsetenv("EGL_PLATFORM") //nolint:errcheck // it fails only for a name with "=" in it
	}
}
