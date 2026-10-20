// SPDX-License-Identifier: AGPL-3.0-only

//go:build js

package platform

import (
	"bytes"
	"path"
	"strings"
	"sync"
)

// memory keeps the files in memory, under "/", and in the browser's
// IndexedDB, where they outlast the page: they are read from it when the
// game starts, and each change is written to it at once.
type memory struct {
	mu    sync.Mutex
	files map[string][]byte // by clean path
	dirs  map[string]bool   // the folders, "/" included
	db    *database         // nil when the browser keeps nothing
}

func newMemory() *memory {
	return &memory{
		files: map[string][]byte{},
		dirs:  map[string]bool{"/": true},
	}
}

var (
	store     = newMemory()
	storeOnce sync.Once
)

// DefaultStorageProvider returns the platform's storage: here, the browser's. The first
// call reads the files the browser kept; without IndexedDB, the files
// last as long as the page.
func DefaultStorageProvider() *memory {
	storeOnce.Do(func() {
		db, err := openDatabase()
		if err == nil {
			err = db.load(store)
		}

		if err != nil {
			warn("POKéMON Simulator: the saved games won't outlast this page: " + err.Error())

			return
		}

		store.db = db
	})

	return store
}

func (m *memory) Home() string {
	return "/"
}

func (m *memory) Roots() []string {
	return []string{"/"}
}

func (m *memory) List(dir string) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir = path.Clean(dir)
	if !m.dirs[dir] {
		return nil, ErrNotExist
	}

	var out []Entry

	for p := range m.dirs {
		if p != "/" && path.Dir(p) == dir {
			out = append(out, Entry{
				Name: path.Base(p),
				Dir:  true,
			})
		}
	}

	for p := range m.files {
		if path.Dir(p) == dir {
			out = append(out, Entry{
				Name: path.Base(p),
			})
		}
	}

	return out, nil
}

func (m *memory) Stat(pth string) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pth = path.Clean(pth)
	if m.dirs[pth] {
		return Entry{
			Name: path.Base(pth),
			Dir:  true,
		}, nil
	}

	if _, ok := m.files[pth]; ok {
		return Entry{
			Name: path.Base(pth),
		}, nil
	}

	return Entry{}, ErrNotExist
}

func (m *memory) ReadFile(p string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.files[path.Clean(p)]
	if !ok {
		return nil, ErrNotExist
	}

	return bytes.Clone(b), nil
}

func (m *memory) WriteFile(pth string, data []byte) error {
	m.mu.Lock()

	pth = path.Clean(pth)
	if !m.dirs[path.Dir(pth)] {
		m.mu.Unlock()

		return ErrNotExist
	}

	m.files[pth] = bytes.Clone(data)
	db := m.db
	m.mu.Unlock()

	if db == nil {
		return nil
	}

	return db.put(pth, false, data)
}

func (m *memory) Mkdir(pth string) error {
	m.mu.Lock()

	pth = path.Clean(pth)
	if !m.dirs[path.Dir(pth)] {
		m.mu.Unlock()

		return ErrNotExist
	}

	m.dirs[pth] = true
	db := m.db
	m.mu.Unlock()

	if db == nil {
		return nil
	}

	return db.put(pth, true, nil)
}

// restore adds a file or folder read from the database, and the folders
// it is in.
func (m *memory) restore(pth string, dir bool, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pth = path.Clean(pth)
	if !strings.HasPrefix(pth, "/") {
		return
	}

	for d := path.Dir(pth); d != "/"; d = path.Dir(d) {
		m.dirs[d] = true
	}

	if dir {
		m.dirs[pth] = true

		return
	}

	m.files[pth] = data
}

func (m *memory) Join(dir, name string) string {
	return path.Join(dir, name)
}

func (m *memory) Dir(p string) string {
	return path.Dir(p)
}

func (m *memory) Base(p string) string {
	return path.Base(p)
}

func (m *memory) IsAbs(p string) bool {
	return strings.HasPrefix(p, "/")
}

// Args returns no arguments: a page has no command line.
func Args() []string {
	return nil
}

// Transfers is true: the game's files live in the browser, and the player
// uploads and downloads them.
const Transfers bool = true
