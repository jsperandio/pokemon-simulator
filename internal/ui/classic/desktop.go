// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"cmp"
	"image"
	"io/fs"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The logical screen: the 800×600 desktop the author laid the windows out
// for.
const (
	screenWidth  = 800
	screenHeight = 600
)

// Double clicks, as Windows' defaults have them.
const (
	doubleClickTime = 500 * time.Millisecond
	doubleClickDist = 4 // SM_CXDOUBLECLK, SM_CYDOUBLECLK
	caretBlink      = 530 * time.Millisecond
)

// A popup is a window above all others that takes the mouse first: a
// combo box's list.
type popup interface {
	bounds() image.Rectangle
	paint(cv canvas)
	mouseDown(p image.Point)
	mouseMove(p image.Point, down bool)
	// mouseUp reports whether the popup is done and should close.
	mouseUp(p image.Point) bool
	tick(now time.Time) bool
	wheel(dy int)
	// dismiss closes it from outside, as a click elsewhere does.
	dismiss()
}

// ticker is a widget that does something while time passes: the
// auto-repeat of a held button.
type ticker interface {
	tick(now time.Time)
}

// input is one frame's input, in screen coordinates.
type input struct {
	mouse    image.Point
	down     bool // the left button is held
	pressed  bool // ... went down in this frame
	released bool // ... went up in this frame
	wheel    int  // notches, positive away from the user
	keys     []ebiten.Key
	keysUp   []ebiten.Key
	chars    []rune
	mods     mods
	dropped  fs.FS // the files dropped on the game, if any
}

// desktop is the screen: its windows in z-order, the popups, and the
// routing of the mouse and the keyboard.
type desktop struct {
	res     *resources
	windows []*window // bottom to top
	active  *window
	modals  []*window
	popups  []popup
	menu    *menuBar // the menu bar while it tracks the mouse or keys
	dirty   bool
	now     time.Time

	mouse      image.Point
	capture    widget // the pressed widget: it gets the drags and the release
	popupPress bool   // the press began in a popup
	drag       *window
	dragFrom   image.Point // the mouse's offset into the dragged window
	capWindow  *window     // whose caption button is held
	hover      widget
	lastClick  time.Time
	lastPos    image.Point
	lastTarget any
	cursor     ebiten.CursorShapeType
	altAlone   bool // Alt is down and no other key came with it
	caretOn    bool // the caret's phase when last painted
}

func newDesktop(res *resources) *desktop {
	return &desktop{res: res, dirty: true}
}

// add registers a window, hidden.
func (d *desktop) add(w *window) {
	d.windows = append(d.windows, w)
}

// remove forgets a window for good, as freeing a dialog does.
func (d *desktop) remove(w *window) {
	d.hide(w)
	if i := slices.Index(d.windows, w); i >= 0 {
		d.windows = slices.Delete(d.windows, i, i+1)
	}
	for _, o := range d.windows {
		if o.prevActive == w {
			o.prevActive = nil
		}
	}
}

// show shows a window above the others and activates it, as a form's Show
// does.
func (d *desktop) show(w *window) {
	if !w.visible {
		w.visible = true
		if w.onShow != nil {
			w.onShow()
		}
	}
	d.raise(w)
	d.activate(w)
	d.dirty = true
}

// hide hides a window; if it was active, the topmost visible window takes
// its place.
func (d *desktop) hide(w *window) {
	if !w.visible {
		return
	}
	w.visible = false
	d.release(w)
	if d.active == w {
		d.active = nil
		w.active = false
		if w.onDeactivate != nil {
			w.onDeactivate()
		}
		d.activateTop()
	}
	if w.onHide != nil {
		w.onHide()
	}
	d.dirty = true
}

// release forgets the mouse and the popups of a window that goes away.
func (d *desktop) release(w *window) {
	if d.capture != nil && d.capture.base().win == w {
		d.capture = nil
	}
	if d.hover != nil && d.hover.base().win == w {
		d.hover = nil
	}
	if d.drag == w {
		d.drag = nil
	}
	if d.capWindow == w {
		d.capWindow = nil
		w.capPressed = capNone
	}
	if d.menu != nil && d.menu.win == w {
		d.menu.exit()
	}
	d.dismissPopups()
}

