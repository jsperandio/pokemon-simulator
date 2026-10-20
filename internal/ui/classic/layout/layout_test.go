// SPDX-License-Identifier: AGPL-3.0-only

package layout

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/pokemon-simulator/assets"
)

func Test_All(t *testing.T) {
	t.Parallel()

	t.Run("when the forms are listed, should be 11, each listed once", func(t *testing.T) {
		t.Parallel()

		require.Len(t, All, 11)

		seen := map[string]bool{}
		for _, frm := range All {
			assert.False(t, seen[frm.Name], "form %s listed twice", frm.Name)

			seen[frm.Name] = true
		}
	})

	t.Run("when a form is listed, should have the class T and its name", func(t *testing.T) {
		t.Parallel()

		for _, frm := range All {
			assert.Equal(t, "T"+frm.Name, frm.Class, "form %s", frm.Name)
		}
	})

	t.Run("when the main windows are listed, should be placed as in the original", func(t *testing.T) {
		t.Parallel()

		// The original places its three main windows at fixed screen positions:
		// Battle Info under the main window, Statistics to its right.
		for _, cas := range []struct {
			form   *Component
			x, y   int
			cw, ch string
		}{
			{&Form4, 189, 105, "422", "270"},
			{&Form7, 189, 419, "422", "159"},
			{&Form5, 617, 105, "177", "473"},
		} {
			assert.Equal(t, cas.x, cas.form.Bounds.X, "%s left edge", cas.form.Name)
			assert.Equal(t, cas.y, cas.form.Bounds.Y, "%s top edge", cas.form.Name)
			assert.Equal(t, cas.cw, cas.form.Props["ClientWidth"], "%s client width", cas.form.Name)
			assert.Equal(t, cas.ch, cas.form.Props["ClientHeight"], "%s client height", cas.form.Name)
		}
	})

	t.Run("when the bitmaps and pictures are listed, should be embedded files", func(t *testing.T) {
		t.Parallel()

		// Image lists and pictures point at embedded asset files.
		var pths []string

		for _, frm := range All {
			frm.Walk(func(cmp *Component) {
				for _, prop := range []string{"Bitmap", "Picture"} {
					pth, ok := cmp.Props[prop]
					if !ok {
						continue
					}

					pths = append(pths, pth)
				}
			})
		}

		assert.Len(t, pths, 5, "asset references")

		for _, pth := range pths {
			_, err := fs.Stat(assets.FS, pth)
			assert.NoError(t, err, "%s", pth)
		}
	})
}

func Test_Form4(t *testing.T) {
	t.Parallel()

	t.Run("when the menu is walked, should hold 86 menu items", func(t *testing.T) {
		t.Parallel()

		items := 0

		Form4.Walk(func(cmp *Component) {
			if cmp.Class == "TMenuItem" {
				items++
			}
		})

		assert.Equal(t, 86, items, "menu items")
	})

	t.Run("when FightTrainer1 is walked, should hold 45 trainers", func(t *testing.T) {
		t.Parallel()

		assert.Len(t, fightTrainers(t), 45, "trainers")
	})

	t.Run("when FightTrainer1 is walked, should include the named trainers", func(t *testing.T) {
		t.Parallel()

		trainers := fightTrainers(t)

		for _, name := range []string{
			"Brock",
			"Gary",
			"Clair",
			"Wallace",
			"Tate && Liza",
			"Me",
		} {
			assert.Contains(t, trainers, name, "trainer %q", name)
		}
	})
}

// fightTrainers returns the captions of the trainers' menu items under
// FightTrainer1.
func fightTrainers(t *testing.T) []string {
	t.Helper()

	fight := Form4.Find("FightTrainer1")
	require.NotNil(t, fight, "no FightTrainer1 menu")

	var trainers []string

	fight.Walk(func(cmp *Component) {
		if cmp.Class == "TMenuItem" && cmp.Events["OnClick"] != "" {
			trainers = append(trainers, cmp.Caption)
		}
	})

	return trainers
}

func Test_Component_Find(t *testing.T) {
	t.Parallel()

	t.Run("when a list is found by name, should hold the three starters", func(t *testing.T) {
		t.Parallel()

		starters := Form4.Find("ListView1")
		require.NotNil(t, starters)

		want := []ListItem{{"Bulbasaur", 0}, {"Charmander", 3}, {"Squirtle", 6}}
		assert.Equal(t, want, starters.List)
	})

	t.Run("when a button is found, should be hidden with its click handler", func(t *testing.T) {
		t.Parallel()

		btn := Form4.Find("EvolveButton")
		require.NotNil(t, btn)

		assert.True(t, btn.Hidden)
		assert.Equal(t, "EvolveButtonClick", btn.Events["OnClick"])
	})

	t.Run("when the name dialog's edit box is found, should default to Ash", func(t *testing.T) {
		t.Parallel()

		edt := OKRightDlg.Find("Edit1")
		require.NotNil(t, edt)

		assert.Equal(t, "Ash", edt.Caption)
	})

	t.Run("when the name is not in the form, should give nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, Form4.Find("NoSuchComponent"))
	})
}
