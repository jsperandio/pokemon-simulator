// SPDX-License-Identifier: AGPL-3.0-only

// Package platform is what the game needs from the system it runs on:
// where its files live, and showing a page in the browser.
package platform

import (
	"errors"
	"time"
)

// ErrNotExist is returned for a path that doesn't exist.
var ErrNotExist = errors.New("file does not exist")

// Files is the storage the game's files live in: the disk, or the
// browser's storage on the web. Paths are the storage's own.
type Files interface {
	// Home is the folder the file dialogs show first.
	Home() string
	// Roots lists the top-level folders: the drives on Windows.
	Roots() []string
	// List returns a folder's entries, in no particular order.
	List(dir string) ([]Entry, error)
	Stat(path string) (Entry, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	Mkdir(path string) error

	Join(dir, name string) string
	Dir(path string) string
	// Base returns the last element of a path.
	Base(path string) string
	IsAbs(path string) bool
}

// Entry is a file or folder.
type Entry struct {
	Name    string
	Dir     bool
	Size    int64
	ModTime time.Time
}
