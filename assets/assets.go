// SPDX-License-Identifier: AGPL-3.0-only

// Package assets embeds the original game's data, images and help manual,
// and the free fonts and icons that stand in for Windows' own. The
// embedded files are not covered by the AGPL; see README.md in this
// directory.
package assets

import "embed"

// FS holds the embedded assets by path relative to this directory, such as
// "pokegame.dat", "sprites/front64.png", "help/index.md" or
// "wine/sserife.fon".
//
//go:embed pokegame.dat sprites help wine/*.fon wine/*.png liberation/*.ttf
var FS embed.FS
