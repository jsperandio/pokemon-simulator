// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// Window metrics of Windows 98's standard scheme.
const (
	captionHeight      int = 19 // SM_CYCAPTION: an 18-pixel bar and a line
	smallCaptionHeight int = 16 // SM_CYSMCAPTION
	menuHeight         int = 19 // SM_CYMENU
	captionButtonW     int = 16
	captionButtonH     int = 14
)

// borderStyle is a form's BorderStyle.
type borderStyle int

const (
	bsSizeable borderStyle = iota // the VCL's default
	bsNone
	bsSingle
	bsDialog
	bsToolWindow
)

// Caption buttons.
const (
	capNone int = iota
	capMin
	capMax
	capClose
)

// window is a form: a top-level window on the desktop.
type window struct {
	desk    *desktop
	res     *resources
	name    string
	caption string
	style   borderStyle
	sysMenu bool // biSystemMenu: the icon and the close button
	minBox  bool
	maxBox  bool
	pos     image.Point // the outer top-left corner, on the desktop
	client  image.Point // the client area's size
	design  image.Point // the client area's size in the layout
	fits    bool        // AutoSize: the client area is the controls' extent
	pinned  bool        // docked: the caption doesn't drag it
	color   color.RGBA
	visible bool
	active  bool
	kids    []widget
	focus   widget
	noFocus bool // the focused control was disabled, which left nothing with the focus
	menu    *menuBar

	modal      bool
	prevActive *window          // the window active before ShowModal
	onModal    func(result int) // when a modal window closes
	onClose    func()           // the close button or Alt+F4; nil hides the window
	onClick    func()           // a click on the form itself (OnClick)
	onKey      func(key ebiten.Key, mds mods) bool

	onShow, onHide           func()
	onActivate, onDeactivate func()

	maximized bool
	restore   image.Rectangle
	onResize  func()

	icon *image.NRGBA // the caption's icon, if not the program's

	capPressed int // the caption button held down
	capInside  bool
}

// borderStyles are the BorderStyles a form may have besides the default.
var borderStyles = map[string]borderStyle{
	"bsNone":       bsNone,
	"bsSingle":     bsSingle,
	"bsDialog":     bsDialog,
	"bsToolWindow": bsToolWindow,
}

// newWindow makes a form and its controls from its layout.
func newWindow(dsk *desktop, comp *layout.Component) *window {
	wdw := &window{
		desk:    dsk,
		res:     dsk.res,
		name:    comp.Name,
		caption: comp.Caption,
		visible: false,
	}
	props := comp.Props
	wdw.style = borderStyles[props["BorderStyle"]]
	wdw.color = vclColor(props["Color"], colFace)
	wdw.setIcons(props["BorderIcons"])

	for i := range comp.Children {
		if wgt := build(wdw, nil, &comp.Children[i]); wgt != nil {
			wdw.kids = append(wdw.kids, wgt)
		}
	}

	cw, _ := strconv.Atoi(props["ClientWidth"])
	ch, _ := strconv.Atoi(props["ClientHeight"])
	wdw.client = image.Pt(cw, ch)

	if cw == 0 || ch == 0 {
		wdw.client = wdw.extent()
	}

	wdw.design = wdw.client
	wdw.pos = image.Pt(comp.Bounds.X, comp.Bounds.Y)

	switch props["Position"] {
	case "poScreenCenter", "poDesktopCenter", "poMainFormCenter":
		wdw.center()
	}

	for _, wgt := range wdw.kids {
		wgt.base().parent = nil
	}

	keepHomes(wdw.kids, wdw.client)

	// The labels measure their captions as they load (TCustomLabel.Loaded).
	// The form's AutoSize applies once it has a window, after Position
	// placed it: the loading screen grows to the right, to its title.
	measureLabels(wdw.kids)

	wdw.fits = props["AutoSize"] == "True"
	wdw.fit()

	return wdw
}

// setIcons sets the caption's icon and buttons from BorderIcons: dialogs
// and tool windows have no minimize and maximize buttons.
func (w *window) setIcons(icons string) {
	if icons == "" {
		icons = "[biSystemMenu, biMinimize, biMaximize]"
	}

	w.sysMenu = strings.Contains(icons, "biSystemMenu")
	w.minBox = w.sysMenu && strings.Contains(icons, "biMinimize") && w.fullCaption()
	w.maxBox = w.sysMenu && strings.Contains(icons, "biMaximize") && w.fullCaption()
}

