// SPDX-License-Identifier: AGPL-3.0-only

//go:build !noanim

package anim

import "embed"

// files holds front.json and the frames, front/NNN.png.
//
//go:embed front.json front
var files embed.FS
