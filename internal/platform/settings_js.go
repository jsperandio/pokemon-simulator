// SPDX-License-Identifier: AGPL-3.0-only

//go:build js

package platform

import "syscall/js"

// settingsKey is the settings' key in the page's local storage.
const settingsKey string = "pokemon-simulator/settings.ini"

// Settings is where the game keeps the player's preferences between runs:
// the page's local storage, apart from the saves in IndexedDB, so that the
// file dialogs don't list them.
type Settings struct{}

// UserSettings returns the settings in the page's local storage.
func UserSettings() Settings {
	return Settings{}
}

// Read returns the settings' bytes, or ErrNotExist when there are none or
// the browser keeps nothing.
func (Settings) Read() ([]byte, error) {
	var val js.Value

	err := catch(func() {
		val = js.Global().Get("localStorage").Call("getItem", settingsKey)
	})
	if err != nil || val.Type() != js.TypeString {
		return nil, ErrNotExist
	}

	return []byte(val.String()), nil
}

// Write keeps the settings' bytes.
func (Settings) Write(data []byte) error {
	return catch(func() {
		js.Global().Get("localStorage").Call("setItem", settingsKey, string(data))
	})
}
