// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// The file dialogs are the port's own, after Windows 98's Open and Save
// As dialogs: the original used the system's, which a game drawn on its
// own desktop, or running in a browser, can't.

// fileOptions are what a TOpenDialog or TSaveDialog asks for.
type fileOptions struct {
	save      bool
	filter    string // the VCL's Filter: "Name|*.a;*.b|Name|*.c"
	defExt    string // added to a name without an extension
	overwrite bool   // ofOverwritePrompt
	mustExist bool   // ofFileMustExist
}

// fileFilter is one of a dialog's file types.
type fileFilter struct {
	name     string
	patterns []string
}

// parseFilter splits a VCL Filter into its file types.
func parseFilter(s string) []fileFilter {
	parts := strings.Split(s, "|")

	var out []fileFilter

	for i := 0; i+1 < len(parts); i += 2 {
		out = append(out, fileFilter{parts[i], strings.Split(strings.ToLower(parts[i+1]), ";")})
	}

	if len(out) == 0 {
		out = []fileFilter{{"All files (*.*)", []string{"*.*"}}}
	}

	return out
}

// matches reports whether a file name matches one of the patterns; "*.*"
// matches every name, as on Windows.
func (f fileFilter) matches(name string) bool {
	name = strings.ToLower(name)

	for _, p := range f.patterns {
		if p == "*.*" || p == "*" {
			return true
		}

		if ok, _ := path.Match(p, name); ok {
			return true
		}
	}

	return false
}

// fileDialog is an open file dialog.
type fileDialog struct {
	a       *App
	w       *window
	opts    fileOptions
	files   platform.Files
	filters []fileFilter
	dir     string
	entries []platform.Entry // as the view lists them
	result  string

	lookIn *comboBox
	view   *listView
	name   *edit
	types  *comboBox
	up     *toolButton
	dirs   []string // lookIn's folders

	uploadButton, downloadButton *toolButton     // on the web
	uploads                      <-chan []string // the files being uploaded
}

// The view's icons, the look-in list's drive and the toolbar's glyph: o
// outline, y yellow, w white, k black, s shadow, c face, g green.
const (
	iconFolder = `
................
................
..ooooo.........
.oyyyyyo........
oyyyyyyyoooooo..
owwwwwwwwwwwwwk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
owyyyyyyyyyyyyk.
okkkkkkkkkkkkkk.
................
................
................
`
	iconDocument = `
..ssssssss......
..swwwwwwss.....
..swwwwwwsws....
..swwwwwwskks...
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..swwwwwwwwwks..
..sssssssssskk..
...kkkkkkkkkkk..
................
`
	iconDrive = `
................
................
................
................
................
..kkkkkkkkkkkkk.
.kwwwwwwwwwwwwsk
kwcccccccccccssk
kssssssssssssssk
kwccccccccccccsk
kwccccccccggccsk
kwccccccccccccsk
kssssssssssssssk
.kkkkkkkkkkkkkk.
................
................
`
	glyphUp = `
..ooooo.........
.oyyyyyo........
oyyyyyyyoooooo..
owwwwwwkwwwwwwk.
owyyyykkkyyyyyk.
owyyykkkkkyyyyk.
owyyyyykyyyyyyk.
owyyyyykyyyyyyk.
owyyyyykkkkkyyk.
owyyyyyyyyyyyyk.
okkkkkkkkkkkkkk.
`
)

var iconColors = map[byte]color.RGBA{
	'o': rgb(0x808000), 'y': rgb(0xFFFF00), 'w': rgb(0xFFFFFF), 'k': rgb(0x000000), 's': rgb(0x808080),
	'c': rgb(0xC0C0C0), 'g': rgb(0x00FF00),
}

// The file icons' frames.
const (
	fileIconFolder = iota
	fileIconDocument
	fileIconDrive
)

// fileIcons returns the file dialog's image list: a folder, a document and
// a drive.
func (r *resources) fileIcons() *imageList {
	if l := r.lists["fileIcons"]; l != nil {
		return l
	}

	img := image.NewRGBA(image.Rect(0, 0, 48, 16))
	cv := newCanvas(img)
	cv.pattern(0, 0, iconFolder, iconColors)
	cv.pattern(16, 0, iconDocument, iconColors)
	cv.pattern(32, 0, iconDrive, iconColors)

	n := image.NewNRGBA(img.Rect)
	copy(n.Pix, img.Pix) // the pixels are opaque or fully transparent

	l := &imageList{img: n, size: 16, cols: 3}
	r.lists["fileIcons"] = l

	return l
}

