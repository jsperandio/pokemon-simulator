// SPDX-License-Identifier: AGPL-3.0-only

package layout

import (
	"slices"
	"testing"

	"github.com/jsperandio/pokemon-simulator/assets"
)

func TestAllForms(t *testing.T) {
	if len(All) != 11 {
		t.Fatalf("got %d forms, want 11", len(All))
	}

	seen := map[string]bool{}
	for _, f := range All {
		if seen[f.Name] {
			t.Errorf("form %s listed twice", f.Name)
		}

		seen[f.Name] = true
		if f.Class != "T"+f.Name {
			t.Errorf("form %s has class %s", f.Name, f.Class)
		}
	}
}

// The original places its three main windows at fixed screen positions:
// Battle Info under the main window, Statistics to its right.
func TestWindowArrangement(t *testing.T) {
	checks := []struct {
		form   *Component
		x, y   int
		cw, ch string
	}{
		{&Form4, 189, 105, "422", "270"},
		{&Form7, 189, 419, "422", "159"},
		{&Form5, 617, 105, "177", "473"},
	}
	for _, c := range checks {
		b := c.form.Bounds
		if b.X != c.x || b.Y != c.y || c.form.Props["ClientWidth"] != c.cw || c.form.Props["ClientHeight"] != c.ch {
			t.Errorf("%s at (%d,%d) %sx%s, want (%d,%d) %sx%s", c.form.Name, b.X, b.Y,
				c.form.Props["ClientWidth"], c.form.Props["ClientHeight"], c.x, c.y, c.cw, c.ch)
		}
	}
}

func TestMainMenu(t *testing.T) {
	items := 0

	Form4.Walk(func(c *Component) {
		if c.Class == "TMenuItem" {
			items++
		}
	})

	if items != 86 {
		t.Errorf("got %d menu items, want 86", items)
	}

	fight := Form4.Find("FightTrainer1")
	if fight == nil {
		t.Fatal("no FightTrainer1 menu")
	}

	var trainers []string

	fight.Walk(func(c *Component) {
		if c.Class == "TMenuItem" && c.Events["OnClick"] != "" {
			trainers = append(trainers, c.Caption)
		}
	})

	if len(trainers) != 45 {
		t.Errorf("got %d trainers, want 45: %v", len(trainers), trainers)
	}

	for _, name := range []string{
		"Brock",
		"Gary",
		"Clair",
		"Wallace",
		"Tate && Liza",
		"Me",
	} {
		if !slices.Contains(trainers, name) {
			t.Errorf("trainer %q missing", name)
		}
	}
}

func TestComponents(t *testing.T) {
	starters := Form4.Find("ListView1")
	want := []ListItem{{"Bulbasaur", 0}, {"Charmander", 3}, {"Squirtle", 6}}

	if starters == nil || !slices.Equal(starters.List, want) {
		t.Errorf("starters: got %+v, want %+v", starters, want)
	}

	if b := Form4.Find("EvolveButton"); b == nil || b.Events["OnClick"] != "EvolveButtonClick" || !b.Hidden {
		t.Errorf("EvolveButton: got %+v", b)
	}

	if e := OKRightDlg.Find("Edit1"); e == nil || e.Caption != "Ash" {
		t.Errorf("name dialog default: got %+v", e)
	}

	if Form4.Find("NoSuchComponent") != nil {
		t.Error("Find returned a component for an unknown name")
	}
}

// Image lists and pictures point at embedded asset files.
func TestAssetReferences(t *testing.T) {
	refs := 0

	for _, f := range All {
		f.Walk(func(c *Component) {
			for _, prop := range []string{"Bitmap", "Picture"} {
				path, ok := c.Props[prop]
				if !ok {
					continue
				}

				refs++

				if _, err := assets.FS.Open(path); err != nil {
					t.Errorf("%s.%s.%s: %v", f.Name, c.Name, prop, err)
				}
			}
		})
	}

	if refs != 5 {
		t.Errorf("got %d asset references, want 5", refs)
	}
}
