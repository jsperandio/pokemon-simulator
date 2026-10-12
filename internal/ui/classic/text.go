// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/winfont"
)

// A face is a font at one size, measured as GDI measures it.
type face interface {
	height() int // the cell height (tmHeight)
	ascent() int // from the cell's top to the baseline
	width(s string) int
	// draw draws s with its cell's top-left corner at (x, y) and returns
	// the x after it.
	draw(cv canvas, x, y int, s string, c color.RGBA) int
}

// bitmapFace is a size of a Windows bitmap font; bold is synthesized as
// Windows does, by drawing every glyph twice, one pixel apart.
type bitmapFace struct {
	f    *winfont.Font
	bold bool
}

func (b bitmapFace) height() int        { return b.f.Height }
func (b bitmapFace) ascent() int        { return b.f.Ascent }
func (b bitmapFace) width(s string) int { return b.f.Width(s, b.bold) }

func (b bitmapFace) draw(cv canvas, x, y int, s string, c color.RGBA) int {
	p := image.Pt(x, y).Add(cv.org)
	return b.f.Draw(cv.img, cv.clip, p.X, p.Y, s, c, b.bold) - cv.org.X
}

// trueTypeFace is a TrueType font at a pixel size, with GDI's metrics: the
// ascent and descent are rounded up, and glyphs advance by whole pixels,
// without kerning.
type trueTypeFace struct {
	f        font.Face
	asc, hgt int
}

// newTrueType returns font data at an em height of px pixels: a VCL font
// with Height = -px.
func newTrueType(data []byte, px int) (*trueTypeFace, error) {
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}
	ff, err := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(px), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	m := ff.Metrics()
	asc, desc := m.Ascent.Ceil(), m.Descent.Ceil()
	return &trueTypeFace{f: ff, asc: asc, hgt: asc + desc}, nil
}

func (t *trueTypeFace) height() int { return t.hgt }
func (t *trueTypeFace) ascent() int { return t.asc }

func (t *trueTypeFace) width(s string) int {
	w := 0
	for _, r := range s {
		a, _ := t.f.GlyphAdvance(r)
		w += a.Round()
	}
	return w
}

func (t *trueTypeFace) draw(cv canvas, x, y int, s string, c color.RGBA) int {
	dst, ok := cv.img.SubImage(cv.clip).(*image.RGBA)
	if !ok || dst.Rect.Empty() {
		return x + t.width(s)
	}
	src := image.NewUniform(c)
	for _, r := range s {
		dot := fixed.P(cv.org.X+x, cv.org.Y+y+t.asc)
		if dr, mask, mp, _, ok := t.f.Glyph(dot, r); ok {
			draw.DrawMask(dst, dr, src, image.Point{}, mask, mp, draw.Over)
		}
		a, _ := t.f.GlyphAdvance(r)
		x += a.Round()
	}
	return x
}

// Captions use "&" to mark the next character as an accelerator, drawn
// underlined; "&&" is a literal "&".

// plain returns a caption without its accelerator marks.
func plain(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '&' && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// accel returns a caption's accelerator, lowercased, or 0.
func accel(s string) rune {
	for i := 0; i+1 < len(s); i++ {
		if s[i] != '&' {
			continue
		}
		if s[i+1] != '&' {
			return []rune(strings.ToLower(s[i+1:]))[0]
		}
		i++
	}
	return 0
}

// drawCaption draws a caption at (x, y), underlining its accelerator one
// pixel below the baseline, and returns the x after it.
func drawCaption(cv canvas, f face, x, y int, s string, c color.RGBA) int {
	for len(s) > 0 {
		i := strings.IndexByte(s, '&')
		if i < 0 || i+1 >= len(s) {
			return f.draw(cv, x, y, s, c)
		}
		x = f.draw(cv, x, y, s[:i], c)
		s = s[i+1:]
		if s[0] == '&' {
			x = f.draw(cv, x, y, "&", c)
			s = s[1:]
			continue
		}
		n := len(string([]rune(s)[0]))
		x0 := x
		x = f.draw(cv, x, y, s[:n], c)
		cv.hline(x0, x, y+f.ascent()+1, c)
		s = s[n:]
	}
	return x
}

// drawDisabled draws a caption as Windows draws disabled text: embossed,
// in white one pixel down and right, then in the shadow colour.
func drawDisabled(cv canvas, f face, x, y int, s string) int {
	drawCaption(cv, f, x+1, y+1, s, colHighlight)
	return drawCaption(cv, f, x, y, s, colShadow)
}

// wrap breaks text into lines no wider than w, at spaces, as DrawText's
// DT_WORDBREAK does; a word too long for a line stays whole.
func wrap(f face, text string, w int) []string {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		para = strings.TrimSuffix(para, "\r")
		line := ""
		for _, word := range strings.SplitAfter(para, " ") {
			if line != "" && f.width(strings.TrimRight(line+word, " ")) > w {
				lines = append(lines, strings.TrimRight(line, " "))
				line = ""
			}
			line += word
		}
		lines = append(lines, strings.TrimRight(line, " "))
	}
	return lines
}
