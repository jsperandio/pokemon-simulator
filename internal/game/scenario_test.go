// SPDX-License-Identifier: AGPL-3.0-only

package game

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jsperandio/pokemon-simulator/assets"
	"github.com/jsperandio/pokemon-simulator/internal/delphi"
	"github.com/jsperandio/pokemon-simulator/internal/gamedata"
	"github.com/jsperandio/pokemon-simulator/internal/save"
)

// The scenarios in testdata/scenarios are traces of the original game,
// recorded under Wine by a debugger that played them: the setup, then
// each action ("> ...") followed by what the original did, in order: its
// Random calls, Battle Info lines, dialogs, variables ("state") and, on
// quit, the lines of the save it wrote that changed. TestScenarios plays
// them on the port, which must produce the same lines.
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
// The actions are the probe's; player.do lists them.

// With -saves <dir>, the save each scenario quits with is written to
// <dir>/<scenario>.pgs, for the original to load (portsave).
var saveDir = flag.String("saves", "", "write the save each scenario quits with to this directory")

func TestScenarios(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "scenarios", "*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no scenarios")
	}
	data := loadData(t)
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".txt")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			p := &player{scenario: name, data: data}
			want, err := p.play(string(src))
			if err != nil {
				t.Errorf("%v\n%s", err, tail(p.out, 8))
				return
			}
			compare(t, want, p.out)
		})
	}
}

func loadData(t *testing.T) *gamedata.Data {
	t.Helper()
	raw, err := assets.FS.ReadFile("pokegame.dat")
	if err != nil {
		t.Fatal(err)
	}
	d, err := gamedata.Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// compare reports the first line where the port's trace differs from the
// original's.
func compare(t *testing.T, want, got []string) {
	t.Helper()
	for i := range max(len(want), len(got)) {
		w, g := "(end)", "(end)"
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			t.Errorf("line %d differs:\noriginal: %s\nport:     %s\nthe port's trace before it:\n%s",
				i+1, w, g, tail(got[:min(i, len(got))], 8))
			return
		}
	}
}

func tail(lines []string, n int) string {
	return "\t" + strings.Join(lines[max(len(lines)-n, 0):], "\n\t")
}

