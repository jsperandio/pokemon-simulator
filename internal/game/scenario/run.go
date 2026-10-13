// SPDX-License-Identifier: AGPL-3.0-only

package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
)

// Run plays every scenario in dir, each as a subtest, through the driver
// newDriver makes for it, and checks that the port's trace is the
// original's. If saves isn't "", the save each scenario quits with is
// written there, for the original to load (portsave).
func Run(t *testing.T, dir string, data *gamedata.Data, newDriver func(t *testing.T) Driver, saves string) {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if len(paths) == 0 {
		t.Fatal("no scenarios")
	}

	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".txt")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			p := &Player{
				Name:      name,
				Dir:       dir,
				Data:      data,
				NewDriver: func() Driver { return newDriver(t) },
				Saves:     saves,
			}

			want, err := p.Play(string(src))
			if err != nil {
				t.Errorf("%v\n%s", err, tail(p.out, 8))

				return
			}

			compare(t, want, p.out)
		})
	}
}

// compare reports the first line where the port's trace differs from the
// original's.
func compare(t *testing.T, want, got []string) {
	t.Helper()

	for i := range max(len(want), len(got)) {
		w, g := "(end)", "(end)"
		if i < len(want) {
			w = want[i]
		}

		if i < len(got) {
			g = got[i]
		}

		if w != g {
			t.Errorf("line %d differs:\noriginal: %s\nport:     %s\nthe port's trace before it:\n%s",
				i+1, w, g, tail(got[:min(i, len(got))], 8))

			return
		}
	}
}

func tail(lines []string, n int) string {
	return "\t" + strings.Join(lines[max(len(lines)-n, 0):], "\n\t")
}
