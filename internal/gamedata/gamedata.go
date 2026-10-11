// SPDX-License-Identifier: AGPL-3.0-only

// Package gamedata loads the game's definitions from the original
// POKéGAME.dat: the type chart, species, badges, moves, tournament prizes
// and trainer names.
package gamedata

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
)

// Number of entries in each section of POKéGAME.dat.
const (
	NumTypes        = 17
	NumSpecies      = 386
	NumBadges       = 24
	NumMoves        = 119
	NumPrizes       = 18
	NumTrainerNames = 64
)

// Type is a POKéMON or move type. NoType marks the missing second type of a
// single-type species.
type Type int

// Types, in the order of the original type chart.
const (
	NoType Type = iota
	Normal
	Fire
	Water
	Grass
	Electric
	Ice
	Fighting
	Poison
	Ground
	Flying
	Psychic
	Bug
	Rock
	Ghost
	Dragon
	Dark
	Steel
)

// Species is a POKéMON species. IDs follow the original's "Old POKéDEX"
// order: 1–151 Kanto, 152–251 Johto, 252–386 Hoenn.
type Species struct {
	ID          int
	Name        string
	Types       [2]Type
	EvolveLevel int    // level at which it can evolve; 0 if it can't by level
	EvolveTo    int    // species it evolves into; 0 if none or chosen by the game
	Moves       [4]int // its four fixed moves, by move ID
}

// Move is an attack.
type Move struct {
	ID    int
	Name  string
	Power int
	Type  Type
}

// Data is the content of POKéGAME.dat.
type Data struct {
	TypeNames     [NumTypes]string
	Effectiveness [NumTypes][NumTypes]float64 // [attacking type-1][defending type-1]
	Species       []Species                   // Species[i] has ID i+1
	Badges        []string
	Moves         []Move   // Moves[i] has ID i+1
	Prizes        []string // tournament prizes
	TrainerNames  []string // names given to random trainers
}

// Load parses POKéGAME.dat. Like the original's loader, it reads the file
// one line per value, section after section, and ignores anything after
// the last section.
func Load(raw []byte) (*Data, error) {
	text := strings.ReplaceAll(delphi.DecodeANSI(raw), "\r\n", "\n")
	r := &lineReader{lines: strings.Split(text, "\n")}
	d := &Data{Species: make([]Species, NumSpecies), Moves: make([]Move, NumMoves)}

	for a := range NumTypes {
		for t := range NumTypes {
			d.Effectiveness[a][t] = r.float()
		}
	}
	for i := range d.Species {
		d.Species[i].ID = i + 1
		d.Species[i].Name = r.text()
	}
	d.Badges = r.texts(NumBadges)
	for i := range d.Species {
		d.Species[i].Types = [2]Type{Type(r.int()), Type(r.int())}
	}
	for i := range d.Species {
		d.Species[i].EvolveLevel = r.int()
	}
	for i := range d.Species {
		d.Species[i].EvolveTo = r.int()
	}
	for i := range d.Moves {
		d.Moves[i].ID = i + 1
		d.Moves[i].Name = r.text()
	}
	for i := range d.Moves {
		d.Moves[i].Power = r.int()
	}
	for i := range d.Moves {
		d.Moves[i].Type = Type(r.int())
	}
	for i := range d.TypeNames {
		d.TypeNames[i] = r.text()
	}
	for i := range d.Species {
		for m := range d.Species[i].Moves {
			d.Species[i].Moves[m] = r.int()
		}
	}
	d.Prizes = r.texts(NumPrizes)
	d.TrainerNames = r.texts(NumTrainerNames)
	if r.err != nil {
		return nil, r.err
	}
	return d, d.validate()
}

// validate checks that every reference points at an existing entry, so
// that a modified file fails here rather than in the middle of a battle.
func (d *Data) validate() error {
	for _, s := range d.Species {
		if s.Types[0] < Normal || s.Types[0] > Steel || s.Types[1] < NoType || s.Types[1] > Steel {
			return fmt.Errorf("pokegame.dat: %s has invalid types %v", s.Name, s.Types)
		}
		if s.EvolveTo < 0 || s.EvolveTo > NumSpecies {
			return fmt.Errorf("pokegame.dat: %s evolves into unknown species %d", s.Name, s.EvolveTo)
		}
		for _, m := range s.Moves {
			if d.MoveByID(m) == nil {
				return fmt.Errorf("pokegame.dat: %s knows unknown move %d", s.Name, m)
			}
		}
	}
	for _, m := range d.Moves {
		if m.Type < Normal || m.Type > Steel {
			return fmt.Errorf("pokegame.dat: move %s has invalid type %d", m.Name, m.Type)
		}
	}
	return nil
}

// SpeciesByID returns the species with the given ID, or nil.
func (d *Data) SpeciesByID(id int) *Species {
	if id < 1 || id > len(d.Species) {
		return nil
	}
	return &d.Species[id-1]
}

// MoveByID returns the move with the given ID, or nil.
func (d *Data) MoveByID(id int) *Move {
	if id < 1 || id > len(d.Moves) {
		return nil
	}
	return &d.Moves[id-1]
}

// TypeName returns the name of t, or "" for NoType.
func (d *Data) TypeName(t Type) string {
	if t < Normal || t > Steel {
		return ""
	}
	return d.TypeNames[t-1]
}

// Effect returns the damage multiplier of an attack of type atk against a
// defender of type def. NoType neither strengthens nor weakens (1).
func (d *Data) Effect(atk, def Type) float64 {
	if atk < Normal || atk > Steel || def < Normal || def > Steel {
		return 1
	}
	return d.Effectiveness[atk-1][def-1]
}

// lineReader hands out the lines of the file one value at a time and
// remembers the first error.
type lineReader struct {
	lines []string
	n     int
	err   error
}

func (r *lineReader) text() string {
	if r.err != nil {
		return ""
	}
	if r.n >= len(r.lines) {
		r.err = fmt.Errorf("pokegame.dat: unexpected end of file at line %d", r.n+1)
		return ""
	}
	r.n++
	return r.lines[r.n-1]
}

func (r *lineReader) texts(n int) []string {
	s := make([]string, n)
	for i := range s {
		s[i] = r.text()
	}
	return s
}

func (r *lineReader) int() int {
	s := r.text()
	if r.err != nil {
		return 0
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		r.err = fmt.Errorf("pokegame.dat: line %d: want a whole number, got %q", r.n, s)
	}
	return v
}

func (r *lineReader) float() float64 {
	s := r.text()
	if r.err != nil {
		return 0
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		r.err = fmt.Errorf("pokegame.dat: line %d: want a number, got %q", r.n, s)
	}
	return v
}
