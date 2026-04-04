package button

import (
	"math"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ── HoldButton (RFC-004 §4.3, Stufe 3) ────────────────────────
//
// A button that must be held for a configured duration before the
// action fires. A radial progress ring around the button fills
// clockwise during the hold; releasing early cancels.

const (
	defaultHoldDuration = 2 * time.Second
	holdReleaseSnapDur  = 200 * time.Millisecond // snap-back duration on release
	holdFlashDuration   = 900 * time.Millisecond // completion blink duration
	holdFlashCycles     = 3.0                    // number of blink cycles
	progressRingStroke  = 4.0                    // ring stroke width (dp)
	progressRingGap     = 6.0                    // gap between button edge and ring

	// Minimum ring radius based on a standard finger pad (~10 mm).
	// At the baseline density (160 dpi) 1 dp ≈ 0.16 mm, so 10 mm ≈ 63 dp
	// → radius ≈ 28 dp.  The ring is drawn at least this large so it
	// always encloses the touch area regardless of button dimensions.
	fingerRadius = 28.0 // dp – half of a ~10 mm finger pad
)

// HoldButtonState holds mutable animation state for a HoldButton.
// Allocate with NewHoldButtonState and store in your Model.
type HoldButtonState struct {
	holding    bool
	touchX, touchY float32              // touch origin for ring centering
	progress   anim.Anim[float32]      // 0→1 (fill) or 1→0 (release snap-back)
	flashAnim  anim.Anim[float32]      // 0→1 over ~900ms for completion blink
	flashing   bool                    // true during post-completion flash
	ripple     ui.RippleState
	completed  bool // latched on completion, reset on next press
}

// NewHoldButtonState creates a ready-to-use state.
func NewHoldButtonState() *HoldButtonState { return &HoldButtonState{} }

// Tick advances internal animations. Call from your update on TickMsg.
// Returns true if still animating.
func (s *HoldButtonState) Tick(dt time.Duration) bool {
	r := s.ripple.Tick(dt)
	p := s.progress.Tick(dt)
	f := s.flashAnim.Tick(dt)

	// Detect hold completion → start flash.
	if s.holding && s.progress.IsDone() && s.progress.Value() >= 0.99 {
		s.completed = true
		s.holding = false
		s.flashing = true
		s.flashAnim.SetImmediate(0)
		s.flashAnim.SetTarget(1, holdFlashDuration, anim.Linear)
		f = true
	}

	// End of flash → hide ring.
	if s.flashing && s.flashAnim.IsDone() {
		s.flashing = false
		s.progress.SetImmediate(0)
	}

	return r || p || f
}

// IsCompleted returns true once and resets the flag. Use in your update
// to detect when the hold gesture finishes.
func (s *HoldButtonState) IsCompleted() bool {
	if s.completed {
		s.completed = false
		return true
	}
	return false
}

// HoldButton is an element requiring a sustained press to activate.
type HoldButton struct {
	ui.BaseElement
	Label        string
	HoldDuration time.Duration // 0 → defaultHoldDuration
	OnComplete   func()
	Variant      ui.ButtonVariant
	State        *HoldButtonState
}

// Hold creates a filled HoldButton with sensible defaults.
func Hold(label string, holdDur time.Duration, onComplete func(), state *HoldButtonState) ui.Element {
	return HoldButton{
		Label:        label,
		HoldDuration: holdDur,
		Variant:      ui.ButtonFilled,
		OnComplete:   onComplete,
		State:        state,
	}
}

// LayoutSelf implements ui.Layouter.
func (n HoldButton) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas
	ix := ctx.IX
	fs := ctx.Focus
	st := n.State
	if st == nil {
		st = &HoldButtonState{}
	}

	holdDur := n.HoldDuration
	if holdDur == 0 {
		holdDur = defaultHoldDuration
	}

	// Measure label.
	style := tokens.Typography.Label
	metrics := canvas.MeasureText(n.Label, style)
	contentW := int(math.Ceil(float64(metrics.Width)))
	contentH := int(math.Ceil(float64(metrics.Ascent)))
	w := contentW + (ui.ButtonPadX * 2)
	h := contentH + (ui.ButtonPadY * 2)

	buttonRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Register drag + release for press-and-hold detection.
	// onDrag fires continuously while held; onRelease fires on finger-up.
	hoverOpacity := ix.RegisterSurfaceDrag(buttonRect,
		func(x, y float32) {
			// On first drag call, start the hold.
			if !st.holding && !st.completed {
				st.holding = true
				st.touchX, st.touchY = x, y
				st.progress.SetImmediate(0)
				st.progress.SetTarget(1, holdDur, anim.Linear)
				st.ripple.Trigger(x, y, maxRippleRadius(x, y, buttonRect.X, buttonRect.Y, buttonRect.W, buttonRect.H))
			}
		},
		func(x, y float32) {
			// Release: if not completed, snap back.
			if st.holding {
				st.holding = false
				cur := st.progress.Value()
				if cur < 0.99 {
					st.progress.SetImmediate(cur)
					st.progress.SetTarget(0, holdReleaseSnapDur, anim.OutCubic)
				}
			}
		},
	)

	// Focus.
	var focused bool
	if fs != nil {
		uid := fs.NextElementUID()
		fs.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = fs.IsElementFocused(uid)
	}

	// Check completion and fire callback.
	if st.IsCompleted() && n.OnComplete != nil {
		n.OnComplete()
	}

	// Colours: use status error (red/danger) for hold buttons.
	fillColor := tokens.Colors.Status.Error
	borderColor := draw.Color{R: fillColor.R * 0.7, G: fillColor.G * 0.7, B: fillColor.B * 0.7, A: 1}
	textColor := tokens.Colors.Text.OnAccent
	if hoverOpacity > 0 {
		fillColor = ui.LerpColor(fillColor, ui.HoverHighlight(fillColor), hoverOpacity)
	}

	// Draw button background.
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

	// Ripple overlay — use text colour so the pulse contrasts with the button fill.
	st.ripple.Draw(canvas, buttonRect, tokens.Radii.Button, textColor)

	// Draw label centred.
	canvas.DrawText(n.Label,
		draw.Pt(float32(area.X+(w-contentW)/2), float32(area.Y+(h-contentH)/2)),
		style, textColor)

	// ── Radial progress ring (centered on touch point) ────────
	if st.flashing {
		flashT := st.flashAnim.Value()
		blinkOpacity := float32(math.Abs(math.Sin(float64(flashT) * holdFlashCycles * math.Pi)))
		drawProgressRing(canvas, st.touchX, st.touchY, 1.0, blinkOpacity, tokens.Colors.Accent.Primary)
	} else if progress := st.progress.Value(); progress > 0.001 {
		drawProgressRing(canvas, st.touchX, st.touchY, progress, 1.0, tokens.Colors.Accent.Primary)
	}

	// Total bounds include the fixed-size ring outside the button.
	ringR := fingerRadius + progressRingGap + progressRingStroke + 1
	ringMarginX := int(ringR) - w/2
	ringMarginY := int(ringR) - h/2
	if ringMarginX < 0 {
		ringMarginX = 0
	}
	if ringMarginY < 0 {
		ringMarginY = 0
	}
	return ui.Bounds{
		X: area.X - ringMarginX, Y: area.Y - ringMarginY,
		W: w + ringMarginX*2, H: h + ringMarginY*2,
		Baseline: ringMarginY + ui.ButtonPadY + contentH,
	}
}

