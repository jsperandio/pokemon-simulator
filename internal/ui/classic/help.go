// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/help"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// The manual. The original opened its WinHelp file, POKéGAME.HLP, at the
// "Index" topic (Application.HelpJump). The port shows the same pages,
// converted to Markdown, in a window of its own after WinHelp's: its
// buttons, then the topic's non-scrolling region, red titles on the yellow
// the help file's window definition asks for, then the topic, which
// scrolls. The text is Arial, as the help file's: 10 pt, with the banner
// at 18 pt and the titles at 14 pt, all bold and red (\cf1). Links are
// WinHelp's green, underlined hotspots.

// The help file's window: (50, 6, 924, 960) in WinHelp's units, 1024ths of
// the screen.
var (
	helpPos  = image.Pt(mulDiv(50, screenWidth, 1024), mulDiv(6, screenHeight, 1024))
	helpSize = image.Pt(mulDiv(924, screenWidth, 1024), mulDiv(960, screenHeight, 1024))
)

const (
	helpBannerText string = "POKéMON Simulator" // the first line of every topic

	helpTextPx   int = 13 // Arial 10 pt, at 96 dpi
	helpBannerPx int = 24 // 18 pt
	helpTitlePx  int = 19 // 14 pt

	helpBarHeight int = 29 // the buttons' bar
	helpMargin    int = 6  // the topic's, left and right
	helpIndent    int = 18 // a list item's text, from the margin
	helpPad       int = 4  // inside a port note's box and a table's cells
)

var (
	helpYellow = rgb(0xFFFF00) // the non-scrolling region: (255, 255, 0)
	helpRed    = rgb(0xFF0000)
	helpGreen  = rgb(0x008000)
)

// helpPlace is a page, and how far down it was scrolled.
type helpPlace struct {
	page string
	pos  int
}

// helpWindow is the manual's window.
type helpWindow struct {
	app      *Application
	wdw      *window
	view     *helpView
	back     *button
	history  []helpPlace
	pages    map[string]*help.Page
	pictures map[string]*image.RGBA
	openURL  func(url string) error
}

// arial returns the manual's Arial at an em height of px pixels.
func (hw *helpWindow) arial(px int, style fontStyle) face {
	return hw.app.res.arial(px, style)
}

func newHelpWindow(app *Application) *helpWindow {
	hw := &helpWindow{
		app:      app,
		pages:    map[string]*help.Page{},
		pictures: map[string]*image.RGBA{},
		openURL:  platform.OpenURL,
	}

	const frame int = 4 // sizeable
	comp := layout.Component{
		Class:   "THelpWindow",
		Name:    "Help",
		Caption: helpBannerText,
		Bounds:  layout.Rect{X: helpPos.X, Y: helpPos.Y},
		Props: map[string]string{
			"ClientWidth":  strconv.Itoa(helpSize.X - 2*frame),
			"ClientHeight": strconv.Itoa(helpSize.Y - 2*frame - captionHeight),
		},
		Children: []layout.Component{
			{
				Class:   "TButton",
				Name:    "Contents",
				Caption: "&Contents",
				Bounds:  layout.Rect{X: 4, Y: 3, W: 75, H: 23},
				Props:   map[string]string{"TabStop": "False"},
			},
			{
				Class:   "TButton",
				Name:    "Back",
				Caption: "&Back",
				Bounds:  layout.Rect{X: 81, Y: 3, W: 75, H: 23},
				Props:   map[string]string{"TabStop": "False"},
			},
		},
	}
	wdw := newWindow(app.desk, &comp)
	hw.wdw = wdw
	wdw.icon = helpIcon()
	view := &helpView{
		hw:    hw,
		sel:   -1,
		press: -1,
	}
	hw.view = view
	addWidget(wdw, view, image.Rect(0, helpBarHeight, wdw.client.X, wdw.client.Y))

	view.windowed, view.tabStop, view.color = true, true, colWindow
	view.scroll = scrollBar{
		vertical: true,
		line:     app.res.arial(helpTextPx, plainStyle).height(),
	}
	view.scroll.onScroll = view.invalidate
	wdw.focus = view

	// The buttons leave the focus in the topic, as WinHelp's do.
	get[*button](wdw, "Contents").onClick = func() {
		hw.jump(help.Contents)
		wdw.setFocus(view, false)
	}
	hw.back = get[*button](wdw, "Back")
	hw.back.onClick = func() {
		hw.goBack()
		wdw.setFocus(view, false)
	}
	wdw.onResize = func() {
		view.setBounds(image.Rect(0, helpBarHeight, wdw.client.X, wdw.client.Y))
	}
	wdw.onClose = func() {
		app.desk.hide(wdw)

		hw.history = nil
		view.name = ""
	}
	app.desk.add(wdw)

	return hw
}

