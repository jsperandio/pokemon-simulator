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

func (tbx *textBox) String() string {
	return string(tbx.text)
}

// set replaces the text, putting the caret at its end.
func (tbx *textBox) set(s string) {
	tbx.text = []rune(s)
	tbx.caret, tbx.anchor = len(tbx.text), len(tbx.text)
	tbx.blink = time.Now()
}

func (tbx *textBox) selectAll() {
	tbx.anchor, tbx.caret = 0, len(tbx.text)
	tbx.blink = time.Now()
}

func (tbx *textBox) sel() (from, to int) {
	return min(tbx.anchor, tbx.caret), max(tbx.anchor, tbx.caret)
}

func (tbx *textBox) changed() {
	tbx.blink = time.Now()
	if tbx.onChange != nil {
		tbx.onChange()
	}
}

// replace replaces the selection with s, within the length limit.
func (tbx *textBox) replace(s string) {
	if tbx.readOnly {
		return
	}

	a, b := tbx.sel()
	in := []rune(s)

	if tbx.maxLen > 0 {
		room := tbx.maxLen - (len(tbx.text) - (b - a))
		if room < len(in) {
			in = in[:max(room, 0)]
		}
	}

	if len(in) == 0 && a == b {
		return
	}

	tbx.text = append(tbx.text[:a:a], append(in, tbx.text[b:]...)...)
	tbx.caret = a + len(in)
	tbx.anchor = tbx.caret
	tbx.changed()
}

// typed handles a typed character: one cp1252 can hold, that the box
// accepts.
func (tbx *textBox) typed(r rune) {
	control := r < 0x20 || r == 0x7f
	outsideANSI := string(delphi.EncodeANSI(string(r))) == "?" && r != '?'

	if control || outsideANSI {
		return
	}

	if tbx.valid != nil && !tbx.valid(r) {
		return
	}

	tbx.replace(string(r))
}

// key handles the editing keys; it reports whether k was one.
func (tbx *textBox) key(k ebiten.Key, m mods) bool {
	move := func(to int) {
		tbx.caret = max(min(to, len(tbx.text)), 0)
		if !m.shift {
			tbx.anchor = tbx.caret
		}

		tbx.blink = time.Now()
	}
	a, b := tbx.sel()

	switch k {
	case ebiten.KeyLeft:
		switch {
		case !m.shift && a != b:
			move(a)
		case m.ctrl:
			move(tbx.wordLeft())
		default:
			move(tbx.caret - 1)
		}
	case ebiten.KeyRight:
		switch {
		case !m.shift && a != b:
			move(b)
		case m.ctrl:
			move(tbx.wordRight())
		default:
			move(tbx.caret + 1)
		}
	case ebiten.KeyHome:
		move(0)
	case ebiten.KeyEnd:
		move(len(tbx.text))
	case ebiten.KeyBackspace:
		tbx.backspace()
	case ebiten.KeyDelete:
		tbx.del()
	default:
		return false
	}

	return true
}

// backspace deletes the selection, or the character before the caret.
func (tbx *textBox) backspace() {
	if tbx.readOnly {
		return
	}

	if a, b := tbx.sel(); a == b {
		if a == 0 {
			return
		}

		tbx.anchor = a - 1
	}

	tbx.replace("")
}

// del deletes the selection, or the character after the caret.
func (tbx *textBox) del() {
	if tbx.readOnly {
		return
	}

	if a, b := tbx.sel(); a == b {
		if b == len(tbx.text) {
			return
		}

		tbx.caret = b + 1
	}

	tbx.replace("")
}

func (tbx *textBox) wordLeft() int {
	i := tbx.caret
	for i > 0 && tbx.text[i-1] == ' ' {
		i--
	}

	for i > 0 && tbx.text[i-1] != ' ' {
		i--
	}

	return i
}

func (tbx *textBox) wordRight() int {
	i := tbx.caret
	for i < len(tbx.text) && tbx.text[i] != ' ' {
		i++
	}

	for i < len(tbx.text) && tbx.text[i] == ' ' {
		i++
	}

	return i
}

// at returns the caret position nearest to x (relative to the text's
// start, scroll included).
func (tbx *textBox) at(f face, x int) int {
	x += tbx.scroll
	w := 0

	for i := range tbx.text {
		cw := f.width(string(tbx.text[i]))
		if x < w+cw/2 {
			return i
		}

		w += cw
	}

	return len(tbx.text)
}

