// SPDX-License-Identifier: AGPL-3.0-only

package scenario

import (
	"fmt"

	"github.com/jsperandio/pokemon-simulator/internal/game"
)

// Direct does the actions by calling the game's handlers, as the
// original's controls did, with the Player as the game's UI.
type Direct struct {
	game *game.Game
}

func (d *Direct) Launch(p *Player, file string, save []byte, hook func(*game.Game)) error {
	d.game = game.New(p.Data, p)
	hook(d.game)

	if save == nil {
		return nil
	}

	return d.game.Load(file, save)
}

func (d *Direct) Menu(path []*game.MenuItem) error {
	m := path[len(path)-1]
	if !d.game.ClickMenu(m.Name) {
		return fmt.Errorf("%s isn't a game command", m.Name)
	}

	return nil
}

func (d *Direct) Choose(i int) error {
	d.game.ClickParty(i)

	return nil
}

func (d *Direct) Attack(k int) error {
	d.game.Main.AttackIndex = k
	d.game.Go()

	return nil
}

func (d *Direct) Item(k int) error {
	d.game.UseItem(k)

	return nil
}

func (d *Direct) End() error {
	d.game.EndFight()

	return nil
}

func (d *Direct) Click(path string) error {
	gam := d.game
	press := map[string]func(){
		"Form1.AboutButton":  func() {},
		"Form4.Button2":      gam.PickGo,
		"Form4.Button3":      gam.PickCancel,
		"Form4.Button4":      gam.PickRandom,
		"Form4.EvolveButton": gam.Evolve,
		"Form8.Button1":      gam.ClosePokedex,
		"Form9.Button1":      gam.Buy,
		"Form9.Button2":      gam.LeavePokemart,
		"FormA.Button1":      gam.CloseTypes,
		"FormB.Button1":      gam.Draw,
		"FormB.Button2":      gam.CloseGameCorner,
	}[path]

	if press == nil {
		return fmt.Errorf("unknown button %s", path)
	}

	press()

	return nil
}

func (d *Direct) Radio(path string, idx int) error {
	g := d.game

	switch path {
	case "Form4.EvolveChoice":
		g.ChooseEvolution(idx)
	case "Form9.RadioGroup1":
		g.SelectPokemartItem(idx)
	case "FormB.RadioGroup1":
		g.SelectPrize(idx)
	case "FormB.RadioGroup2":
		g.SelectNumber(idx)
	default:
		return fmt.Errorf("unknown radio group %s", path)
	}

	return nil
}

func (d *Direct) Combo(path string, i int) error {
	g := d.game

	switch path {
	case "Form4.ComboBox1":
		g.PickSpecies(i)
	case "Form8.ComboBox1":
		g.SelectPokedex(i)
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
	g := d.game

	switch path {
	case "Form4.SpinEdit1":
		g.PickLevel(n)
	case "Form9.SpinEdit1":
		g.SetPokemartQuantity(n)
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
		return d.game.Main.Caption, nil
	case "Form9.Edit1":
		return d.game.Pokemart.Total, nil
	}

	return "", fmt.Errorf("no text for %s", path)
}

func (d *Direct) AltF4(form string) error {
	gam := d.game
	closeForm := map[string]func(){
		"Form5": gam.CloseStats,
		"Form7": gam.CloseInfo,
		"Form8": gam.ClosePokedex,
		"Form9": gam.LeavePokemart,
		"FormA": gam.CloseTypes,
		"FormB": gam.CloseGameCorner,
	}[form]

	if closeForm == nil {
		return fmt.Errorf("no Alt+F4 for %s", form)
	}

	closeForm()

	return nil
}

func (d *Direct) New(file, name string, ok bool) error {
	d.game.NewGame(file, name, ok)

	return nil
}

func (d *Direct) Load(file string, save []byte) error {
	return d.game.Load(file, save)
}

func (d *Direct) Starter(k int) error {
	d.game.ChooseStarter(k)

	return nil
}

func (d *Direct) Export(file string) error {
	d.game.Exported(file)

	return nil
}

func (d *Direct) Quit() ([]byte, error) {
	return d.game.Close(), nil
}