// jump shows a page, from its top; the page shown before goes in the
// history.
func (hw *helpWindow) jump(page string) {
	hv := hw.view
	if hw.wdw.visible && hv.name != "" && hv.name != page {
		hw.history = append(hw.history, helpPlace{hv.name, hv.scroll.pos})
	}

	hv.open(page, 0)
	hw.back.setEnabled(len(hw.history) > 0)
}

// goBack is the Back button: the page shown before, where it was.
func (hw *helpWindow) goBack() {
	n := len(hw.history)
	if n == 0 {
		return
	}

	hpl := hw.history[n-1]
	hw.history = hw.history[:n-1]
	hw.view.open(hpl.page, hpl.pos)
	hw.back.setEnabled(n > 1)
}

// page returns a page of the manual, read once.
func (hw *helpWindow) page(name string) *help.Page {
	if pag, ok := hw.pages[name]; ok {
		return pag
	}

	var pag *help.Page
	if dat, err := assets.FS.ReadFile(path.Join("help", name)); err == nil {
		pag = help.Parse(dat)
	}

	hw.pages[name] = pag

	return pag
}

// picture returns a picture of the manual, decoded once.
func (hw *helpWindow) picture(name string) *image.RGBA {
	if img, ok := hw.pictures[name]; ok {
		return img
	}

	var img *image.RGBA

	if dat, err := assets.FS.ReadFile(path.Join("help", name)); err == nil {
		if src, err := png.Decode(bytes.NewReader(dat)); err == nil {
			rect := src.Bounds()
			img = image.NewRGBA(image.Rectangle{Max: rect.Size()})
			draw.Draw(img, img.Rect, src, rect.Min, draw.Src)
		}
	}

	hw.pictures[name] = img

	return img
}

