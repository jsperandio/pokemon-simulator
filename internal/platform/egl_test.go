// SPDX-License-Identifier: AGPL-3.0-only

package platform

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_EGLForX11 runs its cases one after another: t.Setenv can't be used
// in a parallel test.
func Test_EGLForX11(t *testing.T) {
	t.Run("when EGL_PLATFORM names another window system, should forget it", func(t *testing.T) {
		for _, val := range []string{"wayland", "surfaceless"} {
			t.Setenv("EGL_PLATFORM", val)
			EGLForX11()

			_, set := os.LookupEnv("EGL_PLATFORM")
			assert.False(t, set, "EGL_PLATFORM=%s is still set", val)
		}
	})

	t.Run("when EGL_PLATFORM is x11, should keep it", func(t *testing.T) {
		t.Setenv("EGL_PLATFORM", "x11")
		EGLForX11()

		assert.Equal(t, "x11", os.Getenv("EGL_PLATFORM"))
	})
}
