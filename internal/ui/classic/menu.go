// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// The menus are the VCL's owner-drawn menus, as a main menu with Images
// draws them on Windows 98.
const (
	menuItemHeight int           = 19 // the image's 16 pixels and 3
	menuFrame      int           = 3  // a raised edge and a line of face
	menuShowDelay  time.Duration = 400 * time.Millisecond
)

// menuEntry is a menu item as the classic UI draws it; its flags are the
// game's.
type menuEntry struct {
	item     *game.MenuItem
	caption  string // with the accelerator AutoHotkeys gives it
	image    int    // ImageIndex, or -1
	shortcut string // the shortcut's text, as "F1"
	key      ebiten.Key
	kids     []*menuEntry
}

func (me *menuEntry) name() string {
	return me.item.Name
}

// visibleKids lists the submenu's visible items.
func (me *menuEntry) visibleKids() []*menuEntry {
	var out []*menuEntry

	for _, k := range me.kids {
		if k.item.Visible {
			out = append(out, k)
		}
	}

	return out
}

// menuPopup is an open submenu.
type menuPopup struct {
	entry *menuEntry
	items []*menuEntry
	rect  image.Rectangle // on the screen
	sel   int             // the selected item, or -1
	child int             // the item whose submenu is open, or -1
	// The submenu to open once the mouse has rested on its item.
	pending   int
	pendingAt time.Time
}

func (mp *menuPopup) itemRect(i int) image.Rectangle {
	y := mp.rect.Min.Y + menuFrame + i*menuItemHeight

	return image.Rect(mp.rect.Min.X+menuFrame, y, mp.rect.Max.X-menuFrame, y+menuItemHeight)
}

func (mp *menuPopup) itemAt(q image.Point) int {
	for i := range mp.items {
		if q.In(mp.itemRect(i)) {
			return i
		}
	}

	return -1
}

// opensSubmenu reports whether the popup's selected item opens a submenu.
func (mp *menuPopup) opensSubmenu() bool {
	return mp.sel >= 0 && len(mp.items[mp.sel].kids) > 0 && mp.items[mp.sel].item.Enabled
}

// menuBar is a window's main menu: the bar under its caption and, while
// it tracks the mouse or the keyboard (menu mode), its open submenus.
type menuBar struct {
	win     *window
	res     *resources
	entries []*menuEntry
	images  *imageList
	// onCommand runs a command once the menus have closed.
	onCommand func(e *menuEntry)

	hot     int          // the bar item under the mouse, outside menu mode
	sel     int          // the selected bar item in menu mode, or -1
	open    []*menuPopup // the open submenus, outermost first
	pressed int          // the bar item the mouse pressed, or -1
}

// newMenuBar builds a window's menu from the game's menu and the form's
// design: the images and the shortcuts.
func newMenuBar(w *window, items []*game.MenuItem, design *layout.Component) *menuBar {
	m := &menuBar{
		win:     w,
		res:     w.res,
		hot:     -1,
		sel:     -1,
		pressed: -1,
	}
	m.images = w.res.imageList(design.Props["Images"])
	props := map[string]map[string]string{}

	design.Walk(func(c *layout.Component) {
		props[c.Name] = c.Props
	})

	var build func(items []*game.MenuItem) []*menuEntry
	build = func(items []*game.MenuItem) []*menuEntry {
		out := make([]*menuEntry, 0, len(items))

		for _, it := range items {
			e := &menuEntry{
				item:    it,
				caption: it.Caption,
				image:   -1,
				key:     -1,
			}
			if v, err := strconv.Atoi(props[it.Name]["ImageIndex"]); err == nil {
				e.image = v
			}

			if v, err := strconv.Atoi(props[it.Name]["ShortCut"]); err == nil {
				e.key, e.shortcut = shortcutKey(v)
			}

			e.kids = build(it.Items)
			out = append(out, e)
		}

		autoHotkeys(out)

		return out
	}
	m.entries = build(items)
	w.menu = m

	return m
}

