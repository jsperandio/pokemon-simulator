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
	animatedSprites bool      // Graphics: Enable animated sprites
	shinyMark       shinyMark // Graphics: the shiny mark in the POKéMON lists
	choosePrompt    bool      // Messages: Select a POKéMON reminder, game.ChoosePrompt
	fillScreen      bool      // Layout: Fill the screen, rather than separate windows
	colourMessages  bool      // Battle Info: Colour the messages
	whiteBackground bool      // Battle Info: a white background, rather than grey
}

// defaultPreferences are the choices before the player makes any.
func defaultPreferences() preferences {
	return preferences{
		animatedSprites: true,
		shinyMark:       goldSparkles,
		choosePrompt:    true,
		colourMessages:  true,
	}
}

// loadPreferences reads the preferences from dat, an INI file; what is
// missing or unreadable keeps its default.
func loadPreferences(dat []byte) preferences {
	prf := defaultPreferences()
	ini := parseINI(dat)
	prf.animatedSprites = ini.boolean("Graphics", "AnimatedSprites", prf.animatedSprites)
	prf.choosePrompt = ini.boolean("Messages", "SelectPokemon", prf.choosePrompt)
	prf.fillScreen = ini.boolean("Layout", "FillScreen", prf.fillScreen)
	prf.colourMessages = ini.boolean("BattleInfo", "ColourMessages", prf.colourMessages)
	prf.whiteBackground = ini.boolean("BattleInfo", "WhiteBackground", prf.whiteBackground)

	if key, ok := ini.value("Graphics", "ShinyMark"); ok {
		if sm, known := markByKey(key); known {
			prf.shinyMark = sm
		}
	}

	return prf
}

// store writes the preferences into dat, an INI file, keeping the lines it
// doesn't know.
func (prf preferences) store(dat []byte) []byte {
	ini := parseINI(dat)
	ini.setBoolean("Graphics", "AnimatedSprites", prf.animatedSprites)
	ini.set("Graphics", "ShinyMark", shinyMarks[prf.shinyMark].key)
	ini.setBoolean("Messages", "SelectPokemon", prf.choosePrompt)
	ini.setBoolean("Layout", "FillScreen", prf.fillScreen)
	ini.setBoolean("BattleInfo", "ColourMessages", prf.colourMessages)
	ini.setBoolean("BattleInfo", "WhiteBackground", prf.whiteBackground)

	return ini.bytes()
}

