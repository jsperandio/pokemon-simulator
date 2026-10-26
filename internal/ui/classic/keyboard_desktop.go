// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

// newKeyboard returns nil: on the desktop the player types on the
// computer's keyboard.
func newKeyboard() *keyboard {
	return nil
}
