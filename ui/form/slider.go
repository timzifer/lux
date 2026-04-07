package form

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Layout constants for slider.
const (
	sliderTrackH   = 4
	sliderHeight   = 20
	sliderThumbD   = 16
	sliderMaxWidth = 200
)

// Slider is a continuous value selector (0.0-1.0).
type Slider struct {
	ui.BaseElement
	Value    float32
	OnChange func(float32)
	Disabled bool
}

// NewSlider creates a slider element.
func NewSlider(value float32, onChange func(float32)) ui.Element {
	return Slider{Value: value, OnChange: onChange}
}

// SliderDisabled creates a disabled slider.
func SliderDisabled(value float32) ui.Element {
	return Slider{Value: value, Disabled: true}
}

// LayoutSelf implements ui.Layouter.
func (n Slider) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	// Touch-adaptive sizing: grow thumb to meet MinTouchTarget.
	thumbD := sliderThumbD
	height := sliderHeight
	if ctx.IsTouch() && ctx.Profile != nil {
		thumbD = int(ctx.Profile.MinTouchTarget)
		height = thumbD + 4
	}

	trackW := sliderMaxWidth
	if area.W < trackW {
		trackW = area.W
	}

	// Register draggable hit target and get hover opacity atomically.
	sliderRect := draw.R(float32(area.X), float32(area.Y), float32(trackW), float32(height))
	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterDrag(sliderRect, nil)
	} else {
		var dragFn func(x, y float32)
		if n.OnChange != nil {
			areaX := float32(area.X)
			tw := float32(trackW)
			onChange := n.OnChange
			dragFn = func(x, _ float32) {
				v := (x - areaX) / tw
				if v < 0 {
					v = 0
				}
				if v > 1 {
					v = 1
				}
				onChange(v)
			}
		}
		hoverOpacity = ix.RegisterDrag(sliderRect, dragFn)
	}

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	trackY := area.Y + (height-sliderTrackH)/2

	// Track background
	trackColor := tokens.Colors.Surface.Pressed
	if hoverOpacity > 0 {
		trackColor = ui.LerpColor(trackColor, tokens.Colors.Surface.Hovered, hoverOpacity)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(trackY), float32(trackW), float32(sliderTrackH)),
		float32(sliderTrackH)/2, draw.SolidPaint(trackColor))

	// Filled portion
	val := n.Value
	if val < 0 {
		val = 0
	}
	if val > 1 {
		val = 1
	}
	filledW := int(float32(trackW) * val)
	filledColor := tokens.Colors.Accent.Primary
	if n.Disabled {
		filledColor = ui.DisabledColor(filledColor, tokens.Colors.Surface.Base)
	}
	if filledW > 0 {
		canvas.FillRoundRect(
			draw.R(float32(area.X), float32(trackY), float32(filledW), float32(sliderTrackH)),
			float32(sliderTrackH)/2, draw.SolidPaint(filledColor))
	}

	// Thumb — pressed visual differentiation (RFC-008 §9.3).
	thumbX := area.X + filledW - thumbD/2
	if thumbX < area.X {
		thumbX = area.X
	}
	thumbY := area.Y + (height-thumbD)/2
	thumbRect := draw.R(float32(thumbX), float32(thumbY), float32(thumbD), float32(thumbD))
	thumbColor := tokens.Colors.Accent.Primary
	if hoverOpacity >= 0.9 {
		pressedT := (hoverOpacity - 0.9) / 0.1
		thumbColor = ui.LerpColor(thumbColor, tokens.Colors.Accent.Secondary, pressedT)
	}
	if n.Disabled {
		thumbColor = ui.DisabledColor(thumbColor, tokens.Colors.Surface.Base)
	}
	canvas.FillEllipse(thumbRect, draw.SolidPaint(thumbColor))

	// Focus glow on the slider thumb (RFC-008 §9.4).
	if focused {
		ui.DrawFocusRing(canvas, thumbRect, float32(thumbD)/2, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: trackW, H: height}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Slider) TreeEqual(other ui.Element) bool {
	nb, ok := other.(Slider)
	return ok && n.Value == nb.Value
}

// ResolveChildren implements ui.ChildResolver. Slider is a leaf.
func (n Slider) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Slider) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	an := a11y.AccessNode{
		Role:   a11y.RoleSlider,
		States: a11y.AccessStates{Disabled: n.Disabled},
		NumericValue: &a11y.AccessNumericValue{
			Current: float64(n.Value),
			Min:     0,
			Max:     1,
			Step:    0, // continuous
		},
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}
