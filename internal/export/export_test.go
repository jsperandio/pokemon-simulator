// SPDX-License-Identifier: AGPL-3.0-only

package export

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testdata/export.htm was written by the original under Wine (the export
// scenario): Probe's party of Bulbasaur L.5, Mewtwo L.100 and Treecko
// L.50, Squirtle registered too.
func TestPage(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "export.htm"))
	if err != nil {
		t.Fatal(err)
	}
	got := Page(Info{
		Player: "Probe",
		Party: []Member{
			{Entry{1, "Bulbasaur"}, 5, 125},
			{Entry{150, "Mewtwo"}, 100, 1000000},
			{Entry{252, "Treecko"}, 50, 130000},
		},
		Dex:      []Entry{{1, "Bulbasaur"}, {7, "Squirtle"}, {150, "Mewtwo"}, {252, "Treecko"}},
		DexCount: 3,
		Date:     time.Date(2026, 10, 11, 17, 32, 51, 0, time.Local),
	})
	if bytes.Equal(got, want) {
		return
	}
	g := strings.Split(string(got), "\r\n")
	w := strings.Split(string(want), "\r\n")
	for i := range max(len(g), len(w)) {
		var a, b string
		if i < len(g) {
			a = g[i]
		}
		if i < len(w) {
			b = w[i]
		}
		if a != b {
			t.Fatalf("line %d:\nport:     %q\noriginal: %q", i+1, a, b)
		}
	}
	t.Fatal("the pages differ")
}
