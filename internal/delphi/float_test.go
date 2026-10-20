// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import "testing"

// Int(39 / 50 × 10000) is 7799 in Extended: the quotient rounds just below
// 0.78. The original's Statistics window shows 77.99 for 39 wins and 11
// losses; float64 would give 7800.
func TestExtended(t *testing.T) {
	q := NewExtended(39).Quo(NewExtended(50)).Mul(NewExtended(10000)).Int()
	if got := q.Float64(); got != 7799 {
		t.Errorf("Int(39/50 × 10000) = %v, want 7799", got)
	}

	if got := NewExtended(7799).Quo(NewExtended(100)).Float64(); got != 77.99 {
		t.Errorf("7799/100 = %v, want 77.99", got)
	}
}
