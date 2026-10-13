// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"strconv"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// mulDiv is Windows' MulDiv: a × b / c, rounded.
func mulDiv(a, b, c int) int { return (a*b + c/2) / c }

// messageDialog builds a message dialog as Delphi's CreateMessageDialog
// does, in the message font: an icon, the text wrapped at half the
// screen's width, and the buttons centred under them.
func messageDialog(d *desktop, kind game.DialogKind, yesNo bool, text string) *window {
	res := d.res
	f := res.sans
	dx, dy := res.charWidth, f.height() // the dialog units
	horzMargin, vertMargin := mulDiv(8, dx, 4), mulDiv(8, dy, 8)
	horzSpacing, vertSpacing := mulDiv(10, dx, 4), mulDiv(10, dy, 8)
	buttonWidth, buttonHeight := mulDiv(50, dx, 4), mulDiv(14, dy, 8)
	buttonSpacing := mulDiv(4, dx, 4)

	type btn struct {
		caption string
		result  int
	}
	buttons := []btn{{"OK", mrOk}}

	if yesNo {
		buttons = []btn{{"&Yes", mrYes}, {"&No", mrNo}}
	}

	for _, b := range buttons {
		buttonWidth = max(buttonWidth, f.width(plain(b.caption))+8)
	}

	lines := wrap(f, text, screenWidth/2)
	textW := 0

	for _, l := range lines {
		textW = max(textW, f.width(l))
	}

	textH := len(lines) * dy
	iconTextW, iconTextH := textW, textH
	icon := res.dlgIcons[kind]

	if icon != nil {
		iconTextW += 32 + horzSpacing
		iconTextH = max(iconTextH, 32)
	}

	n := len(buttons)
	groupW := buttonWidth*n + buttonSpacing*(n-1)
	clientW := max(iconTextW, groupW) + 2*horzMargin
	clientH := iconTextH + buttonHeight + vertSpacing + 2*vertMargin

	title := game.Title

	switch kind {
	case game.Warning:
		title = "Warning"
	case game.Error:
		title = "Error"
	case game.Information:
		title = "Information"
	case game.Confirmation:
		title = "Confirm"
	}

	comp := layout.Component{
		Class:   "TMessageForm",
		Name:    "MessageForm",
		Caption: title,
		Props: map[string]string{
			"BorderStyle":  "bsDialog",
			"ClientWidth":  strconv.Itoa(clientW),
			"ClientHeight": strconv.Itoa(clientH),
		},
	}
	if icon != nil {
		comp.Children = append(comp.Children, layout.Component{
			Class: "TImage", Name: "Image",
			Bounds: layout.Rect{X: horzMargin, Y: vertMargin, W: 32, H: 32},
		})
	}

	comp.Children = append(comp.Children, layout.Component{
		Class: "TLabel", Name: "Message", Caption: text,
		Bounds: layout.Rect{X: iconTextW - textW + horzMargin, Y: vertMargin, W: textW, H: textH},
		Props:  map[string]string{"AutoSize": "False", "WordWrap": "True"},
	})
	x := (clientW - groupW) / 2

	for i, b := range buttons {
		props := map[string]string{
			"ModalResult": strconv.Itoa(b.result),
			"TabOrder":    strconv.Itoa(i),
		}
		if i == 0 {
			props["Default"] = "True" // OK or Yes
		}

		if i == n-1 {
			props["Cancel"] = "True" // OK or No
		}

		comp.Children = append(comp.Children, layout.Component{
			Class: "TButton", Name: "Button" + strconv.Itoa(i+1), Caption: b.caption,
			Bounds: layout.Rect{X: x, Y: iconTextH + vertMargin + vertSpacing, W: buttonWidth, H: buttonHeight},
			Props:  props,
		})
		x += buttonWidth + buttonSpacing
	}

	w := newWindow(d, &comp)
	if p, ok := w.find("Image").(*picture); ok {
		p.img, p.alpha = icon, true
	}

	// Left := Screen.Width div 2 − Width div 2, and the same for Top.
	s := w.size()
	w.pos.X, w.pos.Y = screenWidth/2-s.X/2, screenHeight/2-s.Y/2
	d.add(w)

	return w
}
