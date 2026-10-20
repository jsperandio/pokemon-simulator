// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
func Test_fileDialog(t *testing.T) {
	t.Parallel()

	t.Run("when the folder is under the saves folder, should list its ancestors from the root down, each indented under its parent", func(t *testing.T) {
		t.Parallel()

		ta, dir := newLoadDialog(t)
		cb := ta.fileDialog.lookIn
		chain := ancestry(dir)

		require.Len(t, cb.items, len(chain), "look in %q", cb.items)
		require.Equal(t, len(chain)-1, cb.index, "look in %q, item %d", cb.items, cb.index)

		for i, pth := range chain {
			nam, ico := filepath.Base(pth), fileIconFolder
			if i == 0 {
				nam, ico = pth, fileIconDrive
			}

			assert.Equal(t, nam, cb.items[i], "item %d", i)
			assert.Equal(t, ico, cb.icons[i], "the icon of item %d", i)
			assert.Equal(t, i*lookInIndent, cb.indents[i], "the indent of item %d", i)
		}
	})

	t.Run("when a folder in the look-in list is chosen, should open it", func(t *testing.T) {
		t.Parallel()

		ta, dir := newLoadDialog(t)
		fd := ta.fileDialog
		cb := fd.lookIn
		chain := ancestry(dir)

		rect := cb.button()
		ta.click(widgetOrigin(cb).Add(rect.Min).Add(rect.Size().Div(2)))

		dl := cb.dropped
		require.NotNil(t, dl, "the list didn't drop down")

		ta.shot("52-look-in")

		i := len(chain) - 2 // saves
		ta.click(dl.rect.Min.Add(image.Pt(5, 1+(i-dl.top)*cb.itemHeight+cb.itemHeight/2)))

		assert.Equal(t, filepath.Join(ta.dir, "saves"), fd.dir, "the folder opened")
		assert.Equal(t, i, cb.index, "the item chosen")
		assert.Equal(t, "saves", cb.items[i], "the item's name")
	})
}

// newLoadDialog starts the program with the Load dialog open in the folder
// saves/old of its test folder, and returns that folder too.
func newLoadDialog(t *testing.T) (ta *testApp, dir string) {
	t.Helper()

	ta = newTestApp(t)
	dir = filepath.Join(ta.dir, "saves", "old")

	require.NoError(t, os.MkdirAll(dir, 0o750))

	ta.lastDir = dir
	ta.wait(80)
	ta.clickOn(ta.start.wdw, "LoadButton")
	ta.wait(1)

	require.NotNil(t, ta.fileDialog, "no file dialog")

	return ta, dir
}