// drawProgressRing draws a clockwise circular arc from 12-o'clock,
// centered on (cx, cy) — typically the touch origin.
func drawProgressRing(canvas draw.Canvas, cx, cy, progress, opacity float32, color draw.Color) {
	if progress <= 0 || opacity <= 0 {
		return
	}

	// Fixed radius based on a standard finger pad (~10 mm ≈ 28 dp).
	r := float32(fingerRadius) + progressRingGap

	ringColor := draw.Color{R: color.R, G: color.G, B: color.B, A: 0.9 * opacity}

	if progress >= 0.999 {
		// Full ring — draw two semicircles.
		p := draw.NewPath().
			MoveTo(draw.Pt(cx, cy-r)).
			ArcTo(r, r, 0, false, true, draw.Pt(cx, cy+r)).
			ArcTo(r, r, 0, false, true, draw.Pt(cx, cy-r)).
			Close().
			Build()
		canvas.StrokePath(p, draw.Stroke{
			Paint: draw.SolidPaint(ringColor),
			Width: progressRingStroke,
		})
		return
	}

	// Partial arc: start at 12-o'clock, sweep clockwise by progress * 2*PI.
	angle := float64(progress) * 2 * math.Pi
	endX := cx + r*float32(math.Sin(angle))
	endY := cy - r*float32(math.Cos(angle))

	large := progress > 0.5

	p := draw.NewPath().
		MoveTo(draw.Pt(cx, cy-r)).
		ArcTo(r, r, 0, large, true, draw.Pt(endX, endY)).
		Build()

	canvas.StrokePath(p, draw.Stroke{
		Paint: draw.SolidPaint(ringColor),
		Width: progressRingStroke,
	})
}

// TreeEqual implements ui.TreeEqualizer.
func (n HoldButton) TreeEqual(other ui.Element) bool {
	_, ok := other.(HoldButton)
	return ok && false
}

// ResolveChildren implements ui.ChildResolver.
func (n HoldButton) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n HoldButton) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	accessNode := a11y.AccessNode{
		Role:  a11y.RoleButton,
		Label: n.Label,
	}
	if n.OnComplete != nil {
		accessNode.Actions = []a11y.AccessAction{
			{Name: "activate", Trigger: n.OnComplete},
		}
	}
	b.AddNode(accessNode, parentIdx, a11y.Rect{})
}