// shortcutKey decodes a TShortCut without modifiers: a function key.
func shortcutKey(v int) (key ebiten.Key, name string) {
	const vkF1 int = 0x70
	if vk := v & 0xff; v < 0x100 && vk >= vkF1 && vk < vkF1+12 {
		n := vk - vkF1

		return ebiten.KeyF1 + ebiten.Key(n), "F" + strconv.Itoa(n+1)
	}

	return -1, ""
}

// autoHotkeys gives the items of a menu accelerators, as the VCL's
// AutoHotkeys does at run time: in order, each item without one takes the
// first letter or digit of its caption that no other item has taken.
func autoHotkeys(entries []*menuEntry) {
	used := map[byte]bool{}

	for _, e := range entries {
		if a := accel(e.caption); a != 0 && a < 0x80 {
			used[strings.ToUpper(string(a))[0]] = true
		}
	}

	for _, e := range entries {
		if accel(e.caption) != 0 {
			continue
		}

		s := e.caption
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}

			if isAccelChar(c) && !used[c] {
				used[c] = true
				e.caption = s[:i] + "&" + s[i:]

				break
			}
		}
	}
}

// isAccelChar reports whether c can be a menu item's accelerator: an
// upper-case letter or a digit.
func isAccelChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// barRects returns the bar items' rectangles on the screen; a hidden
// item's is empty.
func (mb *menuBar) barRects() []image.Rectangle {
	r := mb.win.menuRect()
	x := r.Min.X
	out := make([]image.Rectangle, len(mb.entries))

	for i, e := range mb.entries {
		if !e.item.Visible {
			continue
		}

		w := mb.res.sans.width(plain(e.caption)) + 2*mb.res.charWidth
		out[i] = image.Rect(x, r.Min.Y, x+w, r.Min.Y+menuHeight-1)
		x += w
	}

	return out
}

func (mb *menuBar) barAt(p image.Point) int {
	for i, r := range mb.barRects() {
		if p.In(r) {
			return i
		}
	}

	return -1
}

// active reports whether the menu is in menu mode.
func (mb *menuBar) active() bool {
	return mb.win.desk.menu == mb
}

func (mb *menuBar) paint(cv canvas, r image.Rectangle) {
	cv.fill(r, colFace)

	f := mb.res.sans
	for i, ir := range mb.barRects() {
		if ir.Empty() {
			continue
		}

		e := mb.entries[i]

		onSel := mb.active() && i == mb.sel
		switch {
		case onSel && len(mb.open) > 0, onSel && mb.pressed == i:
			cv.edge(ir, colShadow, colHighlight)
		case onSel, !mb.active() && i == mb.hot && mb.win.active:
			cv.edge(ir, colHighlight, colShadow)
		}

		x := ir.Min.X + mb.res.charWidth
		y := ir.Min.Y + (ir.Dy()-f.height())/2

		drawText(cv, f, x, y, e.caption, colText, e.item.Enabled)
	}
}

// hover hot-tracks the bar item under the mouse.
func (mb *menuBar) hover(p image.Point, inside bool) {
	i := -1
	if inside {
		i = mb.barAt(p)
	}

	if i != mb.hot {
		mb.hot = i
		mb.win.desk.dirty = true
	}
}

// popupFor lays out the submenu of e at (x, y), kept on the screen; flip
// gives the x to use when it doesn't fit on the right.
func (mb *menuBar) popupFor(e *menuEntry, x, y, flip int) *menuPopup {
	p := &menuPopup{
		entry:   e,
		items:   e.visibleKids(),
		sel:     -1,
		child:   -1,
		pending: -1,
	}
	w := 0

	for _, k := range p.items {
		w = max(w, 16+15+mb.res.sans.width(plain(k.caption)+k.shortcut)+7+2*mb.res.charWidth)
	}

	size := image.Pt(w+2*menuFrame, len(p.items)*menuItemHeight+2*menuFrame)
	if x+size.X > screenWidth {
		x = flip - size.X
	}

	x = max(min(x, screenWidth-size.X), 0)
	y = max(min(y, screenHeight-size.Y), 0)
	p.rect = image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+size.X, y+size.Y)}

	return p
}

// enter starts menu mode with bar item i selected and, if open, its
// submenu shown.
func (mb *menuBar) enter(i int, open bool) {
	d := mb.win.desk
	d.menu = mb
	mb.hot = -1
	mb.selectBar(i, open)
}

