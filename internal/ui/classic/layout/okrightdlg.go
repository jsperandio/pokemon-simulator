// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TOKRightDlg form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// OKRightDlg is TOKRightDlg ("Enter your name:"), the name dialog.
var OKRightDlg = Component{
	Class:   "TOKRightDlg",
	Name:    "OKRightDlg",
	Bounds:  Rect{313, 270, 0, 0},
	Caption: "Enter your name:",
	Props: map[string]string{
		"BorderIcons":  "[]",
		"BorderStyle":  "bsDialog",
		"ClientHeight": "25",
		"ClientWidth":  "153",
		"Color":        "clBtnFace",
		"ParentFont":   "True",
		"Position":     "poScreenCenter",
	},
	Children: []Component{
		{
			Class:   "TButton",
			Name:    "OKBtn",
			Bounds:  Rect{100, 0, 53, 25},
			Caption: "OK",
			Events: map[string]string{
				"OnClick": "OKBtnClick",
			},
			Props: map[string]string{
				"Default":     "True",
				"ModalResult": "1",
				"TabOrder":    "1",
			},
		},
		{
			Class:   "TEdit",
			Name:    "Edit1",
			Bounds:  Rect{0, 2, 97, 21},
			Caption: "Ash",
			Props: map[string]string{
				"MaxLength": "15",
				"TabOrder":  "0",
			},
		},
	},
}
