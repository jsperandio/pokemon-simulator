// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"slices"
	"strconv"
	"strings"

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
	fits    bool        // AutoSize: the client area is the controls' extent
	color   color.RGBA
	visible bool
	active  bool
	kids    []widget
	focus   widget
	noFocus bool // the focused control was disabled, which left nothing with the focus
	menu    *menuBar

	modal       bool
	modalResult int
	prevActive  *window          // the window active before ShowModal
	onModal     func(result int) // when a modal window closes
	onClose     func()           // the close button; nil hides the window
	onClick     func()           // a click on the form itself (OnClick)
	onKey       func(k ebiten.Key, m mods) bool

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
func newWindow(d *desktop, comp *layout.Component) *window {
	w := &window{
		desk:    d,
		res:     d.res,
		name:    comp.Name,
		caption: comp.Caption,
		visible: false,
	}
	props := comp.Props
	w.style = borderStyles[props["BorderStyle"]]
	w.color = vclColor(props["Color"], colFace)
	w.setIcons(props["BorderIcons"])

	for i := range comp.Children {
		if k := build(w, nil, &comp.Children[i]); k != nil {
			w.kids = append(w.kids, k)
		}
	}

	cw, _ := strconv.Atoi(props["ClientWidth"])
	ch, _ := strconv.Atoi(props["ClientHeight"])
	w.client = image.Pt(cw, ch)

	if cw == 0 || ch == 0 {
		w.client = w.extent()
	}

	w.pos = image.Pt(comp.Bounds.X, comp.Bounds.Y)

	switch props["Position"] {
	case "poScreenCenter", "poDesktopCenter", "poMainFormCenter":
		w.center()
	}

	for _, k := range w.kids {
		k.base().parent = nil
	}

	// The labels measure their captions as they load (TCustomLabel.Loaded).
	// The form's AutoSize applies once it has a window, after Position
	// placed it: the loading screen grows to the right, to its title.
	measureLabels(w.kids)

	w.fits = props["AutoSize"] == "True"
	w.fit()

	return w
}

// setIcons sets the caption's icon and buttons from BorderIcons: dialogs
// and tool windows have no minimize and maximize buttons.
func (w *window) setIcons(icons string) {
	if icons == "" {
		icons = "[biSystemMenu, biMinimize, biMaximize]"
	}

	w.sysMenu = strings.Contains(icons, "biSystemMenu")
	w.minBox = w.sysMenu && strings.Contains(icons, "biMinimize") &&
		w.style != bsDialog && w.style != bsToolWindow
	w.maxBox = w.sysMenu && strings.Contains(icons, "biMaximize") &&
		w.style != bsDialog && w.style != bsToolWindow
}

// measureLabels sizes the labels among ws and their children to their
// captions, but the aligned ones.
func measureLabels(ws []widget) {
	for _, k := range ws {
		if l, ok := k.(*label); ok && l.prop("Align") == "" {
			l.autoSize()
		}

		measureLabels(k.base().kids)
	}
}

// extent is the size that holds the visible controls.
func (w *window) extent() image.Point {
	var e image.Point

	for _, k := range w.kids {
		if c := k.base(); c.visible {
			e = image.Pt(max(e.X, c.r.Max.X), max(e.Y, c.r.Max.Y))
		}
	}

	return e
}

// fit is the form's AutoSize (TWinControl.DoAutoSize), which follows its
// controls as they show, hide or resize: the POKéDEX shrinks to its list
// while it has no selection. The controls anchored to the right or bottom
// edge move with it.
func (w *window) fit() {
	if !w.fits {
		return
	}

	d := w.extent().Sub(w.client)
	if d == (image.Point{}) {
		return
	}

	w.client = w.client.Add(d)
	for _, k := range w.kids {
		c := k.base()
		anchors := c.prop("Anchors")

		if strings.Contains(anchors, "akRight") {
			if strings.Contains(anchors, "akLeft") {
				c.r.Max.X += d.X
			} else {
				c.r = c.r.Add(image.Pt(d.X, 0))
			}
		}

		if strings.Contains(anchors, "akBottom") {
			if strings.Contains(anchors, "akTop") {
				c.r.Max.Y += d.Y
			} else {
				c.r = c.r.Add(image.Pt(0, d.Y))
			}
		}
	}

	if w.desk != nil {
		w.desk.dirty = true
	}
}

// center puts the window in the middle of the screen, as poScreenCenter
// does: (Screen.Width − Width) div 2.
func (w *window) center() {
	s := w.size()
	w.pos = image.Pt((screenWidth-s.X)/2, (screenHeight-s.Y)/2)
}

