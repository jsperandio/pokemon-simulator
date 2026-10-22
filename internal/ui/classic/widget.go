// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"slices"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// A widget is a control of a window: the VCL's TControl.
type widget interface {
	// base returns the control embedded in the widget: the state every
	// control shares, such as its geometry, parent and visibility.
	base() *control
	// paint draws the control on a canvas whose origin is its top-left
	// corner; its children are painted afterwards.
	paint(cv canvas)
}

// Optional widget behaviour.
type (
	// clicker takes mouse presses. Points are relative to the widget;
	// after a press, the widget gets drags and the release wherever the
	// mouse goes.
	clicker interface {
		mouseDown(pnt image.Point, double bool)
		mouseDrag(pnt image.Point)
		mouseUp(pnt image.Point)
	}
	// hoverer follows the mouse while no button is down; out reports
	// that it left.
	hoverer interface {
		hover(pnt image.Point, out bool)
	}
	// keyer takes the keyboard while it has the focus.
	keyer interface {
		key(key ebiten.Key, mds mods) bool
		char(chr rune)
	}
	// keyUpper is told when a key goes up while it has the focus.
	keyUpper interface {
		keyUp(key ebiten.Key)
	}
	// wheeler takes the mouse wheel.
	wheeler interface {
		wheel(dy int)
	}
	// focuser can take the focus.
	focuser interface {
		focusable() bool
	}
	// focusLoser is told when it loses the focus.
	focusLoser interface {
		lostFocus()
	}
	// focusGainer is told when it gets the focus; byMouse reports that
	// the mouse gave it.
	focusGainer interface {
		gotFocus(byMouse bool)
	}
	// cursorer chooses the mouse cursor over it.
	cursorer interface {
		cursor(pnt image.Point) ebiten.CursorShapeType
	}
)

// mods are the modifier keys held.
type mods struct {
	shift bool
	ctrl  bool
	alt   bool
}

// control is the state every control shares: geometry, tree links,
// focus order and look.
type control struct {
	self     widget
	name     string
	rect     image.Rectangle // in the parent's coordinates
	visible  bool
	enabled  bool
	windowed bool // a TWinControl: painted above its graphic siblings
	tabStop  bool
	tabOrder int
	parent   widget
	kids     []widget
	wdw      *window

	caption   string
	color     color.RGBA // background
	font      face
	fontColor color.RGBA
	props     map[string]string

	onClick    func()
	onDblClick func()
}

func (c *control) base() *control {
	return c
}

// local is the control's rectangle in its own coordinates.
func (c *control) local() image.Rectangle {
	return image.Rectangle{Max: c.rect.Size()}
}

// shown reports whether the control and its parents are visible.
func (c *control) shown() bool {
	for up := c; up != nil; up = parentOf(up) {
		if !up.visible {
			return false
		}
	}

	return true
}

func parentOf(ctl *control) *control {
	if ctl.parent == nil {
		return nil
	}

	return ctl.parent.base()
}

// usable reports whether the control and its parents are enabled, as
// Windows disables a disabled window's children.
func (c *control) usable() bool {
	for up := c; up != nil; up = parentOf(up) {
		if !up.enabled {
			return false
		}
	}

	return true
}

// origin is the control's top-left corner in its window's client
// coordinates.
func (c *control) origin() image.Point {
	pnt := c.rect.Min
	for up := parentOf(c); up != nil; up = parentOf(up) {
		pnt = pnt.Add(up.rect.Min)
	}

	return pnt
}

// focused reports whether the control has its window's focus.
func (c *control) focused() bool {
	return c.wdw != nil && c.wdw.focus == c.self
}

// invalidate asks for a repaint.
func (c *control) invalidate() {
	if c.wdw != nil && c.wdw.desk != nil {
		c.wdw.desk.dirty = true
	}
}

func (c *control) setCaption(txt string) {
	if c.caption != txt {
		c.caption = txt
		c.invalidate()

		if asz, ok := c.self.(interface{ autoSize() }); ok {
			asz.autoSize()
		}
	}
}

func (c *control) setVisible(vis bool) {
	if c.visible != vis {
		c.visible = vis
		c.invalidate()

		if !vis && c.wdw != nil && c.wdw.focusHidden() {
			c.wdw.focus = nil
		}

		c.refit()
	}
}

