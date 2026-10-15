// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strconv"
	"strings"

	"golang.org/x/image/font/opentype"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/winfont"
)

// resources are the decoded assets the classic UI draws with.
type resources struct {
	sans, bold face // MS Sans Serif 8, as the forms and captions use it
	ttf        [3]*opentype.Font
	arials     map[arialKey]face
	lists      map[string]*imageList
	pictures   map[string]*image.NRGBA
	dlgIcons   map[game.DialogKind]*image.NRGBA
	appIcon    *image.NRGBA // the caption's 16×16 icon
	charWidth  int          // the font's average character width (dialog units)
}

func loadResources() (*resources, error) {
	res := &resources{
		arials:   map[arialKey]face{},
		lists:    map[string]*imageList{},
		pictures: map[string]*image.NRGBA{},
		dlgIcons: map[game.DialogKind]*image.NRGBA{},
	}

	dat, err := assets.FS.ReadFile("wine/sserife.fon")
	if err != nil {
		return nil, err
	}

	fonts, err := winfont.Parse(dat)
	if err != nil {
		return nil, err
	}

	res.sans = bitmapFace{fnt: fonts[0]}
	res.bold = bitmapFace{fnt: fonts[0], bold: true}

	for st, name := range []string{"Regular", "Bold", "Italic"} {
		dat, err := assets.FS.ReadFile("liberation/LiberationSans-" + name + ".ttf")
		if err != nil {
			return nil, err
		}

		if res.ttf[st], err = opentype.Parse(dat); err != nil {
			return nil, fmt.Errorf("liberation sans %s: %w", name, err)
		}
	}

	// GetAveCharSize: the width of A–Z and a–z, over 52.
	var az strings.Builder
	for chr := 'A'; chr <= 'Z'; chr++ {
		az.WriteRune(chr)
	}

	for chr := 'a'; chr <= 'z'; chr++ {
		az.WriteRune(chr)
	}

	res.charWidth = res.sans.width(az.String()) / 52

	for _, lst := range []struct {
		names []string
		path  string
		size  int
	}{
		{[]string{"PKMNImages", "Form4.PKMNImages"}, "sprites/front64.png", 64},
		{[]string{"SmallPKMNImg", "Form4.SmallPKMNImg"}, "sprites/icons32.png", 32},
		{[]string{"ImageList1"}, "sprites/small16.png", 16},
	} {
		img, err := loadPNG(lst.path)
		if err != nil {
			return nil, err
		}

		il := &imageList{img: img, size: lst.size, cols: img.Rect.Dx() / lst.size}
		for _, nam := range lst.names {
			res.lists[nam] = il
		}
	}

	for _, pth := range []string{"sprites/pokeball.png", "sprites/icon.png"} {
		if res.pictures[pth], err = loadPNG(pth); err != nil {
			return nil, err
		}
	}

	for kind, pth := range map[game.DialogKind]string{
		game.Warning:      "wine/oic_bang.png",
		game.Error:        "wine/oic_hand.png",
		game.Information:  "wine/oic_note.png",
		game.Confirmation: "wine/oic_ques.png",
	} {
		if res.dlgIcons[kind], err = loadPNG(pth); err != nil {
			return nil, err
		}
	}

	res.appIcon = shrink(res.pictures["sprites/icon.png"])

	return res, nil
}

func loadPNG(path string) (*image.NRGBA, error) {
	dat, err := assets.FS.ReadFile(path)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(dat))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	out := image.NewNRGBA(img.Bounds())
	draw.Draw(out, out.Rect, img, img.Bounds().Min, draw.Src)

	return out, nil
}

// shrink halves an icon, keeping every other pixel, as Windows draws a
// 32×32 icon in a caption when there is no 16×16 one.
func shrink(src *image.NRGBA) *image.NRGBA {
	rect := src.Rect
	out := image.NewNRGBA(image.Rect(0, 0, rect.Dx()/2, rect.Dy()/2))

	for y := range out.Rect.Dy() {
		for x := range out.Rect.Dx() {
			out.SetNRGBA(x, y, src.NRGBAAt(rect.Min.X+2*x, rect.Min.Y+2*y))
		}
	}

	return out
}

// fontFor returns the face a component's own font asks for, or nil to
// use its parent's: the forms only change the font to Arial.
func (r *resources) fontFor(props map[string]string) face {
	if props["Font.Name"] != "Arial" {
		return nil
	}

	hgt, err := strconv.Atoi(props["Font.Height"])
	if err != nil {
		return nil
	}

	return r.arial(-hgt, plainStyle)
}

