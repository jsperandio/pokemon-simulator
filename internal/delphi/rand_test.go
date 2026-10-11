// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import "testing"

// The original's own Random calls after RandSeed was set to 1: a wild
// search at no badges (species, then the level rules), then an attack.
func TestRandom(t *testing.T) {
	calls := []struct{ n, want int }{
		{386, 12}, {30, 25}, {30, 6}, {20, 5}, {30, 20}, {20, 6}, {20, 3},
		{10, 3}, {10, 4}, {19, 1}, {9, 4},
	}
	r := Rand{Seed: 1}
	for i, c := range calls {
		if got := r.Random(c.n); got != c.want {
			t.Fatalf("call %d: Random(%d) = %d, want %d", i+1, c.n, got, c.want)
		}
	}
}

func TestRandomHook(t *testing.T) {
	var seen [][2]int
	r := Rand{Seed: 27, OnCall: func(n, v int) { seen = append(seen, [2]int{n, v}) }}
	for range 10000 { // a Game Corner draw: the last one counts
		r.Random(10)
	}
	if len(seen) != 10000 || seen[9999] != [2]int{10, 2} {
		t.Errorf("after 10000 draws from seed 27: %d calls, last %v; want the last to be 2", len(seen), seen[len(seen)-1])
	}
}
