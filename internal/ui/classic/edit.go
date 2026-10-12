// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// textBox is the editing part of an edit, spin edit or combo box: one line
// of text with a caret and a selection.
type textBox struct {
	text     []rune
	caret    int
	anchor   int // the selection runs from anchor to caret
	scroll   int // pixels scrolled out on the left
	maxLen   int // 0: no limit
	readOnly bool
	keepSel  bool              // ES_NOHIDESEL: the selection shows without the focus
	valid    func(r rune) bool // characters that may be typed
	onChange func()            // after any change of the text
	blink    time.Time         // when the caret was last moved
	dragging bool
}

func (t *textBox) String() string { return string(t.text) }

// set replaces the text, putting the caret at its end.
func (t *textBox) set(s string) {
	t.text = []rune(s)
	t.caret, t.anchor = len(t.text), len(t.text)
	t.blink = time.Now()
}

func (t *textBox) selectAll() {
	t.anchor, t.caret = 0, len(t.text)
	t.blink = time.Now()
}

func (t *textBox) sel() (int, int) { return min(t.anchor, t.caret), max(t.anchor, t.caret) }

func (t *textBox) changed() {
	t.blink = time.Now()
	if t.onChange != nil {
		t.onChange()
	}
}

// replace replaces the selection with s, within the length limit.
func (t *textBox) replace(s string) {
	if t.readOnly {
		return
	}
	a, b := t.sel()
	in := []rune(s)
	if t.maxLen > 0 {
		room := t.maxLen - (len(t.text) - (b - a))
		if room < len(in) {
			in = in[:max(room, 0)]
		}
	}
	if len(in) == 0 && a == b {
		return
	}
	t.text = append(t.text[:a:a], append(in, t.text[b:]...)...)
	t.caret = a + len(in)
	t.anchor = t.caret
	t.changed()
}

// typed handles a typed character: one cp1252 can hold, that the box
// accepts.
func (t *textBox) typed(r rune) {
	if r < 0x20 || r == 0x7f || string(delphi.EncodeANSI(string(r))) == "?" && r != '?' {
		return
	}
	if t.valid != nil && !t.valid(r) {
		return
	}
	t.replace(string(r))
}

// key handles the editing keys; it reports whether k was one.
func (t *textBox) key(k ebiten.Key, m mods) bool {
	move := func(to int) {
		t.caret = max(min(to, len(t.text)), 0)
		if !m.shift {
			t.anchor = t.caret
		}
		t.blink = time.Now()
	}
	a, b := t.sel()
	switch k {
	case ebiten.KeyLeft:
		if !m.shift && a != b {
			move(a)
		} else if m.ctrl {
			move(t.wordLeft())
		} else {
			move(t.caret - 1)
		}
	case ebiten.KeyRight:
		if !m.shift && a != b {
			move(b)
		} else if m.ctrl {
			move(t.wordRight())
		} else {
			move(t.caret + 1)
		}
	case ebiten.KeyHome:
		move(0)
	case ebiten.KeyEnd:
		move(len(t.text))
	case ebiten.KeyBackspace:
		if t.readOnly {
			return true
		}
		if a == b {
			if a == 0 {
				return true
			}
			t.anchor = a - 1
		}
		t.replace("")
	case ebiten.KeyDelete:
		if t.readOnly {
			return true
		}
		if a == b {
			if b == len(t.text) {
				return true
			}
			t.caret = b + 1
		}
		t.replace("")
	default:
		return false
	}
	return true
}

func (t *textBox) wordLeft() int {
	i := t.caret
	for i > 0 && t.text[i-1] == ' ' {
		i--
	}
	for i > 0 && t.text[i-1] != ' ' {
		i--
	}
	return i
}

func (t *textBox) wordRight() int {
	i := t.caret
	for i < len(t.text) && t.text[i] != ' ' {
		i++
	}
	for i < len(t.text) && t.text[i] == ' ' {
		i++
	}
	return i
}

// at returns the caret position nearest to x (relative to the text's
// start, scroll included).
func (t *textBox) at(f face, x int) int {
	x += t.scroll
	w := 0
	for i := range t.text {
		cw := f.width(string(t.text[i]))
		if x < w+cw/2 {
			return i
		}
		w += cw
	}
	return len(t.text)
}

