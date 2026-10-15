// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// ancestry returns dir's ancestors from its root down, then dir.
func ancestry(dir string) []string {
	var chain []string

	for pth := dir; ; pth = filepath.Dir(pth) {
		chain = append(chain, pth)
		if filepath.Dir(pth) == pth {
			break
		}
	}

	slices.Reverse(chain)

	return chain
}

// The look-in list shows the roots with a drive's icon and the folder's
// ancestors under its root, each with a folder's icon and indented under
// its parent; choosing one opens it.
func TestLookIn(t *testing.T) {
	ta := newTestApp(t)
	dir := filepath.Join(ta.dir, "saves", "old")

	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	ta.lastDir = dir
	ta.wait(80)
	ta.clickOn(ta.start.wdw, "LoadButton")
	ta.wait(1)

	fd := ta.fileDialog
	if fd == nil {
		t.Fatal("no file dialog")
	}

	cb := fd.lookIn
	chain := ancestry(dir)

	if len(cb.items) != len(chain) || cb.index != len(chain)-1 {
		t.Fatalf("look in %q, item %d", cb.items, cb.index)
	}

	for i, pth := range chain {
		name, icon := filepath.Base(pth), fileIconFolder
		if i == 0 {
			name, icon = pth, fileIconDrive
		}

		if cb.items[i] != name || cb.icons[i] != icon || cb.indents[i] != i*lookInIndent {
			t.Errorf("item %d: %q, icon %d, indent %d", i, cb.items[i], cb.icons[i], cb.indents[i])
		}
	}

	rect := cb.button()
	ta.click(widgetOrigin(cb).Add(rect.Min).Add(rect.Size().Div(2)))

	dl := cb.dropped
	if dl == nil {
		t.Fatal("the list didn't drop down")
	}

	ta.shot("52-look-in")

	i := len(chain) - 2 // saves
	ta.click(dl.rect.Min.Add(image.Pt(5, 1+(i-dl.top)*cb.itemHeight+cb.itemHeight/2)))

	if want := filepath.Join(ta.dir, "saves"); fd.dir != want {
		t.Errorf("in %s, want %s", fd.dir, want)
	}

	if cb.index != i || cb.items[i] != "saves" {
		t.Errorf("look in %q, item %d", cb.items, cb.index)
	}
}
