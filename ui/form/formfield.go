package form

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
	"github.com/timzifer/lux/validation"
)

// FormField layout constants.
const (
	formFieldLabelGap = 4  // gap between label and child
	formFieldHintGap  = 4  // gap between child and hint/error text
	infoIconSize      = 14 // size of the info icon in dp
	infoIconPad       = 4  // padding around the info icon hit target
)

// FormField wraps an arbitrary form element with an optional label, hint,
// and validation error message. It acts as a decorator that adds context
// without modifying the inner field element.
//
// The hint display mode is determined by the theme's HintMode token,
// but can be overridden per field via [WithHintMode].
//
//   - [theme.HintModeLabel] renders the hint as a small text label below the field.
//   - [theme.HintModeIcon] renders an info-icon button next to the label;
//     hovering it shows the hint text in a tooltip.
type FormField struct {
	ui.BaseElement
	Child    ui.Element            // the wrapped form element
	Label    string                // optional label displayed above the field
	Hint     string                // optional help text
	Result   validation.FieldResult // validation result; non-empty Error triggers error state
	HintMode *theme.HintMode       // nil = use theme default
}

// FormFieldOption configures a FormField.
type FormFieldOption func(*FormField)

// WithLabel sets the label displayed above the field.
func WithLabel(label string) FormFieldOption {
	return func(f *FormField) { f.Label = label }
}

// WithHint sets the hint text for the field.
func WithHint(hint string) FormFieldOption {
	return func(f *FormField) { f.Hint = hint }
}

// WithValidation attaches a validation result to the field.
func WithValidation(r validation.FieldResult) FormFieldOption {
	return func(f *FormField) { f.Result = r }
}

// WithHintMode overrides the theme's default hint display mode for this field.
func WithHintMode(mode theme.HintMode) FormFieldOption {
	return func(f *FormField) { f.HintMode = &mode }
}

// NewFormField creates a FormField wrapping the given child element.
func NewFormField(child ui.Element, opts ...FormFieldOption) ui.Element {
	ff := FormField{Child: child}
	for _, opt := range opts {
		opt(&ff)
	}
	return ff
}

// hintMode returns the effective hint mode for this field.
func (n FormField) hintMode(tokens theme.TokenSet) theme.HintMode {
	if n.HintMode != nil {
		return *n.HintMode
	}
	return tokens.HintMode
}

// LayoutSelf implements ui.Layouter.
func (n FormField) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas

	labelStyle := tokens.Typography.Label
	hintStyle := tokens.Typography.LabelSmall
	hasError := !n.Result.Valid()
	hMode := n.hintMode(tokens)

	y := area.Y
	totalW := area.W

	// ── Label row (label + optional info icon) ──────────────────

	if n.Label != "" {
		labelColor := tokens.Colors.Text.Secondary
		if hasError {
			labelColor = tokens.Colors.Status.Error
		}

		labelMetrics := canvas.MeasureText(n.Label, labelStyle)
		labelW := int(math.Ceil(float64(labelMetrics.Width)))
		labelH := int(math.Ceil(float64(labelMetrics.Ascent)))

		canvas.DrawText(n.Label, draw.Pt(float32(area.X), float32(y)), labelStyle, labelColor)

		// Info icon next to label (HintModeIcon).
		if n.Hint != "" && hMode == theme.HintModeIcon {
			n.renderInfoIcon(ctx, area.X+labelW+infoIconPad, y, labelH)
		}

		y += labelH + formFieldLabelGap
	} else if n.Hint != "" && hMode == theme.HintModeIcon {
		// No label but hint with icon mode: render icon inline before child.
		n.renderInfoIcon(ctx, area.X, y, infoIconSize)
	}

	// ── Child element ───────────────────────────────────────────

	childArea := ui.Bounds{X: area.X, Y: y, W: totalW, H: area.H - (y - area.Y)}
	childBounds := ctx.LayoutChild(n.Child, childArea)
	y += childBounds.H

	// ── Error border tint on child ──────────────────────────────

	if hasError {
		errBorderRect := draw.R(
			float32(childBounds.X)-1,
			float32(childBounds.Y)-1,
			float32(childBounds.W)+2,
			float32(childBounds.H)+2,
		)
		errColor := tokens.Colors.Status.Error
		errColor.A = 0.6
		canvas.StrokeRoundRect(errBorderRect, tokens.Radii.Input, draw.Stroke{
			Paint: draw.SolidPaint(errColor),
			Width: 1.5,
		})
	}

	// ── Hint text (HintModeLabel) or error message ──────────────

	belowText := ""
	belowColor := tokens.Colors.Text.Secondary
	if hasError {
		belowText = n.Result.Error
		belowColor = tokens.Colors.Status.Error
	} else if n.Hint != "" && hMode == theme.HintModeLabel {
		belowText = n.Hint
		belowColor = tokens.Colors.Text.Secondary
	}

	if belowText != "" {
		y += formFieldHintGap
		canvas.DrawText(belowText, draw.Pt(float32(area.X), float32(y)), hintStyle, belowColor)
		m := canvas.MeasureText(belowText, hintStyle)
		y += int(math.Ceil(float64(m.Ascent)))
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: y - area.Y}
}