// activateTop activates the topmost visible window.
func (d *desktop) activateTop() {
	for i := len(d.windows) - 1; i >= 0; i-- {
		if w := d.windows[i]; w.visible {
			d.activate(w)
			return
		}
	}
}

// raise puts a window above the others.
func (d *desktop) raise(w *window) {
	if i := slices.Index(d.windows, w); i >= 0 && i != len(d.windows)-1 {
		d.windows = append(slices.Delete(d.windows, i, i+1), w)
		d.dirty = true
	}
}

// activate makes w the active window: its caption lights up and its
// focused control gets the keyboard. While a modal window is up, only it
// can be active.
func (d *desktop) activate(w *window) {
	if m := d.topModal(); m != nil && w != m {
		w = m
	}
	if w == d.active {
		if w != nil {
			d.raise(w)
		}
		return
	}
	prev := d.active
	if prev != nil {
		prev.active = false
		d.dismissPopups()
		if d.menu != nil {
			d.menu.exit()
		}
		if l, ok := prev.focus.(interface{ lostFocus() }); ok {
			l.lostFocus()
		}
	}
	d.active = w
	if w != nil {
		w.active = true
		d.raise(w)
		if w.focus == nil || !canFocus(w.focus) {
			w.focus = nil
			w.focusFirst()
		} else if g, ok := w.focus.(interface{ gotFocus(bool) }); ok {
			g.gotFocus(false)
		}
	}
	d.dirty = true
	if prev != nil && prev.onDeactivate != nil {
		prev.onDeactivate()
	}
	if w != nil && w.onActivate != nil {
		w.onActivate()
	}
}

func (d *desktop) topModal() *window {
	if n := len(d.modals); n > 0 {
		return d.modals[n-1]
	}
	return nil
}

// showModal shows a window as ShowModal does: until it closes, the other
// windows take no input.
func (d *desktop) showModal(w *window) {
	w.modal = true
	w.modalResult = mrNone
	w.prevActive = d.active
	d.modals = append(d.modals, w)
	d.show(w)
}

// closeModal hides a modal window and gives the activation back to the
// window that had it.
func (d *desktop) closeModal(w *window) {
	i := slices.Index(d.modals, w)
	if i < 0 {
		return
	}
	d.modals = slices.Delete(d.modals, i, i+1)
	w.modal = false
	prev := w.prevActive
	w.prevActive = nil
	d.hide(w)
	if prev != nil && prev.visible {
		d.activate(prev)
	}
}

func (d *desktop) openPopup(p popup) {
	d.popups = append(d.popups, p)
	d.dirty = true
}

func (d *desktop) closePopup(p popup) {
	if i := slices.Index(d.popups, p); i >= 0 {
		d.popups = slices.Delete(d.popups, i, i+1)
		d.popupPress = false
		d.dirty = true
	}
}

// dismissPopups closes every popup.
func (d *desktop) dismissPopups() {
	for len(d.popups) > 0 {
		n := len(d.popups)
		d.popups[n-1].dismiss()
		if len(d.popups) == n { // it didn't close itself
			d.popups = d.popups[:n-1]
		}
	}
	d.popupPress = false
}

// windowAt returns the topmost visible window at p, or nil.
func (d *desktop) windowAt(p image.Point) *window {
	for i := len(d.windows) - 1; i >= 0; i-- {
		if w := d.windows[i]; w.visible && p.In(w.rect()) {
			return w
		}
	}
	return nil
}

// blocked reports whether a modal window keeps w from taking input.
func (d *desktop) blocked(w *window) bool {
	m := d.topModal()
	return m != nil && w != m
}

