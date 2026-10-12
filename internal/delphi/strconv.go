// SPDX-License-Identifier: AGPL-3.0-only

package delphi

// StrToInt parses s as Delphi's StrToInt does: leading spaces, an optional
// sign, then decimal digits, or hexadecimal ones after "$", "x" or "0x",
// within an Integer's range. ok is false where StrToInt raises
// EConvertError.
func StrToInt(s string) (n int32, ok bool) {
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	neg, base := false, uint64(10)
	if i < len(s) {
		switch s[i] {
		case '$', 'x', 'X':
			base = 16
			i++
		case '0':
			if i+1 < len(s) && s[i+1]|0x20 == 'x' {
				base = 16
				i += 2
			}
		case '-':
			neg = true
			i++
		case '+':
			i++
		}
	}
	if i >= len(s) {
		return 0, false
	}
	var v uint64
	for ; i < len(s); i++ {
		c := s[i]
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
	switch {
	case base == 16 && v <= 1<<32-1:
		return int32(uint32(v)), true // "$FFFFFFFF" is -1
	case neg && v <= 1<<31:
		return int32(-int64(v)), true
	case !neg && v <= 1<<31-1:
		return int32(v), true
	}
	return 0, false
}