func (c *control) setEnabled(ena bool) {
	if c.enabled != ena {
		c.enabled = ena
		c.invalidate()

		if wdw := c.wdw; !ena && wdw != nil && wdw.focusDisabled() {
			// A disabled window can't have the focus: Windows takes it
			// away, to no window.
			wdw.setFocus(nil, false)

			wdw.noFocus = true
		}
	}
}

func (c *control) setBounds(rect image.Rectangle) {
	if c.rect != rect {
		c.rect = rect
		c.invalidate()
		c.refit()
	}
}

// refit fits an AutoSize form to its controls again, after one of them
// showed, hid or moved.
func (c *control) refit() {
	if c.parent == nil && c.wdw != nil {
		c.wdw.fit()
	}
}

// property returns a design-time property, or "".
func (c *control) property(name string) string {
	return c.props[name]
}

// propertyInt returns an integer property, or def.
func (c *control) propertyInt(name string, def int) int {
	if val, err := strconv.Atoi(c.props[name]); err == nil {
		return val
	}

	return def
}

// classes make the widgets of the classes of components that have one.
var classes = map[string]func() widget{
	"TLabel": func() widget {
		return &label{}
	},
	"TButton": func() widget {
		return &button{}
	},
	"TGroupBox": func() widget {
		return &groupBox{}
	},
	"TCheckBox": func() widget {
		return &checkBox{}
	},
	"TListBox": func() widget {
		return &listBox{}
	},
	"TRadioGroup": func() widget {
		return &radioGroup{}
	},
	"TPanel": func() widget {
		return &panel{}
	},
	"TBevel": func() widget {
		return &bevel{}
	},
	"TImage": func() widget {
		return &picture{}
	},
	"TGauge": func() widget {
		return &gauge{}
	},
	"TListView": func() widget {
		return &listView{}
	},
	"TComboBox": func() widget {
		return &comboBox{}
	},
	"TSpinEdit": func() widget {
		return &spinEdit{}
	},
	"TEdit": func() widget {
		return &edit{}
	},
	"TMemo": func() widget {
		return &memo{}
	},
	"TMediaPlayer": func() widget {
		return &panel{}
	}, // hidden for good
}

// build makes the widget for a component and its children, or returns nil
// for the non-visual ones (menus, timers, dialogs, image lists).
func build(win *window, parent widget, comp *layout.Component) widget {
	newWidget, ok := classes[comp.Class]
	if !ok {
		return nil
	}

	wgt := newWidget()
	ctl := wgt.base()
	ctl.self = wgt
	ctl.wdw = win
	ctl.parent = parent
	ctl.name = comp.Name
	bds := comp.Bounds
	ctl.rect = image.Rect(bds.X, bds.Y, bds.X+bds.W, bds.Y+bds.H)
	ctl.visible = !comp.Hidden
	ctl.enabled = !comp.Disabled
	ctl.caption = comp.Caption
	ctl.props = comp.Props
	ctl.tabOrder = ctl.propertyInt("TabOrder", 0)
	ctl.font, ctl.fontColor = win.res.sans, colText
	pc := win.color

	if parent != nil {
		pc = parent.base().color
	}

	ctl.color = pc
	if ctl.property("ParentColor") != "True" {
		ctl.color = vclColor(ctl.property("Color"), pc)
	}

	if fnt := win.res.fontFor(ctl.props); fnt != nil {
		ctl.font = fnt
	}

	ctl.fontColor = vclColor(ctl.property("Font.Color"), colText)

	if stp, ok := wgt.(interface{ setup(*layout.Component) }); ok {
		stp.setup(comp)
	}

	for i := range comp.Children {
		if kid := build(win, wgt, &comp.Children[i]); kid != nil {
			ctl.kids = append(ctl.kids, kid)
		}
	}

	return wgt
}

// paintTree paints wgt and its children: the graphic children first, then
// the windowed ones, each group in creation order, as the VCL does.
func paintTree(cv canvas, wgt widget) {
	ctl := wgt.base()
	if !ctl.visible {
		return
	}

	sub := cv.sub(ctl.rect)
	wgt.paint(sub)

	for _, windowed := range []bool{false, true} {
		for _, kid := range ctl.kids {
			if kid.base().windowed == windowed {
				paintTree(sub, kid)
			}
		}
	}

	if pov, ok := wgt.(interface{ paintOver(canvas) }); ok {
		pov.paintOver(sub)
	}
}