// fit scrolls the text so that the caret shows in a box w pixels wide.
func (t *textBox) fit(f face, w int) {
	cx := f.width(string(t.text[:t.caret]))
	if cx-t.scroll > w-1 {
		t.scroll = cx - w + 1 + w/4
	}
	if cx < t.scroll {
		t.scroll = max(cx-w/4, 0)
	}
	if f.width(string(t.text))-t.scroll < w-1 {
		t.scroll = max(f.width(string(t.text))-w+1, 0)
	}
}

// paint draws the text in r with its selection (when focused) and the
// blinking caret.
func (t *textBox) paint(cv canvas, r image.Rectangle, f face, fg color.RGBA, focused bool) {
	t.fit(f, r.Dx())
	in := cv.sub(r)
	x := -t.scroll
	s := string(t.text)
	a, b := t.sel()
	if !focused && !t.keepSel || a == b {
		f.draw(in, x, 0, s, fg)
	} else {
		pre, mid, post := string(t.text[:a]), string(t.text[a:b]), string(t.text[b:])
		x = f.draw(in, x, 0, pre, fg)
		w := f.width(mid)
		in.fill(image.Rect(x, 0, x+w, f.height()), colSelection)
		x = f.draw(in, x, 0, mid, colSelText)
		f.draw(in, x, 0, post, fg)
	}
	if focused && a == b && !t.readOnly && time.Since(t.blink)%(1060*time.Millisecond) < 530*time.Millisecond {
		cx := f.width(string(t.text[:t.caret])) - t.scroll
		for y := range f.height() {
			in.invert(cx, y)
		}
	}
}

func (t *textBox) press(f face, x int, double bool, shift bool) {
	if double {
		t.anchor = t.at(f, x)
		t.caret = t.anchor
		t.anchor = t.wordLeft()
		t.caret = t.wordRight()
		return
	}
	t.caret = t.at(f, x)
	if !shift {
		t.anchor = t.caret
	}
	t.dragging = true
	t.blink = time.Now()
}

func (t *textBox) drag(f face, x int) {
	if t.dragging {
		t.caret = t.at(f, x)
	}
}

// edit is a TEdit.
type edit struct {
	ctl
	tb       textBox
	inset    int    // the text's top-left corner, from the control's
	onChange func() // OnChange
}

func (e *edit) setup(comp *layout.Component) {
	e.windowed = true
	e.inset = 3 // the border, and the format rectangle's pixel inside it
	e.tabStop = e.prop("TabStop") != "False"
	e.color = vclColor(e.prop("Color"), colWindow)
	e.tb.set(comp.Caption)
	e.tb.maxLen = e.propInt("MaxLength", 0)
	e.tb.readOnly = e.prop("ReadOnly") == "True"
	e.tb.onChange = func() {
		e.invalidate()
		if e.onChange != nil {
			e.onChange()
		}
	}
}

func (e *edit) focusable() bool { return true }

func (e *edit) text() string { return e.tb.String() }

// setText sets the text as the program does: OnChange runs.
func (e *edit) setText(s string) {
	if e.tb.String() != s {
		e.tb.set(s)
		e.tb.changed()
	}
}

// gotFocus selects everything, as AutoSelect does, unless a click gave it
// the focus.
func (e *edit) gotFocus(byMouse bool) {
	if !byMouse {
		e.tb.selectAll()
	}
	e.invalidate()
}

func (e *edit) paint(cv canvas) {
	r := e.local()
	in := cv.sunken(r)
	fg := e.fontColor
	if !e.usable() {
		fg = colGrayText // on the control's colour: the VCL answers WM_CTLCOLORSTATIC with it
	}
	cv.fill(in, e.color)
	e.tb.paint(cv, image.Rect(e.inset, e.inset, r.Max.X-3, r.Max.Y-2), e.font, fg, e.focused() && e.win.active)
}

func (e *edit) mouseDown(p image.Point, double bool) {
	e.tb.press(e.font, p.X-e.inset, double, false)
	e.invalidate()
}