// fit scrolls the text so that the caret shows in a box w pixels wide.
func (tbx *textBox) fit(f face, w int) {
	cx := f.width(string(tbx.text[:tbx.caret]))
	if cx-tbx.scroll > w-1 {
		tbx.scroll = cx - w + 1 + w/4
	}

	if cx < tbx.scroll {
		tbx.scroll = max(cx-w/4, 0)
	}

	if f.width(string(tbx.text))-tbx.scroll < w-1 {
		tbx.scroll = max(f.width(string(tbx.text))-w+1, 0)
	}
}

// caretBlink is how long the caret stays on, and then off, as Windows'
// default has it.
const caretBlink time.Duration = 530 * time.Millisecond

// blinkOn reports whether a caret that moved since ago shows: it is on for
// caretBlink and off for as long.
func blinkOn(since time.Duration) bool {
	return since%(2*caretBlink) < caretBlink
}

// paint draws the text in r with its selection (when focused) and the
// blinking caret.
func (tbx *textBox) paint(cv canvas, r image.Rectangle, f face, fg color.RGBA, focused bool) {
	tbx.fit(f, r.Dx())

	in := cv.sub(r)
	x := -tbx.scroll
	s := string(tbx.text)
	a, b := tbx.sel()

	// The selection shows with the focus, or always with ES_NOHIDESEL.
	showSel := (focused || tbx.keepSel) && a != b
	if showSel {
		pre, mid, post := string(tbx.text[:a]), string(tbx.text[a:b]), string(tbx.text[b:])
		x = f.draw(in, x, 0, pre, fg)
		w := f.width(mid)
		in.fill(image.Rect(x, 0, x+w, f.height()), colSelection)

		x = f.draw(in, x, 0, mid, colSelText)
		f.draw(in, x, 0, post, fg)
	} else {
		f.draw(in, x, 0, s, fg)
	}

	hasCaret := focused && a == b && !tbx.readOnly
	if hasCaret && blinkOn(time.Since(tbx.blink)) {
		cx := f.width(string(tbx.text[:tbx.caret])) - tbx.scroll
		for y := range f.height() {
			in.invert(cx, y)
		}
	}
}

func (tbx *textBox) press(f face, x int, double, shift bool) {
	if double {
		tbx.anchor = tbx.at(f, x)
		tbx.caret = tbx.anchor
		tbx.anchor = tbx.wordLeft()
		tbx.caret = tbx.wordRight()

		return
	}

	tbx.caret = tbx.at(f, x)
	if !shift {
		tbx.anchor = tbx.caret
	}

	tbx.dragging = true
	tbx.blink = time.Now()
}

func (tbx *textBox) drag(f face, x int) {
	if tbx.dragging {
		tbx.caret = tbx.at(f, x)
	}
}

// edit is a TEdit.
type edit struct {
	control
	tb       textBox
	inset    int    // the text's top-left corner, from the control's
	onChange func() // OnChange
}

func (e *edit) setup(comp *layout.Component) {
	e.windowed = true
	e.inset = 3 // the border, and the format rectangle's pixel inside it
	e.tabStop = e.property("TabStop") != "False"
	e.color = vclColor(e.property("Color"), colWindow)
	e.tb.set(comp.Caption)

	e.tb.maxLen = e.propertyInt("MaxLength", 0)
	e.tb.readOnly = e.property("ReadOnly") == "True"
	e.tb.onChange = func() {
		e.invalidate()

		if e.onChange != nil {
			e.onChange()
		}
	}
}

func (e *edit) focusable() bool {
	return true
}

func (e *edit) text() string {
	return e.tb.String()
}

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
	e.tb.paint(cv, image.Rect(e.inset, e.inset, r.Max.X-3, r.Max.Y-2), e.font, fg, e.focused() && e.wdw.active)
}

func (e *edit) mouseDown(p image.Point, double bool) {
	e.tb.press(e.font, p.X-e.inset, double, false)
	e.invalidate()
}

func (e *edit) mouseDrag(p image.Point) {
	e.tb.drag(e.font, p.X-e.inset)
	e.invalidate()
}

func (e *edit) mouseUp(image.Point) {
	e.tb.dragging = false
}

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

func (e *edit) cursor(image.Point) ebiten.CursorShapeType {
	return ebiten.CursorShapeText
}

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

