// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The original's Statistics window shows 77.99 for 39 wins and 11 losses;
// float64 would give 78.
func Test_Extended_Quo(t *testing.T) {
	t.Parallel()

	t.Run("when 39/50 is scaled by 10000 and truncated, should give 7799 as the x87 did", func(t *testing.T) {
		t.Parallel()

		hundredths := NewExtended(39).Quo(NewExtended(50)).Mul(NewExtended(10000)).Int()

		assert.Equal(t, 7799.0, hundredths.Float64()) //nolint:testifylint // exact: in Extended the quotient rounds just below 0.78
	})

	t.Run("when 7799 is divided by 100, should give the Double 77.99", func(t *testing.T) {
		t.Parallel()

		percent := NewExtended(7799).Quo(NewExtended(100))

		assert.Equal(t, 77.99, percent.Float64()) //nolint:testifylint // exact: the Double the window prints
	})
}
