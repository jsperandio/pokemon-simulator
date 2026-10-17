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
	rdr := &reader{lines: strings.Split(text, "\n")}
	sav := &File{}
	sav.PlayerName = rdr.text()
	v2 := sav.PlayerName == Header

	if v2 {
		sav.PlayerName = rdr.text()
	}

	sav.BeatLance, sav.BeatAsh, sav.BeatLink = rdr.boolean(), rdr.boolean(), rdr.boolean()
	if v2 {
		sav.GotJirachi, sav.GotDeoxys = rdr.boolean(), rdr.boolean()
	}

	sav.GymProgress = uint8(rdr.number()) //nolint:gosec // the game keeps the low bits
	for _, c := range sav.counters() {
		*c = uint32(rdr.number()) //nolint:gosec // the game keeps the low bits
	}

	dex := v1Species
	if v2 {
		dex = NumSpecies
	}

	for s := 1; s <= dex; s++ {
		sav.InDex[s] = rdr.boolean()
	}

	for i := range sav.Owned {
		sav.Owned[i] = Slot{
			Species: int32(rdr.number()),  //nolint:gosec // the game keeps the low bits
			Level:   uint8(rdr.number()),  //nolint:gosec // the game keeps the low bits
			XP:      uint32(rdr.number()), //nolint:gosec // the game keeps the low bits
		}
	}

	if rdr.err != nil {
		return nil, rdr.err
	}

	return sav, nil
}

// reader hands out the lines one value at a time and keeps the first error.
type reader struct {
	lines []string
	idx   int
	err   error
}

func (r *reader) text() string {
	if r.err != nil {
		return ""
	}

	if r.idx >= len(r.lines) {
		r.err = fmt.Errorf("save: the file ends at line %d", r.idx+1)

		return ""
	}

	r.idx++

	return delphi.DecodeANSI([]byte(r.lines[r.idx-1]))
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
		r.err = fmt.Errorf("save: line %d: want a whole number, got %q", r.idx, s)
	}

	return v
}
