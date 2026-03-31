package display

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// SpanStyle overrides text style and color for a Span within a Paragraph.
// Zero values inherit from the theme's Body style.
type SpanStyle struct {
	Style draw.TextStyle // zero = inherit from theme Body
	Color draw.Color     // zero = use theme Text.Primary
}

// Span is a styled run of text within a Paragraph.
type Span struct {
	Text  string
	Style SpanStyle
}

// ── Inline Widgets (RFC-003 §5.5) ──────────────────────────────

// ParagraphContent is the sealed interface for items that can appear
// in a RichParagraph's Content slice: text spans, inline widgets, and
// inline/float/block images.
type ParagraphContent interface{ isParagraphContent() }

func (Span) isParagraphContent()         {}
func (InlineWidget) isParagraphContent() {}
func (ImageSpan) isParagraphContent()    {}

// InlineWidget embeds an arbitrary Element in the text flow.
// Width and height are determined by the element itself via intrinsic
// measurement (MeasureChild). Baseline alignment: the bottom edge of
// the widget sits on the text baseline, shifted up by Baseline dp.
type InlineWidget struct {
	Element  ui.Element
	Baseline float32 // 0 = bottom on text baseline; positive = shift up
}

// InlineElement creates an InlineWidget with default baseline alignment.
func InlineElement(el ui.Element) InlineWidget {
	return InlineWidget{Element: el}
}

// InlineElementWithBaseline creates an InlineWidget with a custom baseline offset.
func InlineElementWithBaseline(el ui.Element, baseline float32) InlineWidget {
	return InlineWidget{Element: el, Baseline: baseline}
}

// ── Images (HTML §4.8.3) ────────────────────────────────────────

// ImageFloat controls how an ImageSpan is positioned relative to the
// surrounding text flow, following HTML float semantics.
type ImageFloat uint8

const (
	// ImageFloatNone places the image inline in the text flow (default).
	// The image behaves like an InlineWidget with baseline alignment.
	ImageFloatNone ImageFloat = iota

	// ImageFloatLeft places the image at the left paragraph margin.
	// Subsequent inline content flows on the right side of the image.
	ImageFloatLeft

	// ImageFloatRight places the image at the right paragraph margin.
	// Subsequent inline content flows on the left side of the image.
	ImageFloatRight

	// ImageFloatBlock renders the image as a full-width block element,
	// breaking the text flow before and after it.
	ImageFloatBlock
)

// ImageSpan embeds an image into a RichParagraph's content.
//
// Placement behaviour by Float value:
//   - ImageFloatNone  – inline in text, respects Baseline like InlineWidget.
//   - ImageFloatLeft  – floated to the left margin; text wraps on the right.
//   - ImageFloatRight – floated to the right margin; text wraps on the left.
//   - ImageFloatBlock – occupies the full paragraph width on its own line.
//
// Size rules (Width/Height in dp):
//   - Both zero:  defaults to current line height × line height (square).
//   - Only Width: Height = Width (square).
//   - Only Height: Width = Height (square).
//   - Both set:   used as-is.
//
// Opacity 0 is treated as fully opaque (1.0).
type ImageSpan struct {
	ImageID   draw.ImageID
	Alt       string              // accessibility label (like HTML alt="")
	Width     float32             // dp; see size rules above
	Height    float32             // dp; see size rules above
	ScaleMode draw.ImageScaleMode // default ImageScaleStretch
	Opacity   float32             // 0 = 1.0
	Float     ImageFloat
	Baseline  float32 // ImageFloatNone only: shift bottom edge up from text baseline
}

// ImageSpanOption is a functional option for ImageSpan constructors.
type ImageSpanOption func(*ImageSpan)

// WithImageSpanSize sets explicit width and height in dp.
func WithImageSpanSize(w, h float32) ImageSpanOption {
	return func(s *ImageSpan) { s.Width = w; s.Height = h }
}

// WithImageSpanScaleMode sets the scale mode (Fit, Fill, Stretch).
func WithImageSpanScaleMode(m draw.ImageScaleMode) ImageSpanOption {
	return func(s *ImageSpan) { s.ScaleMode = m }
}

// WithImageSpanAlt sets the accessibility alt text.
func WithImageSpanAlt(alt string) ImageSpanOption {
	return func(s *ImageSpan) { s.Alt = alt }
}

