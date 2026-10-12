// SPDX-License-Identifier: AGPL-3.0-only

// Package scenario plays traces of the original game on the port.
//
// The traces, in internal/game/testdata/scenarios, were recorded under
// Wine by a debugger that played them on the original: the setup, then
// each action ("> ...") followed by what the original did, in order: its
// Random calls, Battle Info lines, dialogs, variables ("state") and, on
// quit, the lines of the save it wrote that changed. A Player plays one on
// the port and records the same lines. It does the actions through a
// Driver: Direct calls the game's handlers, as the original's controls
// did, and a user interface's driver uses its controls.
//
// The setup commands describe the save a scenario starts from:
//
//	startscreen                    start at the start screen, with no save
//	version 1                      the save is in version 1
//	name <text>                    the player's name (default Probe)
//	badges <n>                     GymProgress
//	flags <5 × 0|1>                BeatLance BeatAsh BeatLink GotJirachi GotDeoxys
//	<counter> <n>                  Money … SuperPotions (PokeBalls is 3 by default)
//	dex <species>...               registered in the POKéDEX
//	party <species> <level> [<xp>] a party member, registered; XP defaults to level³
//	portsave <scenario>            start from the save the port writes quitting
//	                               that scenario instead: its trace is the
//	                               original loading the port's save
//
// The actions are the probe's; Player.do lists them.
package scenario

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jsperandio/pokemon-simulator/internal/delphi"
	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// Driver does a scenario's actions on the port. The Player checks that
// the original's player could do an action before it asks for it; the
// driver then does it and waits until the game waits for the player,
// answering the game's dialogs with Player.Answer and recording them with
// Player.Dialog.
type Driver interface {
	// Launch makes the game, calls hook with it, and starts it: on the
	// save in file, or at the start screen when save is nil. file is the
	// path the original had, `C:\Temp\probe-<scenario>.pgs`.
	Launch(p *Player, file string, save []byte, hook func(*game.Game)) error
	// Menu runs a main menu command, given by its items from the bar's.
	Menu(path []*game.MenuItem) error
	// Choose double-clicks party member i (0-based) in Statistics.
	Choose(i int) error
	// Attack chooses attack k (0–3), or Run! (4), and presses Go!.
	Attack(k int) error
	// Item presses item button k (0–5) in Statistics.
	Item(k int) error
	// End presses End Fight.
	End() error
	// Click presses a button, by form and name: "Form9.Button1". The
	// About box that Form1.AboutButton shows is closed with OK.
	Click(path string) error
	// Radio chooses item k (0-based) of a radio group.
	Radio(path string, k int) error
	// Combo chooses item i (0-based) of a combo box.
	Combo(path string, i int) error
	// Type types a number into a spin edit.
	Type(path string, n int) error
	// Text returns the game window's caption ("Form4") or the Mart's
	// total ("Form9.Edit1").
	Text(path string) (string, error)
	// New starts a new game from the start screen, saved in file, for
	// the name typed.
	New(file, name string) error
	// Load loads the game save, in file, from the start screen.
	Load(file string, save []byte) error
	// Starter chooses starter k (0-based).
	Starter(k int) error
	// Export exports the POKéMON information to file.
	Export(file string) error
	// Quit closes the game window, which ends the program, and returns
	// the save it wrote, nil if none.
	Quit() ([]byte, error)
}

// Player plays a scenario on the port and records the same trace the
// probe recorded on the original. It answers the dialogs as the probe did.
type Player struct {
	Name      string // the scenario's
	Dir       string // the scenarios' folder, for portsave
	Data      *gamedata.Data
	NewDriver func() Driver       // the driver for this scenario, and for those portsave plays
	Saves     string              // if not "", the folder for the save the scenario quits with
	After     func(action string) // if set, runs after each action, "> start" included

	G *game.Game // the game, once launched

	d       Driver
	out     []string
	rand    []randCall // the current run of Random calls, not written yet
	answers []bool     // for the next Yes/No dialogs

	startScreen bool
	version     int
	setup       save.File
	portSave    []byte // the save to start from, as the port wrote it (portsave)
	start       []byte // the save to compare with on quit
	saved       []byte // the save written on quit
}

