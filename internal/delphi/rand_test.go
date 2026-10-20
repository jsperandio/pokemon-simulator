// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Rand_Random(t *testing.T) {
	t.Parallel()

	// The original's own Random calls after RandSeed was set to 1: a wild
	// search at no badges (species, then the level rules), then an attack.
	t.Run("when the seed is 1, should draw what the original drew", func(t *testing.T) {
		t.Parallel()

		calls := []struct {
			n, want int
		}{
			{386, 12},
			{30, 25},
			{30, 6},
			{20, 5},
			{30, 20},
			{20, 6},
			{20, 3},
			{10, 3},
			{10, 4},
			{19, 1},
			{9, 4},
		}
		rnd := Rand{
			Seed: 1,
		}

		for i, cas := range calls {
			assert.Equal(t, cas.want, rnd.Random(cas.n), "call %d: Random(%d)", i+1, cas.n)
		}
	})

	t.Run("when OnCall is set, should see every call and its result", func(t *testing.T) {
		t.Parallel()

		var seen [][2]int

		rnd := Rand{
			Seed: 27,
			OnCall: func(n, v int) {
				seen = append(seen, [2]int{n, v})
			},
		}

		for range 10000 { // a Game Corner draw: the last one counts
			rnd.Random(10)
		}

		require.Len(t, seen, 10000)
		assert.Equal(t, [2]int{10, 2}, seen[9999])
	})
}
