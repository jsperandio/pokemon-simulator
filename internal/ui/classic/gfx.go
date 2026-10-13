// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
)

// rgb is a colour written as 0xRRGGBB.
func rgb(v uint32) color.RGBA {
	return color.RGBA{R: uint8(v >> 16 & 0xff), G: uint8(v >> 8 & 0xff), B: uint8(v & 0xff), A: 0xff}
}

// Windows 98's standard colour scheme.
var (
	colFace         = rgb(0xC0C0C0) // COLOR_3DFACE
	colHighlight    = rgb(0xFFFFFF) // COLOR_3DHIGHLIGHT
	colLight        = rgb(0xDFDFDF) // COLOR_3DLIGHT
	colShadow       = rgb(0x808080) // COLOR_3DSHADOW
	colDkShadow     = rgb(0x000000) // COLOR_3DDKSHADOW
	colWindow       = rgb(0xFFFFFF) // COLOR_WINDOW
	colText         = rgb(0x000000) // COLOR_WINDOWTEXT, COLOR_BTNTEXT, COLOR_MENUTEXT
	colSelection    = rgb(0x000080) // COLOR_HIGHLIGHT
	colSelText      = rgb(0xFFFFFF) // COLOR_HIGHLIGHTTEXT
	colGrayText     = rgb(0x808080) // COLOR_GRAYTEXT
	colFrame        = rgb(0x000000) // COLOR_WINDOWFRAME
	colCaption      = rgb(0x000080) // COLOR_ACTIVECAPTION
	colCaption2     = rgb(0x1084D0) // COLOR_GRADIENTACTIVECAPTION
	colCaptionText  = rgb(0xFFFFFF) // COLOR_CAPTIONTEXT
	colInactive     = rgb(0x808080) // COLOR_INACTIVECAPTION
	colInactive2    = rgb(0xC0C0C0) // COLOR_GRADIENTINACTIVECAPTION
	colInactiveText = rgb(0xC0C0C0) // COLOR_INACTIVECAPTIONTEXT
	colDesktop      = rgb(0x008080) // COLOR_DESKTOP
	colInfo         = rgb(0xFFFFE1) // COLOR_INFOBK
)

// vclColors are the VCL's colour constants the forms use.
var vclColors = map[string]color.RGBA{
	"clBlack": rgb(0x000000), "clMaroon": rgb(0x800000), "clGreen": rgb(0x008000),
	"clOlive": rgb(0x808000), "clNavy": rgb(0x000080), "clPurple": rgb(0x800080),
	"clTeal": rgb(0x008080), "clGray": rgb(0x808080), "clSilver": rgb(0xC0C0C0),
	"clRed": rgb(0xFF0000), "clLime": rgb(0x00FF00), "clYellow": rgb(0xFFFF00),
	"clBlue": rgb(0x0000FF), "clFuchsia": rgb(0xFF00FF), "clAqua": rgb(0x00FFFF),
	"clWhite":   rgb(0xFFFFFF),
	"clBtnFace": colFace, "clWindow": colWindow, "clWindowText": colText,
	"clBtnText": colText, "clBtnShadow": colShadow, "clBtnHighlight": colHighlight,
	"clHighlight": colSelection, "clHighlightText": colSelText, "clGrayText": colGrayText,
	"clInfoBk": colInfo,
}

// vclColor returns a colour property's value, or def if it is unset or
// unknown.
func vclColor(name string, def color.RGBA) color.RGBA {
	if c, ok := vclColors[name]; ok {
		return c
	}

	return def
}

// A canvas draws on part of an image, with its own origin and clipping,
// as a device context does.
type canvas struct {
	img  *image.RGBA
	org  image.Point     // where (0, 0) is, in img
	clip image.Rectangle // in img
}

func newCanvas(img *image.RGBA) canvas {
	return canvas{img: img, clip: img.Rect}
}

