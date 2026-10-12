// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"io/fs"
	"slices"
	"strings"

	"github.com/jsperandio/pokemon-simulator/internal/platform"
)

// On the web, the game's files live in the browser. The file dialogs get
// two more toolbar buttons there: Upload copies files from the player's
// computer into the folder the dialog shows, and Download gives the player
// a copy of the file chosen. Files dropped on the game are uploaded too.

// transfers moves files between the game's storage and the player's
// computer.
type transfers interface {
	// upload copies the files the player picks into dir; their names
	// come on the channel once they are there.
	upload(dir string) <-chan []string
	download(name string, data []byte) error
}

// browser is the browser's uploads and downloads.
type browser struct{ files platform.Files }

func (b browser) upload(dir string) <-chan []string     { return platform.Upload(b.files, dir) }
func (browser) download(name string, data []byte) error { return platform.Download(name, data) }

// The toolbar's glyphs: an arrow out of a tray, and into one.
const (
	glyphUpload = `
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
	glyphDownload = `
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
func (f *fileDialog) addTransfers() {
	f.uploadButton = &toolButton{glyph: glyphUpload, onPress: f.upload}
	addWidget(f.w, f.uploadButton, image.Rect(304, 4, 327, 26))
	f.downloadButton = &toolButton{glyph: glyphDownload, onPress: f.download}
	addWidget(f.w, f.downloadButton, image.Rect(328, 4, 351, 26))
}

// upload asks for files to put in the folder shown.
func (f *fileDialog) upload() {
	f.uploads = f.a.xfer.upload(f.dir)
}

// poll shows the files uploaded, once they are.
func (f *fileDialog) poll() {
	if f.uploads == nil {
		return
	}
	select {
	case names := <-f.uploads:
		f.uploads = nil
		f.uploaded(names)
	default:
	}
}

// uploaded lists the folder again, with the first file uploaded chosen.
func (f *fileDialog) uploaded(names []string) {
	f.list()
	if len(names) == 0 {
		return
	}
	if i := slices.IndexFunc(f.entries, func(e platform.Entry) bool { return e.Name == names[0] }); i >= 0 {
		f.view.selected = i
		f.view.ensureVisible(i)
	}
	f.name.setText(names[0])
	f.a.desk.dirty = true
}

// download gives the player the file chosen in the view, or named in the
// name box.
func (f *fileDialog) download() {
	name := strings.TrimSpace(f.name.text())
	if e, ok := f.selectedEntry(); ok && !e.Dir {
		name = e.Name
	}
	if name == "" {
		return
	}
	p := name
	if !f.files.IsAbs(p) {
		p = f.files.Join(f.dir, p)
	}
	b, err := f.files.ReadFile(p)
	if err == nil {
		err = f.a.xfer.download(f.files.Base(p), b)
	}
	if err != nil {
		f.alert(false, f.files.Base(p)+"\n"+err.Error(), nil)
	}
}

// importDropped uploads the files dropped on the game into the folder the
// file dialog shows, or else the home folder.
func (a *App) importDropped(fsys fs.FS) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return
	}
	dir := a.files.Home()
	if a.fileDlg != nil {
		dir = a.fileDlg.dir
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := fs.ReadFile(fsys, e.Name())
		if err == nil {
			err = a.files.WriteFile(a.files.Join(dir, e.Name()), b)
		}
		if err == nil {
			names = append(names, e.Name())
		}
	}
	if a.fileDlg != nil && len(names) > 0 {
		a.fileDlg.uploaded(names)
	}
}