// fullCaption reports whether the window has a regular caption, which can
// have the icon and the minimize and maximize buttons: it is neither a
// dialog nor a tool window.
func (w *window) fullCaption() bool {
	return w.style != bsDialog && w.style != bsToolWindow
}

// hasSysIcon reports whether the caption shows the system menu's icon.
func (w *window) hasSysIcon() bool {
	return w.sysMenu && w.fullCaption()
}

// measureLabels sizes the labels among ws and their children to their
// captions, but the aligned ones.
func measureLabels(ws []widget) {
	for _, wgt := range ws {
		if lbl, ok := wgt.(*label); ok && lbl.property("Align") == "" {
			lbl.autoSize()
		}

		measureLabels(wgt.base().kids)
	}
}

// extent is the size that holds the visible controls.
func (w *window) extent() image.Point {
	var pnt image.Point

	for _, wgt := range w.kids {
		if ctl := wgt.base(); ctl.visible {
			pnt = image.Pt(max(pnt.X, ctl.rect.Max.X), max(pnt.Y, ctl.rect.Max.Y))
		}
	}

	return pnt
}

// fit is the form's AutoSize (TWinControl.DoAutoSize), which follows its
// controls as they show, hide or resize: the POKéDEX shrinks to its list
// while it has no selection. The controls anchored to the right or bottom
// edge move with it.
func (w *window) fit() {
	if w.fits {
		w.resize(w.extent())
	}
}

// widen makes the window's own client area dx wider than the layout's,
// its controls following their anchors.
func (w *window) widen(dx int) {
	w.resize(w.client.Add(image.Pt(dx, 0)))

	w.design = w.client
}

// resize gives the window a client area of the given size, and its
// controls follow their anchors.
func (w *window) resize(client image.Point) {
	if client == w.client {
		return
	}

	w.client = client
	for _, wgt := range w.kids {
		wgt.base().anchor(client)
	}

	if w.desk != nil {
		w.desk.dirty = true
	}
}

// center puts the window in the middle of the screen, as poScreenCenter
// does: (Screen.Width − Width) div 2.
func (w *window) center() {
	pnt := w.size()
	w.pos = image.Pt((screenWidth-pnt.X)/2, (screenHeight-pnt.Y)/2)
}

// find returns the control with the given name.
func (w *window) find(name string) widget {
	var (
		found widget
		walk  func(ws []widget)
	)
	walk = func(ws []widget) {
		for _, wgt := range ws {
			if found != nil {
				return
			}

			if wgt.base().name == name {
				found = wgt

				return
			}

			walk(wgt.base().kids)
		}
	}
	walk(w.kids)

	return found
}

// get returns a window's control by name; the layouts are fixed, so a
// missing one is a bug.
func get[T widget](wdw *window, name string) T {
	wgt, ok := wdw.find(name).(T)
	if !ok {
		panic(fmt.Sprintf("classic: %s has no %s of that kind", wdw.name, name))
	}

	return wgt
}

func (w *window) frameWidth() int {
	switch w.style {
	case bsNone:
		return 0
	case bsSizeable:
		return 4 // SM_CXFRAME
	default:
		return 3 // SM_CXFIXEDFRAME
	}
}

func (w *window) captionH() int {
	switch w.style {
	case bsNone:
		return 0
	case bsToolWindow:
		return smallCaptionHeight
	default:
		return captionHeight
	}
}

func (w *window) menuH() int {
	if w.menu == nil {
		return 0
	}

	return menuHeight
}

// nonClient is the size the frame, the caption and the menu bar add to
// the client area.
func (w *window) nonClient() image.Point {
	return w.size().Sub(w.client)
}

// size is the outer size.
func (w *window) size() image.Point {
	frm := w.frameWidth()

	return image.Pt(w.client.X+2*frm, w.client.Y+2*frm+w.captionH()+w.menuH())
}

func (w *window) rect() image.Rectangle {
	return image.Rectangle{
		Min: w.pos,
		Max: w.pos.Add(w.size()),
	}
}