// hit returns the deepest visible widget under pnt (in wgt's parent's
// coordinates), searching in the reverse of the painting order.
func hit(wgt widget, pnt image.Point) widget {
	ctl := wgt.base()
	if !ctl.visible || !pnt.In(ctl.rect) {
		return nil
	}

	if fnd := hitKids(ctl.kids, pnt.Sub(ctl.rect.Min)); fnd != nil {
		return fnd
	}

	return wgt
}

// hitKids returns the deepest visible widget under pnt among kids (in
// their parent's client coordinates), or nil: the windowed ones first, as
// they paint over the others, and the last painted first.
func hitKids(kids []widget, pnt image.Point) widget {
	for _, windowed := range []bool{true, false} {
		for _, kid := range slices.Backward(kids) {
			if kid.base().windowed != windowed {
				continue
			}

			if fnd := hit(kid, pnt); fnd != nil {
				return fnd
			}
		}
	}

	return nil
}

// label is a TLabel.
type label struct {
	control
	align       string // Alignment: taLeftJustify, taCenter, taRightJustify
	auto        bool   // AutoSize
	transparent bool
	wordWrap    bool
	// focusControl is what the label's accelerator focuses.
	focusControl widget
}

func (lb *label) setup(*layout.Component) {
	lb.align = lb.property("Alignment")
	lb.auto = lb.property("AutoSize") != "False"
	lb.transparent = lb.property("Transparent") == "True"
	lb.wordWrap = lb.property("WordWrap") == "True"
}

// autoSize resizes the label to its caption, as AutoSize does; an empty
// caption measures as a space.
func (lb *label) autoSize() {
	if !lb.auto {
		return
	}

	txt := plain(lb.caption)
	if txt == "" {
		txt = " "
	}

	wid := lb.font.width(txt)
	if mw := lb.propertyInt("Constraints.MaxWidth", 0); mw > 0 && wid > mw {
		wid = mw
	}

	lb.setBounds(image.Rect(lb.rect.Min.X, lb.rect.Min.Y, lb.rect.Min.X+wid, lb.rect.Min.Y+lb.font.height()))
}

// isAccelTarget reports whether accelerator lr focuses l's control: l is
// shown, has a control and its caption's accelerator is lr.
func (lb *label) isAccelTarget(lr rune) bool {
	return lb.focusControl != nil && lb.shown() && accel(lb.caption) == lr
}

func (lb *label) paint(cv canvas) {
	if !lb.transparent {
		cv.fill(lb.local(), lb.color)
	}

	lines := []string{lb.caption}
	if lb.wordWrap {
		lines = wrap(lb.font, lb.caption, lb.rect.Dx())
	}

	for i, lin := range lines {
		x, y := 0, i*lb.font.height()
		switch lb.align {
		case "taCenter":
			x = (lb.rect.Dx() - lb.font.width(plain(lin))) / 2
		case "taRightJustify":
			x = lb.rect.Dx() - lb.font.width(plain(lin))
		}

		drawText(cv, lb.font, x, y, lin, lb.fontColor, lb.usable())
	}
}

// button is a TButton.
type button struct {
	control
	isDefault, isCancel bool
	modalResult         int
	down                bool // the mouse is pressing it
	inside              bool // ... and is over it
	keyDown             bool // the space bar is pressing it
}

func (b *button) setup(*layout.Component) {
	b.windowed = true
	b.tabStop = b.property("TabStop") != "False"
	b.isDefault = b.property("Default") == "True"
	b.isCancel = b.property("Cancel") == "True"
	b.modalResult = b.propertyInt("ModalResult", 0)
}

func (b *button) focusable() bool {
	return true
}

// pushed reports whether it is drawn pushed in.
func (b *button) pushed() bool {
	return (b.down && b.inside) || b.keyDown
}

// looksDefault reports whether it has the default button's black frame:
// when it has the focus, or when it is the default and no other button
// has it.
func (b *button) looksDefault() bool {
	if b.focused() {
		return true
	}

	if !b.isDefault || b.wdw == nil {
		return false
	}

	_, other := b.wdw.focus.(*button)

	return !other && b.wdw.active
}

