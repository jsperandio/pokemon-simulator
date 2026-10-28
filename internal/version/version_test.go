// SPDX-License-Identifier: AGPL-3.0-only

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_title(t *testing.T) {
	t.Parallel()

	for _, cas := range []struct {
		name string
		tag  string
		want string
	}{
		{
			name: "when the tag is a release's, should show its major and minor numbers",
			tag:  "v4.6.0",
			want: "POKéMON Simulator V4.6",
		},
		{
			name: "when the tag is a release candidate's, should show its release's numbers",
			tag:  "v4.6.0-rc.1",
			want: "POKéMON Simulator V4.6",
		},
		{
			name: "when a number has two digits, should show it whole",
			tag:  "v4.10.12",
			want: "POKéMON Simulator V4.10",
		},
		{
			name: "when commits follow the tag, should show a development build",
			tag:  "v4.5.1-38-g7d3938a",
			want: "POKéMON Simulator (dev)",
		},
		{
			name: "when the tree has changes, should show a development build",
			tag:  "v4.6.0-dirty",
			want: "POKéMON Simulator (dev)",
		},
		{
			name: "when the repository has no v* tag, should show a development build",
			tag:  "v0.0.0-7d3938a",
			want: "POKéMON Simulator (dev)",
		},
		{
			name: "when the build sets no tag, should show a development build",
			tag:  "dev",
			want: "POKéMON Simulator (dev)",
		},
	} {
		t.Run(cas.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, cas.want, title(cas.tag))
		})
	}
}