func (w *window) clientOrigin() image.Point {
	frm := w.frameWidth()

	return w.pos.Add(image.Pt(frm, frm+w.captionH()+w.menuH()))
}

func (w *window) clientRect() image.Rectangle {
	pnt := w.clientOrigin()

	return image.Rectangle{
		Min: pnt,
		Max: pnt.Add(w.client),
	}
}

// captionRect is the caption bar on the desktop, without its bottom line.
func (w *window) captionRect() image.Rectangle {
	frm := w.frameWidth()
	pnt := w.size()

	return image.Rect(frm, frm, pnt.X-frm, frm+w.captionH()-1).Add(w.pos)
}

func (w *window) menuRect() image.Rectangle {
	frm := w.frameWidth()
	pnt := w.size()

	return image.Rect(frm, frm+w.captionH(), pnt.X-frm, frm+w.captionH()+w.menuH()).Add(w.pos)
}

// captionButton returns a caption button's rectangle on the desktop, or
// an empty one if the window has none.
func (w *window) captionButton(cbt int) image.Rectangle {
	if !w.sysMenu || w.style == bsNone || w.style == bsToolWindow {
		return image.Rectangle{}
	}

	cr := w.captionRect()
	closeB := image.Rect(cr.Max.X-2-captionButtonW, cr.Min.Y+2, cr.Max.X-2, cr.Min.Y+2+captionButtonH)
	maxB := closeB.Sub(image.Pt(captionButtonW+2, 0))
	minB := maxB.Sub(image.Pt(captionButtonW, 0))

	switch cbt {
	case capClose:
		return closeB
	case capMax:
		if w.maxBox || w.minBox {
			return maxB
		}
	case capMin:
		if w.maxBox || w.minBox {
			return minB
		}
	}

	return image.Rectangle{}
}

// The caption buttons' glyphs.
const (
	glyphClose string = `
##....##
.##..##.
..####..
...##...
..####..
.##..##.
##....##
`
	glyphMin string = `
######
######
`
	glyphMax string = `
#########
#########
#.......#
#.......#
#.......#
#.......#
#.......#
#.......#
#########
`
	glyphRestore string = `
..######
..######
..#....#
######.#
######.#
#....###
#....#..
#....#..
######..
`
)

func (w *window) paint(cv canvas) {
	if !w.visible {
		return
	}

	rect := w.rect()
	cv.fill(rect, colFace)

	if w.style != bsNone {
		cv.raised(rect)
		w.paintCaption(cv)
	}

	if w.menu != nil {
		w.menu.paint(cv, w.menuRect())
	}

	cr := w.clientRect()
	cv.fill(cr, w.color)

	sub := cv.sub(cr)

	for _, windowed := range []bool{false, true} {
		for _, wgt := range w.kids {
			if wgt.base().windowed == windowed {
				paintTree(sub, wgt)
			}
		}
	}
}

// paintCaption paints the caption bar: its gradient, the icon, the
// caption and the caption buttons.
func (w *window) paintCaption(cv canvas) {
	cr := w.captionRect()
	lft, rgt, fg := colInactive, colInactive2, colInactiveText

	if w.active {
		lft, rgt, fg = colCaption, colCaption2, colCaptionText
	}

	cv.gradient(cr, lft, rgt)

	x := cr.Min.X + 2
	icon := w.icon

	if icon == nil {
		icon = w.res.appIcon
	}

	if w.hasSysIcon() && icon != nil {
		cv.blend(x, cr.Min.Y+1, icon, icon.Rect)

		x += 18
	}

	textEnd := cr.Max.X

	for _, cbt := range []int{capMin, capMax, capClose} {
		if br := w.captionButton(cbt); !br.Empty() {
			textEnd = min(textEnd, br.Min.X-2)
		}
	}

	tc := cv.sub(image.Rect(cr.Min.X, cr.Min.Y, textEnd, cr.Max.Y))
	w.res.bold.draw(tc, x-cr.Min.X, (cr.Dy()-w.res.bold.height())/2, w.caption, fg)

	for _, cbt := range []int{capMin, capMax, capClose} {
		if br := w.captionButton(cbt); !br.Empty() {
			w.paintCaptionButton(cv, cbt, br)
		}
	}
}