// sub returns a canvas for r (in cv's coordinates), with r's top-left
// corner as its origin, clipped to r.
func (cv canvas) sub(r image.Rectangle) canvas {
	r = r.Add(cv.org)

	return canvas{img: cv.img, org: r.Min, clip: cv.clip.Intersect(r)}
}

// fill fills r.
func (cv canvas) fill(r image.Rectangle, c color.RGBA) {
	r = r.Add(cv.org).Intersect(cv.clip)
	if r.Empty() {
		return
	}

	row := cv.img.Pix[cv.img.PixOffset(r.Min.X, r.Min.Y):]
	n := r.Dx() * 4

	for i := 0; i < n; i += 4 {
		row[i], row[i+1], row[i+2], row[i+3] = c.R, c.G, c.B, c.A
	}

	first := row[:n]
	for y := r.Min.Y + 1; y < r.Max.Y; y++ {
		copy(cv.img.Pix[cv.img.PixOffset(r.Min.X, y):], first)
	}
}

// pix sets one pixel.
func (cv canvas) pix(x, y int, c color.RGBA) {
	p := image.Pt(x, y).Add(cv.org)
	if p.In(cv.clip) {
		i := cv.img.PixOffset(p.X, p.Y)
		cv.img.Pix[i], cv.img.Pix[i+1], cv.img.Pix[i+2], cv.img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
}

// at returns a pixel, or black outside the image.
func (cv canvas) at(x, y int) color.RGBA {
	p := image.Pt(x, y).Add(cv.org)
	if !p.In(cv.img.Rect) {
		return color.RGBA{A: 0xff}
	}

	return cv.img.RGBAAt(p.X, p.Y)
}

// invert inverts one pixel, as an XOR with white does.
func (cv canvas) invert(x, y int) {
	p := image.Pt(x, y).Add(cv.org)
	if p.In(cv.clip) {
		i := cv.img.PixOffset(p.X, p.Y)
		cv.img.Pix[i] ^= 0xff
		cv.img.Pix[i+1] ^= 0xff
		cv.img.Pix[i+2] ^= 0xff
	}
}

// hline draws [x0, x1) on row y; vline draws [y0, y1) on column x.
func (cv canvas) hline(x0, x1, y int, c color.RGBA) {
	cv.fill(image.Rect(x0, y, x1, y+1), c)
}

func (cv canvas) vline(x, y0, y1 int, c color.RGBA) {
	cv.fill(image.Rect(x, y0, x+1, y1), c)
}

// frame draws r's outline.
func (cv canvas) frame(r image.Rectangle, c color.RGBA) {
	cv.edge(r, c, c)
}

// edge draws r's outline with its top and left sides in tl and its bottom
// and right sides in br, which win at the corners, as DrawEdge does.
func (cv canvas) edge(r image.Rectangle, tl, br color.RGBA) {
	cv.hline(r.Min.X, r.Max.X, r.Min.Y, tl)
	cv.vline(r.Min.X, r.Min.Y, r.Max.Y, tl)
	cv.hline(r.Min.X, r.Max.X, r.Max.Y-1, br)
	cv.vline(r.Max.X-1, r.Min.Y, r.Max.Y, br)
}

// The 3D borders: two one-pixel edges, the outer one first. sunken returns
// the rectangle inside it.
func (cv canvas) raised(r image.Rectangle) { // windows, panels
	cv.edge(r, colLight, colDkShadow)
	cv.edge(r.Inset(1), colHighlight, colShadow)
}

func (cv canvas) raisedSoft(r image.Rectangle) { // buttons
	cv.edge(r, colHighlight, colDkShadow)
	cv.edge(r.Inset(1), colLight, colShadow)
}

func (cv canvas) sunken(r image.Rectangle) image.Rectangle { // edit boxes, lists
	cv.edge(r, colShadow, colHighlight)

	r = r.Inset(1)
	cv.edge(r, colDkShadow, colLight)

	return r.Inset(1)
}

func (cv canvas) sunkenSoft(r image.Rectangle) { // pushed buttons
	cv.edge(r, colDkShadow, colHighlight)
	cv.edge(r.Inset(1), colShadow, colLight)
}

// focusRect inverts every other pixel of r's outline, as DrawFocusRect
// does. The dots fall where x + y is even on the screen, as Windows lines
// the patterns of all the windows up on the screen.
func (cv canvas) focusRect(r image.Rectangle) {
	if r.Empty() {
		return
	}

	dot := func(x, y int) {
		if (x+cv.org.X+y+cv.org.Y)&1 == 0 {
			cv.invert(x, y)
		}
	}
	for x := r.Min.X; x < r.Max.X; x++ {
		dot(x, r.Min.Y)

		if r.Dy() > 1 {
			dot(x, r.Max.Y-1)
		}
	}

	for y := r.Min.Y + 1; y < r.Max.Y-1; y++ {
		dot(r.Min.X, y)

		if r.Dx() > 1 {
			dot(r.Max.X-1, y)
		}
	}
}

// gradient fills r with a horizontal gradient from a to b, as captions are
// drawn.
func (cv canvas) gradient(r image.Rectangle, a, b color.RGBA) {
	w := r.Dx()
	for x := range w {
		t := 0
		if w > 1 {
			t = x * 256 / (w - 1)
		}

		mix := func(p, q uint8) uint8 {
			return uint8((int(p)*(256-t) + int(q)*t) / 256) //nolint:gosec // a weighted mean of two uint8s
		}
		cv.vline(r.Min.X+x, r.Min.Y, r.Max.Y, color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 0xff})
	}
}