func (b *button) paint(cv canvas) {
	rect := b.local()
	cv.fill(rect, colFace)

	def := b.looksDefault()
	if def {
		cv.frame(rect, colFrame)

		rect = rect.Inset(1)
	}

	switch {
	case b.pushed() && def:
		cv.frame(rect, colShadow)
	case b.pushed():
		cv.sunkenSoft(rect)
	default:
		cv.raisedSoft(rect)
	}

	text := plain(b.caption)
	x := (b.rect.Dx() - b.font.width(text)) / 2
	y := (b.rect.Dy() - b.font.height()) / 2

	if b.pushed() {
		x, y = x+1, y+1
	}

	drawText(cv, b.font, x, y, b.caption, b.fontColor, b.usable())

	if b.focused() && b.wdw.active {
		cv.focusRect(b.local().Inset(3))
	}
}

func (b *button) mouseDown(image.Point, bool) {
	b.down, b.inside = true, true
	b.invalidate()
}

func (b *button) mouseDrag(pnt image.Point) {
	if in := pnt.In(b.local()); in != b.inside {
		b.inside = in
		b.invalidate()
	}
}

func (b *button) mouseUp(pnt image.Point) {
	click := b.down && pnt.In(b.local())
	b.down = false
	b.invalidate()

	if click {
		b.click()
	}
}

func (b *button) key(key ebiten.Key, _ mods) bool {
	if key == ebiten.KeySpace {
		b.keyDown = true
		b.invalidate()

		return true
	}

	return false
}

func (b *button) char(rune) {}

// keyUp ends a space-bar press.
func (b *button) keyUp(key ebiten.Key) {
	if key == ebiten.KeySpace && b.keyDown {
		b.keyDown = false
		b.invalidate()
		b.click()
	}
}

// click runs the button's OnClick, then closes a modal window with its
// ModalResult.
func (b *button) click() {
	if !b.usable() || !b.shown() {
		return
	}

	if b.onClick != nil {
		b.onClick()
	}

	if b.modalResult != 0 && b.wdw != nil {
		b.wdw.endModal(b.modalResult)
	}
}

// groupBox is a TGroupBox.
type groupBox struct {
	control
}

func (gb *groupBox) setup(*layout.Component) {
	gb.windowed = true
}

func (gb *groupBox) paint(cv canvas) {
	cv.fill(gb.local(), gb.color)
	paintGroupFrame(cv, gb.local(), gb.font, gb.caption, gb.color, gb.fontColor)
}

// paintGroupFrame draws a group box's etched frame and its caption, as
// TCustomGroupBox.Paint does.
func paintGroupFrame(cv canvas, rect image.Rectangle, fnt face, caption string, bg, fg color.RGBA) {
	top := fnt.height()/2 - 1
	cv.frame(image.Rect(rect.Min.X+1, top+1, rect.Max.X, rect.Max.Y), colHighlight)
	cv.frame(image.Rect(rect.Min.X, top, rect.Max.X-1, rect.Max.Y-1), colShadow)

	if caption != "" {
		wid := fnt.width(plain(caption))
		cv.fill(image.Rect(8, 0, 8+wid, fnt.height()), bg)
		drawCaption(cv, fnt, 8, 0, caption, fg)
	}
}

// radioGroup is a TRadioGroup: a group box of radio buttons laid out by
// TCustomRadioGroup.ArrangeButtons.
type radioGroup struct {
	control
	items    []string
	index    int // ItemIndex
	columns  int
	pressing int // the button the mouse is pressing, or -1
	inside   bool
}

func (rg *radioGroup) setup(comp *layout.Component) {
	rg.windowed = true
	rg.tabStop = rg.property("TabStop") == "True"

	rg.items = slices.Clone(comp.Items)
	rg.index = rg.propertyInt("ItemIndex", -1)
	rg.columns = max(rg.propertyInt("Columns", 1), 1)
	rg.pressing = -1
}

func (rg *radioGroup) focusable() bool {
	return len(rg.items) > 0
}

func (rg *radioGroup) setItems(items []string) {
	rg.items = items
	rg.invalidate()
}

func (rg *radioGroup) setIndex(i int) {
	if rg.index != i {
		rg.index = i
		rg.invalidate()
	}
}

// buttonRect is radio button i's rectangle.
func (rg *radioGroup) buttonRect(i int) image.Rectangle {
	n := len(rg.items)
	perCol := (n + rg.columns - 1) / rg.columns
	wid := (rg.rect.Dx() - 10) / rg.columns
	th := rg.font.height()
	space := rg.rect.Dy() - th - 5
	hgt := space / perCol
	top := th + 1 + space%perCol/2
	x := i/perCol*wid + 8
	y := i%perCol*hgt + top

	return image.Rect(x, y, x+wid, y+hgt)
}

