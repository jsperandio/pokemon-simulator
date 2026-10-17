// SPDX-License-Identifier: AGPL-3.0-only

//go:build js

package platform

import (
	"path"
	"strings"
	"sync"
	"time"
)

// memory keeps the files in memory, under "/", and in the browser's
// IndexedDB, where they outlast the page: they are read from it when the
// game starts, and each change is written to it at once.
type memory struct {
	mu    sync.Mutex
	files map[string][]byte    // by clean path
	dirs  map[string]time.Time // the folders, "/" included
	times map[string]time.Time
	db    *database // nil when the browser keeps nothing
}

func newMemory() *memory {
	return &memory{
		files: map[string][]byte{},
		dirs:  map[string]time.Time{"/": time.Now()},
		times: map[string]time.Time{},
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
	if _, ok := m.dirs[dir]; !ok {
		return nil, ErrNotExist
	}

	var out []Entry

	for p, t := range m.dirs {
		if p != "/" && path.Dir(p) == dir {
			out = append(out, Entry{
				Name:    path.Base(p),
				Dir:     true,
				ModTime: t,
			})
		}
	}

	for p, b := range m.files {
		if path.Dir(p) == dir {
			out = append(out, Entry{
				Name:    path.Base(p),
				Size:    int64(len(b)),
				ModTime: m.times[p],
			})
		}
	}

	return out, nil
}

func (m *memory) Stat(pth string) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pth = path.Clean(pth)
	if t, ok := m.dirs[pth]; ok {
		return Entry{
			Name:    path.Base(pth),
			Dir:     true,
			ModTime: t,
		}, nil
	}

	if b, ok := m.files[pth]; ok {
		return Entry{
			Name:    path.Base(pth),
			Size:    int64(len(b)),
			ModTime: m.times[pth],
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

	return append([]byte(nil), b...), nil
}

func (m *memory) WriteFile(pth string, data []byte) error {
	m.mu.Lock()

	pth = path.Clean(pth)
	if _, ok := m.dirs[path.Dir(pth)]; !ok {
		m.mu.Unlock()

		return ErrNotExist
	}

	now := time.Now()

	m.files[pth] = append([]byte(nil), data...)
	m.times[pth] = now
	db := m.db
	m.mu.Unlock()

	if db == nil {
		return nil
	}

	return db.put(pth, false, now, data)
}

func (m *memory) Mkdir(pth string) error {
	m.mu.Lock()

	pth = path.Clean(pth)
	if _, ok := m.dirs[path.Dir(pth)]; !ok {
		m.mu.Unlock()

		return ErrNotExist
	}

	now := time.Now()
	m.dirs[pth] = now
	db := m.db
	m.mu.Unlock()

	if db == nil {
		return nil
	}

	return db.put(pth, true, now, nil)
}

// restore adds a file or folder read from the database, and the folders
// it is in.
func (m *memory) restore(pth string, dir bool, t time.Time, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pth = path.Clean(pth)
	if !strings.HasPrefix(pth, "/") {
		return
	}

	for d := path.Dir(pth); d != "/"; d = path.Dir(d) {
		if _, ok := m.dirs[d]; !ok {
			m.dirs[d] = t
		}
	}

	if dir {
		m.dirs[pth] = t

		return
	}

	m.files[pth] = data
	m.times[pth] = t
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

// OpenPage shows a page in a new tab.
func OpenPage(_ string, page []byte) error {
	return openTab(page)
}

// OpenURL opens a web address in a new tab; a mailto: one opens the
// browser's e-mail program.
func OpenURL(url string) error {
	return openURL(url)
}

// Args returns no arguments: a page has no command line.
func Args() []string {
	return nil
}

// Transfers is true: the game's files live in the browser, and the player
// uploads and downloads them.
const Transfers bool = true
