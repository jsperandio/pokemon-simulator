// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"cmp"
	"image"
	"io/fs"
	"slices"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The logical screen: the 800×600 desktop the author laid the windows out
// for.
const (
	screenWidth  int = 800
	screenHeight int = 600
)

// Double clicks, as Windows' defaults have them.
const (
	doubleClickTime time.Duration = 500 * time.Millisecond
	doubleClickDist int           = 4 // SM_CXDOUBLECLK, SM_CYDOUBLECLK
)

// A popup is a window above all others that takes the mouse first: a
// combo box's list.
type popup interface {
	bounds() image.Rectangle
	paint(cv canvas)
	mouseDown(pnt image.Point)
	mouseMove(pnt image.Point, down bool)
	// mouseUp reports whether the popup is done and should close.
	mouseUp(pnt image.Point) bool
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

// active reports whether the player did anything in the frame: moved the
// mouse on the desktop, from or to prev, or used it or the keyboard. The
// mouse moving elsewhere on the screen doesn't count, as the original
// heard of it only over its windows.
func (in *input) active(prev image.Point) bool {
	desktop := image.Rect(0, 0, screenWidth, screenHeight)
	moved := in.mouse != prev && (in.mouse.In(desktop) || prev.In(desktop))
	typed := len(in.chars) > 0 || in.dropped != nil

	return moved || in.down || in.acted() || typed
}

// acted reports whether the frame has a mouse button or wheel event, or a
// key that came down or went up.
func (in *input) acted() bool {
	return in.pressed || in.released || in.wheel != 0 || len(in.keys) > 0 || len(in.keysUp) > 0
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
	hint       hint

	onMinimize func() // a minimize button: the program's window
}

func newDesktop(res *resources) *desktop {
	return &desktop{res: res, dirty: true}
}

// add registers a window, hidden.
func (d *desktop) add(wdw *window) {
	d.windows = append(d.windows, wdw)
}

// remove forgets a window for good, as freeing a dialog does.
func (d *desktop) remove(wdw *window) {
	d.hide(wdw)

	if i := slices.Index(d.windows, wdw); i >= 0 {
		d.windows = slices.Delete(d.windows, i, i+1)
	}

	for _, oth := range d.windows {
		if oth.prevActive == wdw {
			oth.prevActive = nil
		}
	}
}

// show shows a window above the others and activates it, as a form's Show
// does.
func (d *desktop) show(wdw *window) {
	if !wdw.visible {
		wdw.visible = true
		if wdw.onShow != nil {
			wdw.onShow()
		}
	}

	d.raise(wdw)
	d.activate(wdw)

	d.dirty = true
}

// hide hides a window; if it was active, the topmost visible window takes
// its place.
func (d *desktop) hide(wdw *window) {
	if !wdw.visible {
		return
	}

	wdw.visible = false
	d.release(wdw)

	if d.active == wdw {
		d.active = nil
		wdw.active = false

		if wdw.onDeactivate != nil {
			wdw.onDeactivate()
		}

		d.activateTop()
	}

	if wdw.onHide != nil {
		wdw.onHide()
	}

	d.dirty = true
}

// release forgets the mouse and the popups of a window that goes away.
func (d *desktop) release(wdw *window) {
	if d.capture != nil && d.capture.base().wdw == wdw {
		d.capture = nil
	}

	if d.hover != nil && d.hover.base().wdw == wdw {
		d.hover = nil
	}

	if d.drag == wdw {
		d.drag = nil
	}

	if d.capWindow == wdw {
		d.capWindow = nil
		wdw.capPressed = capNone
	}

	if d.menu != nil && d.menu.win == wdw {
		d.menu.exit()
	}

	d.dismissPopups()
}

// activateTop activates the topmost visible window.
func (d *desktop) activateTop() {
	for _, wdw := range slices.Backward(d.windows) {
		if wdw.visible {
			d.activate(wdw)

			return
		}
	}
}

// raise puts a window above the others.
func (d *desktop) raise(wdw *window) {
	if i := slices.Index(d.windows, wdw); i >= 0 && i != len(d.windows)-1 {
		d.windows = append(slices.Delete(d.windows, i, i+1), wdw)
		d.dirty = true
	}
}

// activate makes wdw the active window: its caption lights up and its
// focused control gets the keyboard. While a modal window is up, only it
// can be active.
func (d *desktop) activate(wdw *window) {
	if mdl := d.topModal(); mdl != nil && wdw != mdl {
		wdw = mdl
	}

	if wdw == d.active {
		if wdw != nil {
			d.raise(wdw)
		}

		return
	}

	prev := d.active
	if prev != nil {
		d.leave(prev)
	}

	d.active = wdw
	if wdw != nil {
		d.enter(wdw)
	}

	d.dirty = true

	if prev != nil && prev.onDeactivate != nil {
		prev.onDeactivate()
	}

	if wdw != nil && wdw.onActivate != nil {
		wdw.onActivate()
	}
}

// leave turns off the window that was active, and its menus, popups and
// focus.
func (d *desktop) leave(wdw *window) {
	wdw.active = false

	d.dismissPopups()

	if d.menu != nil {
		d.menu.exit()
	}

	if lfc, ok := wdw.focus.(interface{ lostFocus() }); ok {
		lfc.lostFocus()
	}
}

// enter lights up the window that becomes active, above the others, and
// gives the keyboard to its focused control, or to its first one.
func (d *desktop) enter(wdw *window) {
	wdw.active = true
	wdw.noFocus = false
	d.raise(wdw)

	if wdw.focus == nil || !canFocus(wdw.focus) {
		wdw.focus = nil
		wdw.focusFirst()

		return
	}

	if gfc, ok := wdw.focus.(interface{ gotFocus(bool) }); ok {
		gfc.gotFocus(false)
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
func (d *desktop) showModal(wdw *window) {
	wdw.modal = true
	wdw.modalResult = mrNone
	wdw.prevActive = d.active
	d.modals = append(d.modals, wdw)
	d.show(wdw)
}

// closeModal hides a modal window and gives the activation back to the
// window that had it.
func (d *desktop) closeModal(wdw *window) {
	i := slices.Index(d.modals, wdw)
	if i < 0 {
		return
	}

	d.modals = slices.Delete(d.modals, i, i+1)
	wdw.modal = false
	prev := wdw.prevActive
	wdw.prevActive = nil
	d.hide(wdw)

	if prev != nil && prev.visible {
		d.activate(prev)
	}
}

func (d *desktop) openPopup(pop popup) {
	d.popups = append(d.popups, pop)
	d.dirty = true
}

func (d *desktop) closePopup(pop popup) {
	if i := slices.Index(d.popups, pop); i >= 0 {
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

// windowAt returns the topmost visible window at pnt, or nil.
func (d *desktop) windowAt(pnt image.Point) *window {
	for _, wdw := range slices.Backward(d.windows) {
		if wdw.visible && pnt.In(wdw.rect()) {
			return wdw
		}
	}

	return nil
}

// blocked reports whether a modal window keeps wdw from taking input.
func (d *desktop) blocked(wdw *window) bool {
	mdl := d.topModal()

	return mdl != nil && wdw != mdl
}

// takesInput reports whether window wdw gets the mouse and the keys: it
// exists, no modal window blocks it and no popup is open.
func (d *desktop) takesInput(wdw *window) bool {
	return wdw != nil && !d.blocked(wdw) && len(d.popups) == 0
}

// update handles one frame's input.
func (d *desktop) update(in *input, now time.Time) {
	d.now = now
	pnt := image.Pt(max(min(in.mouse.X, screenWidth-1), 0), max(min(in.mouse.Y, screenHeight-1), 0))
	moved := pnt != d.mouse
	d.mouse = pnt

	if moved {
		d.mouseMove(pnt, in.down && !in.pressed)
	}

	if in.pressed {
		d.mouseDown(pnt)
	}

	if in.released {
		d.mouseUp(pnt)
	}

	if in.wheel != 0 {
		d.wheel(in.wheel)
	}

	d.keys(in)

	d.updateHint(pnt, in.acted())
	d.tick(now)
	d.updateCursor(pnt)
}

// keys handles one frame's keys, and the characters they type.
func (d *desktop) keys(in *input) {
	// Keys that come down in the same frame come in key code order, the
	// letters before Alt; the modifiers go first, as they were pressed
	// before the keys they modify. Alt+M in one frame isn't Alt alone.
	slices.SortStableFunc(in.keys, func(fst, snd ebiten.Key) int {
		return cmp.Compare(keyOrder(fst), keyOrder(snd))
	})

	// The characters the keys type go to the active window, unless the
	// menu took the keys, or they opened another window: a menu's loop
	// eats its characters, as a dialog the menu opens doesn't get them.
	active, menu := d.active, false
	for _, key := range in.keys {
		menu = menu || d.menu != nil
		d.keyDown(key, in.mods)

		menu = menu || d.menu != nil
	}

	shortcut := in.mods.alt || in.mods.ctrl
	if !shortcut && !menu && d.active == active {
		for _, chr := range in.chars {
			d.char(chr)
		}
	}

	for _, key := range in.keysUp {
		d.keyUp(key)
	}
}

// widgetOrigin returns a widget's top-left corner on the screen.
func widgetOrigin(wgt widget) image.Point {
	ctl := wgt.base()

	return ctl.wdw.clientOrigin().Add(ctl.origin())
}

// isDouble reports whether a press on target at pnt makes a double click.
func (d *desktop) isDouble(pnt image.Point, target any) bool {
	dp := pnt.Sub(d.lastPos)
	sameTarget := target == d.lastTarget
	inTime := d.now.Sub(d.lastClick) <= doubleClickTime
	inRange := abs(dp.X) <= doubleClickDist/2 && abs(dp.Y) <= doubleClickDist/2
	double := sameTarget && inTime && inRange

	d.lastTarget = target
	if double {
		d.lastTarget = nil // a third click starts over
	}

	d.lastClick, d.lastPos = d.now, pnt

	return double
}

func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}

func (d *desktop) mouseDown(pnt image.Point) {
	d.dirty = true
	if d.menu != nil {
		d.menu.mouseDown(pnt)

		return
	}

	if len(d.popups) > 0 {
		d.popupDown(pnt)

		return
	}

	wdw := d.windowAt(pnt)
	if wdw == nil {
		return
	}

	if d.blocked(wdw) {
		d.activate(d.topModal())

		return
	}

	d.activate(wdw)

	// A press on the frame does nothing.
	switch {
	case wdw.style != bsNone && pnt.In(wdw.captionRect()):
		d.captionDown(wdw, pnt)
	case wdw.menu != nil && pnt.In(wdw.menuRect()):
		d.menu = wdw.menu
		wdw.menu.mouseDown(pnt)
	case pnt.In(wdw.clientRect()):
		d.clientDown(wdw, pnt)
	}
}

// popupDown handles a press while popups are open: on the top one, or
// elsewhere, which only closes them.
func (d *desktop) popupDown(pnt image.Point) {
	top := d.popups[len(d.popups)-1]
	if !pnt.In(top.bounds()) {
		d.dismissPopups()

		return
	}

	d.popupPress = true

	top.mouseDown(pnt)
}

// clientDown handles a press in a window's client area: on a control, or
// on the form.
func (d *desktop) clientDown(wdw *window, pnt image.Point) {
	wgt := wdw.widgetAt(pnt)
	// A disabled control takes no clicks: its parent gets them, as
	// WindowFromPoint skips disabled windows.
	for wgt != nil && !wgt.base().usable() {
		wgt = wgt.base().parent
	}

	double := d.isDouble(pnt, any(wgt))
	if wgt == nil {
		if wdw.onClick != nil {
			wdw.onClick()
		}

		return
	}

	if canFocus(wgt) {
		wdw.setFocus(wgt, true)
	}

	if clk, ok := wgt.(clicker); ok {
		d.capture = wgt
		clk.mouseDown(pnt.Sub(widgetOrigin(wgt)), double)

		return
	}

	if ctl := wgt.base(); ctl.onClick != nil {
		ctl.onClick()
	}
}

// captionDown handles a press on a window's caption: a caption button,
// a double click, or the start of a drag.
func (d *desktop) captionDown(wdw *window, pnt image.Point) {
	for _, cbt := range []int{capMin, capMax, capClose} {
		if pnt.In(wdw.captionButton(cbt)) {
			wdw.capPressed, wdw.capInside = cbt, true
			d.capWindow = wdw

			return
		}
	}

	if d.isDouble(pnt, wdw) {
		icon := wdw.hasSysIcon() && pnt.X < wdw.captionRect().Min.X+20
		switch {
		case icon:
			wdw.close() // a double click on the icon closes the window
		case wdw.maxBox:
			wdw.toggleMaximize()
		}

		return
	}

	if !wdw.maximized {
		d.drag = wdw
		d.dragFrom = pnt.Sub(wdw.pos)
	}
}

// mouseMove follows the mouse: drags, hot-tracking and hovering.
func (d *desktop) mouseMove(pnt image.Point, down bool) {
	d.lastTarget = nil
	if d.menu != nil {
		d.menu.mouseMove(pnt, down)

		return
	}

	switch {
	case d.drag != nil:
		d.drag.pos = pnt.Sub(d.dragFrom)
		d.dirty = true

		return
	case d.capWindow != nil:
		wdw := d.capWindow
		if in := pnt.In(wdw.captionButton(wdw.capPressed)); in != wdw.capInside {
			wdw.capInside = in
			d.dirty = true
		}

		return
	}

	if n := len(d.popups); n > 0 {
		d.popups[n-1].mouseMove(pnt, down && (d.popupPress || d.capture != nil))

		d.dirty = true
		if d.popupPress {
			return
		}
	}

	if d.capture != nil {
		if clk, ok := d.capture.(clicker); ok {
			clk.mouseDrag(pnt.Sub(widgetOrigin(d.capture)))
		}

		return
	}

	d.hoverAt(pnt)
}

// hoverAt hot-tracks the list views and the menu bars as the mouse moves
// over them.
func (d *desktop) hoverAt(pnt image.Point) {
	var over widget
	wdw := d.windowAt(pnt)

	for _, mb := range d.menuBars() {
		mb.hover(pnt, wdw == mb.win && d.takesInput(wdw))
	}

	if d.takesInput(wdw) && pnt.In(wdw.clientRect()) {
		over = wdw.widgetAt(pnt)
	}

	if over != d.hover {
		if hvr, ok := d.hover.(hoverer); ok {
			hvr.hover(image.Point{}, true)
		}

		d.hover = over
	}

	if hvr, ok := over.(hoverer); ok {
		hvr.hover(pnt.Sub(widgetOrigin(over)), false)
	}
}

// menuBars lists the visible windows' menu bars.
func (d *desktop) menuBars() []*menuBar {
	var out []*menuBar

	for _, wdw := range d.windows {
		if wdw.visible && wdw.menu != nil {
			out = append(out, wdw.menu)
		}
	}

	return out
}

func (d *desktop) mouseUp(pnt image.Point) {
	d.dirty = true
	if d.menu != nil {
		d.menu.mouseUp(pnt)

		return
	}

	switch {
	case d.drag != nil:
		d.drag = nil

		return
	case d.capWindow != nil:
		d.captionUp(pnt)

		return
	}

	if d.popupPress {
		d.popupPress = false
		if n := len(d.popups); n > 0 && d.popups[n-1].mouseUp(pnt) {
			d.popups[n-1].dismiss()
		}

		return
	}

	if d.capture != nil {
		wgt := d.capture
		d.capture = nil

		if clk, ok := wgt.(clicker); ok {
			clk.mouseUp(pnt.Sub(widgetOrigin(wgt)))
		}

		// A press on a combo box's button can end on its list.
		if n := len(d.popups); n > 0 && pnt.In(d.popups[n-1].bounds()) && d.popups[n-1].mouseUp(pnt) {
			d.popups[n-1].dismiss()
		}
	}
}

// captionUp ends a press on a caption button, which acts if the mouse is
// still on it.
func (d *desktop) captionUp(pnt image.Point) {
	wdw := d.capWindow
	cbt := wdw.capPressed
	inside := pnt.In(wdw.captionButton(cbt))
	wdw.capPressed, d.capWindow = capNone, nil

	if !inside {
		return
	}

	switch cbt {
	case capClose:
		wdw.close()
	case capMax:
		if wdw.maxBox {
			wdw.toggleMaximize()
		}
	case capMin:
		if wdw.minBox && d.onMinimize != nil {
			d.onMinimize()
		}
	}
}

// wheel sends the mouse wheel to the popup, or to the focused control of
// the active window.
func (d *desktop) wheel(dy int) {
	if d.menu != nil {
		return
	}

	d.dirty = true
	if n := len(d.popups); n > 0 {
		d.popups[n-1].wheel(dy)

		return
	}

	if wdw := d.active; wdw != nil && wdw.focus != nil {
		if whl, ok := wdw.focus.(wheeler); ok && wdw.focus.base().usable() {
			whl.wheel(dy)
		}
	}
}

func (d *desktop) keyDown(key ebiten.Key, mds mods) {
	d.dirty = true
	if key == ebiten.KeyAltLeft || key == ebiten.KeyAltRight {
		d.altAlone = true

		return
	}

	d.altAlone = false
	if d.menu != nil {
		d.menu.key(key)

		return
	}

	wdw := d.active
	if wdw == nil {
		return
	}

	if d.menuKey(wdw, key, mds) {
		return
	}

	switch {
	case mds.alt && key == ebiten.KeyF4:
		if wdw.sysMenu && wdw.style != bsNone {
			wdw.close()
		}

		return
	case mds.alt && keyRune(key) != 0:
		// Alt and a letter: a button's accelerator.
		wdw.accelerator(keyRune(key))

		return
	}

	wdw.key(key, mds)
}

// menuKey gives a key to the active window's menu bar, and reports whether
// it took it: F10 enters the bar, Alt and a letter opens a menu, and a
// shortcut runs its command.
func (d *desktop) menuKey(wdw *window, key ebiten.Key, mds mods) bool {
	if wdw.menu == nil || d.blocked(wdw) || len(d.popups) > 0 {
		return false
	}

	switch {
	case key == ebiten.KeyF10 && !mds.shift && !mds.ctrl:
		d.menu = wdw.menu
		wdw.menu.enterKeyboard()
	case mds.alt && wdw.menu.openByKey(key):
		d.menu = wdw.menu
	default:
		return wdw.menu.shortcut(key, mds)
	}

	return true
}

func (d *desktop) keyUp(key ebiten.Key) {
	if (key == ebiten.KeyAltLeft || key == ebiten.KeyAltRight) && d.altAlone {
		d.altAlone = false
		d.dirty = true

		if d.menu != nil {
			d.menu.exit()

			return
		}

		if wdw := d.active; d.takesInput(wdw) && wdw.menu != nil {
			d.menu = wdw.menu
			wdw.menu.enterKeyboard()
		}

		return
	}

	if wdw := d.active; wdw != nil {
		if btn, ok := wdw.focus.(*button); ok {
			btn.keyUp(key)
		}
	}
}

func (d *desktop) char(chr rune) {
	if d.menu != nil {
		d.menu.char(chr)

		return
	}

	if wdw := d.active; wdw != nil {
		d.dirty = true

		wdw.char(chr)
	}
}

// keyRune returns the letter or digit a key types, lowercased, or 0.
func keyRune(key ebiten.Key) rune {
	switch {
	case key >= ebiten.KeyA && key <= ebiten.KeyZ:
		return 'a' + rune(key-ebiten.KeyA)
	case key >= ebiten.KeyDigit0 && key <= ebiten.KeyDigit9:
		return '0' + rune(key-ebiten.KeyDigit0)
	}

	return 0
}

// keyOrder is 0 for the modifier keys and 1 for the others.
func keyOrder(key ebiten.Key) int {
	switch key {
	case ebiten.KeyAltLeft, ebiten.KeyAltRight, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyShiftLeft, ebiten.KeyShiftRight, ebiten.KeyMetaLeft, ebiten.KeyMetaRight:
		return 0
	default:
		return 1
	}
}

// tick runs what time drives: auto-repeat, the menus' delays and the
// caret.
func (d *desktop) tick(now time.Time) {
	if n := len(d.popups); n > 0 && d.popups[n-1].tick(now) {
		d.dirty = true
	}

	if tck, ok := d.capture.(ticker); ok {
		tck.tick(now)
	}

	if d.menu != nil {
		d.menu.tick(now)
	}

	if wdw := d.active; wdw != nil && wdw.focus != nil {
		var since time.Duration

		switch f := wdw.focus.(type) {
		case *edit:
			since = now.Sub(f.tb.blink)
		case *spinEdit:
			since = now.Sub(f.tb.blink)
		case *comboBox:
			since = now.Sub(f.tb.blink)
		default:
			return
		}

		if on := blinkOn(since); on != d.caretOn {
			d.caretOn = on
			d.dirty = true
		}
	}
}

func (d *desktop) updateCursor(pnt image.Point) {
	cur := ebiten.CursorShapeDefault

	if d.menu == nil && len(d.popups) == 0 && d.drag == nil {
		wgt := d.capture
		if wgt == nil {
			if wdw := d.windowAt(pnt); wdw != nil && !d.blocked(wdw) && pnt.In(wdw.clientRect()) {
				wgt = wdw.widgetAt(pnt)
			}
		}

		if cu, ok := wgt.(cursorer); ok && wgt.base().usable() {
			cur = cu.cursor(pnt.Sub(widgetOrigin(wgt)))
		}
	}

	d.cursor = cur
}

// paint draws the desktop.
func (d *desktop) paint(img *image.RGBA) {
	cv := newCanvas(img)
	cv.fill(img.Rect, colDesktop)

	for _, wdw := range d.windows {
		wdw.paint(cv)
	}

	for _, pop := range d.popups {
		pop.paint(cv)
	}

	if d.menu != nil {
		d.menu.paintPopups(cv)
	}

	d.hint.paint(cv, d.res.sans)

	d.dirty = false
}

func (d *desktop) updateHint(pnt image.Point, cancel bool) {
	hnt := &d.hint
	wgt := d.hintTarget(pnt)

	if cancel || wgt != hnt.ctl {
		if !hnt.rect.Empty() {
			hnt.rect = image.Rectangle{}
			d.dirty = true
		}

		hnt.ctl, hnt.since, hnt.done = wgt, d.now, false
		if cancel {
			hnt.ctl = nil // CancelHint: the pause starts again
		}

		return
	}

	if wgt == nil || hnt.done {
		return
	}

	switch {
	case hnt.rect.Empty() && d.now.Sub(hnt.since) >= hintPause:
		// CalcHintRect adds 6 and 2 to the text's size, ActivateHint 4 more
		// to the height, and keeps the window on the screen.
		fnt := d.res.sans
		hnt.text = hintText(wgt)
		size := image.Pt(fnt.width(hnt.text)+6, fnt.height()+6)
		org := pnt.Add(image.Pt(0, hintCursorMargin))
		org.X = max(min(org.X, screenWidth-size.X), 0)
		org.Y = max(min(org.Y, screenHeight-size.Y), 0)
		hnt.rect = image.Rectangle{Min: org, Max: org.Add(size)}
		d.dirty = true
	case !hnt.rect.Empty() && d.now.Sub(hnt.since) >= hintPause+hintHidePause:
		hnt.rect = image.Rectangle{}
		hnt.done = true
		d.dirty = true
	}
}

// hintTarget returns the control under the mouse if its hint may show,
// or nil.
func (d *desktop) hintTarget(pnt image.Point) widget {
	if d.busy() {
		return nil
	}

	wdw := d.windowAt(pnt)
	if wdw == nil || !pnt.In(wdw.clientRect()) {
		return nil
	}

	wgt := wdw.widgetAt(pnt)
	if !hintable(wgt) {
		return nil
	}

	return wgt
}

// busy reports whether the desktop is in the middle of something that keeps
// hints from showing: a menu or a popup is open, a window is dragged, a
// control holds the mouse, or a modal window is up.
func (d *desktop) busy() bool {
	return d.menu != nil || len(d.popups) > 0 || d.drag != nil || d.capture != nil || d.topModal() != nil
}

// hintable reports whether control wgt has a hint to show: it is shown, has
// ShowHint on and a hint.
func hintable(wgt widget) bool {
	return wgt != nil && wgt.base().shown() && wgt.base().property("ShowHint") == "True" && hintText(wgt) != ""
}

// hintText is a control's hint, the part before a "|" (GetShortHint).
func hintText(wgt widget) string {
	text, _, _ := strings.Cut(wgt.base().property("Hint"), "|")

	return text
}

// The application's hint delays (HintPause, HintHidePause), and the arrow
// cursor's height below its hot spot, which a hint shows under
// (GetCursorHeightMargin).
const (
	hintPause        time.Duration = 500 * time.Millisecond
	hintHidePause    time.Duration = 2500 * time.Millisecond
	hintCursorMargin int           = 20
)

// hint is the hint window (THintWindow) of a control with ShowHint and a
// Hint: it shows once the mouse has been on the control for HintPause, and
// hides after HintHidePause, when the mouse leaves the control, or at a
// click or a key (IsHintMsg).
type hint struct {
	ctl   widget    // the control the mouse is on, if it has a hint
	since time.Time // when the mouse came onto it
	text  string
	rect  image.Rectangle // the window, while it shows
	done  bool            // it showed and hid: not again on this control
}

// paint draws the hint window: a black border, the tooltip colour, and the
// text 2 pixels into the client area.
func (h *hint) paint(cv canvas, fnt face) {
	if h.rect.Empty() {
		return
	}

	cv.fill(h.rect, colInfo)
	cv.frame(h.rect, colFrame)
	fnt.draw(cv.sub(h.rect), 3, 3, h.text, colText)
}
