// SPDX-License-Identifier: AGPL-3.0-only

package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
)

// Run plays every scenario in dir, each as a parallel subtest, through the
// driver newDriver makes for it, and checks that the port's trace is the
// original's. If saves isn't "", the save each scenario quits with is
// written there, for the original to load (portsave).
func Run(t *testing.T, dir string, data *gamedata.Data, newDriver func(t *testing.T) Driver, saves string) {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	require.NoError(t, err)
	require.NotEmpty(t, paths, "no scenarios in %s", dir)

	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".txt")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			src, err := os.ReadFile(path) //nolint:gosec // a scenario the glob found
			require.NoError(t, err)

			ply := &Player{
				Name: name,
				Dir:  dir,
				Data: data,
				NewDriver: func() Driver {
					return newDriver(t)
				},
				Saves: saves,
			}

			want, err := ply.Play(string(src))
			require.NoError(t, err, "the port's trace before it:\n%s", tail(ply.out, 8))

			compare(t, want, ply.out)
		})
	}
}

// reporter is what compare needs of a test: testing.T, or a fake that
// records what compare reports.
type reporter interface {
	Helper()
	Errorf(format string, args ...any)
}

// compare reports the first line where the port's trace differs from the
// original's, with the port's lines before it.
func compare(t reporter, want, got []string) {
	t.Helper()

	idx := firstDifference(want, got)

	assert.Equal(t, lineAt(want, idx), lineAt(got, idx), "line %d differs; the port's trace before it:\n%s",
		idx+1, tail(got[:min(idx, len(got))], 8))
}

// firstDifference returns the index of the first line where the traces
// differ, or the longer one's length when they don't.
func firstDifference(want, got []string) int {
	for i := range max(len(want), len(got)) {
		if lineAt(want, i) != lineAt(got, i) {
			return i
		}
	}

	return max(len(want), len(got))
}

// lineAt returns line i of a trace, or "(end)" past its end.
func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}

	return "(end)"
}

func tail(lines []string, n int) string {
	return "\t" + strings.Join(lines[max(len(lines)-n, 0):], "\n\t")
}
