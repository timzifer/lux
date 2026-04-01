package html

import (
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"
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

// display_text_empty returns a minimal empty text element.
// Named with underscore to avoid collision with display.Text import.
func display_text_empty() ui.Element {
	return layout.Sized(0, 0)
}
