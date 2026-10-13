// SPDX-License-Identifier: AGPL-3.0-only

package game

// MenuItem is an item of the game window's main menu (Form4.MainMenu1).
type MenuItem struct {
	Name    string // the original's component name, e.g. "Brock1"
	Caption string // as designed; "&&" stands for "&"
	Enabled bool
	Visible bool
	Checked bool
	Items   []*MenuItem
}

// Item returns the menu item with the given name, or nil.
func (g *Game) Item(name string) *MenuItem { return g.menu[name] }

// ClickMenu runs a menu command, if its item and the items above it are
// enabled and visible. The File, Music and Help commands other than
// "Repeat Music" belong to the user interface: ClickMenu reports false for
// them, and for unknown or unavailable items.
func (g *Game) ClickMenu(name string) bool {
	if !g.available(name) {
		return false
	}

	if t, ok := trainers[name]; ok {
		g.fightTrainer(name, t)

		return true
	}

	for tier, t := range tiers {
		if t.item == name {
			g.enterTournament(tier)

			return true
		}
	}

	switch name {
	case "FindPKMN1":
		g.FindPKMN()
	case "Mewtwo1":
		g.StartWildBattle(150, 100)
	case "Buy1":
		g.OpenMart()
	case "Evolve1":
		g.toggleEvolveMode()
	case "ReleasePKMN1":
		g.toggleReleaseMode()
	case "ViewDEX1":
		g.OpenDex()
	case "TypeComparisons1":
		g.OpenTypes()
	case "GameCorner1":
		g.OpenCorner()
	case "RepeatMusic1", "AlwaysshowattacksinPOKDEX1":
		m := g.menu[name]
		m.Checked = !m.Checked
	case "MakeVisible1":
		g.menu[name].Enabled = false // and nothing else
	default:
		return false
	}

	return true
}

// available reports whether a menu item can be clicked: it and every item
// above it are enabled and visible.
func (g *Game) available(name string) bool {
	var find func(items []*MenuItem) bool
	find = func(items []*MenuItem) bool {
		for _, m := range items {
			if !m.Enabled || !m.Visible {
				continue
			}

			if m.Name == name || find(m.Items) {
				return true
			}
		}

		return false
	}

	return find(g.Menu)
}

// enable sets the Enabled flag of the named items.
func (g *Game) enable(on bool, names ...string) {
	for _, n := range names {
		g.menu[n].Enabled = on
	}
}

// The menu's flags as designed (the DFM).
const (
	disabled = 1 << iota
	hidden
	checked
)

type menuDef struct {
	name, caption string
	flags         int
	items         []menuDef
}

// buildMenu creates the main menu as designed.
func (g *Game) buildMenu() {
	g.menu = map[string]*MenuItem{}

	var build func(defs []menuDef) []*MenuItem
	build = func(defs []menuDef) []*MenuItem {
		items := make([]*MenuItem, 0, len(defs))
		for _, d := range defs {
			m := &MenuItem{
				Name:    d.name,
				Caption: d.caption,
				Enabled: d.flags&disabled == 0,
				Visible: d.flags&hidden == 0,
				Checked: d.flags&checked != 0,
				Items:   build(d.items),
			}
			g.menu[m.Name] = m
			items = append(items, m)
		}

		return items
	}
	g.Menu = build(mainMenu)
}

