// SPDX-License-Identifier: AGPL-3.0-only

package platform

import "syscall/js"

// OpenPage shows a page in a new tab, from a blob URL.
func OpenPage(_ string, page []byte) error {
	blob := js.Global().Get("Blob").New([]any{uint8Array(page)}, map[string]any{"type": "text/html"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	js.Global().Call("open", url, "_blank")

	return nil
}

// OpenURL opens a web address in a new tab; a mailto: one opens the
// browser's e-mail program.
func OpenURL(url string) error {
	js.Global().Call("open", url, "_blank", "noopener")

	return nil
}
