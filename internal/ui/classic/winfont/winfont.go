// SPDX-License-Identifier: AGPL-3.0-only

// Package winfont reads Windows bitmap fonts (.fon files: FNT resources in
// a 16-bit NE executable) and draws text with them, pixel for pixel as
// Windows does.
package winfont

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
)

// Font is one size of a bitmap font. Text is drawn in cp1252, the
// original's code page: characters outside it are drawn as "?".
type Font struct {
	Name   string
	Points int
	Height int // the cell height in pixels
	Ascent int // pixels from the cell's top to the baseline

	first, last, dflt byte
	glyphs            []glyph // first..last
}

type glyph struct {
	width int
	rows  []uint32 // one bit per pixel, bit 31 the leftmost
}

// Parse returns the fonts of a .fon file.
func Parse(b []byte) ([]*Font, error) {
	if len(b) < 0x40 || string(b[:2]) != "MZ" {
		return nil, errors.New("winfont: not an executable")
	}

	ne := int(binary.LittleEndian.Uint32(b[0x3C:]))
	if ne+0x26 > len(b) || string(b[ne:ne+2]) != "NE" {
		return nil, errors.New("winfont: not a 16-bit (NE) executable")
	}

	p := ne + int(binary.LittleEndian.Uint16(b[ne+0x24:]))
	if p+2 > len(b) {
		return nil, errors.New("winfont: bad resource table")
	}

	shift := binary.LittleEndian.Uint16(b[p:])
	p += 2

	var fonts []*Font

	for {
		if p+8 > len(b) {
			return nil, errors.New("winfont: truncated resource table")
		}

		typ, n := binary.LittleEndian.Uint16(b[p:]), int(binary.LittleEndian.Uint16(b[p+2:]))
		p += 8

		if typ == 0 {
			break
		}

		for range n {
			if p+12 > len(b) {
				return nil, errors.New("winfont: truncated resource table")
			}

			off := int(binary.LittleEndian.Uint16(b[p:])) << shift
			size := int(binary.LittleEndian.Uint16(b[p+2:])) << shift
			p += 12

			if typ != 0x8008 { // RT_FONT
				continue
			}

			if off+size > len(b) {
				return nil, errors.New("winfont: resource out of the file")
			}

			f, err := parseFNT(b[off : off+size])
			if err != nil {
				return nil, err
			}

			fonts = append(fonts, f)
		}
	}

	if len(fonts) == 0 {
		return nil, errors.New("winfont: no fonts")
	}

	return fonts, nil
}

// parseFNT reads a version 2 or 3 FNT resource.
func parseFNT(b []byte) (*Font, error) {
	if len(b) < 118 {
		return nil, errors.New("winfont: truncated font")
	}

	version := u16(b, 0)

	if version != 0x200 && version != 0x300 {
		return nil, fmt.Errorf("winfont: FNT version %#x", version)
	}

	if u16(b, 66)&1 != 0 {
		return nil, errors.New("winfont: a vector font")
	}

	f := &Font{
		Name:   faceName(b, u32(b, 105)),
		Points: u16(b, 68),
		Ascent: u16(b, 74),
		Height: u16(b, 88),
		first:  b[95],
		last:   b[96],
	}
	f.dflt = f.first + b[97]

	if err := f.readGlyphs(b, version); err != nil {
		return nil, err
	}

	return f, nil
}

func u16(b []byte, o int) int {
	return int(binary.LittleEndian.Uint16(b[o:]))
}

func u32(b []byte, o int) int {
	return int(binary.LittleEndian.Uint32(b[o:]))
}

// faceName reads the face name that starts at face, ended by a NUL, or
// returns "" for none.
func faceName(b []byte, face int) string {
	if face <= 0 || face >= len(b) {
		return ""
	}

	end := face
	for end < len(b) && b[end] != 0 {
		end++
	}

	return string(b[face:end])
}

// readGlyphs reads the character table of an FNT resource of the given
// version, and the glyphs' bitmaps it points at.
func (f *Font) readGlyphs(b []byte, version int) error {
	table, entry := 118, 4
	if version == 0x300 {
		table, entry = 148, 6
	}

	n := int(f.last) - int(f.first) + 1
	if n <= 0 || table+n*entry > len(b) || f.Height <= 0 {
		return errors.New("winfont: bad character table")
	}

	f.glyphs = make([]glyph, n)
	for i := range f.glyphs {
		e := table + i*entry
		w := u16(b, e)

		var off int

		if version == 0x300 {
			off = u32(b, e+2)
		} else {
			off = u16(b, e+2)
		}

		cols := (w + 7) / 8
		if w > 32 || off+cols*f.Height > len(b) {
			return fmt.Errorf("winfont: glyph %d out of range", int(f.first)+i)
		}

		g := glyph{width: w, rows: make([]uint32, f.Height)}
		// Each byte column holds 8 pixels of every row, top to bottom.
		for c := range cols {
			for y := range f.Height {
				g.rows[y] |= uint32(b[off+c*f.Height+y]) << (24 - 8*c)
			}
		}

		f.glyphs[i] = g
	}

	return nil
}

func (f *Font) glyph(c byte) *glyph {
	if c < f.first || c > f.last {
		c = f.dflt
	}

	return &f.glyphs[c-f.first]
}

// Width is the width of s in pixels; bold text is one pixel wider per
// character, as Windows emboldens bitmap fonts.
func (f *Font) Width(s string, bold bool) int {
	w := 0
	for _, c := range delphi.EncodeANSI(s) {
		w += f.glyph(c).width
		if bold {
			w++
		}
	}

	return w
}

// Draw draws s with its cell's top-left corner at (x, y), clipped to clip,
// and returns the x after it. Bold draws every glyph twice, one pixel
// apart.
func (f *Font) Draw(dst *image.RGBA, clip image.Rectangle, x, y int, s string, c color.RGBA, bold bool) int {
	clip = clip.Intersect(dst.Rect)

	for _, ch := range delphi.EncodeANSI(s) {
		g := f.glyph(ch)
		f.drawGlyph(dst, clip, x, y, g, c)

		if bold {
			f.drawGlyph(dst, clip, x+1, y, g, c)

			x++
		}

		x += g.width
	}

	return x
}

func (f *Font) drawGlyph(dst *image.RGBA, clip image.Rectangle, x, y int, g *glyph, c color.RGBA) {
	for row, bits := range g.rows {
		py := y + row
		if py < clip.Min.Y || py >= clip.Max.Y || bits == 0 {
			continue
		}

		for col := range g.width {
			if bits&(1<<(31-col)) == 0 {
				continue
			}

			px := x + col
			if px < clip.Min.X || px >= clip.Max.X {
				continue
			}

			i := dst.PixOffset(px, py)
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = c.R, c.G, c.B, c.A
		}
	}
}
