// SPDX-License-Identifier: AGPL-3.0-only

package save

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
)

// Read parses a save of either version. Unlike the original, which stops
// with an I/O error half-way through a damaged file, it reads nothing then.
func Read(b []byte) (*File, error) {
	text := strings.TrimSuffix(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	r := &reader{lines: strings.Split(text, "\n")}
	f := &File{}
	f.PlayerName = r.text()
	v2 := f.PlayerName == Header

	if v2 {
		f.PlayerName = r.text()
	}

	f.BeatLance, f.BeatAsh, f.BeatLink = r.boolean(), r.boolean(), r.boolean()
	if v2 {
		f.GotJirachi, f.GotDeoxys = r.boolean(), r.boolean()
	}

	f.GymProgress = uint8(r.number()) //nolint:gosec // the game keeps the low bits
	for _, c := range f.counters() {
		*c = uint32(r.number()) //nolint:gosec // the game keeps the low bits
	}

	dex := v1Species
	if v2 {
		dex = NumSpecies
	}

	for s := 1; s <= dex; s++ {
		f.InDex[s] = r.boolean()
	}

	for i := range f.Owned {
		f.Owned[i] = Slot{
			Species: int32(r.number()),  //nolint:gosec // the game keeps the low bits
			Level:   uint8(r.number()),  //nolint:gosec // the game keeps the low bits
			XP:      uint32(r.number()), //nolint:gosec // the game keeps the low bits
		}
	}

	if r.err != nil {
		return nil, r.err
	}

	return f, nil
}

// reader hands out the lines one value at a time and keeps the first error.
type reader struct {
	lines []string
	n     int
	err   error
}

func (r *reader) text() string {
	if r.err != nil {
		return ""
	}

	if r.n >= len(r.lines) {
		r.err = fmt.Errorf("save: the file ends at line %d", r.n+1)

		return ""
	}

	r.n++

	return delphi.DecodeANSI([]byte(r.lines[r.n-1]))
}

// boolean is true for "TRUE" only, as the game's comparison is.
func (r *reader) boolean() bool {
	return r.text() == "TRUE"
}

// number reads an integer; the game reads Int64s and keeps the low bits.
func (r *reader) number() int64 {
	s := r.text()
	if r.err != nil {
		return 0
	}

	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		r.err = fmt.Errorf("save: line %d: want a whole number, got %q", r.n, s)
	}

	return v
}
