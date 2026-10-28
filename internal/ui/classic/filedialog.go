// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
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
	save       bool
	filter     string // the VCL's Filter: "Name|*.a;*.b|Name|*.c"
	defaultExt string // added to a name without an extension
	overwrite  bool   // ofOverwritePrompt
	mustExist  bool   // ofFileMustExist
	fileName   string // FileName: the name box's first text, which Save As numbers if the folder has it
}

// fileFilter is one of a dialog's file types.
type fileFilter struct {
	name     string
	patterns []string
}

// parseFilter splits a VCL Filter into its file types.
func parseFilter(txt string) []fileFilter {
	parts := strings.Split(txt, "|")

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

	for _, pat := range ff.patterns {
		if pat == "*.*" || pat == "*" {
			return true
		}

		if ok, _ := path.Match(pat, name); ok {
			return true
		}
	}

	return false
}

// navigator is what a file dialog needs of the game's storage: to browse
// it, and to read the file chosen.
type navigator interface {
	Roots() []string
	List(dir string) ([]platform.Entry, error)
	Stat(path string) (platform.Entry, error)
	ReadFile(path string) ([]byte, error)
	Join(dir, name string) string
	Dir(path string) string
	Base(path string) string
	IsAbs(path string) bool
}

// fileDialog is an open file dialog.
type fileDialog struct {
	app     *Application
	wdw     *window
	opts    fileOptions
	files   navigator
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

func newFileDialog(app *Application, opts fileOptions) *fileDialog {
	fd := &fileDialog{
		app:     app,
		opts:    opts,
		files:   app.files,
		filters: parseFilter(opts.filter),
	}
	title, in, ok, typ := "Open", "Look &in:", "&Open", "Files of &type:"

	if opts.save {
		title, in, ok, typ = "Save As", "Save &in:", "&Save", "Save as &type:"
	}

	comp := layout.Component{
		Class:   "TFileDialog",
		Name:    "FileDialog",
		Caption: title,
		Props: map[string]string{
			"BorderStyle":  "bsDialog",
			"ClientWidth":  "420",
			"ClientHeight": "240",
			"Position":     "poScreenCenter",
		},
		Children: []layout.Component{
			{
				Class:   "TLabel",
				Name:    "LookInLabel",
				Caption: in,
				Bounds:  layout.Rect{X: 6, Y: 10, W: 50, H: 13},
			},
			{
				Class:  "TComboBox",
				Name:   "LookIn",
				Bounds: layout.Rect{X: 74, Y: 5, W: 198, H: 21},
				Props: map[string]string{
					"Style":      "csDropDownList",
					"ItemHeight": "16",
					"TabOrder":   "0",
				},
			},
			{
				Class:  "TListView",
				Name:   "View",
				Bounds: layout.Rect{X: 6, Y: 33, W: 408, H: 138},
				Props:  map[string]string{"TabOrder": "1"},
			},
			{
				Class:   "TLabel",
				Name:    "NameLabel",
				Caption: "File &name:",
				Bounds:  layout.Rect{X: 8, Y: 183, W: 50, H: 13},
			},
			{
				Class:  "TEdit",
				Name:   "FileName",
				Bounds: layout.Rect{X: 81, Y: 179, W: 233, H: 21},
				Props:  map[string]string{"TabOrder": "2"},
			},
			{
				Class:   "TLabel",
				Name:    "TypeLabel",
				Caption: typ,
				Bounds:  layout.Rect{X: 8, Y: 209, W: 70, H: 13},
			},
			{
				Class:  "TComboBox",
				Name:   "Types",
				Bounds: layout.Rect{X: 81, Y: 205, W: 233, H: 21},
				Props:  map[string]string{"Style": "csDropDownList", "TabOrder": "3"},
			},
			{
				Class:   "TButton",
				Name:    "OK",
				Caption: ok,
				Bounds:  layout.Rect{X: 333, Y: 179, W: 75, H: 23},
				Props:   map[string]string{"Default": "True", "TabOrder": "4"},
			},
			{
				Class:   "TButton",
				Name:    "Cancel",
				Caption: "Cancel",
				Bounds:  layout.Rect{X: 333, Y: 205, W: 75, H: 23},
				Props: map[string]string{
					"Cancel":      "True",
					"ModalResult": strconv.Itoa(mrCancel),
					"TabOrder":    "5",
				},
			},
		},
	}
	wdw := newWindow(app.desk, &comp)
	fd.wdw = wdw
	fd.lookIn = wdw.find("LookIn").(*comboBox)
	fd.view = wdw.find("View").(*listView)
	fd.name = wdw.find("FileName").(*edit)
	fd.types = wdw.find("Types").(*comboBox)
	wdw.find("LookInLabel").(*label).focusControl = fd.lookIn
	wdw.find("NameLabel").(*label).focusControl = fd.name
	wdw.find("TypeLabel").(*label).focusControl = fd.types
	fd.view.images = app.res.fileIcons()
	fd.lookIn.images = app.res.fileIcons()

	fd.up = &toolButton{
		glyph:   glyphUp,
		onPress: fd.upOneLevel,
	}
	addWidget(wdw, fd.up, image.Rect(280, 4, 303, 26))

	if app.xfer != nil {
		fd.addTransfers()
	}

	for _, ft := range fd.filters {
		fd.types.items = append(fd.types.items, ft.name)
	}

	fd.types.index = 0
	fd.types.onSelect = func(int) {
		fd.list()
	}
	fd.lookIn.onSelect = func(i int) {
		if i >= 0 && i < len(fd.dirs) {
			fd.navigate(fd.dirs[i])
		}
	}
	fd.view.onSelect = fd.viewClick
	fd.view.onDblClick = fd.open
	wdw.find("OK").(*button).onClick = fd.okClick
	wdw.onKey = func(key ebiten.Key, mds mods) bool {
		if key == ebiten.KeyBackspace && wdw.focus == fd.view {
			fd.upOneLevel()

			return true
		}

		return false
	}
	dir := app.lastDir

	if dir == "" {
		dir = app.files.Home()
	}

	fd.navigate(dir)

	if nam := opts.fileName; nam != "" {
		if opts.save {
			nam = fd.unusedName(nam)
		}

		fd.name.setText(nam)
	}

	wdw.focus = fd.name
	app.desk.add(wdw)

	return fd
}

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

	for cur := dir; ; cur = fd.files.Dir(cur) {
		chain = append(chain, cur)
		if fd.files.Dir(cur) == cur {
			break
		}
	}

	slices.Reverse(chain)

	var (
		items          []string
		icons, indents []int
	)
	add := func(pth string, depth int) {
		name, icon := fd.files.Base(pth), fileIconFolder
		if depth == 0 {
			name, icon = pth, fileIconDrive
		}

		fd.dirs = append(fd.dirs, pth)
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

		for depth, pth := range chain {
			add(pth, depth)
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

	for _, ent := range entries {
		switch {
		case strings.HasPrefix(ent.Name, "."):
		case ent.Dir:
			dirs = append(dirs, ent)
		case ft.matches(ent.Name):
			files = append(files, ent)
		}
	}

	byName := func(fst, snd platform.Entry) int {
		return strings.Compare(strings.ToLower(fst.Name), strings.ToLower(snd.Name))
	}
	slices.SortFunc(dirs, byName)
	slices.SortFunc(files, byName)

	fd.entries = slices.Concat(dirs, files)
	items := make([]listItem, len(fd.entries))

	for i, ent := range fd.entries {
		img := fileIconDocument
		if ent.Dir {
			img = fileIconFolder
		}

		items[i] = listItem{
			caption: ent.Name,
			image:   img,
		}
	}

	fd.view.setItems(items)

	fd.view.selected, fd.view.hot = -1, -1
	fd.view.scroll.pos = 0
	fd.view.laid = false
	fd.view.invalidate()
}

// unusedName is name, or, if a file or folder in the folder shown has it
// in any case, the first of name2, name3... (before its extension) that
// none has.
func (fd *fileDialog) unusedName(name string) string {
	entries, _ := fd.files.List(fd.dir)
	used := func(nam string) bool {
		return slices.ContainsFunc(entries, func(ent platform.Entry) bool {
			return strings.EqualFold(ent.Name, nam)
		})
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	nam := name

	for n := 2; used(nam); n++ {
		nam = stem + strconv.Itoa(n) + ext
	}

	return nam
}

func (fd *fileDialog) selectedEntry() (platform.Entry, bool) {
	if i := fd.view.selected; i >= 0 && i < len(fd.entries) {
		return fd.entries[i], true
	}

	return platform.Entry{}, false
}

// viewClick puts a selected file's name in the name box.
func (fd *fileDialog) viewClick() {
	if ent, ok := fd.selectedEntry(); ok && !ent.Dir {
		fd.name.setText(ent.Name)
	}
}

// open is a double click on the view: a folder opens, a file is chosen.
func (fd *fileDialog) open() {
	ent, ok := fd.selectedEntry()
	if !ok {
		return
	}

	if ent.Dir {
		fd.navigate(fd.files.Join(fd.dir, ent.Name))

		return
	}

	fd.name.setText(ent.Name)
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
	if fd.wdw.focus == fd.view || txt == "" {
		if ent, ok := fd.selectedEntry(); ok && ent.Dir {
			fd.navigate(fd.files.Join(fd.dir, ent.Name))

			return
		}
	}

	if txt == "" {
		return
	}

	pth := txt
	if !fd.files.IsAbs(pth) {
		pth = fd.files.Join(fd.dir, pth)
	}

	if st, err := fd.files.Stat(pth); err == nil && st.Dir {
		fd.navigate(pth)
		fd.name.setText("")

		return
	}

	fd.choose(fd.withDefaultExt(pth))
}

// withDefaultExt adds the default extension to a file name without one, when
// the dialog saves or a file has that name.
func (fd *fileDialog) withDefaultExt(pth string) string {
	if strings.Contains(fd.files.Base(pth), ".") || fd.opts.defaultExt == "" {
		return pth
	}

	withExt := pth + "." + strings.TrimPrefix(fd.opts.defaultExt, ".")
	if _, err := fd.files.Stat(withExt); fd.opts.save || err == nil {
		return withExt
	}

	return pth
}

// choose checks a file's path as the dialog's options ask, then chooses
// it.
func (fd *fileDialog) choose(pth string) {
	if st, err := fd.files.Stat(fd.files.Dir(pth)); err != nil || !st.Dir {
		fd.alert(false, pth+"\nPath does not exist.\nPlease verify the correct path was given.", nil)

		return
	}

	_, err := fd.files.Stat(pth)
	exists := err == nil

	switch {
	case fd.opts.save && exists && fd.opts.overwrite:
		fd.alert(true, pth+" already exists.\nDo you want to replace it?", func() {
			fd.accept(pth)
		})
	case !fd.opts.save && !exists && fd.opts.mustExist:
		fd.alert(false, fd.files.Base(pth)+"\nFile not found.\nPlease verify the correct file name was given.", nil)
	default:
		fd.accept(pth)
	}
}

// alert shows a warning over the dialog; yes runs if the player answers
// Yes.
func (fd *fileDialog) alert(yesNo bool, text string, yes func()) {
	dsk := fd.app.desk
	md := messageDialog(dsk, game.Warning, yesNo, text, "")
	md.caption = fd.wdw.caption
	md.onModal = func(result int) {
		dsk.remove(md)

		if result == mrYes && yes != nil {
			yes()
		}
	}
	dsk.showModal(md)
}

func (fd *fileDialog) accept(pth string) {
	fd.result = pth
	fd.app.lastDir = fd.files.Dir(pth)
	fd.wdw.endModal(mrOk)
}

// The toolbar's glyphs: the folder one level up, painted with iconColors,
// and an arrow out of a tray, and into one.
const (
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
	addWidget(fd.wdw, fd.uploadButton, image.Rect(304, 4, 327, 26))

	fd.downloadButton = &toolButton{
		glyph:   glyphDownload,
		onPress: fd.download,
	}
	addWidget(fd.wdw, fd.downloadButton, image.Rect(328, 4, 351, 26))
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

	if i := slices.IndexFunc(fd.entries, func(ent platform.Entry) bool {
		return ent.Name == names[0]
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
	if ent, ok := fd.selectedEntry(); ok && !ent.Dir {
		name = ent.Name
	}

	if name == "" {
		return
	}

	pth := name
	if !fd.files.IsAbs(pth) {
		pth = fd.files.Join(fd.dir, pth)
	}

	dat, err := fd.files.ReadFile(pth)
	if err == nil {
		err = fd.app.xfer.download(fd.files.Base(pth), dat)
	}

	if err != nil {
		fd.alert(false, fd.files.Base(pth)+"\n"+err.Error(), nil)
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

func (tb *toolButton) mouseDrag(pnt image.Point) {
	if in := pnt.In(tb.local()); in != tb.inside {
		tb.inside = in
		tb.invalidate()
	}
}

func (tb *toolButton) mouseUp(pnt image.Point) {
	click := tb.down && pnt.In(tb.local())
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
func addWidget(wdw *window, wgt widget, rect image.Rectangle) {
	ctl := wgt.base()
	ctl.self = wgt
	ctl.wdw = wdw
	ctl.rect = rect
	ctl.visible, ctl.enabled = true, true
	ctl.font, ctl.fontColor, ctl.color = wdw.res.sans, colText, wdw.color
	wdw.kids = append(wdw.kids, wgt)
}