// enterKeyboard starts menu mode from F10 or Alt: the first item is
// selected and nothing opens.
func (mb *menuBar) enterKeyboard() {
	mb.enter(mb.nextBar(-1, 1), false)
}

// selectBar selects bar item i, closing the open submenus, and opens its
// own if asked to and it is enabled.
func (mb *menuBar) selectBar(i int, open bool) {
	mb.sel = i
	mb.open = nil
	mb.win.desk.dirty = true

	if !open || i < 0 {
		return
	}

	e := mb.entries[i]
	if !e.item.Enabled || len(e.visibleKids()) == 0 {
		return
	}

	r := mb.barRects()[i]
	mb.open = []*menuPopup{mb.popupFor(e, r.Min.X, r.Max.Y+1, r.Max.X)}
}

// nextBar returns the next visible bar item after i in direction dir.
func (mb *menuBar) nextBar(i, dir int) int {
	n := len(mb.entries)
	for range n {
		i = (i + dir + n) % n
		if mb.entries[i].item.Visible {
			return i
		}
	}

	return -1
}

// openChild opens the submenu of item i of the popup at level l.
func (mb *menuBar) openChild(l, i int) {
	mb.open = mb.open[:l+1]
	p := mb.open[l]
	p.child, p.pending = -1, -1
	e := p.items[i]

	if !e.item.Enabled || len(e.visibleKids()) == 0 {
		return
	}

	r := p.itemRect(i)
	c := mb.popupFor(e, p.rect.Max.X-menuFrame, r.Min.Y-menuFrame, p.rect.Min.X+menuFrame)
	p.child = i

	mb.open = append(mb.open, c)
	mb.win.desk.dirty = true
}

// closeFrom closes the popups from level l on.
func (mb *menuBar) closeFrom(l int) {
	if l < len(mb.open) {
		mb.open = mb.open[:l]
		if l > 0 {
			mb.open[l-1].child = -1
		}

		mb.win.desk.dirty = true
	}
}

// exit leaves menu mode.
func (mb *menuBar) exit() {
	mb.open = nil
	mb.sel, mb.pressed = -1, -1

	if d := mb.win.desk; d.menu == mb {
		d.menu = nil
		d.dirty = true
	}
}

// run leaves menu mode and runs a command.
func (mb *menuBar) run(e *menuEntry) {
	mb.exit()

	if mb.onCommand != nil {
		mb.onCommand(e)
	}
}

// popupAt returns the level and item under p, or -1 for the level.
func (mb *menuBar) popupAt(p image.Point) (level, item int) {
	for l, v := range slices.Backward(mb.open) {
		if p.In(v.rect) {
			return l, v.itemAt(p)
		}
	}

	return -1, -1
}

func (mb *menuBar) mouseDown(p image.Point) {
	if l, i := mb.popupAt(p); l >= 0 {
		mb.pressed = -1
		mb.selectItem(l, i)

		if i >= 0 && len(mb.open[l].items[i].kids) > 0 {
			mb.openChild(l, i)
		}

		return
	}

	if p.In(mb.win.menuRect()) {
		i := mb.barAt(p)
		if i < 0 {
			mb.exit()

			return
		}

		if i == mb.sel && len(mb.open) > 0 {
			mb.exit() // a second click on the item closes its menu

			return
		}

		mb.enter(i, true)

		mb.pressed = i

		return
	}

	// Outside: the menus close and the click goes on to what is there.
	mb.exit()
	mb.win.desk.mouseDown(p)
}

// selectItem selects item i of the popup at level l, closing what is
// open below it if the item changes.
func (mb *menuBar) selectItem(l, i int) {
	p := mb.open[l]
	if p.sel == i {
		return
	}

	p.sel = i
	if p.child >= 0 && p.child != i {
		mb.closeFrom(l + 1)
	}

	p.pending = -1
	if i >= 0 && len(p.items[i].kids) > 0 && p.child != i {
		p.pending, p.pendingAt = i, mb.win.desk.now
	}

	mb.win.desk.dirty = true
}