func (e *edit) mouseDrag(p image.Point) {
	e.tb.drag(e.font, p.X-e.inset)
	e.invalidate()
}

func (e *edit) mouseUp(p image.Point) { e.tb.dragging = false }

func (e *edit) key(k ebiten.Key, m mods) bool {
	if e.tb.key(k, m) {
		e.invalidate()
		return true
	}
	return false
}

func (e *edit) char(r rune) {
	e.tb.typed(r)
	e.invalidate()
}

func (e *edit) cursor(image.Point) ebiten.CursorShapeType { return ebiten.CursorShapeText }

// spinEdit is a TSpinEdit from the Delphi samples: an edit box with two
// arrow buttons, holding an integer.
type spinEdit struct {
	edit
	minValue, maxValue int
	increment          int
	pressed            int // 0 none, 1 up, 2 down
	inside             bool
	next               time.Time
	onChangeValue      func(v int)
}

func (s *spinEdit) setup(comp *layout.Component) {
	s.edit.setup(comp)
	s.inset = 2 // SetEditRect puts the text at the client area's corner
	s.minValue = s.propInt("MinValue", 0)
	s.maxValue = s.propInt("MaxValue", 0)
	s.increment = s.propInt("Increment", 1)
	s.tb.set(strconv.Itoa(s.propInt("Value", 0)))
	// IsValidChar: digits, signs and the decimal separator.
	s.tb.valid = func(r rune) bool { return r >= '0' && r <= '9' || r == '+' || r == '-' || r == '.' }
	s.tb.onChange = func() {
		s.invalidate()
		if s.onChangeValue != nil {
			s.onChangeValue(s.value())
		}
	}
}

// value is TSpinEdit.Value: the text as an integer, or MinValue when it
// isn't one.
func (s *spinEdit) value() int {
	if v, ok := delphi.StrToInt(s.tb.String()); ok {
		return int(v)
	}
	return s.minValue
}

// check clamps v to the range, unless MaxValue = MinValue (CheckValue).
func (s *spinEdit) check(v int) int {
	if s.maxValue != s.minValue {
		v = max(min(v, s.maxValue), s.minValue)
	}
	return v
}

// setValue sets Value as the program does: OnChange runs if the text
// changes.
func (s *spinEdit) setValue(v int) { s.setText(strconv.Itoa(s.check(v))) }

// lostFocus clamps the value, as CMExit does.
func (s *spinEdit) lostFocus() {
	if v := s.value(); s.check(v) != v {
		s.setValue(v)
	}
}

// buttons returns the up and down buttons: TSpinButton sits inside the
// edit's border, 15 pixels wide and 5 short of the edit's height, and its
// down button covers the up button's last row (AdjustSize).
func (s *spinEdit) buttons() (up, down image.Rectangle) {
	r := s.local().Inset(2)
	x := r.Min.X + s.r.Dx() - 15 - 5
	h := s.r.Dy() - 5
	up = image.Rect(x, r.Min.Y, x+15, r.Min.Y+h/2)
	down = image.Rect(x, up.Max.Y-1, x+15, r.Min.Y+h)
	return
}

// The SPINUP and SPINDOWN glyphs.
const (
	glyphSpinUp = `
.........
....#....
...###...
..#####..
.#######.
.........
`
	glyphSpinDown = `
.........
.#######.
..#####..
...###...
....#....
.........
`
)

func (s *spinEdit) paint(cv canvas) {
	r := s.local()
	in := cv.sunken(r)
	fg := s.fontColor
	if !s.usable() {
		fg = colGrayText // on the control's colour: the VCL answers WM_CTLCOLORSTATIC with it
	}
	cv.fill(in, s.color)
	up, down := s.buttons()
	s.tb.paint(cv, image.Rect(s.inset, s.inset, up.Min.X-1, r.Max.Y-2), s.font, fg, s.focused() && s.win.active)
	// Two speed buttons, the down one painted last, each drawn by
	// DrawFrameControl with its glyph centred inside the frame.
	for i, b := range []image.Rectangle{up, down} {
		pushed := s.pressed == i+1 && s.inside
		cv.fill(b, colFace)
		off := 0
		if pushed {
			cv.sunkenSoft(b)
			off = 1
		} else {
			cv.raisedSoft(b)
		}
		glyph := glyphSpinUp
		if i == 1 {
			glyph = glyphSpinDown
		}
		in := b.Inset(2)
		x := in.Min.X + (in.Dx()-9+1)/2 + off
		y := in.Min.Y + (in.Dy()-6+1)/2 + off
		if s.usable() {
			cv.pattern(x, y, glyph, map[byte]color.RGBA{'#': colText})
		} else {
			cv.pattern(x+1, y+1, glyph, map[byte]color.RGBA{'#': colHighlight})
			cv.pattern(x, y, glyph, map[byte]color.RGBA{'#': colShadow})
		}
	}
}

