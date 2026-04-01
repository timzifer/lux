package html

import (
	"math"
	"strings"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// ── Display classification ─────────────────────────────────────────

// defaultDisplay returns the default CSS display value for an HTML tag.
var defaultDisplay = map[string]string{
	// Block-level elements
	"html": "block", "body": "block",
	"div": "block", "p": "block", "section": "block", "article": "block",
	"header": "block", "footer": "block", "main": "block", "nav": "block",
	"aside": "block", "blockquote": "block", "pre": "block",
	"h1": "block", "h2": "block", "h3": "block",
	"h4": "block", "h5": "block", "h6": "block",
	"form": "block", "fieldset": "block", "details": "block",
	"figure": "block", "figcaption": "block", "address": "block",
	"hr": "block", "dl": "block", "dt": "block", "dd": "block",

	// List items
	"li": "list-item",
	"ul": "block", "ol": "block",

	// Table elements
	"table":    "table",
	"tr":       "table-row",
	"td":       "table-cell",
	"th":       "table-cell",
	"thead":    "table-header-group",
	"tbody":    "table-row-group",
	"tfoot":    "table-footer-group",
	"caption":  "table-caption",
	"col":      "table-column",
	"colgroup": "table-column-group",

	// Inline elements (default for unknown tags too)
	"span": "inline", "a": "inline", "b": "inline", "strong": "inline",
	"i": "inline", "em": "inline", "u": "inline", "s": "inline",
	"code": "inline", "small": "inline", "sub": "inline", "sup": "inline",
	"abbr": "inline", "label": "inline", "mark": "inline", "q": "inline",
	"cite": "inline", "dfn": "inline", "var": "inline", "kbd": "inline",
	"samp": "inline", "time": "inline", "del": "inline", "ins": "inline",
	"br": "inline", "img": "inline",

	// Hidden elements
	"script": "none", "style": "none", "head": "none",
	"title": "none", "meta": "none", "link": "none", "template": "none",

	// Replaced inline elements
	"input":    "inline-block",
	"select":   "inline-block",
	"textarea": "inline-block",
	"button":   "inline-block",
	"progress": "inline-block",
}

// resolveDisplay returns the effective CSS display value for a DOM node.
// It checks the computed style first, falling back to HTML tag defaults.
func resolveDisplay(node *dom.Node, style css.StyleDeclaration) string {
	if v := style.Get("display"); v != "" {
		return strings.TrimSpace(v)
	}
	if d, ok := defaultDisplay[strings.ToLower(node.Tag)]; ok {
		return d
	}
	return "inline"
}

// isBlockDisplay returns true if the display value produces a
// block-level element in the HTML formatting context.
func isBlockDisplay(display string) bool {
	switch display {
	case "block", "flex", "grid", "table", "list-item":
		return true
	}
	return false
}

// isInlineDisplay returns true if the display value produces an
// inline-level element.
func isInlineDisplay(display string) bool {
	switch display {
	case "inline", "inline-block":
		return true
	}
	return false
}

// isTableDisplay returns true if the display value is a table-related value.
func isTableDisplay(display string) bool {
	return strings.HasPrefix(display, "table")
}

// ── Font-size resolution ──────────────────────────────────────────

// resolvedFontSize returns the computed font-size in dp for a style,
// using parentFontSize to resolve em/rem units.
func resolvedFontSize(style css.StyleDeclaration, parentFontSize float32) float32 {
	if v := style.Get("font-size"); v != "" {
		if size, ok := css.ParseFontSizeWith(v, parentFontSize); ok {
			return size
		}
	}
	if parentFontSize > 0 {
		return parentFontSize
	}
	return css.DefaultFontSize
}

// ── CSS to SpanStyle ────────────────────────────────────────────────

// toSpanStyle converts resolved CSS properties into a display.SpanStyle
// for use in RichText Spans. parentFontSize is the inherited font-size
// for resolving em units.
func toSpanStyle(style css.StyleDeclaration, parentFontSize float32) display.SpanStyle {
	var ts draw.TextStyle
	var color draw.Color

	if v := style.Get("font-family"); v != "" {
		v = strings.Trim(v, `"'`)
		if i := strings.IndexByte(v, ','); i >= 0 {
			v = strings.TrimSpace(v[:i])
			v = strings.Trim(v, `"'`)
		}
		ts.FontFamily = v
	}

	if v := style.Get("font-size"); v != "" {
		if size, ok := css.ParseFontSizeWith(v, parentFontSize); ok {
			ts.Size = size
		}
	}

	if v := style.Get("font-weight"); v != "" {
		if w, _, ok := css.ParseFontWeight(v); ok {
			ts.Weight = w
		}
	}

	if v := style.Get("font-style"); v != "" {
		if v == "italic" || v == "oblique" {
			ts.Style = draw.FontStyleItalic
		}
	}

	if v := style.Get("line-height"); v != "" {
		if lh, ok := css.ParseLineHeight(v); ok {
			ts.LineHeight = lh
		}
	}

	if v := style.Get("letter-spacing"); v != "" {
		fontSize := resolvedFontSize(style, parentFontSize)
		if ls, ok := css.ResolveDimension(v, fontSize); ok {
			ts.Tracking = ls
		}
	}

	if v := style.Get("color"); v != "" {
		if c, ok := css.ParseColor(v); ok {
			color = c
		}
	}

	return display.SpanStyle{Style: ts, Color: color}
}

// ── CSS to ParagraphStyle ───────────────────────────────────────────

// toParagraphStyle converts resolved CSS properties into a
// display.ParagraphStyle for block-level formatting.
func toParagraphStyle(style css.StyleDeclaration) display.ParagraphStyle {
	var ps display.ParagraphStyle

	if v := style.Get("text-align"); v != "" {
		if ta, ok := css.ParseTextAlign(v); ok {
			ps.Align = ta
		}
	}

	if v := style.Get("line-height"); v != "" {
		if lh, ok := css.ParseLineHeight(v); ok {
			ps.LineHeight = lh
		}
	}

	if v := style.Get("text-indent"); v != "" {
		if indent, ok := css.ParseDimension(v); ok {
			ps.Indent = indent
		}
	}

	return ps
}

// ── CSS to Flex ─────────────────────────────────────────────────────

// toFlexContainer converts CSS flex container properties into a
// layout.Flex element populated with children.
func toFlexContainer(style css.StyleDeclaration, children []ui.Element) layout.Flex {
	f := layout.Flex{
		Direction: layout.FlexRow,
		Justify:   layout.JustifyStart,
		Align:     layout.AlignStretch,
		Children:  children,
	}

	if v := style.Get("flex-direction"); v != "" {
		switch v {
		case "row":
			f.Direction = layout.FlexRow
		case "column":
			f.Direction = layout.FlexColumn
		case "row-reverse":
			f.Direction = layout.FlexRowReverse
		case "column-reverse":
			f.Direction = layout.FlexColumnReverse
		}
	}

	if v := style.Get("flex-wrap"); v != "" {
		switch v {
		case "nowrap":
			f.Wrap = layout.FlexNoWrap
		case "wrap":
			f.Wrap = layout.FlexWrapOn
		case "wrap-reverse":
			f.Wrap = layout.FlexWrapReverse
		}
	}

	if v := style.Get("justify-content"); v != "" {
		switch v {
		case "flex-start", "start":
			f.Justify = layout.JustifyStart
		case "flex-end", "end":
			f.Justify = layout.JustifyEnd
		case "center":
			f.Justify = layout.JustifyCenter
		case "space-between":
			f.Justify = layout.JustifySpaceBetween
		case "space-around":
			f.Justify = layout.JustifySpaceAround
		case "space-evenly":
			f.Justify = layout.JustifySpaceEvenly
		}
	}

	if v := style.Get("align-items"); v != "" {
		switch v {
		case "flex-start", "start":
			f.Align = layout.AlignStart
		case "flex-end", "end":
			f.Align = layout.AlignEnd
		case "center":
			f.Align = layout.AlignCenter
		case "stretch":
			f.Align = layout.AlignStretch
		}
	}

	if v := style.Get("align-content"); v != "" {
		switch v {
		case "flex-start", "start":
			f.AlignContent = layout.AlignContentStart
		case "flex-end", "end":
			f.AlignContent = layout.AlignContentEnd
		case "center":
			f.AlignContent = layout.AlignContentCenter
		case "space-between":
			f.AlignContent = layout.AlignContentSpaceBetween
		case "space-around":
			f.AlignContent = layout.AlignContentSpaceAround
		case "stretch":
			f.AlignContent = layout.AlignContentStretch
		}
	}

	if v := style.Get("gap"); v != "" {
		if g, ok := css.ParseDimension(v); ok {
			f.RowGap = g
			f.ColGap = g
		}
	}
	if v := style.Get("row-gap"); v != "" {
		if g, ok := css.ParseDimension(v); ok {
			f.RowGap = g
		}
	}
	if v := style.Get("column-gap"); v != "" {
		if g, ok := css.ParseDimension(v); ok {
			f.ColGap = g
		}
	}

	return f
}

// ── StyledBox ───────────────────────────────────────────────────────

// StyledBox is a wrapper element that applies CSS box-model styling
// (background, border, border-radius, padding, margin, width, height)
// around a child element.
type StyledBox struct {
	ui.BaseElement
	Child          ui.Element
	Background     draw.Color
	BorderColor    draw.Color
	BorderWidth    [4]float32 // top, right, bottom, left
	HasBorderStyle bool       // true if border-style is set (not "none")
	BorderRadius   float32
	Width          float32    // 0 = auto
	Height         float32    // 0 = auto
	WidthPct       float32    // 0 = not percentage; fraction (0.5 = 50%)
	Padding        [4]float32 // top, right, bottom, left
	Margin         [4]float32 // top, right, bottom, left
}

// borderWidthSum returns the total horizontal or vertical border width.
func (n StyledBox) borderH() int { return int(n.BorderWidth[1]) + int(n.BorderWidth[3]) } // right + left
func (n StyledBox) borderV() int { return int(n.BorderWidth[0]) + int(n.BorderWidth[2]) } // top + bottom
func (n StyledBox) hasBorder() bool {
	return n.BorderWidth != [4]float32{} && (n.BorderColor != (draw.Color{}) || n.HasBorderStyle)
}

// LayoutSelf implements ui.Layouter.
func (n StyledBox) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas

	bT, bR, bB, bL := int(n.BorderWidth[0]), int(n.BorderWidth[1]), int(n.BorderWidth[2]), int(n.BorderWidth[3])

	// Compute padding (needed before dimension resolution for content-box).
	pT := int(n.Padding[0])
	pR := int(n.Padding[1])
	pB := int(n.Padding[2])
	pL := int(n.Padding[3])

	// Apply margin.
	mx := area.X + int(n.Margin[3]) // left
	my := area.Y + int(n.Margin[0]) // top
	mw := area.W - int(n.Margin[1]) - int(n.Margin[3])
	mh := area.H - int(n.Margin[0]) - int(n.Margin[2])

	// CSS width/height refer to the CONTENT area (content-box model).
	// We convert to full box dimensions by adding padding and border.
	if n.WidthPct > 0 {
		contentW := int(float32(area.W) * n.WidthPct)
		mw = contentW + pL + pR + bL + bR
	}
	if n.Width > 0 {
		mw = int(n.Width) + pL + pR + bL + bR
	}
	if n.Height > 0 {
		mh = int(n.Height) + pT + pB + bT + bB
	}
	contentArea := ui.Bounds{
		X: mx + pL + bL,
		Y: my + pT + bT,
		W: max(mw-pL-pR-bL-bR, 0),
		H: max(mh-pT-pB-bT-bB, 0),
	}

	// Determine box dimensions.
	// For explicit dimensions, use the computed value.
	// For auto dimensions, measure the child first (without painting).
	hasExplicitW := n.Width > 0 || n.WidthPct > 0
	hasExplicitH := n.Height > 0
	var cb ui.Bounds
	boxW, boxH := mw, mh

	if (!hasExplicitW || !hasExplicitH) && n.Child != nil {
		// Measure child to determine auto dimensions.
		mb := ctx.MeasureChild(n.Child, contentArea)
		if !hasExplicitW {
			boxW = mb.W + pL + pR + bL + bR
		}
		if !hasExplicitH {
			boxH = mb.H + pT + pB + bT + bB
		}
	}

	boxRect := draw.R(float32(mx), float32(my), float32(boxW), float32(boxH))

	// Draw background BEFORE children so it appears behind them.
	if n.Background != (draw.Color{}) {
		if n.BorderRadius > 0 {
			canvas.FillRoundRect(boxRect, n.BorderRadius, draw.SolidPaint(n.Background))
		} else {
			canvas.FillRect(boxRect, draw.SolidPaint(n.Background))
		}
	}

	// Draw border BEFORE children so it appears behind content.
	if n.hasBorder() {
		bc := n.BorderColor
		if bc == (draw.Color{}) {
			bc = draw.Color{R: 0, G: 0, B: 0, A: 1}
		}
		paint := draw.SolidPaint(bc)
		fmx, fmy := float32(mx), float32(my)
		fbW, fbH := float32(boxW), float32(boxH)
		bwT, bwR, bwB, bwL := n.BorderWidth[0], n.BorderWidth[1], n.BorderWidth[2], n.BorderWidth[3]
		// Top (full width)
		if bwT > 0 {
			canvas.FillRect(draw.R(fmx, fmy, fbW, bwT), paint)
		}
		// Bottom (full width)
		if bwB > 0 {
			canvas.FillRect(draw.R(fmx, fmy+fbH-bwB, fbW, bwB), paint)
		}
		// Left (between top and bottom borders)
		if bwL > 0 {
			canvas.FillRect(draw.R(fmx, fmy+bwT, bwL, fbH-bwT-bwB), paint)
		}
		// Right (between top and bottom borders)
		if bwR > 0 {
			canvas.FillRect(draw.R(fmx+fbW-bwR, fmy+bwT, bwR, fbH-bwT-bwB), paint)
		}
	}

	// Layout child AFTER background/border so content renders on top.
	if n.Child != nil {
		cb = ctx.LayoutChild(n.Child, contentArea)
	}

	// For auto-sized dimensions, use actual child bounds after layout.
	if !hasExplicitW {
		boxW = cb.W + pL + pR + bL + bR
	}
	if !hasExplicitH {
		boxH = cb.H + pT + pB + bT + bB
	}

	totalW := boxW + int(n.Margin[1]) + int(n.Margin[3])
	totalH := boxH + int(n.Margin[0]) + int(n.Margin[2])

	return ui.Bounds{
		X:        area.X,
		Y:        area.Y,
		W:        totalW,
		H:        totalH,
		Baseline: int(n.Margin[0]) + pT + bT + cb.Baseline,
	}
}

