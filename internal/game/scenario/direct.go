// SPDX-License-Identifier: AGPL-3.0-only

package scenario

import (
	"fmt"

	"github.com/jsperandio/pokemon-simulator/internal/game"
)

// Direct does the actions by calling the game's handlers, as the
// original's controls did, with the Player as the game's UI.
type Direct struct {
	g *game.Game
}

func (d *Direct) Launch(p *Player, file string, save []byte, hook func(*game.Game)) error {
	d.g = game.New(p.Data, p)
	hook(d.g)

	if save == nil {
		return nil
	}

	return d.g.Load(file, save)
}

func (d *Direct) Menu(path []*game.MenuItem) error {
	m := path[len(path)-1]
	if !d.g.ClickMenu(m.Name) {
		return fmt.Errorf("%s isn't a game command", m.Name)
	}

	return nil
}

func (d *Direct) Choose(i int) error {
	d.g.ClickParty(i)

	return nil
}

func (d *Direct) Attack(k int) error {
	d.g.Main.AttackIndex = k
	d.g.Go()

	return nil
}

func (d *Direct) Item(k int) error {
	d.g.UseItem(k)

	return nil
}

func (d *Direct) End() error {
	d.g.EndFight()

	return nil
}

func (d *Direct) Click(path string) error {
	g := d.g
	press := map[string]func(){
		"Form1.AboutButton":  func() {},
		"Form4.Button2":      g.PickGo,
		"Form4.Button3":      g.PickCancel,
		"Form4.Button4":      g.PickRandom,
		"Form4.EvolveButton": g.Evolve,
		"Form8.Button1":      g.CloseDex,
		"Form9.Button1":      g.Buy,
		"Form9.Button2":      g.LeaveMart,
		"FormA.Button1":      g.CloseTypes,
		"FormB.Button1":      g.Draw,
		"FormB.Button2":      g.CloseCorner,
	}[path]

	if press == nil {
		return fmt.Errorf("unknown button %s", path)
	}

	press()

	return nil
}

func (d *Direct) Radio(path string, k int) error {
	g := d.g

	switch path {
	case "Form4.EvolveChoice":
		g.ChooseEvolution(k)
	case "Form9.RadioGroup1":
		g.SelectMartItem(k)
	case "FormB.RadioGroup1":
		g.SelectPrize(k)
	case "FormB.RadioGroup2":
		g.SelectNumber(k)
	default:
		return fmt.Errorf("unknown radio group %s", path)
	}

	return nil
}

func (d *Direct) Combo(path string, i int) error {
	g := d.g

	switch path {
	case "Form4.ComboBox1":
		g.PickSpecies(i)
	case "Form8.ComboBox1":
		g.SelectDex(i)
	case "FormA.ComboBox1":
		g.SelectTypes(i, g.Types.Attack)
	case "FormA.ComboBox2":
		g.SelectTypes(g.Types.Species, i)
	default:
		return fmt.Errorf("unknown combo box %s", path)
	}

	return nil
}

func (d *Direct) Type(path string, n int) error {
	g := d.g

	switch path {
	case "Form4.SpinEdit1":
		g.PickLevel(n)
	case "Form9.SpinEdit1":
		g.SetMartQuantity(n)
	case "FormB.SpinEdit1":
		g.SetBet(n)
	default:
		return fmt.Errorf("unknown spin edit %s", path)
	}

	return nil
}

func (d *Direct) Text(path string) (string, error) {
	switch path {
	case "Form4":
		return d.g.Main.Caption, nil
	case "Form9.Edit1":
		return d.g.Mart.Total, nil
	}

	return "", fmt.Errorf("no text for %s", path)
}

func (d *Direct) New(file, name string) error {
	d.g.NewGame(file, name)

	return nil
}

func (d *Direct) Load(file string, save []byte) error { return d.g.Load(file, save) }

func (d *Direct) Starter(k int) error {
	d.g.ChooseStarter(k)

	return nil
}

func (d *Direct) Export(file string) error {
	d.g.Exported(file)

	return nil
}

func (d *Direct) Quit() ([]byte, error) { return d.g.Close(), nil }
