// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import "strings"

// StrToInt parses s as Delphi's StrToInt does: leading spaces, an optional
// sign, then decimal digits, or hexadecimal ones after "$", "x" or "0x",
// within an Integer's range. ok is false where StrToInt raises
// EConvertError.
func StrToInt(s string) (n int32, ok bool) {
	neg, base, digits := splitNumber(s)
	if digits == "" {
		return 0, false
	}

	v, ok := parseDigits(digits, base)
	if !ok {
		return 0, false
	}

	switch {
	case base == 16 && v < 1<<32:
		return int32(uint32(v)), true //nolint:gosec // the bits, signed: "$FFFFFFFF" is -1
	case neg && v <= 1<<31:
		return int32(-int64(v)), true
	case !neg && v < 1<<31:
		return int32(v), true
	}

	return 0, false
}

// splitNumber skips StrToInt's leading spaces, and the sign or the
// hexadecimal prefix after them.
func splitNumber(s string) (neg bool, base uint64, digits string) {
	s = strings.TrimLeft(s, " ")
	if s == "" {
		return false, 10, ""
	}

	switch s[0] {
	case '$', 'x', 'X':
		return false, 16, s[1:]
	case '0':
		if len(s) > 1 && s[1]|0x20 == 'x' {
			return false, 16, s[2:]
		}
	case '-':
		return true, 10, s[1:]
	case '+':
		return false, 10, s[1:]
	}

	return false, 10, s
}

// parseDigits reads digits in base 10 or 16, as long as the value stays
// within 2³².
func parseDigits(digits string, base uint64) (v uint64, ok bool) {
	for i := range len(digits) {
		c := digits[i]

		var d uint64

		switch {
		case '0' <= c && c <= '9':
			d = uint64(c - '0')
		case base == 16 && 'a' <= c|0x20 && c|0x20 <= 'f':
			d = uint64(c|0x20-'a') + 10
		default:
			return 0, false
		}

		v = v*base + d
		if v > 1<<32 {
			return 0, false
		}
	}

	return v, true
}
