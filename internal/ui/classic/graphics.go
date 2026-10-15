// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
)

// rgb is a colour written as 0xRRGGBB.
func rgb(val uint32) color.RGBA {
	return color.RGBA{
		R: uint8(val >> 16 & 0xff),
		G: uint8(val >> 8 & 0xff),
		B: uint8(val & 0xff),
		A: 0xff,
	}
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
	"clBlack":         rgb(0x000000),
	"clMaroon":        rgb(0x800000),
	"clGreen":         rgb(0x008000),
	"clOlive":         rgb(0x808000),
	"clNavy":          rgb(0x000080),
	"clPurple":        rgb(0x800080),
	"clTeal":          rgb(0x008080),
	"clGray":          rgb(0x808080),
	"clSilver":        rgb(0xC0C0C0),
	"clRed":           rgb(0xFF0000),
	"clLime":          rgb(0x00FF00),
	"clYellow":        rgb(0xFFFF00),
	"clBlue":          rgb(0x0000FF),
	"clFuchsia":       rgb(0xFF00FF),
	"clAqua":          rgb(0x00FFFF),
	"clWhite":         rgb(0xFFFFFF),
	"clBtnFace":       colFace,
	"clWindow":        colWindow,
	"clWindowText":    colText,
	"clBtnText":       colText,
	"clBtnShadow":     colShadow,
	"clBtnHighlight":  colHighlight,
	"clHighlight":     colSelection,
	"clHighlightText": colSelText,
	"clGrayText":      colGrayText,
	"clInfoBk":        colInfo,
}

