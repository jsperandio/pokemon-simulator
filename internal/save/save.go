// SPDX-License-Identifier: AGPL-3.0-only

// Package save reads and writes the original's .pgs save files: a text
// file with one value per line (CRLF, cp1252), 30,405 lines in version 2.
// Version 1 files (no header, no Jirachi/Deoxys flags, 251 POKéDEX
// entries) still load; saving always writes version 2, as the original did.
package save

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
)

const (
	// Header starts a version 2 file.
	Header = "VERSION2SAVEFILEIDENTIFIER"
	// Slots is the size of the party arrays, all of which are saved.
	Slots = 10000
	// NumSpecies is the number of POKéDEX entries in version 2.
	NumSpecies = 386
	// v1Species is the number of POKéDEX entries in version 1.
	v1Species = 251
)

// File is a save. The counters are Cardinals, as in the game.
type File struct {
	PlayerName  string
	BeatLance   bool
	BeatAsh     bool
	BeatLink    bool // the author's own trainer
	GotJirachi  bool
	GotDeoxys   bool
	GymProgress uint8 // badges won

	Money, Wins, Losses, Fights, Catches                                  uint32
	PokeBalls, GreatBalls, UltraBalls, MasterBalls, Potions, SuperPotions uint32

	InDex [NumSpecies + 1]bool // InDex[s]: species s registered; [0] unused
	Owned [Slots]Slot          // the party arrays, packed from slot 0; species 0 ends the party
}

// Slot is one entry of the party arrays.
type Slot struct {
	Species int32
	Level   uint8
	XP      uint32
}

// Party returns the slots up to the first empty one, as the game reads them.
func (f *File) Party() []Slot {
	for i, s := range f.Owned {
		if s.Species <= 0 {
			return f.Owned[:i]
		}
	}

	return f.Owned[:]
}

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
		f.Owned[i] = Slot{Species: int32(r.number()), Level: uint8(r.number()), XP: uint32(r.number())} //nolint:gosec // the game keeps the low bits
	}

	if r.err != nil {
		return nil, r.err
	}

	return f, nil
}

// Bytes writes the save in version 2, byte for byte as the original does.
func (f *File) Bytes() []byte {
	var b bytes.Buffer
	b.Grow(3 * Slots * 4)

	line := func(s string) { b.WriteString(s); b.WriteString("\r\n") }
	boolean := func(v bool) {
		if v {
			line("TRUE")
		} else {
			line("FALSE")
		}
	}

	line(Header)
	b.Write(delphi.EncodeANSI(f.PlayerName))
	b.WriteString("\r\n")

	for _, v := range []bool{f.BeatLance, f.BeatAsh, f.BeatLink, f.GotJirachi, f.GotDeoxys} {
		boolean(v)
	}

	line(strconv.Itoa(int(f.GymProgress)))

	for _, c := range f.counters() {
		line(strconv.FormatUint(uint64(*c), 10))
	}

	for s := 1; s <= NumSpecies; s++ {
		boolean(f.InDex[s])
	}

	for _, s := range f.Owned {
		line(strconv.Itoa(int(s.Species)))
		line(strconv.Itoa(int(s.Level)))
		line(strconv.FormatUint(uint64(s.XP), 10))
	}

	return b.Bytes()
}

// counters lists the Cardinal fields in file order.
func (f *File) counters() []*uint32 {
	return []*uint32{
		&f.Money, &f.Wins, &f.Losses, &f.Fights, &f.Catches,
		&f.PokeBalls, &f.GreatBalls, &f.UltraBalls, &f.MasterBalls, &f.Potions, &f.SuperPotions,
	}
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
func (r *reader) boolean() bool { return r.text() == "TRUE" }

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
