// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// scrollSize is a scroll bar's thickness (SM_CXVSCROLL, SM_CYHSCROLL).
const scrollSize = 16

// Scroll bar parts.
const (
	partNone = iota
	partLess // the up or left arrow
	partMore // the down or right arrow
	partPageLess
	partPageMore
	partThumb
)

// scrollBar is a Windows 98 scroll bar, drawn and driven by its owner.
type scrollBar struct {
	vertical bool
	r        image.Rectangle // in the owner's coordinates
	min, max int             // the range
	page     int             // what one screenful shows
	line     int             // what an arrow scrolls by: 1 if 0
	pos      int

	pressed  int // the part held down
	inside   bool
	grab     int // thumb drags: the mouse's offset into the thumb
	next     time.Time
	onScroll func()
}

// enabled reports whether there is anything to scroll.
func (s *scrollBar) enabled() bool { return s.max-s.min >= s.page }

// limit is the largest position.
func (s *scrollBar) limit() int { return max(s.max-s.page+1, s.min) }

func (s *scrollBar) set(pos int) {
	pos = max(min(pos, s.limit()), s.min)
	if pos != s.pos {
		s.pos = pos
		if s.onScroll != nil {
			s.onScroll()
		}
	}
}

// length is the bar's length; arrow is an arrow button's.
func (s *scrollBar) length() int {
	if s.vertical {
		return s.r.Dy()
	}

	return s.r.Dx()
}

func (s *scrollBar) arrow() int { return min(scrollSize, s.length()/2) }

// thumb returns the thumb's start and length along the bar, or a length
// of 0 when there is none.
func (s *scrollBar) thumb() (start, length int) {
	a := s.arrow()
	track := s.length() - 2*a

	if !s.enabled() || track < 8 {
		return 0, 0
	}

	n := s.max - s.min + 1
	l := max(track*s.page/n, 8)

	if l > track {
		return 0, 0
	}

	free := track - l

	return a + free*(s.pos-s.min)/max(n-s.page, 1), l
}

// along returns p's coordinate along the bar.
func (s *scrollBar) along(p image.Point) int {
	if s.vertical {
		return p.Y - s.r.Min.Y
	}

	return p.X - s.r.Min.X
}

func (s *scrollBar) partAt(p image.Point) int {
	if !p.In(s.r) || !s.enabled() {
		return partNone
	}

	a, t := s.along(p), s.arrow()
	ts, tl := s.thumb()

	switch {
	case a < t:
		return partLess
	case a >= s.length()-t:
		return partMore
	case tl > 0 && a < ts:
		return partPageLess
	case tl > 0 && a >= ts+tl:
		return partPageMore
	case tl > 0:
		return partThumb
	}

	return partNone
}

// press handles a mouse press at p; it reports whether p is on the bar.
func (s *scrollBar) press(p image.Point) bool {
	if !p.In(s.r) {
		return false
	}

	s.pressed = s.partAt(p)
	s.inside = true

	if s.pressed == partThumb {
		ts, _ := s.thumb()
		s.grab = s.along(p) - ts
	}

	s.step()

	s.next = time.Now().Add(400 * time.Millisecond)

	return true
}

// step does what the pressed part does once.
func (s *scrollBar) step() {
	switch s.pressed {
	case partLess:
		s.set(s.pos - max(s.line, 1))
	case partMore:
		s.set(s.pos + max(s.line, 1))
	case partPageLess:
		s.set(s.pos - max(s.page, 1))
	case partPageMore:
		s.set(s.pos + max(s.page, 1))
	}
}

func (s *scrollBar) drag(p image.Point) {
	if s.pressed == partThumb {
		_, tl := s.thumb()
		a := s.arrow()
		free := s.length() - 2*a - tl

		if free > 0 {
			n := s.max - s.min + 1 - s.page
			s.set(s.min + ((s.along(p)-s.grab-a)*n+free/2)/free)
		}

		return
	}

	s.inside = s.partAt(p) == s.pressed
}