// openFile shows a file dialog and waits for the player; it returns the
// chosen file's path. It runs on the handler's goroutine.
func (a *App) openFile(opts fileOptions) (string, bool) {
	f := a.newFileDialog(opts)
	a.fileDlg = f
	r := a.modal(f.w)
	a.fileDlg = nil
	a.desk.remove(f.w)

	return f.result, r == mrOk
}

func (a *App) newFileDialog(opts fileOptions) *fileDialog {
	f := &fileDialog{a: a, opts: opts, files: a.files, filters: parseFilter(opts.filter)}
	title, in, ok, typ := "Open", "Look &in:", "&Open", "Files of &type:"

	if opts.save {
		title, in, ok, typ = "Save As", "Save &in:", "&Save", "Save as &type:"
	}

	comp := layout.Component{
		Class: "TFileDialog", Name: "FileDialog", Caption: title,
		Props: map[string]string{"BorderStyle": "bsDialog", "ClientWidth": "420", "ClientHeight": "240", "Position": "poScreenCenter"},
		Children: []layout.Component{
			{Class: "TLabel", Name: "LookInLabel", Caption: in, Bounds: layout.Rect{X: 6, Y: 10, W: 50, H: 13}},
			{
				Class: "TComboBox", Name: "LookIn", Bounds: layout.Rect{X: 74, Y: 5, W: 198, H: 21},
				Props: map[string]string{"Style": "csDropDownList", "ItemHeight": "16", "TabOrder": "0"},
			},
			{
				Class: "TListView", Name: "View", Bounds: layout.Rect{X: 6, Y: 33, W: 408, H: 138},
				Props: map[string]string{"TabOrder": "1"},
			},
			{Class: "TLabel", Name: "NameLabel", Caption: "File &name:", Bounds: layout.Rect{X: 8, Y: 183, W: 50, H: 13}},
			{
				Class: "TEdit", Name: "FileName", Bounds: layout.Rect{X: 81, Y: 179, W: 233, H: 21},
				Props: map[string]string{"TabOrder": "2"},
			},
			{Class: "TLabel", Name: "TypeLabel", Caption: typ, Bounds: layout.Rect{X: 8, Y: 209, W: 70, H: 13}},
			{
				Class: "TComboBox", Name: "Types", Bounds: layout.Rect{X: 81, Y: 205, W: 233, H: 21},
				Props: map[string]string{"Style": "csDropDownList", "TabOrder": "3"},
			},
			{
				Class: "TButton", Name: "OK", Caption: ok, Bounds: layout.Rect{X: 333, Y: 179, W: 75, H: 23},
				Props: map[string]string{"Default": "True", "TabOrder": "4"},
			},
			{
				Class: "TButton", Name: "Cancel", Caption: "Cancel", Bounds: layout.Rect{X: 333, Y: 205, W: 75, H: 23},
				Props: map[string]string{"Cancel": "True", "ModalResult": strconv.Itoa(mrCancel), "TabOrder": "5"},
			},
		},
	}
	w := newWindow(a.desk, &comp)
	f.w = w
	f.lookIn = w.find("LookIn").(*comboBox)
	f.view = w.find("View").(*listView)
	f.name = w.find("FileName").(*edit)
	f.types = w.find("Types").(*comboBox)
	w.find("LookInLabel").(*label).focusControl = f.lookIn
	w.find("NameLabel").(*label).focusControl = f.name
	w.find("TypeLabel").(*label).focusControl = f.types
	f.view.images = a.res.fileIcons()
	f.lookIn.images = a.res.fileIcons()

	f.up = &toolButton{glyph: glyphUp, onPress: f.upOneLevel}
	addWidget(w, f.up, image.Rect(280, 4, 303, 26))

	if a.xfer != nil {
		f.addTransfers()
	}

	for _, ft := range f.filters {
		f.types.items = append(f.types.items, ft.name)
	}

	f.types.index = 0
	f.types.onSelect = func(int) { f.list() }
	f.lookIn.onSelect = func(i int) {
		if i >= 0 && i < len(f.dirs) {
			f.navigate(f.dirs[i])
		}
	}
	f.view.onSelect = f.viewClick
	f.view.onDblClick = f.open
	w.find("OK").(*button).onClick = f.okClick
	w.onKey = func(k ebiten.Key, m mods) bool {
		if k == ebiten.KeyBackspace && w.focus == f.view {
			f.upOneLevel()

			return true
		}

		return false
	}
	dir := a.lastDir

	if dir == "" {
		dir = a.files.Home()
	}

	f.navigate(dir)

	w.focus = f.name
	a.desk.add(w)

	return f
}

