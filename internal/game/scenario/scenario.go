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
// The actions are the probe's, listed in actions.
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

	Game *game.Game // the game, once launched

	drv     Driver
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

type randCall struct {
	limit int
	value int
}

// scenarioError stops a scenario from inside the game's calls.
type scenarioError struct{ error }

// Fail stops the scenario with an error.
func (p *Player) Fail(format string, a ...any) {
	panic(scenarioError{fmt.Errorf(format, a...)})
}

// Trace returns the lines recorded so far.
func (p *Player) Trace() []string {
	return p.out
}

func (p *Player) emit(format string, a ...any) {
	p.flush()

	p.out = append(p.out, fmt.Sprintf(format, a...))
}

// random records a Random call. Runs of more than 16 calls with the same
// range (the Game Corner's 10,000) are one line, as the probe wrote them.
func (p *Player) random(n, v int) {
	if len(p.rand) > 0 && p.rand[0].limit != n {
		p.flush()
	}

	p.rand = append(p.rand, randCall{n, v})
}

func (p *Player) flush() {
	r := p.rand
	p.rand = nil

	if len(r) > 16 {
		last := r[len(r)-1]
		p.out = append(p.out, fmt.Sprintf("rand %d = %d (the last of %d)", last.limit, last.value, len(r)))

		return
	}

	for _, c := range r {
		p.out = append(p.out, fmt.Sprintf("rand %d = %d", c.limit, c.value))
	}
}

var kindNames = [...]string{
	"warning",
	"error",
	"information",
	"confirmation",
	"custom",
}

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
			e, ok := r.(scenarioError)
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
var counterNames = []string{
	"Money", "Wins", "Losses", "Fights", "Catches",
	"PokeBalls", "GreatBalls", "UltraBalls", "MasterBalls", "Potions", "SuperPotions",
}

// counter returns the save's counter of that name, or nil.
func counter(f *save.File, name string) *uint32 {
	all := []*uint32{
		&f.Money, &f.Wins, &f.Losses, &f.Fights, &f.Catches,
		&f.PokeBalls, &f.GreatBalls, &f.UltraBalls, &f.MasterBalls, &f.Potions, &f.SuperPotions,
	}

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

	switch cmd {
	case "startscreen":
		p.startScreen = true

		return nil
	case "name":
		p.setup.PlayerName = arg

		return nil
	case "portsave":
		return p.portsave(arg)
	}

	var nums []int

	for f := range strings.FieldsSeq(arg) {
		n, err := strconv.ParseInt(f, 0, 64)
		if err != nil {
			return err
		}

		nums = append(nums, int(n))
	}

	return p.setupNumbers(cmd, nums)
}

// portsave starts the scenario from the save the port writes quitting
// scenario name.
func (p *Player) portsave(name string) error {
	src, err := os.ReadFile(filepath.Join(p.Dir, name+".txt")) //nolint:gosec // a scenario in this one's folder
	if err != nil {
		return err
	}

	sub := &Player{
		Name:      name,
		Dir:       p.Dir,
		Data:      p.Data,
		NewDriver: p.NewDriver,
		Saves:     p.Saves,
	}
	if _, err := sub.Play(string(src)); err != nil {
		return fmt.Errorf("%s: %v", name, err)
	}

	if sub.saved == nil {
		return fmt.Errorf("%s doesn't quit", name)
	}

	p.portSave = sub.saved

	return nil
}

// setupNumbers applies a setup command that takes numbers.
func (p *Player) setupNumbers(cmd string, nums []int) error {
	want := func(n int) error {
		if len(nums) != n {
			return fmt.Errorf("want %d numbers", n)
		}

		return nil
	}
	sav := &p.setup

	if c := counter(sav, cmd); c != nil {
		if err := want(1); err != nil {
			return err
		}

		*c = uint32(nums[0]) //nolint:gosec // a value of the original's save

		return nil
	}

	switch cmd {
	case "version":
		if err := want(1); err != nil {
			return err
		}

		p.version = nums[0]
	case "badges":
		if err := want(1); err != nil {
			return err
		}

		sav.GymProgress = uint8(nums[0]) //nolint:gosec // a value of the original's save
	case "flags":
		if err := want(5); err != nil {
			return err
		}

		sav.BeatLance, sav.BeatAsh, sav.BeatLink, sav.GotJirachi, sav.GotDeoxys = nums[0] != 0, nums[1] != 0, nums[2] != 0, nums[3] != 0, nums[4] != 0
	case "dex":
		for _, species := range nums {
			sav.InDex[species] = true
		}
	case "party":
		return p.setupParty(nums)
	default:
		return errors.New("unknown setup command")
	}

	return nil
}

