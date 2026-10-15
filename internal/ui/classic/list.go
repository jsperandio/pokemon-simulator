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
const scrollSize int = 16

// Scroll bar parts.
const (
	partNone int = iota
	partLess     // the up or left arrow
	partMore     // the down or right arrow
	partPageLess
	partPageMore
	partThumb
)

// scrollBar is a Windows 98 scroll bar, drawn and driven by its owner.
type scrollBar struct {
	vertical bool
	rect     image.Rectangle // in the owner's coordinates
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
func (sb *scrollBar) enabled() bool {
	return sb.max-sb.min >= sb.page
}

// limit is the largest position.
func (sb *scrollBar) limit() int {
	return max(sb.max-sb.page+1, sb.min)
}

func (sb *scrollBar) set(pos int) {
	pos = max(min(pos, sb.limit()), sb.min)
	if pos != sb.pos {
		sb.pos = pos
		if sb.onScroll != nil {
			sb.onScroll()
		}
	}
}

// length is the bar's length; arrow is an arrow button's.
func (sb *scrollBar) length() int {
	if sb.vertical {
		return sb.rect.Dy()
	}

	return sb.rect.Dx()
}

func (sb *scrollBar) arrow() int {
	return min(scrollSize, sb.length()/2)
}

// thumb returns the thumb's start and length along the bar, or a length
// of 0 when there is none.
func (sb *scrollBar) thumb() (start, length int) {
	a := sb.arrow()
	track := sb.length() - 2*a

	if !sb.enabled() || track < 8 {
		return 0, 0
	}

	n := sb.max - sb.min + 1
	l := max(track*sb.page/n, 8)

	if l > track {
		return 0, 0
	}

	free := track - l

	return a + free*(sb.pos-sb.min)/max(n-sb.page, 1), l
}

// along returns p's coordinate along the bar.
func (sb *scrollBar) along(p image.Point) int {
	if sb.vertical {
		return p.Y - sb.rect.Min.Y
	}

	return p.X - sb.rect.Min.X
}

func (sb *scrollBar) partAt(p image.Point) int {
	if !p.In(sb.rect) || !sb.enabled() {
		return partNone
	}

	a, t := sb.along(p), sb.arrow()
	ts, tl := sb.thumb()

	switch {
	case a < t:
		return partLess
	case a >= sb.length()-t:
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
func (sb *scrollBar) press(p image.Point) bool {
	if !p.In(sb.rect) {
		return false
	}

	sb.pressed = sb.partAt(p)
	sb.inside = true

	if sb.pressed == partThumb {
		ts, _ := sb.thumb()
		sb.grab = sb.along(p) - ts
	}

	sb.step()

	sb.next = time.Now().Add(400 * time.Millisecond)

	return true
}

// step does what the pressed part does once.
func (sb *scrollBar) step() {
	switch sb.pressed {
	case partLess:
		sb.set(sb.pos - max(sb.line, 1))
	case partMore:
		sb.set(sb.pos + max(sb.line, 1))
	case partPageLess:
		sb.set(sb.pos - max(sb.page, 1))
	case partPageMore:
		sb.set(sb.pos + max(sb.page, 1))
	}
}

func (sb *scrollBar) drag(p image.Point) {
	if sb.pressed == partThumb {
		_, tl := sb.thumb()
		a := sb.arrow()
		free := sb.length() - 2*a - tl

		if free > 0 {
			n := sb.max - sb.min + 1 - sb.page
			sb.set(sb.min + ((sb.along(p)-sb.grab-a)*n+free/2)/free)
		}

		return
	}

	sb.inside = sb.partAt(p) == sb.pressed
}

func (sb *scrollBar) release() {
	sb.pressed = partNone
}

// tick repeats a held arrow or page click; it reports whether anything
// moved.
func (sb *scrollBar) tick(now time.Time) bool {
	// Only an arrow or the track repeats, with the mouse on it.
	notHeld := sb.pressed == partNone || sb.pressed == partThumb || !sb.inside
	if notHeld || now.Before(sb.next) {
		return false
	}

	before := sb.pos
	sb.step()

	sb.next = now.Add(50 * time.Millisecond)

	return sb.pos != before
}

func (sb *scrollBar) paint(cv canvas) {
	r := sb.rect
	en := sb.enabled()
	a := sb.arrow()

	var less, more, track image.Rectangle

	if sb.vertical {
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

	ts, tl := sb.thumb()

	pageHeld := sb.pressed >= partPageLess && sb.pressed <= partPageMore && sb.inside
	if tl > 0 && pageHeld {
		// The held part of the track turns dark.
		from, to := ts+tl, sb.length()-a
		if sb.pressed == partPageLess {
			from, to = a, ts
		}

		cv.fill(sb.span(track, from, to), colDkShadow)
	}

	dir := [2]int{arrowUp, arrowDown}
	if !sb.vertical {
		dir = [2]int{arrowLeft, arrowRight}
	}

	paintArrowButton(cv, less, dir[0], en, sb.pressed == partLess && sb.inside, 4)
	paintArrowButton(cv, more, dir[1], en, sb.pressed == partMore && sb.inside, 4)

	if tl > 0 {
		t := sb.span(track, ts, ts+tl)
		cv.fill(t, colFace)
		cv.raised(t)
	}
}

// span returns the part of the track from a to b along the bar.
func (sb *scrollBar) span(track image.Rectangle, a, b int) image.Rectangle {
	if sb.vertical {
		return image.Rect(track.Min.X, sb.rect.Min.Y+a, track.Max.X, sb.rect.Min.Y+b)
	}

	return image.Rect(sb.rect.Min.X+a, track.Min.Y, sb.rect.Min.X+b, track.Max.Y)
}

// Arrow directions.
const (
	arrowUp int = iota
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
	control
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

func (lv *listView) setup(comp *layout.Component) {
	lv.windowed = true
	lv.tabStop = lv.property("TabStop") != "False"
	lv.border = lv.property("BorderStyle") != "bsNone"
	lv.color = vclColor(lv.property("Color"), colWindow)
	lv.images = lv.wdw.res.imageList(lv.property("SmallImages"))
	lv.hotTrack = lv.property("HotTrackStyles") != ""

	for _, it := range comp.List {
		lv.items = append(lv.items, listItem{it.Caption, it.ImageIndex})
	}

	lv.selected, lv.hot = -1, -1
	lv.scroll.onScroll = lv.invalidate
}

func (lv *listView) focusable() bool {
	return true
}

// setItems replaces the items, keeping the selection when it still exists.
func (lv *listView) setItems(items []listItem) {
	same := len(items) == len(lv.items)
	for i := 0; same && i < len(items); i++ {
		same = items[i] == lv.items[i]
	}

	if same {
		return
	}

	lv.items = items
	lv.laid = false

	if lv.selected >= len(items) {
		lv.selected = -1
	}

	if lv.hot >= len(items) {
		lv.hot = -1
	}

	lv.invalidate()
}

// client is the rectangle inside the border.
func (lv *listView) client() image.Rectangle {
	r := lv.local()
	if lv.border {
		r = r.Inset(2)
	}

	return r
}

// metrics returns an item's size, the icon's size and the items per
// column, laying the scroll bar out.
func (lv *listView) metrics() (item, icon image.Point, perCol int) {
	if !lv.laid {
		lv.item, lv.icon, lv.perCol = lv.layout()
		lv.laid = true
	}

	return lv.item, lv.icon, lv.perCol
}

func (lv *listView) layout() (item, icon image.Point, perCol int) {
	if lv.images != nil {
		icon = image.Pt(lv.images.size, lv.images.size)
	}

	h := max(icon.Y, lv.font.height()) + 1
	w := 0

	for _, it := range lv.items {
		w = max(w, lv.font.width(it.caption))
	}

	w = max(128, w+icon.X+12)
	c := lv.client()
	bar := &lv.scroll
	perCol = max(c.Dy()/h, 1)
	cols := (len(lv.items) + perCol - 1) / perCol
	// The bar shows when there are more columns than fit, at least one
	// always fitting, however wide (LISTVIEW_UpdateHScroll).
	if cols > max(c.Dx()/w, 1) && c.Dy() > scrollSize {
		perCol = max((c.Dy()-scrollSize)/h, 1)
		cols = (len(lv.items) + perCol - 1) / perCol
		bar.rect = image.Rect(c.Min.X, c.Max.Y-scrollSize, c.Max.X, c.Max.Y)
		bar.min, bar.max, bar.page = 0, cols-1, max(c.Dx()/w, 1)
		bar.set(bar.pos)
	} else {
		bar.rect = image.Rectangle{}
		bar.pos = 0
	}

	return image.Pt(w, h), icon, perCol
}

// itemRect is item i's rectangle.
func (lv *listView) itemRect(i int) image.Rectangle {
	item, _, perCol := lv.metrics()
	c := lv.client()
	x := c.Min.X + (i/perCol-lv.scroll.pos)*item.X
	y := c.Min.Y + i%perCol*item.Y

	return image.Rect(x, y, x+item.X, y+item.Y)
}

// at returns the item whose icon or caption is at p, or -1.
func (lv *listView) at(p image.Point) int {
	if !p.In(lv.client()) || p.In(lv.scroll.rect) {
		return -1
	}

	_, icon, _ := lv.metrics()
	for i, it := range lv.items {
		r := lv.itemRect(i)
		r.Max.X = r.Min.X + icon.X + 2 + lv.font.width(it.caption) + 4

		if p.In(r) {
			return i
		}
	}

	return -1
}

func (lv *listView) paint(cv canvas) {
	r := lv.local()
	if lv.border {
		cv.sunken(r)
	}

	c := lv.client()
	cv.fill(c, lv.color)

	_, icon, _ := lv.metrics()
	in := cv.sub(c)
	in.org = cv.org // keep the list's coordinates, clipped to the inside
	focus := lv.focused() && lv.wdw.active

	for i := range lv.items {
		if ir := lv.itemRect(i); ir.Overlaps(c) {
			lv.paintItem(in, i, ir, icon, focus)
		}
	}

	if lv.scroll.rect != (image.Rectangle{}) {
		lv.scroll.paint(cv)
	}
}

// paintItem paints item i in its rectangle: its icon, then its caption,
// selected or hot-tracked.
func (lv *listView) paintItem(in canvas, i int, ir image.Rectangle, icon image.Point, focus bool) {
	it := lv.items[i]

	sel := i == lv.selected
	if lv.images != nil && it.image >= 0 {
		lv.images.drawFrame(in, ir.Min.X, ir.Min.Y, it.image, sel && focus)
	}

	tx := ir.Min.X + icon.X + 2
	ty := ir.Min.Y + (ir.Dy()-lv.font.height())/2
	tw := lv.font.width(it.caption)
	fg := lv.fontColor
	label := image.Rect(tx-2, ty, tx+tw+2, ty+lv.font.height())

	if sel && focus {
		in.fill(label, colSelection)

		fg = colSelText
	}

	hot := lv.hotTrack && i == lv.hot && lv.usable()
	if hot && (!sel || !focus) {
		fg = rgb(0x0000FF)
	}

	lv.font.draw(in, tx, ty, it.caption, fg)

	if hot {
		in.hline(tx, tx+tw, ty+lv.font.ascent()+1, fg)
	}

	if sel && focus {
		in.focusRect(label)
	}
}

func (lv *listView) mouseDown(p image.Point, double bool) {
	if lv.scroll.press(p) {
		lv.invalidate()

		return
	}

	k := lv.at(p)
	if lv.selected != k {
		lv.selected = k
		lv.invalidate()

		if lv.onSelect != nil {
			lv.onSelect()
		}
	}

	lv.pressing = true
	if double && k >= 0 && lv.onDblClick != nil {
		lv.onDblClick()
	}
}

func (lv *listView) mouseDrag(p image.Point) {
	if lv.scroll.pressed != partNone {
		lv.scroll.drag(p)
		lv.invalidate()
	}
}

func (lv *listView) mouseUp(image.Point) {
	if lv.scroll.pressed != partNone {
		lv.scroll.release()
		lv.invalidate()

		return
	}

	if lv.pressing {
		lv.pressing = false
		if lv.onClick != nil {
			lv.onClick()
		}
	}
}

func (lv *listView) hover(p image.Point, out bool) {
	k := -1
	if !out {
		k = lv.at(p)
	}

	if k != lv.hot {
		lv.hot = k
		lv.invalidate()
	}
}

func (lv *listView) cursor(p image.Point) ebiten.CursorShapeType {
	if lv.hotTrack && lv.at(p) >= 0 {
		return ebiten.CursorShapePointer
	}

	return ebiten.CursorShapeDefault
}

func (lv *listView) tick(now time.Time) {
	if lv.scroll.tick(now) {
		lv.invalidate()
	}
}

func (lv *listView) key(k ebiten.Key, _ mods) bool {
	n := len(lv.items)
	if n == 0 {
		return false
	}

	_, _, perCol := lv.metrics()
	s := lv.selected

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
	if s != lv.selected {
		lv.selected = s
		lv.ensureVisible(s)
		lv.invalidate()

		if lv.onSelect != nil {
			lv.onSelect()
		}
	}

	return true
}

func (lv *listView) char(rune) {}

// ensureVisible scrolls item i's column into view.
func (lv *listView) ensureVisible(i int) {
	_, _, perCol := lv.metrics()
	col := i / perCol

	switch {
	case col < lv.scroll.pos:
		lv.scroll.set(col)
	case lv.scroll.page > 0 && col >= lv.scroll.pos+lv.scroll.page:
		lv.scroll.set(col - lv.scroll.page + 1)
	}
}

func (lv *listView) wheel(dy int) {
	if lv.scroll.rect != (image.Rectangle{}) {
		lv.scroll.set(lv.scroll.pos - dy)
		lv.invalidate()
	}
}

// dropList is a combo box's drop-down list: a popup list box with a black
// border, eight items high, that tracks the mouse.
type dropList struct {
	combo  *comboBox
	rect   image.Rectangle // on the desktop
	top    int             // the first item shown
	hot    int
	scroll scrollBar
	rows   int
}

func newDropList(c *comboBox) *dropList {
	d := &dropList{combo: c, hot: c.index}
	d.rows = min(len(c.items), 8)
	h := max(d.rows, 1)*c.itemHeight + 2
	o := c.wdw.clientOrigin().Add(c.origin())
	d.rect = image.Rect(o.X, o.Y+c.rect.Dy(), o.X+c.rect.Dx(), o.Y+c.rect.Dy()+h)

	if d.rect.Max.Y > screenHeight {
		d.rect = d.rect.Sub(image.Pt(0, h+c.rect.Dy()))
	}

	if len(c.items) > d.rows {
		d.scroll = scrollBar{
			vertical: true,
			rect:     image.Rect(d.rect.Dx()-1-scrollSize, 1, d.rect.Dx()-1, d.rect.Dy()-1),
			min:      0,
			max:      len(c.items) - 1,
			page:     d.rows,
		}
	}

	d.show(max(c.index, 0))

	d.scroll.onScroll = func() {
		d.top = d.scroll.pos
	}

	return d
}

// show scrolls item i into view.
func (dl *dropList) show(i int) {
	switch {
	case i < dl.top:
		dl.top = i
	case i >= dl.top+dl.rows:
		dl.top = i - dl.rows + 1
	}

	dl.top = max(min(dl.top, len(dl.combo.items)-dl.rows), 0)
	dl.scroll.pos = dl.top
}

func (dl *dropList) bounds() image.Rectangle {
	return dl.rect
}

func (dl *dropList) paint(cv canvas) {
	cv = cv.sub(dl.rect)
	r := image.Rectangle{Max: dl.rect.Size()}
	cv.frame(r, colFrame)

	in := r.Inset(1)
	if dl.scroll.max > 0 {
		in.Max.X -= scrollSize
	}

	cv.fill(in, colWindow)

	c := dl.combo
	for row := range dl.rows {
		i := dl.top + row
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

		if i == dl.hot {
			cv.fill(ir, colSelection)

			fg = colSelText
		}

		c.font.draw(cv.sub(ir), tx, (c.itemHeight-c.font.height())/2, c.items[i], fg)
	}

	if dl.scroll.max > 0 {
		dl.scroll.paint(cv)
	}
}

// itemAt returns the item at p (on the desktop), or -1.
func (dl *dropList) itemAt(p image.Point) int {
	q := p.Sub(dl.rect.Min)
	in := image.Rectangle{Max: dl.rect.Size()}.Inset(1)

	if dl.scroll.max > 0 {
		in.Max.X -= scrollSize
	}

	if !q.In(in) {
		return -1
	}

	i := dl.top + (q.Y-in.Min.Y)/dl.combo.itemHeight
	if i >= len(dl.combo.items) {
		return -1
	}

	return i
}

func (dl *dropList) mouseDown(p image.Point) {
	if dl.scroll.press(p.Sub(dl.rect.Min)) {
		return
	}
}

func (dl *dropList) mouseMove(p image.Point, down bool) {
	if dl.scroll.pressed != partNone {
		dl.scroll.drag(p.Sub(dl.rect.Min))

		return
	}

	if i := dl.itemAt(p); i >= 0 {
		dl.hot = i
	}
}

func (dl *dropList) mouseUp(p image.Point) bool {
	if dl.scroll.pressed != partNone {
		dl.scroll.release()

		return false
	}

	if i := dl.itemAt(p); i >= 0 {
		dl.combo.pick(i)

		return true
	}

	return false
}

func (dl *dropList) tick(now time.Time) bool {
	return dl.scroll.tick(now)
}

// dismiss closes the list, as a click outside it does.
func (dl *dropList) dismiss() {
	dl.combo.closeUp()
}

func (dl *dropList) wheel(dy int) {
	if dl.scroll.max > 0 {
		dl.scroll.set(dl.scroll.pos - dy*3)
	}
}

func (dl *dropList) key(k ebiten.Key) bool {
	n := len(dl.combo.items)

	switch k {
	case ebiten.KeyUp:
		dl.hot = max(dl.hot-1, 0)
	case ebiten.KeyDown:
		dl.hot = min(dl.hot+1, n-1)
	case ebiten.KeyPageUp:
		dl.hot = max(dl.hot-dl.rows, 0)
	case ebiten.KeyPageDown:
		dl.hot = min(dl.hot+dl.rows, n-1)
	case ebiten.KeyHome:
		dl.hot = 0
	case ebiten.KeyEnd:
		dl.hot = n - 1
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if dl.hot >= 0 {
			dl.combo.pick(dl.hot)
		}

		return true
	default:
		return false
	}

	dl.show(max(dl.hot, 0))

	if dl.hot >= 0 {
		dl.combo.pick(dl.hot)

		dl.combo.dropped = dl
	}

	return true
}
