// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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
	keepSel  bool                // ES_NOHIDESEL: the selection shows without the focus
	valid    func(chr rune) bool // characters that may be typed
	onChange func()              // after any change of the text
	blink    time.Time           // when the caret was last moved
	dragging bool
}

func (tbx *textBox) String() string {
	return string(tbx.text)
}

// set replaces the text, putting the caret at its end.
func (tbx *textBox) set(txt string) {
	tbx.text = []rune(txt)
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

// replace replaces the selection with txt, within the length limit.
func (tbx *textBox) replace(txt string) {
	if tbx.readOnly {
		return
	}

	beg, end := tbx.sel()
	in := []rune(txt)

	if tbx.maxLen > 0 {
		room := tbx.maxLen - (len(tbx.text) - (end - beg))
		if room < len(in) {
			in = in[:max(room, 0)]
		}
	}

	if len(in) == 0 && beg == end {
		return
	}

	tbx.text = append(tbx.text[:beg:beg], append(in, tbx.text[end:]...)...)
	tbx.caret = beg + len(in)
	tbx.anchor = tbx.caret
	tbx.changed()
}

// typed handles a typed character: one cp1252 can hold, that the box
// accepts.
func (tbx *textBox) typed(chr rune) {
	control := chr < 0x20 || chr == 0x7f
	outsideANSI := string(delphi.EncodeANSI(string(chr))) == "?" && chr != '?'

	if control || outsideANSI {
		return
	}

	if tbx.valid != nil && !tbx.valid(chr) {
		return
	}

	tbx.replace(string(chr))
}

// key handles the editing keys; it reports whether key was one.
func (tbx *textBox) key(key ebiten.Key, mds mods) bool {
	move := func(to int) {
		tbx.caret = max(min(to, len(tbx.text)), 0)
		if !mds.shift {
			tbx.anchor = tbx.caret
		}

		tbx.blink = time.Now()
	}
	beg, end := tbx.sel()

	switch key {
	case ebiten.KeyLeft:
		switch {
		case !mds.shift && beg != end:
			move(beg)
		case mds.ctrl:
			move(tbx.wordLeft())
		default:
			move(tbx.caret - 1)
		}
	case ebiten.KeyRight:
		switch {
		case !mds.shift && beg != end:
			move(end)
		case mds.ctrl:
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

	if beg, end := tbx.sel(); beg == end {
		if beg == 0 {
			return
		}

		tbx.anchor = beg - 1
	}

	tbx.replace("")
}

// del deletes the selection, or the character after the caret.
func (tbx *textBox) del() {
	if tbx.readOnly {
		return
	}

	if beg, end := tbx.sel(); beg == end {
		if end == len(tbx.text) {
			return
		}

		tbx.caret = end + 1
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
func (tbx *textBox) at(fnt face, x int) int {
	x += tbx.scroll
	wid := 0

	for i := range tbx.text {
		cw := fnt.width(string(tbx.text[i]))
		if x < wid+cw/2 {
			return i
		}

		wid += cw
	}

	return len(tbx.text)
}

// fit scrolls the text so that the caret shows in a box wid pixels wide.
func (tbx *textBox) fit(fnt face, wid int) {
	cx := fnt.width(string(tbx.text[:tbx.caret]))
	if cx-tbx.scroll > wid-1 {
		tbx.scroll = cx - wid + 1 + wid/4
	}

	if cx < tbx.scroll {
		tbx.scroll = max(cx-wid/4, 0)
	}

	if fnt.width(string(tbx.text))-tbx.scroll < wid-1 {
		tbx.scroll = max(fnt.width(string(tbx.text))-wid+1, 0)
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

// paint draws the text in rect with its selection (when focused) and the
// blinking caret.
func (tbx *textBox) paint(cv canvas, rect image.Rectangle, fnt face, fg color.RGBA, focused bool) {
	tbx.fit(fnt, rect.Dx())

	in := cv.sub(rect)
	x := -tbx.scroll
	txt := string(tbx.text)
	beg, end := tbx.sel()

	// The selection shows with the focus, or always with ES_NOHIDESEL.
	showSel := (focused || tbx.keepSel) && beg != end
	if showSel {
		pre, mid, post := string(tbx.text[:beg]), string(tbx.text[beg:end]), string(tbx.text[end:])
		x = fnt.draw(in, x, 0, pre, fg)
		wid := fnt.width(mid)
		in.fill(image.Rect(x, 0, x+wid, fnt.height()), colSelection)

		x = fnt.draw(in, x, 0, mid, colSelText)
		fnt.draw(in, x, 0, post, fg)
	} else {
		fnt.draw(in, x, 0, txt, fg)
	}

	hasCaret := focused && beg == end && !tbx.readOnly
	if hasCaret && blinkOn(time.Since(tbx.blink)) {
		cx := fnt.width(string(tbx.text[:tbx.caret])) - tbx.scroll
		for y := range fnt.height() {
			in.invert(cx, y)
		}
	}
}

func (tbx *textBox) press(fnt face, x int, double, shift bool) {
	if double {
		tbx.anchor = tbx.at(fnt, x)
		tbx.caret = tbx.anchor
		tbx.anchor = tbx.wordLeft()
		tbx.caret = tbx.wordRight()

		return
	}

	tbx.caret = tbx.at(fnt, x)
	if !shift {
		tbx.anchor = tbx.caret
	}

	tbx.dragging = true
	tbx.blink = time.Now()
}

func (tbx *textBox) drag(fnt face, x int) {
	if tbx.dragging {
		tbx.caret = tbx.at(fnt, x)
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
func (e *edit) setText(txt string) {
	if e.tb.String() != txt {
		e.tb.set(txt)
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
	rect := e.local()
	in := cv.sunken(rect)
	fg := e.fontColor

	if !e.usable() {
		fg = colGrayText // on the control's colour: the VCL answers WM_CTLCOLORSTATIC with it
	}

	cv.fill(in, e.color)
	e.tb.paint(cv, image.Rect(e.inset, e.inset, rect.Max.X-3, rect.Max.Y-2), e.font, fg, e.focused() && e.wdw.active)
}

func (e *edit) mouseDown(pnt image.Point, double bool) {
	e.tb.press(e.font, pnt.X-e.inset, double, false)
	e.invalidate()
}

func (e *edit) mouseDrag(pnt image.Point) {
	e.tb.drag(e.font, pnt.X-e.inset)
	e.invalidate()
}

func (e *edit) mouseUp(image.Point) {
	e.tb.dragging = false
}

func (e *edit) key(key ebiten.Key, mds mods) bool {
	if e.tb.key(key, mds) {
		e.invalidate()

		return true
	}

	return false
}

func (e *edit) char(chr rune) {
	e.tb.typed(chr)
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
	onChangeValue      func(val int)
}

func (se *spinEdit) setup(comp *layout.Component) {
	se.edit.setup(comp)

	se.inset = 2 // SetEditRect puts the text at the client area's corner
	se.minValue = se.propertyInt("MinValue", 0)
	se.maxValue = se.propertyInt("MaxValue", 0)
	se.increment = se.propertyInt("Increment", 1)
	se.tb.set(strconv.Itoa(se.propertyInt("Value", 0)))

	// IsValidChar: digits, signs and the decimal separator.
	se.tb.valid = func(chr rune) bool {
		return isDigit(chr) || strings.ContainsRune("+-.", chr)
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
	if val, ok := delphi.StrToInt(se.tb.String()); ok {
		return int(val)
	}

	return se.minValue
}

// check clamps val to the range, unless MaxValue = MinValue (CheckValue).
func (se *spinEdit) check(val int) int {
	if se.maxValue != se.minValue {
		val = max(min(val, se.maxValue), se.minValue)
	}

	return val
}

// setValue sets Value as the program does: OnChange runs if the text
// changes.
func (se *spinEdit) setValue(val int) {
	se.setText(strconv.Itoa(se.check(val)))
}

// lostFocus clamps the value, as CMExit does.
func (se *spinEdit) lostFocus() {
	if val := se.value(); se.check(val) != val {
		se.setValue(val)
	}
}

// buttons returns the up and down buttons: TSpinButton sits inside the
// edit's border, 15 pixels wide and 5 short of the edit's height, and its
// down button covers the up button's last row (AdjustSize).
func (se *spinEdit) buttons() (up, down image.Rectangle) {
	rect := se.local().Inset(2)
	x := rect.Min.X + se.rect.Dx() - 15 - 5
	hgt := se.rect.Dy() - 5
	up = image.Rect(x, rect.Min.Y, x+15, rect.Min.Y+hgt/2)
	down = image.Rect(x, up.Max.Y-1, x+15, rect.Min.Y+hgt)

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
	rect := se.local()
	in := cv.sunken(rect)
	fg := se.fontColor

	if !se.usable() {
		fg = colGrayText // on the control's colour: the VCL answers WM_CTLCOLORSTATIC with it
	}

	cv.fill(in, se.color)

	up, down := se.buttons()
	se.tb.paint(cv, image.Rect(se.inset, se.inset, up.Min.X-1, rect.Max.Y-2), se.font, fg, se.focused() && se.wdw.active)

	// Two speed buttons, the down one painted last, each drawn by
	// DrawFrameControl with its glyph centred inside the frame.
	for i, box := range []image.Rectangle{up, down} {
		off := cv.softButton(box, se.pressed == i+1 && se.inside)

		glyph := glyphSpinUp
		if i == 1 {
			glyph = glyphSpinDown
		}

		in := box.Inset(2)
		x := in.Min.X + (in.Dx()-9+1)/2 + off
		y := in.Min.Y + (in.Dy()-6+1)/2 + off

		if !se.usable() {
			cv.embossed(x, y, glyph)

			continue
		}

		cv.pattern(x, y, glyph, map[byte]color.RGBA{'#': colText})
	}
}

func (se *spinEdit) mouseDown(pnt image.Point, double bool) {
	up, down := se.buttons()
	switch {
	case pnt.In(down): // on top of the up button
		se.pressed = 2
	case pnt.In(up):
		se.pressed = 1
	default:
		se.edit.mouseDown(pnt, double)

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

func (se *spinEdit) mouseDrag(pnt image.Point) {
	if se.pressed == 0 {
		se.edit.mouseDrag(pnt)

		return
	}

	up, down := se.buttons()
	rect := up

	if se.pressed == 2 {
		rect = down
	}

	if in := pnt.In(rect); in != se.inside {
		se.inside = in
		se.invalidate()
	}
}

func (se *spinEdit) mouseUp(pnt image.Point) {
	if se.pressed == 0 {
		se.edit.mouseUp(pnt)

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

func (se *spinEdit) key(key ebiten.Key, mds mods) bool {
	switch key {
	case ebiten.KeyUp:
		se.spin(1)
	case ebiten.KeyDown:
		se.spin(-1)
	default:
		return se.edit.key(key, mds)
	}

	return true
}

func (se *spinEdit) cursor(pnt image.Point) ebiten.CursorShapeType {
	if up, down := se.buttons(); pnt.In(up) || pnt.In(down) {
		return ebiten.CursorShapeDefault
	}

	return ebiten.CursorShapeText
}

// lineStyle is how a memo draws a line: in a colour, and in bold or not.
type lineStyle struct {
	color color.RGBA
	bold  bool
}

// memoRow is a row of a memo's text: a line, or the part of one that
// wrapping leaves on the row.
type memoRow struct {
	text string
	line int // the line's index
}

// memo is a read-only TMemo: Battle Info, with a vertical scroll bar, and
// the preview of its colours in Preferences, with none. Each line can have
// a style of its own, which the original's memo doesn't have.
type memo struct {
	control
	lines  []string    // as the program set them
	styles []lineStyle // the lines', as the program set them; none is the font's
	rows   []memoRow   // the lines wrapped to the width
	bar    bool        // ScrollBars is ssVertical
	laid   bool        // the rows and the scroll bar fit the size
	scroll scrollBar
}

func (m *memo) setup(comp *layout.Component) {
	m.windowed = true
	m.tabStop = m.property("TabStop") != "False"
	m.color = vclColor(m.property("Color"), colWindow)
	m.bar = m.property("ScrollBars") == "ssVertical"

	m.lines = slices.Clone(comp.Items)
	m.scroll = scrollBar{
		vertical: true,
	}
	m.scroll.onScroll = m.invalidate
}

func (m *memo) focusable() bool {
	return true
}

// textRect returns the client rectangle for the text: inside the border,
// left of the scroll bar.
func (m *memo) textRect() image.Rectangle {
	rect := m.local().Inset(2)
	if m.bar {
		rect.Max.X -= scrollSize
	}

	return rect
}

// barRect returns the scroll bar's rectangle, along the right edge inside
// the border, or an empty one without the bar.
func (m *memo) barRect() image.Rectangle {
	if !m.bar {
		return image.Rectangle{}
	}

	rect := m.local().Inset(2)

	return image.Rect(rect.Max.X-scrollSize, rect.Min.Y, rect.Max.X, rect.Max.Y)
}

// setLines replaces the text and its styles: styles may be shorter than
// lines, or nil, and the lines without one have the font's. Lines added at
// the end scroll into view.
func (m *memo) setLines(lines []string, styles []lineStyle) {
	same := slices.Equal(lines, m.lines) && slices.Equal(styles, m.styles)
	if same {
		return
	}

	grew := len(lines) > len(m.lines)
	m.lines = slices.Clone(lines)
	m.styles = slices.Clone(styles)
	m.layout()

	if grew {
		m.scroll.set(m.scroll.limit())
	}

	m.invalidate()
}

// style returns the face and colour line i is drawn in: its style's, or
// the font's.
func (m *memo) style(i int) (face, color.RGBA) {
	if i >= len(m.styles) {
		return m.font, m.fontColor
	}

	sty := m.styles[i]
	if sty.bold {
		return boldFace(m.font), sty.color
	}

	return m.font, sty.color
}

// layout wraps the lines, as WordWrap does, each in its own face, and
// sizes the scroll bar.
func (m *memo) layout() {
	tr := m.textRect()
	m.rows = m.rows[:0]

	for i, lin := range m.lines {
		fnt, _ := m.style(i)
		for _, txt := range wrap(fnt, lin, tr.Dx()-2) {
			m.rows = append(m.rows, memoRow{
				text: txt,
				line: i,
			})
		}
	}

	m.scroll.rect = m.barRect()
	m.scroll.min, m.scroll.max = 0, len(m.rows)-1
	m.scroll.page = (tr.Dy() - 2) / m.font.height()
	m.scroll.set(m.scroll.pos)

	m.laid = true
}

// resized wraps the lines again, and moves the scroll bar, for the new
// size.
func (m *memo) resized() {
	m.layout()
}

func (m *memo) paint(cv canvas) {
	if !m.laid {
		m.layout()
	}

	rect := m.local()
	cv.sunken(rect)

	tr := m.textRect()
	cv.fill(tr, m.color)

	// The format rectangle is a pixel inside the client area.
	in := cv.sub(tr)

	for i := m.scroll.pos; i < len(m.rows); i++ {
		y := 1 + (i-m.scroll.pos)*m.font.height()
		if y >= tr.Dy() {
			break
		}

		row := m.rows[i]
		fnt, clr := m.style(row.line)
		fnt.draw(in, 1, y, row.text, clr)
	}

	if m.bar {
		m.scroll.paint(cv)
	}
}

func (m *memo) mouseDown(pnt image.Point, _ bool) {
	if m.scroll.press(pnt) {
		m.invalidate()
	}
}

func (m *memo) mouseDrag(pnt image.Point) {
	if m.scroll.pressed != partNone {
		m.scroll.drag(pnt)
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

func (m *memo) key(key ebiten.Key, mm mods) bool {
	bar := &m.scroll

	switch key {
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

	cb.items = slices.Clone(comp.Items)
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
// does (ItemIndex := -1, the items again, ItemIndex := idx): the form isn't
// active yet, so the text isn't selected.
func (cb *comboBox) reselect() {
	idx := cb.index
	cb.setIndex(-1)
	cb.setIndex(idx)
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
	rect := cb.local()
	in := cv.sunken(rect)
	fg := cb.fontColor

	if !cb.usable() {
		fg = colGrayText // on the control's colour: the VCL answers WM_CTLCOLORSTATIC with it
	}

	cv.fill(in, cb.color)

	box := cb.button()
	if cb.listOnly {
		cb.paintSelection(cv, image.Rect(in.Min.X+1, in.Min.Y+1, box.Min.X-1, in.Max.Y-1), fg)
	} else {
		cb.tb.paint(cv, image.Rect(3, 3, box.Min.X-1, rect.Max.Y-3), cb.font, fg, cb.focused() && cb.wdw.active)
	}

	paintArrowButton(cv, box, arrowDown, cb.usable(), cb.pressing && cb.dropped != nil, 5)
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

	dl := cb.dropped
	cb.dropped = nil
	cb.wdw.desk.closePopup(dl)
	cb.invalidate()
}

func (cb *comboBox) mouseDown(pnt image.Point, double bool) {
	if pnt.In(cb.button()) || cb.listOnly {
		cb.pressing = true
		if cb.dropped != nil {
			cb.closeUp()
		} else {
			cb.dropDown()
		}

		return
	}

	cb.tb.press(cb.font, pnt.X-3, double, false)
	cb.invalidate()
}

func (cb *comboBox) mouseDrag(pnt image.Point) {
	if !cb.pressing && !cb.listOnly {
		cb.tb.drag(cb.font, pnt.X-3)
		cb.invalidate()
	}
}

func (cb *comboBox) mouseUp(image.Point) {
	cb.pressing = false
	cb.tb.dragging = false
	cb.invalidate()
}

// opensDropDown reports whether key with modifiers mds drops the list
// down: F4, or Alt and Down.
func opensDropDown(key ebiten.Key, mds mods) bool {
	return key == ebiten.KeyF4 || (mds.alt && key == ebiten.KeyDown)
}

// closesDropDown reports whether key with modifiers mds closes the dropped
// list up: Escape, F4, or Alt and Up or Down.
func closesDropDown(key ebiten.Key, mds mods) bool {
	altArrow := mds.alt && (key == ebiten.KeyUp || key == ebiten.KeyDown)

	return key == ebiten.KeyEscape || key == ebiten.KeyF4 || altArrow
}

func (cb *comboBox) key(key ebiten.Key, mds mods) bool {
	if cb.dropped != nil && cb.droppedKey(key, mds) {
		return true
	}

	switch {
	case opensDropDown(key, mds):
		cb.dropDown()
	case key == ebiten.KeyUp && !mds.alt:
		cb.pick(max(cb.index-1, 0))
	case key == ebiten.KeyDown && !mds.alt:
		cb.pick(min(cb.index+1, len(cb.items)-1))
	case cb.listOnly:
		switch key {
		case ebiten.KeyHome:
			cb.pick(0)
		case ebiten.KeyEnd:
			cb.pick(len(cb.items) - 1)
		default:
			return false
		}
	case key == ebiten.KeyBackspace:
		cb.backspace()
	default:
		if !cb.tb.key(key, mds) {
			return false
		}
	}

	cb.invalidate()

	return true
}

// droppedKey gives a key to the dropped-down list, and reports whether it
// took it. Escape, F4 and Alt with an arrow close the list up, as Enter
// does after the list takes it.
func (cb *comboBox) droppedKey(key ebiten.Key, mds mods) bool {
	if closesDropDown(key, mds) {
		cb.closeUp()

		return true
	}

	if !cb.dropped.key(key) {
		return false
	}

	if key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter {
		cb.closeUp()
	}

	cb.invalidate()

	return true
}

// backspace deletes as AutoComplete does: the completed part first.
func (cb *comboBox) backspace() {
	switch beg, end := cb.tb.sel(); {
	case beg != end:
		cb.tb.replace("")
	case beg > 0:
		cb.tb.anchor = beg - 1
		cb.tb.replace("")
	}

	cb.typedText()
}

// char types a character with AutoComplete: the text so far plus the
// character is looked up as a prefix of the items (CB_FINDSTRING, ignoring
// case); a match completes the text and selects the completion.
func (cb *comboBox) char(chr rune) {
	if chr < 0x20 {
		return
	}

	if cb.listOnly {
		// The next item starting with the letter.
		n := len(cb.items)
		for off := 1; off <= n; off++ {
			i := (max(cb.index, -1) + off) % n
			if strings.HasPrefix(strings.ToLower(cb.items[i]), strings.ToLower(string(chr))) {
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

	beg, _ := cb.tb.sel()
	prefix := string(cb.tb.text[:beg]) + string(chr)

	for i, item := range cb.items {
		if !strings.HasPrefix(strings.ToLower(item), strings.ToLower(prefix)) {
			continue
		}

		cb.tb.set(prefix + string([]rune(item)[utf8.RuneCountInString(prefix):]))

		cb.tb.anchor, cb.tb.caret = utf8.RuneCountInString(prefix), len(cb.tb.text)
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

	cb.tb.typed(chr)
	cb.typedText()
}

// typedText follows a text the items don't complete: the selection is
// lost.
func (cb *comboBox) typedText() {
	i := -1

	for idx, item := range cb.items {
		if strings.EqualFold(item, cb.tb.String()) {
			i = idx
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

func (cb *comboBox) cursor(pnt image.Point) ebiten.CursorShapeType {
	if pnt.In(cb.button()) || cb.listOnly {
		return ebiten.CursorShapeDefault
	}

	return ebiten.CursorShapeText
}
