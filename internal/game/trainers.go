// SPDX-License-Identifier: AGPL-3.0-only

package game

// Trainers, gym leaders and the Elite Four (notes/trainers.md). Every
// trainer has one POKéMON.

// trainer is what a "Fight a Trainer" menu item starts.
type trainer struct {
	name    string
	species int
	level   int
	battle  uint8 // BattleType
	gym     int   // a gym leader's badge: BattleType 3 when GymProgress is this; -1 for others
}

// trainers maps the menu items to their trainers.
var trainers = map[string]trainer{
	"Red1":       {"Red", 4, 5, 2, -1},
	"Blue1":      {"Blue", 7, 5, 2, -1},
	"Green1":     {"Green", 1, 5, 2, -1},
	"Yellow1":    {"Yellow", 25, 5, 2, -1},
	"Gold1":      {"Gold", 152, 10, 2, -1},
	"Silver1":    {"Silver", 155, 10, 2, -1},
	"Crystal1":   {"Crystal", 158, 10, 2, -1},
	"Ruby1":      {"Ruby", 256, 25, 2, -1},
	"Sapphire1":  {"Sapphire", 259, 25, 2, -1},
	"Emerald1":   {"Emerald", 253, 25, 2, -1},
	"ProfOak1":   {"Prof. Oak", 99, 50, 2, -1},
	"ProfElm1":   {"Prof. Elm", 172, 50, 2, -1},
	"ProfBirch1": {"Prof. Birch", 350, 50, 2, -1},
	"Jesse1":     {"Jesse", 24, 75, 2, -1},
	"James1":     {"James", 110, 75, 2, -1},
	"Giovanni1":  {"Giovanni", 150, 100, 2, -1},

	"Brock1":   {"Brock", 74, 12, 2, 0},
	"Misty1":   {"Misty", 120, 18, 2, 1},
	"LtSurge1": {"Lt. Surge", 26, 22, 2, 2},
	"Erika1":   {"Erika", 45, 25, 2, 3},
	"Sabrina1": {"Sabrina", 49, 28, 2, 4},
	"Janine1":  {"Janine", 89, 32, 2, 5},
	"Blaine1":  {"Blaine", 126, 35, 2, 6},
	"Gary1":    {"Gary", 34, 40, 2, 7},

	"Falkner1": {"Falkner", 17, 50, 2, 8},
	"Bugsy1":   {"Bugsy", 123, 53, 2, 9},
	"Whitney1": {"Whitney", 241, 57, 2, 10},
	"Morty1":   {"Morty", 94, 61, 2, 11},
	"Chuck1":   {"Chuck", 62, 64, 2, 12},
	"Jasmine1": {"Jasmine", 208, 70, 2, 13},
	"Pryce1":   {"Pryce", 87, 75, 2, 14},
	"Clair1":   {"Clair", 230, 82, 2, 15},

	"Roxanne1":  {"Roxanne", 299, 100, 2, 16},
	"Brawly1":   {"Brawly", 296, 100, 2, 17},
	"Wattson1":  {"Wattson", 82, 100, 2, 18},
	"Flannery1": {"Flannery", 324, 100, 2, 19},
	"Norman1":   {"Norman", 289, 100, 2, 20},
	"Winona1":   {"Winona", 334, 100, 2, 21},
	"TateLiza1": {"Tate & Liza", 337, 100, 2, 22},
	"Wallace1":  {"Wallace", 350, 100, 2, 23},

	"EliteFour1":     {"Will", 178, 40, 4, -1},
	"Ash1":           {"Ash", 25, 95, 9, -1},
	"Me1":            {"Link", 151, 100, 10, -1}, // the author
	"RBYEliteFour1":  {"Lorelei", 131, 100, 71, -1},
	"RuSaEliteFour1": {"Sidney", 262, 100, 81, -1},
}

// fightTrainer is a trainer's menu item (TForm4.<Name>Click).
func (g *Game) fightTrainer(item string, t trainer) {
	g.StartTrainerBattle(t.species, t.level, t.name)
	g.BattleType = t.battle
	switch {
	case t.gym >= 0 && int(g.GymProgress) == t.gym:
		g.BattleType = 3 // the badge you're due; a rematch is an ordinary win
	case item == "Ash1" && g.BeatAsh, item == "Me1" && g.BeatLink:
		g.BattleType = 2 // beaten before: no reward
	}
}

// gymUnlocks lists the item each badge count enables (EnableGymLeaders).
var gymUnlocks = [...]string{
	1: "Misty1", "LtSurge1", "Erika1", "Sabrina1", "Janine1", "Blaine1", "Gary1",
	"EliteFour1",
	"Bugsy1", "Whitney1", "Morty1", "Chuck1", "Jasmine1", "Pryce1", "Clair1",
	"Ash1",
	"Brawly1", "Wattson1", "Flannery1", "Norman1", "Winona1", "TateLiza1", "Wallace1",
	"Elites1",
}

// unlockAll enables what the progress has earned, after loading a game.
func (g *Game) unlockAll() {
	for n := 1; n < len(gymUnlocks); n++ {
		if int(g.GymProgress) >= n {
			g.enable(true, gymUnlocks[n])
		}
	}
	if g.BeatLance {
		g.enable(true, "JohtoGymLeaders1")
	}
	if g.BeatAsh {
		g.enable(true, "Me1")
	}
	if g.BeatLink {
		g.enable(true, "Mewtwo1", "HoennGymLeaders1")
	}
}

