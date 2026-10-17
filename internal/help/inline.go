// SPDX-License-Identifier: AGPL-3.0-only

package help

import "strings"

// style is the style text takes where it is in a paragraph.
type style struct {
	bold, italic bool
	link         string
}

// inliner builds the spans of a block's text.
type inliner struct {
	spans []Span
}

// parseInline reads a block's text, its lines joined by "\n": emphasis,
// links, images, escapes and line breaks.
func parseInline(s string) []Span {
	var p inliner
	p.run(s, style{})

	return p.spans
}

// canMerge reports whether text in style sty can join s: s is text of that
// style, not a break or an image.
func (s *Span) canMerge(sty style) bool {
	return !s.Break && s.Image == "" && s.Bold == sty.bold && s.Italic == sty.italic && s.Link == sty.link
}

// text adds text in a style, to the last span if it has the same.
func (p *inliner) text(t string, sty style) {
	if t == "" {
		return
	}

	if n := len(p.spans); n > 0 {
		last := &p.spans[n-1]
		if last.canMerge(sty) {
			last.Text += t

			return
		}
	}

	p.spans = append(p.spans, Span{
		Text:   t,
		Bold:   sty.bold,
		Italic: sty.italic,
		Link:   sty.link,
	})
}

// run adds the spans of src, whose text takes style sty.
func (p *inliner) run(src string, sty style) {
	var buf strings.Builder

	for i := 0; i < len(src); {
		chr := src[i]
		switch {
		case chr == '\\' && i+1 < len(src) && src[i+1] == '\n':
			// A backslash at the end of a line breaks it.
			p.flush(&buf, sty)

			p.spans = append(p.spans, Span{Break: true})
			i = skipSpaces(src, i+2)
		case chr == '\\' && i+1 < len(src) && isPunct(src[i+1]):
			buf.WriteByte(src[i+1])

			i += 2
		case chr == '\n':
			// Two spaces at the end of a line break it; otherwise the
			// lines run on, with a space between them.
			t := buf.String()
			buf.Reset()
			buf.WriteString(strings.TrimRight(t, " "))

			if strings.HasSuffix(t, "  ") {
				p.flush(&buf, sty)

				p.spans = append(p.spans, Span{Break: true})
			} else {
				buf.WriteByte(' ')
			}

			i = skipSpaces(src, i+1)
		case chr == '!' && i+1 < len(src) && src[i+1] == '[', chr == '[':
			i = p.link(src, i, sty, &buf)
		case chr == '*':
			i = p.emphasis(src, i, sty, &buf)
		default:
			buf.WriteByte(chr)

			i++
		}
	}

	p.flush(&buf, sty)
}

// flush adds the text in buf, in style sty, and empties buf.
func (p *inliner) flush(buf *strings.Builder, sty style) {
	p.text(buf.String(), sty)
	buf.Reset()
}

// link adds the image or the link at s[i] after the text in buf, and
// returns the index after it. A "!" or a "[" that starts neither is text.
func (p *inliner) link(s string, i int, sty style, buf *strings.Builder) int {
	image := s[i] == '!'

	start := i
	if image {
		start++
	}

	text, target, n, ok := linkAt(s, start)
	if !ok {
		buf.WriteByte(s[i])

		return i + 1
	}

	p.flush(buf, sty)

	if image {
		p.spans = append(p.spans, Span{
			Text:  text,
			Image: target,
			Link:  sty.link,
		})
	} else {
		inner := sty
		inner.link = target
		p.run(text, inner)
	}

	return start + n
}

// emphasis adds the bold or italic text at src[i] after the text in buf, and
// returns the index after it. Stars that start neither are text.
func (p *inliner) emphasis(src string, i int, sty style, buf *strings.Builder) int {
	n := runLength(src, i, '*')
	if n >= 2 {
		if j := closer(src, i+2, "**"); j >= 0 {
			p.flush(buf, sty)

			inner := sty
			inner.bold = true
			p.run(src[i+2:j], inner)

			return j + 2
		}
	}

	if n == 1 {
		if j := closer(src, i+1, "*"); j >= 0 {
			p.flush(buf, sty)

			inner := sty
			inner.italic = true
			p.run(src[i+1:j], inner)

			return j + 1
		}
	}

	buf.WriteString(src[i : i+n])

	return i + n
}

func skipSpaces(s string, i int) int {
	for i < len(s) && s[i] == ' ' {
		i++
	}

	return i
}

// runLength counts the c's from s[i].
func runLength(s string, i int, c byte) int {
	n := 0
	for i+n < len(s) && s[i+n] == c {
		n++
	}

	return n
}

// closer finds the delimiter that closes emphasis opened before from: one
// that follows text, not a space, and stands alone, not as part of a longer
// run. It returns -1 if there is none, or if the emphasis would be empty
// or start with a space.
func closer(src string, from int, delim string) int {
	if from >= len(src) || src[from] == ' ' || src[from] == '\n' {
		return -1
	}

	for j := from + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case '*':
			n := runLength(src, j, '*')
			if n == len(delim) && src[j-1] != ' ' && src[j-1] != '\n' {
				return j
			}

			j += n - 1
		}
	}

	return -1
}

// linkAt reads "[text](target)" at src[i]; n is its length.
func linkAt(src string, i int) (text, target string, n int, ok bool) {
	depth := 0

	for j := i; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case '[':
			depth++
		case ']':
			depth--
			if depth > 0 {
				continue
			}

			if j+1 >= len(src) || src[j+1] != '(' {
				return "", "", 0, false
			}

			end := strings.IndexByte(src[j+2:], ')')
			if end < 0 {
				return "", "", 0, false
			}

			target = strings.TrimSpace(src[j+2 : j+2+end])
			target = strings.TrimSuffix(strings.TrimPrefix(target, "<"), ">")

			return src[i+1 : j], target, j + 2 + end + 1 - i, true
		}
	}

	return "", "", 0, false
}

// isPunct reports whether c is ASCII punctuation, which a backslash
// escapes.
func isPunct(c byte) bool {
	return strings.IndexByte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", c) >= 0
}
