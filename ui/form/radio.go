package form

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Radio is a single-choice option. Group multiple Radio elements
// in a Column; the user's model owns which option is selected.
type Radio struct {
	ui.BaseElement
	Label    string
	Selected bool
	OnSelect func()
	Disabled bool
}

// NewRadio creates a radio button element.
func NewRadio(label string, selected bool, onSelect func()) ui.Element {
	return Radio{Label: label, Selected: selected, OnSelect: onSelect}
}

// RadioDisabled creates a disabled radio button.
func RadioDisabled(label string, selected bool) ui.Element {
	return Radio{Label: label, Selected: selected, Disabled: true}
}

// LayoutSelf implements ui.Layouter.
func (n Radio) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	metrics := canvas.MeasureText(n.Label, style)
	labelW := int(math.Ceil(float64(metrics.Width)))
	labelH := int(math.Ceil(float64(metrics.Ascent)))
	totalH := max(checkboxSize, labelH)
	totalW := checkboxSize + checkboxGap + labelW

	// Register hit target and get hover opacity atomically.
	radioRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(totalH))
	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterHit(radioRect, nil)
	} else {
		hoverOpacity = ix.RegisterHit(radioRect, n.OnSelect)
	}

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	boxY := area.Y + (totalH-checkboxSize)/2
	circleRect := draw.R(float32(area.X), float32(boxY), float32(checkboxSize), float32(checkboxSize))

	// Outer circle
	outerColor := tokens.Colors.Stroke.Border
	if n.Disabled {
		outerColor = ui.DisabledColor(outerColor, tokens.Colors.Surface.Base)
	}
	canvas.FillEllipse(circleRect, draw.SolidPaint(outerColor))

	// Inner fill — two-stage hover→pressed visual (RFC-008 §9.3).
	fillColor := tokens.Colors.Surface.Elevated
	if hoverOpacity > 0 {
		fillColor = ui.LerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
		if hoverOpacity >= 0.9 {
			pressedT := (hoverOpacity - 0.9) / 0.1
			fillColor = ui.LerpColor(fillColor, tokens.Colors.Surface.Pressed, pressedT)
		}
	}
	canvas.FillEllipse(
		draw.R(float32(area.X+checkboxBorder), float32(boxY+checkboxBorder),
			float32(checkboxSize-checkboxBorder*2), float32(checkboxSize-checkboxBorder*2)),
		draw.SolidPaint(fillColor))

	// Focus glow on the radio circle (RFC-008 §9.4).
	if focused {
		ui.DrawFocusRing(canvas, circleRect, float32(checkboxSize)/2, tokens)
	}

	// Selected dot
	if n.Selected {
		dotSize := 8
		dotOffset := (checkboxSize - dotSize) / 2
		dotColor := tokens.Colors.Accent.Primary
		if n.Disabled {
			dotColor = ui.DisabledColor(dotColor, tokens.Colors.Surface.Base)
		}
		canvas.FillEllipse(
			draw.R(float32(area.X+dotOffset), float32(boxY+dotOffset), float32(dotSize), float32(dotSize)),
			draw.SolidPaint(dotColor))
	}

	// Label
	labelX := area.X + checkboxSize + checkboxGap
	labelY := area.Y + (totalH-labelH)/2
	labelColor := tokens.Colors.Text.Primary
	if n.Disabled {
		labelColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(n.Label, draw.Pt(float32(labelX), float32(labelY)), style, labelColor)

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Radio) TreeEqual(other ui.Element) bool {
	nb, ok := other.(Radio)
	return ok && n.Label == nb.Label && n.Selected == nb.Selected
}

// ResolveChildren implements ui.ChildResolver. Radio is a leaf.
func (n Radio) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Radio) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleCheckbox,
		Label:  n.Label,
		States: a11y.AccessStates{Checked: n.Selected},
	}, parentIdx, a11y.Rect{})
}
