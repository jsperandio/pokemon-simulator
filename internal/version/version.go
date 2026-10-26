// SPDX-License-Identifier: AGPL-3.0-only

// Package version is the program's version, which make's builds take from
// the git tag of the commit they build.
package version

import (
	"regexp"
	"strings"
)

// name is the program's name, which its title follows with the version.
const name string = "POKéMON Simulator"

// tag is the git tag the program was built from, as git describe gives
// it. make's builds set it with -ldflags -X; other builds, such as go run
// and the tests, keep "dev".
var tag = "dev"

// releaseTag matches a release's tag, vX.Y.Z or vX.Y.Z-rc.N, and takes
// its X.Y. A tag that git describe gives with commits after it (-N-gHASH)
// or changes to it (-dirty) is a development build's.
var releaseTag = regexp.MustCompile(`^v(\d+\.\d+)\.\d+(?:-rc\.\d+)?$`)

// Title is the program's title, as its windows show it: its name, then
// "VX.Y" for a release, or "(dev)" for any other build.
func Title() string {
	return title(tag)
}

// Number is the version the About box and the loading screen show: the
// tag without its "v", or "dev".
func Number() string {
	return strings.TrimPrefix(tag, "v")
}

// title is the program's title for tag.
func title(tag string) string {
	mat := releaseTag.FindStringSubmatch(tag)
	if mat == nil {
		return name + " (dev)"
	}

	return name + " V" + mat[1]
}
