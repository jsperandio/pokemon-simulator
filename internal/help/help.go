// SPDX-License-Identifier: AGPL-3.0-only

// Package help reads the game's manual: the Markdown pages of assets/help,
// converted from the original's WinHelp file. It understands the subset of
// Markdown those pages use, and gives a user interface their blocks of
// styled text to lay out.
package help

import "strings"

// Contents is the manual's contents page, the topic the original opens
// (Application.HelpJump("Index")).
const Contents string = "index.md"

// Page is a page of the manual.
type Page struct {
	Title  string // its "#" heading
	Blocks []Block
}

// Kind is a block's kind.
type Kind int

const (
	Paragraph Kind = iota
	Heading        // a heading below the title
	Quote          // a quoted block: the port's notes
	List
	Table
)

// Block is a paragraph, a heading, a quote, a list or a table.
type Block struct {
	Kind   Kind
	Level  int      // a heading's: 2 for "##"
	Spans  []Span   // a paragraph's or a heading's text
	Blocks []Block  // a quote's
	Items  []Item   // a list's
	Rows   [][]Cell // a table's, the header row first
}

// Item is a list item. Its marker is "" in a bulleted list, and as written
// ("1)", "2.") in a numbered one.
type Item struct {
	Marker string
	Spans  []Span
}

// Cell is a table cell's text.
type Cell []Span

// Span is a run of text in one style, an image, or a line break.
type Span struct {
	Text   string
	Bold   bool
	Italic bool
	Link   string // a link's target: a page ("menus.md") or a URL
	Image  string // an image's file, beside the pages; Text is its description
	Break  bool   // a hard line break
}

// Parse reads a page.
func Parse(src []byte) *Page {
	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	blocks := parseBlocks(lines)
	p := &Page{}

	if len(blocks) > 0 && blocks[0].Kind == Heading && blocks[0].Level == 1 {
		p.Title = PlainText(blocks[0].Spans)
		blocks = blocks[1:]
	}

	p.Blocks = blocks

	return p
}

// PlainText returns spans' text without its styles.
func PlainText(spans []Span) string {
	var sb strings.Builder

	for _, s := range spans {
		switch {
		case s.Break:
			sb.WriteByte('\n')
		case s.Image == "":
			sb.WriteString(s.Text)
		}
	}

	return sb.String()
}

// parseBlocks reads lines into blocks.
func parseBlocks(lines []string) []Block {
	var out []Block

	for i := 0; i < len(lines); {
		line := lines[i]
		switch {
		case isBlank(line):
			i++
		case headingLevel(line) > 0:
			n := headingLevel(line)
			text := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
			text = strings.TrimSpace(strings.TrimRight(text, "#"))
			out = append(out, Block{
				Kind:  Heading,
				Level: n,
				Spans: parseInline(text),
			})
			i++
		case isQuote(line):
			var inner []string
			for ; i < len(lines) && isQuote(lines[i]); i++ {
				inner = append(inner, unquote(lines[i]))
			}

			out = append(out, Block{
				Kind:   Quote,
				Blocks: parseBlocks(inner),
			})
		case isTable(lines, i):
			var b Block
			b, i = parseTable(lines, i)
			out = append(out, b)
		case listMarker(line) != "":
			var b Block
			b, i = parseList(lines, i)
			out = append(out, b)
		default:
			start := i
			for i++; i < len(lines) && !isBlank(lines[i]) && !startsBlock(lines[i]); i++ {
			}

			text := strings.Join(trimLines(lines[start:i]), "\n")
			out = append(out, Block{
				Kind:  Paragraph,
				Spans: parseInline(text),
			})
		}
	}

	return out
}

func isBlank(line string) bool {
	return strings.TrimSpace(line) == ""
}

// indent returns a line without the up to 3 spaces that may lead a block.
func indent(line string) (string, bool) {
	t := strings.TrimLeft(line, " ")

	return t, len(line)-len(t) <= 3
}

// headingLevel returns the level of a heading line, or 0.
func headingLevel(line string) int {
	t, ok := indent(line)
	if !ok {
		return 0
	}

	n := len(t) - len(strings.TrimLeft(t, "#"))
	if !headingMarker(n, t) {
		return 0
	}

	return n
}

// headingMarker reports whether the n "#"s that lead t make a heading's
// marker: one to six of them, then a space or the end of the line.
func headingMarker(n int, t string) bool {
	return n >= 1 && n <= 6 && (len(t) == n || t[n] == ' ')
}

