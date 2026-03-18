package ui

import "github.com/timzifer/lux/draw"

// nullCanvas wraps a real Canvas but discards all draw calls.
// Only MeasureText delegates to the real canvas so that Flex/Grid
// can perform a measurement pass without painting.
type nullCanvas struct {
	delegate draw.Canvas
}

func (n nullCanvas) FillRect(draw.Rect, draw.Paint)                              {}
func (n nullCanvas) FillRoundRect(draw.Rect, float32, draw.Paint)                {}
func (n nullCanvas) FillRoundRectCorners(draw.Rect, draw.CornerRadii, draw.Paint) {}
func (n nullCanvas) FillEllipse(draw.Rect, draw.Paint)                           {}
func (n nullCanvas) StrokeRect(draw.Rect, draw.Stroke)                           {}
func (n nullCanvas) StrokeRoundRect(draw.Rect, float32, draw.Stroke)             {}
func (n nullCanvas) StrokeRoundRectCorners(draw.Rect, draw.CornerRadii, draw.Stroke) {}
func (n nullCanvas) StrokeEllipse(draw.Rect, draw.Stroke)                        {}
func (n nullCanvas) StrokeLine(draw.Point, draw.Point, draw.Stroke)              {}
func (n nullCanvas) FillPath(draw.Path, draw.Paint)                              {}
func (n nullCanvas) StrokePath(draw.Path, draw.Stroke)                           {}
func (n nullCanvas) DrawText(string, draw.Point, draw.TextStyle, draw.Color)     {}
func (n nullCanvas) DrawImage(draw.ImageID, draw.Rect, draw.ImageOptions)        {}
func (n nullCanvas) DrawShadow(draw.Rect, draw.Shadow)                           {}
func (n nullCanvas) PushClip(draw.Rect)                                          {}
func (n nullCanvas) PopClip()                                                    {}
func (n nullCanvas) PushTransform(draw.Transform)                                {}
func (n nullCanvas) PopTransform()                                               {}
func (n nullCanvas) PushOffset(float32, float32)                                 {}
func (n nullCanvas) PushOpacity(float32)                                         {}
func (n nullCanvas) PopOpacity()                                                 {}
func (n nullCanvas) Save()                                                       {}
func (n nullCanvas) Restore()                                                    {}

func (n nullCanvas) MeasureText(text string, style draw.TextStyle) draw.TextMetrics {
	return n.delegate.MeasureText(text, style)
}

func (n nullCanvas) Bounds() draw.Rect { return n.delegate.Bounds() }
func (n nullCanvas) DPR() float32      { return n.delegate.DPR() }
