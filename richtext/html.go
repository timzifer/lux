package richtext

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// FromHTML parses an HTML string and returns an AttributedString.
// Supports semantic HTML tags (<b>, <i>, <u>, <s>, <strong>, <em>, <p>,
// <h1>-<h6>, <ul>, <ol>, <li>, <br>, <pre>, <code>, <span>, <div>, <img>),
// inline style attributes, and <style> blocks with CSS selectors.
func FromHTML(html string) (AttributedString, error) {
	doc, err := dom.ParseHTML(html)
	if err != nil {
		return AttributedString{}, err
	}

	// Extract <style> blocks.
	var sheets []*css.StyleSheet
	for _, styleEl := range doc.GetElementsByTagName("style") {
		text := styleEl.TextContent()
		if text != "" {
			sheet, err := css.ParseStyleSheet(text)
			if err == nil {
				sheets = append(sheets, sheet)
			}
		}
	}

	c := converter{sheets: sheets}
	c.walkNode(doc, 0)

	// Trim trailing newline if present.
	text := c.text.String()
	if strings.HasSuffix(text, "\n") {
		text = text[:len(text)-1]
		// Adjust attrs that end beyond the new length.
		for i := range c.attrs {
			if c.attrs[i].End > len(text) {
				c.attrs[i].End = len(text)
			}
		}
	}

	as := AttributedString{Text: text, Attrs: c.attrs}
	return as.Normalized(), nil
}

// blockElements are HTML elements that produce paragraph breaks.
var blockElements = map[string]bool{
	"p": true, "div": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "li": true, "blockquote": true,
	"pre": true, "section": true, "article": true, "header": true,
	"footer": true, "main": true, "nav": true, "aside": true,
}

type converter struct {
	sheets []*css.StyleSheet
	text   strings.Builder
	attrs  []Attr
}

func (c *converter) walkNode(n *dom.Node, listLevel int) {
	switch n.Type {
	case dom.TextNode:
		c.text.WriteString(n.Data)

	case dom.ElementNode:
		tag := strings.ToLower(n.Tag)

		// Skip non-visual elements.
		if tag == "style" || tag == "script" || tag == "head" || tag == "title" {
			return
		}

		// Handle <br> as newline.
		if tag == "br" {
			c.text.WriteString("\n")
			return
		}

		// Handle <img> as inline image placeholder.
		if tag == "img" {
			c.handleImg(n)
			return
		}

		start := c.text.Len()

		// Track list nesting.
		childListLevel := listLevel
		if tag == "ul" || tag == "ol" {
			childListLevel = listLevel + 1
		}

		// Recurse into children.
		for child := n.FirstChild; child != nil; child = child.NextSib {
			c.walkNode(child, childListLevel)
		}

		end := c.text.Len()

		// Apply block-element newline.
		if blockElements[tag] && end > start {
			c.text.WriteString("\n")
			end = c.text.Len() // include newline in paragraph range for para attrs
		}

		if end <= start {
			return
		}

		// Resolve CSS style for this element.
		style := css.Resolve(n, c.sheets)

		// Convert CSS properties to richtext attributes.
		c.applyCSS(start, end, style)

		// Handle list items.
		if tag == "li" {
			c.handleListItem(n, start, end, listLevel, style)
		}

	case dom.DocumentNode:
		for child := n.FirstChild; child != nil; child = child.NextSib {
			c.walkNode(child, listLevel)
		}
	}
}

func (c *converter) handleImg(n *dom.Node) {
	placeholder := "\uFFFC" // U+FFFC OBJECT REPLACEMENT CHARACTER
	start := c.text.Len()
	c.text.WriteString(placeholder)
	end := c.text.Len()

	img := ImageAttachment{
		Alt: n.Attr("alt"),
	}
	if w := n.Attr("width"); w != "" {
		if v, err := strconv.ParseFloat(w, 32); err == nil {
			img.Width = float32(v)
		}
	}
	if h := n.Attr("height"); h != "" {
		if v, err := strconv.ParseFloat(h, 32); err == nil {
			img.Height = float32(v)
		}
	}
	c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: ImageAttr(img)})
}