// TreeEqual implements ui.TreeEqualizer.
func (n StyledBox) TreeEqual(other ui.Element) bool {
	o, ok := other.(StyledBox)
	if !ok {
		return false
	}
	return n.Background == o.Background &&
		n.BorderColor == o.BorderColor &&
		n.BorderWidth == o.BorderWidth &&
		n.HasBorderStyle == o.HasBorderStyle &&
		n.BorderRadius == o.BorderRadius &&
		n.Width == o.Width &&
		n.Height == o.Height &&
		n.WidthPct == o.WidthPct &&
		n.Padding == o.Padding &&
		n.Margin == o.Margin
}

// ResolveChildren implements ui.ChildResolver.
func (n StyledBox) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	if n.Child != nil {
		out.Child = resolve(n.Child, 0)
	}
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n StyledBox) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	if n.Child != nil {
		b.Walk(n.Child, parentIdx)
	}
}

// ── Box model application ───────────────────────────────────────────

// applyBoxStyle extracts CSS box-model properties (padding, margin,
// background, border, width, height) from a computed style and wraps
// the element in a StyledBox if any visual properties are set.
// fontSize is the resolved font-size for em unit resolution.
func applyBoxStyle(el ui.Element, style css.StyleDeclaration, fontSize float32) ui.Element {
	if el == nil {
		return nil
	}

	var box StyledBox
	box.Child = el
	hasStyle := false

	if fontSize <= 0 {
		fontSize = css.DefaultFontSize
	}

	// Background.
	if v := style.Get("background-color"); v != "" {
		if c, ok := css.ParseColor(v); ok {
			box.Background = c
			hasStyle = true
		}
	}
	if v := style.Get("background"); v != "" {
		// Simple case: treat as color if parseable.
		if c, ok := css.ParseColor(v); ok {
			box.Background = c
			hasStyle = true
		}
	}

	// Border — separate properties first, then shorthand can override.
	if v := style.Get("border-width"); v != "" {
		if dims, ok := css.ResolveBoxDimensions(v, fontSize); ok {
			box.BorderWidth = dims
			hasStyle = true
		}
	}
	if v := style.Get("border-color"); v != "" {
		if c, ok := css.ParseColor(v); ok {
			box.BorderColor = c
			hasStyle = true
		}
	}
	if v := style.Get("border-style"); v != "" {
		v = strings.TrimSpace(v)
		if v != "none" && v != "" {
			box.HasBorderStyle = true
			hasStyle = true
		}
	}
	if v := style.Get("border-radius"); v != "" {
		if br, ok := css.ResolveDimension(v, fontSize); ok {
			box.BorderRadius = br
			hasStyle = true
		}
	}
	// Legacy shorthand fallback: if "border" is still present (e.g. from
	// inline styles that bypassed Set expansion), parse it directly.
	if v := style.Get("border"); v != "" {
		parseBorderShorthand(v, &box, fontSize)
		if box.BorderWidth != ([4]float32{}) {
			hasStyle = true
		}
	}

	// Padding.
	if v := style.Get("padding"); v != "" {
		if dims, ok := css.ResolveBoxDimensions(v, fontSize); ok {
			box.Padding = dims
			hasStyle = true
		}
	}
	applyIndividualSide(style, "padding-top", &box.Padding[0], &hasStyle, fontSize)
	applyIndividualSide(style, "padding-right", &box.Padding[1], &hasStyle, fontSize)
	applyIndividualSide(style, "padding-bottom", &box.Padding[2], &hasStyle, fontSize)
	applyIndividualSide(style, "padding-left", &box.Padding[3], &hasStyle, fontSize)

	// Margin.
	if v := style.Get("margin"); v != "" {
		if dims, ok := css.ResolveBoxDimensions(v, fontSize); ok {
			box.Margin = dims
			hasStyle = true
		}
	}
	applyIndividualSide(style, "margin-top", &box.Margin[0], &hasStyle, fontSize)
	applyIndividualSide(style, "margin-right", &box.Margin[1], &hasStyle, fontSize)
	applyIndividualSide(style, "margin-bottom", &box.Margin[2], &hasStyle, fontSize)
	applyIndividualSide(style, "margin-left", &box.Margin[3], &hasStyle, fontSize)

	// Dimensions.
	if v := style.Get("width"); v != "" {
		if css.IsPercentage(v) {
			if pct, ok := css.ParsePercentage(v); ok {
				box.WidthPct = pct
				hasStyle = true
			}
		} else if w, ok := css.ResolveDimension(v, fontSize); ok {
			box.Width = w
			hasStyle = true
		}
	}
	if v := style.Get("height"); v != "" {
		if h, ok := css.ResolveDimension(v, fontSize); ok {
			box.Height = h
			hasStyle = true
		}
	}

	if !hasStyle {
		return el
	}
	return box
}

