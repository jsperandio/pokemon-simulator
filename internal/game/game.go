// SPDX-License-Identifier: AGPL-3.0-only

// Package game is the original game's logic: the player's progress,
// battles, trainers, tournaments, items and the side screens, with every
// rule, message and random draw as POKéMON Simulator V4.5 had them.
//
// It has no user interface. Where the original's logic reads its own
// windows (is "End Fight" showing? which item is selected?), that window
// state is part of the game's state: the user interface draws it and calls
// the handlers, which are named after the original's. A handler runs to
// completion and asks the user interface for dialogs as it goes, waiting
// for the answer, as the original's modal dialogs did.
package game

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/save"
	"github.com/jsperandio/pokemon-simulator/internal/version"
)

// ShinyOdds is the chance of a shiny POKéMON, 1 in ShinyOdds: the games'
// since POKéMON X and Y. The original has no shinies.
const ShinyOdds int = 4096

// RandomShiny rolls for a shiny POKéMON at ShinyOdds, with a generator of
// its own: the original's draws, which Rand replays, stay as they were.
func RandomShiny() bool {
	return rand.IntN(ShinyOdds) == 0 //nolint:gosec // a game's chance, not a secret
}

// DialogKind is a message dialog's icon and title (Delphi's TMsgDlgType).
type DialogKind int

// Dialog kinds, in TMsgDlgType order.
const (
	Warning DialogKind = iota
	Error
	Information
	Confirmation
	Custom
)

// UI is what the game asks of its user interface.
type UI interface {
	// Message shows a message dialog with an OK button, or Yes and No
	// buttons when yesNo is set, and waits for the player: it returns
	// true for OK or Yes.
	Message(kind DialogKind, yesNo bool, text string) bool
}

// Game is a game in progress: the player's progress, as saved, and the
// state of the battle and the windows.
type Game struct {
	save.File // the player's progress

	Data *gamedata.Data
	Rand *delphi.Rand
	UI   UI

	// OnLog, if set, sees every line added to Battle Info.
	OnLog func(line string)

	// OnShow, if set, sees every window the original shows with Show, in
	// order: Show brings a window to the front and activates it, even one
	// that is showing already.
	OnShow func(w Window)

	// OnDraws, if set, sees the Game Corner's draws of a play, which the
	// original showed one by one before telling the result.
	OnDraws func(draws []int)

	// RollShiny, if set, tells whether a starter, or the POKéMON a search
	// or the Mewtwo fight meets, is shiny, such as RandomShiny. Unset, none
	// is.
	RollShiny func() bool

	// The battle. The party indexes are 1-based, as in the original;
	// 0 means none.
	InWildBattle, InTrainerBattle, InTournament bool
	BattleType                                  uint8 // what EndBattle does; not reset between battles
	OwnIndex                                    int   // the party member fighting
	OwnHP                                       int
	EnemySpecies                                int
	EnemyLevel                                  uint8
	EnemyHP                                     int
	EnemyShiny                                  bool
	MinLevel, MaxLevel                          uint8 // the levels allowed against trainers

	// Search Again's: how the wild battle was found, and whether the one
	// starting is its own, whose fighter comes out with no prompt.
	search search
	again  bool

	TournamentTrainers [8]uint8 // 1-based indexes into Data.TrainerNames
	TournamentSpecies  [8]int
	TournamentLevels   [8]uint8

	EvolveMode, ReleaseMode bool // the next double click on the party evolves or releases
	EvolveIndex             int  // the party member waiting for EvolveButton

	// SaveFile is where Close saves: the file chosen for a new game, or
	// the one loaded.
	SaveFile string

	Menu       []*MenuItem // the game window's main menu
	Start      StartScreen
	Main       MainWindow
	Stats      StatsWindow
	Info       InfoWindow
	Pokedex    PokedexWindow
	Pokemart   PokemartWindow
	Types      TypesWindow
	GameCorner GameCornerWindow

	menu map[string]*MenuItem // Menu by item name
}

// Window is one of the original's windows, for OnShow.
type Window int

// The windows the game shows.
const (
	MainForm       Window = iota // the game window (Form4)
	StatsForm                    // Statistics (Form5)
	InfoForm                     // Battle Info (Form7)
	PokedexForm                  // the POKéDEX (Form8)
	PokemartForm                 // the Mart (Form9)
	TypesForm                    // Type Comparisons (FormA)
	GameCornerForm               // the Game Corner (FormB)
)

// show tells the user interface that the original shows window w here.
func (g *Game) show(w Window) {
	if g.OnShow != nil {
		g.OnShow(w)
	}
}

// StartScreen is the start screen (Form1).
type StartScreen struct {
	Visible bool
}

