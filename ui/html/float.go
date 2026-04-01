package html

import (
	"github.com/timzifer/lux/ui"
)

// FloatSide indicates whether an element floats left, right, or not at all.
type FloatSide int

const (
	FloatNone  FloatSide = iota
	FloatLeft            // CSS float: left
	FloatRight           // CSS float: right
)

// ClearSide indicates which floats an element should clear.
type ClearSide int

const (
	ClearNone  ClearSide = iota
	ClearLeft            // CSS clear: left
	ClearRight           // CSS clear: right
	ClearBoth            // CSS clear: both
)

// FloatChild wraps an element with its float/clear metadata.
type FloatChild struct {
	Element ui.Element
	Float   FloatSide
	Clear   ClearSide
}

// FloatLayout arranges children according to CSS float semantics.
// Left-floated children stack from the left, right-floated from the
// right. Non-floated (normal flow) children take the full width
// below the current float context. clear: left/right/both causes
// the element to be placed below the relevant floats.
type FloatLayout struct {
	ui.BaseElement
	Children      []FloatChild
	ContainFloats bool // true if this container establishes a BFC (e.g. floated parent, overflow:hidden)
}

// floatRect tracks the position and size of a placed float.
type floatRect struct {
	x, y, w, h int
	side        FloatSide
}

// findLeftFloatPos computes the X position and available width for a
// left float at the given Y position, accounting for existing floats.
// Only floats that vertically overlap floatY are considered.
func findLeftFloatPos(area ui.Bounds, floatY int, leftFloats, rightFloats []floatRect) (floatX, availW int) {
	floatX = area.X
	for _, lf := range leftFloats {
		if lf.y <= floatY && lf.y+lf.h > floatY {
			if lf.x+lf.w > floatX {
				floatX = lf.x + lf.w
			}
		}
	}
	availW = area.X + area.W - floatX
	for _, rf := range rightFloats {
		if rf.y <= floatY && rf.y+rf.h > floatY {
			rEdge := rf.x
			if rAvail := rEdge - floatX; rAvail < availW {
				availW = rAvail
			}
		}
	}
	if availW < 0 {
		availW = 0
	}
	return
}