// vclColor returns a colour property's value, or def if it is unset or
// unknown.
func vclColor(name string, def color.RGBA) color.RGBA {
	if clr, ok := vclColors[name]; ok {
		return clr
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
	return canvas{
		img:  img,
		clip: img.Rect,
	}
}

// sub returns a canvas for rect (in cv's coordinates), with rect's top-left
// corner as its origin, clipped to rect.
func (cv canvas) sub(rect image.Rectangle) canvas {
	rect = rect.Add(cv.org)

	return canvas{
		img:  cv.img,
		org:  rect.Min,
		clip: cv.clip.Intersect(rect),
	}
}

// fill fills rect.
func (cv canvas) fill(rect image.Rectangle, clr color.RGBA) {
	rect = rect.Add(cv.org).Intersect(cv.clip)
	if rect.Empty() {
		return
	}

	row := cv.img.Pix[cv.img.PixOffset(rect.Min.X, rect.Min.Y):]
	n := rect.Dx() * 4

	for i := 0; i < n; i += 4 {
		row[i], row[i+1], row[i+2], row[i+3] = clr.R, clr.G, clr.B, clr.A
	}

	first := row[:n]
	for y := rect.Min.Y + 1; y < rect.Max.Y; y++ {
		copy(cv.img.Pix[cv.img.PixOffset(rect.Min.X, y):], first)
	}
}

// pix sets one pixel.
func (cv canvas) pix(x, y int, clr color.RGBA) {
	pnt := image.Pt(x, y).Add(cv.org)
	if pnt.In(cv.clip) {
		i := cv.img.PixOffset(pnt.X, pnt.Y)
		cv.img.Pix[i], cv.img.Pix[i+1], cv.img.Pix[i+2], cv.img.Pix[i+3] = clr.R, clr.G, clr.B, clr.A
	}
}

// at returns a pixel, or black outside the image.
func (cv canvas) at(x, y int) color.RGBA {
	pnt := image.Pt(x, y).Add(cv.org)
	if !pnt.In(cv.img.Rect) {
		return color.RGBA{A: 0xff}
	}

	return cv.img.RGBAAt(pnt.X, pnt.Y)
}

// invert inverts one pixel, as an XOR with white does.
func (cv canvas) invert(x, y int) {
	pnt := image.Pt(x, y).Add(cv.org)
	if pnt.In(cv.clip) {
		i := cv.img.PixOffset(pnt.X, pnt.Y)
		cv.img.Pix[i] ^= 0xff
		cv.img.Pix[i+1] ^= 0xff
		cv.img.Pix[i+2] ^= 0xff
	}
}

// hline draws [x0, x1) on row y; vline draws [y0, y1) on column x.
func (cv canvas) hline(x0, x1, y int, clr color.RGBA) {
	cv.fill(image.Rect(x0, y, x1, y+1), clr)
}

func (cv canvas) vline(x, y0, y1 int, clr color.RGBA) {
	cv.fill(image.Rect(x, y0, x+1, y1), clr)
}

// frame draws rect's outline.
func (cv canvas) frame(rect image.Rectangle, clr color.RGBA) {
	cv.edge(rect, clr, clr)
}

// edge draws rect's outline with its top and left sides in tl and its bottom
// and right sides in br, which win at the corners, as DrawEdge does.
func (cv canvas) edge(rect image.Rectangle, tl, br color.RGBA) {
	cv.hline(rect.Min.X, rect.Max.X, rect.Min.Y, tl)
	cv.vline(rect.Min.X, rect.Min.Y, rect.Max.Y, tl)
	cv.hline(rect.Min.X, rect.Max.X, rect.Max.Y-1, br)
	cv.vline(rect.Max.X-1, rect.Min.Y, rect.Max.Y, br)
}

// The 3D borders: two one-pixel edges, the outer one first. sunken returns
// the rectangle inside it.
func (cv canvas) raised(rect image.Rectangle) { // windows, panels
	cv.edge(rect, colLight, colDkShadow)
	cv.edge(rect.Inset(1), colHighlight, colShadow)
}

func (cv canvas) raisedSoft(rect image.Rectangle) { // buttons
	cv.edge(rect, colHighlight, colDkShadow)
	cv.edge(rect.Inset(1), colLight, colShadow)
}

func (cv canvas) sunken(rect image.Rectangle) image.Rectangle { // edit boxes, lists
	cv.edge(rect, colShadow, colHighlight)

	rect = rect.Inset(1)
	cv.edge(rect, colDkShadow, colLight)

	return rect.Inset(1)
}

func (cv canvas) sunkenSoft(rect image.Rectangle) { // pushed buttons
	cv.edge(rect, colDkShadow, colHighlight)
	cv.edge(rect.Inset(1), colShadow, colLight)
}

// softButton paints a button's face in rect, raised or sunken while pushed,
// and returns how far its glyph moves: one pixel down and right when
// pushed.
func (cv canvas) softButton(rect image.Rectangle, pushed bool) (off int) {
	cv.fill(rect, colFace)

	if pushed {
		cv.sunkenSoft(rect)

		return 1
	}

	cv.raisedSoft(rect)

	return 0
}

// embossed paints a disabled glyph: in white one pixel down and right,
// then in the shadow colour.
func (cv canvas) embossed(x, y int, rows string) {
	cv.pattern(x+1, y+1, rows, map[byte]color.RGBA{'#': colHighlight})
	cv.pattern(x, y, rows, map[byte]color.RGBA{'#': colShadow})
}

// focusRect inverts every other pixel of rect's outline, as DrawFocusRect
// does. The dots fall where x + y is even on the screen, as Windows lines
// the patterns of all the windows up on the screen.
func (cv canvas) focusRect(rect image.Rectangle) {
	if rect.Empty() {
		return
	}

	dot := func(x, y int) {
		if (x+cv.org.X+y+cv.org.Y)&1 == 0 {
			cv.invert(x, y)
		}
	}
	for x := rect.Min.X; x < rect.Max.X; x++ {
		dot(x, rect.Min.Y)

		if rect.Dy() > 1 {
			dot(x, rect.Max.Y-1)
		}
	}

	for y := rect.Min.Y + 1; y < rect.Max.Y-1; y++ {
		dot(rect.Min.X, y)

		if rect.Dx() > 1 {
			dot(rect.Max.X-1, y)
		}
	}
}

// gradient fills rect with a horizontal gradient from lft to rgt, as captions are
// drawn.
func (cv canvas) gradient(rect image.Rectangle, lft, rgt color.RGBA) {
	wid := rect.Dx()
	for x := range wid {
		fac := 0
		if wid > 1 {
			fac = x * 256 / (wid - 1)
		}

		mix := func(lhs, rhs uint8) uint8 {
			return uint8((int(lhs)*(256-fac) + int(rhs)*fac) / 256) //nolint:gosec // a weighted mean of two uint8s
		}
		cv.vline(rect.Min.X+x, rect.Min.Y, rect.Max.Y, color.RGBA{mix(lft.R, rgt.R), mix(lft.G, rgt.G), mix(lft.B, rgt.B), 0xff})
	}
}

// draw copies src's rectangle sr to (x, y), skipping transparent pixels:
// the sprites' masks are one bit deep.
func (cv canvas) draw(x, y int, src *image.NRGBA, sr image.Rectangle) {
	for sy := sr.Min.Y; sy < sr.Max.Y; sy++ {
		for sx := sr.Min.X; sx < sr.Max.X; sx++ {
			clr := src.NRGBAAt(sx, sy)
			if clr.A < 0x80 {
				continue
			}

			cv.pix(x+sx-sr.Min.X, y+sy-sr.Min.Y, color.RGBA{clr.R, clr.G, clr.B, 0xff})
		}
	}
}

// copy copies src's rectangle sr to (x, y), as BitBlt does.
func (cv canvas) copy(x, y int, src *image.RGBA, sr image.Rectangle) {
	dr := image.Rectangle{
		Min: image.Pt(x, y),
		Max: image.Pt(x, y).Add(sr.Size()),
	}.Add(cv.org)
	rect := dr.Intersect(cv.clip)

	if rect.Empty() {
		return
	}

	draw.Draw(cv.img, rect, src, sr.Min.Add(rect.Min.Sub(dr.Min)), draw.Src)
}

// blend draws src's rectangle sr at (x, y) with its alpha channel, for the
// dialog icons.
func (cv canvas) blend(x, y int, src *image.NRGBA, sr image.Rectangle) {
	for sy := sr.Min.Y; sy < sr.Max.Y; sy++ {
		for sx := sr.Min.X; sx < sr.Max.X; sx++ {
			spx := src.NRGBAAt(sx, sy)
			if spx.A == 0 {
				continue
			}

			px, py := x+sx-sr.Min.X, y+sy-sr.Min.Y
			dpx := cv.at(px, py)
			alp := int(spx.A)
			mix := func(fgd, bgd uint8) uint8 {
				return uint8((int(fgd)*alp + int(bgd)*(255-alp) + 127) / 255) //nolint:gosec // a weighted mean of two uint8s
			}
			cv.pix(px, py, color.RGBA{mix(spx.R, dpx.R), mix(spx.G, dpx.G), mix(spx.B, dpx.B), 0xff})
		}
	}
}

// pattern draws a picture given as rows of characters, one per pixel:
// each character's colour comes from colors, and spaces are transparent.
func (cv canvas) pattern(x, y int, rows string, colors map[byte]color.RGBA) {
	for dy, row := range strings.Split(strings.Trim(rows, "\n"), "\n") {
		for dx := range len(row) {
			if clr, ok := colors[row[dx]]; ok {
				cv.pix(x+dx, y+dy, clr)
			}
		}
	}
}
