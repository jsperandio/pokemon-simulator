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

// dialogTitle is a message dialog's caption, as its kind gives it.
func dialogTitle(kind game.DialogKind) string {
	switch kind {
	case game.Warning:
		return "Warning"
	case game.Error:
		return "Error"
	case game.Information:
		return "Information"
	case game.Confirmation:
		return "Confirm"
	default: // Custom
		return game.Title
	}
}

// dontShowAgain is the check box's caption in the boxes the player can
// stop showing, which the original doesn't have.
const dontShowAgain string = "&Don't show this message again"

// messageDialog builds a message dialog as Delphi's CreateMessageDialog
// does, in the message font: an icon, the text wrapped at half the
// screen's width, and the buttons centred under them. With a check box's
// caption, check, the check box goes between the text and the buttons,
// lined up with the text, as Windows' boxes that ask whether to show them
// again have it.
func messageDialog(dsk *desktop, kind game.DialogKind, yesNo bool, text, check string) *window {
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

	// The check box: its box and the text 17 pixels in, as paintCheck
	// draws them, below the text.
	checkW, checkH, blockH := 0, 0, iconTextH
	if check != "" {
		checkW, checkH = 17+fnt.width(plain(check))+2, mulDiv(10, dy, 8)
		blockH += vertSpacing + checkH
	}

	n := len(buttons)
	groupW := buttonWidth*n + buttonSpacing*(n-1)
	textX := iconTextW - textW + horzMargin
	clientW := max(iconTextW, textX-horzMargin+checkW, groupW) + 2*horzMargin
	clientH := blockH + buttonHeight + vertSpacing + 2*vertMargin

	comp := layout.Component{
		Class:   "TMessageForm",
		Name:    "MessageForm",
		Caption: dialogTitle(kind),
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
			X: textX,
			Y: vertMargin,
			W: textW,
			H: textH,
		},
		Props: map[string]string{
			"AutoSize": "False",
			"WordWrap": "True",
		},
	})
	if check != "" {
		comp.Children = append(comp.Children, layout.Component{
			Class:   "TCheckBox",
			Name:    "DontShowAgain",
			Caption: check,
			Bounds: layout.Rect{
				X: textX,
				Y: iconTextH + vertMargin + vertSpacing,
				W: checkW,
				H: checkH,
			},
			Props: map[string]string{
				"TabOrder": strconv.Itoa(len(buttons)), // after them: OK keeps the focus
			},
		})
	}

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
				Y: blockH + vertMargin + vertSpacing,
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
