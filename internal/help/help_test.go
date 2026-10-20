// SPDX-License-Identifier: AGPL-3.0-only

package help

import (
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
)

// Test_Parse parses the sample page once, and the manual's pages once; the
// subtests only read them.
func Test_Parse(t *testing.T) {
	t.Parallel()

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
	sample := Parse([]byte(src))

	require.Len(t, sample.Blocks, 5)

	t.Run("when the sample has a heading, should title the page with it", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "Items", sample.Title)
	})

	t.Run("when the sample has a port note, should quote it with its bold and its link", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Block{
			Kind: Quote,
			Blocks: []Block{{Kind: Paragraph, Spans: []Span{
				{Text: "Port note:", Bold: true},
				{Text: " see the "},
				{Text: "tracker", Link: "https://example.org/issues"},
				{Text: "."},
			}}},
		}, sample.Blocks[0])
	})

	t.Run("when the sample has soft-wrapped lines, should join them and keep the hard breaks", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Block{
			Kind: Paragraph,
			Spans: []Span{
				{Text: "During battles, you can use items."},
				{Break: true},
				{Text: "Line two"},
				{Break: true},
				{Text: "three"},
			},
		}, sample.Blocks[1])
	})

	t.Run("when the sample has a bullet list with a continuation line, should join the line to its item", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Block{
			Kind: List,
			Items: []Item{
				{Spans: []Span{{Text: "POKéBalls"}}},
				{Spans: []Span{{Text: "Potions, which heal"}}},
			},
		}, sample.Blocks[2])
	})

	t.Run("when the sample has a numbered list with an image, should keep the markers and the image", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Block{
			Kind: List,
			Items: []Item{
				{Marker: "1)", Spans: []Span{{Text: "one "}, {Image: "bm3.png"}}},
				{Marker: "2)", Spans: []Span{{Text: "two"}}},
			},
		}, sample.Blocks[3])
	})

	t.Run("when the sample has a table with an empty fourth column, should keep the column", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Block{
			Kind: Table,
			Rows: [][]Cell{
				{{{Text: "Difficulty"}}, {{Text: "Fee"}}, {{Text: "Prizes"}}, nil},
				{{{Text: "Easy"}}, {{Text: "2000¥"}}, {{Text: "3 Potions"}}, nil},
			},
		}, sample.Blocks[4])
	})

	// The manual's pages each read with a title and blocks, and their links
	// and images lead to pages and files that exist, or out to the web or
	// e-mail. No markup is left in their text.
	pages := readManual(t)
	names := slices.Sorted(maps.Keys(pages))

	t.Run("when a page of the manual is read, should give it a title and blocks", func(t *testing.T) {
		t.Parallel()

		for _, name := range names {
			assert.NotEmpty(t, pages[name].Title, "%s has no title", name)
			assert.NotEmpty(t, pages[name].Blocks, "%s is empty", name)
		}
	})

	t.Run("when a page of the manual has links and images, should lead them to pages, files, the web or mail", func(t *testing.T) {
		t.Parallel()

		for _, name := range names {
			checkLinks(t, pages, name, spansOf(pages[name].Blocks))
		}
	})

	t.Run("when a page of the manual is read, should leave no markup in its text", func(t *testing.T) {
		t.Parallel()

		for _, name := range names {
			checkMarkup(t, name, spansOf(pages[name].Blocks))
		}
	})
}

// readManual parses the pages of the manual, by file name. It requires the
// index, the topic the original opens first.
func readManual(t *testing.T) map[string]*Page {
	t.Helper()

	names, err := fs.Glob(assets.FS, "help/*.md")
	require.NoError(t, err)

	pages := map[string]*Page{}

	for _, name := range names {
		src, err := assets.FS.ReadFile(name)
		require.NoError(t, err, "%s", name)

		pages[path.Base(name)] = Parse(src)
	}

	require.NotNil(t, pages[Contents], "no %s", Contents)

	return pages
}

