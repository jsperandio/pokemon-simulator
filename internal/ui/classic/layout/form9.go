// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TForm9 form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// Form9 is TForm9 ("Welcome to POKéMON Mart! Click an item for a
// description."), the POKéMON Mart.
var Form9 = Component{
	Class:   "TForm9",
	Name:    "Form9",
	Bounds:  Rect{137, 206, 0, 0},
	Caption: "Welcome to POKéMON Mart! Click an item for a description.",
	Events: map[string]string{
		"OnShow": "FormShow",
	},
	Props: map[string]string{
		"BorderIcons":  "[]",
		"BorderStyle":  "bsSingle",
		"ClientHeight": "65",
		"ClientWidth":  "471",
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
			Bounds:  Rect{0, 4, 21, 13},
			Caption: "Buy:",
		},
		{
			Class:  "TLabel",
			Name:   "Label2",
			Bounds: Rect{0, 24, 3, 13},
			Props: map[string]string{
				"Constraints.MaxWidth": "145",
			},
		},
		{
			Class:  "TLabel",
			Name:   "Label3",
			Bounds: Rect{0, 40, 3, 13},
			Props: map[string]string{
				"Constraints.MaxWidth": "145",
			},
		},
		{
			Class:   "TLabel",
			Name:    "Label4",
			Bounds:  Rect{344, 0, 27, 13},
			Caption: "Price:",
		},
		{
			Class:  "TSpinEdit",
			Name:   "SpinEdit1",
			Bounds: Rect{24, 0, 121, 22},
			Events: map[string]string{
				"OnChange": "SpinEdit1Change",
			},
			Props: map[string]string{
				"MaxValue": "2147483647",
				"MinValue": "0",
				"TabOrder": "0",
				"Value":    "0",
			},
		},
		{
			Class:   "TRadioGroup",
			Name:    "RadioGroup1",
			Bounds:  Rect{152, 0, 185, 65},
			Caption: "Item to buy:",
			Items: []string{
				"POKéBalls",
				"Great Balls",
				"Ultra Balls",
				"Potions",
				"Super Potions",
			},
			Events: map[string]string{
				"OnClick": "RadioGroup1Click",
			},
			Props: map[string]string{
				"Columns":  "2",
				"TabOrder": "1",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button1",
			Bounds:  Rect{424, 0, 41, 17},
			Caption: "Buy",
			Events: map[string]string{
				"OnClick": "Button1Click",
			},
			Props: map[string]string{
				"TabOrder": "2",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button2",
			Bounds:  Rect{424, 16, 41, 17},
			Caption: "Leave",
			Events: map[string]string{
				"OnClick": "Button2Click",
			},
			Props: map[string]string{
				"TabOrder": "3",
			},
		},
		{
			Class:    "TEdit",
			Name:     "Edit1",
			Bounds:   Rect{344, 16, 73, 21},
			Disabled: true,
			Props: map[string]string{
				"TabOrder": "4",
			},
		},
	},
}
