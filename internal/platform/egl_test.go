// SPDX-License-Identifier: AGPL-3.0-only

package platform

import (
	"os"
	"testing"
)

// TestEGLForX11 checks that a window system other than X11 named in
// EGL_PLATFORM is forgotten, and X11 kept.
func TestEGLForX11(t *testing.T) {
	for _, c := range []struct{ value, want string }{
		{"wayland", ""},
		{"surfaceless", ""},
		{"x11", "x11"},
	} {
		t.Setenv("EGL_PLATFORM", c.value)
		EGLForX11()
		if got := os.Getenv("EGL_PLATFORM"); got != c.want {
			t.Errorf("EGL_PLATFORM %q: became %q, want %q", c.value, got, c.want)
		}
	}
}
