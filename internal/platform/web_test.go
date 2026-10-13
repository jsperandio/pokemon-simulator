// SPDX-License-Identifier: AGPL-3.0-only

//go:build js

package platform

import (
	"errors"
	"slices"
	"testing"
	"time"
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

	var names []string
	entries, _ := m.List("/")

	for _, e := range entries {
		names = append(names, e.Name)
	}

	slices.Sort(names)

	if !slices.Equal(names, []string{"saves", "song.ogg"}) {
		t.Errorf("listed %q", names)
	}

	// The database's records come back with the folders they are in.
	r := newMemory()
	when := time.UnixMilli(1_000_000)
	r.restore("/a/b/c.pgs", false, when, []byte("c"))
	r.restore("/empty", true, when, nil)
	r.restore("relative", false, when, []byte("?"))

	for _, p := range []string{"/a", "/a/b", "/empty"} {
		if e, err := r.Stat(p); err != nil || !e.Dir {
			t.Errorf("%s: %+v, %v", p, e, err)
		}
	}

	if e, err := r.Stat("/a/b/c.pgs"); err != nil || e.Size != 1 || !e.ModTime.Equal(when) {
		t.Errorf("restored file: %+v, %v", e, err)
	}

	if len(r.files) != 1 {
		t.Errorf("files %v", r.files)
	}
}
