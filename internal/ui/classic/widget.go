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
		mouseDown(p image.Point, double bool)
		mouseDrag(p image.Point)
		mouseUp(p image.Point)
	}
	// hoverer follows the mouse while no button is down; out reports
	// that it left.
	hoverer interface {
		hover(p image.Point, out bool)
	}
	// keyer takes the keyboard while it has the focus.
	keyer interface {
		key(k ebiten.Key, m mods) bool
		char(r rune)
	}
	// wheeler takes the mouse wheel.
	wheeler interface {
		wheel(dy int)
	}
	// focuser can take the focus.
	focuser interface {
		focusable() bool
	}
	// cursorer chooses the mouse cursor over it.
	cursorer interface {
		cursor(p image.Point) ebiten.CursorShapeType
	}
)

// mods are the modifier keys held.
type mods struct{ shift, ctrl, alt bool }

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

func parentOf(c *control) *control {
	if c.parent == nil {
		return nil
	}

	return c.parent.base()
}

// origin is the control's top-left corner in its window's client
// coordinates.
func (c *control) origin() image.Point {
	p := c.rect.Min
	for up := parentOf(c); up != nil; up = parentOf(up) {
		p = p.Add(up.rect.Min)
	}

	return p
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

func (c *control) setCaption(s string) {
	if c.caption != s {
		c.caption = s
		c.invalidate()

		if a, ok := c.self.(interface{ autoSize() }); ok {
			a.autoSize()
		}
	}
}

func (c *control) setVisible(v bool) {
	if c.visible != v {
		c.visible = v
		c.invalidate()

		if !v && c.wdw != nil && c.wdw.focusHidden() {
			c.wdw.focus = nil
		}

		c.refit()
	}
}

func (c *control) setEnabled(v bool) {
	if c.enabled != v {
		c.enabled = v
		c.invalidate()

		if w := c.wdw; !v && w != nil && w.focusDisabled() {
			// A disabled window can't have the focus: Windows takes it
			// away, to no window.
			w.setFocus(nil, false)

			w.noFocus = true
		}
	}
}

func (c *control) setBounds(r image.Rectangle) {
	if c.rect != r {
		c.rect = r
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
	if v, err := strconv.Atoi(c.props[name]); err == nil {
		return v
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

	wdg := newWidget()
	c := wdg.base()
	c.self = wdg
	c.wdw = win
	c.parent = parent
	c.name = comp.Name
	bds := comp.Bounds
	c.rect = image.Rect(bds.X, bds.Y, bds.X+bds.W, bds.Y+bds.H)
	c.visible = !comp.Hidden
	c.enabled = !comp.Disabled
	c.caption = comp.Caption
	c.props = comp.Props
	c.tabOrder = c.propertyInt("TabOrder", 0)
	c.font, c.fontColor = win.res.sans, colText
	pc := win.color

	if parent != nil {
		pc = parent.base().color
	}

	c.color = pc
	if c.property("ParentColor") != "True" {
		c.color = vclColor(c.property("Color"), pc)
	}

	if f := win.res.fontFor(c.props); f != nil {
		c.font = f
	}

	c.fontColor = vclColor(c.property("Font.Color"), colText)

	if s, ok := wdg.(interface{ setup(*layout.Component) }); ok {
		s.setup(comp)
	}

	for i := range comp.Children {
		if k := build(win, wdg, &comp.Children[i]); k != nil {
			c.kids = append(c.kids, k)
		}
	}

	return wdg
}

// paintTree paints w and its children: the graphic children first, then
// the windowed ones, each group in creation order, as the VCL does.
func paintTree(cv canvas, w widget) {
	c := w.base()
	if !c.visible {
		return
	}

	sub := cv.sub(c.rect)
	w.paint(sub)

	for _, windowed := range []bool{false, true} {
		for _, k := range c.kids {
			if k.base().windowed == windowed {
				paintTree(sub, k)
			}
		}
	}

	if p, ok := w.(interface{ paintOver(canvas) }); ok {
		p.paintOver(sub)
	}
}

// hit returns the deepest visible widget under p (in w's parent's
// coordinates), searching in the reverse of the painting order.
func hit(w widget, p image.Point) widget {
	c := w.base()
	if !c.visible || !p.In(c.rect) {
		return nil
	}

	q := p.Sub(c.rect.Min)
	for _, windowed := range []bool{true, false} {
		for _, k := range slices.Backward(c.kids) {
			if k.base().windowed != windowed {
				continue
			}

			if h := hit(k, q); h != nil {
				return h
			}
		}
	}

	return w
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

func (l *label) setup(*layout.Component) {
	l.align = l.property("Alignment")
	l.auto = l.property("AutoSize") != "False"
	l.transparent = l.property("Transparent") == "True"
	l.wordWrap = l.property("WordWrap") == "True"
}

// autoSize resizes the label to its caption, as AutoSize does; an empty
// caption measures as a space.
func (l *label) autoSize() {
	if !l.auto {
		return
	}

	s := plain(l.caption)
	if s == "" {
		s = " "
	}

	w := l.font.width(s)
	if mw := l.propertyInt("Constraints.MaxWidth", 0); mw > 0 && w > mw {
		w = mw
	}

	l.setBounds(image.Rect(l.rect.Min.X, l.rect.Min.Y, l.rect.Min.X+w, l.rect.Min.Y+l.font.height()))
}

// isAccelTarget reports whether accelerator lr focuses l's control: l is
// shown, has a control and its caption's accelerator is lr.
func (l *label) isAccelTarget(lr rune) bool {
	return l.focusControl != nil && l.shown() && accel(l.caption) == lr
}

func (l *label) paint(cv canvas) {
	if !l.transparent {
		cv.fill(l.local(), l.color)
	}

	lines := []string{l.caption}
	if l.wordWrap {
		lines = wrap(l.font, l.caption, l.rect.Dx())
	}

	for i, s := range lines {
		x, y := 0, i*l.font.height()
		switch l.align {
		case "taCenter":
			x = (l.rect.Dx() - l.font.width(plain(s))) / 2
		case "taRightJustify":
			x = l.rect.Dx() - l.font.width(plain(s))
		}

		drawText(cv, l.font, x, y, s, l.fontColor, l.usable())
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
	r := b.local()
	cv.fill(r, colFace)

	def := b.looksDefault()
	if def {
		cv.frame(r, colFrame)

		r = r.Inset(1)
	}

	switch {
	case b.pushed() && def:
		cv.frame(r, colShadow)
	case b.pushed():
		cv.sunkenSoft(r)
	default:
		cv.raisedSoft(r)
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

func (b *button) mouseDrag(p image.Point) {
	if in := p.In(b.local()); in != b.inside {
		b.inside = in
		b.invalidate()
	}
}

func (b *button) mouseUp(p image.Point) {
	click := b.down && p.In(b.local())
	b.down = false
	b.invalidate()

	if click {
		b.click()
	}
}

func (b *button) key(k ebiten.Key, _ mods) bool {
	if k == ebiten.KeySpace {
		b.keyDown = true
		b.invalidate()

		return true
	}

	return false
}

func (b *button) char(rune) {}

// keyUp ends a space-bar press.
func (b *button) keyUp(k ebiten.Key) {
	if k == ebiten.KeySpace && b.keyDown {
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
type groupBox struct{ control }

func (g *groupBox) setup(*layout.Component) {
	g.windowed = true
}

func (g *groupBox) paint(cv canvas) {
	cv.fill(g.local(), g.color)
	paintGroupFrame(cv, g.local(), g.font, g.caption, g.color, g.fontColor)
}

// paintGroupFrame draws a group box's etched frame and its caption, as
// TCustomGroupBox.Paint does.
func paintGroupFrame(cv canvas, r image.Rectangle, f face, caption string, bg, fg color.RGBA) {
	top := f.height()/2 - 1
	cv.frame(image.Rect(r.Min.X+1, top+1, r.Max.X, r.Max.Y), colHighlight)
	cv.frame(image.Rect(r.Min.X, top, r.Max.X-1, r.Max.Y-1), colShadow)

	if caption != "" {
		w := f.width(plain(caption))
		cv.fill(image.Rect(8, 0, 8+w, f.height()), bg)
		drawCaption(cv, f, 8, 0, caption, fg)
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

func (g *radioGroup) setup(comp *layout.Component) {
	g.windowed = true
	g.tabStop = g.property("TabStop") == "True"

	g.items = append([]string(nil), comp.Items...)
	g.index = g.propertyInt("ItemIndex", -1)
	g.columns = max(g.propertyInt("Columns", 1), 1)
	g.pressing = -1
}

func (g *radioGroup) focusable() bool {
	return len(g.items) > 0
}

func (g *radioGroup) setItems(items []string) {
	g.items = items
	g.invalidate()
}

func (g *radioGroup) setIndex(i int) {
	if g.index != i {
		g.index = i
		g.invalidate()
	}
}

// buttonRect is radio button i's rectangle.
func (g *radioGroup) buttonRect(i int) image.Rectangle {
	n := len(g.items)
	perCol := (n + g.columns - 1) / g.columns
	w := (g.rect.Dx() - 10) / g.columns
	th := g.font.height()
	space := g.rect.Dy() - th - 5
	h := space / perCol
	top := th + 1 + space%perCol/2
	x := i/perCol*w + 8
	y := i%perCol*h + top

	return image.Rect(x, y, x+w, y+h)
}

func (g *radioGroup) paint(cv canvas) {
	cv.fill(g.local(), g.color)
	paintGroupFrame(cv, g.local(), g.font, g.caption, g.color, g.fontColor)

	for i, item := range g.items {
		r := g.buttonRect(i)
		pushed := g.pressing == i && g.inside
		// The focus is on the checked item, or on the first when none is.
		current := i == g.index || (g.index < 0 && i == 0)
		focus := g.focused() && g.wdw.active && current
		paintRadio(cv.sub(r), r.Size(), g.font, item, i == g.index, pushed, g.usable(), focus, g.fontColor)
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
func paintRadio(cv canvas, size image.Point, f face, caption string, checked, pushed, enabled, focus bool, fg color.RGBA) {
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

	ty := size.Y/2 - f.height()/2
	drawText(cv, f, 17, ty, caption, fg, enabled)

	if focus {
		w := f.width(plain(caption))
		cv.focusRect(image.Rect(16, ty-1, 17+w+1, ty+f.height()))
	}
}

func (g *radioGroup) at(p image.Point) int {
	for i := range g.items {
		if p.In(g.buttonRect(i)) {
			return i
		}
	}

	return -1
}

func (g *radioGroup) mouseDown(p image.Point, _ bool) {
	g.pressing = g.at(p)
	g.inside = true
	g.invalidate()
}

func (g *radioGroup) mouseDrag(p image.Point) {
	if g.pressing < 0 {
		return
	}

	if in := p.In(g.buttonRect(g.pressing)); in != g.inside {
		g.inside = in
		g.invalidate()
	}
}

func (g *radioGroup) mouseUp(p image.Point) {
	k := g.pressing
	g.pressing = -1
	g.invalidate()

	if k >= 0 && p.In(g.buttonRect(k)) {
		g.choose(k)
	}
}

// choose checks item k; OnClick runs only when the item changes, as a
// click on the checked button does nothing.
func (g *radioGroup) choose(k int) {
	if k == g.index || !g.usable() {
		return
	}

	g.setIndex(k)

	if g.onClick != nil {
		g.onClick()
	}
}

func (g *radioGroup) key(k ebiten.Key, _ mods) bool {
	n := len(g.items)
	if n == 0 {
		return false
	}

	switch k {
	case ebiten.KeyUp, ebiten.KeyLeft:
		g.choose((max(g.index, 0) + n - 1) % n)
	case ebiten.KeyDown, ebiten.KeyRight:
		g.choose((g.index + 1) % n)
	default:
		return false
	}

	return true
}

func (g *radioGroup) char(rune) {}

// panel is a TPanel.
type panel struct{ control }

func (p *panel) setup(*layout.Component) {
	p.windowed = true
}

func (p *panel) paint(cv canvas) {
	r := p.local()
	cv.fill(r, p.color)

	if p.property("BevelOuter") != "bvNone" {
		cv.edge(r, colHighlight, colShadow)
	}

	if p.caption != "" {
		w := p.font.width(plain(p.caption))
		drawCaption(cv, p.font, (r.Dx()-w)/2, (r.Dy()-p.font.height())/2, p.caption, p.fontColor)
	}
}

// bevel is a TBevel (the box shape, lowered, is all the game uses).
type bevel struct{ control }

func (b *bevel) paint(cv canvas) {
	r := b.local()
	c1, c2 := colShadow, colHighlight

	if b.property("Style") == "bsRaised" {
		c1, c2 = c2, c1
	}

	// BevelRect(0, 0, W−1, H−1): two polylines, the second one ending
	// before its last point.
	w, h := r.Dx()-1, r.Dy()-1
	cv.vline(0, 0, h+1, c1)
	cv.hline(0, w, 0, c1)
	cv.vline(w, 0, h+1, c2)
	cv.hline(1, w+1, h, c2)
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
	p := delphi.NewExtended(float64(y)).Mul(oneHundredth)

	return int(delphi.NewExtended(float64(z)).Mul(p).Int().Float64())
}

// percent is PercentDone: Trunc(done × 100 / range).
func (g *gauge) percent() int {
	if g.max == g.min {
		return 0
	}

	return (g.progress - g.min) * 100 / (g.max - g.min)
}

func (g *gauge) paint(cv canvas) {
	r := g.local()
	cv.fill(r, colDkShadow) // the image starts black

	in := r.Inset(1) // bsSingle
	cv.fill(in, g.back)

	// PaintAsBar: W is one pixel wider than the inside, and the bar's right
	// edge is FillSize itself.
	w := in.Dx() + 1
	fill := min(solveForX(g.percent(), w), w)

	if fill > 0 {
		cv.fill(image.Rect(in.Min.X, in.Min.Y, fill, in.Max.Y).Intersect(in), g.fore)
	}

	// PaintAsText: white text XORed over the image, clipped to the inside.
	s := strconv.Itoa(g.percent()) + "%"
	x := (w - g.font.width(s)) / 2
	y := (in.Dy() + 1 - g.font.height()) / 2
	tmp := image.NewRGBA(image.Rectangle{Max: r.Size()})
	g.font.draw(newCanvas(tmp), x, y, s, colHighlight)

	for py := in.Min.Y; py < in.Max.Y; py++ {
		for px := in.Min.X; px < in.Max.X; px++ {
			if tmp.RGBAAt(px, py).A != 0 {
				cv.invert(px, py)
			}
		}
	}
}
