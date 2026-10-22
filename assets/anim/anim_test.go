// SPDX-License-Identifier: AGPL-3.0-only

//go:build !noanim

package anim

import (
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// species is the number of species the game has.
const species int = 386

func Test_Load(t *testing.T) {
	t.Parallel()

	sprites := make([]*Sprite, species+1)
	errs := make([]error, species+1)

	for spc := 1; spc <= species; spc++ {
		sprites[spc], errs[spc] = Load(spc)
	}

	t.Run("when every species is loaded, should find its animation", func(t *testing.T) {
		t.Parallel()

		for spc := 1; spc <= species; spc++ {
			if !assert.NoError(t, errs[spc], "species %d", spc) {
				continue
			}

			assert.NotEmpty(t, sprites[spc].Steps, "species %d has no steps", spc)
		}
	})

	t.Run("when a step is read, should show a frame of the sheet for some ticks", func(t *testing.T) {
		t.Parallel()

		for spc := 1; spc <= species; spc++ {
			spr := sprites[spc]
			require.NotNil(t, spr, "species %d", spc)

			for i, stp := range spr.Steps {
				assert.Positive(t, stp.Ticks, "species %d, step %d", spc, i)
				assert.GreaterOrEqual(t, stp.Frame, 0, "species %d, step %d", spc, i)
				assert.True(t, spr.Frame(stp.Frame).In(spr.Sheet.Rect), "species %d, step %d: frame %d is off the sheet", spc, i, stp.Frame)
			}
		}
	})

	t.Run("when a sheet is decoded, should hold whole frames only", func(t *testing.T) {
		t.Parallel()

		for spc := 1; spc <= species; spc++ {
			spr := sprites[spc]
			require.NotNil(t, spr, "species %d", spc)

			assert.Equal(t, spr.Size.Y, spr.Sheet.Rect.Dy(), "species %d", spc)
			assert.Zero(t, spr.Sheet.Rect.Dx()%spr.Size.X, "species %d", spc)
		}
	})

	t.Run("when the species is out of range, should fail", func(t *testing.T) {
		t.Parallel()

		_, err := Load(0)
		assert.Error(t, err)

		_, err = Load(species + 1)
		assert.Error(t, err)
	})
}

func Test_Sprite_Frame(t *testing.T) {
	t.Parallel()

	t.Run("when frames are asked for, should lay them side by side", func(t *testing.T) {
		t.Parallel()

		spr := &Sprite{
			Size: image.Pt(30, 40),
		}

		assert.Equal(t, image.Rect(0, 0, 30, 40), spr.Frame(0))
		assert.Equal(t, image.Rect(60, 0, 90, 40), spr.Frame(2))
	})
}

func Test_Sprite_Duration(t *testing.T) {
	t.Parallel()

	t.Run("when the steps are summed, should last their ticks", func(t *testing.T) {
		t.Parallel()

		spr := &Sprite{
			Steps: []Step{
				{
					Frame: 0,
					Ticks: 3,
				},
				{
					Frame: 1,
					Ticks: 2,
				},
			},
		}

		assert.Equal(t, 5*Tick, spr.Duration())
	})
}

func Test_Sprite_At(t *testing.T) {
	t.Parallel()

	spr := &Sprite{
		Steps: []Step{
			{
				Frame: 2,
				Ticks: 3,
			},
			{
				Frame: 0,
				Ticks: 1,
			},
		},
	}

	t.Run("when it starts, should show the first step's frame", func(t *testing.T) {
		t.Parallel()

		frm, ok := spr.At(0)
		assert.True(t, ok)
		assert.Equal(t, 2, frm)
	})

	t.Run("when a step's ticks have passed, should show the next one's frame", func(t *testing.T) {
		t.Parallel()

		frm, ok := spr.At(3*Tick - time.Nanosecond)
		assert.True(t, ok)
		assert.Equal(t, 2, frm)

		frm, ok = spr.At(3 * Tick)
		assert.True(t, ok)
		assert.Equal(t, 0, frm)
	})

	t.Run("when every step has passed, should be over", func(t *testing.T) {
		t.Parallel()

		_, ok := spr.At(4 * Tick)
		assert.False(t, ok)
	})
}

func Test_Tick(t *testing.T) {
	t.Parallel()

	t.Run("when converted, should be a Game Boy Advance frame", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 16742706*time.Nanosecond, Tick)
	})
}
