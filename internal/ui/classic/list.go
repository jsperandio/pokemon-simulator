// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"slices"
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
	arw := sb.arrow()
	track := sb.length() - 2*arw

	if !sb.enabled() || track < 8 {
		return 0, 0
	}

	n := sb.max - sb.min + 1
	siz := max(track*sb.page/n, 8)

	if siz > track {
		return 0, 0
	}

	free := track - siz

	return arw + free*(sb.pos-sb.min)/max(n-sb.page, 1), siz
}

// along returns pnt's coordinate along the bar.
func (sb *scrollBar) along(pnt image.Point) int {
	if sb.vertical {
		return pnt.Y - sb.rect.Min.Y
	}

	return pnt.X - sb.rect.Min.X
}

func (sb *scrollBar) partAt(pnt image.Point) int {
	if !pnt.In(sb.rect) || !sb.enabled() {
		return partNone
	}

	pos, arw := sb.along(pnt), sb.arrow()
	ts, tl := sb.thumb()

	switch {
	case pos < arw:
		return partLess
	case pos >= sb.length()-arw:
		return partMore
	case tl > 0 && pos < ts:
		return partPageLess
	case tl > 0 && pos >= ts+tl:
		return partPageMore
	case tl > 0:
		return partThumb
	}

	return partNone
}

