// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// silent answers every dialog with OK or Yes.
type silent struct{ messages []string }

func (s *silent) Message(kind DialogKind, yesNo bool, text string) bool {
	s.messages = append(s.messages, text)
	return true
}

func newGame(t *testing.T) (*Game, *silent) {
	t.Helper()
	ui := &silent{}
	return New(loadData(t), ui), ui
}

// The win percentage is computed in x87 Extended; the Statistics window
// showed 77.99 for 39 wins and 11 losses, and 33.33 for 1 and 2
// (captures/screens/stats-percent-*.png).
func TestWinPercent(t *testing.T) {
	g, _ := newGame(t)
	for _, c := range []struct {
		wins, losses uint32
		want         string
	}{{39, 11, "77.99"}, {1, 2, "33.33"}, {1, 0, "100"}, {0, 3, "0"}, {2, 1, "66.66"}} {
		g.Wins, g.Losses = c.wins, c.losses
		if got, ok := g.WinPercent(); !ok || got != c.want {
			t.Errorf("%d wins, %d losses: %q, want %q", c.wins, c.losses, got, c.want)
		}
	}
	g.Wins, g.Losses = 0, 0
	if _, ok := g.WinPercent(); ok {
		t.Error("a percentage of no fights")
	}
}

// Type Comparisons printed "0.5" and "0.25" for Razor Leaf on Charizard
// (captures/screens/screens-game-types.png).
func TestTypeMultipliers(t *testing.T) {
	g, _ := newGame(t)
	g.OpenTypes()
	if _, _, _, ok := g.TypeMultipliers(); ok {
		t.Error("multipliers with nothing selected")
	}
	g.SelectTypes(5, 3) // Charizard (Fire/Flying), Razor Leaf (Grass)
	a, b, total, ok := g.TypeMultipliers()
	if !ok || a != "0.5" || b != "0.5" || total != "0.25" {
		t.Errorf("Razor Leaf on Charizard: %s × %s = %s", a, b, total)
	}
}

// The POKéDEX count in the Statistics window counts species 1–251 only.
func TestDexCount(t *testing.T) {
	g, _ := newGame(t)
	for _, s := range []int{1, 150, 251, 252, 386} {
		g.InDex[s] = true
	}
	if n := g.DexCount(); n != 3 {
		t.Errorf("DexCount = %d, want 3", n)
	}
}

func TestDamage(t *testing.T) {
	for _, c := range []struct {
		r, power, level int
		crit            bool
		t1, t2          float64
		want            int
	}{
		{16, 70, 5, false, 1, 1, 9},        // wild-tackle-1: Bulbasaur's Tackle on Weedle, 9.5
		{0, 20, 2, false, 1, 1, 1},         // −9.6 truncates to −9, then the minimum
		{19, 100, 100, true, 2, 2, 872},    // 109 × 2 × 4
		{10, 50, 10, false, 0.5, 1, 2},     // 2.5
		{10, 50, 10, false, 0, 1, 1},       // immunity zeroes it later; the minimum first
		{19, 120, 100, false, 2, 0.5, 129}, // 129 × 2 × 0.5
	} {
		if got := Damage(c.r, c.power, c.level, c.crit, c.t1, c.t2); got != c.want {
			t.Errorf("Damage(%d, %d, %d, %t, %g, %g) = %d, want %d", c.r, c.power, c.level, c.crit, c.t1, c.t2, got, c.want)
		}
	}
}

