// SPDX-License-Identifier: AGPL-3.0-only

// Package platform is what the game needs from the system it runs on:
// where its files live, and showing a page in the browser.
package platform

import "errors"

// ErrNotExist is returned for a path that doesn't exist.
var ErrNotExist = errors.New("file does not exist")

// writer is where Upload copies the files the player picks.
type writer interface {
	Join(dir, name string) string
	WriteFile(path string, data []byte) error
}

// Entry is a file or folder.
type Entry struct {
	Name string
	Dir  bool
}