// rewards are the badges and the first wins over Lance, Ash and Link,
// before EndBattle resets the battle.
func (g *Game) rewards() {
	if g.BattleType == 3 {
		g.GymProgress++
		g.message(Information, "You earned the "+g.badge(int(g.GymProgress))+"!")
		if int(g.GymProgress) < len(gymUnlocks) {
			g.enable(true, gymUnlocks[g.GymProgress])
		}
		switch g.GymProgress {
		case 8:
			g.message(Information, "You beat Gary, the final Kanto leader! You can now battle the Elite Four! Also, you get a 2500¥ bonus!")
			g.Money += 2500
		case 16:
			g.message(Information, "You beat Clair, the final Johto leader! You can now battle Ash! Also, you get a 3000¥ bonus!")
			g.Money += 3000
		case 24:
			g.message(Information, "You beat Wallace, the final Hoenn leader! You can now battle the R/B/Y and Ru/Sa Elites! Also, you get a 5000¥ bonus!")
			g.Money += 5000
		}
	}
	if g.BattleType == 8 && !g.BeatLance {
		g.BeatLance = true
		g.message(Information, "You beat Lance, the Elite Four champion! You can now battle the Johto Gym Leaders! Also, you get a Master Ball!")
		g.MasterBalls++
		g.message(Information, "This Ball will catch any POKéMON at any HP. Use it wisely.")
		g.enable(true, "JohtoGymLeaders1")
	}
	if g.BattleType == 9 {
		g.message(Information, "You beat Ash! You can now battle me! Also, you get a Celebi at L.5!")
		g.AddPKMN(251, "Celebi", 5)
		g.enable(true, "Me1")
		g.BeatAsh = true
	}
	if g.BattleType == 10 {
		g.message(Information, "You beat me! You can now battle and/or catch Mewtwo! You can also fight the Hoenn Gym Leaders! And as if that wasn't enough, you also get a Mew at L.10!")
		g.AddPKMN(151, "Mew", 10)
		g.enable(true, "Mewtwo1", "HoennGymLeaders1")
		g.BeatLink = true
	}
}

// badge is BadgeNames[n].
func (g *Game) badge(n int) string {
	if n < 1 || n > len(g.Data.Badges) {
		return ""
	}
	return g.Data.Badges[n-1]
}

// eliteStep is a won Elite Four battle that leads to the next one.
type eliteStep struct {
	after   uint8 // the BattleType won
	message string
	trainer string
	species int
	level   int
	next    uint8
}

// eliteChains are checked in this order, after EndBattle's reset, with
// the R/B/Y champion's reward between 81 and 74: each step sets a
// BattleType that only an earlier check looks for, so one win starts one
// battle and the champion's reward waits for the champion.
var eliteChains = []eliteStep{
	{84, "You have proven your worth against the Ru/Sa Elite Four. Now, it is time for you to fight the Ru/Sa Elite Four Champion: Steven!", "Steven", 376, 100, 85},
	{83, "Congratulations! You beat Glacia! Your next opponent is Drake.", "Drake", 330, 100, 84},
	{82, "Congratulations! You beat Phoebe! Your next opponent is Glacia.", "Glacia", 365, 100, 83},
	{81, "Congratulations! You beat Sidney! Your next opponent is Phoebe.", "Phoebe", 354, 100, 82},
	{74, "You have proven your worth against the R/B/Y Elite Four. Now, it is time for you to fight the R/B/Y Elite Four Champion: Gary!", "Gary", 65, 100, 75},
	{73, "Congratulations! You beat Agatha! Your next opponent is Lance.", "Lance", 149, 100, 74},
	{72, "Congratulations! You beat Bruno! Your next opponent is Agatha.", "Agatha", 94, 100, 73},
	{71, "Congratulations! You beat Lorelei! Your next opponent is Bruno.", "Bruno", 95, 100, 72},
	{7, "You have proven your worth against the G/S/C Elite Four. Now, it is time for you to fight the G/S/C Elite Four Champion: Lance!", "Lance", 149, 50, 8},
	{6, "Congratulations! You beat Bruno! Your next opponent is Karen.", "Karen", 94, 45, 7},
	{5, "Congratulations! You beat Koga! Your next opponent is Bruno.", "Bruno", 95, 43, 6},
	{4, "Congratulations! You beat Will! Your next opponent is Koga.", "Koga", 205, 43, 5},
}

// eliteFour runs, after EndBattle's reset, the R/B/Y and Ru/Sa champion
// rewards and the Elite Four chains, in the original's order.
func (g *Game) eliteFour() {
	if g.BattleType == 85 {
		g.message(Information, "You beat Steven, the Ru/Sa Elite Four champion! You get a bonus of 7500¥!")
		g.Money += 7500
		if !g.GotDeoxys {
			g.GotDeoxys = true
			g.message(Information, "Also, you get a Deoxys at L.5!")
			g.AddPKMN(386, "Deoxys", 5)
		}
	}
	for _, s := range eliteChains[:4] {
		g.eliteStep(s)
	}
	if g.BattleType == 75 {
		g.message(Information, "You beat Gary, the R/B/Y Elite Four champion! You get a bonus of 7500¥!")
		g.Money += 7500
		if !g.GotJirachi {
			g.GotJirachi = true
			g.message(Information, "Also, you get a Jirachi at L.5!")
			g.AddPKMN(385, "Jirachi", 5)
		}
	}
	for _, s := range eliteChains[4:] {
		g.eliteStep(s)
	}
}

// eliteStep starts the next battle of a chain when s's battle was won.
func (g *Game) eliteStep(s eliteStep) {
	if g.BattleType != s.after {
		return
	}
	g.message(Information, s.message)
	g.StartTrainerBattle(s.species, s.level, s.trainer)
	g.BattleType = s.next
	if s.after == 7 && g.menu["JohtoGymLeaders1"].Enabled {
		g.BattleType = 2 // Lance beaten before: no second Master Ball
	}
}
