package form

import (
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// ToggleState tracks the toggle thumb animation.
type ToggleState struct {
	thumbPos anim.Anim[float32] // 0.0 = off, 1.0 = on
	lastOn   bool
	inited   bool
}

// NewToggleState creates a ready-to-use ToggleState.
func NewToggleState() *ToggleState { return &ToggleState{} }

// Update returns the current animation progress [0,1] and starts a
// new transition if the on state has changed.
func (ts *ToggleState) Update(on bool, de theme.DurationEasing) float32 {
	if !ts.inited {
		if on {
			ts.thumbPos.SetImmediate(1.0)
		}
		ts.lastOn = on
		ts.inited = true
		return ts.thumbPos.Value()
	}
	if on != ts.lastOn {
		target := float32(0)
		if on {
			target = 1
		}
		ts.thumbPos.SetTarget(target, de.Duration, de.Easing)
		ts.lastOn = on
	}
	return ts.thumbPos.Value()
}

// Tick advances the toggle animation by dt.
func (ts *ToggleState) Tick(dt time.Duration) {
	if ts != nil {
		ts.thumbPos.Tick(dt)
	}
}

// Layout constants for toggle.
const (
	toggleTrackW   = 36
	toggleTrackH   = 20
	toggleThumbD   = 16
	toggleThumbPad = 2
)

// Toggle is a switch widget with smooth thumb animation.
type Toggle struct {
	ui.BaseElement
	On       bool
	OnToggle func(bool)
	State    *ToggleState
	Disabled bool
}

// NewToggle creates a toggle element. An optional ToggleState pointer enables
// smooth thumb animation; pass nil for instant snap.
func NewToggle(on bool, onToggle func(bool), state ...*ToggleState) ui.Element {
	var s *ToggleState
	if len(state) > 0 {
		s = state[0]
	}
	return Toggle{On: on, OnToggle: onToggle, State: s}
}

// ToggleDisabled creates a disabled toggle.
func ToggleDisabled(on bool) ui.Element {
	return Toggle{On: on, Disabled: true}
}

// LayoutSelf implements ui.Layouter.
func (n Toggle) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	// Register hit target and get hover opacity atomically.
	toggleRect := draw.R(float32(area.X), float32(area.Y), float32(toggleTrackW), float32(toggleTrackH))
	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterHit(toggleRect, nil)
	} else {
		var toggleClickFn func()
		if n.OnToggle != nil {
			on := n.On
			onToggle := n.OnToggle
			toggleClickFn = func() { onToggle(!on) }
		}
		hoverOpacity = ix.RegisterHit(toggleRect, toggleClickFn)
	}

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	// Animation progress: 0 = off, 1 = on.
	var t float32
	if n.State != nil {
		t = n.State.Update(n.On, tokens.Motion.Quick)
	} else {
		if n.On {
			t = 1
		}
	}

	// Track — lerp between off and on colors.
	offTrackColor := tokens.Colors.Surface.Pressed
	onTrackColor := tokens.Colors.Accent.Primary
	var trackColor draw.Color
	switch {
	case t <= 0:
		trackColor = offTrackColor
	case t >= 1:
		trackColor = onTrackColor
	default:
		trackColor = ui.LerpColor(offTrackColor, onTrackColor, t)
	}
	if hoverOpacity > 0 {
		trackColor = ui.LerpColor(trackColor, ui.HoverHighlight(trackColor), hoverOpacity)
		if hoverOpacity >= 0.9 {
			pressedT := (hoverOpacity - 0.9) / 0.1
			trackColor = ui.LerpColor(trackColor, tokens.Colors.Surface.Pressed, pressedT*0.3)
		}
	}
	if n.Disabled {
		trackColor = ui.DisabledColor(trackColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(toggleTrackW), float32(toggleTrackH)),
		float32(toggleTrackH)/2, draw.SolidPaint(trackColor))

	// Thumb — lerp position and color.
	offX := float32(area.X + toggleThumbPad)
	onX := float32(area.X + toggleTrackW - toggleThumbD - toggleThumbPad)
	thumbX := offX + (onX-offX)*t
	thumbY := float32(area.Y + (toggleTrackH-toggleThumbD)/2)
	offThumbColor := tokens.Colors.Text.Secondary
	onThumbColor := tokens.Colors.Text.OnAccent
	var thumbColor draw.Color
	switch {
	case t <= 0:
		thumbColor = offThumbColor
	case t >= 1:
		thumbColor = onThumbColor
	default:
		thumbColor = ui.LerpColor(offThumbColor, onThumbColor, t)
	}
	if n.Disabled {
		thumbColor = ui.DisabledColor(thumbColor, tokens.Colors.Surface.Base)
	}
	canvas.FillEllipse(
		draw.R(thumbX, thumbY, float32(toggleThumbD), float32(toggleThumbD)),
		draw.SolidPaint(thumbColor))

	// Focus glow on the toggle track (RFC-008 §9.4).
	if focused {
		ui.DrawFocusRing(canvas, toggleRect, float32(toggleTrackH)/2, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: toggleTrackW, H: toggleTrackH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Toggle) TreeEqual(other ui.Element) bool {
	nb, ok := other.(Toggle)
	return ok && n.On == nb.On
}

// ResolveChildren implements ui.ChildResolver. Toggle is a leaf.
func (n Toggle) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Toggle) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	an := a11y.AccessNode{
		Role:   a11y.RoleToggle,
		States: a11y.AccessStates{Checked: n.On},
	}
	if n.OnToggle != nil {
		toggle := n.OnToggle
		on := n.On
		an.Actions = []a11y.AccessAction{{Name: "activate", Trigger: func() { toggle(!on) }}}
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}