// The prizes' effects are hard-coded; their names come from
// POKéGAME.dat. Each effect must be what its name says.
func TestTournamentPrizes(t *testing.T) {
	for tier := 1; tier <= 6; tier++ {
		for roll := 1; roll <= 3; roll++ {
			g, ui := newGame(t)
			g.Owned[0].Species, g.Owned[0].Level, g.Owned[0].XP = 25, 50, 125000
			g.BattleType = uint8(tier*10 + 8)
			before := g.File
			// Find a seed whose first Random(3) is the roll.
			for g.Rand.Seed = 0; ; g.Rand.Seed++ {
				seed := g.Rand.Seed
				if g.Rand.Random(3)+1 == roll {
					g.Rand.Seed = seed
					break
				}
			}
			g.handleTournament()
			prize := g.prize(tier, roll)
			if !strings.Contains(strings.Join(ui.messages, "\n"), "You won "+prize+"!") {
				t.Errorf("tier %d roll %d: no \"You won %s!\" in %q", tier, roll, prize, ui.messages)
			}
			got := map[string]int{
				"¥":             int(g.Money) - int(before.Money),
				" POKéBALL":     int(g.PokeBalls) - int(before.PokeBalls),
				" Great Ball":   int(g.GreatBalls) - int(before.GreatBalls),
				" Ultra Ball":   int(g.UltraBalls) - int(before.UltraBalls),
				" Master Ball":  int(g.MasterBalls) - int(before.MasterBalls),
				" Potion":       int(g.Potions) - int(before.Potions),
				" Super Potion": int(g.SuperPotions) - int(before.SuperPotions),
				" POKéMON":      len(g.Party()) - len(before.Party()),
			}
			want := map[string]int{}
			if prize == "a random POKéMON at a random level" {
				want[" POKéMON"] = 1
			} else {
				n, unit, _ := strings.Cut(strings.TrimSuffix(strings.TrimSuffix(prize, "s"), "¥"), " ")
				if strings.HasSuffix(prize, "¥") {
					n, unit = strings.TrimSuffix(prize, "¥"), ""
				}
				v, err := strconv.Atoi(n)
				if err != nil {
					t.Fatalf("tier %d roll %d: prize %q", tier, roll, prize)
				}
				key := " " + unit
				if unit == "" {
					key = "¥"
				}
				want[key] = v
			}
			for k, v := range got {
				if v != want[k] {
					t.Errorf("tier %d roll %d (%s): %s changed by %d, want %d", tier, roll, prize, k, v, want[k])
				}
			}
		}
	}
}

func TestLevelFor(t *testing.T) {
	for _, c := range []struct {
		level uint8
		xp    uint32
		want  uint8
	}{{5, 125, 5}, {5, 215, 5}, {5, 216, 6}, {5, 1000, 10}, {30, 42392, 34}, {50, 0, 50}, {1, 4294967295, 255}} {
		if got := levelFor(c.level, c.xp); got != c.want {
			t.Errorf("levelFor(%d, %d) = %d, want %d", c.level, c.xp, got, c.want)
		}
	}
}

func TestMenu(t *testing.T) {
	g := New(nil, nil)
	n := 0
	var count func(items []*MenuItem)
	count = func(items []*MenuItem) {
		for _, m := range items {
			n++
			count(m.Items)
		}
	}
	count(g.Menu)
	if n != 86 {
		t.Errorf("%d menu items, want the original's 86", n)
	}
	for name, want := range map[string]bool{"Brock1": true, "Misty1": false, "Falkner1": false, "MakeVisible1": false, "Quit1": true} {
		if got := g.available(name); got != want {
			t.Errorf("%s available: %t, want %t", name, got, want)
		}
	}
	if g.ClickMenu("Misty1") || g.ClickMenu("NoSuchItem") || g.ClickMenu("Quit1") {
		t.Error("ClickMenu ran a disabled, unknown or user interface command")
	}
}

// The export scenario's page, as the original wrote it.
func TestExportPage(t *testing.T) {
	g, _ := newGame(t)
	f := save.File{PlayerName: "Probe", PokeBalls: 3}
	f.Owned[0] = save.Slot{Species: 1, Level: 5, XP: 125}
	f.Owned[1] = save.Slot{Species: 150, Level: 100, XP: 1000000}
	f.Owned[2] = save.Slot{Species: 252, Level: 50, XP: 130000}
	for _, s := range []int{1, 7, 150, 252} {
		f.InDex[s] = true
	}
	if err := g.Load(`C:\Temp\probe-export.pgs`, f.Bytes()); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../export/testdata/export.htm")
	if err != nil {
		t.Fatal(err)
	}
	if got := g.ExportPage(time.Date(2026, 10, 11, 17, 32, 51, 0, time.Local)); !bytes.Equal(got, want) {
		t.Errorf("the page differs from the original's:\n%s", got)
	}
}
