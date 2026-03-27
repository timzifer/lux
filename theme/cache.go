package theme

// resolvedTheme holds the cached result of resolving a Theme's tokens
// and draw functions. Computed lazily on first access.
type resolvedTheme struct {
	tokens    TokenSet
	drawFuncs map[WidgetKind]DrawFunc
}

// allWidgetKinds lists every known WidgetKind for cache pre-warming.
var allWidgetKinds = []WidgetKind{
	WidgetKindButton, WidgetKindText, WidgetKindBox, WidgetKindIcon,
	WidgetKindStack, WidgetKindScrollView, WidgetKindDivider, WidgetKindSpacer,
	WidgetKindTextField, WidgetKindCheckbox, WidgetKindRadio, WidgetKindToggle,
	WidgetKindSlider, WidgetKindProgressBar, WidgetKindSelect,
	WidgetKindCard, WidgetKindTabs, WidgetKindAccordion, WidgetKindTooltip,
	WidgetKindBadge, WidgetKindChip, WidgetKindMenuBar, WidgetKindContextMenu,
	WidgetKindTextArea,
	WidgetKindDatePicker, WidgetKindColorPicker, WidgetKindTimePicker,
	WidgetKindNumericInput, WidgetKindSpinner,
	WidgetKindRichTextEditor,
	WidgetKindToolbar,
}

// CachedTheme wraps a Theme and caches the resolved TokenSet and DrawFuncs.
// All access is single-goroutine (app loop), so no mutex is needed.
type CachedTheme struct {
	base     Theme
	resolved *resolvedTheme
}

// NewCachedTheme creates a caching wrapper around the given theme.
func NewCachedTheme(base Theme) *CachedTheme {
	return &CachedTheme{base: base}
}

func (c *CachedTheme) resolve() *resolvedTheme {
	if c.resolved != nil {
		return c.resolved
	}
	r := &resolvedTheme{
		tokens:    c.base.Tokens(),
		drawFuncs: make(map[WidgetKind]DrawFunc, len(allWidgetKinds)),
	}
	for _, kind := range allWidgetKinds {
		r.drawFuncs[kind] = c.base.DrawFunc(kind)
	}
	c.resolved = r
	return r
}

// Tokens returns the cached token set.
func (c *CachedTheme) Tokens() TokenSet {
	return c.resolve().tokens
}

// DrawFunc returns the cached draw function for the given widget kind.
func (c *CachedTheme) DrawFunc(kind WidgetKind) DrawFunc {
	return c.resolve().drawFuncs[kind]
}

// Parent returns the base theme's parent.
func (c *CachedTheme) Parent() Theme {
	return c.base.Parent()
}

// Base returns the underlying unwrapped theme.
func (c *CachedTheme) Base() Theme {
	return c.base
}

// Invalidate clears the cache so the next access re-resolves from the base.
func (c *CachedTheme) Invalidate() {
	c.resolved = nil
}

// WarmUp eagerly resolves the cache (call before the first frame).
func (c *CachedTheme) WarmUp() {
	c.resolve()
}
