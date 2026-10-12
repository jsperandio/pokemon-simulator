// SPDX-License-Identifier: AGPL-3.0-only

package platform

import (
	"syscall/js"
	"time"
)

// Upload asks the player for files with the browser's file picker, and
// copies them into dir. It returns at once: the names of the files copied
// come on the channel once they are, none if the player cancels.
func Upload(files Files, dir string) <-chan []string {
	out := make(chan []string, 1)
	input := js.Global().Get("document").Call("createElement", "input")
	input.Set("type", "file")
	input.Set("multiple", true)
	var onChange, onCancel js.Func
	finish := func(names []string) {
		out <- names
		onChange.Release()
		onCancel.Release()
	}
	onChange = js.FuncOf(func(js.Value, []js.Value) any {
		list := input.Get("files")
		// Reading a file waits for a promise, which a callback can't.
		go func() {
			var names []string
			for i := range list.Get("length").Int() {
				f := list.Index(i)
				buf, err := awaitPromise(f.Call("arrayBuffer"))
				if err != nil {
					warn("POKéMON Simulator: " + err.Error())
					continue
				}
				arr := js.Global().Get("Uint8Array").New(buf)
				b := make([]byte, arr.Get("length").Int())
				js.CopyBytesToGo(b, arr)
				name := f.Get("name").String()
				if err := files.WriteFile(files.Join(dir, name), b); err != nil {
					warn("POKéMON Simulator: " + name + ": " + err.Error())
					continue
				}
				names = append(names, name)
			}
			finish(names)
		}()
		return nil
	})
	onCancel = js.FuncOf(func(js.Value, []js.Value) any {
		finish(nil)
		return nil
	})
	input.Call("addEventListener", "change", onChange)
	input.Call("addEventListener", "cancel", onCancel)
	input.Call("click")
	return out
}

// Download gives the player a copy of a file, through the browser's
// downloads.
func Download(name string, data []byte) error {
	blob := js.Global().Get("Blob").New([]any{uint8Array(data)}, map[string]any{"type": "application/octet-stream"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	doc := js.Global().Get("document")
	a := doc.Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", name)
	doc.Get("body").Call("appendChild", a)
	a.Call("click")
	a.Call("remove")
	go func() {
		time.Sleep(time.Minute)
		js.Global().Get("URL").Call("revokeObjectURL", url)
	}()
	return nil
}

// awaitPromise waits for a promise and returns its value.
func awaitPromise(p js.Value) (js.Value, error) {
	vals := make(chan js.Value, 1)
	errs := make(chan error, 1)
	then := js.FuncOf(func(_ js.Value, args []js.Value) any {
		vals <- args[0]
		return nil
	})
	fail := js.FuncOf(func(_ js.Value, args []js.Value) any {
		errs <- jsError(args[0])
		return nil
	})
	defer then.Release()
	defer fail.Release()
	p.Call("then", then, fail)
	select {
	case v := <-vals:
		return v, nil
	case err := <-errs:
		return js.Undefined(), err
	}
}