// layoutHead draws a page's non-scrolling region: the banner, then the
// page's title, centred, and the line under them.
func (hw *helpWindow) layoutHead(pag *help.Page, width int) *image.RGBA {
	type line struct {
		f    face
		text string
	}
	lines := []line{{hw.arial(helpBannerPx, boldStyle), helpBannerText}}

	if pag.Title != "" && pag.Title != helpBannerText {
		lines = append(lines, line{hw.arial(helpTitlePx, boldStyle), pag.Title})
	}

	height := 2 + 4 + 1
	for _, lin := range lines {
		height += lin.f.height()
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	cv := newCanvas(img)
	cv.fill(img.Rect, helpYellow)

	y := 2
	for _, lin := range lines {
		lin.f.draw(cv, (width-lin.f.width(lin.text))/2, y, lin.text, helpRed)

		y += lin.f.height()
	}

	cv.hline(0, width, height-1, colText)

	return img
}

// layoutTopic lays a page's topic out at a width and draws it.
func (hw *helpWindow) layoutTopic(pag *help.Page, width int) (*image.RGBA, []helpSpot) {
	hf := &helpFlow{hw: hw}
	gap := hf.face(false, false).height() // a blank line
	y := hf.blocks(pag.Blocks, helpMargin, width-helpMargin, gap) + gap
	img := image.NewRGBA(image.Rect(0, 0, width, y))
	cv := newCanvas(img)
	cv.fill(img.Rect, colWindow)

	for _, op := range hf.ops {
		switch {
		case op.img != nil:
			cv.copy(op.rect.Min.X, op.rect.Min.Y, op.img, op.img.Rect)
		case op.fnt != nil:
			op.fnt.draw(cv, op.rect.Min.X, op.rect.Min.Y, op.text, op.clr)
		case op.frame:
			cv.frame(op.rect, op.clr)
		default:
			cv.fill(op.rect, op.clr)
		}
	}

	return img, hf.spots
}

// helpSpot is a hotspot: a link, and the boxes its words take in the
// topic.
type helpSpot struct {
	target string
	boxes  []image.Rectangle
}

// helpView shows a page: its non-scrolling region, with the banner and the
// title, then the topic, which scrolls.
type helpView struct {
	control
	hw     *helpWindow
	name   string // the page's file
	page   *help.Page
	laid   image.Point // the size the page was laid out for
	head   *image.RGBA // the non-scrolling region, its line under it included
	doc    *image.RGBA // the topic
	spots  []helpSpot
	scroll scrollBar
	sel    int // the hotspot Tab chose, or -1
	press  int // the hotspot the mouse pressed, or -1
}

// open shows a page, scrolled down to pos.
func (hv *helpView) open(name string, pos int) {
	pag := hv.hw.page(name)
	if pag == nil {
		return
	}

	hv.name, hv.page = name, pag
	hv.sel, hv.press = -1, -1
	hv.layout()

	hv.scroll.pos = 0
	hv.scroll.set(pos)
	hv.invalidate()
}

// layout lays the page out for the view's size: across the whole width if
// the topic fits, beside a scroll bar if not.
func (hv *helpView) layout() {
	rect := hv.local()
	hv.laid = rect.Size()

	if hv.page == nil {
		return
	}

	hv.head = hv.hw.layoutHead(hv.page, rect.Dx())
	hh := hv.head.Rect.Dy()
	hv.doc, hv.spots = hv.hw.layoutTopic(hv.page, rect.Dx())

	if hv.doc.Rect.Dy() > rect.Dy()-hh {
		hv.doc, hv.spots = hv.hw.layoutTopic(hv.page, rect.Dx()-scrollSize)
	}

	sb := &hv.scroll
	sb.rect = image.Rect(rect.Max.X-scrollSize, hh, rect.Max.X, rect.Max.Y)
	sb.min, sb.max = 0, hv.doc.Rect.Dy()-1
	sb.page = max(rect.Dy()-hh, 1)
	sb.set(sb.pos)
}

func (hv *helpView) paint(cv canvas) {
	rect := hv.local()
	if hv.laid != rect.Size() {
		hv.layout()
	}

	if hv.head == nil {
		cv.fill(rect, colWindow)

		return
	}

	hh := hv.head.Rect.Dy()
	cv.copy(0, 0, hv.head, hv.head.Rect)
	cv.fill(image.Rect(0, hh, rect.Dx(), rect.Dy()), colWindow)
	cv.copy(0, hh, hv.doc, image.Rect(0, hv.scroll.pos, hv.doc.Rect.Dx(), hv.scroll.pos+rect.Dy()-hh))

	if hv.sel >= 0 {
		in := cv.sub(image.Rect(0, hh, hv.doc.Rect.Dx(), rect.Dy()))
		for _, box := range hv.spots[hv.sel].boxes {
			box = box.Sub(image.Pt(0, hv.scroll.pos))
			for y := box.Min.Y; y < box.Max.Y; y++ {
				for x := box.Min.X; x < box.Max.X; x++ {
					in.invert(x, y)
				}
			}
		}
	}

	if hv.scroll.enabled() {
		hv.scroll.paint(cv)
	}
}

func (hv *helpView) focusable() bool {
	return true
}

// spotAt returns the hotspot at pnt, in the view's coordinates, or -1.
func (hv *helpView) spotAt(pnt image.Point) int {
	if hv.head == nil || pnt.Y < hv.head.Rect.Dy() || pnt.X >= hv.doc.Rect.Dx() {
		return -1
	}

	loc := pnt.Add(image.Pt(0, hv.scroll.pos-hv.head.Rect.Dy()))
	for i, hsp := range hv.spots {
		if slices.ContainsFunc(hsp.boxes, loc.In) {
			return i
		}
	}

	return -1
}

func (hv *helpView) mouseDown(pnt image.Point, _ bool) {
	if hv.scroll.enabled() && hv.scroll.press(pnt) {
		hv.invalidate()

		return
	}

	hv.press = hv.spotAt(pnt)
}

func (hv *helpView) mouseDrag(pnt image.Point) {
	if hv.scroll.pressed != partNone {
		hv.scroll.drag(pnt)
		hv.invalidate()
	}
}

func (hv *helpView) mouseUp(pnt image.Point) {
	if hv.scroll.pressed != partNone {
		hv.scroll.release()
		hv.invalidate()

		return
	}

	idx := hv.press
	hv.press = -1

	if idx >= 0 && hv.spotAt(pnt) == idx {
		hv.follow(idx)
	}
}

func (hv *helpView) tick(now time.Time) {
	if hv.scroll.tick(now) {
		hv.invalidate()
	}
}

func (hv *helpView) cursor(pnt image.Point) ebiten.CursorShapeType {
	if hv.spotAt(pnt) >= 0 {
		return ebiten.CursorShapePointer
	}

	return ebiten.CursorShapeDefault
}

func (hv *helpView) wheel(dy int) {
	hv.scroll.set(hv.scroll.pos - 3*dy*hv.scroll.line)
	hv.invalidate()
}

func (hv *helpView) key(key ebiten.Key, mds mods) bool {
	sb := &hv.scroll

	switch key {
	case ebiten.KeyUp:
		sb.set(sb.pos - sb.line)
	case ebiten.KeyDown:
		sb.set(sb.pos + sb.line)
	case ebiten.KeyPageUp:
		sb.set(sb.pos - sb.page)
	case ebiten.KeyPageDown:
		sb.set(sb.pos + sb.page)
	case ebiten.KeyHome:
		sb.set(0)
	case ebiten.KeyEnd:
		sb.set(sb.limit())
	case ebiten.KeyTab:
		hv.choose(mds.shift)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if hv.sel < 0 {
			return false
		}

		hv.follow(hv.sel)
	case ebiten.KeyBackspace:
		hv.hw.goBack()
	default:
		return false
	}

	hv.invalidate()

	return true
}

func (hv *helpView) char(rune) {}

// choose selects the next hotspot, or the one before, and scrolls it into
// view, as Tab does in WinHelp.
func (hv *helpView) choose(back bool) {
	n := len(hv.spots)
	if n == 0 {
		return
	}

	switch {
	case hv.sel < 0 && back:
		hv.sel = n - 1
	case hv.sel < 0:
		hv.sel = 0
	case back:
		hv.sel = (hv.sel + n - 1) % n
	default:
		hv.sel = (hv.sel + 1) % n
	}

	if boxes := hv.spots[hv.sel].boxes; len(boxes) > 0 {
		rect, sb := boxes[0], &hv.scroll
		switch {
		case rect.Min.Y < sb.pos:
			sb.set(rect.Min.Y)
		case rect.Max.Y > sb.pos+sb.page:
			sb.set(rect.Max.Y - sb.page)
		}
	}
}

// follow follows a hotspot: to another page, or out to the browser or the
// e-mail program.
func (hv *helpView) follow(idx int) {
	tgt := hv.spots[idx].target
	for _, scheme := range []string{"http:", "https:", "mailto:"} {
		if strings.HasPrefix(tgt, scheme) {
			if err := hv.hw.openURL(tgt); err != nil {
				app := hv.hw.app
				app.call(func() {
					app.Message(game.Error, false, err.Error())
				})
			}

			return
		}
	}

	hv.hw.jump(tgt)
}

// helpOp is something to draw: text, a picture, or a box, filled or
// outlined.
type helpOp struct {
	rect  image.Rectangle
	clr   color.RGBA
	text  string
	fnt   face
	img   *image.RGBA
	frame bool
}

// helpPiece is text in one face, or a picture, that the flow places.
type helpPiece struct {
	text string
	fnt  face
	clr  color.RGBA
	img  *image.RGBA
	spot int // its hotspot, or -1
	wid  int
}

// helpWord is pieces with no space between them, which stay on one line,
// and the space after them, if any.
type helpWord struct {
	pieces []helpPiece
	wid    int
	space  helpPiece
	brk    bool // a line break after it
}

// helpFlow lays a topic out: what to draw where, and the hotspots.
type helpFlow struct {
	hw    *helpWindow
	ops   []helpOp
	spots []helpSpot
}

// face returns the text's font in a style; the manual has no bold italics.
func (hf *helpFlow) face(bold, italic bool) face {
	st := plainStyle

	switch {
	case bold:
		st = boldStyle
	case italic:
		st = italicStyle
	}

	return hf.hw.arial(helpTextPx, st)
}

// blocks lays blocks out between x0 and x1 from y, a blank line apart, and
// returns the y under them.
func (hf *helpFlow) blocks(bs []help.Block, x0, x1, y int) int {
	text := hf.face(false, false)

	for i, blk := range bs {
		if i > 0 {
			y += text.height()
		}

		switch blk.Kind {
		case help.Paragraph, help.Heading:
			y, _ = hf.flow(blk.Spans, x0, x1, y, blk.Kind == help.Heading)
		case help.Quote:
			// A port note, in a box of the tooltips' colours.
			box := len(hf.ops)
			hf.ops = append(hf.ops,
				helpOp{clr: colInfo},
				helpOp{
					clr:   colText,
					frame: true,
				},
			)
			bottom := hf.blocks(blk.Blocks, x0+1+helpPad, x1-1-helpPad, y+1+helpPad) + helpPad + 1
			hf.ops[box].rect = image.Rect(x0, y, x1, bottom)
			hf.ops[box+1].rect = hf.ops[box].rect
			y = bottom
		case help.List:
			for _, it := range blk.Items {
				var base int
				y, base = hf.flow(it.Spans, x0+helpIndent, x1, y, false)
				marker, x := "•", x0+4

				if it.Marker != "" {
					marker, x = it.Marker, x0
				}

				hf.ops = append(hf.ops, helpOp{
					rect: image.Rect(x, base-text.ascent(), x, base),
					text: marker,
					fnt:  text,
					clr:  colText,
				})
			}
		case help.Table:
			y = hf.table(blk.Rows, x0, y)
		}
	}

	return y
}

// table lays a table out from x0 at y, each cell's text on one line, the
// header row's in bold, and returns the y under it.
func (hf *helpFlow) table(rows [][]help.Cell, x0, y int) int {
	if len(rows) == 0 {
		return y
	}

	widths := make([]int, len(rows[0]))
	for rix, row := range rows {
		for cix, cell := range row {
			widths[cix] = max(widths[cix], hf.measure(cell, rix == 0))
		}
	}

	rowH := hf.face(false, false).height() + 4

	for rix, row := range rows {
		x := x0

		for cix, wid := range widths {
			cw := wid + 2*helpPad
			hf.ops = append(hf.ops, helpOp{
				rect:  image.Rect(x, y, x+cw+1, y+rowH+1),
				clr:   colShadow,
				frame: true,
			})

			if cix < len(row) {
				hf.flow(row[cix], x+1+helpPad, x+1+helpPad+wid, y+3, rix == 0)
			}

			x += cw
		}

		y += rowH
	}

	return y + 1
}

// measure returns the width of text on one line.
func (hf *helpFlow) measure(spans []help.Span, bold bool) int {
	wid := 0

	for _, spn := range spans {
		if spn.Image != "" {
			if img := hf.hw.picture(spn.Image); img != nil {
				wid += img.Rect.Dx()
			}

			continue
		}

		wid += hf.face(bold || spn.Bold, spn.Italic).width(spn.Text)
	}

	return wid
}

// flow lays spans out from y in lines between x0 and x1, breaking them at
// spaces, and returns the y under them and the first line's baseline.
// Bold sets all of them in bold.
func (hf *helpFlow) flow(spans []help.Span, x0, x1, y int, bold bool) (bottom, baseline int) {
	words := hf.words(spans, bold)
	first := -1
	line := make([]helpWord, 0, len(words))
	wid := 0

	for _, wd := range words {
		if n := len(line); n > 0 && wid+line[n-1].space.wid+wd.wid > x1-x0 {
			y = hf.place(line, x0, y, &first)
			line, wid = nil, 0
		}

		if n := len(line); n > 0 {
			wid += line[n-1].space.wid
		}

		line = append(line, wd)
		wid += wd.wid

		if wd.brk {
			y = hf.place(line, x0, y, &first)
			line, wid = nil, 0
		}
	}

	if len(line) > 0 || first < 0 {
		y = hf.place(line, x0, y, &first)
	}

	return y, first
}

// words splits spans into words, and gives each link a hotspot.
func (hf *helpFlow) words(spans []help.Span, bold bool) []helpWord {
	var wb wordBuilder

	link, spot := "", -1

	for _, spn := range spans {
		if spn.Link != link {
			link, spot = spn.Link, -1
			if link != "" {
				hf.spots = append(hf.spots, helpSpot{target: link})
				spot = len(hf.spots) - 1
			}
		}

		switch {
		case spn.Break:
			wb.end(helpPiece{}, true)
		case spn.Image != "":
			if img := hf.hw.picture(spn.Image); img != nil {
				wb.add(helpPiece{
					img:  img,
					spot: spot,
					wid:  img.Rect.Dx(),
				})
			}
		default:
			fnt := hf.face(bold || spn.Bold, spn.Italic)
			clr := colText

			if spn.Link != "" {
				clr = helpGreen
			}

			for i, part := range strings.Split(spn.Text, " ") {
				if i > 0 {
					wb.end(helpPiece{
						fnt:  fnt,
						clr:  clr,
						spot: spot,
						wid:  fnt.width(" "),
					}, false)
				}

				if part != "" {
					wb.add(helpPiece{
						text: part,
						fnt:  fnt,
						clr:  clr,
						spot: spot,
						wid:  fnt.width(part),
					})
				}
			}
		}
	}

	wb.end(helpPiece{}, false)

	return wb.words
}

// place puts a line's words on it from x0, the line's top at y, each piece
// on the baseline, and returns the y under the line.
func (hf *helpFlow) place(line []helpWord, x0, y int, first *int) int {
	asc, desc := 0, 0

	for _, wd := range line {
		for _, hpc := range wd.pieces {
			if hpc.img != nil {
				asc = max(asc, hpc.img.Rect.Dy())

				continue
			}

			asc, desc = max(asc, hpc.fnt.ascent()), max(desc, hpc.fnt.height()-hpc.fnt.ascent())
		}
	}

	if asc == 0 {
		text := hf.face(false, false)
		asc, desc = text.ascent(), text.height()-text.ascent()
	}

	base := y + asc
	bottom := base + desc

	if *first < 0 {
		*first = base
	}

	x := x0

	for i, wd := range line {
		for _, hpc := range wd.pieces {
			hf.put(hpc, x, y, base, bottom)

			x += hpc.wid
		}

		if i < len(line)-1 {
			hf.put(wd.space, x, y, base, bottom)

			x += wd.space.wid
		}
	}

	return bottom
}

// put places a piece at x on a line from top to bottom: text and pictures
// on the baseline, and for a link, its underline and its hotspot's box.
func (hf *helpFlow) put(hpc helpPiece, x, top, base, bottom int) {
	switch {
	case hpc.img != nil:
		hf.ops = append(hf.ops, helpOp{
			rect: image.Rect(x, base-hpc.img.Rect.Dy(), x+hpc.wid, base),
			img:  hpc.img,
		})
	case hpc.text != "":
		hf.ops = append(hf.ops, helpOp{
			rect: image.Rect(x, base-hpc.fnt.ascent(), x+hpc.wid, base),
			text: hpc.text,
			fnt:  hpc.fnt,
			clr:  hpc.clr,
		})
	}

	if hpc.spot < 0 || hpc.wid == 0 {
		return
	}

	if hpc.img == nil {
		hf.ops = append(hf.ops, helpOp{
			rect: image.Rect(x, base+1, x+hpc.wid, base+2),
			clr:  hpc.clr,
		})
	}

	hsp := &hf.spots[hpc.spot]
	box := image.Rect(x, top, x+hpc.wid, bottom)

	if n := len(hsp.boxes); n > 0 && hsp.boxes[n-1].Max.X == x && hsp.boxes[n-1].Min.Y == top {
		hsp.boxes[n-1].Max.X = box.Max.X

		return
	}

	hsp.boxes = append(hsp.boxes, box)
}

// wordBuilder builds words out of pieces.
type wordBuilder struct {
	words []helpWord
	cur   helpWord
}

// add adds a piece to the current word.
func (wb *wordBuilder) add(hpc helpPiece) {
	wb.cur.pieces = append(wb.cur.pieces, hpc)
	wb.cur.wid += hpc.wid
}

// end ends the current word with a space, or a line break after it.
func (wb *wordBuilder) end(space helpPiece, brk bool) {
	if len(wb.cur.pieces) == 0 {
		// Spaces in a row count as one.
		if n := len(wb.words); n > 0 {
			wb.words[n-1].brk = wb.words[n-1].brk || brk
			if wb.words[n-1].space.wid == 0 {
				wb.words[n-1].space = space
			}
		}

		return
	}

	wb.cur.space, wb.cur.brk = space, brk
	wb.words = append(wb.words, wb.cur)
	wb.cur = helpWord{}
}

// helpIcon is the help window's icon: a book with a question mark.
func helpIcon() *image.NRGBA {
	const rows string = `
................
...kkkkkkkkkkk..
..kbbbbbbbbbbbk.
..kbbbbyyyybbbk.
..kbbbyybbyybbk.
..kbbbbbbbyybbk.
..kbbbbbbyybbbk.
..kbbbbbyybbbbk.
..kbbbbbyybbbbk.
..kbbbbbbbbbbbk.
..kbbbbbyybbbbk.
..kbbbbbbbbbbbk.
..kkkkkkkkkkkkk.
..kwwwwwwwwwwwk.
...kkkkkkkkkkkk.
................
`
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	newCanvas(img).pattern(0, 0, rows, map[byte]color.RGBA{
		'k': colText,
		'b': colSelection,
		'y': helpYellow,
		'w': colWindow,
	})

	icon := image.NewNRGBA(img.Rect)
	copy(icon.Pix, img.Pix) // the pixels are opaque or fully transparent

	return icon
}