// mainMenu is Form4's menu as designed. Every command item's handler is
// its name + "Click" in the original.
var mainMenu = []menuDef{
	{"File1", "File", 0, []menuDef{
		{"ExportPOKMONinformation1", "Export POKéMON information...", 0, nil},
		{"Quit1", "Quit", 0, nil},
	}},
	{"Action1", "Actions", 0, []menuDef{
		{"FightTrainer1", "Fight a Trainer", 0, []menuDef{
			{"Trainers1", "Trainers", 0, []menuDef{
				{"Beginner1", "Beginner", 0, []menuDef{
					{"Red1", "Red", 0, nil},
					{"Blue1", "Blue", 0, nil},
					{"Green1", "Green", 0, nil},
					{"Yellow1", "Yellow", 0, nil},
					{"Gold1", "Gold", 0, nil},
					{"Silver1", "Silver", 0, nil},
					{"Crystal1", "Crystal", 0, nil},
				}},
				{"Novice", "Novice", 0, []menuDef{
					{"Ruby1", "Ruby", 0, nil},
					{"Sapphire1", "Sapphire", 0, nil},
					{"Emerald1", "Emerald", 0, nil},
				}},
				{"Intermediate1", "Intermediate", 0, []menuDef{
					{"ProfOak1", "Prof.Oak", 0, nil},
					{"ProfElm1", "Prof.Elm", 0, nil},
					{"ProfBirch1", "Prof.Birch", 0, nil},
				}},
				{"Advanced1", "Advanced", 0, []menuDef{
					{"Jesse1", "Jesse", 0, nil},
					{"James1", "James", 0, nil},
				}},
				{"Expert1", "Expert", 0, []menuDef{
					{"Giovanni1", "Giovanni", 0, nil},
				}},
			}},
			{"KantoGymLeaders1", "Kanto Gym Leaders", 0, []menuDef{
				{"Brock1", "Brock", 0, nil},
				{"Misty1", "Misty", disabled, nil},
				{"LtSurge1", "Lt. Surge", disabled, nil},
				{"Erika1", "Erika", disabled, nil},
				{"Sabrina1", "Sabrina", disabled, nil},
				{"Janine1", "Janine", disabled, nil},
				{"Blaine1", "Blaine", disabled, nil},
				{"Gary1", "Gary", disabled, nil},
			}},
			{"EliteFour1", "G/S/C Elite Four", disabled, nil},
			{"JohtoGymLeaders1", "Johto Gym Leaders", disabled, []menuDef{
				{"Falkner1", "Falkner", 0, nil},
				{"Bugsy1", "Bugsy", disabled, nil},
				{"Whitney1", "Whitney", disabled, nil},
				{"Morty1", "Morty", disabled, nil},
				{"Chuck1", "Chuck", disabled, nil},
				{"Jasmine1", "Jasmine", disabled, nil},
				{"Pryce1", "Pryce", disabled, nil},
				{"Clair1", "Clair", disabled, nil},
			}},
			{"Ash1", "Ash", disabled, nil},
			{"Me1", "Me", disabled, nil},
			{"HoennGymLeaders1", "Hoenn Gym Leaders (L.100)", disabled, []menuDef{
				{"Roxanne1", "Roxanne", 0, nil},
				{"Brawly1", "Brawly", disabled, nil},
				{"Wattson1", "Wattson", disabled, nil},
				{"Flannery1", "Flannery", disabled, nil},
				{"Norman1", "Norman", disabled, nil},
				{"Winona1", "Winona", disabled, nil},
				{"TateLiza1", "Tate && Liza", disabled, nil},
				{"Wallace1", "Wallace", disabled, nil},
			}},
			{"Elites1", "Other Elites", disabled, []menuDef{
				{"RBYEliteFour1", "R/B/Y Elite Four", 0, nil},
				{"RuSaEliteFour1", "Ru/Sa Elite Four", 0, nil},
			}},
		}},
		{"FightWild1", "Fight a wild POKéMON", 0, []menuDef{
			{"FindPKMN1", "Search for a POKéMON", 0, nil},
			{"Mewtwo1", "Mewtwo", disabled, nil},
		}},
		{"Tournaments1", "Tournaments", 0, []menuDef{
			{"VeryEasyLevels2101", "Very Easy (Levels 2-10)", 0, nil},
			{"EasyLevels10301", "Easy (Levels 10-30)", 0, nil},
			{"IntermediateLevels30501", "Intermediate (Levels 30-50)", 0, nil},
			{"HardLevels50701", "Hard (Levels 50-70)", 0, nil},
			{"VeryHardLevels70991", "Very Hard (Levels 70-99)", 0, nil},
			{"PrimeCupLevel1001", "Prime Cup (Level 100)", 0, nil},
		}},
		{"Buy1", "Buy Items", 0, nil},
		{"Evolve1", "Evolve a POKéMON", 0, nil},
		{"ReleasePKMN1", "Release a POKéMON", 0, nil},
		{"ViewDEX1", "View POKéDEX", 0, nil},
		{"TypeComparisons1", "Type Comparisons", 0, nil},
		{"GameCorner1", "Game Corner", 0, nil},
	}},
	{"SelectMusic1", "Music", 0, []menuDef{
		{"Openfile1", "Open file...", 0, nil},
		{"RepeatMusic1", "Repeat Music", checked, nil},
		{"StopMusic1", "Stop Music", 0, nil},
	}},
	{"Help1", "Help", 0, []menuDef{
		{"Contents1", "Contents", 0, nil},
		{"About1", "About...", 0, nil},
	}},
	// Hidden for good: nothing shows it, and F2 only disables its first
	// item (notes/misc.md).
	{"Debug1", "Debug", hidden, []menuDef{
		{"MakeVisible1", "Make Visible", 0, nil},
		{"AlwaysshowattacksinPOKDEX1", "Always show attacks in POKéDEX", 0, nil},
	}},
}
