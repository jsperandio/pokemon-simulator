// SPDX-License-Identifier: AGPL-3.0-only

package platform

import (
	"errors"
	"syscall/js"
	"time"
)

// database is the game's IndexedDB database. Its one object store holds a
// record for each file and folder, by path: {dir, time, data}.
type database struct {
	db js.Value
}

const (
	dbName    = "pokemon-simulator"
	dbVersion = 1
	dbStore   = "files"
)

// openDatabase opens the database, making its store the first time.
func openDatabase() (*database, error) {
	idb := js.Global().Get("indexedDB")
	if !idb.Truthy() {
		return nil, errors.New("the browser has no IndexedDB")
	}

	var req js.Value

	if err := catch(func() { req = idb.Call("open", dbName, dbVersion) }); err != nil {
		return nil, err
	}

	upgrade := js.FuncOf(func(js.Value, []js.Value) any {
		db := req.Get("result")
		if !db.Get("objectStoreNames").Call("contains", dbStore).Bool() {
			db.Call("createObjectStore", dbStore)
		}

		return nil
	})

	defer upgrade.Release()

	req.Set("onupgradeneeded", upgrade)

	if err := await(req, "success"); err != nil {
		return nil, err
	}

	return &database{req.Get("result")}, nil
}

// load reads every record into m.
func (d *database) load(m *memory) error {
	var req js.Value

	if err := catch(func() {
		req = d.db.Call("transaction", dbStore, "readonly").Call("objectStore", dbStore).Call("openCursor")
	}); err != nil {
		return err
	}

	done := make(chan error, 1)
	next := js.FuncOf(func(js.Value, []js.Value) any {
		c := req.Get("result")
		if c.IsNull() || c.IsUndefined() {
			done <- nil

			return nil
		}

		v := c.Get("value")
		t := time.UnixMilli(int64(v.Get("time").Float()))

		var data []byte

		if arr := v.Get("data"); arr.Truthy() {
			data = make([]byte, arr.Get("length").Int())
			js.CopyBytesToGo(data, arr)
		}

		m.restore(c.Get("key").String(), v.Get("dir").Truthy(), t, data)
		c.Call("continue")

		return nil
	})
	fail := js.FuncOf(func(js.Value, []js.Value) any {
		done <- jsError(req.Get("error"))

		return nil
	})

	defer next.Release()
	defer fail.Release()

	req.Set("onsuccess", next)
	req.Set("onerror", fail)

	return <-done
}

// put writes a file's or a folder's record and waits until it is kept.
func (d *database) put(p string, dir bool, t time.Time, data []byte) error {
	rec := js.Global().Get("Object").New()
	rec.Set("dir", dir)
	rec.Set("time", t.UnixMilli())

	if !dir {
		rec.Set("data", uint8Array(data))
	}

	var tx js.Value

	if err := catch(func() {
		tx = d.db.Call("transaction", dbStore, "readwrite")
		tx.Call("objectStore", dbStore).Call("put", rec, p)
	}); err != nil {
		return err
	}

	return await(tx, "complete")
}

// await waits for an IndexedDB request's success, or a transaction's
// completion, given by ok, or for its failure.
func await(target js.Value, ok string) error {
	done := make(chan error, 1)
	onOK := js.FuncOf(func(js.Value, []js.Value) any {
		done <- nil

		return nil
	})
	onErr := js.FuncOf(func(js.Value, []js.Value) any {
		done <- jsError(target.Get("error"))

		return nil
	})

	defer onOK.Release()
	defer onErr.Release()

	target.Set("on"+ok, onOK)
	target.Set("onerror", onErr)
	target.Set("onabort", onErr)

	return <-done
}

// catch runs f, turning a JavaScript exception into an error.
func catch(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(js.Error)
			if !ok {
				panic(r)
			}

			err = e
		}
	}()

	f()

	return nil
}

// jsError turns a DOMException, or nothing, into an error.
func jsError(v js.Value) error {
	if v.Truthy() && v.Get("message").Truthy() {
		return errors.New(v.Get("message").String())
	}

	return errors.New("the browser's storage failed")
}

// uint8Array copies b into a new Uint8Array.
func uint8Array(b []byte) js.Value {
	arr := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(arr, b)

	return arr
}

// warn writes a warning to the browser's console.
func warn(msg string) {
	js.Global().Get("console").Call("warn", msg)
}
