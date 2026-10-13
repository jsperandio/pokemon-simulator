// SPDX-License-Identifier: AGPL-3.0-only

package help

import (
	"io/fs"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/jsperandio/pokemon-simulator/assets"
)

func TestParse(t *testing.T) {
	src := "# Items\r\n" +
		"\r\n" +
		"> **Port note:** see the [tracker](https://example.org/issues).\r\n" +
		"\r\n" +
		"During battles,\r\n" +
		"you can use items.\\\r\n" +
		"Line two  \r\n" +
		"three\r\n" +
		"\r\n" +
		"- POKéBalls\r\n" +
		"- Potions,\r\n" +
		"  which heal\r\n" +
		"\r\n" +
		"1) one ![](bm3.png)\r\n" +
		"2) two\r\n" +
		"\r\n" +
		"| Difficulty | Fee | Prizes |  |\r\n" +
		"| --- | --- | --- | --- |\r\n" +
		"| Easy | 2000¥ | 3 Potions |\r\n"
	p := Parse([]byte(src))

	if p.Title != "Items" {
		t.Errorf("title %q", p.Title)
	}

	want := []Block{
		{Kind: Quote, Blocks: []Block{{Kind: Paragraph, Spans: []Span{
			{Text: "Port note:", Bold: true},
			{Text: " see the "},
			{Text: "tracker", Link: "https://example.org/issues"},
			{Text: "."},
		}}}},
		{Kind: Paragraph, Spans: []Span{
			{Text: "During battles, you can use items."},
			{Break: true},
			{Text: "Line two"},
			{Break: true},
			{Text: "three"},
		}},
		{Kind: List, Items: []Item{
			{Spans: []Span{{Text: "POKéBalls"}}},
			{Spans: []Span{{Text: "Potions, which heal"}}},
		}},
		{Kind: List, Items: []Item{
			{Marker: "1)", Spans: []Span{{Text: "one "}, {Image: "bm3.png"}}},
			{Marker: "2)", Spans: []Span{{Text: "two"}}},
		}},
		{Kind: Table, Rows: [][]Cell{
			{{{Text: "Difficulty"}}, {{Text: "Fee"}}, {{Text: "Prizes"}}, nil},
			{{{Text: "Easy"}}, {{Text: "2000¥"}}, {{Text: "3 Potions"}}, nil},
		}},
	}
	if !reflect.DeepEqual(p.Blocks, want) {
		t.Errorf("blocks\n got %+v\nwant %+v", p.Blocks, want)
	}
}

func TestInline(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []Span
	}{
		{`files (\*.mp3, \*.wav)`, []Span{{Text: "files (*.mp3, *.wav)"}}},
		{"the word codec (*pl. codices*) is", []Span{
			{Text: "the word codec ("}, {Text: "pl. codices", Italic: true}, {Text: ") is"},
		}},
		{"*NOTE: **all** of it*", []Span{
			{Text: "NOTE: ", Italic: true}, {Text: "all", Bold: true, Italic: true}, {Text: " of it", Italic: true},
		}},
		{"a * b ** c *", []Span{{Text: "a * b ** c *"}}},
		{"see [**Music**](music.md).", []Span{
			{Text: "see "}, {Text: "Music", Bold: true, Link: "music.md"}, {Text: "."},
		}},
		{"[e-mail me.](mailto:mickeym@go.to)", []Span{{Text: "e-mail me.", Link: "mailto:mickeym@go.to"}}},
		{"[a] (b) and [c]", []Span{{Text: "[a] (b) and [c]"}}},
		{"x: ![a picture](bm4.png)", []Span{{Text: "x: "}, {Text: "a picture", Image: "bm4.png"}}},
	} {
		if got := parseInline(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseInline(%q)\n got %+v\nwant %+v", c.in, got, c.want)
		}
	}
}

// Every page of the manual reads, with a title, and its links and images
// lead to pages and files that exist, or out to the web or e-mail.
func TestManual(t *testing.T) {
	names, err := fs.Glob(assets.FS, "help/*.md")
	if err != nil {
		t.Fatal(err)
	}

	pages := map[string]*Page{}

	for _, n := range names {
		b, err := assets.FS.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}

		pages[path.Base(n)] = Parse(b)
	}

	if pages[Contents] == nil {
		t.Fatalf("no %s", Contents)
	}

	for name, p := range pages {
		if p.Title == "" {
			t.Errorf("%s has no title", name)
		}

		if len(p.Blocks) == 0 {
			t.Errorf("%s is empty", name)
		}

		checkBlocks(t, pages, name, p.Blocks)
	}
}

// checkBlocks checks the spans of blocks, and of the blocks, items and
// cells in them, on a page of the manual.
func checkBlocks(t *testing.T, pages map[string]*Page, page string, bs []Block) {
	t.Helper()

	for _, b := range bs {
		checkSpans(t, pages, page, b.Spans)
		checkBlocks(t, pages, page, b.Blocks)

		for _, it := range b.Items {
			checkSpans(t, pages, page, it.Spans)
		}

		for _, row := range b.Rows {
			for _, c := range row {
				checkSpans(t, pages, page, c)
			}
		}
	}
}

// checkSpans checks that spans' images and links lead somewhere, and that
// no markup is left in their text.
func checkSpans(t *testing.T, pages map[string]*Page, page string, spans []Span) {
	t.Helper()

	for _, s := range spans {
		switch {
		case s.Image != "":
			if _, err := fs.Stat(assets.FS, "help/"+s.Image); err != nil {
				t.Errorf("%s: image %s: %v", page, s.Image, err)
			}
		case s.Link == "":
		case strings.HasPrefix(s.Link, "http://"), strings.HasPrefix(s.Link, "https://"),
			strings.HasPrefix(s.Link, "mailto:"):
		case pages[s.Link] == nil:
			t.Errorf("%s: link to %s", page, s.Link)
		}

		if strings.Contains(s.Text, "**") || strings.Contains(s.Text, "](") || strings.Contains(s.Text, `\`) {
			t.Errorf("%s: markup left in %q", page, s.Text)
		}
	}
}