// lookInIndent is the look-in list's indent for each level of folders.
const lookInIndent = 10

// navigate shows a folder.
func (f *fileDialog) navigate(dir string) {
	if st, err := f.files.Stat(dir); err != nil || !st.Dir {
		return
	}

	f.dir = dir
	// Look in: the roots, with the folder's ancestors under its own, each
	// indented under its parent.
	f.dirs = nil

	var chain []string

	for d := dir; ; d = f.files.Dir(d) {
		chain = append(chain, d)
		if f.files.Dir(d) == d {
			break
		}
	}

	slices.Reverse(chain)

	var (
		items          []string
		icons, indents []int
	)
	add := func(d string, depth int) {
		name, icon := f.files.Base(d), fileIconFolder
		if depth == 0 {
			name, icon = d, fileIconDrive
		}

		f.dirs = append(f.dirs, d)
		items = append(items, name)
		icons = append(icons, icon)
		indents = append(indents, depth*lookInIndent)
	}
	roots := f.files.Roots()

	if !slices.Contains(roots, chain[0]) { // a root the storage doesn't list
		roots = append(roots, chain[0])
	}

	sel := 0

	for _, root := range roots {
		if root != chain[0] {
			add(root, 0)

			continue
		}

		for depth, d := range chain {
			add(d, depth)
		}

		sel = len(items) - 1
	}

	f.lookIn.items, f.lookIn.icons, f.lookIn.indents = items, icons, indents
	f.lookIn.index = sel
	f.lookIn.invalidate()
	f.list()
}

// list fills the view: the folders, then the files of the chosen type,
// each sorted by name; hidden names (with a leading dot) are left out.
func (f *fileDialog) list() {
	entries, _ := f.files.List(f.dir)
	ft := f.filters[max(f.types.index, 0)]

	var dirs, files []platform.Entry

	for _, e := range entries {
		switch {
		case strings.HasPrefix(e.Name, "."):
		case e.Dir:
			dirs = append(dirs, e)
		case ft.matches(e.Name):
			files = append(files, e)
		}
	}

	byName := func(a, b platform.Entry) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}
	slices.SortFunc(dirs, byName)
	slices.SortFunc(files, byName)

	f.entries = append(dirs, files...)
	items := make([]listItem, len(f.entries))

	for i, e := range f.entries {
		img := 1
		if e.Dir {
			img = 0
		}

		items[i] = listItem{e.Name, img}
	}

	f.view.setItems(items)

	f.view.selected, f.view.hot = -1, -1
	f.view.scroll.pos = 0
	f.view.laid = false
	f.view.invalidate()
}

func (f *fileDialog) selectedEntry() (platform.Entry, bool) {
	if i := f.view.selected; i >= 0 && i < len(f.entries) {
		return f.entries[i], true
	}

	return platform.Entry{}, false
}

// viewClick puts a selected file's name in the name box.
func (f *fileDialog) viewClick() {
	if e, ok := f.selectedEntry(); ok && !e.Dir {
		f.name.setText(e.Name)
	}
}

// open is a double click on the view: a folder opens, a file is chosen.
func (f *fileDialog) open() {
	e, ok := f.selectedEntry()
	if !ok {
		return
	}

	if e.Dir {
		f.navigate(f.files.Join(f.dir, e.Name))

		return
	}

	f.name.setText(e.Name)
	f.okClick()
}

