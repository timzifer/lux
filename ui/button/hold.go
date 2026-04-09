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
	defaultHoldDuration  = 2 * time.Second
	holdFlashDuration    = 400 * time.Millisecond // completion blink duration (snappy)
	holdFlashCycles      = 4.0                    // number of blink cycles
	puffDuration         = 250 * time.Millisecond // puff-dismiss duration on early release
	puffScaleMax         = 1.6                    // ring grows to 1.6× radius during puff
	progressRingStroke   = 3.0                    // ring stroke width (dp)
	progressRingFallback = 24.0                   // fallback ring radius when no profile (dp)
)

// HoldButtonState holds mutable animation state for a HoldButton.
// Allocate with NewHoldButtonState and store in your Model.
type HoldButtonState struct {
	holding   bool
	progress  anim.Anim[float32] // 0→1 (fill) or 1→0 (release snap-back)
	flashAnim anim.Anim[float32] // 0→1 over ~400ms for completion blink
	flashing  bool               // true during post-completion flash
	ripple    RippleState
	completed bool // latched on completion, reset on next press

	// Puff-dismiss: ring expands + fades on early release.
	puffing      bool
	puffProgress float32            // frozen arc fraction at moment of release
	puffOpacity  anim.Anim[float32] // 1→0
	puffScale    anim.Anim[float32] // 1→puffScaleMax

	// Touch origin for the progress ring (absolute screen coords).
	ringCX, ringCY float32
	ringRadius     float32
}

// NewHoldButtonState creates a ready-to-use state.
func NewHoldButtonState() *HoldButtonState { return &HoldButtonState{} }

// Tick advances internal animations. Call from your update on TickMsg.
// Returns true if still animating.
func (s *HoldButtonState) Tick(dt time.Duration) bool {
	r := s.ripple.Tick(dt)
	p := s.progress.Tick(dt)
	f := s.flashAnim.Tick(dt)
	po := s.puffOpacity.Tick(dt)
	ps := s.puffScale.Tick(dt)

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

	// End of puff → clean up.
	if s.puffing && s.puffOpacity.IsDone() {
		s.puffing = false
		s.progress.SetImmediate(0)
	}

	return r || p || f || po || ps
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

	// Enforce MinTouchTarget for touch/HMI profiles (RFC-004 §2.5).
	if ctx.Profile != nil && ctx.Profile.MinTouchTarget > 0 {
		minT := int(ctx.Profile.MinTouchTarget)
		if w < minT {
			w = minT
		}
		if h < minT {
			h = minT
		}
	}

	buttonRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Derive ring radius from interaction profile (finger size).
	ringR := float32(progressRingFallback)
	if ctx.Profile != nil {
		ringR = float32(ctx.Profile.MinTouchTarget) / 2
	}

	// Register drag + release for press-and-hold detection.
	// onDrag fires continuously while held; onRelease fires on finger-up.
	hoverOpacity := ix.RegisterSurfaceDrag(buttonRect,
		func(x, y float32) {
			// On first drag call, start the hold.
			if !st.holding && !st.completed {
				st.holding = true
				st.ringCX = x
				st.ringCY = y
				st.ringRadius = ringR
				st.progress.SetImmediate(0)
				st.progress.SetTarget(1, holdDur, anim.Linear)
				st.ripple.Trigger(x, y, maxRippleRadius(x, y, buttonRect.X, buttonRect.Y, buttonRect.W, buttonRect.H))
			}
		},
		func(x, y float32) {
			// Release: if not completed, puff-dismiss.
			if st.holding {
				st.holding = false
				cur := st.progress.Value()
				if cur < 0.99 {
					st.puffing = true
					st.puffProgress = cur
					st.progress.SetImmediate(0) // stop normal ring immediately
					st.puffOpacity.SetImmediate(1)
					st.puffOpacity.SetTarget(0, puffDuration, anim.OutCubic)
					st.puffScale.SetImmediate(1)
					st.puffScale.SetTarget(puffScaleMax, puffDuration, anim.OutCubic)
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

	// Request continued frames while any animation is active.
	if st.holding || st.flashing || st.puffing || st.ripple.Active() || st.progress.Value() > 0.001 {
		ix.SetNeedsFrame()
	}

	// ── Radial progress ring (centred on touch point) ────────
	if st.puffing {
		// Puff-dismiss: frozen arc, expanding radius, fading opacity.
		puffR := st.ringRadius * st.puffScale.Value()
		drawProgressRing(canvas, st.ringCX, st.ringCY, puffR, st.puffProgress, st.puffOpacity.Value(), tokens.Colors.Accent.Primary)
	} else if st.flashing {
		flashT := st.flashAnim.Value()
		blinkOpacity := float32(math.Abs(math.Sin(float64(flashT) * holdFlashCycles * math.Pi)))
		drawProgressRing(canvas, st.ringCX, st.ringCY, st.ringRadius, 1.0, blinkOpacity, tokens.Colors.Accent.Primary)
	} else if progress := st.progress.Value(); progress > 0.001 {
		drawProgressRing(canvas, st.ringCX, st.ringCY, st.ringRadius, progress, 1.0, tokens.Colors.Accent.Primary)
	}

	// Total bounds include the ring outside the button.
	ringMargin := int(ringR + progressRingStroke + 1)
	return ui.Bounds{
		X: area.X - ringMargin, Y: area.Y - ringMargin,
		W: w + ringMargin*2, H: h + ringMargin*2,
		Baseline: ringMargin + ui.ButtonPadY + contentH,
	}
}

// drawProgressRing draws a clockwise circular arc from 12-o'clock centred at (cx, cy).
// r is the ring radius; opacity modulates alpha (0–1) for the completion flash blink.
func drawProgressRing(canvas draw.Canvas, cx, cy, r, progress, opacity float32, color draw.Color) {
	if progress <= 0 || opacity <= 0 {
		return
	}

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
