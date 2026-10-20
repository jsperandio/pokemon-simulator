// SPDX-License-Identifier: AGPL-3.0-only

//go:build js

package platform

import (
	"errors"
	"slices"
	"testing"
)

func TestMemory(t *testing.T) {
	m := newMemory()
	if err := m.WriteFile("/saves/ash.pgs", []byte("x")); !errors.Is(err, ErrNotExist) {
		t.Errorf("a file in a missing folder: %v", err)
	}

	if err := m.Mkdir("/saves"); err != nil {
		t.Fatal(err)
	}

	if err := m.WriteFile("/saves/ash.pgs", []byte("save")); err != nil {
		t.Fatal(err)
	}

	if err := m.WriteFile("/song.ogg", []byte("OggS")); err != nil {
		t.Fatal(err)
	}

	b, err := m.ReadFile("/saves/../saves/ash.pgs")
	if err != nil || string(b) != "save" {
		t.Errorf("read %q, %v", b, err)
	}

	if e, err := m.Stat("/saves"); err != nil || !e.Dir {
		t.Errorf("stat the folder: %+v, %v", e, err)
	}

	entries, _ := m.List("/")
	names := make([]string, 0, len(entries))

	for _, e := range entries {
		names = append(names, e.Name)
	}

	slices.Sort(names)

	if !slices.Equal(names, []string{"saves", "song.ogg"}) {
		t.Errorf("listed %q", names)
	}
}

// The database's records come back with the folders they are in.
func TestMemoryRestore(t *testing.T) {
	r := newMemory()
	r.restore("/a/b/c.pgs", false, []byte("c"))
	r.restore("/empty", true, nil)
	r.restore("relative", false, []byte("?"))

	for _, p := range []string{"/a", "/a/b", "/empty"} {
		if e, err := r.Stat(p); err != nil || !e.Dir {
			t.Errorf("%s: %+v, %v", p, e, err)
		}
	}

	if e, err := r.Stat("/a/b/c.pgs"); err != nil || e.Dir {
		t.Errorf("restored file: %+v, %v", e, err)
	}

	if len(r.files) != 1 {
		t.Errorf("files %v", r.files)
	}
}