// LayoutSelf implements ui.Layouter.
func (n FloatLayout) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area

	// Track the bottom edges of left and right float columns.
	// Float tracking state is maintained in leftFloats/rightFloats slices below.

	// Simple float layout: process children in order.
	var leftFloats, rightFloats []floatRect

	// bottomOf returns the max bottom Y of the given float rects.
	bottomOf := func(rects []floatRect) int {
		maxY := area.Y
		for _, r := range rects {
			if bot := r.y + r.h; bot > maxY {
				maxY = bot
			}
		}
		return maxY
	}

	// cursorY tracks where normal flow content goes.
	cursorY := area.Y
	// floatY tracks the minimum Y for the next float (CSS: a float
	// cannot be placed above a previous float in document order).
	floatTopY := area.Y
	maxW := 0
	maxBottom := area.Y

	for _, child := range n.Children {
		if child.Element == nil {
			continue
		}

		// Apply clear.
		switch child.Clear {
		case ClearLeft:
			bot := bottomOf(leftFloats)
			if bot > cursorY {
				cursorY = bot
			}
		case ClearRight:
			bot := bottomOf(rightFloats)
			if bot > cursorY {
				cursorY = bot
			}
		case ClearBoth:
			botL := bottomOf(leftFloats)
			botR := bottomOf(rightFloats)
			bot := botL
			if botR > bot {
				bot = botR
			}
			if bot > cursorY {
				cursorY = bot
			}
		}

		switch child.Float {
		case FloatLeft:
			// CSS: a float cannot be placed above a previous float.
			floatY := floatTopY
			if cursorY > floatY {
				floatY = cursorY
			}

			// Find placement: try current Y, drop down if no room.
			floatX, availW := findLeftFloatPos(area, floatY, leftFloats, rightFloats)

			// Measure to check if wrapping is needed.
			mb := ctx.MeasureChild(child.Element, ui.Bounds{
				X: floatX, Y: floatY, W: max(availW, 1), H: area.H,
			})

			// If doesn't fit and there are floats blocking, drop below them.
			for mb.W > availW && floatX > area.X {
				// Find bottom of the lowest blocking float at this Y.
				nextY := floatY + 1
				for _, lf := range leftFloats {
					if lf.y <= floatY && lf.y+lf.h > floatY {
						if bot := lf.y + lf.h; bot > nextY {
							nextY = bot
						}
					}
				}
				for _, rf := range rightFloats {
					if rf.y <= floatY && rf.y+rf.h > floatY {
						if bot := rf.y + rf.h; bot > nextY {
							nextY = bot
						}
					}
				}
				floatY = nextY
				floatX, availW = findLeftFloatPos(area, floatY, leftFloats, rightFloats)
				mb = ctx.MeasureChild(child.Element, ui.Bounds{
					X: floatX, Y: floatY, W: max(availW, 1), H: area.H,
				})
				// Safety: if we've dropped past all floats, break.
				if floatX == area.X && availW == area.W {
					break
				}
			}

			cb := ctx.LayoutChild(child.Element, ui.Bounds{
				X: floatX, Y: floatY, W: max(availW, 1), H: area.H,
			})

			leftFloats = append(leftFloats, floatRect{
				x: floatX, y: floatY, w: cb.W, h: cb.H, side: FloatLeft,
			})
			floatTopY = floatY // next float can't go above this one

			// Floated elements do NOT expand the container's height
			// (CSS: floats are out of normal flow). Only track for
			// maxW to ensure the container is wide enough.
			if floatX+cb.W-area.X > maxW {
				maxW = floatX + cb.W - area.X
			}

		case FloatRight:
			floatY := floatTopY
			if cursorY > floatY {
				floatY = cursorY
			}

			// Find available right position.
			floatRightEdge := area.X + area.W
			for _, rf := range rightFloats {
				if rf.y+rf.h > floatY && rf.x < floatRightEdge {
					floatRightEdge = rf.x
				}
			}

			// Compute available width between left floats and right edge.
			availW := floatRightEdge - area.X
			for _, lf := range leftFloats {
				if lf.y+lf.h > floatY {
					leftEdge := lf.x + lf.w
					if leftAvail := floatRightEdge - leftEdge; leftAvail < availW {
						availW = leftAvail
					}
				}
			}
			if availW < 0 {
				availW = 0
			}

			// Layout once, positioned at the right edge. The child's
			// StyledBox uses the available width to compute its size,
			// and we anchor it to the right edge of the container.
			floatX := floatRightEdge - availW
			cb := ctx.LayoutChild(child.Element, ui.Bounds{
				X: floatX, Y: floatY, W: availW, H: area.H,
			})

			// Record the actual position based on the child's rendered width.
			actualX := floatRightEdge - cb.W
			rightFloats = append(rightFloats, floatRect{
				x: actualX, y: floatY, w: cb.W, h: cb.H, side: FloatRight,
			})

			// Floated elements do NOT expand the container's height.
			floatTopY = floatY
			if area.X+area.W-floatX > maxW {
				maxW = area.X + area.W - floatX
			}

		default:
			// Normal flow: full width below floats.
			// The element goes below all current floats.
			flowY := cursorY
			botL := bottomOf(leftFloats)
			botR := bottomOf(rightFloats)
			if botL > flowY {
				flowY = botL
			}
			if botR > flowY {
				flowY = botR
			}

			cb := ctx.LayoutChild(child.Element, ui.Bounds{
				X: area.X, Y: flowY, W: area.W, H: area.H,
			})

			cursorY = flowY + cb.H
			if cb.W > maxW {
				maxW = cb.W
			}
			if cursorY > maxBottom {
				maxBottom = cursorY
			}
		}
	}

	// If this container establishes a BFC (ContainFloats), expand to
	// contain all floated children. Otherwise, only normal-flow children
	// contribute to the height (CSS: floats are out of normal flow).
	if n.ContainFloats {
		botL := bottomOf(leftFloats)
		botR := bottomOf(rightFloats)
		if botL > maxBottom {
			maxBottom = botL
		}
		if botR > maxBottom {
			maxBottom = botR
		}
	}

	totalH := maxBottom - area.Y
	if maxW == 0 {
		maxW = area.W
	}

	return ui.Bounds{
		X: area.X,
		Y: area.Y,
		W: maxW,
		H: totalH,
	}
}

// TreeEqual implements ui.TreeEqualizer.
func (n FloatLayout) TreeEqual(other ui.Element) bool {
	o, ok := other.(FloatLayout)
	if !ok || len(n.Children) != len(o.Children) || n.ContainFloats != o.ContainFloats {
		return false
	}
	for i := range n.Children {
		if n.Children[i].Float != o.Children[i].Float ||
			n.Children[i].Clear != o.Children[i].Clear {
			return false
		}
	}
	return true
}

// ResolveChildren implements ui.ChildResolver.
func (n FloatLayout) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	out.Children = make([]FloatChild, len(n.Children))
	for i, c := range n.Children {
		out.Children[i] = c
		if c.Element != nil {
			out.Children[i].Element = resolve(c.Element, i)
		}
	}
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n FloatLayout) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, c := range n.Children {
		if c.Element != nil {
			b.Walk(c.Element, parentIdx)
		}
	}
}