func applyIndividualSide(style css.StyleDeclaration, prop string, target *float32, hasStyle *bool, fontSize float32) {
	if v := style.Get(prop); v != "" {
		if d, ok := css.ResolveDimension(v, fontSize); ok {
			*target = d
			*hasStyle = true
		}
	}
}

// parseBorderShorthand parses "1px solid #000" style border shorthands.
func parseBorderShorthand(v string, box *StyledBox, fontSize float32) {
	parts := strings.Fields(v)
	for _, part := range parts {
		// Try as dimension (border-width).
		if d, ok := css.ResolveDimension(part, fontSize); ok {
			box.BorderWidth = [4]float32{d, d, d, d}
			box.HasBorderStyle = true
			continue
		}
		// Try as color.
		if c, ok := css.ParseColor(part); ok {
			box.BorderColor = c
			continue
		}
		// Style keywords (solid, dashed, etc.) — mark border as present.
		switch part {
		case "solid", "dashed", "dotted", "double", "groove", "ridge", "inset", "outset":
			box.HasBorderStyle = true
		}
	}
}

// ── Utility ─────────────────────────────────────────────────────────

// clamp restricts v to [lo, hi].
func clamp(v, lo, hi float32) float32 {
	return float32(math.Max(float64(lo), math.Min(float64(hi), float64(v))))
}

// ignore the unused clamp warning for now.
var _ = clamp
