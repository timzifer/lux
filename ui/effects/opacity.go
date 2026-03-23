package effects

import (
	"github.com/timzifer/lux/ui"
)

// ── OpacityBoxElement ───────────────────────────────────────────────

// OpacityBoxElement applies a uniform opacity to all child content.
type OpacityBoxElement struct {
	ui.BaseElement
	Alpha float32
	Child ui.Element
}

func (n OpacityBoxElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	ctx.Canvas.PushOpacity(n.Alpha)
	b := ctx.LayoutChild(n.Child, ctx.Area)
	ctx.Canvas.PopOpacity()
	return b
}

func (n OpacityBoxElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(OpacityBoxElement)
	return ok && n.Alpha == o.Alpha
}

func (n OpacityBoxElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	n.Child = resolve(n.Child, 0)
	return n
}

func (n OpacityBoxElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// OpacityBox applies a uniform opacity to all child content.
func OpacityBox(alpha float32, child ui.Element) ui.Element {
	return OpacityBoxElement{Alpha: alpha, Child: child}
}
