// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
// dropped on the game are uploaded too.
func Test_Transfers(t *testing.T) {
	t.Parallel()

	// One subtest with checkpoints: the dropped file's screenshot shows the
	// uploaded file in the list, and the Download button under the mouse.
	t.Run("when a file is uploaded and downloaded, and files are dropped on the game, should copy them into the dialog's folder and out to the browser", func(t *testing.T) {
		t.Parallel()

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
		require.NotNil(t, fd, "no file dialog")
		require.NotNil(t, fd.uploadButton, "no upload button")

		ta.shot("50-transfers")
		ta.click(widgetOrigin(fd.uploadButton).Add(image.Pt(10, 10)))
		ta.wait(1)

		require.Equal(t, fd.dir, fx.uploadedTo, "the folder uploaded to")
		require.Equal(t, "ash.pgs", fd.name.text(), "the name box after the upload")
		require.GreaterOrEqual(t, fd.view.selected, 0, "the file selected after the upload")

		ta.click(widgetOrigin(fd.downloadButton).Add(image.Pt(10, 10)))

		assert.Equal(t, map[string]string{"ash.pgs": "save"}, fx.downloads, "the files downloaded")

		ta.frame(input{
			dropped: fstest.MapFS{
				"song.ogg": {Data: []byte("OggS")},
				"sub/x":    {Data: []byte("x")},
			},
		})

		assert.Equal(t, "song.ogg", fd.name.text(), "the name box after the drop")

		dat, err := ta.files.ReadFile(ta.files.Join(fd.dir, "song.ogg"))
		assert.NoError(t, err, "the dropped file")
		assert.Equal(t, "OggS", string(dat), "the dropped file's content")

		ta.shot("51-dropped")
	})
}