// draw copies src's rectangle sr to (x, y), skipping transparent pixels:
// the sprites' masks are one bit deep.
func (cv canvas) draw(x, y int, src *image.NRGBA, sr image.Rectangle) {
	for sy := sr.Min.Y; sy < sr.Max.Y; sy++ {
		for sx := sr.Min.X; sx < sr.Max.X; sx++ {
			c := src.NRGBAAt(sx, sy)
			if c.A < 0x80 {
				continue
			}

			cv.pix(x+sx-sr.Min.X, y+sy-sr.Min.Y, color.RGBA{c.R, c.G, c.B, 0xff})
		}
	}
}

// copy copies src's rectangle sr to (x, y), as BitBlt does.
func (cv canvas) copy(x, y int, src *image.RGBA, sr image.Rectangle) {
	dr := image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x, y).Add(sr.Size())}.Add(cv.org)
	r := dr.Intersect(cv.clip)

	if r.Empty() {
		return
	}

	draw.Draw(cv.img, r, src, sr.Min.Add(r.Min.Sub(dr.Min)), draw.Src)
}

// blend draws src's rectangle sr at (x, y) with its alpha channel, for the
// dialog icons.
func (cv canvas) blend(x, y int, src *image.NRGBA, sr image.Rectangle) {
	for sy := sr.Min.Y; sy < sr.Max.Y; sy++ {
		for sx := sr.Min.X; sx < sr.Max.X; sx++ {
			c := src.NRGBAAt(sx, sy)
			if c.A == 0 {
				continue
			}

			px, py := x+sx-sr.Min.X, y+sy-sr.Min.Y
			d := cv.at(px, py)
			a := int(c.A)
			mix := func(s, d uint8) uint8 {
				return uint8((int(s)*a + int(d)*(255-a) + 127) / 255) //nolint:gosec // a weighted mean of two uint8s
			}
			cv.pix(px, py, color.RGBA{mix(c.R, d.R), mix(c.G, d.G), mix(c.B, d.B), 0xff})
		}
	}
}

// pattern draws a picture given as rows of characters, one per pixel:
// each character's colour comes from colors, and spaces are transparent.
func (cv canvas) pattern(x, y int, rows string, colors map[byte]color.RGBA) {
	for dy, row := range strings.Split(strings.Trim(rows, "\n"), "\n") {
		for dx := range len(row) {
			if c, ok := colors[row[dx]]; ok {
				cv.pix(x+dx, y+dy, c)
			}
		}
	}
}
