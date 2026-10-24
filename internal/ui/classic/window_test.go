// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jsperandio/pokemon-simulator/internal/ui/classic/layout"
)

// anchorWindow shows a 200×100 window with a button for each kind of
// anchoring, and a group box anchored to every edge with a button anchored
// to none inside.
func anchorWindow(ta *testApp) *window {
	return ta.showWindow(&layout.Component{
		Class: "TForm",
		Name:  "AnchorForm",
		Props: map[string]string{
			"ClientHeight": "100",
			"ClientWidth":  "200",
		},
		Children: []layout.Component{
			{
				Class:  "TButton",
				Name:   "Stretched",
				Bounds: layout.Rect{X: 10, Y: 10, W: 50, H: 20},
				Props: map[string]string{
					"Anchors": "[akLeft, akTop, akRight, akBottom]",
				},
			},
			{
				Class:  "TButton",
				Name:   "Right",
				Bounds: layout.Rect{X: 140, Y: 10, W: 50, H: 20},
				Props: map[string]string{
					"Anchors": "[akTop, akRight]",
				},
			},
			{
				Class:  "TButton",
				Name:   "Centred",
				Bounds: layout.Rect{X: 75, Y: 40, W: 50, H: 20},
				Props: map[string]string{
					"Anchors": "[]",
				},
			},
			{
				Class:  "TButton",
				Name:   "Fixed",
				Bounds: layout.Rect{X: 10, Y: 70, W: 50, H: 20},
			},
			{
				Class:  "TGroupBox",
				Name:   "Box",
				Bounds: layout.Rect{X: 100, Y: 50, W: 90, H: 40},
				Props: map[string]string{
					"Anchors": "[akLeft, akTop, akRight, akBottom]",
				},
				Children: []layout.Component{
					{
						Class:  "TButton",
						Name:   "Inner",
						Bounds: layout.Rect{X: 20, Y: 10, W: 50, H: 20},
						Props: map[string]string{
							"Anchors": "[]",
						},
					},
				},
			},
		},
	})
}

func Test_window_resize(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		control string
		want    image.Rectangle
	}{
		{
			name:    "when the window grows, should stretch a control anchored to both edges",
			control: "Stretched",
			want:    image.Rect(10, 10, 160, 90),
		},
		{
			name:    "when the window grows, should keep a control anchored right at its distance from the right edge",
			control: "Right",
			want:    image.Rect(240, 10, 290, 30),
		},
		{
			name:    "when the window grows, should move a control anchored to neither edge by half the change",
			control: "Centred",
			want:    image.Rect(125, 70, 175, 90),
		},
		{
			name:    "when the window grows, should leave a control with the default anchors where it was",
			control: "Fixed",
			want:    image.Rect(10, 70, 60, 90),
		},
		{
			name:    "when the window grows, should anchor a group box's children in the group box",
			control: "Inner",
			want:    image.Rect(70, 40, 120, 60),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wdw := anchorWindow(newTestApp(t))
			wdw.resize(image.Pt(300, 160))

			assert.Equal(t, tc.want, wdw.find(tc.control).base().rect)
		})
	}

	t.Run("when the window grows by an odd size and shrinks back, should put every control back where it was", func(t *testing.T) {
		t.Parallel()

		wdw := anchorWindow(newTestApp(t))
		names := []string{"Stretched", "Right", "Centred", "Fixed", "Box", "Inner"}
		before := map[string]image.Rectangle{}

		for _, nam := range names {
			before[nam] = wdw.find(nam).base().rect
		}

		wdw.resize(image.Pt(301, 163))
		wdw.resize(image.Pt(257, 121))
		wdw.resize(image.Pt(200, 100))

		for _, nam := range names {
			assert.Equal(t, before[nam], wdw.find(nam).base().rect, nam)
		}
	})

	t.Run("when a control's anchors change after the window grew, should anchor it from where it is then", func(t *testing.T) {
		t.Parallel()

		wdw := anchorWindow(newTestApp(t))
		wdw.resize(image.Pt(300, 160))

		ctl := wdw.find("Fixed").base()
		ctl.setAnchors(0)
		wdw.resize(image.Pt(200, 100))

		assert.Equal(t, image.Rect(-40, 40, 10, 60), ctl.rect)
	})

	t.Run("when a control is moved after the window grew, should keep the new place as it resizes", func(t *testing.T) {
		t.Parallel()

		wdw := anchorWindow(newTestApp(t))
		wdw.resize(image.Pt(300, 160))

		ctl := wdw.find("Centred").base()
		ctl.setBounds(image.Rect(0, 0, 50, 20))
		wdw.resize(image.Pt(200, 100))

		assert.Equal(t, image.Rect(-50, -30, 0, -10), ctl.rect)
	})
}
