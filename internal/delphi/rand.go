// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import "time"

// Rand is Delphi's random number generator: RandSeed and Random(N). The
// original game seeds it once at start-up (Randomize) and draws everything
// from it, so replaying a seed replays a game.
type Rand struct {
	Seed uint32 // RandSeed

	// OnCall, if set, sees every Random(n) call and its result.
	OnCall func(n, v int)
}

// Random returns a number in [0, n), as Random(N) does: the seed advances
// as RandSeed * $08088405 + 1 and the result is the high half of
// seed × N, both unsigned.
func (r *Rand) Random(n int) int {
	r.Seed = r.Seed*0x08088405 + 1
	v := int(uint64(r.Seed) * uint64(uint32(n)) >> 32) //nolint:gosec // N's 32 bits, read unsigned

	if r.OnCall != nil {
		r.OnCall(n, v)
	}

	return v
}

// Randomize seeds the generator from the clock, as Randomize does (the
// original takes the performance counter's low 32 bits).
func (r *Rand) Randomize() {
	r.Seed = uint32(time.Now().UnixNano()) //nolint:gosec // the clock's low 32 bits
}
