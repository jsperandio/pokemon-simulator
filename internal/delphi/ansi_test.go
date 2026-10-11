// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import "testing"

func TestDecodeANSI(t *testing.T) {
	tests := []struct {
		in   []byte
		want string
	}{
		{[]byte("POK\xe9MON"), "POKéMON"},
		{[]byte("2500\xa5"), "2500¥"},
		{[]byte("\x80\x99"), "€™"},
		{[]byte("\x81"), "\u0081"},
		{nil, ""},
	}
	for _, tt := range tests {
		if got := DecodeANSI(tt.in); got != tt.want {
			t.Errorf("DecodeANSI(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEncodeANSI(t *testing.T) {
	tests := []struct {
		in   string
		want []byte
	}{
		{"POKéMON", []byte("POK\xe9MON")},
		{"2500¥", []byte("2500\xa5")},
		{"€™", []byte("\x80\x99")},
		{"\u0081", []byte("\x81")},
		{"日本", []byte("??")},
		{"", []byte{}},
	}
	for _, tt := range tests {
		if got := EncodeANSI(tt.in); string(got) != string(tt.want) {
			t.Errorf("EncodeANSI(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
