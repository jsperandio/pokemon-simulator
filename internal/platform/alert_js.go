// SPDX-License-Identifier: AGPL-3.0-only

package platform

import "syscall/js"

// Alert tells the player about a problem in the browser's alert box.
func Alert(title, text string) {
	js.Global().Call("alert", title+"\n\n"+text)
}
