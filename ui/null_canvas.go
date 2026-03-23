package ui

import "github.com/timzifer/lux/draw"

// NullCanvas wraps a real Canvas but discards all draw calls.
// Only MeasureText delegates to the real canvas so that Flex/Grid
// can perform a measurement pass without painting.
type NullCanvas struct {
	Delegate draw.Canvas
}

func (n NullCanvas) FillRect(draw.Rect, draw.Paint)                              {}
func (n NullCanvas) FillRoundRect(draw.Rect, float32, draw.Paint)                {}
func (n NullCanvas) FillRoundRectCorners(draw.Rect, draw.CornerRadii, draw.Paint) {}
func (n NullCanvas) FillEllipse(draw.Rect, draw.Paint)                           {}
func (n NullCanvas) StrokeRect(draw.Rect, draw.Stroke)                           {}
func (n NullCanvas) StrokeRoundRect(draw.Rect, float32, draw.Stroke)             {}
func (n NullCanvas) StrokeRoundRectCorners(draw.Rect, draw.CornerRadii, draw.Stroke) {}
func (n NullCanvas) StrokeEllipse(draw.Rect, draw.Stroke)                        {}
func (n NullCanvas) StrokeLine(draw.Point, draw.Point, draw.Stroke)              {}
func (n NullCanvas) FillPath(draw.Path, draw.Paint)                              {}
func (n NullCanvas) StrokePath(draw.Path, draw.Stroke)                           {}
func (n NullCanvas) DrawText(string, draw.Point, draw.TextStyle, draw.Color)      {}
func (n NullCanvas) DrawTextLayout(draw.TextLayout, draw.Point, draw.Color)       {}
func (n NullCanvas) DrawImage(draw.ImageID, draw.Rect, draw.ImageOptions)         {}
func (n NullCanvas) DrawImageSlice(draw.ImageSlice, draw.Rect, draw.ImageOptions) {}
func (n NullCanvas) DrawTexture(draw.TextureID, draw.Rect)                        {}
func (n NullCanvas) DrawShadow(draw.Rect, draw.Shadow)                            {}
func (n NullCanvas) PushClip(draw.Rect)                                           {}
func (n NullCanvas) PushClipRoundRect(draw.Rect, float32)                         {}
func (n NullCanvas) PushClipPath(draw.Path)                                       {}
func (n NullCanvas) PopClip()                                                     {}
func (n NullCanvas) PushTransform(draw.Transform)                                 {}
func (n NullCanvas) PopTransform()                                                {}
func (n NullCanvas) PushOffset(float32, float32)                                  {}
func (n NullCanvas) PushScale(float32, float32)                                   {}
func (n NullCanvas) PushOpacity(float32)                                          {}
func (n NullCanvas) PopOpacity()                                                  {}
func (n NullCanvas) PushBlur(float32)                                             {}
func (n NullCanvas) PopBlur()                                                     {}
func (n NullCanvas) PushLayer(draw.LayerOptions)                                  {}
func (n NullCanvas) PopLayer()                                                    {}
func (n NullCanvas) Save()                                                       {}
func (n NullCanvas) Restore()                                                    {}

func (n NullCanvas) MeasureText(text string, style draw.TextStyle) draw.TextMetrics {
	return n.Delegate.MeasureText(text, style)
}

func (n NullCanvas) Bounds() draw.Rect { return n.Delegate.Bounds() }
func (n NullCanvas) DPR() float32      { return n.Delegate.DPR() }