// update handles one frame's input.
func (d *desktop) update(in *input, now time.Time) {
	d.now = now
	p := image.Pt(max(min(in.mouse.X, screenWidth-1), 0), max(min(in.mouse.Y, screenHeight-1), 0))
	moved := p != d.mouse
	d.mouse = p
	if moved {
		d.mouseMove(p, in.down && !in.pressed)
	}
	if in.pressed {
		d.mouseDown(p)
	}
	if in.released {
		d.mouseUp(p)
	}
	if in.wheel != 0 {
		d.wheel(p, in.wheel)
	}
	// Keys that come down in the same frame come in key code order, the
	// letters before Alt; the modifiers go first, as they were pressed
	// before the keys they modify. Alt+M in one frame isn't Alt alone.
	slices.SortStableFunc(in.keys, func(a, b ebiten.Key) int {
		return cmp.Compare(keyOrder(a), keyOrder(b))
	})
	// The characters the keys type go to the active window, unless the
	// menu took the keys, or they opened another window: a menu's loop
	// eats its characters, as a dialog the menu opens doesn't get them.
	active, menu := d.active, false
	for _, k := range in.keys {
		menu = menu || d.menu != nil
		d.keyDown(k, in.mods)
		menu = menu || d.menu != nil
	}
	if !in.mods.alt && !in.mods.ctrl && !menu && d.active == active {
		for _, r := range in.chars {
			d.char(r, in.mods)
		}
	}
	for _, k := range in.keysUp {
		d.keyUp(k, in.mods)
	}
	d.tick(now)
	d.updateCursor(p)
}

// widgetOrigin returns a widget's top-left corner on the screen.
func widgetOrigin(k widget) image.Point {
	c := k.base()
	return c.win.clientOrigin().Add(c.origin())
}

