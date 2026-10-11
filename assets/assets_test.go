// SPDX-License-Identifier: AGPL-3.0-only

package assets

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"
)

// mdLink matches Markdown links and images: [text](target), ![](target).
var mdLink = regexp.MustCompile(`\]\(([^)]+)\)`)

// Every page of the help manual is reachable from its index, and every
// local link and image points at an embedded file.
func TestHelpLinks(t *testing.T) {
	pages, err := fs.Glob(FS, "help/*.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 14 {
		t.Errorf("got %d help pages, want 14", len(pages))
	}
	linked := map[string]bool{"help/index.md": true}
	for _, page := range pages {
		md, err := fs.ReadFile(FS, page)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(md), "# ") {
			t.Errorf("%s does not start with a title", page)
		}
		for _, m := range mdLink.FindAllStringSubmatch(string(md), -1) {
			target := m[1]
			if strings.Contains(target, ":") {
				continue // mail and web links
			}
			file := path.Join("help", target)
			if _, err := fs.Stat(FS, file); err != nil {
				t.Errorf("%s links to missing %s", page, target)
			}
			linked[file] = true
		}
	}
	for _, page := range pages {
		if !linked[page] {
			t.Errorf("no page links to %s", page)
		}
	}
}
