// SPDX-License-Identifier: AGPL-3.0-only

//go:build js

package platform

import (
	"path"
	"strings"
	"sync"
	"time"
)

// memory keeps the files in memory, under "/".
type memory struct {
	mu    sync.Mutex
	files map[string][]byte    // by clean path
	dirs  map[string]time.Time // the folders, "/" included
	times map[string]time.Time
}

var store = &memory{
	files: map[string][]byte{},
	dirs:  map[string]time.Time{"/": time.Now()},
	times: map[string]time.Time{},
}

// Default returns the platform's storage: here, the browser's.
func Default() Files { return store }

func (m *memory) Home() string    { return "/" }
func (m *memory) Roots() []string { return []string{"/"} }

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
			out = append(out, Entry{Name: path.Base(p), Dir: true, ModTime: t})
		}
	}
	for p, b := range m.files {
		if path.Dir(p) == dir {
			out = append(out, Entry{Name: path.Base(p), Size: int64(len(b)), ModTime: m.times[p]})
		}
	}
	return out, nil
}

func (m *memory) Stat(p string) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p = path.Clean(p)
	if t, ok := m.dirs[p]; ok {
		return Entry{Name: path.Base(p), Dir: true, ModTime: t}, nil
	}
	if b, ok := m.files[p]; ok {
		return Entry{Name: path.Base(p), Size: int64(len(b)), ModTime: m.times[p]}, nil
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

func (m *memory) WriteFile(p string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p = path.Clean(p)
	if _, ok := m.dirs[path.Dir(p)]; !ok {
		return ErrNotExist
	}
	m.files[p] = append([]byte(nil), data...)
	m.times[p] = time.Now()
	return nil
}

func (m *memory) Mkdir(p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p = path.Clean(p)
	if _, ok := m.dirs[path.Dir(p)]; !ok {
		return ErrNotExist
	}
	m.dirs[p] = time.Now()
	return nil
}

func (m *memory) Join(dir, name string) string { return path.Join(dir, name) }
func (m *memory) Dir(p string) string          { return path.Dir(p) }
func (m *memory) Base(p string) string         { return path.Base(p) }
func (m *memory) IsAbs(p string) bool          { return strings.HasPrefix(p, "/") }

// OpenPage shows a page in a new tab.
func OpenPage(_ string, page []byte) error { return openTab(page) }

// OpenURL opens a web address in a new tab; a mailto: one opens the
// browser's e-mail program.
func OpenURL(url string) error { return openURL(url) }

// Args returns no arguments: a page has no command line.
func Args() []string { return nil }