func (rg *radioGroup) paint(cv canvas) {
	cv.fill(rg.local(), rg.color)
	paintGroupFrame(cv, rg.local(), rg.font, rg.caption, rg.color, rg.fontColor)

	for i, item := range rg.items {
		rect := rg.buttonRect(i)
		pushed := rg.pressing == i && rg.inside
		// The focus is on the checked item, or on the first when none is.
		current := i == rg.index || (rg.index < 0 && i == 0)
		focus := rg.focused() && rg.wdw.active && current
		paintRadio(cv.sub(rect), rect.Size(), rg.font, item, i == rg.index, pushed, rg.usable(), focus, rg.fontColor)
	}
}

// radioGlyph is Windows 98's radio button (DFCS_BUTTONRADIO): s shadow, k
// dark shadow, w highlight, l light, i the inside.
const radioGlyph string = `
....ssss....
..sskkkkss..
.skkiiiikkw.
.skiiiiiilw.
skiiiiiiiilw
skiiiiiiiilw
skiiiiiiiilw
skiiiiiiiilw
.skiiiiiilw.
.slliiiillw.
..wwllllww..
....wwww....
`

const radioDot string = `
.##.
####
####
.##.
`

// paintRadio draws a radio button and its caption in a rectangle of the
// given size. As the original's screenshots show them, the glyph's
// 13-pixel box and the line of text are centred with an odd pixel above
// them, the text is 17 pixels in, and the focus rectangle goes one pixel
// round the text but for the bottom.
func paintRadio(cv canvas, size image.Point, fnt face, caption string, checked, pushed, enabled, focus bool, fg color.RGBA) {
	gy := (size.Y - 13 + 1) >> 1
	inside := colWindow

	if pushed || !enabled {
		inside = colFace
	}

	cv.pattern(1, gy, radioGlyph, map[byte]color.RGBA{
		's': colShadow,
		'k': colDkShadow,
		'w': colHighlight,
		'l': colLight,
		'i': inside,
	})

	if checked {
		dot := colText
		if !enabled {
			dot = colShadow
		}

		cv.pattern(5, gy+4, radioDot, map[byte]color.RGBA{'#': dot})
	}

	ty := size.Y/2 - fnt.height()/2
	drawText(cv, fnt, 17, ty, caption, fg, enabled)

	if focus {
		wid := fnt.width(plain(caption))
		cv.focusRect(image.Rect(16, ty-1, 17+wid+1, ty+fnt.height()))
	}
}

func (rg *radioGroup) at(pnt image.Point) int {
	for i := range rg.items {
		if pnt.In(rg.buttonRect(i)) {
			return i
		}
	}

	return -1
}

func (rg *radioGroup) mouseDown(pnt image.Point, _ bool) {
	rg.pressing = rg.at(pnt)
	rg.inside = true
	rg.invalidate()
}

func (rg *radioGroup) mouseDrag(pnt image.Point) {
	if rg.pressing < 0 {
		return
	}

	if in := pnt.In(rg.buttonRect(rg.pressing)); in != rg.inside {
		rg.inside = in
		rg.invalidate()
	}
}

func (rg *radioGroup) mouseUp(pnt image.Point) {
	idx := rg.pressing
	rg.pressing = -1
	rg.invalidate()

	if idx >= 0 && pnt.In(rg.buttonRect(idx)) {
		rg.choose(idx)
	}
}

// choose checks item idx; OnClick runs only when the item changes, as a
// click on the checked button does nothing.
func (rg *radioGroup) choose(idx int) {
	if idx == rg.index || !rg.usable() {
		return
	}

	rg.setIndex(idx)

	if rg.onClick != nil {
		rg.onClick()
	}
}

func (rg *radioGroup) key(key ebiten.Key, _ mods) bool {
	n := len(rg.items)
	if n == 0 {
		return false
	}

	switch key {
	case ebiten.KeyUp, ebiten.KeyLeft:
		rg.choose((max(rg.index, 0) + n - 1) % n)
	case ebiten.KeyDown, ebiten.KeyRight:
		rg.choose((rg.index + 1) % n)
	default:
		return false
	}

	return true
}

func (rg *radioGroup) char(rune) {}

// checkBox is a TCheckBox.
type checkBox struct {
	control
	checked bool // Checked
	down    bool // the mouse is pressing it
	inside  bool // ... and is over it
	keyDown bool // the space bar is pressing it
}

