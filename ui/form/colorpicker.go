package form

import (
	"fmt"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// Layout constants for color picker.
const (
	colorSwatchSize   = 24
	colorPickerGap    = 8
	colorPickerW      = 200
	colorPaletteSize  = 28
	colorPaletteGap   = 4
	colorPaletteCols  = 4
	colorPaletteRows  = 4
	colorPalettePad   = 8
)

// colorPalette is the default 16-color palette.
var colorPalette = []draw.Color{
	{R: 0.93, G: 0.26, B: 0.21, A: 1}, // Red
	{R: 0.91, G: 0.12, B: 0.39, A: 1}, // Pink
	{R: 0.61, G: 0.15, B: 0.69, A: 1}, // Purple
	{R: 0.40, G: 0.23, B: 0.72, A: 1}, // Deep Purple
	{R: 0.25, G: 0.32, B: 0.71, A: 1}, // Indigo
	{R: 0.13, G: 0.59, B: 0.95, A: 1}, // Blue
	{R: 0.01, G: 0.66, B: 0.96, A: 1}, // Light Blue
	{R: 0.00, G: 0.74, B: 0.83, A: 1}, // Cyan
	{R: 0.00, G: 0.59, B: 0.53, A: 1}, // Teal
	{R: 0.30, G: 0.69, B: 0.31, A: 1}, // Green
	{R: 0.55, G: 0.76, B: 0.29, A: 1}, // Light Green
	{R: 0.80, G: 0.86, B: 0.22, A: 1}, // Lime
	{R: 1.00, G: 0.93, B: 0.23, A: 1}, // Yellow
	{R: 1.00, G: 0.76, B: 0.03, A: 1}, // Amber
	{R: 1.00, G: 0.60, B: 0.00, A: 1}, // Orange
	{R: 0.62, G: 0.62, B: 0.62, A: 1}, // Grey
}

// ColorPickerState holds the open/closed state for a ColorPicker dropdown.
type ColorPickerState struct {
	Open bool
}

// ColorPicker is a color selection widget with a swatch and dropdown palette.
type ColorPicker struct {
	ui.BaseElement
	Value    draw.Color
	OnChange func(draw.Color)
	State    *ColorPickerState
	Disabled bool
}

// ColorPickerOption configures a ColorPicker element.
type ColorPickerOption func(*ColorPicker)

// WithColorPickerState links the ColorPicker to state for dropdown behaviour.
func WithColorPickerState(s *ColorPickerState) ColorPickerOption {
	return func(e *ColorPicker) { e.State = s }
}

// WithOnColorChange sets the callback invoked when a color is chosen.
func WithOnColorChange(fn func(draw.Color)) ColorPickerOption {
	return func(e *ColorPicker) { e.OnChange = fn }
}

// WithColorPickerDisabled marks the ColorPicker as disabled.
func WithColorPickerDisabled() ColorPickerOption {
	return func(e *ColorPicker) { e.Disabled = true }
}

// NewColorPicker creates a color picker element.
func NewColorPicker(value draw.Color, opts ...ColorPickerOption) ui.Element {
	el := ColorPicker{Value: value}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// LayoutSelf implements ui.Layouter.
func (n ColorPicker) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	overlays := ctx.Overlays
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := colorPickerW
	if area.W < w {
		w = area.W
	}

	fieldRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Hit target: click toggles dropdown.
	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterHit(fieldRect, nil)
	} else {
		var clickFn func()
		if n.State != nil {
			state := n.State
			clickFn = func() { state.Open = !state.Open }
		}
		hoverOpacity = ix.RegisterHit(fieldRect, clickFn)
	}

	isOpen := n.State != nil && n.State.Open && !n.Disabled

	// Focus management.
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	// Border
	borderColor := tokens.Colors.Stroke.Border
	if n.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(fieldRect, tokens.Radii.Input, draw.SolidPaint(borderColor))

	// Fill with hover
	fillColor := tokens.Colors.Surface.Elevated
	if hoverOpacity > 0 {
		fillColor = ui.LerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
	}
	if n.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	// Color swatch
	swatchX := area.X + textFieldPadX
	swatchY := area.Y + (h-colorSwatchSize)/2
	swatchColor := n.Value
	if n.Disabled {
		swatchColor = ui.DisabledColor(swatchColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(swatchX), float32(swatchY), float32(colorSwatchSize), float32(colorSwatchSize)),
		3, draw.SolidPaint(swatchColor))

	// Hex text
	hexText := colorToHex(n.Value)
	textX := swatchX + colorSwatchSize + colorPickerGap
	textY := area.Y + textFieldPadY
	textColor := tokens.Colors.Text.Primary
	if n.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(hexText, draw.Pt(float32(textX), float32(textY)), style, textColor)

	// Focus glow.
	if focused || isOpen {
		ui.DrawFocusRing(canvas, fieldRect, tokens.Radii.Input, tokens)
	}

	// Dropdown palette overlay.
	if isOpen && overlays != nil {
		dropX := area.X
		dropY := area.Y + h
		paletteW := colorPaletteCols*(colorPaletteSize+colorPaletteGap) - colorPaletteGap + colorPalettePad*2
		paletteH := colorPaletteRows*(colorPaletteSize+colorPaletteGap) - colorPaletteGap + colorPalettePad*2
		state := n.State
		onChange := n.OnChange
		winW := overlays.WindowW
		winH := overlays.WindowH

		// Flip above if needed.
		if dropY+paletteH > winH && area.Y-paletteH >= 0 {
			dropY = area.Y - paletteH
		}

		overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				// Backdrop.
				ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), func() {
					if state != nil {
						state.Open = false
					}
				})

				// Palette background.
				palRect := draw.R(float32(dropX), float32(dropY), float32(paletteW), float32(paletteH))
				canvas.FillRoundRect(palRect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
				canvas.StrokeRoundRect(palRect, tokens.Radii.Input,
					draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

				for i, c := range colorPalette {
					col := i % colorPaletteCols
					row := i / colorPaletteCols
					cx := dropX + colorPalettePad + col*(colorPaletteSize+colorPaletteGap)
					cy := dropY + colorPalettePad + row*(colorPaletteSize+colorPaletteGap)
					cellRect := draw.R(float32(cx), float32(cy), float32(colorPaletteSize), float32(colorPaletteSize))

					color := c
					var cellClick func()
					if onChange != nil || state != nil {
						cellClick = func() {
							if onChange != nil {
								onChange(color)
							}
							if state != nil {
								state.Open = false
							}
						}
					}
					ho := ix.RegisterHit(cellRect, cellClick)
					canvas.FillRoundRect(cellRect, 4, draw.SolidPaint(color))
					if ho > 0 {
						canvas.StrokeRoundRect(cellRect, 4,
							draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Text.Primary), Width: 2})
					}
				}
			},
		})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func colorToHex(c draw.Color) string {
	r := int(c.R*255 + 0.5)
	g := int(c.G*255 + 0.5)
	b := int(c.B*255 + 0.5)
	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if b > 255 {
		b = 255
	}
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// TreeEqual implements ui.TreeEqualizer.
func (n ColorPicker) TreeEqual(other ui.Element) bool {
	nb, ok := other.(ColorPicker)
	return ok && n.Value == nb.Value
}

// ResolveChildren implements ui.ChildResolver. ColorPicker is a leaf.
func (n ColorPicker) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n ColorPicker) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleCombobox,
		Value:  colorToHex(n.Value),
		States: a11y.AccessStates{Disabled: n.Disabled},
	}, parentIdx, a11y.Rect{})
}