func (f *fileDialog) upOneLevel() {
	if parent := f.files.Dir(f.dir); parent != f.dir {
		f.navigate(parent)
	}
}

// okClick is "Open" or "Save": a folder's name opens it; a file's is
// checked as the dialog's options ask, then chosen.
func (f *fileDialog) okClick() {
	text := strings.TrimSpace(f.name.text())
	if f.w.focus == f.view || text == "" {
		if e, ok := f.selectedEntry(); ok && e.Dir {
			f.navigate(f.files.Join(f.dir, e.Name))

			return
		}
	}

	if text == "" {
		return
	}

	p := text
	if !f.files.IsAbs(p) {
		p = f.files.Join(f.dir, p)
	}

	if st, err := f.files.Stat(p); err == nil && st.Dir {
		f.navigate(p)
		f.name.setText("")

		return
	}

	base := f.files.Base(p)
	if !strings.Contains(base, ".") && f.opts.defExt != "" {
		withExt := p + "." + strings.TrimPrefix(f.opts.defExt, ".")
		if _, err := f.files.Stat(withExt); f.opts.save || err == nil {
			p = withExt
		}
	}

	if st, err := f.files.Stat(f.files.Dir(p)); err != nil || !st.Dir {
		f.alert(false, p+"\nPath does not exist.\nPlease verify the correct path was given.", nil)

		return
	}

	_, err := f.files.Stat(p)
	exists := err == nil

	switch {
	case f.opts.save && exists && f.opts.overwrite:
		f.alert(true, p+" already exists.\nDo you want to replace it?", func() { f.accept(p) })
	case !f.opts.save && !exists && f.opts.mustExist:
		f.alert(false, f.files.Base(p)+"\nFile not found.\nPlease verify the correct file name was given.", nil)
	default:
		f.accept(p)
	}
}

// alert shows a warning over the dialog; yes runs if the player answers
// Yes.
func (f *fileDialog) alert(yesNo bool, text string, yes func()) {
	d := f.a.desk
	m := messageDialog(d, game.Warning, yesNo, text)
	m.caption = f.w.caption
	m.onModal = func(r int) {
		d.remove(m)

		if r == mrYes && yes != nil {
			yes()
		}
	}
	d.showModal(m)
}

func (f *fileDialog) accept(p string) {
	f.result = p
	f.a.lastDir = f.files.Dir(p)
	f.w.endModal(mrOk)
}

// toolButton is a flat toolbar button: its glyph alone, with a thin
// raised edge under the mouse and a sunken one while pressed.
type toolButton struct {
	ctl
	glyph   string
	hot     bool
	down    bool
	inside  bool
	onPress func()
}

func (b *toolButton) paint(cv canvas) {
	r := b.local()
	cv.fill(r, colFace)

	off := 0

	switch {
	case b.down && b.inside:
		cv.edge(r, colShadow, colHighlight)

		off = 1
	case b.hot || b.down:
		cv.edge(r, colHighlight, colShadow)
	}

	rows := strings.Split(strings.Trim(b.glyph, "\n"), "\n")
	gx := (r.Dx()-len(rows[0]))/2 + off
	gy := (r.Dy()-len(rows))/2 + off
	cv.pattern(gx, gy, b.glyph, iconColors)
}

func (b *toolButton) mouseDown(p image.Point, double bool) {
	b.down, b.inside = true, true
	b.invalidate()
}

func (b *toolButton) mouseDrag(p image.Point) {
	if in := p.In(b.local()); in != b.inside {
		b.inside = in
		b.invalidate()
	}
}

func (b *toolButton) mouseUp(p image.Point) {
	click := b.down && p.In(b.local())
	b.down = false
	b.invalidate()

	if click && b.onPress != nil {
		b.onPress()
	}
}

func (b *toolButton) hover(p image.Point, out bool) {
	if b.hot != !out {
		b.hot = !out
		b.invalidate()
	}
}

// addWidget adds a control the forms' layouts don't describe.
func addWidget(w *window, k widget, r image.Rectangle) {
	c := k.base()
	c.self = k
	c.win = w
	c.r = r
	c.visible, c.enabled = true, true
	c.font, c.fontColor, c.color = w.res.sans, colText, w.color
	w.kids = append(w.kids, k)
}