func (chb *checkBox) setup(*layout.Component) {
	chb.windowed = true
	chb.tabStop = chb.property("TabStop") != "False"
	chb.checked = chb.property("Checked") == "True"
}

func (chb *checkBox) focusable() bool {
	return true
}

// setChecked checks or clears the box without running OnClick.
func (chb *checkBox) setChecked(checked bool) {
	if chb.checked != checked {
		chb.checked = checked
		chb.invalidate()
	}
}

// toggle checks or clears the box, then runs OnClick, as a click does.
func (chb *checkBox) toggle() {
	if !chb.usable() || !chb.shown() {
		return
	}

	chb.setChecked(!chb.checked)

	if chb.onClick != nil {
		chb.onClick()
	}
}

func (chb *checkBox) paint(cv canvas) {
	rect := chb.local()
	cv.fill(rect, chb.color)

	pushed := (chb.down && chb.inside) || chb.keyDown
	focus := chb.focused() && chb.wdw.active
	paintCheck(cv, rect.Size(), chb.font, chb.caption, chb.checked, pushed, chb.usable(), focus, chb.fontColor)
}

// checkGlyph is Windows 98's check box (DFCS_BUTTONCHECK), a sunken
// 13-pixel square: s shadow, k dark shadow, w highlight, l light, i the
// inside.
const checkGlyph string = `
ssssssssssssw
skkkkkkkkkklw
skiiiiiiiiilw
skiiiiiiiiilw
skiiiiiiiiilw
skiiiiiiiiilw
skiiiiiiiiilw
skiiiiiiiiilw
skiiiiiiiiilw
skiiiiiiiiilw
skiiiiiiiiilw
slllllllllllw
wwwwwwwwwwwww
`

// checkMark is the tick inside a checked box.
const checkMark string = `
......#
.....##
#...###
##.###.
#####..
.###...
..#....
`

// paintCheck draws a check box and its caption in a rectangle of the given
// size, laid out as paintRadio lays a radio button out: the box centred
// with an odd pixel above it, the text 17 pixels in, and the focus
// rectangle one pixel round the text but for the bottom.
func paintCheck(cv canvas, size image.Point, fnt face, caption string, checked, pushed, enabled, focus bool, fg color.RGBA) {
	gy := (size.Y - 13 + 1) >> 1
	inside := colWindow

	if pushed || !enabled {
		inside = colFace
	}

	cv.pattern(0, gy, checkGlyph, map[byte]color.RGBA{
		's': colShadow,
		'k': colDkShadow,
		'w': colHighlight,
		'l': colLight,
		'i': inside,
	})

	if checked {
		mark := colText
		if !enabled {
			mark = colShadow
		}

		cv.pattern(3, gy+3, checkMark, map[byte]color.RGBA{'#': mark})
	}

	ty := size.Y/2 - fnt.height()/2
	drawText(cv, fnt, 17, ty, caption, fg, enabled)

	if focus {
		wid := fnt.width(plain(caption))
		cv.focusRect(image.Rect(16, ty-1, 17+wid+1, ty+fnt.height()))
	}
}

func (chb *checkBox) mouseDown(image.Point, bool) {
	chb.down, chb.inside = true, true
	chb.invalidate()
}

func (chb *checkBox) mouseDrag(pnt image.Point) {
	if in := pnt.In(chb.local()); in != chb.inside {
		chb.inside = in
		chb.invalidate()
	}
}

func (chb *checkBox) mouseUp(pnt image.Point) {
	click := chb.down && pnt.In(chb.local())
	chb.down = false
	chb.invalidate()

	if click {
		chb.toggle()
	}
}

func (chb *checkBox) key(key ebiten.Key, _ mods) bool {
	if key == ebiten.KeySpace {
		chb.keyDown = true
		chb.invalidate()

		return true
	}

	return false
}

func (chb *checkBox) char(rune) {}

// keyUp ends a space-bar press, which toggles the box.
func (chb *checkBox) keyUp(key ebiten.Key) {
	if key == ebiten.KeySpace && chb.keyDown {
		chb.keyDown = false
		chb.invalidate()
		chb.toggle()
	}
}

// panel is a TPanel.
type panel struct {
	control
}

func (p *panel) setup(*layout.Component) {
	p.windowed = true
}

