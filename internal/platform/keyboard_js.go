// SPDX-License-Identifier: AGPL-3.0-only

package platform

import "syscall/js"

// TouchScreen reports whether the player's main pointer is a finger, as on
// a phone or a tablet, where the keyboard is one the screen shows.
func TouchScreen() bool {
	return js.Global().Call("matchMedia", "(pointer: coarse)").Get("matches").Bool()
}

// KeyboardDigits asks the keyboard the screen shows for digits, or for
// letters again: the input mode of Ebitengine's hidden text area, where it
// types. A keyboard takes a new mode only as it comes up, so a text area
// with the focus loses it, and the tap that is opening the keyboard gives
// it back.
func KeyboardDigits(on bool) {
	doc := js.Global().Get("document")

	area := doc.Call("getElementById", "ebitengine-textinput")
	if !area.Truthy() {
		return
	}

	mode := "text"
	if on {
		mode = "numeric"
	}

	if area.Get("inputMode").String() == mode {
		return
	}

	area.Set("inputMode", mode)

	if doc.Get("activeElement").Equal(area) {
		area.Call("blur")
	}
}