func (s *spinEdit) mouseDown(p image.Point, double bool) {
	up, down := s.buttons()
	switch {
	case p.In(down): // on top of the up button
		s.pressed = 2
	case p.In(up):
		s.pressed = 1
	default:
		s.edit.mouseDown(p, double)
		return
	}
	s.inside = true
	s.step()
	s.next = s.win.desk.now.Add(400 * time.Millisecond)
	s.invalidate()
}

// step is one click of the pressed button.
func (s *spinEdit) step() {
	if s.pressed == 1 {
		s.spin(1)
	} else {
		s.spin(-1)
	}
}

// spin adds Increment, or takes it away, as UpClick and DownClick do.
func (s *spinEdit) spin(dir int) {
	if s.tb.readOnly {
		return
	}
	s.setValue(s.value() + dir*s.increment)
	s.tb.selectAll()
}

func (s *spinEdit) mouseDrag(p image.Point) {
	if s.pressed == 0 {
		s.edit.mouseDrag(p)
		return
	}
	up, down := s.buttons()
	b := up
	if s.pressed == 2 {
		b = down
	}
	if in := p.In(b); in != s.inside {
		s.inside = in
		s.invalidate()
	}
}

func (s *spinEdit) mouseUp(p image.Point) {
	if s.pressed == 0 {
		s.edit.mouseUp(p)
		return
	}
	s.pressed = 0
	s.invalidate()
}

// tick repeats a held button, as TTimerSpeedButton does.
func (s *spinEdit) tick(now time.Time) {
	if s.pressed != 0 && s.inside && !now.Before(s.next) {
		s.step()
		s.next = now.Add(100 * time.Millisecond)
	}
}

func (s *spinEdit) key(k ebiten.Key, m mods) bool {
	switch k {
	case ebiten.KeyUp:
		s.spin(1)
	case ebiten.KeyDown:
		s.spin(-1)
	default:
		return s.edit.key(k, m)
	}
	return true
}

func (s *spinEdit) cursor(p image.Point) ebiten.CursorShapeType {
	if up, down := s.buttons(); p.In(up) || p.In(down) {
		return ebiten.CursorShapeDefault
	}
	return ebiten.CursorShapeText
}

// memo is a read-only TMemo with a vertical scroll bar: Battle Info.
type memo struct {
	ctl
	lines  []string // as the program set them
	wraps  []string // wrapped to the width
	scroll scrollBar
}

func (m *memo) setup(comp *layout.Component) {
	m.windowed = true
	m.tabStop = m.prop("TabStop") != "False"
	m.color = vclColor(m.prop("Color"), colWindow)
	m.lines = append([]string(nil), comp.Items...)
	m.scroll = scrollBar{vertical: true}
	m.scroll.onScroll = m.invalidate
}

func (m *memo) focusable() bool { return true }

// text returns the client rectangle for the text.
func (m *memo) textRect() image.Rectangle {
	r := m.local().Inset(2)
	r.Max.X -= scrollSize
	return r
}

// setLines replaces the text; lines added at the end scroll into view.
func (m *memo) setLines(lines []string) {
	if strings.Join(lines, "\n") == strings.Join(m.lines, "\n") {
		return
	}
	grew := len(lines) > len(m.lines)
	m.lines = append(m.lines[:0:0], lines...)
	m.layout()
	if grew {
		m.scroll.set(m.scroll.limit())
	}
	m.invalidate()
}

