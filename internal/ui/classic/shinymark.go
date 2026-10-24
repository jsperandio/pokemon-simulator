// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"image"
	"image/color"
	"strings"
)

// shinyMark is the picture the Statistics window and the starter choice
// draw over a shiny POKéMON's icon, which the original doesn't have; the
// player chooses it in Preferences.
type shinyMark int

// The marks, in Preferences' order.
const (
	goldSparkles shinyMark = iota // the default: Pokémon GO's mark, in gold
	goldSparklesOutlined
	whiteSparklesOutlined
	thinTwinkles
	outlinedTwinkles
	oneTwinkle
	noMark
)

// The marks' colours.
var (
	markGold      = rgb(0xFAC300)
	markBrown     = rgb(0x603800)
	markGrey      = rgb(0x464646)
	markWhite     = rgb(0xFFFFFF)
	markDarkGold  = rgb(0xEBAA00)
	markOrange    = rgb(0xE69600)
	markLightGold = rgb(0xFFD628)
)

// sparklesRows is Pokémon GO's mark: a big four-pointed sparkle low on the
// left and two smaller ones up and to the right; '#' is the fill.
const sparklesRows string = `
          #
         ###
        #####
         ###
    #     #
    #
   ###
  #####    #
######### ###
  #####    #
   ###
    #
    #
`

// outlinedSparklesRows is sparklesRows with an outline, 'k'.
const outlinedSparklesRows string = `
          kkk
         kk#kk
        kk###kk
        k#####k
    kkk kk###kk
    k#k  kk#kk
   kk#kk  kkk
  kk###kk  kkk
kkk#####kkkk#kk
k#########k###k
kkk#####kkkk#kk
  kk###kk  kkk
   kk#kk
    k#k
    kkk
`

// markInfo is a mark: the name Preferences lists, its value in the
// settings, and its picture, from its corner's place on the icon.
type markInfo struct {
	name   string
	key    string
	corner image.Point
	rows   string
	colors map[byte]color.RGBA
}

// shinyMarks describes each mark, by shinyMark.
var shinyMarks = []markInfo{
	goldSparkles: {
		name:   "Gold sparkles",
		key:    "GoldSparkles",
		corner: image.Pt(2, 2),
		rows:   sparklesRows,
		colors: map[byte]color.RGBA{
			'#': markGold,
		},
	},
	goldSparklesOutlined: {
		name:   "Gold sparkles, outlined",
		key:    "GoldSparklesOutlined",
		corner: image.Pt(1, 1),
		rows:   outlinedSparklesRows,
		colors: map[byte]color.RGBA{
			'#': markGold,
			'k': markBrown,
		},
	},
	whiteSparklesOutlined: {
		name:   "White sparkles, outlined",
		key:    "WhiteSparklesOutlined",
		corner: image.Pt(1, 1),
		rows:   outlinedSparklesRows,
		colors: map[byte]color.RGBA{
			'#': markWhite,
			'k': markGrey,
		},
	},
	thinTwinkles: {
		name:   "Three thin twinkles",
		key:    "ThinTwinkles",
		corner: image.Pt(14, 15),
		rows: `
              y
              y
            yyWyy
              y
              y

         y
         y
         y
        yyy
     yyyyWyyyy
        yyy
  y      y
  y      y
yyWyy    y
  y
  y
`,
		colors: map[byte]color.RGBA{
			'y': markDarkGold,
			'W': markWhite,
		},
	},
	outlinedTwinkles: {
		name:   "Three outlined twinkles",
		key:    "OutlinedTwinkles",
		corner: image.Pt(13, 14),
		rows: `
              kkk
              kyk
            kkkykkk
            kyyWyyk
            kkkykkk
              kyk
         kkk  kkk
         kok
         kyk
        kkykk
     kkkkyWykkkk
     koyyWWWyyok
  kkkkkkkyWykkkk
  kyk   kkykk
kkkykkk  kyk
kyyWyyk  kok
kkkykkk  kkk
  kyk
  kkk
`,
		colors: map[byte]color.RGBA{
			'k': markBrown,
			'o': markOrange,
			'y': markLightGold,
			'W': markWhite,
		},
	},
	oneTwinkle: {
		name:   "One twinkle",
		key:    "OneTwinkle",
		corner: image.Pt(21, 21),
		rows: `
    kkk
    kok
    kyk
   kkykk
kkkkyWykkkk
koyyWWWyyok
kkkkyWykkkk
   kkykk
    kyk
    kok
    kkk
`,
		colors: map[byte]color.RGBA{
			'k': markBrown,
			'o': markOrange,
			'y': markLightGold,
			'W': markWhite,
		},
	},
	noMark: {
		name: "None",
		key:  "None",
	},
}

// markNames are the marks' names, in Preferences' order.
func markNames() []string {
	names := make([]string, len(shinyMarks))
	for i, mi := range shinyMarks {
		names[i] = mi.name
	}

	return names
}

// markByKey is the mark a settings value names, whatever its case, and
// whether it names one.
func markByKey(key string) (shinyMark, bool) {
	for i, mi := range shinyMarks {
		if strings.EqualFold(mi.key, key) {
			return shinyMark(i), true
		}
	}

	return goldSparkles, false
}

// draw paints the mark over the icon whose corner is at (x, y).
func (sm shinyMark) draw(cv canvas, x, y int) {
	if sm < 0 || int(sm) >= len(shinyMarks) {
		return
	}

	mi := shinyMarks[sm]
	if mi.rows == "" {
		return
	}

	cv.pattern(x+mi.corner.X, y+mi.corner.Y, mi.rows, mi.colors)
}