// WithImageSpanOpacity sets the image opacity (0.0–1.0; 0 = fully opaque).
func WithImageSpanOpacity(op float32) ImageSpanOption {
	return func(s *ImageSpan) { s.Opacity = op }
}

// WithImageSpanBaseline sets the baseline shift for inline images.
func WithImageSpanBaseline(b float32) ImageSpanOption {
	return func(s *ImageSpan) { s.Baseline = b }
}

// InlineImage creates an ImageSpan placed inline in the text flow.
func InlineImage(id draw.ImageID, opts ...ImageSpanOption) ImageSpan {
	s := ImageSpan{ImageID: id, Float: ImageFloatNone}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// FloatLeftImage creates an ImageSpan floated to the left margin.
func FloatLeftImage(id draw.ImageID, opts ...ImageSpanOption) ImageSpan {
	s := ImageSpan{ImageID: id, Float: ImageFloatLeft}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// FloatRightImage creates an ImageSpan floated to the right margin.
func FloatRightImage(id draw.ImageID, opts ...ImageSpanOption) ImageSpan {
	s := ImageSpan{ImageID: id, Float: ImageFloatRight}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// BlockImage creates an ImageSpan rendered as a full-width block element.
func BlockImage(id draw.ImageID, opts ...ImageSpanOption) ImageSpan {
	s := ImageSpan{ImageID: id, Float: ImageFloatBlock}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// resolveImageSize returns the effective dp size for an ImageSpan, falling
// back to a square of lineH when both Width and Height are zero.
func resolveImageSize(img ImageSpan, lineH float32) (w, h float32) {
	w, h = img.Width, img.Height
	if w == 0 && h == 0 {
		w, h = lineH, lineH
	} else if h == 0 {
		h = w
	} else if w == 0 {
		w = h
	}
	return
}

// imageOpacity returns the effective opacity, treating 0 as 1.0.
func imageOpacity(op float32) float32 {
	if op == 0 {
		return 1.0
	}
	return op
}

// ── RichParagraph & RichTextElement ─────────────────────────────

// RichParagraph is a block-level text unit containing styled spans,
// inline widgets, and/or images.
type RichParagraph struct {
	Spans   []Span             // legacy text-only content
	Content []ParagraphContent // mixed content (takes precedence over Spans)
}

// RichTextElement renders read-only rich-formatted text.
type RichTextElement struct {
	ui.BaseElement
	Paragraphs []RichParagraph
}

// RichText creates a read-only rich-formatted text element.
func RichText(paragraphs ...RichParagraph) ui.Element {
	return RichTextElement{Paragraphs: paragraphs}
}

// RichTextSpans is a convenience for single-paragraph rich text.
func RichTextSpans(spans ...Span) ui.Element {
	return RichTextElement{Paragraphs: []RichParagraph{{Spans: spans}}}
}

// RichTextContent is a convenience constructor for a single paragraph with
// mixed content (spans, inline widgets, images).
func RichTextContent(items ...ParagraphContent) ui.Element {
	return RichTextElement{Paragraphs: []RichParagraph{{Content: items}}}
}

// paragraphContent returns the effective content slice for a paragraph,
// preferring Content if non-empty, otherwise converting Spans.
func paragraphContent(p RichParagraph) []ParagraphContent {
	if len(p.Content) > 0 {
		return p.Content
	}
	out := make([]ParagraphContent, len(p.Spans))
	for i, s := range p.Spans {
		out[i] = s
	}
	return out
}

// ── Layout ──────────────────────────────────────────────────────

// lineItem holds measured data for a single content item on a line.
type lineItem struct {
	content   ParagraphContent
	w, h      int     // measured width and height
	ascent    int     // text ascent (0 for widgets/images)
	baseline  float32 // InlineWidget/ImageSpan baseline offset
	style     draw.TextStyle
	color     draw.Color
	imageID   draw.ImageID        // non-zero for ImageSpan items
	scaleMode draw.ImageScaleMode // for ImageSpan items
	opacity   float32             // for ImageSpan items (already resolved, never 0)
}

// LayoutSelf implements ui.Layouter.
func (n RichTextElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if len(n.Paragraphs) == 0 {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	bodyStyle := ctx.Tokens.Typography.Body
	primaryColor := ctx.Tokens.Colors.Text.Primary
	paraSpacing := int(ctx.Tokens.Spacing.S)
	if paraSpacing <= 0 {
		paraSpacing = 8
	}

	defaultLineH := int(bodyStyle.Size)
	if defaultLineH <= 0 {
		defaultLineH = 14
	}

	cursorY := ctx.Area.Y
	maxW := 0

	for pIdx, para := range n.Paragraphs {
		allContent := paragraphContent(para)
		if len(allContent) == 0 {
			if pIdx < len(n.Paragraphs)-1 {
				cursorY += paraSpacing
			}
			continue
		}

		// ── Pre-pass: separate float/block images from inline content ──
		var inlineContent []ParagraphContent
		var floatLeftImgs, floatRightImgs, blockImgs []ImageSpan
		for _, c := range allContent {
			if img, ok := c.(ImageSpan); ok {
				switch img.Float {
				case ImageFloatLeft:
					floatLeftImgs = append(floatLeftImgs, img)
				case ImageFloatRight:
					floatRightImgs = append(floatRightImgs, img)
				case ImageFloatBlock:
					blockImgs = append(blockImgs, img)
				default:
					inlineContent = append(inlineContent, c)
				}
			} else {
				inlineContent = append(inlineContent, c)
			}
		}

		paraStartY := cursorY

		// ── Place float-left images (stacked vertically at left margin) ──
		floatLeftW, floatLeftH := 0, 0
		for _, img := range floatLeftImgs {
			w, h := resolveImageSize(img, float32(defaultLineH))
			iw, ih := int(math.Ceil(float64(w))), int(math.Ceil(float64(h)))
			r := draw.R(float32(ctx.Area.X+floatLeftW), float32(cursorY+floatLeftH), w, h)
			ctx.Canvas.DrawImageScaled(img.ImageID, r, img.ScaleMode, draw.ImageOptions{Opacity: imageOpacity(img.Opacity)})
			if iw > floatLeftW {
				floatLeftW = iw
			}
			floatLeftH += ih
		}

		// ── Place float-right images (stacked vertically at right margin) ──
		floatRightW, floatRightH := 0, 0
		for _, img := range floatRightImgs {
			w, h := resolveImageSize(img, float32(defaultLineH))
			iw, ih := int(math.Ceil(float64(w))), int(math.Ceil(float64(h)))
			r := draw.R(float32(ctx.Area.X+ctx.Area.W-floatRightW-iw), float32(cursorY+floatRightH), w, h)
			ctx.Canvas.DrawImageScaled(img.ImageID, r, img.ScaleMode, draw.ImageOptions{Opacity: imageOpacity(img.Opacity)})
			if iw > floatRightW {
				floatRightW = iw
			}
			floatRightH += ih
		}

		// ── Measure all inline items in this paragraph ──
		inlineX := ctx.Area.X + floatLeftW
		availW := ctx.Area.W - floatLeftW - floatRightW
		if availW < 0 {
			availW = 0
		}

		items := make([]lineItem, len(inlineContent))
		for i, c := range inlineContent {
			switch v := c.(type) {
			case Span:
				style := resolveSpanStyle(bodyStyle, v.Style)
				color := primaryColor
				if v.Style.Color.A > 0 {
					color = v.Style.Color
				}
				metrics := ctx.Canvas.MeasureText(v.Text, style)
				w := int(math.Ceil(float64(metrics.Width)))
				h := int(math.Ceil(float64(metrics.Ascent)))
				items[i] = lineItem{content: v, w: w, h: h, ascent: h, style: style, color: color}

			case InlineWidget:
				measured := ctx.MeasureChild(v.Element, ui.Bounds{
					X: 0, Y: 0,
					W: availW, H: ctx.Area.H,
				})
				items[i] = lineItem{
					content:  v,
					w:        measured.W,
					h:        measured.H,
					baseline: v.Baseline,
				}

			case ImageSpan:
				// ImageFloatNone — inline in text flow
				w, h := resolveImageSize(v, float32(defaultLineH))
				iw := int(math.Ceil(float64(w)))
				ih := int(math.Ceil(float64(h)))
				items[i] = lineItem{
					content:   v,
					w:         iw,
					h:         ih,
					baseline:  v.Baseline,
					imageID:   v.ImageID,
					scaleMode: v.ScaleMode,
					opacity:   imageOpacity(v.Opacity),
				}
			}
		}

		// ── Break items into lines and paint each line ──
		lineStart := 0

		for lineStart < len(items) {
			// Determine which items fit on this line.
			lineEnd := lineStart
			lineW := 0
			for lineEnd < len(items) {
				itemW := items[lineEnd].w
				if lineEnd > lineStart && lineW+itemW > availW {
					break
				}
				lineW += itemW
				lineEnd++
			}
			// Safety: at least one item per line to avoid infinite loops.
			if lineEnd == lineStart {
				lineEnd = lineStart + 1
				lineW = items[lineStart].w
			}

			// Compute line metrics: max ascent (text baseline) and max descent.
			lineAscent := 0
			lineDescent := 0

			for i := lineStart; i < lineEnd; i++ {
				it := items[i]
				switch it.content.(type) {
				case Span:
					if it.ascent > lineAscent {
						lineAscent = it.ascent
					}

				case InlineWidget, ImageSpan:
					// Widget/image bottom sits on baseline, shifted up by baseline offset.
					aboveBaseline := it.h - int(math.Round(float64(it.baseline)))
					if aboveBaseline < 0 {
						aboveBaseline = 0
					}
					belowBaseline := int(math.Round(float64(it.baseline)))
					if belowBaseline < 0 {
						belowBaseline = 0
					}
					if aboveBaseline > lineAscent {
						lineAscent = aboveBaseline
					}
					if belowBaseline > lineDescent {
						lineDescent = belowBaseline
					}
				}
			}

			// Fallback: ensure minimum line height from body style.
			if lineAscent < defaultLineH {
				lineAscent = defaultLineH
			}

			lineH := lineAscent + lineDescent

			// Paint items on this line.
			cursorX := inlineX
			for i := lineStart; i < lineEnd; i++ {
				it := items[i]
				switch v := it.content.(type) {
				case Span:
					drawY := cursorY + (lineAscent - it.ascent)
					ctx.Canvas.DrawText(v.Text,
						draw.Pt(float32(cursorX), float32(drawY)),
						it.style, it.color)

				case InlineWidget:
					widgetTop := cursorY + lineAscent - it.h + int(math.Round(float64(it.baseline)))
					if widgetTop < cursorY {
						widgetTop = cursorY
					}
					ctx.LayoutChild(v.Element, ui.Bounds{
						X: cursorX, Y: widgetTop,
						W: it.w, H: it.h,
					})

				case ImageSpan:
					imgTop := cursorY + lineAscent - it.h + int(math.Round(float64(it.baseline)))
					if imgTop < cursorY {
						imgTop = cursorY
					}
					r := draw.R(float32(cursorX), float32(imgTop), float32(it.w), float32(it.h))
					ctx.Canvas.DrawImageScaled(it.imageID, r, it.scaleMode, draw.ImageOptions{Opacity: it.opacity})
				}
				cursorX += it.w
			}

			lineEndX := inlineX + lineW
			if lineEndX > maxW {
				maxW = lineEndX - ctx.Area.X
			}

			cursorY += lineH
			lineStart = lineEnd
		}

		// ── Ensure cursorY clears float images ──
		floatMaxH := floatLeftH
		if floatRightH > floatMaxH {
			floatMaxH = floatRightH
		}
		if floatMaxH > 0 {
			floatBottom := paraStartY + floatMaxH
			if cursorY < floatBottom {
				cursorY = floatBottom
			}
		}

		// ── Render block images (full-width, below inline content) ──
		for _, img := range blockImgs {
			w, h := resolveImageSize(img, float32(defaultLineH))
			// Block images fill the full paragraph width when Width == 0.
			if img.Width == 0 {
				w = float32(ctx.Area.W)
			} else if w > float32(ctx.Area.W) {
				w = float32(ctx.Area.W)
			}
			r := draw.R(float32(ctx.Area.X), float32(cursorY), w, h)
			ctx.Canvas.DrawImageScaled(img.ImageID, r, img.ScaleMode, draw.ImageOptions{Opacity: imageOpacity(img.Opacity)})
			cursorY += int(math.Ceil(float64(h)))
			if int(w) > maxW {
				maxW = int(w)
			}
		}

		if pIdx < len(n.Paragraphs)-1 {
			cursorY += paraSpacing
		}
	}

	totalH := cursorY - ctx.Area.Y
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: maxW, H: totalH}
}

// resolveSpanStyle merges a SpanStyle with the theme body style.
// Non-zero fields in ss.Style override the corresponding body defaults.
func resolveSpanStyle(body draw.TextStyle, ss SpanStyle) draw.TextStyle {
	if ss.Style.Size > 0 {
		return ss.Style
	}
	style := body
	if ss.Style.Weight > 0 {
		style.Weight = ss.Style.Weight
	}
	if ss.Style.FontFamily != "" {
		style.FontFamily = ss.Style.FontFamily
	}
	if ss.Style.Style != draw.FontStyleNormal {
		style.Style = ss.Style.Style
	}
	if ss.Style.Tracking != 0 {
		style.Tracking = ss.Style.Tracking
	}
	if ss.Style.LineHeight > 0 {
		style.LineHeight = ss.Style.LineHeight
	}
	return style
}

// ── TreeEqual ───────────────────────────────────────────────────

// TreeEqual implements ui.TreeEqualizer.
func (n RichTextElement) TreeEqual(other ui.Element) bool {
	nb, ok := other.(RichTextElement)
	if !ok || len(n.Paragraphs) != len(nb.Paragraphs) {
		return false
	}
	for i, p := range n.Paragraphs {
		op := nb.Paragraphs[i]
		ac := paragraphContent(p)
		bc := paragraphContent(op)
		if len(ac) != len(bc) {
			return false
		}
		for j, ca := range ac {
			if !contentEqual(ca, bc[j]) {
				return false
			}
		}
	}
	return true
}

// contentEqual compares two ParagraphContent items for structural equality.
func contentEqual(a, b ParagraphContent) bool {
	switch va := a.(type) {
	case Span:
		vb, ok := b.(Span)
		return ok && va.Text == vb.Text && va.Style == vb.Style
	case InlineWidget:
		vb, ok := b.(InlineWidget)
		if !ok || va.Baseline != vb.Baseline {
			return false
		}
		return elementsEqual(va.Element, vb.Element)
	case ImageSpan:
		vb, ok := b.(ImageSpan)
		return ok && va == vb
	}
	return false
}

// elementsEqual compares two Elements for structural equality using
// the TreeEqualizer interface when available.
func elementsEqual(a, b ui.Element) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if te, ok := a.(ui.TreeEqualizer); ok {
		return te.TreeEqual(b)
	}
	return false
}

// ── ResolveChildren ─────────────────────────────────────────────

// ResolveChildren implements ui.ChildResolver.
// Inline widgets are resolved recursively; text spans and images are leaves.
func (n RichTextElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	changed := false
	paragraphs := make([]RichParagraph, len(n.Paragraphs))
	childIdx := 0

	for i, p := range n.Paragraphs {
		if len(p.Content) == 0 {
			paragraphs[i] = p
			continue
		}
		content := make([]ParagraphContent, len(p.Content))
		for j, c := range p.Content {
			if iw, ok := c.(InlineWidget); ok {
				resolved := resolve(iw.Element, childIdx)
				childIdx++
				if resolved != iw.Element {
					changed = true
					content[j] = InlineWidget{Element: resolved, Baseline: iw.Baseline}
				} else {
					content[j] = c
				}
			} else {
				content[j] = c
			}
		}
		paragraphs[i] = RichParagraph{Content: content, Spans: p.Spans}
	}

	if !changed {
		return n
	}
	return RichTextElement{BaseElement: n.BaseElement, Paragraphs: paragraphs}
}

// ── Accessibility ───────────────────────────────────────────────

// WalkAccess implements ui.AccessWalker.
// Walks inline widget children for accessibility tree building.
// ImageSpan items contribute their Alt text via the parent node's label.
func (n RichTextElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, p := range n.Paragraphs {
		for _, c := range p.Content {
			if iw, ok := c.(InlineWidget); ok {
				if aw, ok := iw.Element.(ui.AccessWalker); ok {
					aw.WalkAccess(b, parentIdx)
				}
			}
		}
	}
}
