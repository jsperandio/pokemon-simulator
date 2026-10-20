// SPDX-License-Identifier: AGPL-3.0-only

package scenario

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
)

// fakeT records what compare reports, in place of a test's testing.T.
type fakeT struct {
	reports []string
}

func (ft *fakeT) Helper() {}

func (ft *fakeT) Errorf(format string, args ...any) {
	ft.reports = append(ft.reports, fmt.Sprintf(format, args...))
}

// loadData loads the POKéGAME.dat embedded in the port.
func loadData(t *testing.T) *gamedata.Data {
	t.Helper()

	raw, err := assets.FS.ReadFile("pokegame.dat")
	require.NoError(t, err)

	data, err := gamedata.Load(raw)
	require.NoError(t, err)

	return data
}

// newPlayer returns a Player on the game's handlers, as internal/game's
// Test_Scenarios makes them.
func newPlayer(data *gamedata.Data) *Player {
	return &Player{
		Name: "test",
		Data: data,
		NewDriver: func() Driver {
			return &Direct{}
		},
	}
}

// The replays in internal/game and internal/ui/classic all pass, so they
// never run compare's failing side: with a comparison that finds no
// difference, they would pass whatever the port does.
func Test_compare(t *testing.T) {
	t.Parallel()

	t.Run("when the traces are the same, should report nothing", func(t *testing.T) {
		t.Parallel()

		var ft fakeT
		compare(&ft, []string{"> start", "log A"}, []string{"> start", "log A"})

		assert.Empty(t, ft.reports)
	})

	t.Run("when a line differs, should report its number, both lines and the port's lines before it", func(t *testing.T) {
		t.Parallel()

		var ft fakeT
		compare(&ft, []string{"> start", "log A", "log B"}, []string{"> start", "log A", "log C"})

		require.Len(t, ft.reports, 1)
		assert.Contains(t, ft.reports[0], "line 3 differs")
		assert.Contains(t, ft.reports[0], `expected: "log B"`)
		assert.Contains(t, ft.reports[0], `actual  : "log C"`)
		assert.Contains(t, ft.reports[0], "\tlog A")
	})

	t.Run("when the port's trace stops early, should report the original's next line against its end", func(t *testing.T) {
		t.Parallel()

		var ft fakeT
		compare(&ft, []string{"> start", "log A"}, []string{"> start"})

		require.Len(t, ft.reports, 1)
		assert.Contains(t, ft.reports[0], "line 2 differs")
		assert.Contains(t, ft.reports[0], `expected: "log A"`)
		assert.Contains(t, ft.reports[0], `actual  : "(end)"`)
	})

	t.Run("when the port's trace goes on, should report its extra line against the original's end", func(t *testing.T) {
		t.Parallel()

		var ft fakeT
		compare(&ft, []string{"> start"}, []string{"> start", "log X"})

		require.Len(t, ft.reports, 1)
		assert.Contains(t, ft.reports[0], "line 2 differs")
		assert.Contains(t, ft.reports[0], `expected: "(end)"`)
		assert.Contains(t, ft.reports[0], `actual  : "log X"`)
	})
}

func Test_tail(t *testing.T) {
	t.Parallel()

	t.Run("when there are more lines than asked for, should keep the last ones, indented", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "\tc\n\td", tail([]string{"a", "b", "c", "d"}, 2))
	})

	t.Run("when there are fewer lines than asked for, should keep them all", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "\ta\n\tb", tail([]string{"a", "b"}, 8))
	})
}

// The scenarios are well formed and the port plays them all, so the
// replays never run Play's failing side either: a scenario the port can't
// play must stop, naming its line, rather than play on.
func Test_Player_Play(t *testing.T) {
	t.Parallel()

	data := loadData(t)

	t.Run("when the port does what the original did, should record the same lines", func(t *testing.T) {
		t.Parallel()

		ply := newPlayer(data)

		want, err := ply.Play("# a comment\n\nparty 4 5\n> start\n> quit\nexit 0\nsave unchanged\n")
		require.NoError(t, err)

		assert.Equal(t, []string{"> start", "> quit", "exit 0", "save unchanged"}, want)
		assert.Equal(t, want, ply.Trace())
	})

	t.Run("when there is no start, should fail", func(t *testing.T) {
		t.Parallel()

		_, err := newPlayer(data).Play("party 4 5\n")

		assert.EqualError(t, err, `no "> start"`)
	})

	t.Run("when a setup command is unknown, should fail naming its line", func(t *testing.T) {
		t.Parallel()

		_, err := newPlayer(data).Play("level 5\n> start\n")

		assert.EqualError(t, err, `setup "level 5": unknown setup command`)
	})

	t.Run("when a setup command has too few or too many numbers, should fail naming its line", func(t *testing.T) {
		t.Parallel()

		_, err := newPlayer(data).Play("badges 1 2\n> start\n")
		assert.EqualError(t, err, `setup "badges 1 2": want 1 numbers`)

		_, err = newPlayer(data).Play("party 4\n> start\n")
		assert.EqualError(t, err, `setup "party 4": want species, level and maybe XP`)
	})

	t.Run("when a number is malformed, should fail naming its line", func(t *testing.T) {
		t.Parallel()

		_, err := newPlayer(data).Play("Money lots\n> start\n")
		assert.EqualError(t, err, `setup "Money lots": strconv.ParseInt: parsing "lots": invalid syntax`)

		_, err = newPlayer(data).Play("party 4 5\n> start\n> choose one\n")
		assert.EqualError(t, err, `> choose one: strconv.ParseInt: parsing "one": invalid syntax`)
	})

	t.Run("when an action is unknown, should fail naming its line", func(t *testing.T) {
		t.Parallel()

		_, err := newPlayer(data).Play("party 4 5\n> start\n> jump 3\n")

		assert.EqualError(t, err, `> jump 3: unknown command "jump"`)
	})

	t.Run("when the port won't do an action the original did, should fail naming it", func(t *testing.T) {
		t.Parallel()

		for action, want := range map[string]string{
			"choose 2":            "party member 2 can't be clicked",
			"end":                 `"End Fight" is hidden`,
			"click Form9.Button1": "Form9.Button1 is hidden or disabled",
		} {
			_, err := newPlayer(data).Play("party 4 5\n> start\n> " + action + "\n")
			assert.EqualError(t, err, "> "+action+": "+want, "%s", action)
		}
	})

	t.Run("when the game quits with nothing to save, should fail", func(t *testing.T) {
		t.Parallel()

		_, err := newPlayer(data).Play("startscreen\n> start\n> quit\n")

		assert.EqualError(t, err, "> quit: nothing to save")
	})

	// Answer stops the scenario with Fail, from inside the game's call;
	// Play recovers it as the scenario's error.
	t.Run("when a yes or no question has no answer queued, should stop with Fail's error", func(t *testing.T) {
		t.Parallel()

		src := "party 4 5\n" +
			"> start\n" +
			"> seed 5\n" +
			"> menu Actions > Fight a wild POKéMON > Search for a POKéMON\n" +
			"> choose 1\n"

		_, err := newPlayer(data).Play(src)

		assert.EqualError(t, err, `no answer queued for "Choose Charmander?"`)
	})
}