// preferencesLayout is the Preferences window. It is the project's design,
// not the original's, laid out as a Windows 98 dialog: the categories in
// a list on the left, the selected one's page on the right, Restore
// Defaults under the list, and OK, Cancel and Apply below the page.
var preferencesLayout = layout.Component{
	Class:   "TPreferencesForm",
	Name:    "PreferencesForm",
	Caption: "Preferences",
	Props: map[string]string{
		"BorderIcons":  "[biSystemMenu]",
		"BorderStyle":  "bsDialog",
		"ClientHeight": "230",
		"ClientWidth":  "360",
		"Color":        "clBtnFace",
		"Position":     "poScreenCenter",
	},
	Children: []layout.Component{
		{
			Class:  "TListBox",
			Name:   "Categories",
			Bounds: layout.Rect{X: 8, Y: 8, W: 97, H: 181},
			Items:  []string{"Graphics", "Messages", "Layout", "Battle Info"},
			Props: map[string]string{
				"ItemIndex": "0",
				"TabOrder":  "0",
			},
		},
		{
			Class:   "TGroupBox",
			Name:    "GraphicsPage",
			Bounds:  layout.Rect{X: 113, Y: 8, W: 239, H: 181},
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
					Caption: "Plays POKéMON Emerald's animation when a POKéMON comes into the Battle Arena or is chosen in the POKéDEX.",
					Props: map[string]string{
						"AutoSize": "False",
						"WordWrap": "True",
					},
				},
				{
					Class:   "TLabel",
					Name:    "ShinyMarkLabel",
					Bounds:  layout.Rect{X: 12, Y: 90, W: 215, H: 13},
					Caption: "Shiny mark in the POKéMON lists:",
				},
				{
					Class:  "TComboBox",
					Name:   "ShinyMark",
					Bounds: layout.Rect{X: 12, Y: 106, W: 215, H: 21},
					Props: map[string]string{
						"Style":    "csDropDownList",
						"TabOrder": "1",
					},
				},
				{
					Class:  "TListView",
					Name:   "ShinyMarkPreview",
					Bounds: layout.Rect{X: 12, Y: 133, W: 215, H: 38},
					List: []layout.ListItem{
						{
							Caption:    "Gyarados (shiny)",
							ImageIndex: 129,
						},
					},
					Props: map[string]string{
						"SmallImages": "SmallPKMNImg",
						"TabStop":     "False",
						"ViewStyle":   "vsList",
					},
				},
			},
		},
		{
			Class:   "TGroupBox",
			Name:    "MessagesPage",
			Bounds:  layout.Rect{X: 113, Y: 8, W: 239, H: 181},
			Caption: "Messages",
			Hidden:  true,
			Props: map[string]string{
				"TabOrder": "2",
			},
			Children: []layout.Component{
				{
					Class:   "TCheckBox",
					Name:    "ChoosePrompt",
					Bounds:  layout.Rect{X: 12, Y: 22, W: 215, H: 17},
					Caption: "&Select a POKéMON reminder",
					Props: map[string]string{
						"TabOrder": "0",
					},
				},
				{
					Class:   "TLabel",
					Name:    "ChoosePromptNote",
					Bounds:  layout.Rect{X: 29, Y: 42, W: 198, H: 39},
					Caption: "Shows \"Select a POKéMON from the Statistics Window\" when a battle starts.",
					Props: map[string]string{
						"AutoSize": "False",
						"WordWrap": "True",
					},
				},
			},
		},
		{
			Class:   "TGroupBox",
			Name:    "LayoutPage",
			Bounds:  layout.Rect{X: 113, Y: 8, W: 239, H: 181},
			Caption: "Layout",
			Hidden:  true,
			Props: map[string]string{
				"TabOrder": "3",
			},
			Children: []layout.Component{
				{
					Class:   "TRadioGroup",
					Name:    "Windows",
					Bounds:  layout.Rect{X: 12, Y: 22, W: 215, H: 65},
					Caption: "Windows:",
					Items:   []string{"Separate windows (classic)", "Fill the screen"},
					Props: map[string]string{
						"TabOrder": "0",
						"TabStop":  "True",
					},
				},
				{
					Class:   "TLabel",
					Name:    "WindowsNote",
					Bounds:  layout.Rect{X: 12, Y: 96, W: 215, H: 65},
					Caption: "Fill the screen puts Statistics down the right edge and Battle Info along the bottom, gives the game window the rest, and keeps them in place.",
					Props: map[string]string{
						"AutoSize": "False",
						"WordWrap": "True",
					},
				},
			},
		},
		{
			Class:   "TGroupBox",
			Name:    "BattleInfoPage",
			Bounds:  layout.Rect{X: 113, Y: 8, W: 239, H: 181},
			Caption: "Battle Info",
			Hidden:  true,
			Props: map[string]string{
				"TabOrder": "4",
			},
			Children: []layout.Component{
				{
					Class:   "TCheckBox",
					Name:    "ColourMessages",
					Bounds:  layout.Rect{X: 12, Y: 22, W: 215, H: 17},
					Caption: "&Colour the messages",
					Props: map[string]string{
						"TabOrder": "0",
					},
				},
				{
					Class:   "TLabel",
					Name:    "ColourMessagesNote",
					Bounds:  layout.Rect{X: 29, Y: 42, W: 198, H: 26},
					Caption: "Damage you deal in green, damage you take in red, experience in blue.",
					Props: map[string]string{
						"AutoSize": "False",
						"WordWrap": "True",
					},
				},
				{
					Class:   "TRadioGroup",
					Name:    "Background",
					Bounds:  layout.Rect{X: 12, Y: 72, W: 215, H: 38},
					Caption: "Background:",
					Items:   []string{"Grey (classic)", "White"},
					Props: map[string]string{
						"Columns":  "2",
						"TabOrder": "1",
						"TabStop":  "True",
					},
				},
				{
					Class:  "TMemo",
					Name:   "BattleInfoPreview",
					Bounds: layout.Rect{X: 12, Y: 114, W: 215, H: 58}, // four lines, unscrolled
					Props: map[string]string{
						"Color":    "clBtnFace",
						"ReadOnly": "True",
						"TabStop":  "False",
					},
				},
			},
		},
		{
			Class:   "TButton",
			Name:    "RestoreButton",
			Bounds:  layout.Rect{X: 8, Y: 197, W: 97, H: 25},
			Caption: "Restore &Defaults",
			Props: map[string]string{
				"TabOrder": "5",
			},
		},
		{
			Class:   "TButton",
			Name:    "OKButton",
			Bounds:  layout.Rect{X: 115, Y: 197, W: 75, H: 25},
			Caption: "OK",
			Props: map[string]string{
				"Default":  "True",
				"TabOrder": "6",
			},
		},
		{
			Class:   "TButton",
			Name:    "CancelButton",
			Bounds:  layout.Rect{X: 196, Y: 197, W: 75, H: 25},
			Caption: "Cancel",
			Props: map[string]string{
				"Cancel":      "True",
				"ModalResult": "2",
				"TabOrder":    "7",
			},
		},
		{
			Class:    "TButton",
			Name:     "ApplyButton",
			Bounds:   layout.Rect{X: 277, Y: 197, W: 75, H: 25},
			Caption:  "&Apply",
			Disabled: true,
			Props: map[string]string{
				"TabOrder": "8",
			},
		},
	},
}

// The Layout page's Windows choices.
const (
	windowsSeparate int = iota
	windowsFill
)

