// SPDX-License-Identifier: AGPL-3.0-only

package platform

import "syscall/js"

// openTab opens a page in a new browser tab, from a blob URL.
func openTab(page []byte) error {
	arr := js.Global().Get("Uint8Array").New(len(page))
	js.CopyBytesToJS(arr, page)
	blob := js.Global().Get("Blob").New([]any{arr}, map[string]any{"type": "text/html"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	js.Global().Call("open", url, "_blank")
	return nil
}
