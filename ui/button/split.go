package button

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// SplitItem describes a dropdown menu entry for a split button.
type SplitItem struct {
	Label   string
	OnClick func()
}

// Split is a button with a main action and a dropdown menu trigger.
type Split struct {
	ui.BaseElement
	Label     string
	OnClick   func()
	MenuItems []SplitItem
	OnMenu    func() // fires when dropdown arrow is clicked
}

// NewSplit creates a split button.
func NewSplit(label string, onClick func(), onMenu func(), items []SplitItem) ui.Element {
	return Split{Label: label, OnClick: onClick, MenuItems: items, OnMenu: onMenu}
}

// SplitButton is an alias for NewSplit.
func SplitButton(label string, onClick func(), onMenu func(), items []SplitItem) ui.Element {
	return NewSplit(label, onClick, onMenu, items)
}

// LayoutSelf implements ui.Layouter.
func (n Split) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas
	ix := ctx.IX

	// Measure label.
	style := tokens.Typography.Label
	metrics := canvas.MeasureText(n.Label, style)
	labelW := int(math.Ceil(float64(metrics.Width)))
	labelH := int(math.Ceil(float64(metrics.Ascent)))

	mainW := labelW + ui.ButtonPadX*2
	arrowW := ui.SplitArrowWidth
	h := labelH + ui.ButtonPadY*2

	// Enforce MinTouchTarget for touch/HMI profiles (RFC-004 §2.5).
	if ctx.Profile != nil && ctx.Profile.MinTouchTarget > 0 {
		minT := int(ctx.Profile.MinTouchTarget)
		if mainW < minT {
			mainW = minT
		}
		if h < minT {
			h = minT
		}
		if arrowW < minT {
			arrowW = minT
		}
	}
	totalW := mainW + arrowW

	radius := tokens.Radii.Button

	// Main button hit target.
	mainRect := draw.R(float32(area.X), float32(area.Y), float32(mainW), float32(h))
	mainHover := ix.RegisterHit(mainRect, n.OnClick)

	// Arrow button hit target.
	arrowRect := draw.R(float32(area.X+mainW), float32(area.Y), float32(arrowW), float32(h))
	arrowHover := ix.RegisterHit(arrowRect, n.OnMenu)

	// Draw main button (left rounded corners).
	mainFill := tokens.Colors.Accent.Primary
	if mainHover > 0 {
		mainFill = ui.LerpColor(mainFill, ui.HoverHighlight(mainFill), mainHover)
	}
	// Full rounded rect, then overlay the right half to square off right corners.
	canvas.FillRoundRect(draw.R(float32(area.X), float32(area.Y), float32(mainW+1), float32(h)),
		radius, draw.SolidPaint(mainFill))
	// Square off right side.
	canvas.FillRect(draw.R(float32(area.X+mainW-int(radius)), float32(area.Y), float32(int(radius)+1), float32(h)),
		draw.SolidPaint(mainFill))

	// Draw arrow button (right rounded corners).
	arrowFill := tokens.Colors.Accent.Primary
	if arrowHover > 0 {
		arrowFill = ui.LerpColor(arrowFill, ui.HoverHighlight(arrowFill), arrowHover)
	}
	canvas.FillRoundRect(draw.R(float32(area.X+mainW), float32(area.Y), float32(arrowW), float32(h)),
		radius, draw.SolidPaint(arrowFill))
	// Square off left side.
	canvas.FillRect(draw.R(float32(area.X+mainW), float32(area.Y), float32(int(radius)), float32(h)),
		draw.SolidPaint(arrowFill))

	// Divider line between main and arrow.
	divX := float32(area.X + mainW)
	canvas.FillRect(draw.R(divX, float32(area.Y+4), 1, float32(h-8)),
		draw.SolidPaint(draw.Color{R: 1, G: 1, B: 1, A: 0.3}))

	// Label text centered in main area.
	canvas.DrawText(n.Label,
		draw.Pt(float32(area.X+(mainW-labelW)/2), float32(area.Y+(h-labelH)/2)),
		style, tokens.Colors.Text.OnAccent)

	// Caret icon centered in arrow area.
	iconStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       tokens.Typography.Label.Size * 1.5,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}
	caretDown := "\uE136" // icons.CaretDown
	caretMetrics := canvas.MeasureText(caretDown, iconStyle)
	caretX := float32(area.X+mainW) + (float32(arrowW)-caretMetrics.Width)/2
	caretY := float32(area.Y) + (float32(h)-caretMetrics.Ascent)/2
	canvas.DrawText(caretDown, draw.Pt(caretX, caretY), iconStyle, tokens.Colors.Text.OnAccent)

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: h, Baseline: ui.ButtonPadY + labelH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Split) TreeEqual(other ui.Element) bool {
	_, ok := other.(Split)
	return ok && false
}

// ResolveChildren implements ui.ChildResolver. Split buttons are leaves.
func (n Split) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}