func (s *scrollBar) release() { s.pressed = partNone }

// tick repeats a held arrow or page click; it reports whether anything
// moved.
func (s *scrollBar) tick(now time.Time) bool {
	if s.pressed == partNone || s.pressed == partThumb || !s.inside || now.Before(s.next) {
		return false
	}

	before := s.pos
	s.step()

	s.next = now.Add(50 * time.Millisecond)

	return s.pos != before
}

func (s *scrollBar) paint(cv canvas) {
	r := s.r
	en := s.enabled()
	a := s.arrow()

	var less, more, track image.Rectangle

	if s.vertical {
		less = image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+a)
		more = image.Rect(r.Min.X, r.Max.Y-a, r.Max.X, r.Max.Y)
		track = image.Rect(r.Min.X, less.Max.Y, r.Max.X, more.Min.Y)
	} else {
		less = image.Rect(r.Min.X, r.Min.Y, r.Min.X+a, r.Max.Y)
		more = image.Rect(r.Max.X-a, r.Min.Y, r.Max.X, r.Max.Y)
		track = image.Rect(less.Max.X, r.Min.Y, more.Min.X, r.Max.Y)
	}

	// The track is a checkerboard of the face and highlight colours.
	for y := track.Min.Y; y < track.Max.Y; y++ {
		for x := track.Min.X; x < track.Max.X; x++ {
			c := colFace
			if (x-r.Min.X+y-r.Min.Y)&1 == 1 {
				c = colHighlight
			}

			cv.pix(x, y, c)
		}
	}

	ts, tl := s.thumb()
	if tl > 0 && s.pressed >= partPageLess && s.pressed <= partPageMore && s.inside {
		// The held part of the track turns dark.
		var pr image.Rectangle
		if s.pressed == partPageLess {
			pr = s.span(track, a, ts)
		} else {
			pr = s.span(track, ts+tl, s.length()-a)
		}

		cv.fill(pr, colDkShadow)
	}

	dir := [2]int{arrowUp, arrowDown}
	if !s.vertical {
		dir = [2]int{arrowLeft, arrowRight}
	}

	paintArrowButton(cv, less, dir[0], en, s.pressed == partLess && s.inside, 4)
	paintArrowButton(cv, more, dir[1], en, s.pressed == partMore && s.inside, 4)

	if tl > 0 {
		t := s.span(track, ts, ts+tl)
		cv.fill(t, colFace)
		cv.raised(t)
	}
}

// span returns the part of the track from a to b along the bar.
func (s *scrollBar) span(track image.Rectangle, a, b int) image.Rectangle {
	if s.vertical {
		return image.Rect(track.Min.X, s.r.Min.Y+a, track.Max.X, s.r.Min.Y+b)
	}

	return image.Rect(s.r.Min.X+a, track.Min.Y, s.r.Min.X+b, track.Max.Y)
}

// Arrow directions.
const (
	arrowUp = iota
	arrowDown
	arrowLeft
	arrowRight
)

// paintArrowButton draws a scroll bar or combo box arrow button: a raised
// box, or a flat one while pushed, with a 7×4 triangle, black or embossed
// when disabled. gx is the triangle's left edge in an unpushed 16-pixel
// button.
func paintArrowButton(cv canvas, r image.Rectangle, dir int, enabled, pushed bool, gx int) {
	cv.fill(r, colFace)

	off := 0

	if pushed {
		cv.frame(r, colShadow)

		off = 1
	} else {
		cv.raised(r)
	}

	w, h := 7, 4
	if dir == arrowLeft || dir == arrowRight {
		w, h = 4, 7
	}

	x := r.Min.X + gx - (scrollSize-r.Dx())/2 + off
	y := r.Min.Y + (r.Dy()-h)/2 + off

	if dir == arrowLeft || dir == arrowRight {
		x = r.Min.X + (r.Dx()-w)/2 + off
		y = r.Min.Y + (r.Dy()-h)/2 + off
	}

	if enabled {
		paintTriangle(cv, x, y, dir, colText)

		return
	}

	paintTriangle(cv, x+1, y+1, dir, colHighlight)
	paintTriangle(cv, x, y, dir, colShadow)
}

