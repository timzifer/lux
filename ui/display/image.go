package display

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ImageElement renders a loaded image at a specified or natural size.
type ImageElement struct {
	ui.BaseElement
	ImageID   draw.ImageID
	Width     float32 // dp; 0 = use natural width
	Height    float32 // dp; 0 = use natural height
	ScaleMode draw.ImageScaleMode
	Alt       string  // alt-text for accessibility
	Opacity   float32 // 0 = default (1.0)
}

// ImageOption configures an Image element.
type ImageOption func(*ImageElement)

// WithImageSize sets explicit width and height in dp.
func WithImageSize(w, h float32) ImageOption {
	return func(e *ImageElement) {
		e.Width = w
		e.Height = h
	}
}

// WithImageScaleMode sets the scale mode (Fit, Fill, Stretch).
func WithImageScaleMode(mode draw.ImageScaleMode) ImageOption {
	return func(e *ImageElement) { e.ScaleMode = mode }
}

// WithImageAlt sets the alt-text for accessibility.
func WithImageAlt(alt string) ImageOption {
	return func(e *ImageElement) { e.Alt = alt }
}

// WithImageOpacity sets the opacity (0.0–1.0).
func WithImageOpacity(opacity float32) ImageOption {
	return func(e *ImageElement) { e.Opacity = opacity }
}

// Image renders a loaded image. Use WithImageSize to set explicit dimensions.
// If no size is given, the element uses 0×0 (the caller should specify size).
func Image(id draw.ImageID, opts ...ImageOption) ui.Element {
	e := ImageElement{ImageID: id, Opacity: 1}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// LayoutSelf implements ui.Layouter.
func (n ImageElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	w := int(n.Width)
	h := int(n.Height)
	if w == 0 {
		w = ctx.Area.W
	}
	if h == 0 {
		h = ctx.Area.H
	}
	if w > ctx.Area.W {
		w = ctx.Area.W
	}
	if h > ctx.Area.H {
		h = ctx.Area.H
	}
	opacity := n.Opacity
	if opacity == 0 {
		opacity = 1
	}
	r := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h))
	ctx.Canvas.DrawImage(n.ImageID, r, draw.ImageOptions{Opacity: opacity})
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h, Baseline: h}
}

// TreeEqual implements ui.TreeEqualizer.
// Returns false because ImageElement has function-like options.
func (n ImageElement) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver. ImageElement is a leaf.
func (n ImageElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op for images.
func (n ImageElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}
