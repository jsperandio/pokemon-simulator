// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jsperandio/pokemon-simulator/internal/game"
	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// settingsStore keeps the preferences' bytes between runs: a file on the
// desktop, the page's local storage on the web.
type settingsStore interface {
	Read() ([]byte, error)
	Write(data []byte) error
}

// preferences are the player's choices in the Preferences window, which
// the original doesn't have.
type preferences struct {
	animatedSprites bool // Graphics: Enable animated sprites
}

// defaultPreferences are the choices before the player makes any.
func defaultPreferences() preferences {
	return preferences{
		animatedSprites: true,
	}
}

// loadPreferences reads the preferences from dat, an INI file; what is
// missing or unreadable keeps its default.
func loadPreferences(dat []byte) preferences {
	prf := defaultPreferences()
	ini := parseINI(dat)
	prf.animatedSprites = ini.boolean("Graphics", "AnimatedSprites", prf.animatedSprites)

	return prf
}

// store writes the preferences into dat, an INI file, keeping the lines it
// doesn't know.
func (prf preferences) store(dat []byte) []byte {
	ini := parseINI(dat)
	ini.setBoolean("Graphics", "AnimatedSprites", prf.animatedSprites)

	return ini.bytes()
}

// preferencesLayout is the Preferences window. It is the project's design,
// not the original's, laid out as a Windows 98 dialog: the categories in
// a list on the left, the selected one's page on the right, and OK,
// Cancel and Apply below.
var preferencesLayout = layout.Component{
	Class:   "TPreferencesForm",
	Name:    "PreferencesForm",
	Caption: "Preferences",
	Props: map[string]string{
		"BorderIcons":  "[biSystemMenu]",
		"BorderStyle":  "bsDialog",
		"ClientHeight": "202",
		"ClientWidth":  "360",
		"Color":        "clBtnFace",
		"Position":     "poScreenCenter",
	},
	Children: []layout.Component{
		{
			Class:  "TListBox",
			Name:   "Categories",
			Bounds: layout.Rect{X: 8, Y: 8, W: 97, H: 153},
			Items:  []string{"Graphics"},
			Props: map[string]string{
				"ItemIndex": "0",
				"TabOrder":  "0",
			},
		},
		{
			Class:   "TGroupBox",
			Name:    "GraphicsPage",
			Bounds:  layout.Rect{X: 113, Y: 8, W: 239, H: 153},
			Caption: "Graphics",
			Props: map[string]string{
				"TabOrder": "1",
			},
			Children: []layout.Component{
				{
					Class:   "TCheckBox",
					Name:    "AnimatedSprites",
					Bounds:  layout.Rect{X: 12, Y: 22, W: 215, H: 17},
					Caption: "&Enable animated sprites",
					Props: map[string]string{
						"TabOrder": "0",
					},
				},
				{
					Class:   "TLabel",
					Name:    "AnimatedSpritesNote",
					Bounds:  layout.Rect{X: 29, Y: 42, W: 198, H: 39},
					Caption: "Plays POKéMON Emerald's animation when a POKéMON comes into the Battle Arena.",
					Props: map[string]string{
						"AutoSize": "False",
						"WordWrap": "True",
					},
				},
			},
		},
		{
			Class:   "TButton",
			Name:    "OKButton",
			Bounds:  layout.Rect{X: 115, Y: 169, W: 75, H: 25},
			Caption: "OK",
			Props: map[string]string{
				"Default":  "True",
				"TabOrder": "2",
			},
		},
		{
			Class:   "TButton",
			Name:    "CancelButton",
			Bounds:  layout.Rect{X: 196, Y: 169, W: 75, H: 25},
			Caption: "Cancel",
			Props: map[string]string{
				"Cancel":      "True",
				"ModalResult": "2",
				"TabOrder":    "3",
			},
		},
		{
			Class:    "TButton",
			Name:     "ApplyButton",
			Bounds:   layout.Rect{X: 277, Y: 169, W: 75, H: 25},
			Caption:  "&Apply",
			Disabled: true,
			Props: map[string]string{
				"TabOrder": "4",
			},
		},
	},
}

// preferencesForm is the Preferences window, File → Preferences..., which
// the original doesn't have.
type preferencesForm struct {
	app        *Application
	wdw        *window
	categories *listBox
	pages      []*groupBox // one per category, in the list's order
	animated   *checkBox
	apply      *button
}

func newPreferencesForm(app *Application) *preferencesForm {
	wdw := newWindow(app.desk, &preferencesLayout)
	pf := &preferencesForm{
		app:        app,
		wdw:        wdw,
		categories: get[*listBox](wdw, "Categories"),
		pages: []*groupBox{
			get[*groupBox](wdw, "GraphicsPage"),
		},
		animated: get[*checkBox](wdw, "AnimatedSprites"),
		apply:    get[*button](wdw, "ApplyButton"),
	}

	pf.categories.onSelect = pf.showPage
	pf.animated.onClick = pf.changed
	pf.apply.onClick = func() {
		pf.save()
	}
	get[*button](wdw, "OKButton").onClick = func() {
		if pf.save() {
			wdw.endModal(mrOk)
		}
	}

	app.desk.add(wdw)

	return pf
}