// find returns the control with the given name.
func (w *window) find(name string) widget {
	var (
		found widget
		walk  func(ws []widget)
	)
	walk = func(ws []widget) {
		for _, k := range ws {
			if found != nil {
				return
			}

			if k.base().name == name {
				found = k

				return
			}

			walk(k.base().kids)
		}
	}
	walk(w.kids)

	return found
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

// size is the outer size.
func (w *window) size() image.Point {
	f := w.frameWidth()

	return image.Pt(w.client.X+2*f, w.client.Y+2*f+w.captionH()+w.menuH())
}

func (w *window) rect() image.Rectangle {
	return image.Rectangle{Min: w.pos, Max: w.pos.Add(w.size())}
}

func (w *window) clientOrigin() image.Point {
	f := w.frameWidth()

	return w.pos.Add(image.Pt(f, f+w.captionH()+w.menuH()))
}

func (w *window) clientRect() image.Rectangle {
	o := w.clientOrigin()

	return image.Rectangle{Min: o, Max: o.Add(w.client)}
}

// captionRect is the caption bar on the desktop, without its bottom line.
func (w *window) captionRect() image.Rectangle {
	f := w.frameWidth()
	s := w.size()

	return image.Rect(f, f, s.X-f, f+w.captionH()-1).Add(w.pos)
}

func (w *window) menuRect() image.Rectangle {
	f := w.frameWidth()
	s := w.size()

	return image.Rect(f, f+w.captionH(), s.X-f, f+w.captionH()+w.menuH()).Add(w.pos)
}

// captionButton returns a caption button's rectangle on the desktop, or
// an empty one if the window has none.
func (w *window) captionButton(b int) image.Rectangle {
	if !w.sysMenu || w.style == bsNone || w.style == bsToolWindow {
		return image.Rectangle{}
	}

	cr := w.captionRect()
	closeB := image.Rect(cr.Max.X-2-captionButtonW, cr.Min.Y+2, cr.Max.X-2, cr.Min.Y+2+captionButtonH)
	maxB := closeB.Sub(image.Pt(captionButtonW+2, 0))
	minB := maxB.Sub(image.Pt(captionButtonW, 0))

	switch b {
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

	r := w.rect()
	cv.fill(r, colFace)

	if w.style != bsNone {
		cv.raised(r)
		w.paintCaption(cv)
	}

	if w.menu != nil {
		w.menu.paint(cv, w.menuRect())
	}

	cr := w.clientRect()
	cv.fill(cr, w.color)

	sub := cv.sub(cr)

	for _, windowed := range []bool{false, true} {
		for _, k := range w.kids {
			if k.base().windowed == windowed {
				paintTree(sub, k)
			}
		}
	}
}

// paintCaption paints the caption bar: its gradient, the icon, the
// caption and the caption buttons.
func (w *window) paintCaption(cv canvas) {
	cr := w.captionRect()
	a, b, fg := colInactive, colInactive2, colInactiveText

	if w.active {
		a, b, fg = colCaption, colCaption2, colCaptionText
	}

	cv.gradient(cr, a, b)

	x := cr.Min.X + 2
	icon := w.icon

	if icon == nil {
		icon = w.res.appIcon
	}

	if w.sysMenu && w.style != bsDialog && w.style != bsToolWindow && icon != nil {
		cv.blend(x, cr.Min.Y+1, icon, icon.Rect)

		x += 18
	}

	textEnd := cr.Max.X

	for _, b := range []int{capMin, capMax, capClose} {
		if br := w.captionButton(b); !br.Empty() {
			textEnd = min(textEnd, br.Min.X-2)
		}
	}

	tc := cv.sub(image.Rect(cr.Min.X, cr.Min.Y, textEnd, cr.Max.Y))
	w.res.bold.draw(tc, x-cr.Min.X, (cr.Dy()-w.res.bold.height())/2, w.caption, fg)

	for _, b := range []int{capMin, capMax, capClose} {
		if br := w.captionButton(b); !br.Empty() {
			w.paintCaptionButton(cv, b, br)
		}
	}
}

// paintCaptionButton paints caption button b in its rectangle br: pushed
// while the mouse holds it, and greyed out when it does nothing.
func (w *window) paintCaptionButton(cv canvas, b int, br image.Rectangle) {
	off := cv.softButton(br, w.capPressed == b && w.capInside)

	glyph, gx, gy := glyphClose, 4, 3

	switch b {
	case capMin:
		glyph, gx, gy = glyphMin, 4, 9
	case capMax:
		glyph, gx, gy = glyphMax, 3, 2
		if w.maximized {
			glyph = glyphRestore
		}
	}

	enabled := b != capMax || w.maxBox
	if b == capMin {
		enabled = w.minBox
	}

	if !enabled {
		cv.embossed(br.Min.X+gx, br.Min.Y+gy, glyph)

		return
	}

	cv.pattern(br.Min.X+gx+off, br.Min.Y+gy+off, glyph, map[byte]color.RGBA{'#': colText})
}

// widgetAt returns the control under p (on the desktop), or nil.
func (w *window) widgetAt(p image.Point) widget {
	q := p.Sub(w.clientOrigin())
	for _, windowed := range []bool{true, false} {
		for _, k := range slices.Backward(w.kids) {
			if k.base().windowed != windowed {
				continue
			}

			if h := hit(k, q); h != nil {
				return h
			}
		}
	}

	return nil
}

// tabOrder lists the controls that Tab visits, in order.
func (w *window) tabOrder() []widget {
	var (
		out  []widget
		walk func(ws []widget)
	)
	walk = func(ws []widget) {
		sorted := slices.Clone(ws)
		slices.SortStableFunc(sorted, func(a, b widget) int {
			return a.base().tabOrder - b.base().tabOrder
		})

		for _, k := range sorted {
			c := k.base()
			if !c.visible || !c.enabled {
				continue
			}

			if f, ok := k.(focuser); ok && f.focusable() && c.tabStop {
				out = append(out, k)
			}

			walk(c.kids)
		}
	}
	walk(w.kids)

	return out
}

// canFocus reports whether k can take the focus now.
func canFocus(k widget) bool {
	f, ok := k.(focuser)

	return ok && f.focusable() && k.base().shown() && k.base().usable()
}

// setFocus moves the focus; byMouse tells a click from the keyboard.
func (w *window) setFocus(k widget, byMouse bool) {
	if k == w.focus {
		return
	}

	if k != nil && !canFocus(k) {
		return
	}

	old := w.focus
	w.focus = k
	w.noFocus = false

	if l, ok := old.(interface{ lostFocus() }); ok {
		l.lostFocus()
	}

	if g, ok := k.(interface{ gotFocus(bool) }); ok {
		g.gotFocus(byMouse)
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
	for _, k := range w.tabOrder() {
		if canFocus(k) {
			w.setFocus(k, false)

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
		for _, k := range ws {
			if b, ok := k.(*button); ok && b.shown() && b.usable() {
				out = append(out, b)
			}

			walk(k.base().kids)
		}
	}
	walk(w.kids)

	return out
}

// key handles a key for the window: the focused control first, then
// Tab, Enter (the default button), Escape (the cancel button) and the
// buttons' accelerators. It reports whether the key was used.
func (w *window) key(k ebiten.Key, m mods) bool {
	if w.onKey != nil && w.onKey(k, m) {
		return true
	}

	if w.focus != nil {
		if kh, ok := w.focus.(keyer); ok && kh.key(k, m) {
			return true
		}
	}

	switch k {
	case ebiten.KeyTab:
		w.tab(m.shift)

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
	if b, ok := w.focus.(*button); ok {
		b.click()

		return true
	}

	for _, b := range w.buttons() {
		if b.isDefault {
			b.click()

			return true
		}
	}

	return false
}

// clickCancel is Escape: it clicks the cancel button, and reports whether
// there is one.
func (w *window) clickCancel() bool {
	for _, b := range w.buttons() {
		if b.isCancel {
			b.click()

			return true
		}
	}

	return false
}

// char handles a typed character: the focused control's, or a button's
// accelerator when the focused control takes no text, as dialogs do.
func (w *window) char(r rune) {
	switch w.focus.(type) {
	case *edit, *spinEdit, *comboBox:
		w.focus.(keyer).char(r)

		return
	}

	if w.accelerator(r) {
		return
	}

	if kh, ok := w.focus.(keyer); ok {
		kh.char(r)
	}
}

// accelerator clicks the button whose accelerator r is, or focuses the
// control of the label whose it is; it reports whether there is one.
func (w *window) accelerator(r rune) bool {
	lr := []rune(strings.ToLower(string(r)))[0]
	for _, b := range w.buttons() {
		if accel(b.caption) == lr {
			w.setFocus(b, false)
			b.click()

			return true
		}
	}

	var (
		target widget
		walk   func(ws []widget)
	)
	walk = func(ws []widget) {
		for _, k := range ws {
			if l, ok := k.(*label); ok && target == nil && l.focusControl != nil && l.shown() && accel(l.caption) == lr {
				target = l.focusControl
			}

			walk(k.base().kids)
		}
	}
	walk(w.kids)

	if target != nil && canFocus(target) {
		w.setFocus(target, false)

		return true
	}

	return false
}

// endModal closes a modal window with a result (ModalResult).
func (w *window) endModal(result int) {
	if !w.modal {
		return
	}

	w.modalResult = result
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
		w.restore = image.Rectangle{Min: w.pos, Max: w.client}
		f := w.frameWidth()
		w.pos = image.Pt(-f, -f)
		w.client = image.Pt(screenWidth, screenHeight-w.captionH()-w.menuH())
		w.maximized = true
	}

	if w.onResize != nil {
		w.onResize()
	}

	w.desk.dirty = true
}
