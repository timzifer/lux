package form

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Layout constants for spinner.
const (
	spinnerDefaultSize = 24
	spinnerDotCount    = 8
	spinnerDotRadius   = 2
)

// Spinner is a circular loading indicator.
type Spinner struct {
	ui.BaseElement
	Size  int     // overall diameter; 0 uses spinnerDefaultSize
	Phase float32 // 0.0-1.0, drives rotation position
}

// SpinnerOption configures a Spinner element.
type SpinnerOption func(*Spinner)

// WithSpinnerSize sets the spinner diameter.
func WithSpinnerSize(size int) SpinnerOption {
	return func(s *Spinner) { s.Size = size }
}

// NewSpinner creates a circular loading spinner.
// Phase (0.0-1.0) controls the rotation; derive it from app.TickMsg to animate.
func NewSpinner(phase float32, opts ...SpinnerOption) ui.Element {
	el := Spinner{Phase: phase}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// LayoutSelf implements ui.Layouter.
func (n Spinner) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	size := n.Size
	if size <= 0 {
		size = spinnerDefaultSize
	}

	cx := float32(area.X) + float32(size)/2
	cy := float32(area.Y) + float32(size)/2
	radius := float32(size)/2 - float32(spinnerDotRadius)

	phase := n.Phase
	if phase < 0 {
		phase = 0
	}
	if phase > 1 {
		phase -= float32(int(phase))
	}

	baseColor := tokens.Colors.Accent.Primary

	for i := 0; i < spinnerDotCount; i++ {
		angle := 2*math.Pi*float64(i)/float64(spinnerDotCount) - math.Pi/2
		dx := float32(math.Cos(angle)) * radius
		dy := float32(math.Sin(angle)) * radius

		// Compute opacity: the dot closest to the current phase is brightest.
		dotPhase := float32(i) / float32(spinnerDotCount)
		dist := phase - dotPhase
		if dist < 0 {
			dist = -dist
		}
		if dist > 0.5 {
			dist = 1 - dist
		}
		// Map distance to opacity: closest = 1.0, farthest = 0.15.
		opacity := 1.0 - dist*2
		if opacity < 0.15 {
			opacity = 0.15
		}

		dotColor := baseColor
		dotColor.A = opacity

		dotX := cx + dx - float32(spinnerDotRadius)
		dotY := cy + dy - float32(spinnerDotRadius)
		canvas.FillEllipse(
			draw.R(dotX, dotY, float32(spinnerDotRadius*2), float32(spinnerDotRadius*2)),
			draw.SolidPaint(dotColor))
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: size, H: size}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Spinner) TreeEqual(other ui.Element) bool {
	nb, ok := other.(Spinner)
	return ok && n.Size == nb.Size
}

// ResolveChildren implements ui.ChildResolver. Spinner is a leaf.
func (n Spinner) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Spinner) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleProgressBar,
		States: a11y.AccessStates{Busy: true, ReadOnly: true},
	}, parentIdx, a11y.Rect{})
}
