// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StrToInt(t *testing.T) {
	t.Parallel()

	t.Run("when the text is a decimal number, should return its value", func(t *testing.T) {
		t.Parallel()

		assertNumber(t, "0", 0)
		assertNumber(t, "42", 42)
		assertNumber(t, "+5", 5)
		assertNumber(t, "-5", -5)
	})

	t.Run("when the text starts with spaces, should skip them", func(t *testing.T) {
		t.Parallel()

		assertNumber(t, "  7", 7)
	})

	t.Run("when the value is an Integer's largest or smallest, should return it", func(t *testing.T) {
		t.Parallel()

		assertNumber(t, "2147483647", 2147483647)
		assertNumber(t, "-2147483648", -2147483648)
	})

	t.Run("when the value is past an Integer's range, should fail", func(t *testing.T) {
		t.Parallel()

		assertNotNumber(t, "2147483648")
		assertNotNumber(t, "-2147483649")
	})

	t.Run("when the text has a hexadecimal prefix, should read hexadecimal digits", func(t *testing.T) {
		t.Parallel()

		assertNumber(t, "$1F", 31)
		assertNumber(t, "0x10", 16)
	})

	t.Run("when a hexadecimal value sets the top bit, should read the 32 bits as signed", func(t *testing.T) {
		t.Parallel()

		assertNumber(t, "$FFFFFFFF", -1)
	})

	t.Run("when the text is not a whole number, should fail", func(t *testing.T) {
		t.Parallel()

		for _, txt := range []string{"", " ", "-", "+", "1-2", "5.", "7 ", "--5"} {
			assertNotNumber(t, txt)
		}
	})
}

// assertNumber asserts that StrToInt reads txt as want.
func assertNumber(t *testing.T, txt string, want int32) {
	t.Helper()

	val, ok := StrToInt(txt)

	assert.True(t, ok, "StrToInt(%q) failed", txt)
	assert.Equal(t, want, val, "StrToInt(%q)", txt)
}

// assertNotNumber asserts that StrToInt fails on txt, where Delphi's raises
// EConvertError.
func assertNotNumber(t *testing.T, txt string) {
	t.Helper()

	val, ok := StrToInt(txt)

	assert.False(t, ok, "StrToInt(%q) succeeded", txt)
	assert.Zero(t, val, "StrToInt(%q)", txt)
}
