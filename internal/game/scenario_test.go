// SPDX-License-Identifier: AGPL-3.0-only

package game_test

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/game/scenario"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
)

// With -saves <dir>, the save each scenario quits with is written to
// <dir>/<scenario>.pgs, for the original to load (portsave).
var saveDir = flag.String("saves", "", "write the save each scenario quits with to this directory")

// Test_Scenarios plays the traces of the original in testdata/scenarios on
// the game's handlers, which must produce the same lines (package
// scenario). Each scenario is a subtest named after its file, so
// -run 'Test_Scenarios/gym-brock' plays one.
func Test_Scenarios(t *testing.T) {
	t.Parallel()

	raw, err := assets.FS.ReadFile("pokegame.dat")
	require.NoError(t, err)

	data, err := gamedata.Load(raw)
	require.NoError(t, err)

	direct := func(*testing.T) scenario.Driver {
		return &scenario.Direct{}
	}
	scenario.Run(t, filepath.Join("testdata", "scenarios"), data, direct, *saveDir)
}
