// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TFormA form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// FormA is TFormA ("Type Comparisons"), the Type Comparisons tool.
var FormA = Component{
	Class:   "TFormA",
	Name:    "FormA",
	Bounds:  Rect{192, 107, 0, 0},
	Caption: "Type Comparisons",
	Events: map[string]string{
		"OnShow": "FormShow",
	},
	Props: map[string]string{
		"AutoSize":     "True",
		"BorderIcons":  "[]",
		"BorderStyle":  "bsToolWindow",
		"ClientHeight": "41",
		"ClientWidth":  "377",
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
			Bounds:  Rect{0, 24, 80, 13},
			Caption: "Type 1 Multiplier:",
			Hidden:  true,
		},
		{
			Class:   "TLabel",
			Name:    "Label2",
			Bounds:  Rect{108, 24, 80, 13},
			Caption: "Type 2 Multiplier:",
			Hidden:  true,
		},
		{
			Class:   "TLabel",
			Name:    "Label3",
			Bounds:  Rect{86, 24, 6, 13},
			Caption: "1",
			Hidden:  true,
		},
		{
			Class:   "TLabel",
			Name:    "Label4",
			Bounds:  Rect{195, 24, 6, 13},
			Caption: "1",
			Hidden:  true,
		},
		{
			Class:   "TLabel",
			Name:    "Label5",
			Bounds:  Rect{217, 24, 27, 13},
			Caption: "Total:",
			Hidden:  true,
		},
		{
			Class:   "TLabel",
			Name:    "Label6",
			Bounds:  Rect{248, 24, 28, 13},
			Caption: "1*1=1",
			Hidden:  true,
		},
		{
			Class:   "TLabel",
			Name:    "Label7",
			Bounds:  Rect{0, 4, 34, 13},
			Caption: "Attack:",
		},
		{
			Class:   "TLabel",
			Name:    "Label8",
			Bounds:  Rect{176, 4, 56, 13},
			Caption: "POKéMON:",
		},
		{
			Class:  "TComboBox",
			Name:   "ComboBox1",
			Bounds: Rect{232, 0, 145, 21},
			Events: map[string]string{
				"OnChange": "ComboBox1Change",
			},
			Props: map[string]string{
				"ItemHeight": "13",
				"TabOrder":   "1",
			},
		},
		{
			Class:  "TComboBox",
			Name:   "ComboBox2",
			Bounds: Rect{34, 0, 135, 21},
			Events: map[string]string{
				"OnChange": "ComboBox1Change",
			},
			Props: map[string]string{
				"ItemHeight": "13",
				"TabOrder":   "0",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button1",
			Bounds:  Rect{288, 24, 89, 17},
			Caption: "Close",
			Events: map[string]string{
				"OnClick": "Button1Click",
			},
			Props: map[string]string{
				"TabOrder": "2",
			},
		},
	},
}
