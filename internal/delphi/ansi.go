// SPDX-License-Identifier: AGPL-3.0-only

// Package delphi reproduces the Delphi 7 runtime behaviour the original game
// relies on, so the port computes and prints exactly what it did.
package delphi

import "strings"

// DecodeANSI converts a Delphi AnsiString to UTF-8. The original ran on
// Western Windows, whose ANSI code page is Windows-1252 (cp1252).
func DecodeANSI(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b))
	for _, c := range b {
		if c >= 0x80 && c < 0xA0 {
			if r := cp1252High[c-0x80]; r != 0 {
				sb.WriteRune(r)
				continue
			}
		}
		sb.WriteRune(rune(c))
	}
	return sb.String()
}

// cp1252High maps bytes 0x80–0x9F, where cp1252 differs from Latin-1.
// Zero entries are undefined in cp1252 and fall back to Latin-1.
var cp1252High = [32]rune{
	'€', 0, '‚', 'ƒ', '„', '…', '†', '‡', 'ˆ', '‰', 'Š', '‹', 'Œ', 0, 'Ž', 0,
	0, '‘', '’', '“', '”', '•', '–', '—', '˜', '™', 'š', '›', 'œ', 0, 'ž', 'Ÿ',
}