func (c *converter) handleListItem(n *dom.Node, start, end, level int, style css.StyleDeclaration) {
	// Determine list type from parent.
	parent := n.Parent
	if parent != nil {
		switch strings.ToLower(parent.Tag) {
		case "ul":
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: ListTypeAttr(draw.ListTypeUnordered)})
		case "ol":
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: ListTypeAttr(draw.ListTypeOrdered)})
			// Check for start attribute on <ol>.
			if s := parent.Attr("start"); s != "" {
				if v, err := strconv.Atoi(s); err == nil {
					c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: ListStartAttr(v)})
				}
			}
		}
	}

	if level > 0 {
		c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: ListLevelAttr(level - 1)})
	}

	// List marker from CSS.
	if marker := style.Get("list-style-type"); marker != "" {
		if m, ok := parseListMarker(marker); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: ListMarkerAttr(m)})
		}
	}
}

func (c *converter) applyCSS(start, end int, style css.StyleDeclaration) {
	if start >= end {
		return
	}

	// font-weight
	if v := style.Get("font-weight"); v != "" {
		switch v {
		case "bold":
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: BoldAttr(true)})
		case "normal":
			// explicit normal, no attr needed
		default:
			if w, err := strconv.Atoi(v); err == nil {
				if w >= 700 {
					c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: BoldAttr(true)})
				}
				c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: WeightAttr(draw.FontWeight(w))})
			}
		}
	}

	// font-style
	if v := style.Get("font-style"); v != "" {
		if v == "italic" || v == "oblique" {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: ItalicAttr(true)})
		}
	}

	// text-decoration
	if v := style.Get("text-decoration"); v != "" {
		if strings.Contains(v, "underline") {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: UnderlineAttr(true)})
		}
		if strings.Contains(v, "line-through") {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: StrikethroughAttr(true)})
		}
	}

	// font-family
	if v := style.Get("font-family"); v != "" {
		// Strip quotes around font names.
		v = strings.Trim(v, `"'`)
		// Take the first font family if multiple are specified.
		if i := strings.IndexByte(v, ','); i >= 0 {
			v = strings.TrimSpace(v[:i])
			v = strings.Trim(v, `"'`)
		}
		c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: FontFamilyAttr(v)})
	}

	// color
	if v := style.Get("color"); v != "" {
		if col, ok := css.ParseColor(v); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: ColorAttr(col)})
		}
	}

	// background-color
	if v := style.Get("background-color"); v != "" {
		if col, ok := css.ParseColor(v); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: BgColorAttr(col)})
		}
	}

	// font-size
	if v := style.Get("font-size"); v != "" {
		if size, ok := parseFontSize(v); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: SizeAttr(size)})
		}
	}

	// letter-spacing
	if v := style.Get("letter-spacing"); v != "" {
		if ls, ok := parseDimension(v); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: TrackingAttr(ls)})
		}
	}

	// line-height
	if v := style.Get("line-height"); v != "" {
		if lh, ok := parseLineHeight(v); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: LineHeightAttr(lh)})
		}
	}

	// white-space
	if v := style.Get("white-space"); v != "" {
		if ws, ok := parseWhiteSpace(v); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: WhiteSpaceAttr(ws)})
		}
	}

	// text-align
	if v := style.Get("text-align"); v != "" {
		if ta, ok := parseTextAlign(v); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: AlignAttr(ta)})
		}
	}

	// text-indent
	if v := style.Get("text-indent"); v != "" {
		if indent, ok := parseDimension(v); ok {
			c.attrs = append(c.attrs, Attr{Start: start, End: end, Value: IndentAttr(indent)})
		}
	}
}

// ── CSS value parsers ──────────────────────────────────────────

func parseFontSize(v string) (float32, bool) {
	// Named sizes (approximate dp values).
	switch v {
	case "xx-small":
		return 8, true
	case "x-small":
		return 10, true
	case "small":
		return 12, true
	case "medium":
		return 14, true
	case "large":
		return 16, true
	case "x-large":
		return 20, true
	case "xx-large":
		return 24, true
	}
	// Relative em sizes (multiply by a base of 14dp).
	if strings.HasSuffix(v, "em") {
		if f, err := strconv.ParseFloat(v[:len(v)-2], 32); err == nil {
			return float32(f) * 14, true
		}
	}
	return parseDimension(v)
}

