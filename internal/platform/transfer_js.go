// SPDX-License-Identifier: AGPL-3.0-only

package platform

import (
	"syscall/js"
	"time"
)

// Upload asks the player for files with the browser's file picker, and
// copies them into dir. It returns at once: the names of the files copied
// come on the channel once they are, none if the player cancels.
func Upload(dest writer, dir string) <-chan []string {
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
				file := list.Index(i)

				buf, err := awaitPromise(file.Call("arrayBuffer"))
				if err != nil {
					warn("POKéMON Simulator: " + err.Error())

					continue
				}

				arr := js.Global().Get("Uint8Array").New(buf)
				b := make([]byte, arr.Get("length").Int())
				js.CopyBytesToGo(b, arr)

				name := file.Get("name").String()
				if err := dest.WriteFile(dest.Join(dir, name), b); err != nil {
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
func awaitPromise(promise js.Value) (js.Value, error) {
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

	promise.Call("then", then, fail)

	select {
	case v := <-vals:
		return v, nil
	case err := <-errs:
		return js.Undefined(), err
	}
}

var (
	leaveGuard js.Func
	guarding   bool
)

// WarnOnLeave has the browser ask before the page closes or reloads while
// on is true: a game in progress is saved only when the player quits.
func WarnOnLeave(on bool) {
	if on == guarding {
		return
	}

	guarding = on
	if on {
		leaveGuard = js.FuncOf(func(_ js.Value, args []js.Value) any {
			args[0].Call("preventDefault")
			args[0].Set("returnValue", "")

			return nil
		})
		js.Global().Call("addEventListener", "beforeunload", leaveGuard)

		return
	}

	js.Global().Call("removeEventListener", "beforeunload", leaveGuard)
	leaveGuard.Release()
}