// press handles a mouse press at pnt; it reports whether pnt is on the bar.
func (sb *scrollBar) press(pnt image.Point) bool {
	if !pnt.In(sb.rect) {
		return false
	}

	sb.pressed = sb.partAt(pnt)
	sb.inside = true

	if sb.pressed == partThumb {
		ts, _ := sb.thumb()
		sb.grab = sb.along(pnt) - ts
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

func (sb *scrollBar) drag(pnt image.Point) {
	if sb.pressed == partThumb {
		_, tl := sb.thumb()
		arw := sb.arrow()
		free := sb.length() - 2*arw - tl

		if free > 0 {
			n := sb.max - sb.min + 1 - sb.page
			sb.set(sb.min + ((sb.along(pnt)-sb.grab-arw)*n+free/2)/free)
		}

		return
	}

	sb.inside = sb.partAt(pnt) == sb.pressed
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
	rect := sb.rect
	en := sb.enabled()
	arw := sb.arrow()

	var less, more, track image.Rectangle

	if sb.vertical {
		less = image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+arw)
		more = image.Rect(rect.Min.X, rect.Max.Y-arw, rect.Max.X, rect.Max.Y)
		track = image.Rect(rect.Min.X, less.Max.Y, rect.Max.X, more.Min.Y)
	} else {
		less = image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+arw, rect.Max.Y)
		more = image.Rect(rect.Max.X-arw, rect.Min.Y, rect.Max.X, rect.Max.Y)
		track = image.Rect(less.Max.X, rect.Min.Y, more.Min.X, rect.Max.Y)
	}

	// The track is a checkerboard of the face and highlight colours.
	for y := track.Min.Y; y < track.Max.Y; y++ {
		for x := track.Min.X; x < track.Max.X; x++ {
			clr := colFace
			if (x-rect.Min.X+y-rect.Min.Y)&1 == 1 {
				clr = colHighlight
			}

			cv.pix(x, y, clr)
		}
	}

	ts, tl := sb.thumb()

	pageHeld := sb.pressed >= partPageLess && sb.pressed <= partPageMore && sb.inside
	if tl > 0 && pageHeld {
		// The held part of the track turns dark.
		from, to := ts+tl, sb.length()-arw
		if sb.pressed == partPageLess {
			from, to = arw, ts
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
		thm := sb.span(track, ts, ts+tl)
		cv.fill(thm, colFace)
		cv.raised(thm)
	}
}

// span returns the part of the track from a to end along the bar.
func (sb *scrollBar) span(track image.Rectangle, beg, end int) image.Rectangle {
	if sb.vertical {
		return image.Rect(track.Min.X, sb.rect.Min.Y+beg, track.Max.X, sb.rect.Min.Y+end)
	}

	return image.Rect(sb.rect.Min.X+beg, track.Min.Y, sb.rect.Min.X+end, track.Max.Y)
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
func paintArrowButton(cv canvas, rect image.Rectangle, dir int, enabled, pushed bool, gx int) {
	cv.fill(rect, colFace)

	off := 0

	if pushed {
		cv.frame(rect, colShadow)

		off = 1
	} else {
		cv.raised(rect)
	}

	wid, hgt := 7, 4
	if dir == arrowLeft || dir == arrowRight {
		wid, hgt = 4, 7
	}

	x := rect.Min.X + gx - (scrollSize-rect.Dx())/2 + off
	y := rect.Min.Y + (rect.Dy()-hgt)/2 + off

	if dir == arrowLeft || dir == arrowRight {
		x = rect.Min.X + (rect.Dx()-wid)/2 + off
		y = rect.Min.Y + (rect.Dy()-hgt)/2 + off
	}

	if enabled {
		paintTriangle(cv, x, y, dir, colText)

		return
	}

	paintTriangle(cv, x+1, y+1, dir, colHighlight)
	paintTriangle(cv, x, y, dir, colShadow)
}

// paintTriangle draws a 7×4 arrow (4×7 sideways) at (x, y).
func paintTriangle(cv canvas, x, y, dir int, clr color.RGBA) {
	for i := range 4 {
		switch dir {
		case arrowUp:
			cv.hline(x+3-i, x+4+i, y+i, clr)
		case arrowDown:
			cv.hline(x+i, x+7-i, y+i, clr)
		case arrowLeft:
			cv.vline(x+i, y+3-i, y+4+i, clr)
		case arrowRight:
			cv.vline(x+i, y+i, y+7-i, clr)
		}
	}
}

// listItem is an item of a list view.
type listItem struct {
	caption string
	image   int  // the image list's frame, or -1
	shiny   bool // a shiny POKéMON's: the shiny frame, if any, and the list's mark
}

// listView is a TListView in the list style (vsList): items in columns,
// each an icon from SmallImages and a caption, scrolling sideways.
type listView struct {
	control
	items    []listItem
	images   *imageList
	border   bool
	showOnly bool // a click neither selects an item nor focuses the list
	selected int  // ItemIndex
	hot      int  // the item under the mouse, hot-tracked
	hotTrack bool
	scroll   scrollBar // horizontal, in columns
	pressing bool
	onSelect func() // the selection changed, by mouse or keyboard
	// mark, if set, draws a shiny item's mark over its icon at (x, y).
	mark func(cv canvas, x, y int)
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
		li := listItem{
			caption: it.Caption,
			image:   it.ImageIndex,
		}
		lv.items = append(lv.items, li)
	}

	lv.selected, lv.hot = -1, -1
	lv.scroll.onScroll = lv.invalidate
}

func (lv *listView) focusable() bool {
	return !lv.showOnly
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
	rect := lv.local()
	if lv.border {
		rect = rect.Inset(2)
	}

	return rect
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

	hgt := max(icon.Y, lv.font.height()) + 1
	wid := 0

	for _, it := range lv.items {
		wid = max(wid, lv.font.width(it.caption))
	}

	wid = max(128, wid+icon.X+12)
	rect := lv.client()
	bar := &lv.scroll
	perCol = max(rect.Dy()/hgt, 1)
	cols := (len(lv.items) + perCol - 1) / perCol
	// The bar shows when there are more columns than fit, at least one
	// always fitting, however wide (LISTVIEW_UpdateHScroll).
	if cols > max(rect.Dx()/wid, 1) && rect.Dy() > scrollSize {
		perCol = max((rect.Dy()-scrollSize)/hgt, 1)
		cols = (len(lv.items) + perCol - 1) / perCol
		bar.rect = image.Rect(rect.Min.X, rect.Max.Y-scrollSize, rect.Max.X, rect.Max.Y)
		bar.min, bar.max, bar.page = 0, cols-1, max(rect.Dx()/wid, 1)
		bar.set(bar.pos)
	} else {
		bar.rect = image.Rectangle{}
		bar.pos = 0
	}

	return image.Pt(wid, hgt), icon, perCol
}

// itemRect is item i's rectangle.
func (lv *listView) itemRect(i int) image.Rectangle {
	item, _, perCol := lv.metrics()
	rect := lv.client()
	x := rect.Min.X + (i/perCol-lv.scroll.pos)*item.X
	y := rect.Min.Y + i%perCol*item.Y

	return image.Rect(x, y, x+item.X, y+item.Y)
}

// at returns the item whose icon or caption is at pnt, or -1.
func (lv *listView) at(pnt image.Point) int {
	if !pnt.In(lv.client()) || pnt.In(lv.scroll.rect) {
		return -1
	}

	_, icon, _ := lv.metrics()
	for i, it := range lv.items {
		rect := lv.itemRect(i)
		rect.Max.X = rect.Min.X + icon.X + 2 + lv.font.width(it.caption) + 4

		if pnt.In(rect) {
			return i
		}
	}

	return -1
}

func (lv *listView) paint(cv canvas) {
	rect := lv.local()
	if lv.border {
		cv.sunken(rect)
	}

	cli := lv.client()
	cv.fill(cli, lv.color)

	_, icon, _ := lv.metrics()
	in := cv.sub(cli)
	in.org = cv.org // keep the list's coordinates, clipped to the inside
	focus := lv.focused() && lv.wdw.active

	for i := range lv.items {
		if ir := lv.itemRect(i); ir.Overlaps(cli) {
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
		lv.paintIcon(in, it, ir.Min, sel && focus)
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

// paintIcon draws the icon of item li at pnt, selected or not: for a
// shiny item, the shiny frame if the image list has one, and the list's
// mark over it.
func (lv *listView) paintIcon(in canvas, li listItem, pnt image.Point, selected bool) {
	il := lv.images
	if li.shiny && il.shiny != nil {
		il = il.shiny
	}

	il.drawFrame(in, pnt.X, pnt.Y, li.image, selected)

	if li.shiny && lv.mark != nil {
		lv.mark(in, pnt.X, pnt.Y)
	}
}

func (lv *listView) mouseDown(pnt image.Point, double bool) {
	if lv.scroll.press(pnt) {
		lv.invalidate()

		return
	}

	if lv.showOnly {
		return
	}

	idx := lv.at(pnt)
	if lv.selected != idx {
		lv.selected = idx
		lv.invalidate()

		if lv.onSelect != nil {
			lv.onSelect()
		}
	}

	lv.pressing = true
	if double && idx >= 0 && lv.onDblClick != nil {
		lv.onDblClick()
	}
}

func (lv *listView) mouseDrag(pnt image.Point) {
	if lv.scroll.pressed != partNone {
		lv.scroll.drag(pnt)
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

func (lv *listView) hover(pnt image.Point, out bool) {
	idx := -1
	if !out {
		idx = lv.at(pnt)
	}

	if idx != lv.hot {
		lv.hot = idx
		lv.invalidate()
	}
}

func (lv *listView) cursor(pnt image.Point) ebiten.CursorShapeType {
	if lv.hotTrack && lv.at(pnt) >= 0 {
		return ebiten.CursorShapePointer
	}

	return ebiten.CursorShapeDefault
}

func (lv *listView) tick(now time.Time) {
	if lv.scroll.tick(now) {
		lv.invalidate()
	}
}

func (lv *listView) key(key ebiten.Key, _ mods) bool {
	n := len(lv.items)
	if n == 0 {
		return false
	}

	_, _, perCol := lv.metrics()
	sel := lv.selected

	switch key {
	case ebiten.KeyUp:
		sel = max(sel-1, 0)
	case ebiten.KeyDown:
		sel = min(sel+1, n-1)
	case ebiten.KeyLeft:
		sel = max(sel-perCol, 0)
	case ebiten.KeyRight:
		sel = min(sel+perCol, n-1)
	case ebiten.KeyHome:
		sel = 0
	case ebiten.KeyEnd:
		sel = n - 1
	default:
		return false
	}

	sel = max(sel, 0)
	if sel != lv.selected {
		lv.selected = sel
		lv.ensureVisible(sel)
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

// listBox is a TListBox: lines of text in a sunken white box, one of them
// selected, scrolling up and down.
type listBox struct {
	control
	items    []string
	selected int       // ItemIndex
	scroll   scrollBar // vertical, in items; its position is the first shown
	pressing bool      // the mouse is choosing an item
	onSelect func()    // the selection changed, by mouse or keyboard
}

func (lbx *listBox) setup(comp *layout.Component) {
	lbx.windowed = true
	lbx.tabStop = lbx.property("TabStop") != "False"
	lbx.color = vclColor(lbx.property("Color"), colWindow)
	lbx.items = slices.Clone(comp.Items)
	lbx.selected = lbx.propertyInt("ItemIndex", -1)
	lbx.scroll = scrollBar{
		vertical: true,
		onScroll: lbx.invalidate,
	}
	lbx.layout()
}

func (lbx *listBox) focusable() bool {
	return true
}

// itemHeight is a line's height: the font's, as ItemHeight is.
func (lbx *listBox) itemHeight() int {
	return lbx.font.height()
}

// client is the rectangle inside the border, left of the scroll bar.
func (lbx *listBox) client() image.Rectangle {
	rect := lbx.local().Inset(2)
	if lbx.scroll.rect != (image.Rectangle{}) {
		rect.Max.X -= scrollSize
	}

	return rect
}

// rows is how many whole lines show.
func (lbx *listBox) rows() int {
	return max(lbx.local().Inset(2).Dy()/lbx.itemHeight(), 1)
}

// layout shows the scroll bar when the items don't fit.
func (lbx *listBox) layout() {
	in := lbx.local().Inset(2)
	bar := &lbx.scroll
	bar.rect = image.Rectangle{}
	bar.min, bar.max, bar.page = 0, max(len(lbx.items)-1, 0), lbx.rows()

	if len(lbx.items) > bar.page {
		bar.rect = image.Rect(in.Max.X-scrollSize, in.Min.Y, in.Max.X, in.Max.Y)
	}

	bar.set(bar.pos)
}

// setIndex selects item i, or none for -1, without running OnSelect.
func (lbx *listBox) setIndex(i int) {
	if lbx.selected != i {
		lbx.selected = i
		lbx.ensureVisible(i)
		lbx.invalidate()
	}
}

// choose selects item i as the player does, running OnSelect when the
// selection changes.
func (lbx *listBox) choose(i int) {
	if i < 0 || i >= len(lbx.items) || i == lbx.selected {
		return
	}

	lbx.setIndex(i)

	if lbx.onSelect != nil {
		lbx.onSelect()
	}
}

// itemRect is line i's rectangle, which may be out of view.
func (lbx *listBox) itemRect(i int) image.Rectangle {
	cli := lbx.client()
	y := cli.Min.Y + (i-lbx.scroll.pos)*lbx.itemHeight()

	return image.Rect(cli.Min.X, y, cli.Max.X, y+lbx.itemHeight())
}

// at returns the item at pnt, or -1.
func (lbx *listBox) at(pnt image.Point) int {
	if !pnt.In(lbx.client()) {
		return -1
	}

	i := lbx.scroll.pos + (pnt.Y-lbx.client().Min.Y)/lbx.itemHeight()
	if i >= len(lbx.items) {
		return -1
	}

	return i
}

// ensureVisible scrolls item i into view.
func (lbx *listBox) ensureVisible(i int) {
	switch bar := &lbx.scroll; {
	case i < 0:
	case i < bar.pos:
		bar.set(i)
	case i >= bar.pos+bar.page:
		bar.set(i - bar.page + 1)
	}
}

func (lbx *listBox) paint(cv canvas) {
	cv.sunken(lbx.local())

	cli := lbx.client()
	cv.fill(cli, lbx.color)

	in := cv.sub(cli)
	in.org = cv.org // keep the list's coordinates, clipped to the inside
	// The focus rectangle goes round the selected line, or the first.
	current := max(lbx.selected, 0)
	focus := lbx.focused() && lbx.wdw.active

	for i, item := range lbx.items {
		ir := lbx.itemRect(i)
		if !ir.Overlaps(cli) {
			continue
		}

		fg := lbx.fontColor
		if i == lbx.selected {
			in.fill(ir, colSelection)

			fg = colSelText
		}

		drawText(in, lbx.font, ir.Min.X+2, ir.Min.Y, item, fg, lbx.usable())

		if focus && i == current {
			in.focusRect(ir)
		}
	}

	if lbx.scroll.rect != (image.Rectangle{}) {
		lbx.scroll.paint(cv)
	}
}

func (lbx *listBox) mouseDown(pnt image.Point, _ bool) {
	if lbx.scroll.press(pnt) {
		lbx.invalidate()

		return
	}

	lbx.pressing = true
	lbx.choose(lbx.at(pnt))
}

// mouseDrag moves the scroll bar's thumb, or the selection with the
// mouse, as a list box follows a held button.
func (lbx *listBox) mouseDrag(pnt image.Point) {
	if lbx.scroll.pressed != partNone {
		lbx.scroll.drag(pnt)
		lbx.invalidate()

		return
	}

	if lbx.pressing {
		lbx.choose(lbx.at(pnt))
	}
}

func (lbx *listBox) mouseUp(image.Point) {
	lbx.pressing = false

	if lbx.scroll.pressed != partNone {
		lbx.scroll.release()
		lbx.invalidate()
	}
}

func (lbx *listBox) tick(now time.Time) {
	if lbx.scroll.tick(now) {
		lbx.invalidate()
	}
}

func (lbx *listBox) key(key ebiten.Key, _ mods) bool {
	n := len(lbx.items)
	if n == 0 {
		return false
	}

	sel := lbx.selected

	switch key {
	case ebiten.KeyUp:
		sel--
	case ebiten.KeyDown:
		sel++
	case ebiten.KeyPageUp:
		sel -= lbx.rows() - 1
	case ebiten.KeyPageDown:
		sel += lbx.rows() - 1
	case ebiten.KeyHome:
		sel = 0
	case ebiten.KeyEnd:
		sel = n - 1
	default:
		return false
	}

	lbx.choose(max(min(sel, n-1), 0))

	return true
}

func (lbx *listBox) char(rune) {}

func (lbx *listBox) wheel(dy int) {
	if lbx.scroll.rect != (image.Rectangle{}) {
		lbx.scroll.set(lbx.scroll.pos - dy)
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

func newDropList(cb *comboBox) *dropList {
	dl := &dropList{
		combo: cb,
		hot:   cb.index,
	}
	dl.rows = min(len(cb.items), 8)
	hgt := max(dl.rows, 1)*cb.itemHeight + 2
	pnt := cb.wdw.clientOrigin().Add(cb.origin())
	dl.rect = image.Rect(pnt.X, pnt.Y+cb.rect.Dy(), pnt.X+cb.rect.Dx(), pnt.Y+cb.rect.Dy()+hgt)

	if dl.rect.Max.Y > screenHeight {
		dl.rect = dl.rect.Sub(image.Pt(0, hgt+cb.rect.Dy()))
	}

	if len(cb.items) > dl.rows {
		dl.scroll = scrollBar{
			vertical: true,
			rect:     image.Rect(dl.rect.Dx()-1-scrollSize, 1, dl.rect.Dx()-1, dl.rect.Dy()-1),
			min:      0,
			max:      len(cb.items) - 1,
			page:     dl.rows,
		}
	}

	dl.show(max(cb.index, 0))

	dl.scroll.onScroll = func() {
		dl.top = dl.scroll.pos
	}

	return dl
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
	rect := image.Rectangle{Max: dl.rect.Size()}
	cv.frame(rect, colFrame)

	in := rect.Inset(1)
	if dl.scroll.max > 0 {
		in.Max.X -= scrollSize
	}

	cv.fill(in, colWindow)

	cb := dl.combo
	for row := range dl.rows {
		i := dl.top + row
		if i >= len(cb.items) {
			break
		}

		y := in.Min.Y + row*cb.itemHeight
		ir := image.Rect(in.Min.X, y, in.Max.X, y+cb.itemHeight)
		tx := 2

		if cb.images != nil {
			// Owner-drawn: the item's icon at its indent, then its text,
			// highlighted on its own.
			x := ir.Min.X + 1 + cb.indents[i]
			cb.images.drawFrame(cv, x, y+(cb.itemHeight-cb.images.size)/2, cb.icons[i], false)

			x += cb.images.size + 3
			ir = image.Rect(x, y, min(x+cb.font.width(cb.items[i])+4, in.Max.X), y+cb.itemHeight)
		}

		fg := colText

		if i == dl.hot {
			cv.fill(ir, colSelection)

			fg = colSelText
		}

		cb.font.draw(cv.sub(ir), tx, (cb.itemHeight-cb.font.height())/2, cb.items[i], fg)
	}

	if dl.scroll.max > 0 {
		dl.scroll.paint(cv)
	}
}

// itemAt returns the item at pnt (on the desktop), or -1.
func (dl *dropList) itemAt(pnt image.Point) int {
	loc := pnt.Sub(dl.rect.Min)
	in := image.Rectangle{Max: dl.rect.Size()}.Inset(1)

	if dl.scroll.max > 0 {
		in.Max.X -= scrollSize
	}

	if !loc.In(in) {
		return -1
	}

	i := dl.top + (loc.Y-in.Min.Y)/dl.combo.itemHeight
	if i >= len(dl.combo.items) {
		return -1
	}

	return i
}

func (dl *dropList) mouseDown(pnt image.Point) {
	if dl.scroll.press(pnt.Sub(dl.rect.Min)) {
		return
	}
}

func (dl *dropList) mouseMove(pnt image.Point, down bool) {
	if dl.scroll.pressed != partNone {
		dl.scroll.drag(pnt.Sub(dl.rect.Min))

		return
	}

	if i := dl.itemAt(pnt); i >= 0 {
		dl.hot = i
	}
}

func (dl *dropList) mouseUp(pnt image.Point) bool {
	if dl.scroll.pressed != partNone {
		dl.scroll.release()

		return false
	}

	if i := dl.itemAt(pnt); i >= 0 {
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

func (dl *dropList) key(key ebiten.Key) bool {
	n := len(dl.combo.items)

	switch key {
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
