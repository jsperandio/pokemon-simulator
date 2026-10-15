// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"strconv"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// mulDiv is Windows' MulDiv: num × nmr / den, rounded.
func mulDiv(num, nmr, den int) int {
	return (num*nmr + den/2) / den
}

// messageDialog builds a message dialog as Delphi's CreateMessageDialog
// does, in the message font: an icon, the text wrapped at half the
// screen's width, and the buttons centred under them.
func messageDialog(dsk *desktop, kind game.DialogKind, yesNo bool, text string) *window {
	res := dsk.res
	fnt := res.sans
	dx, dy := res.charWidth, fnt.height() // the dialog units
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
		buttons = []btn{
			{"&Yes", mrYes},
			{"&No", mrNo},
		}
	}

	for _, but := range buttons {
		buttonWidth = max(buttonWidth, fnt.width(plain(but.caption))+8)
	}

	lines := wrap(fnt, text, screenWidth/2)
	textW := 0

	for _, lin := range lines {
		textW = max(textW, fnt.width(lin))
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

	var title string

	switch kind {
	case game.Warning:
		title = "Warning"
	case game.Error:
		title = "Error"
	case game.Information:
		title = "Information"
	case game.Confirmation:
		title = "Confirm"
	default: // Custom
		title = game.Title
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
			Class:  "TImage",
			Name:   "Image",
			Bounds: layout.Rect{X: horzMargin, Y: vertMargin, W: 32, H: 32},
		})
	}

	comp.Children = append(comp.Children, layout.Component{
		Class:   "TLabel",
		Name:    "Message",
		Caption: text,
		Bounds: layout.Rect{
			X: iconTextW - textW + horzMargin,
			Y: vertMargin,
			W: textW,
			H: textH,
		},
		Props: map[string]string{
			"AutoSize": "False",
			"WordWrap": "True",
		},
	})
	x := (clientW - groupW) / 2

	for i, but := range buttons {
		props := map[string]string{
			"ModalResult": strconv.Itoa(but.result),
			"TabOrder":    strconv.Itoa(i),
		}
		if i == 0 {
			props["Default"] = "True" // OK or Yes
		}

		if i == n-1 {
			props["Cancel"] = "True" // OK or No
		}

		comp.Children = append(comp.Children, layout.Component{
			Class:   "TButton",
			Name:    "Button" + strconv.Itoa(i+1),
			Caption: but.caption,
			Bounds: layout.Rect{
				X: x,
				Y: iconTextH + vertMargin + vertSpacing,
				W: buttonWidth,
				H: buttonHeight,
			},
			Props: props,
		})
		x += buttonWidth + buttonSpacing
	}

	wdw := newWindow(dsk, &comp)
	if pic, ok := wdw.find("Image").(*picture); ok {
		pic.img, pic.alpha = icon, true
	}

	// Left := Screen.Width div 2 − Width div 2, and the same for Top.
	pnt := wdw.size()
	wdw.pos.X, wdw.pos.Y = screenWidth/2-pnt.X/2, screenHeight/2-pnt.Y/2
	dsk.add(wdw)

	return wdw
}
