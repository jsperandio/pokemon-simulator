// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"testing"
	"testing/fstest"
)

// fakeTransfers stands in for the browser's file picker and downloads.
type fakeTransfers struct {
	files      writer
	pick       map[string]string // what the player picks: name → content
	uploadedTo string
	downloads  map[string]string
}

func (ft *fakeTransfers) upload(dir string) <-chan []string {
	ft.uploadedTo = dir
	out := make(chan []string, 1)

	var names []string

	for nam, data := range ft.pick {
		if ft.files.WriteFile(ft.files.Join(dir, nam), []byte(data)) == nil {
			names = append(names, nam)
		}
	}

	out <- names

	return out
}

func (ft *fakeTransfers) download(name string, data []byte) error {
	ft.downloads[name] = string(data)

	return nil
}

// On the web, the file dialogs upload and download files, and files
// dropped on the game are uploaded.
func TestTransfers(t *testing.T) {
	ta := newTestApp(t)
	fx := &fakeTransfers{
		files:     ta.files,
		pick:      map[string]string{"ash.pgs": "save"},
		downloads: map[string]string{},
	}
	ta.xfer = fx
	ta.wait(80)
	ta.clickOn(ta.start.wdw, "LoadButton")
	ta.wait(1)

	fd := ta.fileDialog
	if fd == nil || fd.uploadButton == nil {
		t.Fatal("no upload button")
	}

	ta.shot("50-transfers")
	ta.click(widgetOrigin(fd.uploadButton).Add(image.Pt(10, 10)))
	ta.wait(1)

	if fx.uploadedTo != fd.dir || fd.name.text() != "ash.pgs" || fd.view.selected < 0 {
		t.Fatalf("uploaded to %q, name %q, selected %d", fx.uploadedTo, fd.name.text(), fd.view.selected)
	}

	ta.click(widgetOrigin(fd.downloadButton).Add(image.Pt(10, 10)))

	if fx.downloads["ash.pgs"] != "save" {
		t.Errorf("downloads %q", fx.downloads)
	}

	ta.frame(input{dropped: fstest.MapFS{
		"song.ogg": {Data: []byte("OggS")},
		"sub/x":    {Data: []byte("x")},
	}})

	if fd.name.text() != "song.ogg" {
		t.Errorf("after the drop, the name box has %q", fd.name.text())
	}

	if dat, err := ta.files.ReadFile(ta.files.Join(fd.dir, "song.ogg")); err != nil || string(dat) != "OggS" {
		t.Errorf("dropped file: %q, %v", dat, err)
	}

	ta.shot("51-dropped")
}
