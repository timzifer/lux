package button

import (
	"math"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
)

// ── ConfirmButton (RFC-004 §4.3, Stufe 2) ─────────────────────
//
// A two-step confirmation button: first tap enters confirm state
// (different colour + label), second tap within the timeout fires
// OnConfirm. If the timeout expires the button resets silently.

const (
	defaultConfirmTimeout = 3 * time.Second
	confirmTimeoutBarH    = 3 // height of the countdown bar (dp)
)

// confirmPhase tracks the button's two-step state.
type confirmPhase uint8

const (
	confirmIdle    confirmPhase = iota
	confirmPending              // waiting for second tap
)

// ConfirmButtonState holds mutable animation / phase state.
// Allocate with NewConfirmButtonState and store in your Model.
type ConfirmButtonState struct {
	phase       confirmPhase
	timeoutAnim anim.Anim[float32] // 1 → 0 over ConfirmTimeout
	ripple      RippleState
}

// NewConfirmButtonState creates a ready-to-use state.
func NewConfirmButtonState() *ConfirmButtonState { return &ConfirmButtonState{} }

// Tick advances internal animations. Call from your update on TickMsg.
func (s *ConfirmButtonState) Tick(dt time.Duration) bool {
	r := s.ripple.Tick(dt)
	t := s.timeoutAnim.Tick(dt)

	// Auto-reset when timeout animation finishes.
	if s.phase == confirmPending && s.timeoutAnim.IsDone() {
		s.phase = confirmIdle
	}
	return r || t
}

// ConfirmButton is an element that requires two taps for critical actions.
type ConfirmButton struct {
	ui.BaseElement
	Label          string
	ConfirmLabel   string        // shown during confirm phase
	Variant        ui.ButtonVariant
	OnConfirm      func()
	ConfirmTimeout time.Duration // 0 → defaultConfirmTimeout
	State          *ConfirmButtonState
}

// Confirm creates a filled ConfirmButton with sensible defaults.
func Confirm(label, confirmLabel string, onConfirm func(), state *ConfirmButtonState) ui.Element {
	return ConfirmButton{
		Label:        label,
		ConfirmLabel: confirmLabel,
		Variant:      ui.ButtonFilled,
		OnConfirm:    onConfirm,
		State:        state,
	}
}

// LayoutSelf implements ui.Layouter.
func (n ConfirmButton) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas
	ix := ctx.IX
	fs := ctx.Focus
	st := n.State
	if st == nil {
		st = &ConfirmButtonState{}
	}

	timeout := n.ConfirmTimeout
	if timeout == 0 {
		timeout = defaultConfirmTimeout
	}

	// Determine label for current phase.
	label := n.Label
	if st.phase == confirmPending && n.ConfirmLabel != "" {
		label = n.ConfirmLabel
	}

	// Measure label.
	style := tokens.Typography.Label
	metrics := canvas.MeasureText(label, style)
	contentW := int(math.Ceil(float64(metrics.Width)))
	contentH := int(math.Ceil(float64(metrics.Ascent)))
	w := contentW + (ui.ButtonPadX * 2)
	h := contentH + (ui.ButtonPadY * 2)

	buttonRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Focus — register before click handler so uid is available in closure.
	var focused bool
	var focusUID ui.UID
	if fs != nil {
		focusUID = fs.NextElementUID()
		fs.RegisterFocusable(focusUID, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = fs.IsElementFocused(focusUID)
	}

	// Register positional click for ripple origin.
	var hoverOpacity float32
	hoverOpacity = ix.RegisterClickAt(buttonRect, func(x, y float32) {
		switch st.phase {
		case confirmIdle:
			st.phase = confirmPending
			st.timeoutAnim.SetImmediate(1)
			st.timeoutAnim.SetTarget(0, timeout, anim.Linear)
			st.ripple.Trigger(x, y, maxRippleRadius(x, y, buttonRect.X, buttonRect.Y, buttonRect.W, buttonRect.H))
		case confirmPending:
			st.phase = confirmIdle
			st.ripple.Trigger(x, y, maxRippleRadius(x, y, buttonRect.X, buttonRect.Y, buttonRect.W, buttonRect.H))
			if n.OnConfirm != nil {
				n.OnConfirm()
			}
		}
		// Focus the button on click to ensure a repaint is triggered,
		// so the confirm-phase visual change becomes visible.
		if fs != nil {
			fs.SetFocusedUID(focusUID)
		}
	})

	// Colours: idle = normal variant, pending = danger (red).
	var fillColor, borderColor, textColor draw.Color
	if st.phase == confirmPending {
		fillColor = tokens.Colors.Status.Error
		borderColor = draw.Color{R: fillColor.R * 0.7, G: fillColor.G * 0.7, B: fillColor.B * 0.7, A: 1}
		textColor = tokens.Colors.Text.OnAccent
		if hoverOpacity > 0 {
			fillColor = ui.LerpColor(fillColor, ui.HoverHighlight(fillColor), hoverOpacity)
		}
	} else {
		fillColor, borderColor, textColor = ui.ButtonVariantColors(n.Variant, tokens, hoverOpacity)
	}

	// Draw button background (same 2-rect approach as button.Button).
	canvas.FillRoundRect(buttonRect, tokens.Radii.Button, draw.SolidPaint(borderColor))
	innerRadius := tokens.Radii.Button - float32(ui.ButtonBorder)
	if innerRadius < 0 {
		innerRadius = 0
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+ui.ButtonBorder), float32(area.Y+ui.ButtonBorder),
			float32(max(w-ui.ButtonBorder*2, 0)), float32(max(h-ui.ButtonBorder*2, 0))),
		innerRadius, draw.SolidPaint(fillColor))

	// Focus ring.
	if focused {
		ui.DrawFocusRing(canvas, buttonRect, tokens.Radii.Button, tokens)
	}

	// Ripple overlay.
	st.ripple.Draw(canvas, buttonRect, tokens.Radii.Button, tokens.Colors.Accent.Primary)

	// Timeout countdown bar at bottom of button.
	if st.phase == confirmPending {
		progress := st.timeoutAnim.Value() // 1 → 0
		barW := float32(w) * progress
		barY := float32(area.Y+h) - confirmTimeoutBarH
		barRect := draw.R(float32(area.X), barY, barW, confirmTimeoutBarH)
		barColor := draw.Color{R: 1, G: 1, B: 1, A: 0.4}
		canvas.FillRoundRect(barRect, 1, draw.SolidPaint(barColor))
	}

	// Draw label centred.
	canvas.DrawText(label,
		draw.Pt(float32(area.X+(w-contentW)/2), float32(area.Y+(h-contentH)/2)),
		style, textColor)

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: ui.ButtonPadY + contentH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n ConfirmButton) TreeEqual(other ui.Element) bool {
	_, ok := other.(ConfirmButton)
	return ok && false
}

// ResolveChildren implements ui.ChildResolver.
func (n ConfirmButton) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n ConfirmButton) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	label := n.Label
	accessNode := a11y.AccessNode{
		Role:  a11y.RoleButton,
		Label: label,
	}
	if n.OnConfirm != nil {
		accessNode.Actions = []a11y.AccessAction{
			{Name: "activate", Trigger: n.OnConfirm},
		}
	}
	b.AddNode(accessNode, parentIdx, a11y.Rect{})
}

// extractConfirmLabel returns the idle-phase label for accessibility.
func extractConfirmLabel(el ui.Element) string {
	if txt, ok := el.(display.TextElement); ok {
		return txt.Content
	}
	return ""
}
