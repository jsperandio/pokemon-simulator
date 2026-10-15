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

	for _, kid := range me.kids {
		if kid.item.Visible {
			out = append(out, kid)
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

func (mp *menuPopup) itemAt(pnt image.Point) int {
	for i := range mp.items {
		if pnt.In(mp.itemRect(i)) {
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
	onCommand func(me *menuEntry)

	hot     int          // the bar item under the mouse, outside menu mode
	sel     int          // the selected bar item in menu mode, or -1
	open    []*menuPopup // the open submenus, outermost first
	pressed int          // the bar item the mouse pressed, or -1
}

// newMenuBar builds a window's menu from the game's menu and the form's
// design: the images and the shortcuts.
func newMenuBar(wdw *window, items []*game.MenuItem, design *layout.Component) *menuBar {
	mb := &menuBar{
		win:     wdw,
		res:     wdw.res,
		hot:     -1,
		sel:     -1,
		pressed: -1,
	}
	mb.images = wdw.res.imageList(design.Props["Images"])
	props := map[string]map[string]string{}

	design.Walk(func(comp *layout.Component) {
		props[comp.Name] = comp.Props
	})

	var build func(items []*game.MenuItem) []*menuEntry
	build = func(items []*game.MenuItem) []*menuEntry {
		out := make([]*menuEntry, 0, len(items))

		for _, it := range items {
			me := &menuEntry{
				item:    it,
				caption: it.Caption,
				image:   -1,
				key:     -1,
			}
			if val, err := strconv.Atoi(props[it.Name]["ImageIndex"]); err == nil {
				me.image = val
			}

			if val, err := strconv.Atoi(props[it.Name]["ShortCut"]); err == nil {
				me.key, me.shortcut = shortcutKey(val)
			}

			me.kids = build(it.Items)
			out = append(out, me)
		}

		autoHotkeys(out)

		return out
	}
	mb.entries = build(items)
	wdw.menu = mb

	return mb
}

// shortcutKey decodes a TShortCut without modifiers: a function key.
func shortcutKey(val int) (key ebiten.Key, name string) {
	const vkF1 int = 0x70
	if vk := val & 0xff; val < 0x100 && vk >= vkF1 && vk < vkF1+12 {
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

	for _, me := range entries {
		if chr := accel(me.caption); chr != 0 && chr < 0x80 {
			used[strings.ToUpper(string(chr))[0]] = true
		}
	}

	for _, me := range entries {
		if accel(me.caption) != 0 {
			continue
		}

		txt := me.caption
		for i := 0; i < len(txt); i++ {
			chr := txt[i]
			if chr >= 'a' && chr <= 'z' {
				chr -= 'a' - 'A'
			}

			if isAccelChar(chr) && !used[chr] {
				used[chr] = true
				me.caption = txt[:i] + "&" + txt[i:]

				break
			}
		}
	}
}

// isAccelChar reports whether chr can be a menu item's accelerator: an
// upper-case letter or a digit.
func isAccelChar(chr byte) bool {
	return (chr >= 'A' && chr <= 'Z') || (chr >= '0' && chr <= '9')
}

// barRects returns the bar items' rectangles on the screen; a hidden
// item's is empty.
func (mb *menuBar) barRects() []image.Rectangle {
	rect := mb.win.menuRect()
	x := rect.Min.X
	out := make([]image.Rectangle, len(mb.entries))

	for i, me := range mb.entries {
		if !me.item.Visible {
			continue
		}

		wid := mb.res.sans.width(plain(me.caption)) + 2*mb.res.charWidth
		out[i] = image.Rect(x, rect.Min.Y, x+wid, rect.Min.Y+menuHeight-1)
		x += wid
	}

	return out
}

func (mb *menuBar) barAt(pnt image.Point) int {
	for i, rect := range mb.barRects() {
		if pnt.In(rect) {
			return i
		}
	}

	return -1
}

// active reports whether the menu is in menu mode.
func (mb *menuBar) active() bool {
	return mb.win.desk.menu == mb
}

func (mb *menuBar) paint(cv canvas, rect image.Rectangle) {
	cv.fill(rect, colFace)

	fnt := mb.res.sans
	for i, ir := range mb.barRects() {
		if ir.Empty() {
			continue
		}

		me := mb.entries[i]

		onSel := mb.active() && i == mb.sel
		switch {
		case onSel && len(mb.open) > 0, onSel && mb.pressed == i:
			cv.edge(ir, colShadow, colHighlight)
		case onSel, !mb.active() && i == mb.hot && mb.win.active:
			cv.edge(ir, colHighlight, colShadow)
		}

		x := ir.Min.X + mb.res.charWidth
		y := ir.Min.Y + (ir.Dy()-fnt.height())/2

		drawText(cv, fnt, x, y, me.caption, colText, me.item.Enabled)
	}
}

// hover hot-tracks the bar item under the mouse.
func (mb *menuBar) hover(pnt image.Point, inside bool) {
	i := -1
	if inside {
		i = mb.barAt(pnt)
	}

	if i != mb.hot {
		mb.hot = i
		mb.win.desk.dirty = true
	}
}

// popupFor lays out the submenu of me at (x, y), kept on the screen; flip
// gives the x to use when it doesn't fit on the right.
func (mb *menuBar) popupFor(me *menuEntry, x, y, flip int) *menuPopup {
	mp := &menuPopup{
		entry:   me,
		items:   me.visibleKids(),
		sel:     -1,
		child:   -1,
		pending: -1,
	}
	wid := 0

	for _, kid := range mp.items {
		wid = max(wid, 16+15+mb.res.sans.width(plain(kid.caption)+kid.shortcut)+7+2*mb.res.charWidth)
	}

	size := image.Pt(wid+2*menuFrame, len(mp.items)*menuItemHeight+2*menuFrame)
	if x+size.X > screenWidth {
		x = flip - size.X
	}

	x = max(min(x, screenWidth-size.X), 0)
	y = max(min(y, screenHeight-size.Y), 0)
	mp.rect = image.Rectangle{
		Min: image.Pt(x, y),
		Max: image.Pt(x+size.X, y+size.Y),
	}

	return mp
}

// enter starts menu mode with bar item i selected and, if open, its
// submenu shown.
func (mb *menuBar) enter(i int, open bool) {
	dsk := mb.win.desk
	dsk.menu = mb
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

	me := mb.entries[i]
	if !me.item.Enabled || len(me.visibleKids()) == 0 {
		return
	}

	rect := mb.barRects()[i]
	mb.open = []*menuPopup{mb.popupFor(me, rect.Min.X, rect.Max.Y+1, rect.Max.X)}
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

// openChild opens the submenu of item i of the popup at level lvl.
func (mb *menuBar) openChild(lvl, i int) {
	mb.open = mb.open[:lvl+1]
	par := mb.open[lvl]
	par.child, par.pending = -1, -1
	me := par.items[i]

	if !me.item.Enabled || len(me.visibleKids()) == 0 {
		return
	}

	rect := par.itemRect(i)
	chd := mb.popupFor(me, par.rect.Max.X-menuFrame, rect.Min.Y-menuFrame, par.rect.Min.X+menuFrame)
	par.child = i

	mb.open = append(mb.open, chd)
	mb.win.desk.dirty = true
}

// closeFrom closes the popups from level lvl on.
func (mb *menuBar) closeFrom(lvl int) {
	if lvl < len(mb.open) {
		mb.open = mb.open[:lvl]
		if lvl > 0 {
			mb.open[lvl-1].child = -1
		}

		mb.win.desk.dirty = true
	}
}

// exit leaves menu mode.
func (mb *menuBar) exit() {
	mb.open = nil
	mb.sel, mb.pressed = -1, -1

	if dsk := mb.win.desk; dsk.menu == mb {
		dsk.menu = nil
		dsk.dirty = true
	}
}

// run leaves menu mode and runs a command.
func (mb *menuBar) run(me *menuEntry) {
	mb.exit()

	if mb.onCommand != nil {
		mb.onCommand(me)
	}
}

// popupAt returns the level and item under pnt, or -1 for the level.
func (mb *menuBar) popupAt(pnt image.Point) (level, item int) {
	for lvl, mp := range slices.Backward(mb.open) {
		if pnt.In(mp.rect) {
			return lvl, mp.itemAt(pnt)
		}
	}

	return -1, -1
}

func (mb *menuBar) mouseDown(pnt image.Point) {
	if lvl, i := mb.popupAt(pnt); lvl >= 0 {
		mb.pressed = -1
		mb.selectItem(lvl, i)

		if i >= 0 && len(mb.open[lvl].items[i].kids) > 0 {
			mb.openChild(lvl, i)
		}

		return
	}

	if pnt.In(mb.win.menuRect()) {
		i := mb.barAt(pnt)
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
	mb.win.desk.mouseDown(pnt)
}

// selectItem selects item i of the popup at level lvl, closing what is
// open below it if the item changes.
func (mb *menuBar) selectItem(lvl, i int) {
	mp := mb.open[lvl]
	if mp.sel == i {
		return
	}

	mp.sel = i
	if mp.child >= 0 && mp.child != i {
		mb.closeFrom(lvl + 1)
	}

	mp.pending = -1
	if i >= 0 && len(mp.items[i].kids) > 0 && mp.child != i {
		mp.pending, mp.pendingAt = i, mb.win.desk.now
	}

	mb.win.desk.dirty = true
}

func (mb *menuBar) mouseMove(pnt image.Point, down bool) {
	if lvl, i := mb.popupAt(pnt); lvl >= 0 {
		if i >= 0 || mb.open[lvl].child < 0 {
			mb.selectItem(lvl, i)
		}

		return
	}

	if pnt.In(mb.win.menuRect()) {
		// With a menu open, or the button down, the selection follows the mouse.
		following := len(mb.open) > 0 || down
		if i := mb.barAt(pnt); i >= 0 && i != mb.sel && following {
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

func (mb *menuBar) mouseUp(pnt image.Point) {
	pressed := mb.pressed
	mb.pressed = -1

	if lvl, i := mb.popupAt(pnt); lvl >= 0 && i >= 0 {
		me := mb.open[lvl].items[i]
		if len(me.kids) == 0 && me.item.Enabled {
			mb.run(me)
		}

		return
	}

	if pressed >= 0 && len(mb.open) == 0 && mb.barAt(pnt) == pressed {
		// A disabled or empty menu: the click ends menu mode.
		mb.exit()
	}

	mb.win.desk.dirty = true
}

// tick opens a submenu once the mouse has rested on its item.
func (mb *menuBar) tick(now time.Time) {
	for lvl, mp := range mb.open {
		if mp.pending >= 0 && now.Sub(mp.pendingAt) >= menuShowDelay {
			mb.openChild(lvl, mp.pending)

			return
		}
	}
}

func (mb *menuBar) char(rune) {}

// key handles a key in menu mode.
func (mb *menuBar) key(key ebiten.Key) {
	dsk := mb.win.desk
	dsk.dirty = true
	n := len(mb.open)

	var mp *menuPopup

	if n > 0 {
		mp = mb.open[n-1]
	}

	switch key {
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
		if mp != nil && mp.opensSubmenu() {
			mb.openChild(n-1, mp.sel)
		} else {
			mb.selectBar(mb.nextBar(mb.sel, 1), n > 0)
		}

		mb.selectFirst()
	case ebiten.KeyUp:
		mb.upDown(mp, -1)
	case ebiten.KeyDown:
		mb.upDown(mp, 1)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if mp == nil {
			mb.selectBar(mb.sel, true)
			mb.selectFirst()

			return
		}

		mb.activateItem(n - 1)
	default:
		if chr := keyRune(key); chr != 0 {
			mb.letter(mp, chr)
		}
	}
}

// upDown moves the selection of the deepest open popup mp by dir, round
// its items, or drops the bar item's menu down when no popup is open.
func (mb *menuBar) upDown(mp *menuPopup, dir int) {
	if mp == nil {
		mb.selectBar(mb.sel, true)
		mb.selectFirst()

		return
	}

	if cnt := len(mp.items); cnt > 0 {
		i := mp.sel
		if i < 0 && dir < 0 {
			i = 0
		}

		mb.selectItem(len(mb.open)-1, (i+dir+cnt)%cnt)

		mp.pending = -1
	}
}

// letter selects the item whose accelerator chr is: on the bar, which drops
// its menu down, while no popup is open, or else in the deepest open popup
// mp, where it also activates it.
func (mb *menuBar) letter(mp *menuPopup, chr rune) {
	if mp == nil {
		for i, me := range mb.entries {
			if me.item.Visible && accel(me.caption) == chr {
				mb.selectBar(i, true)
				mb.selectFirst()

				return
			}
		}

		return
	}

	for i, me := range mp.items {
		if accel(me.caption) == chr {
			lvl := len(mb.open) - 1
			mb.selectItem(lvl, i)
			mb.activateItem(lvl)

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

// activateItem is Enter on the selected item of the popup at level lvl: a
// submenu opens, a command runs, a disabled item closes the menus.
func (mb *menuBar) activateItem(lvl int) {
	mp := mb.open[lvl]
	if mp.sel < 0 {
		return
	}

	me := mp.items[mp.sel]
	switch {
	case len(me.kids) > 0:
		mb.openChild(lvl, mp.sel)
		mb.selectFirst()
	case me.item.Enabled:
		mb.run(me)
	default:
		mb.exit()
	}
}

// openByKey opens the bar item whose accelerator the key is (Alt and a
// letter); it reports whether there is one.
func (mb *menuBar) openByKey(key ebiten.Key) bool {
	chr := keyRune(key)
	if chr == 0 {
		return false
	}

	for i, me := range mb.entries {
		if me.item.Visible && accel(me.caption) == chr {
			mb.enter(i, true)
			mb.selectFirst()

			return true
		}
	}

	return false
}

// shortcut runs the command whose shortcut the key is, if it and the items
// above it are enabled, as TMenu.IsShortCut does; hidden items count.
func (mb *menuBar) shortcut(key ebiten.Key, mm mods) bool {
	if mm.ctrl || mm.alt || mm.shift {
		return false
	}

	var find func(es []*menuEntry) *menuEntry
	find = func(es []*menuEntry) *menuEntry {
		for _, me := range es {
			if !me.item.Enabled {
				continue
			}

			if me.key == key && len(me.kids) == 0 {
				return me
			}

			if fnd := find(me.kids); fnd != nil {
				return fnd
			}
		}

		return nil
	}
	ent := find(mb.entries)

	if ent == nil {
		return false
	}

	mb.run(ent)

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
	for _, mp := range mb.open {
		cv.fill(mp.rect, colFace)
		cv.raised(mp.rect)

		for i, me := range mp.items {
			mb.paintItem(cv, mp.itemRect(i), me, i == mp.sel)
		}
	}
}

// paintItem draws an item of a submenu, as TMenuItem.AdvancedDrawItem
// draws it, and the arrow Windows adds for a submenu.
func (mb *menuBar) paintItem(cv canvas, rect image.Rectangle, me *menuEntry, sel bool) {
	fnt := mb.res.sans
	fg := colText

	if sel {
		cv.fill(rect, colSelection)

		fg = colSelText
	}

	enabled := me.item.Enabled
	switch {
	case me.image >= 0 && mb.images != nil && enabled:
		mb.images.drawFrame(cv, rect.Min.X+1, rect.Min.Y+1, me.image, false)
	case me.image >= 0 && mb.images != nil:
		mb.images.drawDisabled(cv, rect.Min.X+1, rect.Min.Y+1, me.image)
	case me.item.Checked:
		cv.pattern(rect.Min.X+6, rect.Min.Y+6, menuCheck, map[byte]color.RGBA{'#': fg})
	}

	x, y := rect.Min.X+22, rect.Min.Y+(rect.Dy()-fnt.height())/2
	text := func(x int, txt string) {
		switch {
		case enabled:
			drawCaption(cv, fnt, x, y, txt, fg)
		case sel:
			drawCaption(cv, fnt, x, y, txt, colGrayText)
		default:
			drawDisabled(cv, fnt, x, y, txt)
		}
	}
	text(x, me.caption)

	if me.shortcut != "" {
		text(rect.Max.X-10-fnt.width(me.shortcut), me.shortcut)
	}

	if len(me.kids) > 0 {
		cv.pattern(rect.Max.X-8, rect.Min.Y+(rect.Dy()-7)/2, menuArrow, map[byte]color.RGBA{'#': fg})
	}
}
