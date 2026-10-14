// Layout data of POKéMON Simulator V4.5 (© Michael Madsen, BirdieSoft),
// translated once from its TForm4 form resource; maintained by hand from now on.
// Not covered by the AGPL: see assets/README.md.

package layout

// Form4 is TForm4 ("POKéMON Simulator V4.5"), the main game window: menu,
// starter list, battle arena, evolve panel and wild-fight picker.
var Form4 = Component{
	Class:   "TForm4",
	Name:    "Form4",
	Bounds:  Rect{189, 105, 0, 0},
	Caption: "POKéMON Simulator V4.5",
	Events: map[string]string{
		"OnClose": "FormClose",
		"OnShow":  "FormShow",
	},
	Props: map[string]string{
		"BorderIcons":  "[biSystemMenu]",
		"BorderStyle":  "bsSingle",
		"ClientHeight": "270",
		"ClientWidth":  "422",
		"Color":        "clBtnFace",
		"Font.Charset": "DEFAULT_CHARSET",
		"Font.Color":   "clWindowText",
		"Font.Height":  "-11",
		"Font.Name":    "MS Sans Serif",
		"Font.Style":   "[]",
		"Menu":         "MainMenu1",
	},
	Children: []Component{
		{
			Class:   "TLabel",
			Name:    "Label1",
			Bounds:  Rect{143, 16, 135, 13},
			Caption: "Select a starting POKéMON:",
			Hidden:  true,
		},
		{
			Class:   "TLabel",
			Name:    "Label10",
			Bounds:  Rect{8, 12, 26, 13},
			Caption: "Fight:",
			Hidden:  true,
		},
		{
			Class:   "TLabel",
			Name:    "Label11",
			Bounds:  Rect{8, 36, 37, 13},
			Caption: "at level:",
			Hidden:  true,
		},
		{
			Class:  "TListView",
			Name:   "ListView1",
			Bounds: Rect{147, 32, 133, 217},
			Hidden: true,
			List: []ListItem{
				{Caption: "Bulbasaur", ImageIndex: 0},
				{Caption: "Charmander", ImageIndex: 3},
				{Caption: "Squirtle", ImageIndex: 6},
			},
			Events: map[string]string{
				"OnClick": "ListView1Click",
			},
			Props: map[string]string{
				"Columns":        "<>",
				"HotTrackStyles": "[htHandPoint, htUnderlineHot]",
				"SmallImages":    "PKMNImages",
				"TabOrder":       "0",
				"ViewStyle":      "vsList",
			},
		},
		{
			Class:  "TMediaPlayer",
			Name:   "MediaPlayer1",
			Bounds: Rect{0, 248, 57, 20},
			Hidden: true,
			Props: map[string]string{
				"EnabledButtons": "[btPlay, btStop]",
				"TabOrder":       "1",
				"TabStop":        "False",
				"VisibleButtons": "[btPlay, btStop]",
			},
		},
		{
			Class:   "TRadioGroup",
			Name:    "EvolveChoice",
			Bounds:  Rect{24, 80, 113, 129},
			Caption: "Evolve to:",
			Hidden:  true,
			Props: map[string]string{
				"TabOrder": "2",
				"TabStop":  "True",
			},
		},
		{
			Class:   "TButton",
			Name:    "EvolveButton",
			Bounds:  Rect{24, 208, 113, 25},
			Caption: "Evolve",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "EvolveButtonClick",
			},
			Props: map[string]string{
				"TabOrder": "3",
			},
		},
		{
			Class:   "TComboBox",
			Name:    "ComboBox1",
			Bounds:  Rect{48, 8, 145, 21},
			Caption: "Select a POKéMON",
			Hidden:  true,
			Props: map[string]string{
				"ItemHeight": "13",
				"TabOrder":   "5",
			},
		},
		{
			Class:  "TSpinEdit",
			Name:   "SpinEdit1",
			Bounds: Rect{48, 32, 41, 22},
			Hidden: true,
			Props: map[string]string{
				"MaxValue": "100",
				"MinValue": "2",
				"TabOrder": "6",
				"Value":    "100",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button2",
			Bounds:  Rect{8, 56, 25, 25},
			Caption: "Go!",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button2Click",
			},
			Props: map[string]string{
				"TabOrder": "7",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button3",
			Bounds:  Rect{32, 56, 41, 25},
			Caption: "Cancel",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button3Click",
			},
			Props: map[string]string{
				"TabOrder": "8",
			},
		},
		{
			Class:   "TButton",
			Name:    "Button4",
			Bounds:  Rect{72, 56, 49, 25},
			Caption: "Random",
			Hidden:  true,
			Events: map[string]string{
				"OnClick": "Button4Click",
			},
			Props: map[string]string{
				"TabOrder": "9",
			},
		},
		{
			Class:   "TGroupBox",
			Name:    "GroupBox1",
			Bounds:  Rect{120, 16, 297, 241},
			Caption: "Battle Arena",
			Hidden:  true,
			Props: map[string]string{
				"TabOrder": "4",
			},
			Children: []Component{
				{
					Class:  "TLabel",
					Name:   "Label2",
					Bounds: Rect{72, 24, 3, 13},
				},
				{
					Class:   "TLabel",
					Name:    "Label3",
					Bounds:  Rect{72, 136, 32, 13},
					Caption: "Label3",
				},
				{
					Class:   "TLabel",
					Name:    "Label8",
					Bounds:  Rect{8, 80, 53, 13},
					Caption: "1000/1000",
					Hidden:  true,
					Props: map[string]string{
						"Alignment": "taRightJustify",
						"AutoSize":  "False",
					},
				},
				{
					Class:   "TLabel",
					Name:    "Label9",
					Bounds:  Rect{8, 176, 53, 13},
					Caption: "1000/1000",
					Props: map[string]string{
						"Alignment": "taRightJustify",
						"AutoSize":  "False",
					},
				},
				{
					Class:  "TGauge",
					Name:   "Gauge1",
					Bounds: Rect{208, 24, 81, 12},
					Hidden: true,
					Props: map[string]string{
						"BackColor":   "clBtnFace",
						"Color":       "clGreen",
						"ForeColor":   "clGreen",
						"ParentColor": "False",
						"Progress":    "0",
					},
				},
				{
					Class:  "TGauge",
					Name:   "Gauge2",
					Bounds: Rect{176, 136, 113, 12},
					Props: map[string]string{
						"BackColor":   "clBtnFace",
						"Color":       "clNavy",
						"ForeColor":   "clGreen",
						"ParentColor": "False",
						"Progress":    "100",
					},
				},
				{
					Class:  "TImage",
					Name:   "Image1",
					Bounds: Rect{32, 190, 16, 16},
					Hidden: true,
					Props: map[string]string{
						"Picture":     "sprites/pokeball.png",
						"Transparent": "True",
					},
				},
				{
					Class:   "TRadioGroup",
					Name:    "RadioGroup1",
					Bounds:  Rect{72, 40, 217, 89},
					Caption: "'s Attacks:",
					Hidden:  true,
					Items:   []string{"Attack #1", "Attack #2", "Attack #3", "Attack #4", "Run!"},
					Props: map[string]string{
						"TabOrder": "0",
						"TabStop":  "True",
					},
				},
				{
					Class:   "TGroupBox",
					Name:    "GroupBox2",
					Bounds:  Rect{72, 152, 217, 81},
					Caption: "Enemy 's Attacks:",
					Props: map[string]string{
						"TabOrder": "2",
					},
					Children: []Component{
						{
							Class:   "TLabel",
							Name:    "Label4",
							Bounds:  Rect{8, 16, 73, 13},
							Caption: "Opp. Attack #1",
						},
						{
							Class:   "TLabel",
							Name:    "Label5",
							Bounds:  Rect{8, 32, 73, 13},
							Caption: "Opp. Attack #2",
						},
						{
							Class:   "TLabel",
							Name:    "Label6",
							Bounds:  Rect{8, 48, 73, 13},
							Caption: "Opp. Attack #3",
						},
						{
							Class:   "TLabel",
							Name:    "Label7",
							Bounds:  Rect{8, 64, 73, 13},
							Caption: "Opp. Attack #4",
						},
					},
				},
				{
					Class:   "TButton",
					Name:    "Button1",
					Bounds:  Rect{248, 8, 43, 15},
					Caption: "Go!",
					Hidden:  true,
					Events: map[string]string{
						"OnClick": "Button1Click",
					},
					Props: map[string]string{
						"TabOrder": "1",
					},
				},
				{
					Class:  "TListView",
					Name:   "ListView2",
					Bounds: Rect{4, 16, 65, 65},
					List:   []ListItem{{Caption: "", ImageIndex: -1}},
					Props: map[string]string{
						"BorderStyle": "bsNone",
						"Color":       "clBtnFace",
						"Columns":     "<>",
						"SmallImages": "PKMNImages",
						"TabOrder":    "3",
						"TabStop":     "False",
						"ViewStyle":   "vsList",
					},
				},
				{
					Class:  "TListView",
					Name:   "ListView3",
					Bounds: Rect{4, 112, 65, 65},
					List:   []ListItem{{Caption: "", ImageIndex: 0}},
					Props: map[string]string{
						"BorderStyle": "bsNone",
						"Color":       "clBtnFace",
						"Columns":     "<>",
						"SmallImages": "PKMNImages",
						"TabOrder":    "4",
						"TabStop":     "False",
						"ViewStyle":   "vsList",
					},
				},
				{
					Class:   "TButton",
					Name:    "EndButton",
					Bounds:  Rect{8, 208, 57, 25},
					Caption: "End Fight",
					Hidden:  true,
					Events: map[string]string{
						"OnClick": "EndButtonClick",
					},
					Props: map[string]string{
						"TabOrder": "5",
					},
				},
			},
		},
		{
			Class:  "TMainMenu",
			Name:   "MainMenu1",
			Bounds: Rect{24, 140, 0, 0},
			Props: map[string]string{
				"Images": "ImageList1",
			},
			Children: []Component{
				{
					Class:   "TMenuItem",
					Name:    "File1",
					Caption: "File",
					Children: []Component{
						{
							Class:   "TMenuItem",
							Name:    "ExportPOKMONinformation1",
							Caption: "Export POKéMON information...",
							Events: map[string]string{
								"OnClick": "ExportPOKMONinformation1Click",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "Quit1",
							Caption: "Quit",
							Events: map[string]string{
								"OnClick": "Quit1Click",
							},
							Props: map[string]string{
								"ImageIndex": "0",
							},
						},
					},
				},
				{
					Class:   "TMenuItem",
					Name:    "Action1",
					Caption: "Actions",
					Children: []Component{
						{
							Class:   "TMenuItem",
							Name:    "FightTrainer1",
							Caption: "Fight a Trainer",
							Children: []Component{
								{
									Class:   "TMenuItem",
									Name:    "Trainers1",
									Caption: "Trainers",
									Children: []Component{
										{
											Class:   "TMenuItem",
											Name:    "Beginner1",
											Caption: "Beginner",
											Children: []Component{
												{
													Class:   "TMenuItem",
													Name:    "Red1",
													Caption: "Red",
													Events: map[string]string{
														"OnClick": "Red1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "Blue1",
													Caption: "Blue",
													Events: map[string]string{
														"OnClick": "Blue1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "Green1",
													Caption: "Green",
													Events: map[string]string{
														"OnClick": "Green1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "Yellow1",
													Caption: "Yellow",
													Events: map[string]string{
														"OnClick": "Yellow1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "Gold1",
													Caption: "Gold",
													Events: map[string]string{
														"OnClick": "Gold1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "Silver1",
													Caption: "Silver",
													Events: map[string]string{
														"OnClick": "Silver1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "Crystal1",
													Caption: "Crystal",
													Events: map[string]string{
														"OnClick": "Crystal1Click",
													},
												},
											},
										},
										{
											Class:   "TMenuItem",
											Name:    "Novice",
											Caption: "Novice",
											Children: []Component{
												{
													Class:   "TMenuItem",
													Name:    "Ruby1",
													Caption: "Ruby",
													Events: map[string]string{
														"OnClick": "Ruby1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "Sapphire1",
													Caption: "Sapphire",
													Events: map[string]string{
														"OnClick": "Sapphire1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "Emerald1",
													Caption: "Emerald",
													Events: map[string]string{
														"OnClick": "Emerald1Click",
													},
												},
											},
										},
										{
											Class:   "TMenuItem",
											Name:    "Intermediate1",
											Caption: "Intermediate",
											Children: []Component{
												{
													Class:   "TMenuItem",
													Name:    "ProfOak1",
													Caption: "Prof.Oak",
													Events: map[string]string{
														"OnClick": "ProfOak1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "ProfElm1",
													Caption: "Prof.Elm",
													Events: map[string]string{
														"OnClick": "ProfElm1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "ProfBirch1",
													Caption: "Prof.Birch",
													Events: map[string]string{
														"OnClick": "ProfBirch1Click",
													},
												},
											},
										},
										{
											Class:   "TMenuItem",
											Name:    "Advanced1",
											Caption: "Advanced",
											Children: []Component{
												{
													Class:   "TMenuItem",
													Name:    "Jesse1",
													Caption: "Jesse",
													Events: map[string]string{
														"OnClick": "Jesse1Click",
													},
												},
												{
													Class:   "TMenuItem",
													Name:    "James1",
													Caption: "James",
													Events: map[string]string{
														"OnClick": "James1Click",
													},
												},
											},
										},
										{
											Class:   "TMenuItem",
											Name:    "Expert1",
											Caption: "Expert",
											Children: []Component{
												{
													Class:   "TMenuItem",
													Name:    "Giovanni1",
													Caption: "Giovanni",
													Events: map[string]string{
														"OnClick": "Giovanni1Click",
													},
												},
											},
										},
									},
								},
								{
									Class:   "TMenuItem",
									Name:    "KantoGymLeaders1",
									Caption: "Kanto Gym Leaders",
									Children: []Component{
										{
											Class:   "TMenuItem",
											Name:    "Brock1",
											Caption: "Brock",
											Events: map[string]string{
												"OnClick": "Brock1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Misty1",
											Caption:  "Misty",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Misty1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "LtSurge1",
											Caption:  "Lt. Surge",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "LtSurge1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Erika1",
											Caption:  "Erika",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Erika1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Sabrina1",
											Caption:  "Sabrina",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Sabrina1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Janine1",
											Caption:  "Janine",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Janine1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Blaine1",
											Caption:  "Blaine",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Blaine1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Gary1",
											Caption:  "Gary",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Gary1Click",
											},
										},
									},
								},
								{
									Class:    "TMenuItem",
									Name:     "EliteFour1",
									Caption:  "G/S/C Elite Four",
									Disabled: true,
									Events: map[string]string{
										"OnClick": "EliteFour1Click",
									},
								},
								{
									Class:    "TMenuItem",
									Name:     "JohtoGymLeaders1",
									Caption:  "Johto Gym Leaders",
									Disabled: true,
									Children: []Component{
										{
											Class:   "TMenuItem",
											Name:    "Falkner1",
											Caption: "Falkner",
											Events: map[string]string{
												"OnClick": "Falkner1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Bugsy1",
											Caption:  "Bugsy",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Bugsy1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Whitney1",
											Caption:  "Whitney",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Whitney1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Morty1",
											Caption:  "Morty",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Morty1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Chuck1",
											Caption:  "Chuck",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Chuck1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Jasmine1",
											Caption:  "Jasmine",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Jasmine1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Pryce1",
											Caption:  "Pryce",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Pryce1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Clair1",
											Caption:  "Clair",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Clair1Click",
											},
										},
									},
								},
								{
									Class:    "TMenuItem",
									Name:     "Ash1",
									Caption:  "Ash",
									Disabled: true,
									Events: map[string]string{
										"OnClick": "Ash1Click",
									},
								},
								{
									Class:    "TMenuItem",
									Name:     "Me1",
									Caption:  "Me",
									Disabled: true,
									Events: map[string]string{
										"OnClick": "Me1Click",
									},
								},
								{
									Class:    "TMenuItem",
									Name:     "HoennGymLeaders1",
									Caption:  "Hoenn Gym Leaders (L.100)",
									Disabled: true,
									Children: []Component{
										{
											Class:   "TMenuItem",
											Name:    "Roxanne1",
											Caption: "Roxanne",
											Events: map[string]string{
												"OnClick": "Roxanne1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Brawly1",
											Caption:  "Brawly",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Brawly1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Wattson1",
											Caption:  "Wattson",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Wattson1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Flannery1",
											Caption:  "Flannery",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Flannery1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Norman1",
											Caption:  "Norman",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Norman1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Winona1",
											Caption:  "Winona",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Winona1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "TateLiza1",
											Caption:  "Tate && Liza",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "TateLiza1Click",
											},
										},
										{
											Class:    "TMenuItem",
											Name:     "Wallace1",
											Caption:  "Wallace",
											Disabled: true,
											Events: map[string]string{
												"OnClick": "Wallace1Click",
											},
										},
									},
								},
								{
									Class:    "TMenuItem",
									Name:     "Elites1",
									Caption:  "Other Elites",
									Disabled: true,
									Children: []Component{
										{
											Class:   "TMenuItem",
											Name:    "RBYEliteFour1",
											Caption: "R/B/Y Elite Four",
											Events: map[string]string{
												"OnClick": "RBYEliteFour1Click",
											},
										},
										{
											Class:   "TMenuItem",
											Name:    "RuSaEliteFour1",
											Caption: "Ru/Sa Elite Four",
											Events: map[string]string{
												"OnClick": "RuSaEliteFour1Click",
											},
										},
									},
								},
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "FightWild1",
							Caption: "Fight a wild POKéMON",
							Children: []Component{
								{
									Class:   "TMenuItem",
									Name:    "FindPKMN1",
									Caption: "Search for a POKéMON",
									Events: map[string]string{
										"OnClick": "FindPKMN1Click",
									},
								},
								{
									Class:    "TMenuItem",
									Name:     "Mewtwo1",
									Caption:  "Mewtwo",
									Disabled: true,
									Events: map[string]string{
										"OnClick": "Mewtwo1Click",
									},
								},
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "Tournaments1",
							Caption: "Tournaments",
							Children: []Component{
								{
									Class:   "TMenuItem",
									Name:    "VeryEasyLevels2101",
									Caption: "Very Easy (Levels 2-10)",
									Events: map[string]string{
										"OnClick": "VeryEasyLevels2101Click",
									},
								},
								{
									Class:   "TMenuItem",
									Name:    "EasyLevels10301",
									Caption: "Easy (Levels 10-30)",
									Events: map[string]string{
										"OnClick": "EasyLevels10301Click",
									},
								},
								{
									Class:   "TMenuItem",
									Name:    "IntermediateLevels30501",
									Caption: "Intermediate (Levels 30-50)",
									Events: map[string]string{
										"OnClick": "IntermediateLevels30501Click",
									},
								},
								{
									Class:   "TMenuItem",
									Name:    "HardLevels50701",
									Caption: "Hard (Levels 50-70)",
									Events: map[string]string{
										"OnClick": "HardLevels50701Click",
									},
								},
								{
									Class:   "TMenuItem",
									Name:    "VeryHardLevels70991",
									Caption: "Very Hard (Levels 70-99)",
									Events: map[string]string{
										"OnClick": "VeryHardLevels70991Click",
									},
								},
								{
									Class:   "TMenuItem",
									Name:    "PrimeCupLevel1001",
									Caption: "Prime Cup (Level 100)",
									Events: map[string]string{
										"OnClick": "PrimeCupLevel1001Click",
									},
								},
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "Buy1",
							Caption: "Buy Items",
							Events: map[string]string{
								"OnClick": "Buy1Click",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "Evolve1",
							Caption: "Evolve a POKéMON",
							Events: map[string]string{
								"OnClick": "Evolve1Click",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "ReleasePKMN1",
							Caption: "Release a POKéMON",
							Events: map[string]string{
								"OnClick": "ReleasePKMN1Click",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "ViewDEX1",
							Caption: "View POKéDEX",
							Events: map[string]string{
								"OnClick": "ViewDEX1Click",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "TypeComparisons1",
							Caption: "Type Comparisons",
							Events: map[string]string{
								"OnClick": "TypeComparisons1Click",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "GameCorner1",
							Caption: "Game Corner",
							Events: map[string]string{
								"OnClick": "GameCorner1Click",
							},
						},
					},
				},
				{
					Class:   "TMenuItem",
					Name:    "SelectMusic1",
					Caption: "Music",
					Children: []Component{
						{
							Class:   "TMenuItem",
							Name:    "Openfile1",
							Caption: "Open file...",
							Events: map[string]string{
								"OnClick": "Openfile1Click",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "RepeatMusic1",
							Caption: "Repeat Music",
							Events: map[string]string{
								"OnClick": "RepeatMusic1Click",
							},
							Props: map[string]string{
								"Checked": "True",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "StopMusic1",
							Caption: "Stop Music",
							Events: map[string]string{
								"OnClick": "StopMusic1Click",
							},
						},
					},
				},
				{
					Class:   "TMenuItem",
					Name:    "Help1",
					Caption: "Help",
					Children: []Component{
						{
							Class:   "TMenuItem",
							Name:    "Contents1",
							Caption: "Contents",
							Events: map[string]string{
								"OnClick": "Contents1Click",
							},
							Props: map[string]string{
								"ImageIndex": "1",
								"ShortCut":   "112",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "About1",
							Caption: "About...",
							Events: map[string]string{
								"OnClick": "About1Click",
							},
						},
					},
				},
				{
					Class:   "TMenuItem",
					Name:    "Debug1",
					Caption: "Debug",
					Hidden:  true,
					Children: []Component{
						{
							Class:   "TMenuItem",
							Name:    "MakeVisible1",
							Caption: "Make Visible",
							Events: map[string]string{
								"OnClick": "MakeVisible1Click",
							},
							Props: map[string]string{
								"ShortCut": "113",
							},
						},
						{
							Class:   "TMenuItem",
							Name:    "AlwaysshowattacksinPOKDEX1",
							Caption: "Always show attacks in POKéDEX",
							Events: map[string]string{
								"OnClick": "AlwaysshowattacksinPOKDEX1Click",
							},
						},
					},
				},
			},
		},
		{
			Class:  "TSaveDialog",
			Name:   "SaveDialog1",
			Bounds: Rect{80, 140, 0, 0},
			Props: map[string]string{
				"DefaultExt": "*.pgs",
				"Filter":     "Saved Games (*.PGS)|*.pgs",
				"Options":    "[ofOverwritePrompt, ofHideReadOnly, ofEnableSizing, ofDontAddToRecent]",
			},
		},
		{
			Class:  "TImageList",
			Name:   "ImageList1",
			Bounds: Rect{24, 184, 0, 0},
			Props: map[string]string{
				"Bitmap": "sprites/small16.png",
			},
		},
		{
			Class:  "TImageList",
			Name:   "PKMNImages",
			Bounds: Rect{24, 96, 64, 64},
			Props: map[string]string{
				"Bitmap": "sprites/front64.png",
			},
		},
		{
			Class:  "TOpenDialog",
			Name:   "OpenDialog1",
			Bounds: Rect{80, 184, 0, 0},
			Props: map[string]string{
				"Filter":  "All supported formats (*.mp3;*.mpeg;*.mpg;*.mpe;*.m1v;*.mp2;*.mpv2;*.mpa;*.wma;*.wmv;*.wav;*.mid;*.midi;*.rmi)|*.mp3;*.mpeg;*.mpg;*.mpe;*.m1v;*.mp2;*.mpv2;*.mpa;*.wma;*.wmv;*.wav;*.mid;*.midi;*.rmi|MPEG Audio/Video Files (*.mp3;*.mpeg;*.mpg;*.mpe;*.m1v;*.mp2;*.mpv2;*.mpa)|*.mp3;*.mpeg;*.mpg;*.mpe;*.m1v;*.mp2;*.mpv2;*.mpa|Windows Media (*.wma;*.wmv)|*.wma*.wmv|Wave Files (*.wav)|*.wav|MIDI Files (*.mid;*.midi;*.rmi)|*.mid;*.midi;*.rmi|All files (*.*)|*.*",
				"Options": "[ofHideReadOnly, ofPathMustExist, ofFileMustExist, ofEnableSizing, ofDontAddToRecent]",
			},
		},
		{
			Class:  "TTimer",
			Name:   "Timer1",
			Bounds: Rect{216, 0, 0, 0},
			Events: map[string]string{
				"OnTimer": "Timer1Timer",
			},
			Props: map[string]string{
				"Interval": "1",
			},
		},
		{
			Class:  "TSaveDialog",
			Name:   "HTMLExport",
			Bounds: Rect{80, 96, 0, 0},
			Props: map[string]string{
				"DefaultExt": ".htm",
				"Filter":     "HTML Files (*.htm)|*.htm",
				"Options":    "[ofOverwritePrompt, ofHideReadOnly, ofEnableSizing, ofDontAddToRecent]",
			},
		},
		{
			Class:  "TImageList",
			Name:   "SmallPKMNImg",
			Bounds: Rect{24, 224, 32, 32},
			Props: map[string]string{
				"Bitmap": "sprites/icons32.png",
			},
		},
	},
}