// paintTriangle draws a 7×4 arrow (4×7 sideways) at (x, y).
func paintTriangle(cv canvas, x, y, dir int, c color.RGBA) {
	for i := range 4 {
		switch dir {
		case arrowUp:
			cv.hline(x+3-i, x+4+i, y+i, c)
		case arrowDown:
			cv.hline(x+i, x+7-i, y+i, c)
		case arrowLeft:
			cv.vline(x+i, y+3-i, y+4+i, c)
		case arrowRight:
			cv.vline(x+i, y+i, y+7-i, c)
		}
	}
}

// listItem is an item of a list view.
type listItem struct {
	caption string
	image   int // the image list's frame, or -1
}

// listView is a TListView in the list style (vsList): items in columns,
// each an icon from SmallImages and a caption, scrolling sideways.
type listView struct {
	ctl
	items    []listItem
	images   *imageList
	border   bool
	selected int // ItemIndex
	hot      int // the item under the mouse, hot-tracked
	hotTrack bool
	scroll   scrollBar // horizontal, in columns
	pressing bool
	onSelect func() // the selection changed, by mouse or keyboard
	// The layout, computed when the items change.
	laid       bool
	item, icon image.Point
	perCol     int
}

func (l *listView) setup(comp *layout.Component) {
	l.windowed = true
	l.tabStop = l.prop("TabStop") != "False"
	l.border = l.prop("BorderStyle") != "bsNone"
	l.color = vclColor(l.prop("Color"), colWindow)
	l.images = l.win.res.imageList(l.prop("SmallImages"))
	l.hotTrack = l.prop("HotTrackStyles") != ""

	for _, it := range comp.List {
		l.items = append(l.items, listItem{it.Caption, it.ImageIndex})
	}

	l.selected, l.hot = -1, -1
	l.scroll.onScroll = l.invalidate
}

func (l *listView) focusable() bool { return true }

// setItems replaces the items, keeping the selection when it still exists.
func (l *listView) setItems(items []listItem) {
	same := len(items) == len(l.items)
	for i := 0; same && i < len(items); i++ {
		same = items[i] == l.items[i]
	}

	if same {
		return
	}

	l.items = items
	l.laid = false

	if l.selected >= len(items) {
		l.selected = -1
	}

	if l.hot >= len(items) {
		l.hot = -1
	}

	l.invalidate()
}

// client is the rectangle inside the border.
func (l *listView) client() image.Rectangle {
	r := l.local()
	if l.border {
		r = r.Inset(2)
	}

	return r
}

// metrics returns an item's size, the icon's size and the items per
// column, laying the scroll bar out.
func (l *listView) metrics() (item, icon image.Point, perCol int) {
	if !l.laid {
		l.item, l.icon, l.perCol = l.layout()
		l.laid = true
	}

	return l.item, l.icon, l.perCol
}

func (l *listView) layout() (item, icon image.Point, perCol int) {
	if l.images != nil {
		icon = image.Pt(l.images.size, l.images.size)
	}

	h := max(icon.Y, l.font.height()) + 1
	w := 0

	for _, it := range l.items {
		w = max(w, l.font.width(it.caption))
	}

	w = max(128, w+icon.X+12)
	c := l.client()
	perCol = max(c.Dy()/h, 1)
	cols := (len(l.items) + perCol - 1) / perCol
	// The bar shows when there are more columns than fit, at least one
	// always fitting, however wide (LISTVIEW_UpdateHScroll).
	if cols > max(c.Dx()/w, 1) && c.Dy() > scrollSize {
		perCol = max((c.Dy()-scrollSize)/h, 1)
		cols = (len(l.items) + perCol - 1) / perCol
		l.scroll.r = image.Rect(c.Min.X, c.Max.Y-scrollSize, c.Max.X, c.Max.Y)
		l.scroll.min, l.scroll.max, l.scroll.page = 0, cols-1, max(c.Dx()/w, 1)
		l.scroll.set(l.scroll.pos)
	} else {
		l.scroll.r = image.Rectangle{}
		l.scroll.pos = 0
	}

	return image.Pt(w, h), icon, perCol
}