func (se *spinEdit) setup(comp *layout.Component) {
	se.edit.setup(comp)

	se.inset = 2 // SetEditRect puts the text at the client area's corner
	se.minValue = se.propertyInt("MinValue", 0)
	se.maxValue = se.propertyInt("MaxValue", 0)
	se.increment = se.propertyInt("Increment", 1)
	se.tb.set(strconv.Itoa(se.propertyInt("Value", 0)))

	// IsValidChar: digits, signs and the decimal separator.
	se.tb.valid = func(r rune) bool {
		return isDigit(r) || strings.ContainsRune("+-.", r)
	}
	se.tb.onChange = func() {
		se.invalidate()

		if se.onChangeValue != nil {
			se.onChangeValue(se.value())
		}
	}
}

// value is TSpinEdit.Value: the text as an integer, or MinValue when it
// isn't one.
func (se *spinEdit) value() int {
	if v, ok := delphi.StrToInt(se.tb.String()); ok {
		return int(v)
	}

	return se.minValue
}

// check clamps v to the range, unless MaxValue = MinValue (CheckValue).
func (se *spinEdit) check(v int) int {
	if se.maxValue != se.minValue {
		v = max(min(v, se.maxValue), se.minValue)
	}

	return v
}

// setValue sets Value as the program does: OnChange runs if the text
// changes.
func (se *spinEdit) setValue(v int) {
	se.setText(strconv.Itoa(se.check(v)))
}

// lostFocus clamps the value, as CMExit does.
func (se *spinEdit) lostFocus() {
	if v := se.value(); se.check(v) != v {
		se.setValue(v)
	}
}

// buttons returns the up and down buttons: TSpinButton sits inside the
// edit's border, 15 pixels wide and 5 short of the edit's height, and its
// down button covers the up button's last row (AdjustSize).
func (se *spinEdit) buttons() (up, down image.Rectangle) {
	r := se.local().Inset(2)
	x := r.Min.X + se.rect.Dx() - 15 - 5
	h := se.rect.Dy() - 5
	up = image.Rect(x, r.Min.Y, x+15, r.Min.Y+h/2)
	down = image.Rect(x, up.Max.Y-1, x+15, r.Min.Y+h)

	return
}

// The SPINUP and SPINDOWN glyphs.
const (
	glyphSpinUp string = `
.........
....#....
...###...
..#####..
.#######.
.........
`
	glyphSpinDown string = `
.........
.#######.
..#####..
...###...
....#....
.........
`
)

func (se *spinEdit) paint(cv canvas) {
	r := se.local()
	in := cv.sunken(r)
	fg := se.fontColor

	if !se.usable() {
		fg = colGrayText // on the control's colour: the VCL answers WM_CTLCOLORSTATIC with it
	}

	cv.fill(in, se.color)

	up, down := se.buttons()
	se.tb.paint(cv, image.Rect(se.inset, se.inset, up.Min.X-1, r.Max.Y-2), se.font, fg, se.focused() && se.wdw.active)

	// Two speed buttons, the down one painted last, each drawn by
	// DrawFrameControl with its glyph centred inside the frame.
	for i, b := range []image.Rectangle{up, down} {
		off := cv.softButton(b, se.pressed == i+1 && se.inside)

		glyph := glyphSpinUp
		if i == 1 {
			glyph = glyphSpinDown
		}

		in := b.Inset(2)
		x := in.Min.X + (in.Dx()-9+1)/2 + off
		y := in.Min.Y + (in.Dy()-6+1)/2 + off

		if !se.usable() {
			cv.embossed(x, y, glyph)

			continue
		}

		cv.pattern(x, y, glyph, map[byte]color.RGBA{'#': colText})
	}
}

func (se *spinEdit) mouseDown(p image.Point, double bool) {
	up, down := se.buttons()
	switch {
	case p.In(down): // on top of the up button
		se.pressed = 2
	case p.In(up):
		se.pressed = 1
	default:
		se.edit.mouseDown(p, double)

		return
	}

	se.inside = true
	se.step()

	now := se.wdw.desk.now
	se.next = now.Add(400 * time.Millisecond)
	se.invalidate()
}

// step is one click of the pressed button.
func (se *spinEdit) step() {
	dir := -1
	if se.pressed == 1 {
		dir = 1
	}

	se.spin(dir)
}

// spin adds Increment, or takes it away, as UpClick and DownClick do.
func (se *spinEdit) spin(dir int) {
	if se.tb.readOnly {
		return
	}

	se.setValue(se.value() + dir*se.increment)
	se.tb.selectAll()
}

func (se *spinEdit) mouseDrag(p image.Point) {
	if se.pressed == 0 {
		se.edit.mouseDrag(p)

		return
	}

	up, down := se.buttons()
	b := up

	if se.pressed == 2 {
		b = down
	}

	if in := p.In(b); in != se.inside {
		se.inside = in
		se.invalidate()
	}
}

