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

// text adds text in a style, to the last span if it has the same.
func (p *inliner) text(t string, st style) {
	if t == "" {
		return
	}
	if n := len(p.spans); n > 0 {
		last := &p.spans[n-1]
		if !last.Break && last.Image == "" && last.Bold == st.bold && last.Italic == st.italic && last.Link == st.link {
			last.Text += t
			return
		}
	}
	p.spans = append(p.spans, Span{Text: t, Bold: st.bold, Italic: st.italic, Link: st.link})
}

func (p *inliner) run(s string, st style) {
	var buf strings.Builder
	flush := func() {
		p.text(buf.String(), st)
		buf.Reset()
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && s[i+1] == '\n':
			// A backslash at the end of a line breaks it.
			flush()
			p.spans = append(p.spans, Span{Break: true})
			i = skipSpaces(s, i+2)
		case c == '\\' && i+1 < len(s) && isPunct(s[i+1]):
			buf.WriteByte(s[i+1])
			i += 2
		case c == '\n':
			// Two spaces at the end of a line break it; otherwise the
			// lines run on, with a space between them.
			t := buf.String()
			buf.Reset()
			buf.WriteString(strings.TrimRight(t, " "))
			if strings.HasSuffix(t, "  ") {
				flush()
				p.spans = append(p.spans, Span{Break: true})
			} else {
				buf.WriteByte(' ')
			}
			i = skipSpaces(s, i+1)
		case c == '!' && i+1 < len(s) && s[i+1] == '[':
			text, target, n, ok := linkAt(s, i+1)
			if !ok {
				buf.WriteByte(c)
				i++
				continue
			}
			flush()
			p.spans = append(p.spans, Span{Text: text, Image: target, Link: st.link})
			i += 1 + n
		case c == '[':
			text, target, n, ok := linkAt(s, i)
			if !ok {
				buf.WriteByte(c)
				i++
				continue
			}
			flush()
			in := st
			in.link = target
			p.run(text, in)
			i += n
		case c == '*':
			n := runLength(s, i, '*')
			if n >= 2 {
				if j := closer(s, i+2, "**"); j >= 0 {
					flush()
					in := st
					in.bold = true
					p.run(s[i+2:j], in)
					i = j + 2
					continue
				}
			}
			if n == 1 {
				if j := closer(s, i+1, "*"); j >= 0 {
					flush()
					in := st
					in.italic = true
					p.run(s[i+1:j], in)
					i = j + 1
					continue
				}
			}
			buf.WriteString(s[i : i+n])
			i += n
		default:
			buf.WriteByte(c)
			i++
		}
	}
	flush()
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
func closer(s string, from int, delim string) int {
	if from >= len(s) || s[from] == ' ' || s[from] == '\n' {
		return -1
	}
	for j := from + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '*':
			n := runLength(s, j, '*')
			if n == len(delim) && s[j-1] != ' ' && s[j-1] != '\n' {
				return j
			}
			j += n - 1
		}
	}
	return -1
}

// linkAt reads "[text](target)" at s[i]; n is its length.
func linkAt(s string, i int) (text, target string, n int, ok bool) {
	depth := 0
	for j := i; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '[':
			depth++
		case ']':
			depth--
			if depth > 0 {
				continue
			}
			if j+1 >= len(s) || s[j+1] != '(' {
				return "", "", 0, false
			}
			end := strings.IndexByte(s[j+2:], ')')
			if end < 0 {
				return "", "", 0, false
			}
			target = strings.TrimSpace(s[j+2 : j+2+end])
			target = strings.TrimSuffix(strings.TrimPrefix(target, "<"), ">")
			return s[i+1 : j], target, j + 2 + end + 1 - i, true
		}
	}
	return "", "", 0, false
}

// isPunct reports whether c is ASCII punctuation, which a backslash
// escapes.
func isPunct(c byte) bool {
	return strings.IndexByte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", c) >= 0
}
