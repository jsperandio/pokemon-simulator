// SPDX-License-Identifier: AGPL-3.0-only

//go:build noanim

package anim

import "embed"

// files is empty: a build with the noanim tag has no animations.
var files embed.FS