type randCall struct{ n, v int }

// errScenario stops a scenario from inside the game's calls.
type errScenario struct{ error }

// Fail stops the scenario with an error.
func (p *Player) Fail(format string, a ...any) {
	panic(errScenario{fmt.Errorf(format, a...)})
}

// Trace returns the lines recorded so far.
func (p *Player) Trace() []string { return p.out }

func (p *Player) emit(format string, a ...any) {
	p.flush()
	p.out = append(p.out, fmt.Sprintf(format, a...))
}

// random records a Random call. Runs of more than 16 calls with the same
// range (the Game Corner's 10,000) are one line, as the probe wrote them.
func (p *Player) random(n, v int) {
	if len(p.rand) > 0 && p.rand[0].n != n {
		p.flush()
	}
	p.rand = append(p.rand, randCall{n, v})
}

func (p *Player) flush() {
	r := p.rand
	p.rand = nil
	if len(r) > 16 {
		last := r[len(r)-1]
		p.out = append(p.out, fmt.Sprintf("rand %d = %d (the last of %d)", last.n, last.v, len(r)))
		return
	}
	for _, c := range r {
		p.out = append(p.out, fmt.Sprintf("rand %d = %d", c.n, c.v))
	}
}

var kindNames = [...]string{"warning", "error", "information", "confirmation", "custom"}

// Answer returns the answer the probe gave a dialog: OK, or the next
// queued answer to a Yes/No question. It stops the scenario when none is
// queued.
func (p *Player) Answer(yesNo bool, text string) bool {
	if !yesNo {
		return true
	}
	if len(p.answers) == 0 {
		p.Fail("no answer queued for %q", text)
	}
	yes := p.answers[0]
	p.answers = p.answers[1:]
	return yes
}

// Dialog records a dialog the game showed, and its answer.
func (p *Player) Dialog(kind game.DialogKind, text string, yesNo, yes bool) {
	answer := "OK"
	if yesNo {
		answer = "No"
		if yes {
			answer = "Yes"
		}
	}
	p.emit("dialog %s %q -> %s", kindNames[kind], text, answer)
}

// Message answers and records a dialog: the game's UI, for Direct.
func (p *Player) Message(kind game.DialogKind, yesNo bool, text string) bool {
	yes := p.Answer(yesNo, text)
	p.Dialog(kind, text, yesNo, yes)
	return yes
}

// Play plays a scenario and returns the original's trace, from
// "> start"; Trace has the port's.
func (p *Player) Play(src string) (want []string, err error) {
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	p.version = 2
	p.setup.PlayerName = "Probe"
	p.setup.PokeBalls = 3
	start := -1
	for i, line := range lines {
		if line == "> start" {
			start = i
			break
		}
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if err := p.setupLine(line); err != nil {
			return nil, fmt.Errorf("setup %q: %v", line, err)
		}
	}
	if start < 0 {
		return nil, errors.New(`no "> start"`)
	}
	want = lines[start:]
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(errScenario)
			if !ok {
				panic(r)
			}
			err = e
		}
		p.flush()
	}()
	p.emit("> start")
	if err := p.launch(); err != nil {
		return want, err
	}
	if p.After != nil {
		p.After("> start")
	}
	for _, line := range want[1:] {
		if !strings.HasPrefix(line, "> ") {
			continue // the original's output
		}
		p.emit("%s", line)
		cmd, arg, _ := strings.Cut(strings.TrimPrefix(line, "> "), " ")
		if err := p.do(cmd, arg); err != nil {
			return want, fmt.Errorf("%s: %v", line, err)
		}
		if p.After != nil {
			p.After(line)
		}
	}
	return want, nil
}

// counterNames are the save's counters, in its order.
var counterNames = []string{"Money", "Wins", "Losses", "Fights", "Catches",
	"PokeBalls", "GreatBalls", "UltraBalls", "MasterBalls", "Potions", "SuperPotions"}

// counter returns the save's counter of that name, or nil.
func counter(f *save.File, name string) *uint32 {
	all := []*uint32{&f.Money, &f.Wins, &f.Losses, &f.Fights, &f.Catches,
		&f.PokeBalls, &f.GreatBalls, &f.UltraBalls, &f.MasterBalls, &f.Potions, &f.SuperPotions}
	for i, n := range counterNames {
		if n == name {
			return all[i]
		}
	}
	return nil
}