func (p *panel) paint(cv canvas) {
	rect := p.local()
	cv.fill(rect, p.color)

	if p.property("BevelOuter") != "bvNone" {
		cv.edge(rect, colHighlight, colShadow)
	}

	if p.caption != "" {
		wid := p.font.width(plain(p.caption))
		drawCaption(cv, p.font, (rect.Dx()-wid)/2, (rect.Dy()-p.font.height())/2, p.caption, p.fontColor)
	}
}

// bevel is a TBevel (the box shape, lowered, is all the game uses).
type bevel struct {
	control
}

func (b *bevel) paint(cv canvas) {
	rect := b.local()
	c1, c2 := colShadow, colHighlight

	if b.property("Style") == "bsRaised" {
		c1, c2 = c2, c1
	}

	// BevelRect(0, 0, W−1, H−1): two polylines, the second one ending
	// before its last point.
	wid, hgt := rect.Dx()-1, rect.Dy()-1
	cv.vline(0, 0, hgt+1, c1)
	cv.hline(0, wid, 0, c1)
	cv.vline(wid, 0, hgt+1, c2)
	cv.hline(1, wid+1, hgt, c2)
}

// picture is a TImage.
type picture struct {
	control
	img   *image.NRGBA
	alpha bool // blend the image's alpha channel, as icons have
}

func (p *picture) setup(*layout.Component) {
	p.img = p.wdw.res.picture(p.property("Picture"))
}

func (p *picture) paint(cv canvas) {
	switch {
	case p.img == nil:
	case p.alpha:
		cv.blend(0, 0, p.img, p.img.Rect)
	default:
		cv.draw(0, 0, p.img, p.img.Rect)
	}
}

// gauge is a TGauge from the Delphi samples, as a horizontal bar with its
// percentage XORed over it.
type gauge struct {
	control
	min, max, progress int
	fore, back         color.RGBA
}

func (g *gauge) setup(*layout.Component) {
	g.min = g.propertyInt("MinValue", 0)
	g.max = g.propertyInt("MaxValue", 100)
	g.progress = g.propertyInt("Progress", 0)
	g.fore = vclColor(g.property("ForeColor"), rgb(0x000000))
	g.back = vclColor(g.property("BackColor"), rgb(0xFFFFFF))
}

// set changes the gauge, clamping the progress as SetProgress does.
func (g *gauge) set(top, progress int, fore color.RGBA) {
	progress = max(min(progress, top), g.min)
	if g.max != top || g.progress != progress || g.fore != fore {
		g.max, g.progress, g.fore = top, progress, fore
		g.invalidate()
	}
}

// oneHundredth is the Extended constant 0.01 of SolveForX.
var oneHundredth = delphi.NewExtended(1).Quo(delphi.NewExtended(100))

// solveForX is Gauges.SolveForX: Trunc(z × (y × 0.01)), on the x87.
func solveForX(y, z int) int {
	pct := delphi.NewExtended(float64(y)).Mul(oneHundredth)

	return int(delphi.NewExtended(float64(z)).Mul(pct).Int().Float64())
}

// percent is PercentDone: Trunc(done × 100 / range).
func (g *gauge) percent() int {
	if g.max == g.min {
		return 0
	}

	return (g.progress - g.min) * 100 / (g.max - g.min)
}

func (g *gauge) paint(cv canvas) {
	rect := g.local()
	cv.fill(rect, colDkShadow) // the image starts black

	in := rect.Inset(1) // bsSingle
	cv.fill(in, g.back)

	// PaintAsBar: W is one pixel wider than the inside, and the bar's right
	// edge is FillSize itself.
	wid := in.Dx() + 1
	fill := min(solveForX(g.percent(), wid), wid)

	if fill > 0 {
		cv.fill(image.Rect(in.Min.X, in.Min.Y, fill, in.Max.Y).Intersect(in), g.fore)
	}

	// PaintAsText: white text XORed over the image, clipped to the inside.
	txt := strconv.Itoa(g.percent()) + "%"
	x := (wid - g.font.width(txt)) / 2
	y := (in.Dy() + 1 - g.font.height()) / 2
	tmp := image.NewRGBA(image.Rectangle{Max: rect.Size()})
	g.font.draw(newCanvas(tmp), x, y, txt, colHighlight)

	for py := in.Min.Y; py < in.Max.Y; py++ {
		for px := in.Min.X; px < in.Max.X; px++ {
			if tmp.RGBAAt(px, py).A != 0 {
				cv.invert(px, py)
			}
		}
	}
}
