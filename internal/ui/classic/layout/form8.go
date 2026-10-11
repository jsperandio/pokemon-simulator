// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TForm8 form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// Form8 is TForm8 ("POKéDEX"), the POKéDEX viewer.
var Form8 = Component{
	Class:   "TForm8",
	Name:    "Form8",
	Bounds:  Rect{279, 195, 0, 0},
	Caption: "POKéDEX",
	Events: map[string]string{
		"OnShow": "FormShow",
	},
	Props: map[string]string{
		"AutoSize":     "True",
		"BorderIcons":  "[]",
		"BorderStyle":  "bsToolWindow",
		"ClientHeight": "161",
		"ClientWidth":  "241",
		"Color":        "clBtnFace",
		"Font.Charset": "DEFAULT_CHARSET",
		"Font.Color":   "clWindowText",
		"Font.Height":  "-11",
		"Font.Name":    "MS Sans Serif",
		"Font.Style":   "[]",
	},
	Children: []Component{
		{
			Class:  "TComboBox",
			Name:   "ComboBox1",
			Bounds: Rect{0, 0, 145, 21},
			Events: map[string]string{
				"OnChange": "ComboBox1Change",
			},
			Props: map[string]string{
				"ItemHeight": "13",
				"TabOrder":   "0",
			},
		},
		{
			Class:  "TPanel",
			Name:   "Panel1",
			Bounds: Rect{0, 24, 241, 137},
			Hidden: true,
			Props: map[string]string{
				"TabOrder": "1",
			},
			Children: []Component{
				{
					Class:   "TLabel",
					Name:    "Label1",
					Bounds:  Rect{72, 8, 28, 13},
					Caption: "Name",
				},
				{
					Class:   "TLabel",
					Name:    "Label2",
					Bounds:  Rect{72, 24, 30, 13},
					Caption: "Type1",
				},
				{
					Class:   "TLabel",
					Name:    "Label3",
					Bounds:  Rect{120, 24, 30, 13},
					Caption: "Type2",
				},
				{
					Class:   "TLabel",
					Name:    "Label4",
					Bounds:  Rect{72, 120, 94, 13},
					Caption: "Can evolve at level ",
				},
				{
					Class:   "TLabel",
					Name:    "Label5",
					Bounds:  Rect{72, 48, 43, 13},
					Caption: "Attack 1:",
				},
				{
					Class:   "TLabel",
					Name:    "Label6",
					Bounds:  Rect{120, 48, 32, 13},
					Caption: "Label6",
				},
				{
					Class:   "TLabel",
					Name:    "Label7",
					Bounds:  Rect{72, 64, 43, 13},
					Caption: "Attack 2:",
				},
				{
					Class:   "TLabel",
					Name:    "Label8",
					Bounds:  Rect{72, 80, 43, 13},
					Caption: "Attack 3:",
				},
				{
					Class:   "TLabel",
					Name:    "Label9",
					Bounds:  Rect{72, 96, 43, 13},
					Caption: "Attack 4:",
				},
				{
					Class:   "TLabel",
					Name:    "Label10",
					Bounds:  Rect{120, 64, 38, 13},
					Caption: "Label10",
				},
				{
					Class:   "TLabel",
					Name:    "Label11",
					Bounds:  Rect{120, 80, 38, 13},
					Caption: "Label11",
				},
				{
					Class:   "TLabel",
					Name:    "Label12",
					Bounds:  Rect{120, 96, 38, 13},
					Caption: "Label12",
				},
				{
					Class:   "TLabel",
					Name:    "Label13",
					Bounds:  Rect{200, 48, 38, 13},
					Caption: "Label13",
				},
				{
					Class:   "TLabel",
					Name:    "Label14",
					Bounds:  Rect{200, 64, 38, 13},
					Caption: "Label14",
				},
				{
					Class:   "TLabel",
					Name:    "Label15",
					Bounds:  Rect{200, 80, 38, 13},
					Caption: "Label15",
				},
				{
					Class:   "TLabel",
					Name:    "Label16",
					Bounds:  Rect{200, 96, 38, 13},
					Caption: "Label16",
				},
				{
					Class:  "TListView",
					Name:   "ListView1",
					Bounds: Rect{4, 8, 65, 65},
					List:   []ListItem{{Caption: "", ImageIndex: 0}},
					Props: map[string]string{
						"BevelEdges":  "[]",
						"BevelInner":  "bvNone",
						"BevelOuter":  "bvNone",
						"BorderStyle": "bsNone",
						"Color":       "clBtnFace",
						"Columns":     "<>",
						"LargeImages": "Form4.PKMNImages",
						"SmallImages": "Form4.PKMNImages",
						"TabOrder":    "0",
						"ViewStyle":   "vsList",
					},
				},
			},
		},
		{
			Class:   "TButton",
			Name:    "Button1",
			Bounds:  Rect{144, 0, 97, 25},
			Caption: "Close POKéDEX",
			Events: map[string]string{
				"OnClick": "Button1Click",
			},
			Props: map[string]string{
				"TabOrder": "2",
			},
		},
	},
}
