// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The look-in list shows the roots with a drive's icon and the folder's
// ancestors under its root, each with a folder's icon and indented under
// its parent; choosing one opens it.
func TestLookIn(t *testing.T) {
	ta := newTestApp(t)
	dir := filepath.Join(ta.dir, "saves", "old")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ta.lastDir = dir
	ta.wait(80)
	ta.clickOn(ta.start.w, "LoadButton")
	ta.wait(1)
	f := ta.fileDlg
	if f == nil {
		t.Fatal("no file dialog")
	}
	c := f.lookIn
	var chain []string
	for d := dir; ; d = filepath.Dir(d) {
		chain = append(chain, d)
		if filepath.Dir(d) == d {
			break
		}
	}
	slices.Reverse(chain)
	if len(c.items) != len(chain) || c.index != len(chain)-1 {
		t.Fatalf("look in %q, item %d", c.items, c.index)
	}
	for i, d := range chain {
		name, icon := filepath.Base(d), fileIconFolder
		if i == 0 {
			name, icon = d, fileIconDrive
		}
		if c.items[i] != name || c.icons[i] != icon || c.indents[i] != i*lookInIndent {
			t.Errorf("item %d: %q, icon %d, indent %d", i, c.items[i], c.icons[i], c.indents[i])
		}
	}

	b := c.button()
	ta.click(widgetOrigin(c).Add(b.Min).Add(b.Size().Div(2)))
	l := c.dropped
	if l == nil {
		t.Fatal("the list didn't drop down")
	}
	ta.shot("52-look-in")
	i := len(chain) - 2 // saves
	ta.click(l.r.Min.Add(image.Pt(5, 1+(i-l.top)*c.itemHeight+c.itemHeight/2)))
	if want := filepath.Join(ta.dir, "saves"); f.dir != want {
		t.Errorf("in %s, want %s", f.dir, want)
	}
	if c.index != i || c.items[i] != "saves" {
		t.Errorf("look in %q, item %d", c.items, c.index)
	}
}
