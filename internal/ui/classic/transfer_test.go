// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"testing"
	"testing/fstest"

	"github.com/jsperandio/pokemon-simulator/internal/platform"
)

// fakeTransfers stands in for the browser's file picker and downloads.
type fakeTransfers struct {
	files      platform.Files
	pick       map[string]string // what the player picks: name → content
	uploadedTo string
	downloads  map[string]string
}

func (ft *fakeTransfers) upload(dir string) <-chan []string {
	ft.uploadedTo = dir
	out := make(chan []string, 1)

	var names []string

	for n, data := range ft.pick {
		if ft.files.WriteFile(ft.files.Join(dir, n), []byte(data)) == nil {
			names = append(names, n)
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
	ta.clickOn(ta.start.w, "LoadButton")
	ta.wait(1)

	f := ta.fileDialog
	if f == nil || f.uploadButton == nil {
		t.Fatal("no upload button")
	}

	ta.shot("50-transfers")
	ta.click(widgetOrigin(f.uploadButton).Add(image.Pt(10, 10)))
	ta.wait(1)

	if fx.uploadedTo != f.dir || f.name.text() != "ash.pgs" || f.view.selected < 0 {
		t.Fatalf("uploaded to %q, name %q, selected %d", fx.uploadedTo, f.name.text(), f.view.selected)
	}

	ta.click(widgetOrigin(f.downloadButton).Add(image.Pt(10, 10)))

	if fx.downloads["ash.pgs"] != "save" {
		t.Errorf("downloads %q", fx.downloads)
	}

	ta.frame(input{dropped: fstest.MapFS{
		"song.ogg": {Data: []byte("OggS")},
		"sub/x":    {Data: []byte("x")},
	}})

	if f.name.text() != "song.ogg" {
		t.Errorf("after the drop, the name box has %q", f.name.text())
	}

	if b, err := ta.files.ReadFile(ta.files.Join(f.dir, "song.ogg")); err != nil || string(b) != "OggS" {
		t.Errorf("dropped file: %q, %v", b, err)
	}

	ta.shot("51-dropped")
}
