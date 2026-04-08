package osk

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// sendFunc is the function used to inject messages into the app loop.
// Set by the framework during integration (see SetSendFunc).
var sendFunc func(any)

// SetSendFunc sets the function used to inject messages into the app loop.
// Called by app.runInternal during initialization.
func SetSendFunc(fn func(any)) { sendFunc = fn }

func send(msg any) {
	if sendFunc != nil {
		sendFunc(msg)
	}
}

// OSKElement is the on-screen keyboard rendered as a framework overlay.
// It implements ui.Layouter and is created by the framework — not by user code.
type OSKElement struct {
	ui.BaseElement
	State  *OSKState
	ScreenW int
	ScreenH int
}

// NewOSKElement creates an OSK element for the given state and screen dimensions.
func NewOSKElement(state *OSKState, screenW, screenH int) ui.Element {
	if state == nil || !state.Visible {
		return ui.Empty()
	}
	return OSKElement{State: state, ScreenW: screenW, ScreenH: screenH}
}

// LayoutSelf renders the on-screen keyboard and registers hit targets (RFC-004 §5.5).
func (el OSKElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	state := el.State
	if state == nil || !state.Visible {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	th := ctx.Theme

	dpr := canvas.DPR()
	_, keyH, gap := ComputeKeySize(el.ScreenW, el.ScreenH, dpr, state.Mode)

	rows := RowsForState(state)
	oskH := float32(len(rows))*(keyH+gap) + gap

	// Position the keyboard at the bottom of the screen.
	oskY := float32(el.ScreenH) - oskH
	oskX := float32(0)
	oskW := float32(el.ScreenW)

	// Check for custom DrawFunc (RFC-004 §5.5: theme renders the OSK).
	if df := th.DrawFunc(theme.WidgetKindOSK); df != nil {
		df(theme.DrawCtx{
			Canvas: canvas,
			Bounds: draw.R(oskX, oskY, oskW, oskH),
			DPR:    dpr,
		}, tokens, state)
		return ui.Bounds{X: int(oskX), Y: int(oskY), W: int(oskW), H: int(oskH)}
	}

	// Default rendering.
	bgColor := tokens.Colors.Surface.Elevated
	bgColor.A = 0.97
	canvas.FillRect(draw.R(oskX, oskY, oskW, oskH), draw.SolidPaint(bgColor))

	// Divider line at the top of the OSK.
	canvas.FillRect(draw.R(oskX, oskY, oskW, 1), draw.SolidPaint(tokens.Colors.Stroke.Divider))

	// Render keys using button-variant styling.
	RenderButtonKeyboard(canvas, tokens, ix, state, el.ScreenW, el.ScreenH, dpr, oskX, oskY, oskW)

	return ui.Bounds{X: int(oskX), Y: int(oskY), W: int(oskW), H: int(oskH)}
}

// keyAction returns a click handler for the given OSK key.
func keyAction(key OSKKey, state *OSKState) func() {
	switch key.Action {
	case OSKActionChar:
		ch := key.Char
		if ch == 0 {
			return func() {} // empty/spacer key
		}
		return func() {
			send(input.CharMsg{Char: ch})
		}
	case OSKActionSpace:
		return func() {
			send(input.CharMsg{Char: ' '})
		}
	case OSKActionBackspace:
		return func() {
			send(input.KeyMsg{Key: input.KeyBackspace, Action: input.KeyPress})
		}
	case OSKActionEnter:
		return func() {
			send(input.KeyMsg{Key: input.KeyEnter, Action: input.KeyPress})
		}
	case OSKActionTab:
		return func() {
			send(input.KeyMsg{Key: input.KeyTab, Action: input.KeyPress})
		}
	case OSKActionShift:
		return func() {
			send(OSKToggleShiftMsg{})
		}
	case OSKActionSwitch:
		return func() {
			send(OSKSwitchLayerMsg{})
		}
	case OSKActionDismiss:
		return func() {
			send(OSKDismissMsg{})
		}
	case OSKActionSign:
		return func() {
			send(OSKSignMsg{})
		}
	case OSKActionDecimal:
		return func() {
			send(input.CharMsg{Char: '.'})
		}
	default:
		return func() {}
	}
}

// ── Internal OSK Messages ───────────────────────────────────────

// OSKToggleShiftMsg toggles the shift state.
type OSKToggleShiftMsg struct{}

// OSKSwitchLayerMsg switches between alpha and numeric/symbol layer.
type OSKSwitchLayerMsg struct{}

// OSKDismissMsg closes the OSK.
type OSKDismissMsg struct{}

// OSKSignMsg toggles the +/- sign on numeric input.
type OSKSignMsg struct{}

// ── Element interface ───────────────────────────────────────────

func (el OSKElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(OSKElement)
	if !ok {
		return false
	}
	if el.State == nil && o.State == nil {
		return true
	}
	if el.State == nil || o.State == nil {
		return false
	}
	return el.State.Visible == o.State.Visible &&
		el.State.Layout == o.State.Layout &&
		el.State.Mode == o.State.Mode &&
		el.State.Shifted == o.State.Shifted &&
		el.ScreenW == o.ScreenW &&
		el.ScreenH == o.ScreenH
}

func (el OSKElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return el
}

func (el OSKElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleGroup,
		Label: "On-Screen Keyboard",
	}, parentIdx, a11y.Rect{})
}