// layout wraps the lines, as WordWrap does, and sizes the scroll bar.
func (m *memo) layout() {
	tr := m.textRect()
	m.wraps = m.wraps[:0]
	for _, l := range m.lines {
		m.wraps = append(m.wraps, wrap(m.font, l, tr.Dx()-2)...)
	}
	r := m.local().Inset(2)
	m.scroll.r = image.Rect(r.Max.X-scrollSize, r.Min.Y, r.Max.X, r.Max.Y)
	m.scroll.min, m.scroll.max = 0, len(m.wraps)-1
	m.scroll.page = (tr.Dy() - 2) / m.font.height()
	m.scroll.set(m.scroll.pos)
}

func (m *memo) paint(cv canvas) {
	if len(m.wraps) == 0 && len(m.lines) > 0 || m.scroll.r.Empty() {
		m.layout()
	}
	r := m.local()
	cv.sunken(r)
	tr := m.textRect()
	cv.fill(tr, m.color)
	// The format rectangle is a pixel inside the client area.
	in := cv.sub(tr)
	for i := m.scroll.pos; i < len(m.wraps); i++ {
		y := 1 + (i-m.scroll.pos)*m.font.height()
		if y >= tr.Dy() {
			break
		}
		m.font.draw(in, 1, y, m.wraps[i], m.fontColor)
	}
	m.scroll.paint(cv)
}

func (m *memo) mouseDown(p image.Point, double bool) {
	if m.scroll.press(p) {
		m.invalidate()
	}
}

func (m *memo) mouseDrag(p image.Point) {
	if m.scroll.pressed != partNone {
		m.scroll.drag(p)
		m.invalidate()
	}
}

func (m *memo) mouseUp(image.Point) {
	if m.scroll.pressed != partNone {
		m.scroll.release()
		m.invalidate()
	}
}

func (m *memo) tick(now time.Time) {
	if m.scroll.tick(now) {
		m.invalidate()
	}
}

func (m *memo) wheel(dy int) {
	m.scroll.set(m.scroll.pos - 3*dy)
	m.invalidate()
}

func (m *memo) key(k ebiten.Key, mm mods) bool {
	switch k {
	case ebiten.KeyUp:
		m.scroll.set(m.scroll.pos - 1)
	case ebiten.KeyDown:
		m.scroll.set(m.scroll.pos + 1)
	case ebiten.KeyPageUp:
		m.scroll.set(m.scroll.pos - m.scroll.page)
	case ebiten.KeyPageDown:
		m.scroll.set(m.scroll.pos + m.scroll.page)
	case ebiten.KeyHome:
		if mm.ctrl {
			m.scroll.set(0)
		}
	case ebiten.KeyEnd:
		if mm.ctrl {
			m.scroll.set(m.scroll.limit())
		}
	default:
		return false
	}
	m.invalidate()
	return true
}

func (m *memo) char(rune) {}

// comboBox is a TComboBox: in the csDropDown style, an edit box with a
// drop-down list and AutoComplete; in the csDropDownList style, the
// selected item over the list.
type comboBox struct {
	ctl
	tb         textBox
	listOnly   bool // csDropDownList
	items      []string
	index      int // ItemIndex
	itemHeight int
	dropped    *dropList
	pressing   bool // the arrow button
	onSelect   func(i int)

	// An owner-drawn list's icons (CBS_OWNERDRAWFIXED): each item's image
	// and indent, in pixels, in the list.
	images  *imageList
	icons   []int
	indents []int
}

func (c *comboBox) setup(comp *layout.Component) {
	c.windowed = true
	c.tabStop = c.prop("TabStop") != "False"
	c.color = vclColor(c.prop("Color"), colWindow)
	c.listOnly = c.prop("Style") == "csDropDownList"
	c.items = append([]string(nil), comp.Items...)
	c.index = -1
	c.itemHeight = c.propInt("ItemHeight", 13)
	c.tb.set(comp.Caption)
	c.tb.keepSel = true // the combo box's edit has ES_NOHIDESEL
	c.tb.onChange = c.invalidate
}

func (c *comboBox) focusable() bool { return true }

