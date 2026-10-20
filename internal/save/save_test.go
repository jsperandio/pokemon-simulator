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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// original returns a save from testdata, as the original game wrote it under
// Wine:
//   - newgame: a new game, Gary, with Charmander;
//   - release: three members, then two released (the arrays are compacted);
//   - tournament-prime-mewtwo: Prime Cup won, a L.29 Mewtwo as the prize;
//   - load-v1: the version 1 save of Test_File_Bytes, loaded and written back.
func original(t *testing.T, name string) []byte {
	t.Helper()

	gz, err := os.ReadFile(filepath.Join("testdata", name+".pgs.gz")) //nolint:gosec // a save in testdata
	require.NoError(t, err)

	z, err := gzip.NewReader(bytes.NewReader(gz))
	require.NoError(t, err)

	b, err := io.ReadAll(z)
	require.NoError(t, err)

	return b
}

// readOriginal reads a save from testdata, and requires that it reads.
func readOriginal(t *testing.T, name string) *File {
	t.Helper()

	sav, err := Read(original(t, name))
	require.NoError(t, err, "%s", name)

	return sav
}

// version1Save returns a version 1 save: no header, three flags, 251 POKéDEX
// entries. The original loads it and writes testdata/load-v1 back.
func version1Save() string {
	var sb strings.Builder
	line := func(s string) {
		sb.WriteString(s)
		sb.WriteString("\r\n")
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

	return sb.String()
}

func Test_Read(t *testing.T) {
	t.Parallel()

	t.Run("when a new game is read, should give the player's name", func(t *testing.T) {
		t.Parallel()

		sav := readOriginal(t, "newgame")

		assert.Equal(t, "Gary", sav.PlayerName)
	})

	t.Run("when a new game is read, should give its POKéBalls and money", func(t *testing.T) {
		t.Parallel()

		sav := readOriginal(t, "newgame")

		assert.Equal(t, uint32(3), sav.PokeBalls, "POKéBalls")
		assert.Zero(t, sav.Money, "money")
	})

	t.Run("when a new game is read, should list Charmander in the POKéDEX but not Bulbasaur", func(t *testing.T) {
		t.Parallel()

		sav := readOriginal(t, "newgame")

		assert.True(t, sav.InPokedex[4], "Charmander's POKéDEX entry")
		assert.False(t, sav.InPokedex[1], "Bulbasaur's POKéDEX entry")
	})

	t.Run("when a new game is read, should have Charmander as its only party member", func(t *testing.T) {
		t.Parallel()

		party := readOriginal(t, "newgame").Party()

		require.Len(t, party, 1)
		assert.Equal(t, Slot{
			Species: 4,
			Level:   5,
			XP:      125,
		}, party[0])
	})

	t.Run("when the Prime Cup is won, should keep the prize Mewtwo in the party", func(t *testing.T) {
		t.Parallel()

		party := readOriginal(t, "tournament-prime-mewtwo").Party()

		require.Len(t, party, 2)
		assert.Equal(t, Slot{
			Species: 150,
			Level:   29,
			XP:      24389,
		}, party[1])
	})

	t.Run("when the Prime Cup is won, should give its money and wins", func(t *testing.T) {
		t.Parallel()

		sav := readOriginal(t, "tournament-prime-mewtwo")

		assert.Equal(t, uint32(42000), sav.Money, "money")
		assert.Equal(t, uint32(8), sav.Wins, "wins")
	})

	t.Run("when the save is truncated, should fail", func(t *testing.T) {
		t.Parallel()

		b := original(t, "newgame")
		_, err := Read(b[:len(b)/2])

		assert.ErrorContains(t, err, "the file ends at line")
	})

	t.Run("when a number line holds a word, should fail", func(t *testing.T) {
		t.Parallel()

		b := original(t, "newgame")
		damaged := bytes.Replace(b, []byte("\r\n125\r\n"), []byte("\r\nlots\r\n"), 1)
		_, err := Read(damaged)

		assert.ErrorContains(t, err, "want a whole number")
	})
}

func Test_File_Party(t *testing.T) {
	t.Parallel()

	t.Run("when a slot has species 0, should end the party before it", func(t *testing.T) {
		t.Parallel()

		var sav File
		sav.Owned[0].Species = 4
		sav.Owned[1].Species = 7
		sav.Owned[3].Species = 1 // past the empty slot

		party := sav.Party()
		require.Len(t, party, 2)
		assert.Equal(t, int32(4), party[0].Species)
		assert.Equal(t, int32(7), party[1].Species)
	})

	t.Run("when every slot holds a POKéMON, should give them all", func(t *testing.T) {
		t.Parallel()

		var sav File
		for i := range sav.Owned {
			sav.Owned[i].Species = 1
		}

		assert.Len(t, sav.Party(), Slots)
	})
}

func Test_File_Bytes(t *testing.T) {
	t.Parallel()

	t.Run("when the original's saves are read and written back, should give the same bytes", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"newgame", "release", "tournament-prime-mewtwo", "load-v1"} {
			want := original(t, name)

			sav, err := Read(want)
			require.NoError(t, err, "%s", name)

			assert.Equal(t, string(want), string(sav.Bytes()), "%s: written back differently", name)
		}
	})

	t.Run("when a version 1 save is read and written back, should give the bytes the original wrote", func(t *testing.T) {
		t.Parallel()

		want := original(t, "load-v1")

		sav, err := Read([]byte(version1Save()))
		require.NoError(t, err)

		assert.Equal(t, string(want), string(sav.Bytes()), "the version 1 save isn't written back as the original wrote it")
	})
}
