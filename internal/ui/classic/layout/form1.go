// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TForm1 form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

import "github.com/jsperandio/pokemon-simulator/internal/version"

// Form1 is TForm1 (the program's title), the start menu.
var Form1 = Component{
	Class:   "TForm1",
	Name:    "Form1",
	Bounds:  Rect{192, 107, 544, 393},
	Caption: version.Title(),
	Events: map[string]string{
		"OnClose":      "FormClose",
		"OnDeactivate": "FormDeactivate",
	},
	Props: map[string]string{
		"AutoSize":     "True",
		"Color":        "clYellow",
		"Font.Charset": "DEFAULT_CHARSET",
		"Font.Color":   "clWindowText",
		"Font.Height":  "-11",
		"Font.Name":    "MS Sans Serif",
		"Font.Style":   "[]",
		"Position":     "poDesktopCenter",
	},
	Children: []Component{
		{
			Class:   "TLabel",
			Name:    "Title",
			Bounds:  Rect{0, 0, 536, 49},
			Caption: version.Title(),
			Props: map[string]string{
				"Align":        "alTop",
				"Alignment":    "taCenter",
				"Color":        "clTeal",
				"Font.Charset": "ANSI_CHARSET",
				"Font.Color":   "clRed",
				"Font.Height":  "-43",
				"Font.Name":    "Arial",
				"Font.Style":   "[]",
				"ParentColor":  "False",
				"ParentFont":   "False",
				"Transparent":  "True",
			},
		},
		{
			Class:   "TButton",
			Name:    "NewButton",
			Bounds:  Rect{0, 45, 536, 65},
			Caption: "Start a new game!",
			Events: map[string]string{
				"OnClick": "NewButtonClick",
			},
			Props: map[string]string{
				"TabOrder": "0",
			},
		},
		{
			Class:   "TButton",
			Name:    "LoadButton",
			Bounds:  Rect{0, 109, 536, 65},
			Caption: "Load a saved game...",
			Events: map[string]string{
				"OnClick": "LoadButtonClick",
			},
			Props: map[string]string{
				"TabOrder": "1",
			},
		},
		{
			Class:   "TButton",
			Name:    "HelpButton",
			Bounds:  Rect{0, 173, 536, 65},
			Caption: "Help!",
			Events: map[string]string{
				"OnClick": "HelpButtonClick",
			},
			Props: map[string]string{
				"TabOrder": "2",
			},
		},
		{
			Class:   "TButton",
			Name:    "AboutButton",
			Bounds:  Rect{0, 237, 536, 65},
			Caption: "About...",
			Events: map[string]string{
				"OnClick": "AboutButtonClick",
			},
			Props: map[string]string{
				"TabOrder": "3",
			},
		},
		{
			Class:   "TButton",
			Name:    "ExitButton",
			Bounds:  Rect{0, 301, 536, 65},
			Caption: "Exit",
			Events: map[string]string{
				"OnClick": "ExitButtonClick",
			},
			Props: map[string]string{
				"TabOrder": "4",
			},
		},
		{
			Class:  "TOpenDialog",
			Name:   "OpenDialog1",
			Bounds: Rect{88, 16, 0, 0},
			Props: map[string]string{
				"DefaultExt":  "*.pgs",
				"Filter":      "Saved Games (*.PGS)|*.pgs",
				"FilterIndex": "0",
				"Options":     "[ofHideReadOnly, ofPathMustExist, ofFileMustExist, ofEnableSizing, ofDontAddToRecent]",
			},
		},
	},
}