// setupParty adds a party member, registered: species, level and maybe XP.
func (p *Player) setupParty(nums []int) error {
	if len(nums) < 2 || len(nums) > 3 {
		return errors.New("want species, level and maybe XP")
	}

	l := uint32(nums[1]) //nolint:gosec // a value of the original's save
	slt := save.Slot{
		Species: int32(nums[0]), //nolint:gosec // the same
		Level:   uint8(nums[1]), //nolint:gosec // the same
		XP:      l * l * l,
	}

	if len(nums) == 3 {
		slt.XP = uint32(nums[2]) //nolint:gosec // a value of the original's save
	}

	s := &p.setup
	s.Owned[len(s.Party())] = slt
	s.InDex[nums[0]] = true

	return nil
}

// savePath is where the probe kept the scenario's save, as the game saw it.
func (p *Player) savePath(ext string) string {
	return `C:\Temp\probe-` + p.Name + ext
}

// launch starts the game: on the setup's save, or at the start screen.
func (p *Player) launch() error {
	p.drv = p.NewDriver()
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

	return p.drv.Launch(p, p.savePath(".pgs"), b, func(g *game.Game) {
		p.Game = g
		g.Rand.OnCall = p.random
		g.OnLog = func(line string) {
			p.emit("log %s", line)
		}
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
	a, ok := actions[cmd]
	if !ok {
		return fmt.Errorf("unknown command %q", cmd)
	}

	return a(p, arg)
}

// actions are the probe's actions, by command. Each one takes the rest of
// its line.
var actions = map[string]func(p *Player, arg string) error{
	"seed":      (*Player).seed,
	"menu":      (*Player).menu,
	"choose":    (*Player).choose,
	"move":      (*Player).move,
	"run":       (*Player).run,
	"fight":     (*Player).fight,
	"fightbest": (*Player).fightBest,
	"set":       (*Player).set,
	"throw":     (*Player).throw,
	"end":       (*Player).end,
	"item":      (*Player).item,
	"click":     (*Player).click,
	"radio":     (*Player).radio,
	"combo":     (*Player).combo,
	"type":      (*Player).typeText,
	"text":      (*Player).text,
	"answer":    (*Player).answer,
	"state":     (*Player).state,
	"menus":     (*Player).dumpMenus,
	"new":       (*Player).newGame,
	"load":      (*Player).load,
	"starter":   (*Player).starter,
	"export":    (*Player).export,
	"quit":      (*Player).quit,
}

// number parses an action's number.
func number(s string) (int, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 0, 64)

	return int(n), err
}

// seed sets RandSeed.
func (p *Player) seed(arg string) error {
	n, err := strconv.ParseUint(arg, 0, 32)
	if err != nil {
		return err
	}

	p.Game.Rand.Seed = uint32(n)

	return nil
}

// partyClickable reports whether party member i (1-based) can be
// double-clicked: the Statistics window shows, with its party enabled.
func partyClickable(g *game.Game, i int) bool {
	return g.Stats.Visible && g.Stats.PartyEnabled && i >= 1 && i <= len(g.Party())
}

// attackable reports whether attack k (1–4), or "Run!" (5), can be chosen:
// the game window has the attacks up and the fight goes on.
func attackable(g *game.Game, k int) bool {
	return k >= 1 && k <= 5 && g.Main.Attacks && g.Main.Go
}

// choose double-clicks party member i (1-based).
func (p *Player) choose(arg string) error {
	i, err := number(arg)
	if err != nil {
		return err
	}

	if !partyClickable(p.Game, i) {
		return fmt.Errorf("party member %d can't be clicked", i)
	}

	return p.drv.Choose(i - 1)
}

// move attacks with attack k (1–4).
func (p *Player) move(arg string) error {
	k, err := number(arg)
	if err != nil {
		return err
	}

	return p.attack(k)
}

// run chooses "Run!".
func (p *Player) run(string) error {
	return p.attack(5)
}

// fight attacks with attack k (1–4) until the battle is over.
func (p *Player) fight(arg string) error {
	k, err := number(arg)
	if err != nil {
		return err
	}

	return p.fightWith(func() int {
		return k
	})
}

// fightBest attacks with the best attack until the battle is over.
func (p *Player) fightBest(string) error {
	return p.fightWith(p.bestMove)
}

// fightWith attacks with the attack move picks until the battle is over.
func (p *Player) fightWith(move func() int) error {
	for range 200 {
		if !p.Game.Main.Go {
			return nil // the battle is over
		}

		if err := p.attack(move()); err != nil {
			return err
		}
	}

	return errors.New("the battle didn't end after 200 turns")
}

// set sets one of the game's variables: "<name> <value>".
func (p *Player) set(arg string) error {
	name, value, _ := strings.Cut(arg, " ")

	val, err := number(value)
	if err != nil {
		return err
	}

	g := p.Game

	switch c := counter(&g.File, name); {
	case name == "EnemyHP":
		g.EnemyHP = val
	case name == "OwnHP":
		g.OwnHP = val
	case c != nil:
		*c = uint32(val) //nolint:gosec // a value the probe set in the original
	default:
		return fmt.Errorf("unknown variable %q", name)
	}

	return nil
}

// throw throws ball k (1–4) until it catches, the battle is over or no
// balls are left.
func (p *Player) throw(arg string) error {
	ball, err := number(arg)
	if err != nil {
		return err
	}

	g := p.Game
	for range 200 {
		if g.Main.EndFight || !g.Stats.Items || !g.Stats.ItemEnabled[ball-1] {
			return nil // caught, the battle is over, or no balls left
		}

		if err := p.drv.Item(ball - 1); err != nil {
			return err
		}
	}

	return errors.New("still throwing after 200 balls")
}

// end presses "End Fight".
func (p *Player) end(string) error {
	if !p.Game.Main.EndFight {
		return errors.New(`"End Fight" is hidden`)
	}

	return p.drv.End()
}

// item presses item button k (1–6).
func (p *Player) item(arg string) error {
	k, err := number(arg)
	if err != nil {
		return err
	}

	if !p.Game.ItemUsable(k - 1) {
		return fmt.Errorf("item button %d is hidden or disabled", k)
	}

	return p.drv.Item(k - 1)
}

// text records the game window's caption or the Mart's total.
func (p *Player) text(arg string) error {
	if arg != "Form4" && arg != "Form9.Edit1" {
		return fmt.Errorf("no text for %s", arg)
	}

	s, err := p.drv.Text(arg)
	if err != nil {
		return err
	}

	p.emit("text %s %q", arg, s)

	return nil
}

// answer queues the answers to the next Yes/No dialogs.
func (p *Player) answer(arg string) error {
	for a := range strings.FieldsSeq(arg) {
		if a != "yes" && a != "no" {
			return errors.New("answers are yes or no")
		}

		p.answers = append(p.answers, a == "yes")
	}

	return nil
}

// newGame starts a new game from the start screen, for the name typed.
func (p *Player) newGame(name string) error {
	if !p.Game.Start.Visible {
		return errors.New("not at the start screen")
	}

	p.emit("namedialog -> %q", name)

	return p.drv.New(p.savePath(".pgs"), name)
}

// load loads the scenario's save from the start screen.
func (p *Player) load(string) error {
	if !p.Game.Start.Visible {
		return errors.New("not at the start screen")
	}

	return p.drv.Load(p.savePath(".pgs"), p.setupBytes())
}

// starter chooses starter k (1-based).
func (p *Player) starter(arg string) error {
	k, err := number(arg)
	if err != nil {
		return err
	}

	if !p.Game.Main.Starters {
		return errors.New("the starters are hidden")
	}

	return p.drv.Starter(k - 1)
}

// export exports the POKéMON information.
func (p *Player) export(string) error {
	if !p.Game.Item("File1").Enabled {
		return errors.New("the File menu is disabled")
	}

	return p.drv.Export(p.savePath(".htm"))
}

// quit closes the game window, and compares the save it wrote with the
// one the scenario started from.
func (p *Player) quit(string) error {
	dat, err := p.drv.Quit()
	if err != nil {
		return err
	}

	p.emit("exit 0")

	if dat == nil {
		return errors.New("nothing to save")
	}

	p.saved = dat
	if p.Saves != "" {
		if err := os.WriteFile(filepath.Join(p.Saves, p.Name+".pgs"), dat, 0o600); err != nil {
			return err
		}
	}

	p.diffSave(dat)

	return nil
}

// attack selects attack k (1–4) or "Run!" (5) and presses "Go!".
func (p *Player) attack(k int) error {
	if !attackable(p.Game, k) {
		return fmt.Errorf("attack %d can't be chosen", k)
	}

	return p.drv.Attack(k - 1)
}

// bestMove is the attack (1–4) with the highest power times type
// multipliers against the enemy, as the probe chose it.
func (p *Player) bestMove() int {
	g := p.Game
	own := g.Data.SpeciesByID(int(g.Owned[g.OwnIndex-1].Species))
	enemy := g.Data.SpeciesByID(g.EnemySpecies)
	best, bestScore := 1, -1.0

	for k, id := range own.Moves {
		score := 0.0
		if mov := g.Data.MoveByID(id); mov != nil {
			score = float64(mov.Power) * g.Data.Effect(mov.Type, enemy.Types[0]) * g.Data.Effect(mov.Type, enemy.Types[1])
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
	items := p.Game.Menu

	var chain []*game.MenuItem

	for part := range strings.SplitSeq(path, " > ") {
		var found *game.MenuItem

		for _, mi := range items {
			if accelerators.Replace(mi.Caption) == part {
				found = mi

				break
			}
		}

		if found == nil {
			return fmt.Errorf("no item %q", part)
		}

		if !found.Enabled || !found.Visible {
			return fmt.Errorf("%q is disabled or hidden", part)
		}

		chain = append(chain, found)
		items = found.Items
	}

	m := chain[len(chain)-1]
	if !p.Game.Main.Visible || (p.Game.Main.Starters && m.Name != "Quit1") {
		return errors.New("the game window isn't up")
	}

	return p.drv.Menu(chain)
}

// dumpMenus dumps the main menu as the probe did.
func (p *Player) dumpMenus(string) error {
	p.menus(p.Game.Menu, 0)

	return nil
}

// menus dumps items, depth levels down.
func (p *Player) menus(items []*game.MenuItem, depth int) {
	for _, m := range items {
		p.emit("menu %s%s [enabled=%t visible=%t]", strings.Repeat("  ", depth), accelerators.Replace(m.Caption), m.Enabled, m.Visible)
		p.menus(m.Items, depth+1)
	}
}

// click presses a button, as long as the player could.
func (p *Player) click(path string) error {
	gam := p.Game

	var shown bool

	switch path {
	case "Form1.AboutButton":
		shown = gam.Start.Visible
	case "Form4.Button2", "Form4.Button3", "Form4.Button4":
		shown = gam.Main.Picker
	case "Form4.EvolveButton":
		shown = len(gam.Main.Choices) > 0
	case "Form8.Button1":
		shown = gam.Dex.Visible
	case "Form9.Button1":
		shown = gam.Mart.Visible && gam.Mart.BuyEnabled
	case "Form9.Button2":
		shown = gam.Mart.Visible
	case "FormA.Button1":
		shown = gam.Types.Visible
	case "FormB.Button1", "FormB.Button2":
		shown = gam.Corner.Visible
	default:
		return fmt.Errorf("unknown button %s", path)
	}

	if !shown {
		return fmt.Errorf("%s is hidden or disabled", path)
	}

	if err := p.drv.Click(path); err != nil {
		return err
	}

	if path == "Form1.AboutButton" {
		p.emit("about -> OK")
	}

	return nil
}

// radio selects item k (1-based) of a radio group: "<path> <k>".
func (p *Player) radio(arg string) error {
	path, k, _ := strings.Cut(arg, " ")

	n, err := number(k)
	if err != nil {
		return err
	}

	g := p.Game

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

	return p.drv.Radio(path, n-1)
}

// combo selects item i (0-based) of a combo box: "<path> <i>".
func (p *Player) combo(arg string) error {
	path, index, _ := strings.Cut(arg, " ")

	i, err := number(index)
	if err != nil {
		return err
	}

	g := p.Game

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

	return p.drv.Combo(path, i)
}

// typeText types a number into a spin edit: "<path> <text>".
func (p *Player) typeText(arg string) error {
	path, text, _ := strings.Cut(arg, " ")
	gam := p.Game

	n, err := strconv.Atoi(text)
	if err != nil {
		return err
	}

	var shown bool

	switch path {
	case "Form4.SpinEdit1":
		shown = gam.Main.Picker
	case "Form9.SpinEdit1":
		shown = gam.Mart.Visible
	case "FormB.SpinEdit1":
		shown = gam.Corner.Visible
	}

	if !shown {
		return fmt.Errorf("%s is hidden or unknown", path)
	}

	return p.drv.Type(path, n)
}

// state dumps the variables as the probe did (notes/state.md).
func (p *Player) state(string) error {
	gam := p.Game
	b := func(v bool) int {
		if v {
			return 1
		}

		return 0
	}
	p.emit("state battle wild=%d trainer=%d tournament=%d type=%d own=%d hp=%d enemy=%d level=%d hp=%d levels=%d-%d",
		b(gam.InWildBattle), b(gam.InTrainerBattle), b(gam.InTournament), gam.BattleType, gam.OwnIndex, gam.OwnHP,
		gam.EnemySpecies, gam.EnemyLevel, gam.EnemyHP, gam.MinLevel, gam.MaxLevel)

	counters := make([]string, 0, len(counterNames))
	for _, n := range counterNames {
		counters = append(counters, fmt.Sprintf("%s=%d", n, *counter(&gam.File, n)))
	}

	var flags []string

	flagNames := []string{
		"BeatLance",
		"BeatAsh",
		"BeatLink",
		"GotJirachi",
		"GotDeoxys",
	}
	flagStates := []bool{
		gam.BeatLance,
		gam.BeatAsh,
		gam.BeatLink,
		gam.GotJirachi,
		gam.GotDeoxys,
	}

	for i, on := range flagStates {
		if on {
			flags = append(flags, flagNames[i])
		}
	}

	p.emit("state player badges=%d %s flags=%s", gam.GymProgress, strings.Join(counters, " "), strings.Join(flags, ","))

	var dex []string

	for s := 1; s <= save.NumSpecies; s++ {
		if gam.InDex[s] {
			dex = append(dex, strconv.Itoa(s))
		}
	}

	p.emit("state dex %s", strings.Join(dex, ","))

	party := make([]string, 0, len(gam.Party()))
	for k, m := range gam.Party() {
		party = append(party, fmt.Sprintf("%d:%d/L%d/%d", k+1, m.Species, m.Level, m.XP))
	}

	p.emit("state party %s", strings.Join(party, " "))

	return nil
}

// diffSave lists the lines of the save written on quit that differ from
// the one the scenario started from (1-based, as in notes/persistence.md).
func (p *Player) diffSave(end []byte) {
	was := strings.Split(string(p.start), "\r\n")
	now := strings.Split(string(end), "\r\n")

	if len(was) != len(now) {
		p.emit("save %d lines, was %d", len(now)-1, len(was)-1)
	}

	same := true

	for i := range min(len(was), len(now)) {
		if was[i] != now[i] {
			p.emit("save line %d: %q -> %q", i+1, delphi.DecodeANSI([]byte(was[i])), delphi.DecodeANSI([]byte(now[i])))

			same = false
		}
	}

	if same && len(was) == len(now) {
		p.emit("save unchanged")
	}
}
