// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"path"
	"slices"
	"strings"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/platform"
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
func (ff fileFilter) matches(name string) bool {
	name = strings.ToLower(name)

	for _, p := range ff.patterns {
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
	app     *Application
	window  *window
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
	iconFolder string = `
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
	iconDocument string = `
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
	iconDrive string = `
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
	glyphUp string = `
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
	'o': rgb(0x808000),
	'y': rgb(0xFFFF00),
	'w': rgb(0xFFFFFF),
	'k': rgb(0x000000),
	's': rgb(0x808080),
	'c': rgb(0xC0C0C0),
	'g': rgb(0x00FF00),
}

// The file icons' frames.
const (
	fileIconFolder int = iota
	fileIconDocument
	fileIconDrive
)

// lookInIndent is the look-in list's indent for each level of folders.
const lookInIndent int = 10

// navigate shows a folder.
func (fd *fileDialog) navigate(dir string) {
	if st, err := fd.files.Stat(dir); err != nil || !st.Dir {
		return
	}

	fd.dir = dir
	// Look in: the roots, with the folder's ancestors under its own, each
	// indented under its parent.
	fd.dirs = nil

	var chain []string

	for d := dir; ; d = fd.files.Dir(d) {
		chain = append(chain, d)
		if fd.files.Dir(d) == d {
			break
		}
	}

	slices.Reverse(chain)

	var (
		items          []string
		icons, indents []int
	)
	add := func(d string, depth int) {
		name, icon := fd.files.Base(d), fileIconFolder
		if depth == 0 {
			name, icon = d, fileIconDrive
		}

		fd.dirs = append(fd.dirs, d)
		items = append(items, name)
		icons = append(icons, icon)
		indents = append(indents, depth*lookInIndent)
	}
	roots := fd.files.Roots()

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

	fd.lookIn.items, fd.lookIn.icons, fd.lookIn.indents = items, icons, indents
	fd.lookIn.index = sel
	fd.lookIn.invalidate()
	fd.list()
}

// list fills the view: the folders, then the files of the chosen type,
// each sorted by name; hidden names (with a leading dot) are left out.
func (fd *fileDialog) list() {
	entries, _ := fd.files.List(fd.dir)
	ft := fd.filters[max(fd.types.index, 0)]

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

	fd.entries = slices.Concat(dirs, files)
	items := make([]listItem, len(fd.entries))

	for i, e := range fd.entries {
		img := 1
		if e.Dir {
			img = 0
		}

		items[i] = listItem{e.Name, img}
	}

	fd.view.setItems(items)

	fd.view.selected, fd.view.hot = -1, -1
	fd.view.scroll.pos = 0
	fd.view.laid = false
	fd.view.invalidate()
}

func (fd *fileDialog) selectedEntry() (platform.Entry, bool) {
	if i := fd.view.selected; i >= 0 && i < len(fd.entries) {
		return fd.entries[i], true
	}

	return platform.Entry{}, false
}

// viewClick puts a selected file's name in the name box.
func (fd *fileDialog) viewClick() {
	if e, ok := fd.selectedEntry(); ok && !e.Dir {
		fd.name.setText(e.Name)
	}
}

// open is a double click on the view: a folder opens, a file is chosen.
func (fd *fileDialog) open() {
	e, ok := fd.selectedEntry()
	if !ok {
		return
	}

	if e.Dir {
		fd.navigate(fd.files.Join(fd.dir, e.Name))

		return
	}

	fd.name.setText(e.Name)
	fd.okClick()
}

func (fd *fileDialog) upOneLevel() {
	if parent := fd.files.Dir(fd.dir); parent != fd.dir {
		fd.navigate(parent)
	}
}

// okClick is "Open" or "Save": a folder's name opens it; a file's is
// checked as the dialog's options ask, then chosen.
func (fd *fileDialog) okClick() {
	txt := strings.TrimSpace(fd.name.text())
	if fd.window.focus == fd.view || txt == "" {
		if e, ok := fd.selectedEntry(); ok && e.Dir {
			fd.navigate(fd.files.Join(fd.dir, e.Name))

			return
		}
	}

	if txt == "" {
		return
	}

	p := txt
	if !fd.files.IsAbs(p) {
		p = fd.files.Join(fd.dir, p)
	}

	if st, err := fd.files.Stat(p); err == nil && st.Dir {
		fd.navigate(p)
		fd.name.setText("")

		return
	}

	fd.choose(fd.withDefaultExt(p))
}

// withDefaultExt adds the default extension to a file name without one, when
// the dialog saves or a file has that name.
func (fd *fileDialog) withDefaultExt(p string) string {
	if strings.Contains(fd.files.Base(p), ".") || fd.opts.defExt == "" {
		return p
	}

	withExt := p + "." + strings.TrimPrefix(fd.opts.defExt, ".")
	if _, err := fd.files.Stat(withExt); fd.opts.save || err == nil {
		return withExt
	}

	return p
}

// choose checks a file's path as the dialog's options ask, then chooses
// it.
func (fd *fileDialog) choose(p string) {
	if st, err := fd.files.Stat(fd.files.Dir(p)); err != nil || !st.Dir {
		fd.alert(false, p+"\nPath does not exist.\nPlease verify the correct path was given.", nil)

		return
	}

	_, err := fd.files.Stat(p)
	exists := err == nil

	switch {
	case fd.opts.save && exists && fd.opts.overwrite:
		fd.alert(true, p+" already exists.\nDo you want to replace it?", func() {
			fd.accept(p)
		})
	case !fd.opts.save && !exists && fd.opts.mustExist:
		fd.alert(false, fd.files.Base(p)+"\nFile not found.\nPlease verify the correct file name was given.", nil)
	default:
		fd.accept(p)
	}
}

// alert shows a warning over the dialog; yes runs if the player answers
// Yes.
func (fd *fileDialog) alert(yesNo bool, text string, yes func()) {
	d := fd.app.desk
	md := messageDialog(d, game.Warning, yesNo, text)
	md.caption = fd.window.caption
	md.onModal = func(result int) {
		d.remove(md)

		if result == mrYes && yes != nil {
			yes()
		}
	}
	d.showModal(md)
}

func (fd *fileDialog) accept(p string) {
	fd.result = p
	fd.app.lastDir = fd.files.Dir(p)
	fd.window.endModal(mrOk)
}

// The toolbar's glyphs: an arrow out of a tray, and into one.
const (
	glyphUpload string = `
......k......
.....kkk.....
....kkkkk....
...kkkkkkk...
.....kkk.....
.....kkk.....
.....kkk.....
k....kkk....k
k...........k
kkkkkkkkkkkkk
`
	glyphDownload string = `
.....kkk.....
.....kkk.....
.....kkk.....
.....kkk.....
...kkkkkkk...
....kkkkk....
.....kkk.....
k.....k.....k
k...........k
kkkkkkkkkkkkk
`
)

// addTransfers adds the Upload and Download buttons after Up One Level.
func (fd *fileDialog) addTransfers() {
	fd.uploadButton = &toolButton{
		glyph:   glyphUpload,
		onPress: fd.upload,
	}
	addWidget(fd.window, fd.uploadButton, image.Rect(304, 4, 327, 26))

	fd.downloadButton = &toolButton{
		glyph:   glyphDownload,
		onPress: fd.download,
	}
	addWidget(fd.window, fd.downloadButton, image.Rect(328, 4, 351, 26))
}

// upload asks for files to put in the folder shown.
func (fd *fileDialog) upload() {
	fd.uploads = fd.app.xfer.upload(fd.dir)
}

// poll shows the files uploaded, once they are.
func (fd *fileDialog) poll() {
	if fd.uploads == nil {
		return
	}

	select {
	case names := <-fd.uploads:
		fd.uploads = nil
		fd.uploaded(names)
	default:
	}
}

// uploaded lists the folder again, with the first file uploaded chosen.
func (fd *fileDialog) uploaded(names []string) {
	fd.list()

	if len(names) == 0 {
		return
	}

	if i := slices.IndexFunc(fd.entries, func(e platform.Entry) bool {
		return e.Name == names[0]
	}); i >= 0 {
		fd.view.selected = i
		fd.view.ensureVisible(i)
	}

	fd.name.setText(names[0])

	fd.app.desk.dirty = true
}

// download gives the player the file chosen in the view, or named in the
// name box.
func (fd *fileDialog) download() {
	name := strings.TrimSpace(fd.name.text())
	if e, ok := fd.selectedEntry(); ok && !e.Dir {
		name = e.Name
	}

	if name == "" {
		return
	}

	p := name
	if !fd.files.IsAbs(p) {
		p = fd.files.Join(fd.dir, p)
	}

	b, err := fd.files.ReadFile(p)
	if err == nil {
		err = fd.app.xfer.download(fd.files.Base(p), b)
	}

	if err != nil {
		fd.alert(false, fd.files.Base(p)+"\n"+err.Error(), nil)
	}
}

// toolButton is a flat toolbar button: its glyph alone, with a thin
// raised edge under the mouse and a sunken one while pressed.
type toolButton struct {
	control
	glyph   string
	hovered bool
	down    bool
	inside  bool
	onPress func()
}

func (tb *toolButton) paint(cv canvas) {
	rect := tb.local()
	cv.fill(rect, colFace)

	off := 0

	switch {
	case tb.down && tb.inside:
		cv.edge(rect, colShadow, colHighlight)

		off = 1
	case tb.hovered || tb.down:
		cv.edge(rect, colHighlight, colShadow)
	}

	rows := strings.Split(strings.Trim(tb.glyph, "\n"), "\n")
	gx := (rect.Dx()-len(rows[0]))/2 + off
	gy := (rect.Dy()-len(rows))/2 + off
	cv.pattern(gx, gy, tb.glyph, iconColors)
}

func (tb *toolButton) mouseDown(image.Point, bool) {
	tb.down, tb.inside = true, true
	tb.invalidate()
}

func (tb *toolButton) mouseDrag(p image.Point) {
	if in := p.In(tb.local()); in != tb.inside {
		tb.inside = in
		tb.invalidate()
	}
}

func (tb *toolButton) mouseUp(imgP image.Point) {
	click := tb.down && imgP.In(tb.local())
	tb.down = false
	tb.invalidate()

	if click && tb.onPress != nil {
		tb.onPress()
	}
}

func (tb *toolButton) hover(_ image.Point, out bool) {
	if tb.hovered != !out {
		tb.hovered = !out
		tb.invalidate()
	}
}

// addWidget adds a control the forms' layouts don't describe.
func addWidget(window *window, wgt widget, rect image.Rectangle) {
	ctr := wgt.base()
	ctr.self = wgt
	ctr.wdw = window
	ctr.rect = rect
	ctr.visible, ctr.enabled = true, true
	ctr.font, ctr.fontColor, ctr.color = window.res.sans, colText, window.color
	window.kids = append(window.kids, wgt)
}