// player plays a scenario on the port and records the same trace the
// probe recorded on the original. It is also the game's UI: it answers
// the dialogs as the probe did.
type player struct {
	scenario string
	data     *gamedata.Data
	g        *Game
	out      []string
	rand     []randCall // the current run of Random calls, not written yet
	answers  []bool     // for the next Yes/No dialogs

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

func (p *player) fail(format string, a ...any) {
	panic(errScenario{fmt.Errorf(format, a...)})
}

func (p *player) emit(format string, a ...any) {
	p.flush()
	p.out = append(p.out, fmt.Sprintf(format, a...))
}

// random records a Random call. Runs of more than 16 calls with the same
// range (the Game Corner's 10,000) are one line, as the probe wrote them.
func (p *player) random(n, v int) {
	if len(p.rand) > 0 && p.rand[0].n != n {
		p.flush()
	}
	p.rand = append(p.rand, randCall{n, v})
}

func (p *player) flush() {
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

// Message answers OK, or the next queued answer to a Yes/No question.
func (p *player) Message(kind DialogKind, yesNo bool, text string) bool {
	answer, yes := "OK", true
	if yesNo {
		if len(p.answers) == 0 {
			p.fail("no answer queued for %q", text)
		}
		yes, p.answers = p.answers[0], p.answers[1:]
		answer = "No"
		if yes {
			answer = "Yes"
		}
	}
	p.emit("dialog %s %q -> %s", kindNames[kind], text, answer)
	return yes
}

// play runs a scenario and returns the original's trace, from "> start".
func (p *player) play(src string) (want []string, err error) {
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
	for _, line := range want[1:] {
		if !strings.HasPrefix(line, "> ") {
			continue // the original's output
		}
		p.emit("%s", line)
		cmd, arg, _ := strings.Cut(strings.TrimPrefix(line, "> "), " ")
		if err := p.do(cmd, arg); err != nil {
			return want, fmt.Errorf("%s: %v", line, err)
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
func (p *player) setupLine(line string) error {
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
		src, err := os.ReadFile(filepath.Join("testdata", "scenarios", arg+".txt"))
		if err != nil {
			return err
		}
		q := &player{scenario: arg, data: p.data}
		if _, err := q.play(string(src)); err != nil {
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
func (p *player) savePath(ext string) string { return `C:\Temp\probe-` + p.scenario + ext }

// launch starts the game: on the setup's save, or at the start screen.
func (p *player) launch() error {
	p.g = New(p.data, p)
	p.g.Rand.OnCall = p.random
	p.g.OnLog = func(line string) { p.emit("log %s", line) }
	// The game always writes version 2; a new game is compared with an
	// empty save.
	p.start = p.setupBytes()
	if p.version == 1 {
		p.start = p.setup.Bytes()
	}
	if p.startScreen && len(p.setup.Party()) == 0 {
		p.start = (&save.File{}).Bytes()
	}
	if p.startScreen {
		return nil
	}
	return p.g.Load(p.savePath(".pgs"), p.setupBytes())
}

// setupBytes is the setup's save in its version, or the port's save.
func (p *player) setupBytes() []byte {
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
func (p *player) do(cmd, arg string) error {
	g := p.g
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
		g.ClickParty(i - 1)
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
			g.UseItem(k - 1)
		}
		return errors.New("still throwing after 200 balls")
	case "end":
		if !g.Main.EndFight {
			return errors.New(`"End Fight" is hidden`)
		}
		g.EndFight()
	case "item":
		k, err := num(arg)
		if err != nil {
			return err
		}
		if k < 1 || k > 6 || !g.Stats.Items || !g.Stats.ItemEnabled[k-1] {
			return fmt.Errorf("item button %d is hidden or disabled", k)
		}
		g.UseItem(k - 1)
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
		switch arg {
		case "Form4":
			p.emit("text %s %q", arg, g.Main.Caption)
		case "Form9.Edit1":
			p.emit("text %s %q", arg, g.Mart.Total)
		default:
			return fmt.Errorf("no text for %s", arg)
		}
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
		g.NewGame(p.savePath(".pgs"), arg)
	case "load":
		if !g.Start.Visible {
			return errors.New("not at the start screen")
		}
		return g.Load(p.savePath(".pgs"), p.setupBytes())
	case "starter":
		k, err := num(arg)
		if err != nil {
			return err
		}
		if !g.Main.Starters {
			return errors.New("the starters are hidden")
		}
		g.ChooseStarter(k - 1)
	case "export":
		if !g.Item("File1").Enabled {
			return errors.New("the File menu is disabled")
		}
		g.Exported(p.savePath(".htm"))
	case "quit":
		b := g.Close()
		p.emit("exit 0")
		if b == nil {
			return errors.New("nothing to save")
		}
		p.saved = b
		if *saveDir != "" {
			if err := os.WriteFile(filepath.Join(*saveDir, p.scenario+".pgs"), b, 0o644); err != nil {
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
func (p *player) attack(k int) error {
	g := p.g
	if k < 1 || k > 5 || !g.Main.Attacks || !g.Main.Go {
		return fmt.Errorf("attack %d can't be chosen", k)
	}
	g.Main.AttackIndex = k - 1
	g.Go()
	return nil
}

// bestMove is the attack (1–4) with the highest power times type
// multipliers against the enemy, as the probe chose it.
func (p *player) bestMove() int {
	g := p.g
	own := g.species(int(g.member(g.OwnIndex).Species))
	enemy := g.species(g.EnemySpecies)
	best, bestScore := 1, -1.0
	for k, id := range own.Moves {
		mv := g.move(id)
		score := float64(mv.Power) * g.Data.Effect(mv.Type, enemy.Types[0]) * g.Data.Effect(mv.Type, enemy.Types[1])
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
func (p *player) menu(path string) error {
	items := p.g.Menu
	var m *MenuItem
	for _, part := range strings.Split(path, " > ") {
		m = nil
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
		items = m.Items
	}
	if !p.g.Main.Visible || p.g.Main.Starters && m.Name != "Quit1" {
		return errors.New("the game window isn't up")
	}
	if !p.g.ClickMenu(m.Name) {
		return fmt.Errorf("%s isn't a game command", m.Name)
	}
	return nil
}

// menus dumps the main menu as the probe did.
func (p *player) menus(items []*MenuItem, depth int) {
	for _, m := range items {
		p.emit("menu %s%s [enabled=%t visible=%t]", strings.Repeat("  ", depth), accelerators.Replace(m.Caption), m.Enabled, m.Visible)
		p.menus(m.Items, depth+1)
	}
}

// click presses a button, as long as the player could.
func (p *player) click(path string) error {
	g := p.g
	type button struct {
		shown bool
		press func()
	}
	buttons := map[string]button{
		"Form1.AboutButton":  {g.Start.Visible, func() { p.emit("about -> OK") }},
		"Form4.Button2":      {g.Main.Picker, g.PickGo},
		"Form4.Button3":      {g.Main.Picker, g.PickCancel},
		"Form4.Button4":      {g.Main.Picker, g.PickRandom},
		"Form4.EvolveButton": {len(g.Main.Choices) > 0, g.Evolve},
		"Form8.Button1":      {g.Dex.Visible, g.CloseDex},
		"Form9.Button1":      {g.Mart.Visible && g.Mart.BuyEnabled, g.Buy},
		"Form9.Button2":      {g.Mart.Visible, g.LeaveMart},
		"FormA.Button1":      {g.Types.Visible, g.CloseTypes},
		"FormB.Button1":      {g.Corner.Visible, g.Draw},
		"FormB.Button2":      {g.Corner.Visible, g.CloseCorner},
	}
	b, ok := buttons[path]
	if !ok {
		return fmt.Errorf("unknown button %s", path)
	}
	if !b.shown {
		return fmt.Errorf("%s is hidden or disabled", path)
	}
	b.press()
	return nil
}

// radio selects item k (1-based) of a radio group.
func (p *player) radio(path string, k int) error {
	g := p.g
	switch {
	case path == "Form4.EvolveChoice" && len(g.Main.Choices) > 0:
		g.ChooseEvolution(k - 1)
	case path == "Form9.RadioGroup1" && g.Mart.Visible:
		g.SelectMartItem(k - 1)
	case path == "FormB.RadioGroup1" && g.Corner.Visible:
		g.SelectPrize(k - 1)
	case path == "FormB.RadioGroup2" && g.Corner.Visible:
		g.SelectNumber(k - 1)
	default:
		return fmt.Errorf("%s is hidden or unknown", path)
	}
	return nil
}

// combo selects item i (0-based) of a combo box.
func (p *player) combo(path string, i int) error {
	g := p.g
	switch {
	case path == "Form4.ComboBox1" && g.Main.Picker:
		g.PickSpecies(i)
	case path == "Form8.ComboBox1" && g.Dex.Visible:
		g.SelectDex(i)
	case path == "FormA.ComboBox1" && g.Types.Visible:
		g.SelectTypes(i, g.Types.Attack)
	case path == "FormA.ComboBox2" && g.Types.Visible:
		g.SelectTypes(g.Types.Species, i)
	default:
		return fmt.Errorf("%s is hidden or unknown", path)
	}
	return nil
}

// typeText types a number into a spin edit.
func (p *player) typeText(path, text string) error {
	g := p.g
	n, err := strconv.Atoi(text)
	if err != nil {
		return err
	}
	switch {
	case path == "Form4.SpinEdit1" && g.Main.Picker:
		g.PickLevel(n)
	case path == "Form9.SpinEdit1" && g.Mart.Visible:
		g.SetMartQuantity(n)
	case path == "FormB.SpinEdit1" && g.Corner.Visible:
		g.SetBet(n)
	default:
		return fmt.Errorf("%s is hidden or unknown", path)
	}
	return nil
}

// state dumps the variables as the probe did (notes/state.md).
func (p *player) state() {
	g := p.g
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
func (p *player) diffSave(end []byte) {
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