// itemRect is item i's rectangle.
func (l *listView) itemRect(i int) image.Rectangle {
	item, _, perCol := l.metrics()
	c := l.client()
	x := c.Min.X + (i/perCol-l.scroll.pos)*item.X
	y := c.Min.Y + i%perCol*item.Y

	return image.Rect(x, y, x+item.X, y+item.Y)
}

// at returns the item whose icon or caption is at p, or -1.
func (l *listView) at(p image.Point) int {
	if !p.In(l.client()) || p.In(l.scroll.r) {
		return -1
	}

	_, icon, _ := l.metrics()
	for i, it := range l.items {
		r := l.itemRect(i)
		r.Max.X = r.Min.X + icon.X + 2 + l.font.width(it.caption) + 4

		if p.In(r) {
			return i
		}
	}

	return -1
}

func (l *listView) paint(cv canvas) {
	r := l.local()
	if l.border {
		cv.sunken(r)
	}

	c := l.client()
	cv.fill(c, l.color)

	_, icon, _ := l.metrics()
	in := cv.sub(c)
	in.org = cv.org // keep the list's coordinates, clipped to the inside
	focus := l.focused() && l.win.active

	for i := range l.items {
		if ir := l.itemRect(i); ir.Overlaps(c) {
			l.paintItem(in, i, ir, icon, focus)
		}
	}

	if l.scroll.r != (image.Rectangle{}) {
		l.scroll.paint(cv)
	}
}

// paintItem paints item i in its rectangle: its icon, then its caption,
// selected or hot-tracked.
func (l *listView) paintItem(in canvas, i int, ir image.Rectangle, icon image.Point, focus bool) {
	it := l.items[i]

	sel := i == l.selected
	if l.images != nil && it.image >= 0 {
		l.images.drawFrame(in, ir.Min.X, ir.Min.Y, it.image, sel && focus)
	}

	tx := ir.Min.X + icon.X + 2
	ty := ir.Min.Y + (ir.Dy()-l.font.height())/2
	tw := l.font.width(it.caption)
	fg := l.fontColor
	label := image.Rect(tx-2, ty, tx+tw+2, ty+l.font.height())

	if sel && focus {
		in.fill(label, colSelection)

		fg = colSelText
	}

	if l.hotTrack && i == l.hot && l.usable() {
		if !sel || !focus {
			fg = rgb(0x0000FF)
		}

		l.font.draw(in, tx, ty, it.caption, fg)
		in.hline(tx, tx+tw, ty+l.font.ascent()+1, fg)
	} else {
		l.font.draw(in, tx, ty, it.caption, fg)
	}

	if sel && focus {
		in.focusRect(label)
	}
}

func (l *listView) mouseDown(p image.Point, double bool) {
	if l.scroll.press(p) {
		l.invalidate()

		return
	}

	k := l.at(p)
	if l.selected != k {
		l.selected = k
		l.invalidate()

		if l.onSelect != nil {
			l.onSelect()
		}
	}

	l.pressing = true
	if double && k >= 0 && l.onDblClick != nil {
		l.onDblClick()
	}
}

func (l *listView) mouseDrag(p image.Point) {
	if l.scroll.pressed != partNone {
		l.scroll.drag(p)
		l.invalidate()
	}
}