// paintCaptionButton paints caption button cbt in its rectangle br: pushed
// while the mouse holds it, and greyed out when it does nothing.
func (w *window) paintCaptionButton(cv canvas, cbt int, br image.Rectangle) {
	off := cv.softButton(br, w.capPressed == cbt && w.capInside)

	glyph, gx, gy := glyphClose, 4, 3

	switch cbt {
	case capMin:
		glyph, gx, gy = glyphMin, 4, 9
	case capMax:
		glyph, gx, gy = glyphMax, 3, 2
		if w.maximized {
			glyph = glyphRestore
		}
	}

	enabled := cbt != capMax || w.maxBox
	if cbt == capMin {
		enabled = w.minBox
	}

	if !enabled {
		cv.embossed(br.Min.X+gx, br.Min.Y+gy, glyph)

		return
	}

	cv.pattern(br.Min.X+gx+off, br.Min.Y+gy+off, glyph, map[byte]color.RGBA{'#': colText})
}

// widgetAt returns the control under pnt (on the desktop), or nil.
func (w *window) widgetAt(pnt image.Point) widget {
	return hitKids(w.kids, pnt.Sub(w.clientOrigin()))
}

// tabOrder lists the controls that Tab visits, in order.
func (w *window) tabOrder() []widget {
	var (
		out  []widget
		walk func(ws []widget)
	)
	walk = func(ws []widget) {
		sorted := slices.Clone(ws)
		slices.SortStableFunc(sorted, func(fst, snd widget) int {
			return fst.base().tabOrder - snd.base().tabOrder
		})

		for _, kid := range sorted {
			ctl := kid.base()
			if !ctl.visible || !ctl.enabled {
				continue
			}

			if foc, ok := kid.(focuser); ok && foc.focusable() && ctl.tabStop {
				out = append(out, kid)
			}

			walk(ctl.kids)
		}
	}
	walk(w.kids)

	return out
}

// canFocus reports whether wgt can take the focus now.
func canFocus(wgt widget) bool {
	foc, ok := wgt.(focuser)

	return ok && foc.focusable() && wgt.base().shown() && wgt.base().usable()
}

// setFocus moves the focus; byMouse tells a click from the keyboard.
func (w *window) setFocus(wgt widget, byMouse bool) {
	if wgt == w.focus {
		return
	}

	if wgt != nil && !canFocus(wgt) {
		return
	}

	old := w.focus
	w.focus = wgt
	w.noFocus = false

	if lfc, ok := old.(focusLoser); ok {
		lfc.lostFocus()
	}

	if gfc, ok := wgt.(focusGainer); ok {
		gfc.gotFocus(byMouse)
	}

	w.desk.dirty = true
}

// focusFirst gives the focus to the first control in tab order, as a
// form that shows does.
func (w *window) focusFirst() {
	if w.focus != nil && canFocus(w.focus) {
		return
	}

	w.focus = nil
	for _, wgt := range w.tabOrder() {
		if canFocus(wgt) {
			w.setFocus(wgt, false)

			return
		}
	}
}

// tab moves the focus to the next (or previous) control in tab order.
func (w *window) tab(back bool) {
	order := w.tabOrder()
	if len(order) == 0 {
		return
	}

	step := 1
	if back {
		step = -1
	}

	i := slices.Index(order, w.focus)
	for range order {
		i = (i + step + len(order)) % len(order)

		if canFocus(order[i]) {
			w.setFocus(order[i], false)

			return
		}
	}
}

// buttons lists the window's visible buttons.
func (w *window) buttons() []*button {
	var (
		out  []*button
		walk func(ws []widget)
	)
	walk = func(ws []widget) {
		for _, wgt := range ws {
			if btn, ok := wgt.(*button); ok && btn.shown() && btn.usable() {
				out = append(out, btn)
			}

			walk(wgt.base().kids)
		}
	}
	walk(w.kids)

	return out
}

// key handles a key for the window: the focused control first, then
// Tab, Enter (the default button), Escape (the cancel button) and the
// buttons' accelerators. It reports whether the key was used.
func (w *window) key(key ebiten.Key, mds mods) bool {
	if w.onKey != nil && w.onKey(key, mds) {
		return true
	}

	if w.focus != nil {
		if kh, ok := w.focus.(keyer); ok && kh.key(key, mds) {
			return true
		}
	}

	switch key {
	case ebiten.KeyTab:
		w.tab(mds.shift)

		return true
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		return w.clickDefault()
	case ebiten.KeyEscape:
		return w.clickCancel()
	default:
		return false
	}
}

