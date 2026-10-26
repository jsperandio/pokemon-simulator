// SPDX-License-Identifier: AGPL-3.0-only

// Package layout holds the form layouts of the original POKéMON Simulator
// V4.5, translated once from its Delphi form (DFM) resources.
//
// Values are the original design-time properties, in pixels at 96 DPI. The
// DFM format omits properties left at their VCL default, so a zero value
// here means "VCL default": Hidden is false because Visible defaults to
// true, and so on.
package layout

import (
	"maps"
	"slices"
)

// All lists every form of the original game.
var All = []*Component{
	&Form1,
	&Form3,
	&Form4,
	&Form5,
	&Form7,
	&Form8,
	&Form9,
	&FormA,
	&FormB,
	&OKRightDlg,
	&AboutBox,
}

// Rect is a component's Left, Top, Width and Height in pixels, relative to
// its parent's client area. Forms store their client size in the
// ClientWidth and ClientHeight props instead.
type Rect struct{ X, Y, W, H int }

// ListItem is a design-time item of a TListView.
type ListItem struct {
	Caption    string
	ImageIndex int
}

// Component is one VCL component and its design-time properties.
type Component struct {
	Class    string            // VCL class, e.g. "TButton"
	Name     string            // component name, e.g. "NewButton"
	Bounds   Rect              // Left, Top, Width, Height
	Caption  string            // Caption, or Text for edit and combo boxes
	Hidden   bool              // Visible = False
	Disabled bool              // Enabled = False
	Items    []string          // Items.Strings or Lines.Strings
	List     []ListItem        // TListView items
	Events   map[string]string // event → handler method, e.g. "OnClick": "Red1Click"
	Props    map[string]string // every other property, as written in the DFM
	Children []Component
}

// Find returns the descendant of c with the given name, or nil.
func (c *Component) Find(name string) *Component {
	for i := range c.Children {
		chd := &c.Children[i]
		if chd.Name == name {
			return chd
		}

		if found := chd.Find(name); found != nil {
			return found
		}
	}

	return nil
}

// Walk calls fn for c and every descendant, depth first.
func (c *Component) Walk(fn func(*Component)) {
	fn(c)

	for i := range c.Children {
		c.Children[i].Walk(fn)
	}
}

// Clone returns a deep copy of c, for a layout the port changes: the
// forms share their slices and maps.
func (c *Component) Clone() Component {
	out := *c
	out.Items = slices.Clone(c.Items)
	out.List = slices.Clone(c.List)
	out.Events = maps.Clone(c.Events)
	out.Props = maps.Clone(c.Props)
	out.Children = slices.Clone(c.Children)

	for i := range out.Children {
		out.Children[i] = c.Children[i].Clone()
	}

	return out
}