func (l *listView) mouseUp(image.Point) {
	if l.scroll.pressed != partNone {
		l.scroll.release()
		l.invalidate()

		return
	}

	if l.pressing {
		l.pressing = false
		if l.onClick != nil {
			l.onClick()
		}
	}
}

func (l *listView) hover(p image.Point, out bool) {
	k := -1
	if !out {
		k = l.at(p)
	}

	if k != l.hot {
		l.hot = k
		l.invalidate()
	}
}

func (l *listView) cursor(p image.Point) ebiten.CursorShapeType {
	if l.hotTrack && l.at(p) >= 0 {
		return ebiten.CursorShapePointer
	}

	return ebiten.CursorShapeDefault
}

func (l *listView) tick(now time.Time) {
	if l.scroll.tick(now) {
		l.invalidate()
	}
}

func (l *listView) key(k ebiten.Key, _ mods) bool {
	n := len(l.items)
	if n == 0 {
		return false
	}

	_, _, perCol := l.metrics()
	s := l.selected

	switch k {
	case ebiten.KeyUp:
		s = max(s-1, 0)
	case ebiten.KeyDown:
		s = min(s+1, n-1)
	case ebiten.KeyLeft:
		s = max(s-perCol, 0)
	case ebiten.KeyRight:
		s = min(s+perCol, n-1)
	case ebiten.KeyHome:
		s = 0
	case ebiten.KeyEnd:
		s = n - 1
	default:
		return false
	}

	s = max(s, 0)
	if s != l.selected {
		l.selected = s
		l.ensureVisible(s)
		l.invalidate()

		if l.onSelect != nil {
			l.onSelect()
		}
	}

	return true
}

func (l *listView) char(rune) {}

// ensureVisible scrolls item i's column into view.
func (l *listView) ensureVisible(i int) {
	_, _, perCol := l.metrics()
	col := i / perCol

	if col < l.scroll.pos {
		l.scroll.set(col)
	} else if l.scroll.page > 0 && col >= l.scroll.pos+l.scroll.page {
		l.scroll.set(col - l.scroll.page + 1)
	}
}

func (l *listView) wheel(dy int) {
	if l.scroll.r != (image.Rectangle{}) {
		l.scroll.set(l.scroll.pos - dy)
		l.invalidate()
	}
}

// dropList is a combo box's drop-down list: a popup list box with a black
// border, eight items high, that tracks the mouse.
type dropList struct {
	combo  *comboBox
	r      image.Rectangle // on the desktop
	top    int             // the first item shown
	hot    int
	scroll scrollBar
	rows   int
}

func newDropList(c *comboBox) *dropList {
	d := &dropList{combo: c, hot: c.index}
	d.rows = min(len(c.items), 8)
	h := max(d.rows, 1)*c.itemHeight + 2
	o := c.win.clientOrigin().Add(c.origin())
	d.r = image.Rect(o.X, o.Y+c.r.Dy(), o.X+c.r.Dx(), o.Y+c.r.Dy()+h)

	if d.r.Max.Y > screenHeight {
		d.r = d.r.Sub(image.Pt(0, h+c.r.Dy()))
	}

	if len(c.items) > d.rows {
		d.scroll = scrollBar{
			vertical: true, r: image.Rect(d.r.Dx()-1-scrollSize, 1, d.r.Dx()-1, d.r.Dy()-1),
			min: 0, max: len(c.items) - 1, page: d.rows,
		}
	}

	d.show(max(c.index, 0))

	d.scroll.onScroll = func() { d.top = d.scroll.pos }

	return d
}

// show scrolls item i into view.
func (d *dropList) show(i int) {
	if i < d.top {
		d.top = i
	} else if i >= d.top+d.rows {
		d.top = i - d.rows + 1
	}

	d.top = max(min(d.top, len(d.combo.items)-d.rows), 0)
	d.scroll.pos = d.top
}

