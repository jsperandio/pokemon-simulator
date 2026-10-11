// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TFormB form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// FormB is TFormB ("Game Corner"), the Game Corner.
var FormB = Component{
	Class:   "TFormB",
	Name:    "FormB",
	Bounds:  Rect{192, 107, 0, 0},
	Caption: "Game Corner",
	Events: map[string]string{
		"OnShow": "FormShow",
	},
	Props: map[string]string{
		"BorderIcons":  "[]",
		"BorderStyle":  "bsSingle",
		"ClientHeight": "153",
		"ClientWidth":  "385",
		"Color":        "clBtnFace",
		"Font.Charset": "DEFAULT_CHARSET",
		"Font.Color":   "clWindowText",
		"Font.Height":  "-11",
		"Font.Name":    "MS Sans Serif",
		"Font.Style":   "[]",
	},
	Children: []Component{
		{
			Class:   "TLabel",
			Name:    "Label1",
			Bounds:  Rect{48, 4, 243, 13},
			Caption: "Amount to bet:                                                        ¥",
		},
		{
			Class:  "TSpinEdit",
			Name:   "SpinEdit1",
			Bounds: Rect{192, 0, 89, 22},
			Props: map[string]string{
				"Increment": "50",
				"MaxValue":  "0",
				"MinValue":  "0",
				"TabOrder":  "0",
				"Value":     "0",
			},
		},
		{
			Class:   "TRadioGroup",
			Name:    "RadioGroup1",
			Bounds:  Rect{0, 24, 193, 129},
			Caption: "Play for:",
			Items:   []string{"Money (Pays out 10:1¥)", "POKéBalls (Pays out 1:100¥)", "Great Balls (Pays out 1:300¥)", "Ultra Balls (Pays out 1:600¥)", "Master Balls (Pays out 1:2500¥)", "Potions (Pays out 1:200¥)", "Super Potions (Pays out 1:500¥)"},
			Props: map[string]string{
				"TabOrder": "1",
			},
		},
		{
			Class:   "TRadioGroup",
			Name:    "RadioGroup2",
			Bounds:  Rect{200, 24, 89, 129},
			Caption: "Pick a number:",
			Items:   []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			Props: map[string]string{
				"Columns":  "2",
				"TabOrder": "2",
			},
		},
		{
			Class:   "TRadioGroup",
			Name:    "RadioGroup3",
			Bounds:  Rect{296, 0, 89, 129},
			Caption: "Lucky number:",
			Items:   []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			Props: map[string]string{
				"Columns":  "2",
				"TabOrder": "3",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button1",
			Bounds:  Rect{296, 128, 89, 25},
			Caption: "Draw number!",
			Events: map[string]string{
				"OnClick": "Button1Click",
			},
			Props: map[string]string{
				"TabOrder": "4",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button2",
			Bounds:  Rect{0, 0, 41, 25},
			Caption: "Close",
			Events: map[string]string{
				"OnClick": "Button2Click",
			},
			Props: map[string]string{
				"TabOrder": "5",
			},
		},
	},
}