// The Battle Info page's Background choices.
const (
	backgroundGrey int = iota
	backgroundWhite
)

// The Battle Info page's preview: a line of each colour its note names,
// and a milestone, with their kinds.
var (
	previewLines = []string{
		"Mew did 8 points of damage to enemy Onix.",
		"Enemy Onix did 9 points of damage to Mew.",
		"Mew gained 397 experience points!",
		"Mew grew to level 13!",
	}
	previewKinds = []game.LineKind{
		game.DamageDealt,
		game.DamageTaken,
		game.ExperienceGained,
		game.LevelUp,
	}
)

// preferencesForm is the Preferences window, File → Preferences..., which
// the original doesn't have.
type preferencesForm struct {
	app        *Application
	wdw        *window
	categories *listBox
	pages      []*groupBox // one per category, in the list's order
	animated   *checkBox
	mark       *comboBox
	preview    *listView // a shiny Gyarados with the mark chosen
	prompt     *checkBox
	windows    *radioGroup // separate windows, or filling the screen
	colours    *checkBox   // Colour the messages
	background *radioGroup // Battle Info's grey or white
	logPreview *memo       // Battle Info's lines as the page has them
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
			get[*groupBox](wdw, "MessagesPage"),
			get[*groupBox](wdw, "LayoutPage"),
			get[*groupBox](wdw, "BattleInfoPage"),
		},
		animated:   get[*checkBox](wdw, "AnimatedSprites"),
		mark:       get[*comboBox](wdw, "ShinyMark"),
		preview:    get[*listView](wdw, "ShinyMarkPreview"),
		prompt:     get[*checkBox](wdw, "ChoosePrompt"),
		windows:    get[*radioGroup](wdw, "Windows"),
		colours:    get[*checkBox](wdw, "ColourMessages"),
		background: get[*radioGroup](wdw, "Background"),
		logPreview: get[*memo](wdw, "BattleInfoPreview"),
		apply:      get[*button](wdw, "ApplyButton"),
	}

	pf.categories.onSelect = pf.showPage
	pf.animated.onClick = pf.changed
	pf.mark.items = markNames()
	pf.mark.onSelect = func(int) {
		pf.preview.invalidate()
		pf.changed()
	}
	pf.preview.items[0].shiny = true
	pf.preview.showOnly = true
	pf.preview.mark = func(cv canvas, x, y int) {
		shinyMark(pf.mark.index).draw(cv, x, y)
	}
	pf.prompt.onClick = pf.changed
	pf.windows.onClick = pf.changed
	pf.colours.onClick = pf.restyled
	pf.background.onClick = pf.restyled
	get[*button](wdw, "RestoreButton").onClick = pf.restore
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
	pf.showChoices(pf.app.preferences)
	pf.apply.setEnabled(false)
	pf.categories.setIndex(0)
	pf.showPage()

	pf.wdw.focus = nil // the window starts at its first control
	pf.app.modal(pf.wdw)
}

// showChoices sets the controls to prf.
func (pf *preferencesForm) showChoices(prf preferences) {
	pf.animated.setChecked(prf.animatedSprites)
	pf.mark.setIndex(int(prf.shinyMark))
	pf.preview.invalidate()
	pf.prompt.setChecked(prf.choosePrompt)
	pf.windows.setIndex(windowsSeparate)

	if prf.fillScreen {
		pf.windows.setIndex(windowsFill)
	}

	pf.colours.setChecked(prf.colourMessages)
	pf.background.setIndex(backgroundGrey)

	if prf.whiteBackground {
		pf.background.setIndex(backgroundWhite)
	}

	pf.showPreview()
}

// chosen returns the preferences as the window shows them.
func (pf *preferencesForm) chosen() preferences {
	return preferences{
		animatedSprites: pf.animated.checked,
		shinyMark:       shinyMark(pf.mark.index),
		choosePrompt:    pf.prompt.checked,
		fillScreen:      pf.windows.index == windowsFill,
		colourMessages:  pf.colours.checked,
		whiteBackground: pf.background.index == backgroundWhite,
	}
}

// showPreview shows the Battle Info page's preview as the page's choices
// would show Battle Info.
func (pf *preferencesForm) showPreview() {
	prf := pf.chosen()

	pf.logPreview.setColor(logBackground(prf.whiteBackground))
	pf.logPreview.setLines(previewLines, logStyles(previewKinds, prf))
}

// restyled follows a change of Battle Info's colours or background: the
// preview shows it, and Apply can keep it.
func (pf *preferencesForm) restyled() {
	pf.showPreview()
	pf.changed()
}

// restore is Restore Defaults: every page's controls go back to the
// defaults, which OK or Apply keep, as any change.
func (pf *preferencesForm) restore() {
	pf.showChoices(defaultPreferences())
	pf.changed()
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
	if err := pf.app.keepPreferences(pf.chosen()); err != nil {
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
	box := messageDialog(dsk, game.Error, false, text, "")
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
