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
	r := &resources{
		arials:   map[arialKey]face{},
		lists:    map[string]*imageList{},
		pictures: map[string]*image.NRGBA{},
		dlgIcons: map[game.DialogKind]*image.NRGBA{},
	}

	b, err := assets.FS.ReadFile("wine/sserife.fon")
	if err != nil {
		return nil, err
	}

	fonts, err := winfont.Parse(b)
	if err != nil {
		return nil, err
	}

	r.sans = bitmapFace{f: fonts[0]}
	r.bold = bitmapFace{f: fonts[0], bold: true}

	for st, name := range []string{"Regular", "Bold", "Italic"} {
		b, err := assets.FS.ReadFile("liberation/LiberationSans-" + name + ".ttf")
		if err != nil {
			return nil, err
		}

		if r.ttf[st], err = opentype.Parse(b); err != nil {
			return nil, fmt.Errorf("liberation sans %s: %w", name, err)
		}
	}

	// GetAveCharSize: the width of A–Z and a–z, over 52.
	var az strings.Builder
	for c := 'A'; c <= 'Z'; c++ {
		az.WriteRune(c)
	}

	for c := 'a'; c <= 'z'; c++ {
		az.WriteRune(c)
	}

	r.charWidth = r.sans.width(az.String()) / 52

	for _, l := range []struct {
		names []string
		path  string
		size  int
	}{
		{[]string{"PKMNImages", "Form4.PKMNImages"}, "sprites/front64.png", 64},
		{[]string{"SmallPKMNImg", "Form4.SmallPKMNImg"}, "sprites/icons32.png", 32},
		{[]string{"ImageList1"}, "sprites/small16.png", 16},
	} {
		img, err := loadPNG(l.path)
		if err != nil {
			return nil, err
		}

		il := &imageList{img: img, size: l.size, cols: img.Rect.Dx() / l.size}
		for _, n := range l.names {
			r.lists[n] = il
		}
	}

	for _, p := range []string{"sprites/pokeball.png", "sprites/icon.png"} {
		if r.pictures[p], err = loadPNG(p); err != nil {
			return nil, err
		}
	}

	for kind, p := range map[game.DialogKind]string{
		game.Warning:      "wine/oic_bang.png",
		game.Error:        "wine/oic_hand.png",
		game.Information:  "wine/oic_note.png",
		game.Confirmation: "wine/oic_ques.png",
	} {
		if r.dlgIcons[kind], err = loadPNG(p); err != nil {
			return nil, err
		}
	}

	r.appIcon = shrink(r.pictures["sprites/icon.png"])

	return r, nil
}

func loadPNG(path string) (*image.NRGBA, error) {
	b, err := assets.FS.ReadFile(path)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(b))
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
	b := src.Rect
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()/2, b.Dy()/2))

	for y := range out.Rect.Dy() {
		for x := range out.Rect.Dx() {
			out.SetNRGBA(x, y, src.NRGBAAt(b.Min.X+2*x, b.Min.Y+2*y))
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

	h, err := strconv.Atoi(props["Font.Height"])
	if err != nil {
		return nil
	}

	return r.arial(-h, plainStyle)
}

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

// arial returns Liberation Sans, standing in for Arial, at an em height of
// px pixels: a VCL font with Height = -px.
func (r *resources) arial(px int, style fontStyle) face {
	k := arialKey{px, style}
	if f, ok := r.arials[k]; ok {
		return f
	}

	f, err := newTrueType(r.ttf[style], px)
	if err != nil { // a size no font has
		return r.sans
	}

	r.arials[k] = f

	return f
}

func (r *resources) imageList(name string) *imageList {
	return r.lists[name]
}

func (r *resources) picture(path string) *image.NRGBA {
	return r.pictures[path]
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
				c := il.img.NRGBAAt(sx, sy)
				// Dark pixels make the shape, as the mask of a disabled
				// image keeps them.
				if c.A >= 0x80 && int(c.R)+int(c.G)+int(c.B) < 3*0xC0 {
					cv.pix(x+sx-fr.Min.X+pass.d, y+sy-fr.Min.Y+pass.d, pass.c)
				}
			}
		}
	}
}