// setupLine applies a setup command to the save the scenario starts from.
func (p *Player) setupLine(line string) error {
	cmd, arg, _ := strings.Cut(line, " ")
	var nums []int
	for _, f := range strings.Fields(arg) {
		n, err := strconv.ParseInt(f, 0, 64)
		if err != nil && cmd != "name" && cmd != "portsave" {
			return err
		}
		nums = append(nums, int(n))
	}
	want := func(n int) error {
		if len(nums) != n {
			return fmt.Errorf("want %d numbers", n)
		}
		return nil
	}
	s := &p.setup
	if c := counter(s, cmd); c != nil {
		if err := want(1); err != nil {
			return err
		}
		*c = uint32(nums[0])
		return nil
	}
	switch cmd {
	case "startscreen":
		p.startScreen = true
	case "portsave":
		src, err := os.ReadFile(filepath.Join(p.Dir, arg+".txt"))
		if err != nil {
			return err
		}
		q := &Player{Name: arg, Dir: p.Dir, Data: p.Data, NewDriver: p.NewDriver, Saves: p.Saves}
		if _, err := q.Play(string(src)); err != nil {
			return fmt.Errorf("%s: %v", arg, err)
		}
		if q.saved == nil {
			return fmt.Errorf("%s doesn't quit", arg)
		}
		p.portSave = q.saved
	case "version":
		if err := want(1); err != nil {
			return err
		}
		p.version = nums[0]
	case "name":
		s.PlayerName = arg
	case "badges":
		if err := want(1); err != nil {
			return err
		}
		s.GymProgress = uint8(nums[0])
	case "flags":
		if err := want(5); err != nil {
			return err
		}
		s.BeatLance, s.BeatAsh, s.BeatLink, s.GotJirachi, s.GotDeoxys = nums[0] != 0, nums[1] != 0, nums[2] != 0, nums[3] != 0, nums[4] != 0
	case "dex":
		for _, sp := range nums {
			s.InDex[sp] = true
		}
	case "party":
		if len(nums) < 2 || len(nums) > 3 {
			return errors.New("want species, level and maybe XP")
		}
		l := uint32(nums[1])
		m := save.Slot{Species: int32(nums[0]), Level: uint8(nums[1]), XP: l * l * l}
		if len(nums) == 3 {
			m.XP = uint32(nums[2])
		}
		s.Owned[len(s.Party())] = m
		s.InDex[nums[0]] = true
	default:
		return errors.New("unknown setup command")
	}
	return nil
}

// savePath is where the probe kept the scenario's save, as the game saw it.
func (p *Player) savePath(ext string) string { return `C:\Temp\probe-` + p.Name + ext }

// launch starts the game: on the setup's save, or at the start screen.
func (p *Player) launch() error {
	p.d = p.NewDriver()
	// The game always writes version 2; a new game is compared with an
	// empty save.
	p.start = p.setupBytes()
	if p.version == 1 {
		p.start = p.setup.Bytes()
	}
	if p.startScreen && len(p.setup.Party()) == 0 {
		p.start = (&save.File{}).Bytes()
	}
	var b []byte
	if !p.startScreen {
		b = p.setupBytes()
	}
	return p.d.Launch(p, p.savePath(".pgs"), b, func(g *game.Game) {
		p.G = g
		g.Rand.OnCall = p.random
		g.OnLog = func(line string) { p.emit("log %s", line) }
	})
}

// setupBytes is the setup's save in its version, or the port's save.
func (p *Player) setupBytes() []byte {
	if p.portSave != nil {
		return p.portSave
	}
	b := p.setup.Bytes()
	if p.version != 1 {
		return b
	}
	// Version 1: no header, three flags, 251 POKéDEX entries.
	lines := strings.SplitAfter(string(b), "\r\n")
	v1 := append([]string{}, lines[1:5]...)        // name, BeatLance, BeatAsh, BeatLink
	v1 = append(v1, lines[7:19+251]...)            // GymProgress, the counters, POKéDEX 1–251
	v1 = append(v1, lines[19+save.NumSpecies:]...) // the party
	return []byte(strings.Join(v1, ""))
}