// setIndex selects item i (-1 for none) as the program does, which
// doesn't run OnChange; the text follows the selection (none empties it),
// all of it selected if the box has the focus, else with the caret at its
// start.
func (c *comboBox) setIndex(i int) {
	if i == c.index {
		return
	}
	c.index = i
	text := ""
	if i >= 0 && i < len(c.items) {
		text = c.items[i]
	}
	c.tb.set(text)
	if c.focused() && c.win.active {
		c.tb.selectAll()
	} else {
		c.tb.caret, c.tb.anchor = 0, 0
	}
	c.invalidate()
}

// reselect selects the item again, as a FormShow that refills the list
// does (ItemIndex := -1, the items again, ItemIndex := k): the form isn't
// active yet, so the text isn't selected.
func (c *comboBox) reselect() {
	k := c.index
	c.setIndex(-1)
	c.setIndex(k)
}

// pick selects item i as the player does: the text becomes the item,
// all selected, and OnChange runs.
func (c *comboBox) pick(i int) {
	if i < 0 || i >= len(c.items) {
		return
	}
	changed := i != c.index
	c.index = i
	c.tb.set(c.items[i])
	c.tb.selectAll()
	c.invalidate()
	if changed && c.onSelect != nil {
		c.onSelect(i)
	}
}

// button is the drop-down button, inside the border.
func (c *comboBox) button() image.Rectangle {
	in := c.local().Inset(2)
	return image.Rect(in.Max.X-scrollSize, in.Min.Y, in.Max.X, in.Max.Y)
}

func (c *comboBox) gotFocus(byMouse bool) {
	c.tb.selectAll()
	c.invalidate()
}

func (c *comboBox) lostFocus() {
	c.closeUp()
}

func (c *comboBox) paint(cv canvas) {
	r := c.local()
	in := cv.sunken(r)
	fg := c.fontColor
	if !c.usable() {
		fg = colGrayText // on the control's colour: the VCL answers WM_CTLCOLORSTATIC with it
	}
	cv.fill(in, c.color)
	b := c.button()
	if c.listOnly {
		// The selected item, highlighted while the box has the focus.
		tr := image.Rect(in.Min.X+1, in.Min.Y+1, b.Min.X-1, in.Max.Y-1)
		focus := c.focused() && c.win.active && c.dropped == nil
		item := c.index >= 0 && c.index < len(c.items)
		tx := 1
		if c.images != nil && item {
			// Owner-drawn: the item's icon, then its text, highlighted on
			// its own; the box doesn't indent it.
			c.images.drawFrame(cv, tr.Min.X+1, tr.Min.Y+(tr.Dy()-c.images.size)/2, c.icons[c.index], false)
			x := tr.Min.X + c.images.size + 4
			tr = image.Rect(x, tr.Min.Y, min(x+c.font.width(c.items[c.index])+4, tr.Max.X), tr.Max.Y)
			tx = 2
		}
		if focus {
			cv.fill(tr, colSelection)
			fg = colSelText
		}
		if item {
			c.font.draw(cv.sub(tr), tx, (tr.Dy()-c.font.height())/2, c.items[c.index], fg)
		}
		if focus {
			cv.focusRect(tr)
		}
	} else {
		c.tb.paint(cv, image.Rect(3, 3, b.Min.X-1, r.Max.Y-3), c.font, fg, c.focused() && c.win.active)
	}
	paintArrowButton(cv, b, arrowDown, c.usable(), c.pressing && c.dropped != nil, 5)
}

func (c *comboBox) dropDown() {
	if c.dropped != nil || len(c.items) == 0 {
		return
	}
	c.dropped = newDropList(c)
	c.win.desk.openPopup(c.dropped)
	c.invalidate()
}

func (c *comboBox) closeUp() {
	if c.dropped == nil {
		return
	}
	d := c.dropped
	c.dropped = nil
	c.win.desk.closePopup(d)
	c.invalidate()
}

func (c *comboBox) mouseDown(p image.Point, double bool) {
	if p.In(c.button()) || c.listOnly {
		c.pressing = true
		if c.dropped != nil {
			c.closeUp()
		} else {
			c.dropDown()
		}
		return
	}
	c.tb.press(c.font, p.X-3, double, false)
	c.invalidate()
}