func parseDimension(v string) (float32, bool) {
	v = strings.TrimSpace(v)
	// Strip known unit suffixes.
	for _, suffix := range []string{"px", "dp", "pt", "em", "rem"} {
		if strings.HasSuffix(v, suffix) {
			v = v[:len(v)-len(suffix)]
			break
		}
	}
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return 0, false
	}
	return float32(f), true
}

func parseLineHeight(v string) (float32, bool) {
	if v == "normal" {
		return 0, false // 0 = inherit
	}
	// Unitless number is a multiplier.
	if f, err := strconv.ParseFloat(v, 32); err == nil {
		return float32(f), true
	}
	return parseDimension(v)
}

func parseWhiteSpace(v string) (WhiteSpace, bool) {
	switch v {
	case "normal":
		return WhiteSpaceNormal, true
	case "pre":
		return WhiteSpacePre, true
	case "nowrap":
		return WhiteSpaceNoWrap, true
	case "pre-wrap":
		return WhiteSpacePreWrap, true
	case "pre-line":
		return WhiteSpacePreLine, true
	}
	return 0, false
}

func parseTextAlign(v string) (draw.TextAlign, bool) {
	switch v {
	case "left", "start":
		return draw.TextAlignLeft, true
	case "center":
		return draw.TextAlignCenter, true
	case "right", "end":
		return draw.TextAlignRight, true
	case "justify":
		return draw.TextAlignJustify, true
	}
	return 0, false
}

func parseListMarker(v string) (draw.ListMarker, bool) {
	switch v {
	case "disc":
		return draw.ListMarkerDisc, true
	case "circle":
		return draw.ListMarkerCircle, true
	case "square":
		return draw.ListMarkerSquare, true
	case "decimal":
		return draw.ListMarkerDecimal, true
	case "lower-alpha", "lower-latin":
		return draw.ListMarkerLowerAlpha, true
	case "upper-alpha", "upper-latin":
		return draw.ListMarkerUpperAlpha, true
	case "lower-roman":
		return draw.ListMarkerLowerRoman, true
	case "upper-roman":
		return draw.ListMarkerUpperRoman, true
	case "none":
		return draw.ListMarkerNone, true
	}
	return 0, false
}

// ── ToHTML ──────────────────────────────────────────────────────

// ToHTML converts an AttributedString to an HTML string.
// It uses semantic HTML tags where possible (<b>, <i>, <u>, <s>) and
// falls back to <span style="..."> for other properties.
func (as AttributedString) ToHTML() string {
	if as.IsEmpty() {
		return ""
	}

	doc := dom.NewDocument()

	// Split text into paragraphs.
	paragraphs := strings.Split(as.Text, "\n")
	offset := 0

	// Group consecutive list-item paragraphs.
	type paraInfo struct {
		text       string
		start, end int
		style      SpanStyle // resolved at paragraph start
	}

	var paras []paraInfo
	for _, pText := range paragraphs {
		pStart := offset
		pEnd := offset + len(pText)
		style := as.ResolveAt(pStart)
		paras = append(paras, paraInfo{text: pText, start: pStart, end: pEnd, style: style})
		offset = pEnd + 1 // +1 for \n
	}

	i := 0
	for i < len(paras) {
		p := paras[i]

		if p.style.ListType != draw.ListTypeNone && p.style.ListType != 0 {
			// Collect consecutive list items at this level.
			listTag := "ul"
			if p.style.ListType == draw.ListTypeOrdered {
				listTag = "ol"
			}
			listEl := dom.NewElement(listTag)
			if p.style.ListStart > 0 && p.style.ListType == draw.ListTypeOrdered {
				listEl.SetAttr("start", strconv.Itoa(p.style.ListStart))
			}
			for i < len(paras) && paras[i].style.ListType == p.style.ListType {
				li := dom.NewElement("li")
				c.renderRuns(li, as, paras[i].start, paras[i].end)
				listEl.AppendChild(li)
				i++
			}
			doc.AppendChild(listEl)
		} else {
			// Regular paragraph.
			pEl := dom.NewElement("p")
			c.renderRuns(pEl, as, p.start, p.end)
			doc.AppendChild(pEl)
			i++
		}
	}

	return dom.Serialize(doc)
}