func (mb *menuBar) mouseMove(p image.Point, down bool) {
	if l, i := mb.popupAt(p); l >= 0 {
		if i >= 0 || mb.open[l].child < 0 {
			mb.selectItem(l, i)
		}

		return
	}

	if p.In(mb.win.menuRect()) {
		// With a menu open, or the button down, the selection follows the mouse.
		following := len(mb.open) > 0 || down
		if i := mb.barAt(p); i >= 0 && i != mb.sel && following {
			mb.selectBar(i, true)

			if down {
				mb.pressed = i
			}
		}

		return
	}

	// Outside every menu: the deepest one loses its selection, unless it
	// has a submenu open.
	if n := len(mb.open); n > 0 && mb.open[n-1].child < 0 && mb.open[n-1].sel >= 0 {
		mb.open[n-1].sel = -1
		mb.open[n-1].pending = -1
		mb.win.desk.dirty = true
	}
}

func (mb *menuBar) mouseUp(p image.Point) {
	pressed := mb.pressed
	mb.pressed = -1

	if l, i := mb.popupAt(p); l >= 0 && i >= 0 {
		e := mb.open[l].items[i]
		if len(e.kids) == 0 && e.item.Enabled {
			mb.run(e)
		}

		return
	}

	if pressed >= 0 && len(mb.open) == 0 && mb.barAt(p) == pressed {
		// A disabled or empty menu: the click ends menu mode.
		mb.exit()
	}

	mb.win.desk.dirty = true
}

// tick opens a submenu once the mouse has rested on its item.
func (mb *menuBar) tick(now time.Time) {
	for l, p := range mb.open {
		if p.pending >= 0 && now.Sub(p.pendingAt) >= menuShowDelay {
			mb.openChild(l, p.pending)

			return
		}
	}
}

func (mb *menuBar) char(rune) {}

// key handles a key in menu mode.
func (mb *menuBar) key(k ebiten.Key) {
	d := mb.win.desk
	d.dirty = true
	n := len(mb.open)

	var p *menuPopup

	if n > 0 {
		p = mb.open[n-1]
	}

	switch k {
	case ebiten.KeyEscape:
		if n > 0 {
			mb.closeFrom(n - 1)
		} else {
			mb.exit()
		}
	case ebiten.KeyF10:
		mb.exit()
	case ebiten.KeyLeft:
		if n > 1 {
			mb.closeFrom(n - 1)
		} else {
			mb.selectBar(mb.nextBar(mb.sel, -1), n > 0)
			mb.selectFirst()
		}
	case ebiten.KeyRight:
		if p != nil && p.opensSubmenu() {
			mb.openChild(n-1, p.sel)
		} else {
			mb.selectBar(mb.nextBar(mb.sel, 1), n > 0)
		}

		mb.selectFirst()
	case ebiten.KeyUp:
		mb.upDown(p, -1)
	case ebiten.KeyDown:
		mb.upDown(p, 1)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if p == nil {
			mb.selectBar(mb.sel, true)
			mb.selectFirst()

			return
		}

		mb.activateItem(n - 1)
	default:
		if r := keyRune(k); r != 0 {
			mb.letter(p, r)
		}
	}
}

// upDown moves the selection of the deepest open popup p by dir, round
// its items, or drops the bar item's menu down when no popup is open.
func (mb *menuBar) upDown(p *menuPopup, dir int) {
	if p == nil {
		mb.selectBar(mb.sel, true)
		mb.selectFirst()

		return
	}

	if c := len(p.items); c > 0 {
		i := p.sel
		if i < 0 && dir < 0 {
			i = 0
		}

		mb.selectItem(len(mb.open)-1, (i+dir+c)%c)

		p.pending = -1
	}
}

// letter selects the item whose accelerator r is: on the bar, which drops
// its menu down, while no popup is open, or else in the deepest open popup
// p, where it also activates it.
func (mb *menuBar) letter(p *menuPopup, r rune) {
	if p == nil {
		for i, e := range mb.entries {
			if e.item.Visible && accel(e.caption) == r {
				mb.selectBar(i, true)
				mb.selectFirst()

				return
			}
		}

		return
	}

	for i, e := range p.items {
		if accel(e.caption) == r {
			l := len(mb.open) - 1
			mb.selectItem(l, i)
			mb.activateItem(l)

			return
		}
	}
}

