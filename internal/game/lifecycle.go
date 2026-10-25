// SPDX-License-Identifier: AGPL-3.0-only

package game

import "github.com/jsperandio/pokemon-simulator/internal/save"

// Starting, loading and quitting.

// Starters are the starter choice's species, in its order.
var Starters = [3]int{1, 4, 7} // Bulbasaur, Charmander, Squirtle

// NewGame is the start screen's "Start a new game!", once the player has
// chosen the save file and closed the name dialog. With OK (ok), the name
// typed is the player's, "Ash" if empty. Closed with Alt+F4, the dialog
// sets nothing: the player has no name, and the game window keeps its
// caption. The player picks a starter next, each of which may be shiny.
func (g *Game) NewGame(file, name string, ok bool) {
	g.SaveFile = file
	g.MinLevel, g.MaxLevel = 2, 100
	g.PokeBalls = 3
	g.GreatBalls, g.UltraBalls, g.MasterBalls, g.Potions, g.SuperPotions = 0, 0, 0, 0, 0
	g.Start.Visible = false

	if ok {
		g.setPlayer(name)
	}

	g.Main.Visible = true
	g.show(MainForm)

	g.InPokedex = [save.NumSpecies + 1]bool{}
	g.Main.Starters = true
	g.enable(false, "File1", "Action1")

	for i := range g.Main.StarterShiny {
		g.Main.StarterShiny[i] = g.rollShiny()
	}
}

// setPlayer is the name dialog's OK (TOKRightDlg.OKBtnClick).
func (g *Game) setPlayer(name string) {
	if name == "" {
		name = "Ash"
	}

	g.PlayerName = name
	g.Main.Caption = Title + " - " + name + " [" + fileName(g.SaveFile) + "]"
	g.Stats.Name = name
}

// ChooseStarter is a click on starter k (0-based, in Starters).
func (g *Game) ChooseStarter(k int) {
	if !g.Main.Starters || k < 0 || k >= len(Starters) {
		return
	}

	s, shiny := Starters[k], g.Main.StarterShiny[k]
	g.AddPKMN(s, g.name(s), 5, shiny)

	if shiny {
		g.log(Shiny, "It's a shiny POKéMON!")
	}

	g.Main.Starters = false
	g.enable(true, "File1", "Action1")

	g.Stats.Visible = true
	g.show(StatsForm)
	g.show(MainForm)
}

// Load starts the game saved in b, read from file: from the start screen's
// "Load a saved game...", or the file the game was started with
// (TForm1.LoadGame). Unlike the original, which stops half-way through a
// damaged file, it changes nothing then.
func (g *Game) Load(file string, b []byte) error {
	f, err := save.Read(b)
	if err != nil {
		return err
	}

	g.MinLevel, g.MaxLevel = 2, 100
	g.SaveFile = file
	g.File = *f
	g.Main.Caption = Title + " - " + g.PlayerName + " [" + fileName(file) + "]"
	g.Stats.Name = g.PlayerName
	g.unlockAll()

	g.Stats.Visible = true
	g.Main.Visible = true
	g.show(StatsForm)
	g.show(MainForm)

	g.Start.Visible = false

	return nil
}

// Close quits the game (TForm4.FormClose, also File → Quit): a tournament
// in progress costs XP, as fleeing does. It returns the save to write to
// SaveFile, or nil before a starter is chosen.
func (g *Game) Close() []byte {
	if g.InTournament {
		g.quitTournament()
	}

	if g.Main.Starters || !g.Main.Visible {
		return nil
	}

	return g.Bytes()
}

// Exported ends File → Export POKéMON information..., once the user
// interface has written the page to file: it asks whether the player
// wants to view it now.
func (g *Game) Exported(file string) bool {
	return g.UI.Message(Information, true, "POKéMON Information exported to "+file+". View file now?")
}