// MainWindow is the game window (Form4).
type MainWindow struct {
	Visible bool
	Caption string

	Starters     bool    // the starter choice (ListView1 and Label1)
	StarterShiny [3]bool // which starters are shiny, each rolled for a new game

	Battle      bool // the battle arena (GroupBox1)
	Go          bool // "Go!" (Button1)
	EndFight    bool // "End Fight" (EndButton)
	Attacks     bool // the fighter's attacks and "Run!" (RadioGroup1)
	AttackIndex int  // RadioGroup1.ItemIndex: 0–3 an attack, 4 "Run!", -1 none
	EnemyImage  int  // ListView3's image: the enemy's species − 1
	Appeared    int  // the wild POKéMON that have appeared, each one animated as it does
	Chosen      int  // the fighters the player has sent out, each one animated as it comes in

	// What the arena shows, set where the original sets its labels and
	// gauges, so it can lag the battle: a Potion's HP shows only once the
	// enemy has attacked, and the enemy's attacks only once the player has
	// read "Select a POKéMON".
	Enemy      Shown // Label3, Label9, Gauge2's progress and GroupBox2's caption
	EnemyMoves Shown // whose attacks Label4–7 list; Gauge2 sits after its Label3
	Own        Shown // Label2, Label8, Gauge1, ListView2 and RadioGroup1's attacks
	Pokeball   bool  // Image1: the wild POKéMON is already in the POKéDEX

	// The HP gauges' colours (Gauge2, Gauge1): the attacks set them, and
	// the end of a battle turns both green.
	EnemyColor, OwnColor GaugeColor

	// The Mewtwo-power picker (ComboBox1, SpinEdit1 and Button2–4).
	Picker      bool
	PickSpecies int // ComboBox1.ItemIndex: species − 1, or -1
	PickLevel   int // SpinEdit1.Value

	// The branched evolution choice (EvolveChoice and EvolveButton),
	// shown while Choices isn't empty; Choices[0] is "Don't evolve".
	Choices []string
	Choice  int // EvolveChoice.ItemIndex
}

// Shown is a fighter as the arena shows it. Species 0 is none: the
// arena's design-time text, or no fighter chosen.
type Shown struct {
	Species, Level int
	HP             int // of Level × 10
	Shiny          bool
}

// GaugeColor is an HP gauge's colour.
type GaugeColor int

// Gauge colours.
const (
	Green GaugeColor = iota
	Yellow
	Red
)

// gaugeColor is the colour an attack gives an HP gauge: yellow at half
// the maximum or less, red at a quarter or less. The original divides in
// Single; with whole HP up to 1000, comparing products is the same.
func gaugeColor(hp, level int) GaugeColor {
	switch maxHP := level * 10; {
	case 4*hp <= maxHP:
		return Red
	case 2*hp <= maxHP:
		return Yellow
	}

	return Green
}

// StatsWindow is the Statistics window (Form5): the party and the items.
type StatsWindow struct {
	Visible      bool
	Name         string // PlayerNameLbl
	PartyEnabled bool   // the party list (ListView1) takes double clicks
	Items        bool   // the item buttons (Button1–6) are shown
	ItemEnabled  [6]bool
}

// LineKind is what a line of Battle Info tells, which the port colours it
// by; the original's lines all look alike.
type LineKind int

// The kinds of Battle Info's lines.
const (
	Plain            LineKind = iota // the rest: who comes out, the moves and how effective they are, the balls
	DamageDealt                      // the fighter's damage to the enemy
	DamageTaken                      // the enemy's damage to the fighter
	CriticalDealt                    // the fighter's critical hit
	CriticalTaken                    // the enemy's critical hit
	EnemyFainted                     // the enemy fainting
	OwnFainted                       // the fighter fainting
	ExperienceGained                 // a win's experience
	LevelUp                          // a level a win gives
	PrizeMoney                       // a win's money
	MoneyLost                        // a loss's money
	ExperienceLost                   // quitting a tournament
	LevelLost                        // a level that quitting a tournament takes
	Caught                           // a ball catching the enemy
	Registered                       // a species added to the POKéDEX
	Healed                           // a Potion's HP
	Shiny                            // "It's a shiny POKéMON!"
)

// InfoWindow is Battle Info (Form7), the battle log.
type InfoWindow struct {
	Visible bool
	Lines   []string
	Kinds   []LineKind // each line's, as long as Lines
}

// Party returns the party's members, as the Statistics window lists them.
func (g *Game) Party() []save.Slot {
	return g.File.Party()
}

