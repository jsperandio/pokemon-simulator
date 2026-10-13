// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import "testing"

func TestStrToInt(t *testing.T) {
	for _, c := range []struct {
		s  string
		n  int32
		ok bool
	}{
		{"0", 0, true},
		{"42", 42, true},
		{"  7", 7, true},
		{"+5", 5, true},
		{"-5", -5, true},
		{"2147483647", 2147483647, true},
		{"-2147483648", -2147483648, true},
		{"2147483648", 0, false},
		{"-2147483649", 0, false},
		{"$1F", 31, true},
		{"0x10", 16, true},
		{"$FFFFFFFF", -1, true},
		{"", 0, false},
		{" ", 0, false},
		{"-", 0, false},
		{"+", 0, false},
		{"1-2", 0, false},
		{"5.", 0, false},
		{"7 ", 0, false},
		{"--5", 0, false},
	} {
		if n, ok := StrToInt(c.s); n != c.n || ok != c.ok {
			t.Errorf("StrToInt(%q) = %d, %v; want %d, %v", c.s, n, ok, c.n, c.ok)
		}
	}
}
