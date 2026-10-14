// SPDX-License-Identifier: AGPL-3.0-only

package save

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The testdata saves were written by the original game under Wine:
//   - newgame: a new game, Gary, with Charmander;
//   - release: three members, then two released (the arrays are compacted);
//   - tournament-prime-mewtwo: Prime Cup won, a L.29 Mewtwo as the prize;
//   - load-v1: the version 1 save of TestVersion1, loaded and written back.
func original(t *testing.T, name string) []byte {
	t.Helper()

	gz, err := os.ReadFile(filepath.Join("testdata", name+".pgs.gz")) //nolint:gosec // a save in testdata
	if err != nil {
		t.Fatal(err)
	}

	z, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		t.Fatal(err)
	}

	b, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func TestRoundTrip(t *testing.T) {
	for _, name := range []string{"newgame", "release", "tournament-prime-mewtwo", "load-v1"} {
		b := original(t, name)

		f, err := Read(b)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if got := f.Bytes(); !bytes.Equal(got, b) {
			t.Errorf("%s: written back differently (%d bytes, want %d)", name, len(got), len(b))
		}
	}
}

func TestFields(t *testing.T) {
	f, err := Read(original(t, "newgame"))
	if err != nil {
		t.Fatal(err)
	}

	if f.PlayerName != "Gary" || f.PokeBalls != 3 || f.Money != 0 || !f.InDex[4] || f.InDex[1] {
		t.Errorf("new game: %q, %d POKéBalls, %d¥, dex[4]=%v", f.PlayerName, f.PokeBalls, f.Money, f.InDex[4])
	}

	if p := f.Party(); len(p) != 1 || p[0] != (Slot{Species: 4, Level: 5, XP: 125}) {
		t.Errorf("new game party: %v", p)
	}

	f, err = Read(original(t, "tournament-prime-mewtwo"))
	if err != nil {
		t.Fatal(err)
	}

	if p := f.Party(); len(p) != 2 || p[1] != (Slot{Species: 150, Level: 29, XP: 24389}) || f.Money != 42000 || f.Wins != 8 {
		t.Errorf("prime cup: party %v, %d¥, %d wins", p, f.Money, f.Wins)
	}
}

// A version 1 save: no header, three flags, 251 POKéDEX entries. The
// original loads it and writes testdata/load-v1 back.
func TestVersion1(t *testing.T) {
	var b strings.Builder
	line := func(s string) {
		b.WriteString(s)
		b.WriteString("\r\n")
	}
	line("Red")
	line("TRUE")
	line("TRUE")
	line("FALSE") // BeatLance, BeatAsh, BeatLink
	line("9")     // badges

	for _, c := range []int{12345, 10, 2, 14, 3, 3, 4, 0, 0, 1, 0} {
		line(strconv.Itoa(c))
	}

	dexed := []int{16, 19, 25, 249}

	for s := 1; s <= 251; s++ {
		line(boolText(slices.Contains(dexed, s)))
	}

	party := [][3]int{{25, 40, 64000}, {249, 70, 343000}}

	for i := range Slots {
		var m [3]int
		if i < len(party) {
			m = party[i]
		}

		for _, v := range m {
			line(strconv.Itoa(v))
		}
	}

	f, err := Read([]byte(b.String()))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(f.Bytes(), original(t, "load-v1")) {
		t.Error("the version 1 save isn't written back as the original wrote it")
	}
}

func TestDamaged(t *testing.T) {
	b := original(t, "newgame")
	if _, err := Read(b[:len(b)/2]); err == nil {
		t.Error("a truncated save was read")
	}

	if _, err := Read(bytes.Replace(b, []byte("\r\n125\r\n"), []byte("\r\nlots\r\n"), 1)); err == nil {
		t.Error("a save with a word for a number was read")
	}
}
