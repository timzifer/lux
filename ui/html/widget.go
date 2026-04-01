package html

import (
	"strings"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// ── ViewOption ──────────────────────────────────────────────────────

// ViewOption configures an HTMLView widget.
type ViewOption func(*HTMLView)

// WithOnLink sets a callback invoked when a link is clicked.
func WithOnLink(fn func(href string)) ViewOption {
	return func(v *HTMLView) { v.OnLink = fn }
}

// WithMaxWidth sets a maximum content width in dp.
func WithMaxWidth(w float32) ViewOption {
	return func(v *HTMLView) { v.MaxWidth = w }
}

// WithBaseURL sets the base URL for resolving relative URLs (future).
func WithBaseURL(url string) ViewOption {
	return func(v *HTMLView) { v.BaseURL = url }
}

// WithScrollable enables scrolling with a maximum viewport height.
func WithScrollable(maxHeight float32) ViewOption {
	return func(v *HTMLView) { v.ScrollHeight = maxHeight }
}

// ── HTMLView Widget ─────────────────────────────────────────────────

// HTMLView is a stateful Widget that renders an HTML Document as a
// native Lux element tree. It implements ui.Widget.
type HTMLView struct {
	// Doc is the parsed HTML document to render.
	Doc *Document

	// BaseURL for resolving relative URLs (future phases).
	BaseURL string

	// MaxWidth constrains the content width (0 = use available width).
	MaxWidth float32

	// ScrollHeight enables scrolling when set to a positive value.
	// The widget is wrapped in a ScrollView with this max height.
	ScrollHeight float32

	// OnLink is called when a hyperlink is clicked.
	OnLink func(href string)
}

// htmlViewState holds persistent state across renders.
type htmlViewState struct {
	scroll *ui.ScrollState
}

// Render implements ui.Widget. It builds the full element tree from
// the HTML document.
func (w HTMLView) Render(ctx ui.RenderCtx, state ui.WidgetState) (ui.Element, ui.WidgetState) {
	s, _ := state.(*htmlViewState)
	if s == nil {
		s = &htmlViewState{
			scroll: &ui.ScrollState{},
		}
	}

	if w.Doc == nil || w.Doc.Root == nil {
		return display_text_empty(), s
	}

	b := &builder{
		sheets: w.Doc.Sheets,
		onLink: w.OnLink,
	}

	el := b.buildElement(w.Doc.Root)
	if el == nil {
		return display_text_empty(), s
	}

	// Apply html element's background-color as a full-width wrapper.
	// Per CSS spec, the html element's background covers the entire canvas.
	if htmlNode := findHTMLElement(w.Doc.Root); htmlNode != nil {
		htmlStyle := css.Resolve(htmlNode, w.Doc.Sheets)
		if v := htmlStyle.Get("background-color"); v != "" {
			if c, ok := css.ParseColor(v); ok {
				el = CanvasBackground{Child: el, Color: c}
			}
		}
	}

	// Apply max width constraint.
	if w.MaxWidth > 0 {
		el = layout.Sized(w.MaxWidth, 0, el)
	}

	// Wrap in ScrollView if scrollable.
	if w.ScrollHeight > 0 {
		el = nav.NewScrollView(el, w.ScrollHeight, s.scroll)
	}

	return el, s
}

// ── Convenience constructors ────────────────────────────────────────

// View creates an HTMLView element from an HTML string.
// Returns an empty text element if parsing fails.
func View(htmlStr string, opts ...ViewOption) ui.Element {
	doc, err := Parse(htmlStr)
	if err != nil {
		return display_text_empty()
	}
	return ViewFromDocument(doc, opts...)
}

// ViewFromDocument creates an HTMLView element from a pre-parsed Document.
func ViewFromDocument(doc *Document, opts ...ViewOption) ui.Element {
	w := HTMLView{Doc: doc}
	for _, opt := range opts {
		opt(&w)
	}
	return ui.Component(w)
}

// findHTMLElement finds the <html> element in the DOM tree.
func findHTMLElement(root *dom.Node) *dom.Node {
	if root == nil {
		return nil
	}
	if root.Type == dom.ElementNode && strings.ToLower(root.Tag) == "html" {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSib {
		if found := findHTMLElement(child); found != nil {
			return found
		}
	}
	return nil
}

// CanvasBackground is a layout element that fills its entire area with
// a background color before rendering its child. Used for the html
// element's background which covers the entire viewport.
type CanvasBackground struct {
	ui.BaseElement
	Child ui.Element
	Color draw.Color
}

// LayoutSelf implements ui.Layouter.
func (n CanvasBackground) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	// Fill the entire area with the background color.
	if n.Color != (draw.Color{}) {
		ctx.Canvas.FillRect(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(area.H)),
			draw.SolidPaint(n.Color))
	}
	// Layout child within the area.
	var cb ui.Bounds
	if n.Child != nil {
		cb = ctx.LayoutChild(n.Child, area)
	}
	return cb
}

// TreeEqual implements ui.TreeEqualizer.
func (n CanvasBackground) TreeEqual(other ui.Element) bool {
	o, ok := other.(CanvasBackground)
	return ok && n.Color == o.Color
}

// ResolveChildren implements ui.ChildResolver.
func (n CanvasBackground) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	if n.Child != nil {
		out.Child = resolve(n.Child, 0)
	}
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n CanvasBackground) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	if n.Child != nil {
		b.Walk(n.Child, parentIdx)
	}
}

// display_text_empty returns a minimal empty text element.
// Named with underscore to avoid collision with display.Text import.
func display_text_empty() ui.Element {
	return layout.Sized(0, 0)
}
