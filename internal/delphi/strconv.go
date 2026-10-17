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

	val, ok := parseDigits(digits, base)
	if !ok {
		return 0, false
	}

	switch {
	case base == 16 && val < 1<<32:
		return int32(uint32(val)), true //nolint:gosec // the bits, signed: "$FFFFFFFF" is -1
	case neg && val <= 1<<31:
		return int32(-int64(val)), true
	case !neg && val < 1<<31:
		return int32(val), true
	}

	return 0, false
}

// splitNumber skips StrToInt's leading spaces, and the sign or the
// hexadecimal prefix after them.
func splitNumber(txt string) (neg bool, base uint64, digits string) {
	txt = strings.TrimLeft(txt, " ")
	if txt == "" {
		return false, 10, ""
	}

	switch txt[0] {
	case '$', 'x', 'X':
		return false, 16, txt[1:]
	case '0':
		if len(txt) > 1 && txt[1]|0x20 == 'x' {
			return false, 16, txt[2:]
		}
	case '-':
		return true, 10, txt[1:]
	case '+':
		return false, 10, txt[1:]
	}

	return false, 10, txt
}

// parseDigits reads digits in base 10 or 16, as long as the value stays
// within 2³².
func parseDigits(digits string, base uint64) (val uint64, ok bool) {
	for i := range len(digits) {
		c := digits[i]

		var dig uint64

		switch {
		case '0' <= c && c <= '9':
			dig = uint64(c - '0')
		case base == 16 && 'a' <= c|0x20 && c|0x20 <= 'f':
			dig = uint64(c|0x20-'a') + 10
		default:
			return 0, false
		}

		val = val*base + dig
		if val > 1<<32 {
			return 0, false
		}
	}

	return val, true
}