// do runs one of the probe's actions on the port.
func (p *Player) do(cmd, arg string) error {
	g, d := p.G, p.d
	num := func(s string) (int, error) {
		n, err := strconv.ParseInt(strings.TrimSpace(s), 0, 64)
		return int(n), err
	}
	switch cmd {
	case "seed":
		n, err := strconv.ParseUint(arg, 0, 32)
		if err != nil {
			return err
		}
		g.Rand.Seed = uint32(n)
	case "menu":
		return p.menu(arg)
	case "choose": // a double click on party member i
		i, err := num(arg)
		if err != nil {
			return err
		}
		if !g.Stats.Visible || !g.Stats.PartyEnabled || i < 1 || i > len(g.Party()) {
			return fmt.Errorf("party member %d can't be clicked", i)
		}
		return d.Choose(i - 1)
	case "move", "run":
		k := 5
		if cmd == "move" {
			n, err := num(arg)
			if err != nil {
				return err
			}
			k = n
		}
		return p.attack(k)
	case "fight", "fightbest":
		k, err := num(arg)
		if cmd == "fight" && err != nil {
			return err
		}
		for range 200 {
			if !g.Main.Go {
				return nil // the battle is over
			}
			if cmd == "fightbest" {
				k = p.bestMove()
			}
			if err := p.attack(k); err != nil {
				return err
			}
		}
		return errors.New("the battle didn't end after 200 turns")
	case "set":
		name, value, _ := strings.Cut(arg, " ")
		v, err := num(value)
		if err != nil {
			return err
		}
		switch c := counter(&g.File, name); {
		case name == "EnemyHP":
			g.EnemyHP = v
		case name == "OwnHP":
			g.OwnHP = v
		case c != nil:
			*c = uint32(v)
		default:
			return fmt.Errorf("unknown variable %q", name)
		}
	case "throw":
		k, err := num(arg)
		if err != nil {
			return err
		}
		for range 200 {
			if g.Main.EndFight || !g.Stats.Items || !g.Stats.ItemEnabled[k-1] {
				return nil // caught, the battle is over, or no balls left
			}
			if err := d.Item(k - 1); err != nil {
				return err
			}
		}
		return errors.New("still throwing after 200 balls")
	case "end":
		if !g.Main.EndFight {
			return errors.New(`"End Fight" is hidden`)
		}
		return d.End()
	case "item":
		k, err := num(arg)
		if err != nil {
			return err
		}
		if k < 1 || k > 6 || !g.Stats.Items || !g.Stats.ItemEnabled[k-1] {
			return fmt.Errorf("item button %d is hidden or disabled", k)
		}
		return d.Item(k - 1)
	case "click":
		return p.click(arg)
	case "radio":
		path, k, _ := strings.Cut(arg, " ")
		n, err := num(k)
		if err != nil {
			return err
		}
		return p.radio(path, n)
	case "combo":
		path, index, _ := strings.Cut(arg, " ")
		i, err := num(index)
		if err != nil {
			return err
		}
		return p.combo(path, i)
	case "type":
		path, text, _ := strings.Cut(arg, " ")
		return p.typeText(path, text)
	case "text":
		if arg != "Form4" && arg != "Form9.Edit1" {
			return fmt.Errorf("no text for %s", arg)
		}
		s, err := d.Text(arg)
		if err != nil {
			return err
		}
		p.emit("text %s %q", arg, s)
	case "answer":
		for _, a := range strings.Fields(arg) {
			if a != "yes" && a != "no" {
				return errors.New("answers are yes or no")
			}
			p.answers = append(p.answers, a == "yes")
		}
	case "state":
		p.state()
	case "menus":
		p.menus(g.Menu, 0)
	case "new":
		if !g.Start.Visible {
			return errors.New("not at the start screen")
		}
		p.emit("namedialog -> %q", arg)
		return d.New(p.savePath(".pgs"), arg)
	case "load":
		if !g.Start.Visible {
			return errors.New("not at the start screen")
		}
		return d.Load(p.savePath(".pgs"), p.setupBytes())
	case "starter":
		k, err := num(arg)
		if err != nil {
			return err
		}
		if !g.Main.Starters {
			return errors.New("the starters are hidden")
		}
		return d.Starter(k - 1)
	case "export":
		if !g.Item("File1").Enabled {
			return errors.New("the File menu is disabled")
		}
		return d.Export(p.savePath(".htm"))
	case "quit":
		b, err := d.Quit()
		if err != nil {
			return err
		}
		p.emit("exit 0")
		if b == nil {
			return errors.New("nothing to save")
		}
		p.saved = b
		if p.Saves != "" {
			if err := os.WriteFile(filepath.Join(p.Saves, p.Name+".pgs"), b, 0o644); err != nil {
				return err
			}
		}
		p.diffSave(b)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
	return nil
}

// attack selects attack k (1–4) or "Run!" (5) and presses "Go!".
func (p *Player) attack(k int) error {
	g := p.G
	if k < 1 || k > 5 || !g.Main.Attacks || !g.Main.Go {
		return fmt.Errorf("attack %d can't be chosen", k)
	}
	return p.d.Attack(k - 1)
}

// bestMove is the attack (1–4) with the highest power times type
// multipliers against the enemy, as the probe chose it.
func (p *Player) bestMove() int {
	g := p.G
	own := g.Data.SpeciesByID(int(g.Owned[g.OwnIndex-1].Species))
	enemy := g.Data.SpeciesByID(g.EnemySpecies)
	best, bestScore := 1, -1.0
	for k, id := range own.Moves {
		score := 0.0
		if mv := g.Data.MoveByID(id); mv != nil {
			score = float64(mv.Power) * g.Data.Effect(mv.Type, enemy.Types[0]) * g.Data.Effect(mv.Type, enemy.Types[1])
		}
		if score > bestScore {
			best, bestScore = k+1, score
		}
	}
	return best
}

// accelerators drops the "&" that marks a menu accelerator; "&&" is a
// literal "&".
var accelerators = strings.NewReplacer("&&", "&", "&", "")

// menu runs a main menu command given by captions: "Actions > Buy Items".
func (p *Player) menu(path string) error {
	items := p.G.Menu
	var chain []*game.MenuItem
	for _, part := range strings.Split(path, " > ") {
		var m *game.MenuItem
		for _, it := range items {
			if accelerators.Replace(it.Caption) == part {
				m = it
				break
			}
		}
		if m == nil {
			return fmt.Errorf("no item %q", part)
		}
		if !m.Enabled || !m.Visible {
			return fmt.Errorf("%q is disabled or hidden", part)
		}
		chain = append(chain, m)
		items = m.Items
	}
	m := chain[len(chain)-1]
	if !p.G.Main.Visible || p.G.Main.Starters && m.Name != "Quit1" {
		return errors.New("the game window isn't up")
	}
	return p.d.Menu(chain)
}

// menus dumps the main menu as the probe did.
func (p *Player) menus(items []*game.MenuItem, depth int) {
	for _, m := range items {
		p.emit("menu %s%s [enabled=%t visible=%t]", strings.Repeat("  ", depth), accelerators.Replace(m.Caption), m.Enabled, m.Visible)
		p.menus(m.Items, depth+1)
	}
}

// click presses a button, as long as the player could.
func (p *Player) click(path string) error {
	g := p.G
	var shown bool
	switch path {
	case "Form1.AboutButton":
		shown = g.Start.Visible
	case "Form4.Button2", "Form4.Button3", "Form4.Button4":
		shown = g.Main.Picker
	case "Form4.EvolveButton":
		shown = len(g.Main.Choices) > 0
	case "Form8.Button1":
		shown = g.Dex.Visible
	case "Form9.Button1":
		shown = g.Mart.Visible && g.Mart.BuyEnabled
	case "Form9.Button2":
		shown = g.Mart.Visible
	case "FormA.Button1":
		shown = g.Types.Visible
	case "FormB.Button1", "FormB.Button2":
		shown = g.Corner.Visible
	default:
		return fmt.Errorf("unknown button %s", path)
	}
	if !shown {
		return fmt.Errorf("%s is hidden or disabled", path)
	}
	if err := p.d.Click(path); err != nil {
		return err
	}
	if path == "Form1.AboutButton" {
		p.emit("about -> OK")
	}
	return nil
}

// radio selects item k (1-based) of a radio group.
func (p *Player) radio(path string, k int) error {
	g := p.G
	var shown bool
	switch path {
	case "Form4.EvolveChoice":
		shown = len(g.Main.Choices) > 0
	case "Form9.RadioGroup1":
		shown = g.Mart.Visible
	case "FormB.RadioGroup1", "FormB.RadioGroup2":
		shown = g.Corner.Visible
	}
	if !shown {
		return fmt.Errorf("%s is hidden or unknown", path)
	}
	return p.d.Radio(path, k-1)
}

// combo selects item i (0-based) of a combo box.
func (p *Player) combo(path string, i int) error {
	g := p.G
	var shown bool
	switch path {
	case "Form4.ComboBox1":
		shown = g.Main.Picker
	case "Form8.ComboBox1":
		shown = g.Dex.Visible
	case "FormA.ComboBox1", "FormA.ComboBox2":
		shown = g.Types.Visible
	}
	if !shown {
		return fmt.Errorf("%s is hidden or unknown", path)
	}
	return p.d.Combo(path, i)
}

// typeText types a number into a spin edit.
func (p *Player) typeText(path, text string) error {
	g := p.G
	n, err := strconv.Atoi(text)
	if err != nil {
		return err
	}
	var shown bool
	switch path {
	case "Form4.SpinEdit1":
		shown = g.Main.Picker
	case "Form9.SpinEdit1":
		shown = g.Mart.Visible
	case "FormB.SpinEdit1":
		shown = g.Corner.Visible
	}
	if !shown {
		return fmt.Errorf("%s is hidden or unknown", path)
	}
	return p.d.Type(path, n)
}

// state dumps the variables as the probe did (notes/state.md).
func (p *Player) state() {
	g := p.G
	b := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}
	p.emit("state battle wild=%d trainer=%d tournament=%d type=%d own=%d hp=%d enemy=%d level=%d hp=%d levels=%d-%d",
		b(g.InWildBattle), b(g.InTrainerBattle), b(g.InTournament), g.BattleType, g.OwnIndex, g.OwnHP,
		g.EnemySpecies, g.EnemyLevel, g.EnemyHP, g.MinLevel, g.MaxLevel)
	var cs []string
	for _, n := range counterNames {
		cs = append(cs, fmt.Sprintf("%s=%d", n, *counter(&g.File, n)))
	}
	var flags []string
	for i, on := range []bool{g.BeatLance, g.BeatAsh, g.BeatLink, g.GotJirachi, g.GotDeoxys} {
		if on {
			flags = append(flags, []string{"BeatLance", "BeatAsh", "BeatLink", "GotJirachi", "GotDeoxys"}[i])
		}
	}
	p.emit("state player badges=%d %s flags=%s", g.GymProgress, strings.Join(cs, " "), strings.Join(flags, ","))
	var dex []string
	for s := 1; s <= save.NumSpecies; s++ {
		if g.InDex[s] {
			dex = append(dex, strconv.Itoa(s))
		}
	}
	p.emit("state dex %s", strings.Join(dex, ","))
	var party []string
	for k, m := range g.Party() {
		party = append(party, fmt.Sprintf("%d:%d/L%d/%d", k+1, m.Species, m.Level, m.XP))
	}
	p.emit("state party %s", strings.Join(party, " "))
}

// diffSave lists the lines of the save written on quit that differ from
// the one the scenario started from (1-based, as in notes/persistence.md).
func (p *Player) diffSave(end []byte) {
	a := strings.Split(string(p.start), "\r\n")
	b := strings.Split(string(end), "\r\n")
	if len(a) != len(b) {
		p.emit("save %d lines, was %d", len(b)-1, len(a)-1)
	}
	same := true
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			p.emit("save line %d: %q -> %q", i+1, delphi.DecodeANSI([]byte(a[i])), delphi.DecodeANSI([]byte(b[i])))
			same = false
		}
	}
	if same && len(a) == len(b) {
		p.emit("save unchanged")
	}
}