// spansOf returns the spans of blocks: those of the blocks, those of the
// blocks inside them, and those of their items and cells.
func spansOf(bs []Block) []Span {
	var out []Span

	for _, b := range bs {
		out = append(out, b.Spans...)
		out = append(out, spansOf(b.Blocks)...)

		for _, it := range b.Items {
			out = append(out, it.Spans...)
		}

		for _, row := range b.Rows {
			for _, c := range row {
				out = append(out, c...)
			}
		}
	}

	return out
}

// checkLinks checks that the images and links among spans of a page of the
// manual lead somewhere: to a page of the manual, to an embedded file, or out
// to the web or mail.
func checkLinks(t *testing.T, pages map[string]*Page, page string, spans []Span) {
	t.Helper()

	for _, s := range spans {
		switch {
		case s.Image != "":
			_, err := fs.Stat(assets.FS, "help/"+s.Image)
			assert.NoError(t, err, "%s: image %s", page, s.Image)
		case s.Link == "":
		case strings.HasPrefix(s.Link, "http://"), strings.HasPrefix(s.Link, "https://"),
			strings.HasPrefix(s.Link, "mailto:"):
		default:
			assert.NotNil(t, pages[s.Link], "%s: link to %s", page, s.Link)
		}
	}
}

// checkMarkup checks that no markup is left in the text of spans on a page
// of the manual: no bold or link delimiters, and no escapes.
func checkMarkup(t *testing.T, page string, spans []Span) {
	t.Helper()

	for _, s := range spans {
		for _, mark := range []string{"**", "](", `\`} {
			assert.NotContains(t, s.Text, mark, "%s: %q left in %q", page, mark, s.Text)
		}
	}
}

func Test_parseInline(t *testing.T) {
	t.Parallel()

	for _, cas := range []struct {
		name string
		in   string
		want []Span
	}{
		{
			name: "when a backslash escapes a star, should keep the star as text",
			in:   `files (\*.mp3, \*.wav)`,
			want: []Span{
				{Text: "files (*.mp3, *.wav)"},
			},
		},
		{
			name: "when stars enclose text, should make it italic",
			in:   "the word codec (*pl. codices*) is",
			want: []Span{
				{Text: "the word codec ("},
				{Text: "pl. codices", Italic: true},
				{Text: ") is"},
			},
		},
		{
			name: "when bold is inside italic, should style both",
			in:   "*NOTE: **all** of it*",
			want: []Span{
				{Text: "NOTE: ", Italic: true},
				{Text: "all", Bold: true, Italic: true},
				{Text: " of it", Italic: true},
			},
		},
		{
			name: "when the stars don't pair up, should keep them as text",
			in:   "a * b ** c *",
			want: []Span{
				{Text: "a * b ** c *"},
			},
		},
		{
			name: "when bold text is inside a link, should keep the bold and the link",
			in:   "see [**Music**](music.md).",
			want: []Span{
				{Text: "see "},
				{Text: "Music", Bold: true, Link: "music.md"},
				{Text: "."},
			},
		},
		{
			name: "when a link goes to mail, should keep the address as its target",
			in:   "[e-mail me.](mailto:mickeym@go.to)",
			want: []Span{
				{Text: "e-mail me.", Link: "mailto:mickeym@go.to"},
			},
		},
		{
			name: "when brackets are not followed by a target, should keep them as text",
			in:   "[a] (b) and [c]",
			want: []Span{
				{Text: "[a] (b) and [c]"},
			},
		},
		{
			name: "when an image is written, should give its file and its description",
			in:   "x: ![a picture](bm4.png)",
			want: []Span{
				{Text: "x: "},
				{Text: "a picture", Image: "bm4.png"},
			},
		},
	} {
		t.Run(cas.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, cas.want, parseInline(cas.in), "parseInline(%q)", cas.in)
		})
	}
}