// clickDefault is Enter: it clicks the focused button, or else the default
// button, and reports whether there is one.
func (w *window) clickDefault() bool {
	if btn, ok := w.focus.(*button); ok {
		btn.click()

		return true
	}

	for _, btn := range w.buttons() {
		if btn.isDefault {
			btn.click()

			return true
		}
	}

	return false
}

// clickCancel is Escape: it clicks the cancel button, and reports whether
// there is one.
func (w *window) clickCancel() bool {
	for _, btn := range w.buttons() {
		if btn.isCancel {
			btn.click()

			return true
		}
	}

	return false
}

// char handles a typed character: the focused control's, or a button's
// accelerator when the focused control takes no text, as dialogs do.
func (w *window) char(chr rune) {
	switch w.focus.(type) {
	case *edit, *spinEdit, *comboBox:
		w.focus.(keyer).char(chr)

		return
	}

	if w.accelerator(chr) {
		return
	}

	if kh, ok := w.focus.(keyer); ok {
		kh.char(chr)
	}
}

// focusHidden reports whether the control with the focus is hidden.
func (w *window) focusHidden() bool {
	return w.focus != nil && !w.focus.base().shown()
}

// focusDisabled reports whether the control with the focus is disabled.
func (w *window) focusDisabled() bool {
	return w.focus != nil && !w.focus.base().usable()
}

// accelerator clicks the button whose accelerator chr is, toggles the check
// box whose it is, or focuses the control of the label whose it is; it
// reports whether there is one.
func (w *window) accelerator(chr rune) bool {
	lr := unicode.ToLower(chr)
	for _, btn := range w.buttons() {
		if accel(btn.caption) == lr {
			w.setFocus(btn, false)
			btn.click()

			return true
		}
	}

	var (
		target widget
		walk   func(ws []widget)
	)
	walk = func(ws []widget) {
		for _, wgt := range ws {
			if target == nil && accelTarget(wgt, lr) {
				target = wgt
			}

			walk(wgt.base().kids)
		}
	}
	walk(w.kids)

	switch tgt := target.(type) {
	case *checkBox:
		w.setFocus(tgt, false)
		tgt.toggle()

		return true
	case *label:
		if canFocus(tgt.focusControl) {
			w.setFocus(tgt.focusControl, false)

			return true
		}
	}

	return false
}

// accelTarget reports whether accelerator lr is wgt's: a check box's that
// can take the focus, or a label's that focuses a control.
func accelTarget(wgt widget, lr rune) bool {
	switch ctl := wgt.(type) {
	case *checkBox:
		return canFocus(ctl) && accel(ctl.caption) == lr
	case *label:
		return ctl.isAccelTarget(lr)
	}

	return false
}

// endModal closes a modal window with a result (ModalResult).
func (w *window) endModal(result int) {
	if !w.modal {
		return
	}

	w.desk.closeModal(w)

	if w.onModal != nil {
		w.onModal(result)
	}
}

// close is the close button: OnClose, or for a modal window a Cancel.
func (w *window) close() {
	if w.modal {
		w.endModal(mrCancel)

		return
	}

	if w.onClose != nil {
		w.onClose()

		return
	}

	w.desk.hide(w)
}

// Modal results.
const (
	mrNone   int = 0
	mrOk     int = 1
	mrCancel int = 2
	mrYes    int = 6
	mrNo     int = 7
)

// toggleMaximize maximizes the window to the screen, or restores it.
func (w *window) toggleMaximize() {
	if w.maximized {
		w.pos = w.restore.Min
		w.client = w.restore.Max
		w.maximized = false
	} else {
		w.restore = image.Rectangle{
			Min: w.pos,
			Max: w.client,
		}
		frm := w.frameWidth()
		w.pos = image.Pt(-frm, -frm)
		w.client = image.Pt(screenWidth, screenHeight-w.captionH()-w.menuH())
		w.maximized = true
	}

	if w.onResize != nil {
		w.onResize()
	}

	w.desk.dirty = true
}
