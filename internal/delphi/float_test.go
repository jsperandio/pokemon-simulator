// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import "testing"

func TestFloatToStr(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{0.5, "0.5"},
		{0.25, "0.25"},
		{4, "4"},
		{33.33, "33.33"},
		{77.99, "77.99"},
		{100, "100"},
		{-2.5, "-2.5"},
		{0.00001, "0.00001"},
		{0.0000015, "1.5E-6"},
		{1e20, "1E20"},
		{123456789012345, "123456789012345"},
		{1.0 / 3, "0.333333333333333"},
	}
	for _, tt := range tests {
		if got := FloatToStr(tt.in); got != tt.want {
			t.Errorf("FloatToStr(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Int(39 / 50 × 10000) is 7799 in Extended: the quotient rounds just below
// 0.78. The original's Statistics window shows 77.99 for 39 wins and 11
// losses; float64 would give 7800.
func TestExtended(t *testing.T) {
	q := NewExtended(39).Quo(NewExtended(50)).Mul(NewExtended(10000)).Int()
	if got := q.Float64(); got != 7799 {
		t.Errorf("Int(39/50 × 10000) = %v, want 7799", got)
	}

	if got := NewExtended(7799).Quo(NewExtended(100)).Float64(); FloatToStr(got) != "77.99" {
		t.Errorf("7799/100 prints %q", FloatToStr(got))
	}
}
