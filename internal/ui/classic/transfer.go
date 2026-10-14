// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
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
type browser struct {
	files platform.Files
}

func (b browser) upload(dir string) <-chan []string {
	return platform.Upload(b.files, dir)
}

func (b browser) download(name string, data []byte) error {
	return platform.Download(name, data)
}