func isQuote(line string) bool {
	t, ok := indent(line)

	return ok && strings.HasPrefix(t, ">")
}

// unquote removes a quote line's marker and the space after it.
func unquote(line string) string {
	t, _ := indent(line)
	t = t[1:]

	return strings.TrimPrefix(t, " ")
}

// listMarker returns a list item line's marker ("-", "1)"), or "".
func listMarker(line string) string {
	t, ok := indent(line)
	if !ok || t == "" {
		return ""
	}

	n := 0

	switch t[0] {
	case '-', '*', '+':
		n = 1
	default:
		for n < len(t) && n < 9 && isDigit(t[n]) {
			n++
		}

		if n == 0 || n == len(t) || !isOrderedDelim(t[n]) {
			return ""
		}

		n++
	}

	if n < len(t) && t[n] != ' ' {
		return ""
	}

	return t[:n]
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isOrderedDelim reports whether c ends an ordered list item's number.
func isOrderedDelim(c byte) bool {
	return c == '.' || c == ')'
}

// startsBlock reports whether a line interrupts a paragraph.
func startsBlock(line string) bool {
	return headingLevel(line) > 0 || isQuote(line) || listMarker(line) != ""
}

// trimLines removes the spaces that lead each line of a paragraph.
func trimLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimLeft(l, " ")
	}

	return out
}

// parseList reads a list from line i, and returns the line after it. A
// list goes on across blank lines while its items do, and an item takes
// the lines that follow it up to the next block.
func parseList(lines []string, i int) (blk Block, next int) {
	blk = Block{Kind: List}
	ordered := isOrdered(listMarker(lines[i]))

	for i < len(lines) {
		marker := listMarker(lines[i])
		if marker == "" || isOrdered(marker) != ordered {
			break
		}

		t, _ := indent(lines[i])
		text := []string{strings.TrimSpace(t[len(marker):])}

		for i++; i < len(lines) && !isBlank(lines[i]) && !startsBlock(lines[i]); i++ {
			text = append(text, strings.TrimLeft(lines[i], " "))
		}

		item := Item{Spans: parseInline(strings.Join(text, "\n"))}
		if ordered {
			item.Marker = marker
		}

		blk.Items = append(blk.Items, item)
		j := i

		for j < len(lines) && isBlank(lines[j]) {
			j++
		}

		if j == len(lines) || !startsItem(lines[j], ordered) {
			break
		}

		i = j
	}

	return blk, i
}

// startsItem reports whether line starts an item of a list that is
// ordered, or not, as ordered says.
func startsItem(line string, ordered bool) bool {
	marker := listMarker(line)

	return marker != "" && isOrdered(marker) == ordered
}

func isOrdered(marker string) bool {
	return marker != "" && marker[0] >= '0' && marker[0] <= '9'
}

// isTable reports whether a table starts at line i: a row of cells, then
// a row of dashes.
func isTable(lines []string, i int) bool {
	if i+1 >= len(lines) || !strings.Contains(lines[i], "|") {
		return false
	}

	cells := splitRow(lines[i+1])
	if len(cells) == 0 {
		return false
	}

	for _, c := range cells {
		c = strings.Trim(c, ":")
		if c == "" || strings.Trim(c, "-") != "" {
			return false
		}
	}

	return true
}

// parseTable reads a table from line i, and returns the line after it.
func parseTable(lines []string, i int) (blk Block, next int) {
	cols := len(splitRow(lines[i+1]))
	blk = Block{Kind: Table}
	row := func(line string) []Cell {
		cells := make([]Cell, cols)
		for k, c := range splitRow(line) {
			if k < cols {
				cells[k] = parseInline(c)
			}
		}

		return cells
	}
	blk.Rows = append(blk.Rows, row(lines[i]))

	for i += 2; i < len(lines) && !isBlank(lines[i]) && strings.Contains(lines[i], "|"); i++ {
		blk.Rows = append(blk.Rows, row(lines[i]))
	}

	return blk, i
}

// splitRow splits a table row into its cells' texts, at the bars that
// aren't escaped.
func splitRow(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")

	if strings.HasSuffix(t, "|") && !strings.HasSuffix(t, `\|`) {
		t = t[:len(t)-1]
	}

	var cells []string
	start := 0

	for k := 0; k < len(t); k++ {
		switch t[k] {
		case '\\':
			k++
		case '|':
			cells = append(cells, strings.TrimSpace(t[start:k]))
			start = k + 1
		}
	}

	return append(cells, strings.TrimSpace(t[start:]))
}