// open shows the player's preferences, with nothing to apply, at the first
// category. It runs on the handler's goroutine, which waits for the
// window to close.
func (pf *preferencesForm) open() {
	prf := pf.app.preferences
	pf.animated.setChecked(prf.animatedSprites)
	pf.apply.setEnabled(false)
	pf.categories.setIndex(0)
	pf.showPage()

	pf.wdw.focus = nil // the window starts at its first control
	pf.app.modal(pf.wdw)
}

// chosen returns the preferences as the window shows them.
func (pf *preferencesForm) chosen() preferences {
	return preferences{
		animatedSprites: pf.animated.checked,
	}
}

// showPage shows the selected category's page, and hides the others.
func (pf *preferencesForm) showPage() {
	for i, page := range pf.pages {
		page.setVisible(i == pf.categories.selected)
	}
}

// changed enables Apply when the window shows something to apply.
func (pf *preferencesForm) changed() {
	pf.apply.setEnabled(pf.chosen() != pf.app.preferences)
}

// save makes the window's choices the player's and keeps them, as OK and
// Apply do; it reports whether they could be kept, after telling the
// player why not. They hold for the run either way, and Apply stays
// enabled to try keeping them again.
func (pf *preferencesForm) save() bool {
	app := pf.app
	prf := pf.chosen()
	app.setPreferences(prf)

	dat, _ := app.settings.Read()
	if err := app.settings.Write(prf.store(dat)); err != nil {
		pf.fail("The preferences couldn't be saved: " + err.Error())

		return false
	}

	pf.apply.setEnabled(false)

	return true
}

// fail shows an error over the window, as a box of the user interface's
// own: the game's handler is waiting for the window to close.
func (pf *preferencesForm) fail(text string) {
	dsk := pf.app.desk
	box := messageDialog(dsk, game.Error, false, text)
	box.onModal = func(int) {
		dsk.remove(box)
	}
	dsk.showModal(box)
}

// iniFile is a settings file as Delphi's TIniFile reads and writes it:
// [Section] lines, each followed by its Key=Value lines. Sections and keys
// match whatever their case, and the lines it doesn't know are kept.
type iniFile struct {
	lines []string
}

func parseINI(dat []byte) *iniFile {
	txt := strings.ReplaceAll(string(dat), "\r\n", "\n")
	ini := &iniFile{}

	if txt = strings.TrimSuffix(txt, "\n"); txt != "" {
		ini.lines = strings.Split(txt, "\n")
	}

	return ini
}

// iniSection returns the name of the section that line starts, if it
// starts one.
func iniSection(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if len(line) < 2 || line[0] != '[' || line[len(line)-1] != ']' {
		return "", false
	}

	return strings.TrimSpace(line[1 : len(line)-1]), true
}

// iniKey returns the key that line sets, if it sets one.
func iniKey(line string) (string, bool) {
	key, _, ok := strings.Cut(line, "=")

	return strings.TrimSpace(key), ok
}

// value returns section's key, and whether it is there.
func (ini *iniFile) value(section, key string) (string, bool) {
	cur := ""

	for _, line := range ini.lines {
		if nam, ok := iniSection(line); ok {
			cur = nam

			continue
		}

		if !strings.EqualFold(cur, section) {
			continue
		}

		if k, ok := iniKey(line); ok && strings.EqualFold(k, key) {
			_, val, _ := strings.Cut(line, "=")

			return strings.TrimSpace(val), true
		}
	}

	return "", false
}

// boolean is ReadBool: section's key as an integer, true when not 0, or
// def when it is missing or isn't a number.
func (ini *iniFile) boolean(section, key string, def bool) bool {
	val, ok := ini.value(section, key)
	if !ok {
		return def
	}

	num, err := strconv.Atoi(val)
	if err != nil {
		return def
	}

	return num != 0
}

// set writes section's key: on its line if it has one, or else at the end
// of the section, or else in a new section at the end.
func (ini *iniFile) set(section, key, val string) {
	cur, end := "", -1 // end: where the section's next key would go

	for i, line := range ini.lines {
		if nam, ok := iniSection(line); ok {
			cur = nam

			if strings.EqualFold(nam, section) {
				end = i + 1
			}

			continue
		}

		if !strings.EqualFold(cur, section) {
			continue
		}

		if k, ok := iniKey(line); ok && strings.EqualFold(k, key) {
			ini.lines[i] = key + "=" + val

			return
		}

		if strings.TrimSpace(line) != "" {
			end = i + 1
		}
	}

	entry := key + "=" + val
	if end >= 0 {
		ini.lines = slices.Insert(ini.lines, end, entry)

		return
	}

	if len(ini.lines) > 0 {
		ini.lines = append(ini.lines, "")
	}

	ini.lines = append(ini.lines, "["+section+"]", entry)
}

// setBoolean is WriteBool: 1 or 0.
func (ini *iniFile) setBoolean(section, key string, val bool) {
	num := 0
	if val {
		num = 1
	}

	ini.set(section, key, strconv.Itoa(num))
}

func (ini *iniFile) bytes() []byte {
	return []byte(strings.Join(ini.lines, "\n") + "\n")
}
