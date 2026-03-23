package ui

import "github.com/timzifer/lux/draw"

// NullCanvas wraps a real Canvas but discards all draw calls.
// Only MeasureText delegates to the real canvas so that Flex/Grid
// can perform a measurement pass without painting.
type NullCanvas struct {
	Delegate draw.Canvas
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
func (n nullCanvas) DrawText(string, draw.Point, draw.TextStyle, draw.Color)      {}
func (n nullCanvas) DrawTextLayout(draw.TextLayout, draw.Point, draw.Color)       {}
func (n nullCanvas) DrawImage(draw.ImageID, draw.Rect, draw.ImageOptions)                          {}
func (n nullCanvas) DrawImageScaled(draw.ImageID, draw.Rect, draw.ImageScaleMode, draw.ImageOptions) {}
func (n nullCanvas) DrawImageSlice(draw.ImageSlice, draw.Rect, draw.ImageOptions)                  {}
func (n nullCanvas) DrawTexture(draw.TextureID, draw.Rect)                        {}
func (n nullCanvas) DrawShadow(draw.Rect, draw.Shadow)                            {}
func (n nullCanvas) PushClip(draw.Rect)                                           {}
func (n nullCanvas) PushClipRoundRect(draw.Rect, float32)                         {}
func (n nullCanvas) PushClipPath(draw.Path)                                       {}
func (n nullCanvas) PopClip()                                                     {}
func (n nullCanvas) PushTransform(draw.Transform)                                 {}
func (n nullCanvas) PopTransform()                                                {}
func (n nullCanvas) PushOffset(float32, float32)                                  {}
func (n nullCanvas) PushScale(float32, float32)                                   {}
func (n nullCanvas) PushOpacity(float32)                                          {}
func (n nullCanvas) PopOpacity()                                                  {}
func (n nullCanvas) PushBlur(float32)                                             {}
func (n nullCanvas) PopBlur()                                                     {}
func (n nullCanvas) PushLayer(draw.LayerOptions)                                  {}
func (n nullCanvas) PopLayer()                                                    {}
func (n nullCanvas) Save()                                                       {}
func (n nullCanvas) Restore()                                                    {}

func (n NullCanvas) MeasureText(text string, style draw.TextStyle) draw.TextMetrics {
	return n.Delegate.MeasureText(text, style)
}

func (n NullCanvas) Bounds() draw.Rect { return n.Delegate.Bounds() }
func (n NullCanvas) DPR() float32      { return n.Delegate.DPR() }