// toHTMLConverter provides stateless methods for ToHTML conversion.
var c = toHTMLConverter{}

type toHTMLConverter struct{}

func (toHTMLConverter) renderRuns(parent *dom.Node, as AttributedString, start, end int) {
	if start >= end {
		return
	}
	runs := as.StyleRuns(start, end)
	for _, run := range runs {
		text := as.Text[run.Start:run.End]
		if text == "" {
			continue
		}

		node := c.styledNode(text, run.Style)
		parent.AppendChild(node)
	}
}

func (toHTMLConverter) styledNode(text string, style SpanStyle) *dom.Node {
	textNode := dom.NewText(text)

	// Build from inside out: text → semantic tags → span with remaining CSS.
	var current *dom.Node = textNode

	// Wrap in semantic tags.
	if style.Strikethrough {
		s := dom.NewElement("s")
		s.AppendChild(current)
		current = s
	}
	if style.Underline {
		u := dom.NewElement("u")
		u.AppendChild(current)
		current = u
	}
	if style.Italic {
		em := dom.NewElement("i")
		em.AppendChild(current)
		current = em
	}
	if style.Bold {
		b := dom.NewElement("b")
		b.AppendChild(current)
		current = b
	}

	// Collect remaining CSS properties.
	var cssProps []string

	if style.FontFamily != "" {
		cssProps = append(cssProps, fmt.Sprintf("font-family:%s", style.FontFamily))
	}
	if style.Weight != 0 && !style.Bold {
		cssProps = append(cssProps, fmt.Sprintf("font-weight:%d", style.Weight))
	}
	if style.Color != (draw.Color{}) {
		cssProps = append(cssProps, "color:"+css.FormatColor(style.Color))
	}
	if style.BgColor != (draw.Color{}) {
		cssProps = append(cssProps, "background-color:"+css.FormatColor(style.BgColor))
	}
	if style.Size != 0 {
		cssProps = append(cssProps, fmt.Sprintf("font-size:%.0fpx", style.Size))
	}
	if style.Tracking != 0 {
		cssProps = append(cssProps, fmt.Sprintf("letter-spacing:%.2fem", style.Tracking))
	}
	if style.LineHeight != 0 {
		cssProps = append(cssProps, fmt.Sprintf("line-height:%.2f", style.LineHeight))
	}
	if style.WhiteSpace != WhiteSpaceNormal {
		cssProps = append(cssProps, "white-space:"+whiteSpaceCSS(style.WhiteSpace))
	}
	if style.Align != draw.TextAlignLeft {
		cssProps = append(cssProps, "text-align:"+textAlignCSS(style.Align))
	}
	if style.Indent != 0 {
		cssProps = append(cssProps, fmt.Sprintf("text-indent:%.0fpx", style.Indent))
	}

	// Wrap in <span> if there are CSS properties and current is just text or already a tag.
	if len(cssProps) > 0 {
		span := dom.NewElement("span")
		span.SetAttr("style", strings.Join(cssProps, ";"))
		span.AppendChild(current)
		current = span
	}

	return current
}

func whiteSpaceCSS(ws WhiteSpace) string {
	switch ws {
	case WhiteSpacePre:
		return "pre"
	case WhiteSpaceNoWrap:
		return "nowrap"
	case WhiteSpacePreWrap:
		return "pre-wrap"
	case WhiteSpacePreLine:
		return "pre-line"
	default:
		return "normal"
	}
}

func textAlignCSS(ta draw.TextAlign) string {
	switch ta {
	case draw.TextAlignCenter:
		return "center"
	case draw.TextAlignRight:
		return "right"
	case draw.TextAlignJustify:
		return "justify"
	default:
		return "left"
	}
}
