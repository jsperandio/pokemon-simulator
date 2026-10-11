// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TAboutBox form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// AboutBox is TAboutBox ("POKéMON Simulator V4.5"), the About box.
var AboutBox = Component{
	Class:   "TAboutBox",
	Name:    "AboutBox",
	Bounds:  Rect{200, 108, 0, 0},
	Caption: "POKéMON Simulator V4.5",
	Props: map[string]string{
		"AutoSize":     "True",
		"BorderIcons":  "[]",
		"BorderStyle":  "bsDialog",
		"ClientHeight": "98",
		"ClientWidth":  "241",
		"Color":        "clBtnFace",
		"Font.Charset": "DEFAULT_CHARSET",
		"Font.Color":   "clWindowText",
		"Font.Height":  "-11",
		"Font.Name":    "MS Sans Serif",
		"Font.Style":   "[]",
		"Position":     "poScreenCenter",
	},
	Children: []Component{
		{
			Class:   "TButton",
			Name:    "OKButton",
			Bounds:  Rect{71, 73, 82, 25},
			Caption: "OK",
			Props: map[string]string{
				"Default":     "True",
				"ModalResult": "1",
				"TabOrder":    "0",
			},
		},
		{
			Class:  "TPanel",
			Name:   "Panel1",
			Bounds: Rect{0, 0, 241, 65},
			Props: map[string]string{
				"BevelOuter":  "bvNone",
				"ParentColor": "True",
				"TabOrder":    "1",
			},
			Children: []Component{
				{
					Class:   "TLabel",
					Name:    "ProductName",
					Bounds:  Rect{50, 2, 99, 13},
					Caption: "POKéMON Simulator",
				},
				{
					Class:   "TLabel",
					Name:    "Version",
					Bounds:  Rect{50, 26, 88, 13},
					Caption: "Version 4.5 (32-bit)",
				},
				{
					Class:  "TImage",
					Name:   "ProgramIcon",
					Bounds: Rect{2, 2, 32, 32},
					Props: map[string]string{
						"AutoSize": "True",
						"Picture":  "sprites/icon.png",
						"Stretch":  "True",
					},
				},
				{
					Class:   "TLabel",
					Name:    "Copyright",
					Bounds:  Rect{50, 50, 152, 13},
					Caption: "Copyright©BirdieSoft 2002-2004",
				},
			},
		},
	},
}
