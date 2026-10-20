// SPDX-License-Identifier: AGPL-3.0-only

// Package save reads and writes the original's .pgs save files: a text
// file with one value per line (CRLF, cp1252), 30,405 lines in version 2.
// Version 1 files (no header, no Jirachi/Deoxys flags, 251 POKéDEX
// entries) still load; saving always writes version 2, as the original did.
package save

import (
	"bytes"
	"slices"
	"strconv"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
)

const (
	// Header starts a version 2 file.
	Header string = "VERSION2SAVEFILEIDENTIFIER"
	// Slots is the size of the party arrays, all of which are saved.
	Slots int = 10000
	// NumSpecies is the number of POKéDEX entries in version 2.
	NumSpecies int = 386
	// v1Species is the number of POKéDEX entries in version 1.
	v1Species int = 251
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

	InPokedex [NumSpecies + 1]bool // InPokedex[s]: species s registered; [0] unused
	Owned     [Slots]Slot          // the party arrays, packed from slot 0; species 0 ends the party
}

// Slot is one entry of the party arrays.
type Slot struct {
	Species int32
	Level   uint8
	XP      uint32
}

// empty reports whether the slot ends the party.
func (s Slot) empty() bool {
	return s.Species <= 0
}

// Party returns the slots up to the first empty one, as the game reads them.
func (f *File) Party() []Slot {
	if n := slices.IndexFunc(f.Owned[:], Slot.empty); n >= 0 {
		return f.Owned[:n]
	}

	return f.Owned[:]
}

// Bytes writes the save in version 2, byte for byte as the original does.
func (f *File) Bytes() []byte {
	var buf bytes.Buffer
	buf.Grow(3 * Slots * 4)

	line := func(s string) {
		buf.WriteString(s)
		buf.WriteString("\r\n")
	}
	boolean := func(v bool) {
		line(boolText(v))
	}

	line(Header)
	buf.Write(delphi.EncodeANSI(f.PlayerName))
	buf.WriteString("\r\n")

	flags := []bool{
		f.BeatLance,
		f.BeatAsh,
		f.BeatLink,
		f.GotJirachi,
		f.GotDeoxys,
	}

	for _, v := range flags {
		boolean(v)
	}

	line(strconv.Itoa(int(f.GymProgress)))

	for _, c := range f.counters() {
		line(strconv.FormatUint(uint64(*c), 10))
	}

	for s := 1; s <= NumSpecies; s++ {
		boolean(f.InPokedex[s])
	}

	for _, s := range f.Owned[:] {
		line(strconv.Itoa(int(s.Species)))
		line(strconv.Itoa(int(s.Level)))
		line(strconv.FormatUint(uint64(s.XP), 10))
	}

	return buf.Bytes()
}

// boolText is a Boolean as the file writes it.
func boolText(v bool) string {
	if v {
		return "TRUE"
	}

	return "FALSE"
}

// counters lists the Cardinal fields in file order.
func (f *File) counters() []*uint32 {
	return []*uint32{
		&f.Money, &f.Wins, &f.Losses, &f.Fights, &f.Catches,
		&f.PokeBalls, &f.GreatBalls, &f.UltraBalls, &f.MasterBalls, &f.Potions, &f.SuperPotions,
	}
}
