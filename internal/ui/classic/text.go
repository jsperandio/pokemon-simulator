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
	width(txt string) int
	// draw draws s with its cell's top-left corner at (x, y) and returns
	// the x after it.
	draw(cv canvas, x, y int, txt string, clr color.RGBA) int
}

// bitmapFace is a size of a Windows bitmap font; bold is synthesized as
// Windows does, by drawing every glyph twice, one pixel apart.
type bitmapFace struct {
	fnt  *winfont.Font
	bold bool
}

func (bf bitmapFace) height() int {
	return bf.fnt.Height
}

func (bf bitmapFace) ascent() int {
	return bf.fnt.Ascent
}

func (bf bitmapFace) width(txt string) int {
	return bf.fnt.Width(txt, bf.bold)
}

func (bf bitmapFace) draw(cv canvas, x, y int, txt string, clr color.RGBA) int {
	pnt := image.Pt(x, y).Add(cv.org)

	return bf.fnt.Draw(cv.img, cv.clip, pnt.X, pnt.Y, txt, clr, bf.bold) - cv.org.X
}

// trueTypeFace is a TrueType font at a pixel size, with GDI's metrics: the
// ascent and descent are rounded up, and glyphs advance by whole pixels,
// without kerning.
type trueTypeFace struct {
	fnt      font.Face
	asc, hgt int
}

// newTrueTypeFace returns a font at an em height of px pixels: a VCL font with
// Height = -px.
func newTrueTypeFace(fnt *opentype.Font, px int) (*trueTypeFace, error) {
	ff, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    float64(px),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	met := ff.Metrics()
	asc, desc := met.Ascent.Ceil(), met.Descent.Ceil()

	return &trueTypeFace{
		fnt: ff,
		asc: asc,
		hgt: asc + desc,
	}, nil
}

func (ttf *trueTypeFace) height() int {
	return ttf.hgt
}

func (ttf *trueTypeFace) ascent() int {
	return ttf.asc
}

func (ttf *trueTypeFace) width(txt string) int {
	wid := 0

	for _, chr := range txt {
		fix, _ := ttf.fnt.GlyphAdvance(chr)
		wid += fix.Round()
	}

	return wid
}

func (ttf *trueTypeFace) draw(cv canvas, x, y int, txt string, clr color.RGBA) int {
	dst, ok := cv.img.SubImage(cv.clip).(*image.RGBA)
	if !ok || dst.Rect.Empty() {
		return x + ttf.width(txt)
	}

	src := image.NewUniform(clr)

	for _, chr := range txt {
		dot := fixed.P(cv.org.X+x, cv.org.Y+y+ttf.asc)
		if dr, mask, mp, _, ok := ttf.fnt.Glyph(dot, chr); ok {
			draw.DrawMask(dst, dr, src, image.Point{}, mask, mp, draw.Over)
		}

		fix, _ := ttf.fnt.GlyphAdvance(chr)
		x += fix.Round()
	}

	return x
}

// Captions use "&" to mark the next character as an accelerator, drawn
// underlined; "&&" is a literal "&".

// plain returns a caption without its accelerator marks.
func plain(txt string) string {
	if !strings.Contains(txt, "&") {
		return txt
	}

	var bld strings.Builder

	for i := 0; i < len(txt); i++ {
		if txt[i] == '&' && i+1 < len(txt) {
			i++
		}

		bld.WriteByte(txt[i])
	}

	return bld.String()
}

// isDigit reports whether chr is an ASCII digit.
func isDigit(chr rune) bool {
	return chr >= '0' && chr <= '9'
}

// accel returns a caption's accelerator(shortcut), lowercased, or 0.
func accel(txt string) rune {
	for i := 0; i+1 < len(txt); i++ {
		if txt[i] != '&' {
			continue
		}

		if txt[i+1] != '&' {
			return []rune(strings.ToLower(txt[i+1:]))[0]
		}

		i++
	}

	return 0
}

// drawCaption draws a caption at (x, y), underlining its accelerator one
// pixel below the baseline.
func drawCaption(cv canvas, fnt face, x, y int, txt string, clr color.RGBA) {
	for txt != "" {
		i := strings.IndexByte(txt, '&')
		if i < 0 || i+1 >= len(txt) {
			fnt.draw(cv, x, y, txt, clr)

			return
		}

		x = fnt.draw(cv, x, y, txt[:i], clr)
		txt = txt[i+1:]

		if txt[0] == '&' {
			x = fnt.draw(cv, x, y, "&", clr)
			txt = txt[1:]

			continue
		}

		n := len(string([]rune(txt)[0]))
		x0 := x
		x = fnt.draw(cv, x, y, txt[:n], clr)
		cv.hline(x0, x, y+fnt.ascent()+1, clr)

		txt = txt[n:]
	}
}

// drawDisabled draws a caption as Windows draws disabled text: embossed,
// in white one pixel down and right, then in the shadow colour.
func drawDisabled(cv canvas, fnt face, x, y int, txt string) {
	drawCaption(cv, fnt, x+1, y+1, txt, colHighlight)
	drawCaption(cv, fnt, x, y, txt, colShadow)
}

// drawText draws a caption in colour clr, or disabled when it isn't enabled.
func drawText(cv canvas, fnt face, x, y int, txt string, clr color.RGBA, enabled bool) {
	if !enabled {
		drawDisabled(cv, fnt, x, y, txt)

		return
	}

	drawCaption(cv, fnt, x, y, txt, clr)
}

// wrap breaks text into lines no wider than wid, at spaces, as DrawText's
// DT_WORDBREAK does; a word too long for a line stays whole.
func wrap(fnt face, text string, wid int) []string {
	var lines []string

	for para := range strings.SplitSeq(text, "\n") {
		para = strings.TrimSuffix(para, "\r")
		line := ""

		for _, word := range strings.SplitAfter(para, " ") {
			if line != "" && fnt.width(strings.TrimRight(line+word, " ")) > wid {
				lines = append(lines, strings.TrimRight(line, " "))
				line = ""
			}

			line += word
		}

		lines = append(lines, strings.TrimRight(line, " "))
	}

	return lines
}