func (se *spinEdit) mouseUp(p image.Point) {
	if se.pressed == 0 {
		se.edit.mouseUp(p)

		return
	}

	se.pressed = 0
	se.invalidate()
}

// tick repeats a held button, as TTimerSpeedButton does.
func (se *spinEdit) tick(now time.Time) {
	if se.pressed != 0 && se.inside && !now.Before(se.next) {
		se.step()

		se.next = now.Add(100 * time.Millisecond)
	}
}

func (se *spinEdit) key(k ebiten.Key, m mods) bool {
	switch k {
	case ebiten.KeyUp:
		se.spin(1)
	case ebiten.KeyDown:
		se.spin(-1)
	default:
		return se.edit.key(k, m)
	}

	return true
}

func (se *spinEdit) cursor(p image.Point) ebiten.CursorShapeType {
	if up, down := se.buttons(); p.In(up) || p.In(down) {
		return ebiten.CursorShapeDefault
	}

	return ebiten.CursorShapeText
}

// memo is a read-only TMemo with a vertical scroll bar: Battle Info.
type memo struct {
	control
	lines  []string // as the program set them
	wraps  []string // wrapped to the width
	scroll scrollBar
}

func (m *memo) setup(comp *layout.Component) {
	m.windowed = true
	m.tabStop = m.property("TabStop") != "False"
	m.color = vclColor(m.property("Color"), colWindow)

	m.lines = append([]string(nil), comp.Items...)
	m.scroll = scrollBar{vertical: true}
	m.scroll.onScroll = m.invalidate
}

func (m *memo) focusable() bool {
	return true
}

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
	m.scroll.rect = image.Rect(r.Max.X-scrollSize, r.Min.Y, r.Max.X, r.Max.Y)
	m.scroll.min, m.scroll.max = 0, len(m.wraps)-1
	m.scroll.page = (tr.Dy() - 2) / m.font.height()
	m.scroll.set(m.scroll.pos)
}

