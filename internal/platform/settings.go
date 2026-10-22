// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package platform

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Settings is where the game keeps the player's preferences between runs:
// a file in the user's folder for programs' settings.
type Settings struct {
	path string // empty when the system has no such folder
}

// UserSettings returns the settings file in the user's folder for
// programs' settings: ~/.config/pokemon-simulator/settings.ini on Linux,
// %AppData%\pokemon-simulator\settings.ini on Windows, and
// ~/Library/Application Support/pokemon-simulator/settings.ini on macOS.
func UserSettings() Settings {
	dir, err := os.UserConfigDir()
	if err != nil {
		return Settings{}
	}

	return NewSettings(filepath.Join(dir, "pokemon-simulator", "settings.ini"))
}

// NewSettings returns the settings kept in the file at path.
func NewSettings(path string) Settings {
	return Settings{
		path: path,
	}
}

// Read returns the settings' bytes, or ErrNotExist when there are none.
func (s Settings) Read() ([]byte, error) {
	if s.path == "" {
		return nil, ErrNotExist
	}

	dat, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotExist
	}

	return dat, err
}

// Write keeps the settings' bytes, making their folder if needed.
func (s Settings) Write(data []byte) error {
	if s.path == "" {
		return errors.New("the system has no folder for the settings")
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o600)
}