// The view's icons and the look-in list's drive: o outline, y yellow, w
// white, k black, s shadow, c face, g green.
const (
	iconFolder string = `
................
................
..ooooo.........
.oyyyyyo........
oyyyyyyyoooooo..
owwwwwwwwwwwwwk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
okkkkkkkkkkkkkk.
................
................
................
`
	iconDocument string = `
..ssssssss......
..swwwwwwss.....
..swwwwwwsws....
..swwwwwwskks...
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..sssssssssskk..
...kkkkkkkkkkk..
................
`
	iconDrive string = `
................
................
................
................
................
..kkkkkkkkkkkkk.
.kwwwwwwwwwwwwsk
kwcccccccccccssk
kssssssssssssssk
kwccccccccccccsk
kwccccccccggccsk
kwccccccccccccsk
kssssssssssssssk
.kkkkkkkkkkkkkk.
................
................
`
)

var iconColors = map[byte]color.RGBA{
	'o': rgb(0x808000),
	'y': rgb(0xFFFF00),
	'w': rgb(0xFFFFFF),
	'k': rgb(0x000000),
	's': rgb(0x808080),
	'c': rgb(0xC0C0C0),
	'g': rgb(0x00FF00),
}

// The file icons' frames.
const (
	fileIconFolder int = iota
	fileIconDocument
	fileIconDrive
)

// fileIcons returns the file dialog's image list: a folder, a document and
// a drive.
func (r *resources) fileIcons() *imageList {
	if il := r.lists["fileIcons"]; il != nil {
		return il
	}

	img := image.NewRGBA(image.Rect(0, 0, 48, 16))
	cv := newCanvas(img)
	cv.pattern(0, 0, iconFolder, iconColors)
	cv.pattern(16, 0, iconDocument, iconColors)
	cv.pattern(32, 0, iconDrive, iconColors)

	nImg := image.NewNRGBA(img.Rect)
	copy(nImg.Pix, img.Pix) // the pixels are opaque or fully transparent

	il := &imageList{
		img:  nImg,
		size: 16,
		cols: 3,
	}
	r.lists["fileIcons"] = il

	return il
}

// arial returns Liberation Sans, standing in for Arial, at an em height of
// px pixels: a VCL font with Height = -px.
func (r *resources) arial(px int, style fontStyle) face {
	key := arialKey{px, style}
	if fnt, ok := r.arials[key]; ok {
		return fnt
	}

	ttf, err := newTrueTypeFace(r.ttf[style], px)
	if err != nil { // a size no font has
		return r.sans
	}

	r.arials[key] = ttf

	return ttf
}

func (r *resources) imageList(name string) *imageList {
	return r.lists[name]
}

func (r *resources) picture(path string) *image.NRGBA {
	return r.pictures[path]
}

// fontStyle is a style of Liberation Sans.
type fontStyle int

const (
	plainStyle fontStyle = iota
	boldStyle
	italicStyle
)

type arialKey struct {
	px    int
	style fontStyle
}

// imageList is a TImageList: square frames in an atlas, row by row.
type imageList struct {
	img  *image.NRGBA
	size int
	cols int
}

func (il *imageList) frame(i int) image.Rectangle {
	x, y := i%il.cols*il.size, i/il.cols*il.size

	return image.Rect(x, y, x+il.size, y+il.size)
}

// drawFrame draws frame i at (x, y). A selected frame is dithered with the
// selection colour, as ILD_SELECTED draws on a 256-colour screen.
func (il *imageList) drawFrame(cv canvas, x, y, i int, selected bool) {
	if i < 0 || i >= il.cols*(il.img.Rect.Dy()/il.size) {
		return
	}

	fr := il.frame(i)
	cv.draw(x, y, il.img, fr)

	if !selected {
		return
	}

	for sy := fr.Min.Y; sy < fr.Max.Y; sy++ {
		for sx := fr.Min.X; sx < fr.Max.X; sx++ {
			if il.img.NRGBAAt(sx, sy).A >= 0x80 && (sx+sy)&1 == 1 {
				cv.pix(x+sx-fr.Min.X, y+sy-fr.Min.Y, colSelection)
			}
		}
	}
}

// drawDisabled draws frame i as an image list draws a disabled image: its
// shape embossed in the highlight and shadow colours.
func (il *imageList) drawDisabled(cv canvas, x, y, i int) {
	fr := il.frame(i)
	for _, pass := range []struct {
		d int
		c color.RGBA
	}{{1, colHighlight}, {0, colShadow}} {
		for sy := fr.Min.Y; sy < fr.Max.Y; sy++ {
			for sx := fr.Min.X; sx < fr.Max.X; sx++ {
				clr := il.img.NRGBAAt(sx, sy)
				// Dark pixels make the shape, as the mask of a disabled
				// image keeps them.
				if clr.A >= 0x80 && int(clr.R)+int(clr.G)+int(clr.B) < 3*0xC0 {
					cv.pix(x+sx-fr.Min.X+pass.d, y+sy-fr.Min.Y+pass.d, pass.c)
				}
			}
		}
	}
}
