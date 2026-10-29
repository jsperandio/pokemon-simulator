// SPDX-License-Identifier: AGPL-3.0-only

package assets

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mdLink matches Markdown links and images: [text](target), ![](target).
var mdLink = regexp.MustCompile(`\]\(([^)]+)\)`)

// Every page of the help manual is linked from another, starting at its
// index, and every local link and image points at an embedded file.
func Test_FS(t *testing.T) {
	t.Parallel()

	pages, err := fs.Glob(FS, "help/*.md")
	require.NoError(t, err)

	t.Run("when the help folder is listed, should hold the manual's 15 pages", func(t *testing.T) {
		t.Parallel()

		assert.Len(t, pages, 15)
	})

	t.Run("when a help page is read, should start with a title", func(t *testing.T) {
		t.Parallel()

		for _, page := range pages {
			assert.True(t, strings.HasPrefix(readPage(t, page), "# "), "%s does not start with a title", page)
		}
	})

	t.Run("when a help page links to a local file, should point at an embedded one", func(t *testing.T) {
		t.Parallel()

		for _, page := range pages {
			for _, target := range localLinks(readPage(t, page)) {
				_, err := fs.Stat(FS, path.Join("help", target))
				assert.NoError(t, err, "%s links to missing %s", page, target)
			}
		}
	})

	t.Run("when the pages' links are gathered, should leave no page without one", func(t *testing.T) {
		t.Parallel()

		linked := map[string]bool{
			"help/index.md": true,
		}

		for _, page := range pages {
			for _, target := range localLinks(readPage(t, page)) {
				linked[path.Join("help", target)] = true
			}
		}

		for _, page := range pages {
			assert.True(t, linked[page], "no page links to %s", page)
		}
	})
}

// readPage returns the text of a help page.
func readPage(t *testing.T, page string) string {
	t.Helper()

	md, err := fs.ReadFile(FS, page)
	require.NoError(t, err)

	return string(md)
}

// localLinks returns the targets of md's links and images inside the
// manual: all but the mail and web links.
func localLinks(md string) []string {
	var targets []string

	for _, mat := range mdLink.FindAllStringSubmatch(md, -1) {
		if !strings.Contains(mat[1], ":") {
			targets = append(targets, mat[1])
		}
	}

	return targets
}
