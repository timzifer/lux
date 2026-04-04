package form

import (
	"fmt"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Layout constants for drum picker.
const (
	drumPickerW       = 120
	drumItemH         = 32
	drumDefaultVisible = 5
)

// DrumItem represents a single selectable value in a DrumPicker (RFC-004 §6.4).
type DrumItem struct {
	Label string
	Value any
}

// DrumPicker is a scroll-wheel picker for discrete values (RFC-004 §6.4).
type DrumPicker struct {
	ui.BaseElement
	Items         []DrumItem
	SelectedIndex int
	VisibleCount  int // Must be odd. Default: 5.
	OnSelect      func(index int)
	Looping       bool
	Haptic        bool
	Disabled      bool
}

// DrumPickerOption configures a DrumPicker element.
type DrumPickerOption func(*DrumPicker)

// WithDrumPickerVisible sets the number of visible items.
func WithDrumPickerVisible(n int) DrumPickerOption {
	return func(d *DrumPicker) { d.VisibleCount = n }
}

// WithOnDrumSelect sets the selection callback.
func WithOnDrumSelect(fn func(int)) DrumPickerOption {
	return func(d *DrumPicker) { d.OnSelect = fn }
}

// WithDrumLooping enables cyclic scrolling.
func WithDrumLooping() DrumPickerOption {
	return func(d *DrumPicker) { d.Looping = true }
}

// WithDrumHaptic enables haptic feedback on raster snap.
func WithDrumHaptic() DrumPickerOption {
	return func(d *DrumPicker) { d.Haptic = true }
}

// WithDrumDisabled disables the picker.
func WithDrumDisabled() DrumPickerOption {
	return func(d *DrumPicker) { d.Disabled = true }
}

// NewDrumPicker creates a drum picker element.
func NewDrumPicker(items []DrumItem, selectedIndex int, opts ...DrumPickerOption) ui.Element {
	el := DrumPicker{Items: items, SelectedIndex: selectedIndex, VisibleCount: drumDefaultVisible}
	for _, o := range opts {
		o(&el)
	}
	// Ensure odd visible count.
	if el.VisibleCount%2 == 0 {
		el.VisibleCount++
	}
	if el.VisibleCount < 3 {
		el.VisibleCount = 3
	}
	return el
}

// WrapIndex wraps an index for looping mode, or clamps for non-looping.
func (d DrumPicker) WrapIndex(idx int) int {
	n := len(d.Items)
	if n == 0 {
		return 0
	}
	if d.Looping {
		return ((idx % n) + n) % n
	}
	if idx < 0 {
		return 0
	}
	if idx >= n {
		return n - 1
	}
	return idx
}

// LayoutSelf implements ui.Layouter.
func (d DrumPicker) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	n := len(d.Items)
	vis := d.VisibleCount
	if vis < 3 {
		vis = drumDefaultVisible
	}

	w := drumPickerW
	if area.W < w {
		w = area.W
	}
	totalH := vis * drumItemH

	// Focus management.
	var focused bool
	if focus != nil && !d.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	outerRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(totalH))

	// Background.
	canvas.FillRoundRect(outerRect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(outerRect, tokens.Radii.Input, draw.Stroke{
		Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1,
	})

	// Selected item highlight band.
	half := vis / 2
	selBandY := area.Y + half*drumItemH
	selBandRect := draw.R(float32(area.X+1), float32(selBandY), float32(max(w-2, 0)), float32(drumItemH))
	canvas.FillRect(selBandRect, draw.SolidPaint(draw.Color{
		R: tokens.Colors.Accent.Primary.R,
		G: tokens.Colors.Accent.Primary.G,
		B: tokens.Colors.Accent.Primary.B,
		A: 0.15,
	}))

	// Draw visible items.
	for i := 0; i < vis; i++ {
		idx := d.SelectedIndex - half + i
		if n == 0 {
			continue
		}

		var itemLabel string
		var validIdx bool
		if d.Looping {
			wrapped := ((idx % n) + n) % n
			itemLabel = d.Items[wrapped].Label
			validIdx = true
		} else if idx >= 0 && idx < n {
			itemLabel = d.Items[idx].Label
			validIdx = true
		}

		if !validIdx {
			continue
		}

		itemY := area.Y + i*drumItemH
		itemRect := draw.R(float32(area.X), float32(itemY), float32(w), float32(drumItemH))

		// Click to select this item.
		if !d.Disabled && d.OnSelect != nil {
			targetIdx := idx
			if d.Looping {
				targetIdx = ((idx % n) + n) % n
			}
			onSelect := d.OnSelect
			ti := targetIdx
			ho := ix.RegisterHit(itemRect, func() { onSelect(ti) })
			if ho > 0 && i != half {
				canvas.FillRect(itemRect, draw.SolidPaint(draw.Color{A: ho * 0.05}))
			}
		}

		// Text styling: selected item is full opacity, others dimmed.
		textColor := tokens.Colors.Text.Primary
		if d.Disabled {
			textColor = tokens.Colors.Text.Disabled
		} else if i != half {
			textColor = tokens.Colors.Text.Secondary
		}

		m := canvas.MeasureText(itemLabel, style)
		textX := float32(area.X) + float32(w)/2 - m.Width/2
		textY := float32(itemY) + float32(drumItemH)/2 - style.Size/2
		canvas.DrawText(itemLabel, draw.Pt(textX, textY), style, textColor)
	}

	// Scroll up/down via drag.
	if !d.Disabled && d.OnSelect != nil && n > 0 {
		onSelect := d.OnSelect
		sel := d.SelectedIndex
		looping := d.Looping
		count := n
		pressY := float32(-1)
		ix.RegisterDrag(outerRect, func(_, y float32) {
			if pressY < 0 {
				pressY = y
				return
			}
			delta := int((pressY - y) / float32(drumItemH))
			if delta == 0 {
				return
			}
			newIdx := sel + delta
			if looping {
				newIdx = ((newIdx % count) + count) % count
			} else {
				if newIdx < 0 {
					newIdx = 0
				}
				if newIdx >= count {
					newIdx = count - 1
				}
			}
			if newIdx != sel {
				onSelect(newIdx)
			}
		})
	}

	if focused {
		ui.DrawFocusRing(canvas, outerRect, tokens.Radii.Input, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: totalH}
}

// TreeEqual implements ui.TreeEqualizer.
func (d DrumPicker) TreeEqual(other ui.Element) bool {
	db, ok := other.(DrumPicker)
	if !ok {
		return false
	}
	if d.SelectedIndex != db.SelectedIndex || len(d.Items) != len(db.Items) {
		return false
	}
	for i := range d.Items {
		if d.Items[i].Label != db.Items[i].Label {
			return false
		}
	}
	return true
}

// ResolveChildren implements ui.ChildResolver. DrumPicker is a leaf.
func (d DrumPicker) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return d
}

// WalkAccess implements ui.AccessWalker.
func (d DrumPicker) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	label := ""
	if d.SelectedIndex >= 0 && d.SelectedIndex < len(d.Items) {
		label = d.Items[d.SelectedIndex].Label
	}
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleListbox,
		Value:  label,
		States: a11y.AccessStates{Disabled: d.Disabled},
		NumericValue: &a11y.AccessNumericValue{
			Current: float64(d.SelectedIndex),
			Min:     0,
			Max:     float64(max(len(d.Items)-1, 0)),
			Step:    1,
		},
	}, parentIdx, a11y.Rect{})
}

// IntItems is a convenience function that creates DrumItem slices from a range of integers.
func IntItems(from, to int) []DrumItem {
	items := make([]DrumItem, 0, abs(to-from)+1)
	step := 1
	if to < from {
		step = -1
	}
	for i := from; ; i += step {
		items = append(items, DrumItem{Label: fmt.Sprintf("%d", i), Value: i})
		if i == to {
			break
		}
	}
	return items
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