// selectFirst selects the first item of the deepest open submenu.
func (mb *menuBar) selectFirst() {
	if n := len(mb.open); n > 0 && len(mb.open[n-1].items) > 0 {
		mb.selectItem(n-1, 0)

		mb.open[n-1].pending = -1
	}
}

// activateItem is Enter on the selected item of the popup at level l: a
// submenu opens, a command runs, a disabled item closes the menus.
func (mb *menuBar) activateItem(l int) {
	p := mb.open[l]
	if p.sel < 0 {
		return
	}

	e := p.items[p.sel]
	switch {
	case len(e.kids) > 0:
		mb.openChild(l, p.sel)
		mb.selectFirst()
	case e.item.Enabled:
		mb.run(e)
	default:
		mb.exit()
	}
}

// openByKey opens the bar item whose accelerator the key is (Alt and a
// letter); it reports whether there is one.
func (mb *menuBar) openByKey(k ebiten.Key) bool {
	r := keyRune(k)
	if r == 0 {
		return false
	}

	for i, e := range mb.entries {
		if e.item.Visible && accel(e.caption) == r {
			mb.enter(i, true)
			mb.selectFirst()

			return true
		}
	}

	return false
}

// shortcut runs the command whose shortcut the key is, if it and the items
// above it are enabled, as TMenu.IsShortCut does; hidden items count.
func (mb *menuBar) shortcut(k ebiten.Key, mm mods) bool {
	if mm.ctrl || mm.alt || mm.shift {
		return false
	}

	var find func(es []*menuEntry) *menuEntry
	find = func(es []*menuEntry) *menuEntry {
		for _, e := range es {
			if !e.item.Enabled {
				continue
			}

			if e.key == k && len(e.kids) == 0 {
				return e
			}

			if f := find(e.kids); f != nil {
				return f
			}
		}

		return nil
	}
	e := find(mb.entries)

	if e == nil {
		return false
	}

	mb.run(e)

	return true
}

// The menus' glyphs: the check mark and the submenu arrow.
const (
	menuCheck string = `
......#
.....##
#...###
##.###.
#####..
.###...
..#....
`
	menuArrow string = `
#...
##..
###.
####
###.
##..
#...
`
)

// paintPopups draws the open submenus.
func (mb *menuBar) paintPopups(cv canvas) {
	for _, p := range mb.open {
		cv.fill(p.rect, colFace)
		cv.raised(p.rect)

		for i, e := range p.items {
			mb.paintItem(cv, p.itemRect(i), e, i == p.sel)
		}
	}
}

// paintItem draws an item of a submenu, as TMenuItem.AdvancedDrawItem
// draws it, and the arrow Windows adds for a submenu.
func (mb *menuBar) paintItem(cv canvas, r image.Rectangle, e *menuEntry, sel bool) {
	f := mb.res.sans
	fg := colText

	if sel {
		cv.fill(r, colSelection)

		fg = colSelText
	}

	enabled := e.item.Enabled
	switch {
	case e.image >= 0 && mb.images != nil && enabled:
		mb.images.drawFrame(cv, r.Min.X+1, r.Min.Y+1, e.image, false)
	case e.image >= 0 && mb.images != nil:
		mb.images.drawDisabled(cv, r.Min.X+1, r.Min.Y+1, e.image)
	case e.item.Checked:
		cv.pattern(r.Min.X+6, r.Min.Y+6, menuCheck, map[byte]color.RGBA{'#': fg})
	}

	x, y := r.Min.X+22, r.Min.Y+(r.Dy()-f.height())/2
	text := func(x int, s string) {
		switch {
		case enabled:
			drawCaption(cv, f, x, y, s, fg)
		case sel:
			drawCaption(cv, f, x, y, s, colGrayText)
		default:
			drawDisabled(cv, f, x, y, s)
		}
	}
	text(x, e.caption)

	if e.shortcut != "" {
		text(r.Max.X-10-f.width(e.shortcut), e.shortcut)
	}

	if len(e.kids) > 0 {
		cv.pattern(r.Max.X-8, r.Min.Y+(r.Dy()-7)/2, menuArrow, map[byte]color.RGBA{'#': fg})
	}
}