// renderInfoIcon draws a small info icon that reveals the hint text as a
// tooltip on hover. ix is used for hit-target registration.
func (n FormField) renderInfoIcon(ctx *ui.LayoutContext, x, y, alignH int) {
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	_ = ix

	iconStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       float32(infoIconSize),
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}

	// Center icon vertically with the label.
	iconY := y + (alignH-infoIconSize)/2
	if iconY < y {
		iconY = y
	}

	iconRect := draw.R(float32(x), float32(iconY), float32(infoIconSize), float32(infoIconSize))

	// Register hover hit target.
	hoverOpacity := float32(0)
	if ix != nil {
		hoverOpacity = ix.RegisterHit(iconRect, nil)
	}

	// Draw the icon.
	iconColor := tokens.Colors.Text.Secondary
	if hoverOpacity > 0.1 {
		iconColor = tokens.Colors.Accent.Primary
	}
	canvas.DrawText(icons.Info, draw.Pt(float32(x), float32(iconY)), iconStyle, iconColor)

	// Show tooltip on hover.
	if hoverOpacity > 0.1 && ctx.Overlays != nil {
		hint := n.Hint
		th := ctx.Theme
		ctx.Overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				hintStyle := tokens.Typography.LabelSmall
				m := canvas.MeasureText(hint, hintStyle)
				tw := int(math.Ceil(float64(m.Width)))
				th2 := int(math.Ceil(float64(m.Ascent)))
				pad := 6

				tipW := tw + pad*2
				tipH := th2 + pad*2
				tipX := int(iconRect.X)
				tipY := int(iconRect.Y+iconRect.H) + 4

				tipRect := draw.R(float32(tipX), float32(tipY), float32(tipW), float32(tipH))

				_ = th
				// Background.
				canvas.FillRoundRect(tipRect,
					tokens.Radii.Button, draw.SolidPaint(tokens.Colors.Stroke.Border))
				innerRect := draw.R(float32(tipX+1), float32(tipY+1), float32(max(tipW-2, 0)), float32(max(tipH-2, 0)))
				canvas.FillRoundRect(innerRect,
					maxf(tokens.Radii.Button-1, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

				// Border.
				canvas.StrokeRoundRect(tipRect, tokens.Radii.Button, draw.Stroke{
					Paint: draw.SolidPaint(tokens.Colors.Stroke.Border),
					Width: 1,
				})

				// Text.
				canvas.DrawText(hint,
					draw.Pt(float32(tipX+pad), float32(tipY+pad)),
					hintStyle, tokens.Colors.Text.Primary)
			},
		})
	}
}

// TreeEqual implements ui.TreeEqualizer.
func (n FormField) TreeEqual(other ui.Element) bool {
	o, ok := other.(FormField)
	return ok && n.Label == o.Label && n.Hint == o.Hint && n.Result == o.Result
}

// ResolveChildren implements ui.ChildResolver.
func (n FormField) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	n.Child = resolve(n.Child, 0)
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n FormField) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	groupNode := a11y.AccessNode{
		Role:  a11y.RoleGroup,
		Label: n.Label,
	}
	if n.Hint != "" {
		groupNode.Description = n.Hint
	}
	if !n.Result.Valid() {
		groupNode.States.Invalid = true
		groupNode.Description = n.Result.Error
	}
	idx := b.AddNode(groupNode, parentIdx, a11y.Rect{})
	b.Walk(n.Child, int32(idx))
}

