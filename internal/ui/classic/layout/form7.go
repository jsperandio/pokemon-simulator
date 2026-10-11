// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TForm7 form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// Form7 is TForm7 ("Battle Info"), the Battle Info log.
var Form7 = Component{
	Class:   "TForm7",
	Name:    "Form7",
	Bounds:  Rect{189, 419, 0, 0},
	Caption: "Battle Info",
	Events: map[string]string{
		"OnShow": "FormShow",
	},
	Props: map[string]string{
		"BorderIcons":  "[]",
		"BorderStyle":  "bsSingle",
		"ClientHeight": "159",
		"ClientWidth":  "422",
		"Color":        "clBtnFace",
		"Font.Charset": "DEFAULT_CHARSET",
		"Font.Color":   "clWindowText",
		"Font.Height":  "-11",
		"Font.Name":    "MS Sans Serif",
		"Font.Style":   "[]",
	},
	Children: []Component{
		{
			Class:  "TMemo",
			Name:   "Memo1",
			Bounds: Rect{0, 0, 422, 159},
			Items:  []string{""},
			Props: map[string]string{
				"Align":      "alClient",
				"Color":      "clBtnFace",
				"ReadOnly":   "True",
				"ScrollBars": "ssVertical",
				"TabOrder":   "0",
			},
		},
	},
}
