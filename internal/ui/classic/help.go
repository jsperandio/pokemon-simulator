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
	a        *Application
	w        *window
	view     *helpView
	back     *button
	history  []helpPlace
	pages    map[string]*help.Page
	pictures map[string]*image.RGBA
	openURL  func(url string) error
}

// arial returns the manual's Arial at an em height of px pixels.
func (hw *helpWindow) arial(px int, style fontStyle) face {
	return hw.a.res.arial(px, style)
}

func newHelpWindow(a *Application) *helpWindow {
	h := &helpWindow{
		a:        a,
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
			"ClientWidth":  itoa(helpSize.X - 2*frame),
			"ClientHeight": itoa(helpSize.Y - 2*frame - captionHeight),
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
	w := newWindow(a.desk, &comp)
	h.w = w
	w.icon = helpIcon()
	view := &helpView{h: h, sel: -1, press: -1}
	h.view = view
	addWidget(w, view, image.Rect(0, helpBarHeight, w.client.X, w.client.Y))

	view.windowed, view.tabStop, view.color = true, true, colWindow
	view.scroll = scrollBar{vertical: true, line: a.res.arial(helpTextPx, plainStyle).height()}
	view.scroll.onScroll = view.invalidate
	w.focus = view

	// The buttons leave the focus in the topic, as WinHelp's do.
	get[*button](w, "Contents").onClick = func() {
		h.jump(help.Contents)
		w.setFocus(view, false)
	}
	h.back = get[*button](w, "Back")
	h.back.onClick = func() {
		h.goBack()
		w.setFocus(view, false)
	}
	w.onResize = func() {
		view.setBounds(image.Rect(0, helpBarHeight, w.client.X, w.client.Y))
	}
	w.onClose = func() {
		a.desk.hide(w)

		h.history = nil
		view.name = ""
	}
	a.desk.add(w)

	return h
}

// jump shows a page, from its top; the page shown before goes in the
// history.
func (hw *helpWindow) jump(page string) {
	v := hw.view
	if hw.w.visible && v.name != "" && v.name != page {
		hw.history = append(hw.history, helpPlace{v.name, v.scroll.pos})
	}

	v.open(page, 0)
	hw.back.setEnabled(len(hw.history) > 0)
}

// goBack is the Back button: the page shown before, where it was.
func (hw *helpWindow) goBack() {
	n := len(hw.history)
	if n == 0 {
		return
	}

	p := hw.history[n-1]
	hw.history = hw.history[:n-1]
	hw.view.open(p.page, p.pos)
	hw.back.setEnabled(n > 1)
}

// page returns a page of the manual, read once.
func (hw *helpWindow) page(name string) *help.Page {
	if p, ok := hw.pages[name]; ok {
		return p
	}

	var p *help.Page
	if b, err := assets.FS.ReadFile(path.Join("help", name)); err == nil {
		p = help.Parse(b)
	}

	hw.pages[name] = p

	return p
}

// picture returns a picture of the manual, decoded once.
func (hw *helpWindow) picture(name string) *image.RGBA {
	if img, ok := hw.pictures[name]; ok {
		return img
	}

	var img *image.RGBA

	if b, err := assets.FS.ReadFile(path.Join("help", name)); err == nil {
		if src, err := png.Decode(bytes.NewReader(b)); err == nil {
			r := src.Bounds()
			img = image.NewRGBA(image.Rectangle{Max: r.Size()})
			draw.Draw(img, img.Rect, src, r.Min, draw.Src)
		}
	}

	hw.pictures[name] = img

	return img
}

// layoutHead draws a page's non-scrolling region: the banner, then the
// page's title, centred, and the line under them.
func (hw *helpWindow) layoutHead(p *help.Page, width int) *image.RGBA {
	type line struct {
		f    face
		text string
	}
	lines := []line{{hw.arial(helpBannerPx, boldStyle), helpBannerText}}

	if p.Title != "" && p.Title != helpBannerText {
		lines = append(lines, line{hw.arial(helpTitlePx, boldStyle), p.Title})
	}

	height := 2 + 4 + 1
	for _, l := range lines {
		height += l.f.height()
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	cv := newCanvas(img)
	cv.fill(img.Rect, helpYellow)

	y := 2
	for _, l := range lines {
		l.f.draw(cv, (width-l.f.width(l.text))/2, y, l.text, helpRed)

		y += l.f.height()
	}

	cv.hline(0, width, height-1, colText)

	return img
}

// layoutTopic lays a page's topic out at a width and draws it.
func (hw *helpWindow) layoutTopic(p *help.Page, width int) (*image.RGBA, []helpSpot) {
	l := &helpFlow{h: hw}
	gap := l.face(false, false).height() // a blank line
	y := l.blocks(p.Blocks, helpMargin, width-helpMargin, gap) + gap
	img := image.NewRGBA(image.Rect(0, 0, width, y))
	cv := newCanvas(img)
	cv.fill(img.Rect, colWindow)

	for _, op := range l.ops {
		switch {
		case op.img != nil:
			cv.copy(op.r.Min.X, op.r.Min.Y, op.img, op.img.Rect)
		case op.f != nil:
			op.f.draw(cv, op.r.Min.X, op.r.Min.Y, op.text, op.c)
		case op.frame:
			cv.frame(op.r, op.c)
		default:
			cv.fill(op.r, op.c)
		}
	}

	return img, l.spots
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
	h      *helpWindow
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
	p := hv.h.page(name)
	if p == nil {
		return
	}

	hv.name, hv.page = name, p
	hv.sel, hv.press = -1, -1
	hv.layout()

	hv.scroll.pos = 0
	hv.scroll.set(pos)
	hv.invalidate()
}

// layout lays the page out for the view's size: across the whole width if
// the topic fits, beside a scroll bar if not.
func (hv *helpView) layout() {
	r := hv.local()
	hv.laid = r.Size()

	if hv.page == nil {
		return
	}

	hv.head = hv.h.layoutHead(hv.page, r.Dx())
	hh := hv.head.Rect.Dy()
	hv.doc, hv.spots = hv.h.layoutTopic(hv.page, r.Dx())

	if hv.doc.Rect.Dy() > r.Dy()-hh {
		hv.doc, hv.spots = hv.h.layoutTopic(hv.page, r.Dx()-scrollSize)
	}

	s := &hv.scroll
	s.rect = image.Rect(r.Max.X-scrollSize, hh, r.Max.X, r.Max.Y)
	s.min, s.max = 0, hv.doc.Rect.Dy()-1
	s.page = max(r.Dy()-hh, 1)
	s.set(s.pos)
}

func (hv *helpView) paint(cv canvas) {
	r := hv.local()
	if hv.laid != r.Size() {
		hv.layout()
	}

	if hv.head == nil {
		cv.fill(r, colWindow)

		return
	}

	hh := hv.head.Rect.Dy()
	cv.copy(0, 0, hv.head, hv.head.Rect)
	cv.fill(image.Rect(0, hh, r.Dx(), r.Dy()), colWindow)
	cv.copy(0, hh, hv.doc, image.Rect(0, hv.scroll.pos, hv.doc.Rect.Dx(), hv.scroll.pos+r.Dy()-hh))

	if hv.sel >= 0 {
		in := cv.sub(image.Rect(0, hh, hv.doc.Rect.Dx(), r.Dy()))
		for _, b := range hv.spots[hv.sel].boxes {
			b = b.Sub(image.Pt(0, hv.scroll.pos))
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
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

// spotAt returns the hotspot at p, in the view's coordinates, or -1.
func (hv *helpView) spotAt(p image.Point) int {
	if hv.head == nil || p.Y < hv.head.Rect.Dy() || p.X >= hv.doc.Rect.Dx() {
		return -1
	}

	d := p.Add(image.Pt(0, hv.scroll.pos-hv.head.Rect.Dy()))
	for i, s := range hv.spots {
		if slices.ContainsFunc(s.boxes, d.In) {
			return i
		}
	}

	return -1
}

func (hv *helpView) mouseDown(p image.Point, _ bool) {
	if hv.scroll.enabled() && hv.scroll.press(p) {
		hv.invalidate()

		return
	}

	hv.press = hv.spotAt(p)
}

func (hv *helpView) mouseDrag(p image.Point) {
	if hv.scroll.pressed != partNone {
		hv.scroll.drag(p)
		hv.invalidate()
	}
}

func (hv *helpView) mouseUp(p image.Point) {
	if hv.scroll.pressed != partNone {
		hv.scroll.release()
		hv.invalidate()

		return
	}

	k := hv.press
	hv.press = -1

	if k >= 0 && hv.spotAt(p) == k {
		hv.follow(k)
	}
}

func (hv *helpView) tick(now time.Time) {
	if hv.scroll.tick(now) {
		hv.invalidate()
	}
}

func (hv *helpView) cursor(p image.Point) ebiten.CursorShapeType {
	if hv.spotAt(p) >= 0 {
		return ebiten.CursorShapePointer
	}

	return ebiten.CursorShapeDefault
}

func (hv *helpView) wheel(dy int) {
	hv.scroll.set(hv.scroll.pos - 3*dy*hv.scroll.line)
	hv.invalidate()
}

func (hv *helpView) key(k ebiten.Key, m mods) bool {
	s := &hv.scroll

	switch k {
	case ebiten.KeyUp:
		s.set(s.pos - s.line)
	case ebiten.KeyDown:
		s.set(s.pos + s.line)
	case ebiten.KeyPageUp:
		s.set(s.pos - s.page)
	case ebiten.KeyPageDown:
		s.set(s.pos + s.page)
	case ebiten.KeyHome:
		s.set(0)
	case ebiten.KeyEnd:
		s.set(s.limit())
	case ebiten.KeyTab:
		hv.choose(m.shift)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if hv.sel < 0 {
			return false
		}

		hv.follow(hv.sel)
	case ebiten.KeyBackspace:
		hv.h.goBack()
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
		b, s := boxes[0], &hv.scroll
		switch {
		case b.Min.Y < s.pos:
			s.set(b.Min.Y)
		case b.Max.Y > s.pos+s.page:
			s.set(b.Max.Y - s.page)
		}
	}
}

// follow follows a hotspot: to another page, or out to the browser or the
// e-mail program.
func (hv *helpView) follow(k int) {
	t := hv.spots[k].target
	for _, scheme := range []string{"http:", "https:", "mailto:"} {
		if strings.HasPrefix(t, scheme) {
			if err := hv.h.openURL(t); err != nil {
				a := hv.h.a
				a.call(func() {
					a.Message(game.Error, false, err.Error())
				})
			}

			return
		}
	}

	hv.h.jump(t)
}

// helpOp is something to draw: text, a picture, or a box, filled or
// outlined.
type helpOp struct {
	r     image.Rectangle
	c     color.RGBA
	text  string
	f     face
	img   *image.RGBA
	frame bool
}

// helpPiece is text in one face, or a picture, that the flow places.
type helpPiece struct {
	text string
	f    face
	c    color.RGBA
	img  *image.RGBA
	spot int // its hotspot, or -1
	w    int
}

// helpWord is pieces with no space between them, which stay on one line,
// and the space after them, if any.
type helpWord struct {
	pieces []helpPiece
	w      int
	space  helpPiece
	brk    bool // a line break after it
}

// helpFlow lays a topic out: what to draw where, and the hotspots.
type helpFlow struct {
	h     *helpWindow
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

	return hf.h.arial(helpTextPx, st)
}

// blocks lays blocks out between x0 and x1 from y, a blank line apart, and
// returns the y under them.
func (hf *helpFlow) blocks(bs []help.Block, x0, x1, y int) int {
	text := hf.face(false, false)

	for i, b := range bs {
		if i > 0 {
			y += text.height()
		}

		switch b.Kind {
		case help.Paragraph, help.Heading:
			y, _ = hf.flow(b.Spans, x0, x1, y, b.Kind == help.Heading)
		case help.Quote:
			// A port note, in a box of the tooltips' colours.
			box := len(hf.ops)
			hf.ops = append(hf.ops, helpOp{c: colInfo}, helpOp{c: colText, frame: true})
			bottom := hf.blocks(b.Blocks, x0+1+helpPad, x1-1-helpPad, y+1+helpPad) + helpPad + 1
			hf.ops[box].r = image.Rect(x0, y, x1, bottom)
			hf.ops[box+1].r = hf.ops[box].r
			y = bottom
		case help.List:
			for _, it := range b.Items {
				var base int
				y, base = hf.flow(it.Spans, x0+helpIndent, x1, y, false)
				marker, x := "•", x0+4

				if it.Marker != "" {
					marker, x = it.Marker, x0
				}

				hf.ops = append(hf.ops, helpOp{
					r:    image.Rect(x, base-text.ascent(), x, base),
					text: marker,
					f:    text,
					c:    colText,
				})
			}
		case help.Table:
			y = hf.table(b.Rows, x0, y)
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
	for r, row := range rows {
		for c, cell := range row {
			widths[c] = max(widths[c], hf.measure(cell, r == 0))
		}
	}

	rowH := hf.face(false, false).height() + 4

	for r, row := range rows {
		x := x0

		for c, w := range widths {
			cw := w + 2*helpPad
			hf.ops = append(hf.ops, helpOp{
				r:     image.Rect(x, y, x+cw+1, y+rowH+1),
				c:     colShadow,
				frame: true,
			})

			if c < len(row) {
				hf.flow(row[c], x+1+helpPad, x+1+helpPad+w, y+3, r == 0)
			}

			x += cw
		}

		y += rowH
	}

	return y + 1
}

// measure returns the width of text on one line.
func (hf *helpFlow) measure(spans []help.Span, bold bool) int {
	w := 0

	for _, s := range spans {
		if s.Image != "" {
			if img := hf.h.picture(s.Image); img != nil {
				w += img.Rect.Dx()
			}

			continue
		}

		w += hf.face(bold || s.Bold, s.Italic).width(s.Text)
	}

	return w
}

// flow lays spans out from y in lines between x0 and x1, breaking them at
// spaces, and returns the y under them and the first line's baseline.
// Bold sets all of them in bold.
func (hf *helpFlow) flow(spans []help.Span, x0, x1, y int, bold bool) (bottom, baseline int) {
	words := hf.words(spans, bold)
	first := -1
	line := make([]helpWord, 0, len(words))
	w := 0

	for _, wd := range words {
		if n := len(line); n > 0 && w+line[n-1].space.w+wd.w > x1-x0 {
			y = hf.place(line, x0, y, &first)
			line, w = nil, 0
		}

		if n := len(line); n > 0 {
			w += line[n-1].space.w
		}

		line = append(line, wd)
		w += wd.w

		if wd.brk {
			y = hf.place(line, x0, y, &first)
			line, w = nil, 0
		}
	}

	if len(line) > 0 || first < 0 {
		y = hf.place(line, x0, y, &first)
	}

	return y, first
}

// words splits spans into words, and gives each link a hotspot.
func (hf *helpFlow) words(spans []help.Span, bold bool) []helpWord {
	var b wordBuilder

	link, spot := "", -1

	for _, s := range spans {
		if s.Link != link {
			link, spot = s.Link, -1
			if link != "" {
				hf.spots = append(hf.spots, helpSpot{target: link})
				spot = len(hf.spots) - 1
			}
		}

		switch {
		case s.Break:
			b.end(helpPiece{}, true)
		case s.Image != "":
			if img := hf.h.picture(s.Image); img != nil {
				b.add(helpPiece{img: img, spot: spot, w: img.Rect.Dx()})
			}
		default:
			f := hf.face(bold || s.Bold, s.Italic)
			c := colText

			if s.Link != "" {
				c = helpGreen
			}

			for i, part := range strings.Split(s.Text, " ") {
				if i > 0 {
					b.end(helpPiece{
						f:    f,
						c:    c,
						spot: spot,
						w:    f.width(" "),
					}, false)
				}

				if part != "" {
					b.add(helpPiece{
						text: part,
						f:    f,
						c:    c,
						spot: spot,
						w:    f.width(part),
					})
				}
			}
		}
	}

	b.end(helpPiece{}, false)

	return b.words
}

// place puts a line's words on it from x0, the line's top at y, each piece
// on the baseline, and returns the y under the line.
func (hf *helpFlow) place(line []helpWord, x0, y int, first *int) int {
	asc, desc := 0, 0

	for _, wd := range line {
		for _, p := range wd.pieces {
			if p.img != nil {
				asc = max(asc, p.img.Rect.Dy())

				continue
			}

			asc, desc = max(asc, p.f.ascent()), max(desc, p.f.height()-p.f.ascent())
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
		for _, p := range wd.pieces {
			hf.put(p, x, y, base, bottom)

			x += p.w
		}

		if i < len(line)-1 {
			hf.put(wd.space, x, y, base, bottom)

			x += wd.space.w
		}
	}

	return bottom
}

// put places a piece at x on a line from top to bottom: text and pictures
// on the baseline, and for a link, its underline and its hotspot's box.
func (hf *helpFlow) put(p helpPiece, x, top, base, bottom int) {
	switch {
	case p.img != nil:
		hf.ops = append(hf.ops, helpOp{
			r:   image.Rect(x, base-p.img.Rect.Dy(), x+p.w, base),
			img: p.img,
		})
	case p.text != "":
		hf.ops = append(hf.ops, helpOp{
			r:    image.Rect(x, base-p.f.ascent(), x+p.w, base),
			text: p.text,
			f:    p.f,
			c:    p.c,
		})
	}

	if p.spot < 0 || p.w == 0 {
		return
	}

	if p.img == nil {
		hf.ops = append(hf.ops, helpOp{r: image.Rect(x, base+1, x+p.w, base+2), c: p.c})
	}

	s := &hf.spots[p.spot]
	box := image.Rect(x, top, x+p.w, bottom)

	if n := len(s.boxes); n > 0 && s.boxes[n-1].Max.X == x && s.boxes[n-1].Min.Y == top {
		s.boxes[n-1].Max.X = box.Max.X

		return
	}

	s.boxes = append(s.boxes, box)
}

// wordBuilder builds words out of pieces.
type wordBuilder struct {
	words []helpWord
	cur   helpWord
}

// add adds a piece to the current word.
func (wb *wordBuilder) add(p helpPiece) {
	wb.cur.pieces = append(wb.cur.pieces, p)
	wb.cur.w += p.w
}

// end ends the current word with a space, or a line break after it.
func (wb *wordBuilder) end(space helpPiece, brk bool) {
	if len(wb.cur.pieces) == 0 {
		// Spaces in a row count as one.
		if n := len(wb.words); n > 0 {
			wb.words[n-1].brk = wb.words[n-1].brk || brk
			if wb.words[n-1].space.w == 0 {
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