// isDouble reports whether a press on target at p makes a double click.
func (d *desktop) isDouble(p image.Point, target any) bool {
	dp := p.Sub(d.lastPos)
	double := target == d.lastTarget && d.now.Sub(d.lastClick) <= doubleClickTime &&
		abs(dp.X) <= doubleClickDist/2 && abs(dp.Y) <= doubleClickDist/2
	if double {
		d.lastTarget = nil // a third click starts over
	} else {
		d.lastTarget = target
	}
	d.lastClick, d.lastPos = d.now, p
	return double
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (d *desktop) mouseDown(p image.Point) {
	d.dirty = true
	if d.menu != nil {
		d.menu.mouseDown(p)
		return
	}
	if n := len(d.popups); n > 0 {
		top := d.popups[n-1]
		if p.In(top.bounds()) {
			d.popupPress = true
			top.mouseDown(p)
			return
		}
		d.dismissPopups()
		return // the click only closes the popup
	}
	w := d.windowAt(p)
	if w == nil {
		return
	}
	if d.blocked(w) {
		d.activate(d.topModal())
		return
	}
	d.activate(w)
	if w.style != bsNone && p.In(w.captionRect()) {
		d.captionDown(w, p)
		return
	}
	if w.menu != nil && p.In(w.menuRect()) {
		d.menu = w.menu
		w.menu.mouseDown(p)
		return
	}
	if !p.In(w.clientRect()) {
		return // the frame
	}
	k := w.widgetAt(p)
	// A disabled control takes no clicks: its parent gets them, as
	// WindowFromPoint skips disabled windows.
	for k != nil && !k.base().usable() {
		k = k.base().parent
	}
	double := d.isDouble(p, any(k))
	if k == nil {
		if w.onClick != nil {
			w.onClick()
		}
		return
	}
	if canFocus(k) {
		w.setFocus(k, true)
	}
	if c, ok := k.(clicker); ok {
		d.capture = k
		c.mouseDown(p.Sub(widgetOrigin(k)), double)
	} else if c := k.base(); c.onClick != nil {
		c.onClick()
	}
}

// captionDown handles a press on a window's caption: a caption button,
// a double click, or the start of a drag.
func (d *desktop) captionDown(w *window, p image.Point) {
	for _, b := range []int{capMin, capMax, capClose} {
		if p.In(w.captionButton(b)) {
			w.capPressed, w.capInside = b, true
			d.capWindow = w
			return
		}
	}
	if d.isDouble(p, w) {
		icon := w.sysMenu && w.style != bsDialog && w.style != bsToolWindow &&
			p.X < w.captionRect().Min.X+20
		switch {
		case icon:
			w.close() // a double click on the icon closes the window
		case w.maxBox:
			w.toggleMaximize()
		}
		return
	}
	if !w.maximized {
		d.drag = w
		d.dragFrom = p.Sub(w.pos)
	}
}

// mouseMove follows the mouse: drags, hot-tracking and hovering.
func (d *desktop) mouseMove(p image.Point, down bool) {
	d.lastTarget = nil
	if d.menu != nil {
		d.menu.mouseMove(p, down)
		return
	}
	switch {
	case d.drag != nil:
		d.drag.pos = p.Sub(d.dragFrom)
		d.dirty = true
		return
	case d.capWindow != nil:
		w := d.capWindow
		if in := p.In(w.captionButton(w.capPressed)); in != w.capInside {
			w.capInside = in
			d.dirty = true
		}
		return
	}
	if n := len(d.popups); n > 0 {
		d.popups[n-1].mouseMove(p, down && d.popupPress || down && d.capture != nil)
		d.dirty = true
		if d.popupPress {
			return
		}
	}
	if d.capture != nil {
		if c, ok := d.capture.(clicker); ok {
			c.mouseDrag(p.Sub(widgetOrigin(d.capture)))
		}
		return
	}
	// Hovering: hot-tracking list views and the menu bar.
	var over widget
	w := d.windowAt(p)
	for _, m := range d.menuBars() {
		m.hover(p, w == m.win && !d.blocked(w) && len(d.popups) == 0)
	}
	if w != nil && !d.blocked(w) && len(d.popups) == 0 && p.In(w.clientRect()) {
		over = w.widgetAt(p)
	}
	if over != d.hover {
		if h, ok := d.hover.(hoverer); ok {
			h.hover(image.Point{}, true)
		}
		d.hover = over
	}
	if h, ok := over.(hoverer); ok {
		h.hover(p.Sub(widgetOrigin(over)), false)
	}
}

// menuBars lists the visible windows' menu bars.
func (d *desktop) menuBars() []*menuBar {
	var out []*menuBar
	for _, w := range d.windows {
		if w.visible && w.menu != nil {
			out = append(out, w.menu)
		}
	}
	return out
}

func (d *desktop) mouseUp(p image.Point) {
	d.dirty = true
	if d.menu != nil {
		d.menu.mouseUp(p)
		return
	}
	switch {
	case d.drag != nil:
		d.drag = nil
		return
	case d.capWindow != nil:
		w := d.capWindow
		b := w.capPressed
		inside := p.In(w.captionButton(b))
		w.capPressed, d.capWindow = capNone, nil
		if inside {
			switch b {
			case capClose:
				w.close()
			case capMax:
				if w.maxBox {
					w.toggleMaximize()
				}
			}
		}
		return
	}
	if d.popupPress {
		d.popupPress = false
		if n := len(d.popups); n > 0 && d.popups[n-1].mouseUp(p) {
			d.popups[n-1].dismiss()
		}
		return
	}
	if d.capture != nil {
		k := d.capture
		d.capture = nil
		if c, ok := k.(clicker); ok {
			c.mouseUp(p.Sub(widgetOrigin(k)))
		}
		// A press on a combo box's button can end on its list.
		if n := len(d.popups); n > 0 && p.In(d.popups[n-1].bounds()) && d.popups[n-1].mouseUp(p) {
			d.popups[n-1].dismiss()
		}
	}
}

// wheel sends the mouse wheel to the popup, or to the focused control of
// the active window.
func (d *desktop) wheel(p image.Point, dy int) {
	if d.menu != nil {
		return
	}
	d.dirty = true
	if n := len(d.popups); n > 0 {
		d.popups[n-1].wheel(dy)
		return
	}
	if w := d.active; w != nil && w.focus != nil {
		if h, ok := w.focus.(wheeler); ok && w.focus.base().usable() {
			h.wheel(dy)
		}
	}
}

func (d *desktop) keyDown(k ebiten.Key, m mods) {
	d.dirty = true
	if k == ebiten.KeyAltLeft || k == ebiten.KeyAltRight {
		d.altAlone = true
		return
	}
	d.altAlone = false
	if d.menu != nil {
		d.menu.key(k, m)
		return
	}
	w := d.active
	if w == nil {
		return
	}
	if w.menu != nil && !d.blocked(w) && len(d.popups) == 0 {
		switch {
		case k == ebiten.KeyF10 && !m.shift && !m.ctrl:
			d.menu = w.menu
			w.menu.enterKeyboard()
			return
		case m.alt && w.menu.openByKey(k):
			d.menu = w.menu
			return
		case w.menu.shortcut(k, m):
			return
		}
	}
	switch {
	case m.alt && k == ebiten.KeyF4:
		if w.sysMenu && w.style != bsNone {
			w.close()
		}
		return
	case m.alt && keyRune(k) != 0:
		// Alt and a letter: a button's accelerator.
		w.accelerator(keyRune(k))
		return
	}
	w.key(k, m)
}

func (d *desktop) keyUp(k ebiten.Key, m mods) {
	if (k == ebiten.KeyAltLeft || k == ebiten.KeyAltRight) && d.altAlone {
		d.altAlone = false
		d.dirty = true
		if d.menu != nil {
			d.menu.exit()
			return
		}
		if w := d.active; w != nil && w.menu != nil && !d.blocked(w) && len(d.popups) == 0 {
			d.menu = w.menu
			w.menu.enterKeyboard()
		}
		return
	}
	if w := d.active; w != nil {
		if b, ok := w.focus.(*button); ok {
			b.keyUp(k)
		}
	}
}

func (d *desktop) char(r rune, m mods) {
	if d.menu != nil {
		d.menu.char(r)
		return
	}
	if w := d.active; w != nil {
		d.dirty = true
		w.char(r, m)
	}
}

// keyRune returns the letter or digit a key types, lowercased, or 0.
func keyRune(k ebiten.Key) rune {
	switch {
	case k >= ebiten.KeyA && k <= ebiten.KeyZ:
		return 'a' + rune(k-ebiten.KeyA)
	case k >= ebiten.KeyDigit0 && k <= ebiten.KeyDigit9:
		return '0' + rune(k-ebiten.KeyDigit0)
	}
	return 0
}

// keyOrder is 0 for the modifier keys and 1 for the others.
func keyOrder(k ebiten.Key) int {
	switch k {
	case ebiten.KeyAltLeft, ebiten.KeyAltRight, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyShiftLeft, ebiten.KeyShiftRight, ebiten.KeyMetaLeft, ebiten.KeyMetaRight:
		return 0
	}
	return 1
}

// tick runs what time drives: auto-repeat, the menus' delays and the
// caret.
func (d *desktop) tick(now time.Time) {
	if n := len(d.popups); n > 0 && d.popups[n-1].tick(now) {
		d.dirty = true
	}
	if t, ok := d.capture.(ticker); ok {
		t.tick(now)
	}
	if d.menu != nil {
		d.menu.tick(now)
	}
	if w := d.active; w != nil && w.focus != nil {
		var since time.Duration
		switch f := w.focus.(type) {
		case *edit:
			since = now.Sub(f.tb.blink)
		case *spinEdit:
			since = now.Sub(f.tb.blink)
		case *comboBox:
			since = now.Sub(f.tb.blink)
		default:
			return
		}
		if on := since%(2*caretBlink) < caretBlink; on != d.caretOn {
			d.caretOn = on
			d.dirty = true
		}
	}
}

func (d *desktop) updateCursor(p image.Point) {
	c := ebiten.CursorShapeDefault
	if d.menu == nil && len(d.popups) == 0 && d.drag == nil {
		k := d.capture
		if k == nil {
			if w := d.windowAt(p); w != nil && !d.blocked(w) && p.In(w.clientRect()) {
				k = w.widgetAt(p)
			}
		}
		if cu, ok := k.(cursorer); ok && k.base().usable() {
			c = cu.cursor(p.Sub(widgetOrigin(k)))
		}
	}
	d.cursor = c
}

// paint draws the desktop.
func (d *desktop) paint(img *image.RGBA) {
	cv := newCanvas(img)
	cv.fill(img.Rect, colDesktop)
	for _, w := range d.windows {
		w.paint(cv)
	}
	for _, p := range d.popups {
		p.paint(cv)
	}
	if d.menu != nil {
		d.menu.paintPopups(cv)
	}
	d.dirty = false
}
