// SPDX-License-Identifier: AGPL-3.0-only

// Package assets embeds the original game's data, images and help manual.
// The embedded files are not covered by the AGPL; see README.md in this
// directory.
package assets

import "embed"

// FS holds the embedded assets by path relative to this directory, such as
// "pokegame.dat", "sprites/front64.png" or "help/index.md".
//
//go:embed pokegame.dat sprites help
var FS embed.FS
