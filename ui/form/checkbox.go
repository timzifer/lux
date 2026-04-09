// Package form provides interactive form elements for the Lux UI framework.
package form

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// Layout constants for checkbox and radio (shared).
const (
	checkboxSize   = 16
	checkboxGap    = 8
	checkboxBorder = 1
)

// Checkbox is a boolean toggle with a label.
type Checkbox struct {
	ui.BaseElement
	Label    string
	Checked  bool
	OnToggle func(bool)
	Disabled bool
}

// NewCheckbox creates a checkbox element.
func NewCheckbox(label string, checked bool, onToggle func(bool)) ui.Element {
	return Checkbox{Label: label, Checked: checked, OnToggle: onToggle}
}

// CheckboxDisabled creates a disabled checkbox.
func CheckboxDisabled(label string, checked bool) ui.Element {
	return Checkbox{Label: label, Checked: checked, Disabled: true}
}

// LayoutSelf implements ui.Layouter.
func (n Checkbox) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	// Touch-adaptive sizing: grow checkbox to meet MinTouchTarget (RFC-004 §2).
	boxSize := checkboxSize
	gap := checkboxGap
	if ctx.IsTouch() && ctx.Profile != nil {
		boxSize = int(ctx.Profile.MinTouchTarget)
		gap = int(ctx.Profile.TouchTargetSpacing) + 4
	}

	style := tokens.Typography.Body
	metrics := canvas.MeasureText(n.Label, style)
	labelW := int(math.Ceil(float64(metrics.Width)))
	labelH := int(math.Ceil(float64(metrics.Ascent)))
	totalH := max(boxSize, labelH)
	totalW := boxSize + gap + labelW

	// Register hit target and get hover opacity atomically.
	checkboxRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(totalH))
	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterHit(checkboxRect, nil)
	} else {
		var clickFn func()
		if n.OnToggle != nil {
			checked := n.Checked
			onToggle := n.OnToggle
			clickFn = func() { onToggle(!checked) }
		}
		hoverOpacity = ix.RegisterHit(checkboxRect, clickFn)
	}

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	boxY := area.Y + (totalH-boxSize)/2
	boxRect := draw.R(float32(area.X), float32(boxY), float32(boxSize), float32(boxSize))

	// Border
	borderColor := tokens.Colors.Stroke.Border
	if n.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(boxRect,
		tokens.Radii.Input, draw.SolidPaint(borderColor))

	// Fill — two-stage hover→pressed visual (RFC-008 §9.3).
	fillColor := tokens.Colors.Surface.Elevated
	if hoverOpacity > 0 {
		fillColor = ui.LerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
		if hoverOpacity >= 0.9 {
			pressedT := (hoverOpacity - 0.9) / 0.1
			fillColor = ui.LerpColor(fillColor, tokens.Colors.Surface.Pressed, pressedT)
		}
	}
	if n.Checked {
		fillColor = tokens.Colors.Accent.Primary
	}
	if n.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+checkboxBorder), float32(boxY+checkboxBorder),
			float32(boxSize-checkboxBorder*2), float32(boxSize-checkboxBorder*2)),
		maxf(tokens.Radii.Input-checkboxBorder, 0), draw.SolidPaint(fillColor))

	// Focus glow on the checkbox box (RFC-008 §9.4).
	if focused {
		ui.DrawFocusRing(canvas, boxRect, tokens.Radii.Input, tokens)
	}

	// Checkmark (Phosphor icon)
	if n.Checked {
		checkStyle := draw.TextStyle{
			FontFamily: "Phosphor",
			Size:       float32(boxSize - checkboxBorder*2 - 2),
			Weight:     draw.FontWeightRegular,
			LineHeight: 1.0,
			Raster:     true,
		}
		canvas.DrawText(icons.Check,
			draw.Pt(float32(area.X+checkboxBorder+1), float32(boxY+checkboxBorder+1)),
			checkStyle, tokens.Colors.Text.OnAccent)
	}

	// Label
	labelX := area.X + boxSize + gap
	labelY := area.Y + (totalH-labelH)/2
	labelColor := tokens.Colors.Text.Primary
	if n.Disabled {
		labelColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(n.Label, draw.Pt(float32(labelX), float32(labelY)), style, labelColor)

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Checkbox) TreeEqual(other ui.Element) bool {
	nb, ok := other.(Checkbox)
	return ok && n.Label == nb.Label && n.Checked == nb.Checked
}

// ResolveChildren implements ui.ChildResolver. Checkbox is a leaf.
func (n Checkbox) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Checkbox) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	an := a11y.AccessNode{
		Role:   a11y.RoleCheckbox,
		Label:  n.Label,
		States: a11y.AccessStates{Checked: n.Checked},
	}
	if n.OnToggle != nil {
		toggle := n.OnToggle
		checked := n.Checked
		an.Actions = []a11y.AccessAction{{Name: "activate", Trigger: func() { toggle(!checked) }}}
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