func (d *dropList) bounds() image.Rectangle { return d.r }

func (d *dropList) paint(cv canvas) {
	cv = cv.sub(d.r)
	r := image.Rectangle{Max: d.r.Size()}
	cv.frame(r, colFrame)

	in := r.Inset(1)
	if d.scroll.max > 0 {
		in.Max.X -= scrollSize
	}

	cv.fill(in, colWindow)

	c := d.combo
	for row := range d.rows {
		i := d.top + row
		if i >= len(c.items) {
			break
		}

		y := in.Min.Y + row*c.itemHeight
		ir := image.Rect(in.Min.X, y, in.Max.X, y+c.itemHeight)
		tx := 2

		if c.images != nil {
			// Owner-drawn: the item's icon at its indent, then its text,
			// highlighted on its own.
			x := ir.Min.X + 1 + c.indents[i]
			c.images.drawFrame(cv, x, y+(c.itemHeight-c.images.size)/2, c.icons[i], false)

			x += c.images.size + 3
			ir = image.Rect(x, y, min(x+c.font.width(c.items[i])+4, in.Max.X), y+c.itemHeight)
		}

		fg := colText

		if i == d.hot {
			cv.fill(ir, colSelection)

			fg = colSelText
		}

		c.font.draw(cv.sub(ir), tx, (c.itemHeight-c.font.height())/2, c.items[i], fg)
	}

	if d.scroll.max > 0 {
		d.scroll.paint(cv)
	}
}

// itemAt returns the item at p (on the desktop), or -1.
func (d *dropList) itemAt(p image.Point) int {
	q := p.Sub(d.r.Min)
	in := image.Rectangle{Max: d.r.Size()}.Inset(1)

	if d.scroll.max > 0 {
		in.Max.X -= scrollSize
	}

	if !q.In(in) {
		return -1
	}

	i := d.top + (q.Y-in.Min.Y)/d.combo.itemHeight
	if i >= len(d.combo.items) {
		return -1
	}

	return i
}

func (d *dropList) mouseDown(p image.Point) {
	if d.scroll.press(p.Sub(d.r.Min)) {
		return
	}
}

func (d *dropList) mouseMove(p image.Point, down bool) {
	if d.scroll.pressed != partNone {
		d.scroll.drag(p.Sub(d.r.Min))

		return
	}

	if i := d.itemAt(p); i >= 0 {
		d.hot = i
	}
}

func (d *dropList) mouseUp(p image.Point) bool {
	if d.scroll.pressed != partNone {
		d.scroll.release()

		return false
	}

	if i := d.itemAt(p); i >= 0 {
		d.combo.pick(i)

		return true
	}

	return false
}

func (d *dropList) tick(now time.Time) bool { return d.scroll.tick(now) }

// dismiss closes the list, as a click outside it does.
func (d *dropList) dismiss() { d.combo.closeUp() }

func (d *dropList) wheel(dy int) {
	if d.scroll.max > 0 {
		d.scroll.set(d.scroll.pos - dy*3)
	}
}

func (d *dropList) key(k ebiten.Key) bool {
	n := len(d.combo.items)

	switch k {
	case ebiten.KeyUp:
		d.hot = max(d.hot-1, 0)
	case ebiten.KeyDown:
		d.hot = min(d.hot+1, n-1)
	case ebiten.KeyPageUp:
		d.hot = max(d.hot-d.rows, 0)
	case ebiten.KeyPageDown:
		d.hot = min(d.hot+d.rows, n-1)
	case ebiten.KeyHome:
		d.hot = 0
	case ebiten.KeyEnd:
		d.hot = n - 1
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if d.hot >= 0 {
			d.combo.pick(d.hot)
		}

		return true
	default:
		return false
	}

	d.show(max(d.hot, 0))

	if d.hot >= 0 {
		d.combo.pick(d.hot)

		d.combo.dropped = d
	}

	return true
}