func (c *comboBox) mouseDrag(p image.Point) {
	if !c.pressing && !c.listOnly {
		c.tb.drag(c.font, p.X-3)
		c.invalidate()
	}
}

func (c *comboBox) mouseUp(p image.Point) {
	c.pressing = false
	c.tb.dragging = false
	c.invalidate()
}

func (c *comboBox) key(k ebiten.Key, m mods) bool {
	if c.dropped != nil && (k == ebiten.KeyEscape || k == ebiten.KeyF4 || m.alt && (k == ebiten.KeyUp || k == ebiten.KeyDown)) {
		c.closeUp()
		return true
	}
	if c.dropped != nil && c.dropped.key(k) {
		if k == ebiten.KeyEnter || k == ebiten.KeyNumpadEnter {
			c.closeUp()
		}
		c.invalidate()
		return true
	}
	switch {
	case k == ebiten.KeyF4 || m.alt && k == ebiten.KeyDown:
		c.dropDown()
	case k == ebiten.KeyUp && !m.alt:
		c.pick(max(c.index-1, 0))
	case k == ebiten.KeyDown && !m.alt:
		c.pick(min(c.index+1, len(c.items)-1))
	case c.listOnly:
		switch k {
		case ebiten.KeyHome:
			c.pick(0)
		case ebiten.KeyEnd:
			c.pick(len(c.items) - 1)
		default:
			return false
		}
	case k == ebiten.KeyBackspace:
		// AutoComplete deletes the completed part first.
		if a, b := c.tb.sel(); a != b {
			c.tb.replace("")
		} else if a > 0 {
			c.tb.anchor = a - 1
			c.tb.replace("")
		}
		c.typedText()
	default:
		if c.tb.key(k, m) {
			c.invalidate()
			return true
		}
		return false
	}
	c.invalidate()
	return true
}

// char types a character with AutoComplete: the text so far plus the
// character is looked up as a prefix of the items (CB_FINDSTRING, ignoring
// case); a match completes the text and selects the completion.
func (c *comboBox) char(r rune) {
	if r < 0x20 {
		return
	}
	if c.listOnly {
		// The next item starting with the letter.
		n := len(c.items)
		for k := 1; k <= n; k++ {
			i := (max(c.index, -1) + k) % n
			if strings.HasPrefix(strings.ToLower(c.items[i]), strings.ToLower(string(r))) {
				c.pick(i)
				if c.dropped != nil {
					c.dropped.hot = i
					c.dropped.show(i)
				}
				return
			}
		}
		return
	}
	a, _ := c.tb.sel()
	prefix := string(c.tb.text[:a]) + string(r)
	for i, item := range c.items {
		if strings.HasPrefix(strings.ToLower(item), strings.ToLower(prefix)) {
			c.tb.set(prefix + string([]rune(item)[len([]rune(prefix)):]))
			c.tb.anchor, c.tb.caret = len([]rune(prefix)), len(c.tb.text)
			if i != c.index {
				c.index = i
				if c.onSelect != nil {
					c.onSelect(i)
				}
			}
			if c.dropped != nil {
				c.dropped.hot = i
				c.dropped.show(i)
			}
			c.invalidate()
			return
		}
	}
	c.tb.typed(r)
	c.typedText()
}

// typedText follows a text the items don't complete: the selection is
// lost.
func (c *comboBox) typedText() {
	i := -1
	for k, item := range c.items {
		if strings.EqualFold(item, c.tb.String()) {
			i = k
		}
	}
	if i != c.index {
		c.index = i
		if c.onSelect != nil {
			c.onSelect(i)
		}
	}
	c.invalidate()
}

func (c *comboBox) wheel(dy int) {
	if c.dropped != nil {
		c.dropped.wheel(dy)
		return
	}
	if dy > 0 {
		c.pick(max(c.index-1, 0))
	} else if dy < 0 {
		c.pick(min(c.index+1, len(c.items)-1))
	}
}

func (c *comboBox) cursor(p image.Point) ebiten.CursorShapeType {
	if p.In(c.button()) || c.listOnly {
		return ebiten.CursorShapeDefault
	}
	return ebiten.CursorShapeText
}