func (m *memo) paint(cv canvas) {
	needsLayout := (len(m.wraps) == 0 && len(m.lines) > 0) || m.scroll.rect.Empty()
	if needsLayout {
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

func (m *memo) mouseDown(p image.Point, _ bool) {
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
	bar := &m.scroll

	switch k {
	case ebiten.KeyUp:
		bar.set(bar.pos - 1)
	case ebiten.KeyDown:
		bar.set(bar.pos + 1)
	case ebiten.KeyPageUp:
		bar.set(bar.pos - bar.page)
	case ebiten.KeyPageDown:
		bar.set(bar.pos + bar.page)
	case ebiten.KeyHome:
		if mm.ctrl {
			bar.set(0)
		}
	case ebiten.KeyEnd:
		if mm.ctrl {
			bar.set(bar.limit())
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
	control
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

func (cb *comboBox) setup(comp *layout.Component) {
	cb.windowed = true
	cb.tabStop = cb.property("TabStop") != "False"
	cb.color = vclColor(cb.property("Color"), colWindow)
	cb.listOnly = cb.property("Style") == "csDropDownList"

	cb.items = append([]string(nil), comp.Items...)
	cb.index = -1
	cb.itemHeight = cb.propertyInt("ItemHeight", 13)
	cb.tb.set(comp.Caption)

	cb.tb.keepSel = true // the combo box's edit has ES_NOHIDESEL
	cb.tb.onChange = cb.invalidate
}

func (cb *comboBox) focusable() bool {
	return true
}

// setIndex selects item i (-1 for none) as the program does, which
// doesn't run OnChange; the text follows the selection (none empties it),
// all of it selected if the box has the focus, else with the caret at its
// start.
func (cb *comboBox) setIndex(i int) {
	if i == cb.index {
		return
	}

	cb.index = i
	text := ""

	if i >= 0 && i < len(cb.items) {
		text = cb.items[i]
	}

	cb.tb.set(text)

	if cb.focused() && cb.wdw.active {
		cb.tb.selectAll()
	} else {
		cb.tb.caret, cb.tb.anchor = 0, 0
	}

	cb.invalidate()
}

// reselect selects the item again, as a FormShow that refills the list
// does (ItemIndex := -1, the items again, ItemIndex := k): the form isn't
// active yet, so the text isn't selected.
func (cb *comboBox) reselect() {
	k := cb.index
	cb.setIndex(-1)
	cb.setIndex(k)
}

// pick selects item i as the player does: the text becomes the item,
// all selected, and OnChange runs.
func (cb *comboBox) pick(i int) {
	if i < 0 || i >= len(cb.items) {
		return
	}

	changed := i != cb.index
	cb.index = i
	cb.tb.set(cb.items[i])
	cb.tb.selectAll()
	cb.invalidate()

	if changed && cb.onSelect != nil {
		cb.onSelect(i)
	}
}

// button is the drop-down button, inside the border.
func (cb *comboBox) button() image.Rectangle {
	in := cb.local().Inset(2)

	return image.Rect(in.Max.X-scrollSize, in.Min.Y, in.Max.X, in.Max.Y)
}

func (cb *comboBox) gotFocus(bool) {
	cb.tb.selectAll()
	cb.invalidate()
}

func (cb *comboBox) lostFocus() {
	cb.closeUp()
}

func (cb *comboBox) paint(cv canvas) {
	r := cb.local()
	in := cv.sunken(r)
	fg := cb.fontColor

	if !cb.usable() {
		fg = colGrayText // on the control's colour: the VCL answers WM_CTLCOLORSTATIC with it
	}

	cv.fill(in, cb.color)

	b := cb.button()
	if cb.listOnly {
		cb.paintSelection(cv, image.Rect(in.Min.X+1, in.Min.Y+1, b.Min.X-1, in.Max.Y-1), fg)
	} else {
		cb.tb.paint(cv, image.Rect(3, 3, b.Min.X-1, r.Max.Y-3), cb.font, fg, cb.focused() && cb.wdw.active)
	}

	paintArrowButton(cv, b, arrowDown, cb.usable(), cb.pressing && cb.dropped != nil, 5)
}

// paintSelection paints the selected item of a list-only box in tr,
// highlighted while the box has the focus.
func (cb *comboBox) paintSelection(cv canvas, tr image.Rectangle, fg color.RGBA) {
	focus := cb.focused() && cb.wdw.active && cb.dropped == nil
	item := cb.index >= 0 && cb.index < len(cb.items)
	tx := 1

	if cb.images != nil && item {
		// Owner-drawn: the item's icon, then its text, highlighted on
		// its own; the box doesn't indent it.
		cb.images.drawFrame(cv, tr.Min.X+1, tr.Min.Y+(tr.Dy()-cb.images.size)/2, cb.icons[cb.index], false)

		x := tr.Min.X + cb.images.size + 4
		tr = image.Rect(x, tr.Min.Y, min(x+cb.font.width(cb.items[cb.index])+4, tr.Max.X), tr.Max.Y)
		tx = 2
	}

	if focus {
		cv.fill(tr, colSelection)

		fg = colSelText
	}

	if item {
		cb.font.draw(cv.sub(tr), tx, (tr.Dy()-cb.font.height())/2, cb.items[cb.index], fg)
	}

	if focus {
		cv.focusRect(tr)
	}
}

func (cb *comboBox) dropDown() {
	if cb.dropped != nil || len(cb.items) == 0 {
		return
	}

	cb.dropped = newDropList(cb)
	cb.wdw.desk.openPopup(cb.dropped)
	cb.invalidate()
}

func (cb *comboBox) closeUp() {
	if cb.dropped == nil {
		return
	}

	d := cb.dropped
	cb.dropped = nil
	cb.wdw.desk.closePopup(d)
	cb.invalidate()
}

func (cb *comboBox) mouseDown(p image.Point, double bool) {
	if p.In(cb.button()) || cb.listOnly {
		cb.pressing = true
		if cb.dropped != nil {
			cb.closeUp()
		} else {
			cb.dropDown()
		}

		return
	}

	cb.tb.press(cb.font, p.X-3, double, false)
	cb.invalidate()
}

func (cb *comboBox) mouseDrag(p image.Point) {
	if !cb.pressing && !cb.listOnly {
		cb.tb.drag(cb.font, p.X-3)
		cb.invalidate()
	}
}

func (cb *comboBox) mouseUp(image.Point) {
	cb.pressing = false
	cb.tb.dragging = false
	cb.invalidate()
}

// opensDropDown reports whether key k with modifiers m drops the list
// down: F4, or Alt and Down.
func opensDropDown(k ebiten.Key, m mods) bool {
	return k == ebiten.KeyF4 || (m.alt && k == ebiten.KeyDown)
}

// closesDropDown reports whether key k with modifiers m closes the dropped
// list up: Escape, F4, or Alt and Up or Down.
func closesDropDown(k ebiten.Key, m mods) bool {
	altArrow := m.alt && (k == ebiten.KeyUp || k == ebiten.KeyDown)

	return k == ebiten.KeyEscape || k == ebiten.KeyF4 || altArrow
}

func (cb *comboBox) key(k ebiten.Key, m mods) bool {
	if cb.dropped != nil && cb.droppedKey(k, m) {
		return true
	}

	switch {
	case opensDropDown(k, m):
		cb.dropDown()
	case k == ebiten.KeyUp && !m.alt:
		cb.pick(max(cb.index-1, 0))
	case k == ebiten.KeyDown && !m.alt:
		cb.pick(min(cb.index+1, len(cb.items)-1))
	case cb.listOnly:
		switch k {
		case ebiten.KeyHome:
			cb.pick(0)
		case ebiten.KeyEnd:
			cb.pick(len(cb.items) - 1)
		default:
			return false
		}
	case k == ebiten.KeyBackspace:
		cb.backspace()
	default:
		if !cb.tb.key(k, m) {
			return false
		}
	}

	cb.invalidate()

	return true
}

// droppedKey gives a key to the dropped-down list, and reports whether it
// took it. Escape, F4 and Alt with an arrow close the list up, as Enter
// does after the list takes it.
func (cb *comboBox) droppedKey(k ebiten.Key, m mods) bool {
	if closesDropDown(k, m) {
		cb.closeUp()

		return true
	}

	if !cb.dropped.key(k) {
		return false
	}

	if k == ebiten.KeyEnter || k == ebiten.KeyNumpadEnter {
		cb.closeUp()
	}

	cb.invalidate()

	return true
}

// backspace deletes as AutoComplete does: the completed part first.
func (cb *comboBox) backspace() {
	switch a, b := cb.tb.sel(); {
	case a != b:
		cb.tb.replace("")
	case a > 0:
		cb.tb.anchor = a - 1
		cb.tb.replace("")
	}

	cb.typedText()
}

// char types a character with AutoComplete: the text so far plus the
// character is looked up as a prefix of the items (CB_FINDSTRING, ignoring
// case); a match completes the text and selects the completion.
func (cb *comboBox) char(r rune) {
	if r < 0x20 {
		return
	}

	if cb.listOnly {
		// The next item starting with the letter.
		n := len(cb.items)
		for k := 1; k <= n; k++ {
			i := (max(cb.index, -1) + k) % n
			if strings.HasPrefix(strings.ToLower(cb.items[i]), strings.ToLower(string(r))) {
				cb.pick(i)

				if cb.dropped != nil {
					cb.dropped.hot = i
					cb.dropped.show(i)
				}

				return
			}
		}

		return
	}

	a, _ := cb.tb.sel()
	prefix := string(cb.tb.text[:a]) + string(r)

	for i, item := range cb.items {
		if !strings.HasPrefix(strings.ToLower(item), strings.ToLower(prefix)) {
			continue
		}

		cb.tb.set(prefix + string([]rune(item)[len([]rune(prefix)):]))

		cb.tb.anchor, cb.tb.caret = len([]rune(prefix)), len(cb.tb.text)
		if i != cb.index {
			cb.index = i
			if cb.onSelect != nil {
				cb.onSelect(i)
			}
		}

		if cb.dropped != nil {
			cb.dropped.hot = i
			cb.dropped.show(i)
		}

		cb.invalidate()

		return
	}

	cb.tb.typed(r)
	cb.typedText()
}

// typedText follows a text the items don't complete: the selection is
// lost.
func (cb *comboBox) typedText() {
	i := -1

	for k, item := range cb.items {
		if strings.EqualFold(item, cb.tb.String()) {
			i = k
		}
	}

	if i != cb.index {
		cb.index = i
		if cb.onSelect != nil {
			cb.onSelect(i)
		}
	}

	cb.invalidate()
}

func (cb *comboBox) wheel(dy int) {
	if cb.dropped != nil {
		cb.dropped.wheel(dy)

		return
	}

	switch {
	case dy > 0:
		cb.pick(max(cb.index-1, 0))
	case dy < 0:
		cb.pick(min(cb.index+1, len(cb.items)-1))
	}
}

func (cb *comboBox) cursor(p image.Point) ebiten.CursorShapeType {
	if p.In(cb.button()) || cb.listOnly {
		return ebiten.CursorShapeDefault
	}

	return ebiten.CursorShapeText
}
