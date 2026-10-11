// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TForm3 form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// Form3 is TForm3 ("Form3"), the loading screen.
var Form3 = Component{
	Class:   "TForm3",
	Name:    "Form3",
	Bounds:  Rect{194, 227, 0, 0},
	Caption: "Form3",
	Events: map[string]string{
		"OnHide": "FormHide",
	},
	Props: map[string]string{
		"AutoSize":     "True",
		"BorderStyle":  "bsNone",
		"ClientHeight": "157",
		"ClientWidth":  "510",
		"Color":        "clYellow",
		"Cursor":       "crHourGlass",
		"Font.Charset": "DEFAULT_CHARSET",
		"Font.Color":   "clWindowText",
		"Font.Height":  "-11",
		"Font.Name":    "MS Sans Serif",
		"Font.Style":   "[]",
		"Position":     "poScreenCenter",
	},
	Children: []Component{
		{
			Class:  "TGauge",
			Name:   "Gauge1",
			Bounds: Rect{58, 98, 441, 15},
			Props: map[string]string{
				"ForeColor": "clNavy",
				"MaxValue":  "1",
				"Progress":  "0",
			},
		},
		{
			Class:   "TLabel",
			Name:    "Label1",
			Bounds:  Rect{215, 80, 41, 13},
			Caption: "Loading:",
			Props: map[string]string{
				"Anchors": "[akTop, akRight]",
			},
		},
		{
			Class:  "TLabel",
			Name:   "Label2",
			Bounds: Rect{265, 80, 3, 13},
		},
		{
			Class:   "TLabel",
			Name:    "Title",
			Bounds:  Rect{0, 0, 510, 49},
			Caption: "POKéMON Simulator V4.5",
			Props: map[string]string{
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
			Class:   "TLabel",
			Name:    "Label3",
			Bounds:  Rect{11, 98, 44, 13},
			Caption: "Progress:",
			Props: map[string]string{
				"Anchors": "[akTop, akRight]",
			},
		},
		{
			Class:   "TLabel",
			Name:    "Label4",
			Bounds:  Rect{28, 122, 27, 13},
			Caption: "Total:",
			Props: map[string]string{
				"Anchors": "[akTop, akRight]",
			},
		},
		{
			Class:  "TGauge",
			Name:   "Gauge2",
			Bounds: Rect{58, 122, 441, 15},
			Props: map[string]string{
				"ForeColor": "clNavy",
				"MaxValue":  "3034",
				"Progress":  "0",
			},
		},
		{
			Class:   "TLabel",
			Name:    "Label5",
			Bounds:  Rect{148, 144, 214, 13},
			Caption: "Version 4.5, Copyright© BirdieSoft 2002-2004",
		},
		{
			Class:   "TLabel",
			Name:    "Label6",
			Bounds:  Rect{177, 40, 157, 32},
			Caption: "By BirdieSoft",
			Props: map[string]string{
				"Color":        "clTeal",
				"Font.Charset": "ANSI_CHARSET",
				"Font.Color":   "clRed",
				"Font.Height":  "-27",
				"Font.Name":    "Arial",
				"Font.Style":   "[]",
				"ParentColor":  "False",
				"ParentFont":   "False",
				"Transparent":  "True",
			},
		},
		{
			Class:  "TTimer",
			Name:   "Timer1",
			Bounds: Rect{8, 56, 0, 0},
			Events: map[string]string{
				"OnTimer": "TimerEventLoad",
			},
			Props: map[string]string{
				"Interval": "1",
			},
		},
	},
}