// New returns a game at the start screen, with the window state of the
// original's forms when they are created. The random generator is seeded
// from the clock, as the original's Randomize did.
func New(data *gamedata.Data, ui UI) *Game {
	gam := &Game{
		Data: data,
		UI:   ui,
		Rand: &delphi.Rand{},
	}

	gam.Rand.Randomize()

	gam.Start.Visible = true

	gam.Main = MainWindow{
		Caption:     version.Title(),
		AttackIndex: -1,
		PickSpecies: -1,
		PickLevel:   100,
	}

	gam.Stats.PartyEnabled = true
	gam.Stats.ItemEnabled = [6]bool{true, true, true, true, true, true}

	gam.Info.Lines = []string{""} // Memo1's design-time text
	gam.Info.Kinds = []LineKind{Plain}

	gam.Pokedex.Index = -1
	gam.Pokemart.Item = -1
	gam.Types.Species, gam.Types.Attack = -1, -1
	gam.GameCorner.Prize, gam.GameCorner.Number, gam.GameCorner.Drawn = -1, -1, -1
	gam.buildMenu()

	return gam
}

// species returns species s, or a nameless placeholder for an index the
// original would read out of its tables.
func (g *Game) species(s int) *gamedata.Species {
	if spc := g.Data.SpeciesByID(s); spc != nil {
		return spc
	}

	return &gamedata.Species{ID: s}
}

// name is SpeciesNames[s].
func (g *Game) name(s int) string {
	return g.species(s).Name
}

// member returns party member i (1-based).
func (g *Game) member(i int) *save.Slot {
	return &g.Owned[i-1]
}

// log adds a line of kind kind to Battle Info.
func (g *Game) log(kind LineKind, line string) {
	g.Info.Lines = append(g.Info.Lines, line)
	g.Info.Kinds = append(g.Info.Kinds, kind)

	if g.OnLog != nil {
		g.OnLog(line)
	}
}

func (g *Game) logf(kind LineKind, format string, a ...any) {
	g.log(kind, fmt.Sprintf(format, a...))
}

// message shows a dialog with an OK button.
func (g *Game) message(kind DialogKind, text string) {
	g.UI.Message(kind, false, text)
}

// confirm asks a Yes/No question (mtConfirmation).
func (g *Game) confirm(text string) bool {
	return g.UI.Message(Confirmation, true, text)
}

// showInfo shows Battle Info; a hidden window clears its log as it shows
// (TForm7.FormShow).
func (g *Game) showInfo() {
	if !g.Info.Visible {
		g.Info.Visible = true
		g.Info.Lines = nil
		g.Info.Kinds = nil
	}

	g.show(InfoForm)
}

// CloseInfo is Alt+F4 on Battle Info (TForm7.Close): the window hides
// until the next battle shows it.
func (g *Game) CloseInfo() {
	g.Info.Visible = false
}

// CloseStats is Alt+F4 on Statistics (TForm5.Close): the window hides, and
// only loading a game or choosing a starter shows it again. In a game that
// goes on, the player can't choose a POKéMON any more.
func (g *Game) CloseStats() {
	g.Stats.Visible = false
}

// partyHas reports whether species s is in the party.
func (g *Game) partyHas(s int) bool {
	return slices.ContainsFunc(g.Party(), func(m save.Slot) bool {
		return int(m.Species) == s
	})
}

// partyHasKind reports whether species s is in the party, shiny or not as
// shiny says.
func (g *Game) partyHasKind(s int, shiny bool) bool {
	return slices.ContainsFunc(g.Party(), func(m save.Slot) bool {
		return int(m.Species) == s && m.Shiny == shiny
	})
}

// rollShiny tells whether a starter, or the POKéMON met, is shiny.
func (g *Game) rollShiny() bool {
	return g.RollShiny != nil && g.RollShiny()
}

// shinyName is a party member's name in the details and the export: the
// species', with "(shiny)" after a shiny one's.
func (g *Game) shinyName(m save.Slot) string {
	name := g.name(int(m.Species))
	if m.Shiny {
		name += " (shiny)"
	}

	return name
}

// levelFor returns the level that xp reaches from level: levels only go up,
// one at a time while (level+1)³ XP is reached. The original has no bound
// and hangs on a damaged save's huge XP; the port stops at 255.
func levelFor(level uint8, xp uint32) uint8 {
	for level < 255 {
		n := uint64(level) + 1

		if n*n*n > uint64(xp) {
			break
		}

		level++
	}

	return level
}

// recomputeLevels brings every party member's level up to its XP, with
// the 1,000,000 XP cap (EndBattle after a loss, the end of a tournament).
func (g *Game) recomputeLevels() {
	for i := range g.Party() {
		m := &g.Owned[i]
		m.Level = levelFor(m.Level, m.XP)

		if m.XP > maxXP {
			m.XP, m.Level = maxXP, 100
		}
	}
}

// maxXP is level 100's XP, the most a POKéMON can have.
const maxXP uint32 = 1000000

// fileName is a path's last element, as the original cuts a file name
// after its last "\"; the port accepts "/" too.
func fileName(path string) string {
	return path[strings.LastIndexAny(path, `\/`)+1:]
}
