// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TForm5 form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// Form5 is TForm5 ("Statistics"), the Statistics window, with the item
// buttons and the owned list.
var Form5 = Component{
	Class:   "TForm5",
	Name:    "Form5",
	Bounds:  Rect{617, 105, 0, 0},
	Caption: "Statistics",
	Events: map[string]string{
		"OnClick":    "FormClick",
		"OnDblClick": "FormClick",
	},
	Props: map[string]string{
		"BorderIcons":  "[]",
		"BorderStyle":  "bsSingle",
		"ClientHeight": "473",
		"ClientWidth":  "177",
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
			Bounds:  Rect{0, 0, 31, 13},
			Caption: "Name:",
		},
		{
			Class:   "TLabel",
			Name:    "Label2",
			Bounds:  Rect{0, 16, 35, 13},
			Caption: "Money:",
		},
		{
			Class:   "TLabel",
			Name:    "Label3",
			Bounds:  Rect{0, 48, 27, 13},
			Caption: "Wins:",
		},
		{
			Class:   "TLabel",
			Name:    "Label4",
			Bounds:  Rect{0, 80, 66, 13},
			Caption: "Losses/Flees:",
		},
		{
			Class:   "TLabel",
			Name:    "Label5",
			Bounds:  Rect{0, 96, 80, 13},
			Caption: "Win Percentage:",
		},
		{
			Class:   "TLabel",
			Name:    "Label6",
			Bounds:  Rect{0, 128, 28, 13},
			Caption: "Items:",
		},
		{
			Class:   "TLabel",
			Name:    "Label7",
			Bounds:  Rect{0, 152, 53, 13},
			Caption: "POKéBalls:",
		},
		{
			Class:   "TLabel",
			Name:    "Label8",
			Bounds:  Rect{0, 168, 54, 13},
			Caption: "Great Balls:",
		},
		{
			Class:   "TLabel",
			Name:    "Label9",
			Bounds:  Rect{0, 184, 50, 13},
			Caption: "Ultra Balls:",
		},
		{
			Class:   "TLabel",
			Name:    "Label10",
			Bounds:  Rect{0, 200, 60, 13},
			Caption: "Master Balls:",
		},
		{
			Class:   "TLabel",
			Name:    "Label11",
			Bounds:  Rect{0, 216, 38, 13},
			Caption: "Potions:",
		},
		{
			Class:   "TLabel",
			Name:    "Label12",
			Bounds:  Rect{0, 232, 69, 13},
			Caption: "Super Potions:",
		},
		{
			Class:  "TLabel",
			Name:   "PlayerNameLbl",
			Bounds: Rect{88, 0, 3, 13},
		},
		{
			Class:   "TLabel",
			Name:    "MoneyLbl",
			Bounds:  Rect{88, 16, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "FightsLbl",
			Bounds:  Rect{88, 32, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "LossesLbl",
			Bounds:  Rect{88, 80, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "Percent",
			Bounds:  Rect{88, 96, 6, 13},
			Caption: "0",
			Props: map[string]string{
				"Constraints.MaxWidth": "73",
			},
		},
		{
			Class:   "TLabel",
			Name:    "Yen",
			Bounds:  Rect{160, 16, 6, 13},
			Caption: "¥",
		},
		{
			Class:  "TBevel",
			Name:   "Bevel1",
			Bounds: Rect{0, 144, 176, 2},
		},
		{
			Class:   "TLabel",
			Name:    "PercentSign",
			Bounds:  Rect{160, 96, 8, 13},
			Caption: "%",
		},
		{
			Class:   "TLabel",
			Name:    "PokeBallsLbl",
			Bounds:  Rect{88, 152, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "GreatBallsLbl",
			Bounds:  Rect{88, 168, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "UltraBallsLbl",
			Bounds:  Rect{88, 184, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "MasterBallsLbl",
			Bounds:  Rect{88, 200, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "PotionsLbl",
			Bounds:  Rect{88, 216, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "SuperPotionsLbl",
			Bounds:  Rect{88, 232, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "Label13",
			Bounds:  Rect{0, 32, 31, 13},
			Caption: "Fights:",
		},
		{
			Class:   "TLabel",
			Name:    "WinsLbl",
			Bounds:  Rect{88, 48, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "Label26",
			Bounds:  Rect{0, 64, 42, 13},
			Caption: "Catches:",
		},
		{
			Class:   "TLabel",
			Name:    "CapturesLbl",
			Bounds:  Rect{88, 64, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "Label14",
			Bounds:  Rect{0, 112, 65, 13},
			Caption: "In POKéDEX:",
		},
		{
			Class:   "TLabel",
			Name:    "OwnedPKMNLbl",
			Bounds:  Rect{88, 112, 6, 13},
			Caption: "0",
		},
		{
			Class:   "TLabel",
			Name:    "Label15",
			Bounds:  Rect{136, 112, 30, 13},
			Caption: "of 386",
		},
		{
			Class:   "TButton",
			Name:    "Button1",
			Bounds:  Rect{0, 152, 75, 17},
			Caption: "POKéBalls:      ",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button1Click",
			},
			Props: map[string]string{
				"TabOrder": "0",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button2",
			Bounds:  Rect{0, 168, 75, 17},
			Caption: "Great Balls:      ",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button2Click",
			},
			Props: map[string]string{
				"TabOrder": "1",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button3",
			Bounds:  Rect{0, 184, 75, 17},
			Caption: "Ultra Balls:       ",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button3Click",
			},
			Props: map[string]string{
				"TabOrder": "2",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button4",
			Bounds:  Rect{0, 200, 75, 17},
			Caption: "Master Balls:    ",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button4Click",
			},
			Props: map[string]string{
				"TabOrder": "3",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button5",
			Bounds:  Rect{0, 216, 75, 17},
			Caption: "Potions:           ",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button5Click",
			},
			Props: map[string]string{
				"TabOrder": "4",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button6",
			Bounds:  Rect{0, 232, 75, 17},
			Caption: "Super Potions: ",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button6Click",
			},
			Props: map[string]string{
				"TabOrder": "5",
			},
		},
		{
			Class:  "TListView",
			Name:   "ListView1",
			Bounds: Rect{0, 248, 177, 225},
			List:   []ListItem{{Caption: "", ImageIndex: -1}},
			Events: map[string]string{
				"OnDblClick": "ListView1Click",
			},
			Props: map[string]string{
				"Align":          "alBottom",
				"ColumnClick":    "False",
				"Columns":        "<>",
				"Hint":           "Doubleclick a POKéMON to see its statistics.",
				"HotTrackStyles": "[htHandPoint, htUnderlineHot]",
				"LargeImages":    "Form4.PKMNImages",
				"ParentShowHint": "False",
				"ShowHint":       "True",
				"SmallImages":    "Form4.SmallPKMNImg",
				"TabOrder":       "6",
				"ViewStyle":      "vsList",
			},
		},
		{
			Class:  "TTimer",
			Name:   "Timer1",
			Bounds: Rect{104, 152, 0, 0},
			Events: map[string]string{
				"OnTimer": "Timer1Timer",
			},
			Props: map[string]string{
				"Interval": "1",
			},
		},
	},
}
