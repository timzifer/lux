// Package ui defines the Widget system and Element types for the
// virtual tree (RFC §4).
package ui

import (
	"math"
	"time"

	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/theme"
)

// ── Widget System (RFC §4) ───────────────────────────────────────

// WidgetState is an open interface — any type qualifies (RFC §4.1).
type WidgetState interface{}

// UID identifies a widget instance across frames.
type UID uint64

// Widget is the core interface for stateful, renderable components
// (RFC §4.2).
type Widget interface {
	// Render returns an Element tree and (optionally updated) state.
	// state is nil on the first call.
	Render(ctx RenderCtx, state WidgetState) (Element, WidgetState)
}

// RenderCtx is the context passed to Widget.Render (RFC §4.2).
type RenderCtx struct {
	UID    UID
	Theme  theme.Theme
	Send   func(any) // local Send bound to this UID
}

// AdoptState is a generic helper that type-asserts the raw state or
// returns a zero-value pointer for the first render (RFC §4.2).
func AdoptState[S WidgetState](raw WidgetState) *S {
	if s, ok := raw.(*S); ok {
		return s
	}
	return new(S)
}

// ── Element Types (RFC §4.3) ─────────────────────────────────────

// Element is the base interface for all virtual-tree nodes.
type Element interface {
	isElement()
}

// LayoutAxis controls how a Box arranges its children.
type LayoutAxis int

const (
	AxisColumn LayoutAxis = iota
	AxisRow
)

// Empty returns an Element that renders nothing.
func Empty() Element { return emptyElement{} }

// Text creates a text element.
func Text(content string) Element { return textElement{Content: content} }

// TextStyled creates a text element with a specific text style.
// Use this for headings or other non-Body text.
func TextStyled(content string, style draw.TextStyle) Element {
	return textElement{Content: content, Style: style}
}

// Button creates a button element with an optional click callback.
func Button(label string, onClick func()) Element {
	return buttonElement{Label: label, OnClick: onClick}
}

// Column stacks children vertically.
func Column(children ...Element) Element {
	return boxElement{Axis: AxisColumn, Children: children}
}

// Row stacks children horizontally.
func Row(children ...Element) Element {
	return boxElement{Axis: AxisRow, Children: children}
}

// WithKey wraps an element with an explicit key for stable UIDs
// across re-parenting (RFC §4.4).
func WithKey(key string, el Element) Element {
	return keyedElement{Key: key, Child: el}
}

// Divider creates a horizontal divider line (RFC-003 §4.1).
func Divider() Element { return dividerElement{} }

// Spacer creates invisible spacing of the given size in dp (RFC-003 §4.1).
func Spacer(size float32) Element { return spacerElement{Size: size} }

// Icon renders a text symbol at the theme's label size (RFC-003 §4.1).
// The name is rendered as-is (typically a single character or emoji).
func Icon(name string) Element { return iconElement{Name: name, Size: 0} }

// IconSize renders a text symbol at a specific size in dp.
func IconSize(name string, size float32) Element { return iconElement{Name: name, Size: size} }

// Stack overlays children on top of each other (z-axis, RFC-003 §4.1).
// First child is the bottom layer, last child is the top layer.
func Stack(children ...Element) Element {
	return stackElement{Children: children}
}

// ScrollView constrains a child to a maximum height, clipping overflow
// and rendering a scrollbar when content exceeds the viewport (RFC-003 §4.1).
// An optional ScrollState pointer drives the vertical offset; pass nil for static views.
func ScrollView(child Element, maxHeight float32, state ...*ScrollState) Element {
	var s *ScrollState
	if len(state) > 0 {
		s = state[0]
	}
	return scrollViewElement{Child: child, MaxHeight: maxHeight, State: s}
}

// ── Tier 2 Constructors (RFC-003 §4.1) ──────────────────────────

// Checkbox creates a boolean toggle with a label.
func Checkbox(label string, checked bool, onToggle func(bool)) Element {
	return checkboxElement{Label: label, Checked: checked, OnToggle: onToggle}
}

// Radio creates a single-choice option. Group multiple Radio elements
// in a Column; the user's model owns which option is selected.
func Radio(label string, selected bool, onSelect func()) Element {
	return radioElement{Label: label, Selected: selected, OnSelect: onSelect}
}

// Toggle creates a switch widget. An optional ToggleState pointer enables
// smooth thumb animation; pass nil for instant snap.
func Toggle(on bool, onToggle func(bool), state ...*ToggleState) Element {
	var s *ToggleState
	if len(state) > 0 {
		s = state[0]
	}
	return toggleElement{On: on, OnToggle: onToggle, State: s}
}

// Slider creates a continuous value selector (0.0–1.0).
func Slider(value float32, onChange func(float32)) Element {
	return sliderElement{Value: value, OnChange: onChange}
}

// ProgressBar creates a determinate progress indicator (0.0–1.0).
func ProgressBar(value float32) Element {
	return progressBarElement{Value: value}
}

// ProgressBarIndeterminate creates an indeterminate progress indicator.
// An optional phase (0.0–1.0) controls the animation position; pass
// a value derived from app.TickMsg to animate the bar.
func ProgressBarIndeterminate(phase ...float32) Element {
	var p float32
	if len(phase) > 0 {
		p = phase[0]
	}
	return progressBarElement{Indeterminate: true, Phase: p}
}

// TextField creates a text input field. If onChange is non-nil and the
// field is focused, keyboard input will call onChange with the updated value.
func TextField(value string, placeholder string, opts ...TextFieldOption) Element {
	el := textFieldElement{Value: value, Placeholder: placeholder}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// TextFieldOption configures a TextField.
type TextFieldOption func(*textFieldElement)

// WithOnChange sets the callback invoked when the text value changes.
func WithOnChange(fn func(string)) TextFieldOption {
	return func(e *textFieldElement) { e.OnChange = fn }
}

// WithFocusState links the TextField to a FocusState for keyboard input.
func WithFocusState(fs *FocusState) TextFieldOption {
	return func(e *textFieldElement) { e.Focus = fs }
}

// Select creates a dropdown selector (visual only — dropdown overlay
// requires a future popup system).
func Select(value string, options []string) Element {
	return selectElement{Value: value, Options: options}
}

// Component creates an element that wraps a Widget. The Reconciler
// expands it by calling Widget.Render with persisted state.
func Component(w Widget) Element {
	return widgetElement{W: w}
}

// ComponentWithKey creates a keyed widget element. The key stabilises the
// widget's UID across re-ordering within the same parent.
func ComponentWithKey(key string, w Widget) Element {
	return widgetElement{W: w, Key: key}
}

// Padding adds inner spacing around a single child (RFC-002 §4.5).
func Padding(insets draw.Insets, child Element) Element {
	return paddingElement{Insets: insets, Child: child}
}

// SizedBox enforces a specific size on a child. If child is omitted,
// it acts as an empty spacer with the given dimensions (RFC-002 §4.5).
func SizedBox(width, height float32, child ...Element) Element {
	var c Element
	if len(child) > 0 {
		c = child[0]
	}
	return sizedBoxElement{Width: width, Height: height, Child: c}
}

// Expanded takes all available space on the main axis within a Flex
// parent. An optional flex factor controls the proportion (default 1).
func Expanded(child Element, flex ...float32) Element {
	grow := float32(1)
	if len(flex) > 0 {
		grow = flex[0]
	}
	return expandedElement{Child: child, Grow: grow}
}

// ── Concrete element structs ─────────────────────────────────────

type emptyElement struct{}

func (emptyElement) isElement() {}

type textElement struct {
	Content string
	Style   draw.TextStyle // zero value = use tokens.Typography.Body
}

func (textElement) isElement() {}

type buttonElement struct {
	Label   string
	OnClick func()
}

func (buttonElement) isElement() {}

type boxElement struct {
	Axis     LayoutAxis
	Children []Element
}

func (boxElement) isElement() {}

type keyedElement struct {
	Key   string
	Child Element
}

func (keyedElement) isElement() {}

type dividerElement struct{}

func (dividerElement) isElement() {}

type spacerElement struct{ Size float32 }

func (spacerElement) isElement() {}

type iconElement struct {
	Name string
	Size float32 // 0 = use theme Label size
}

func (iconElement) isElement() {}

type stackElement struct{ Children []Element }

func (stackElement) isElement() {}

type scrollViewElement struct {
	Child     Element
	MaxHeight float32
	State     *ScrollState // optional; drives vertical offset
}

func (scrollViewElement) isElement() {}

type paddingElement struct {
	Insets draw.Insets
	Child  Element
}

func (paddingElement) isElement() {}

type sizedBoxElement struct {
	Width, Height float32
	Child         Element // nil = empty spacer
}

func (sizedBoxElement) isElement() {}

type expandedElement struct {
	Child Element
	Grow  float32
}

func (expandedElement) isElement() {}

// ── Tier 2 element structs ──────────────────────────────────────

// ── Tier 3 element structs ──────────────────────────────────────

type cardElement struct {
	Child Element
}

func (cardElement) isElement() {}

// Card creates a container with elevated surface, border, and card radius.
func Card(children ...Element) Element {
	if len(children) == 1 {
		return cardElement{Child: children[0]}
	}
	return cardElement{Child: Column(children...)}
}

// TabItem defines a single tab with an arbitrary header Element and content.
type TabItem struct {
	Header  Element // arbitrary widget content (Icon + Text + Badge etc.)
	Content Element
}

type tabsElement struct {
	Items    []TabItem
	Selected int
	OnSelect func(int)
}

func (tabsElement) isElement() {}

// Tabs creates a tabbed container with arbitrary Element headers.
func Tabs(items []TabItem, selected int, onSelect func(int)) Element {
	return tabsElement{Items: items, Selected: selected, OnSelect: onSelect}
}

// AccordionSection defines a collapsible section with header and content.
type AccordionSection struct {
	Header  Element
	Content Element
}

// AccordionState tracks which accordion sections are expanded.
type AccordionState struct {
	Expanded map[int]bool
}

// NewAccordionState creates a ready-to-use AccordionState.
func NewAccordionState() *AccordionState {
	return &AccordionState{Expanded: make(map[int]bool)}
}

type accordionElement struct {
	Sections []AccordionSection
	State    *AccordionState
}

func (accordionElement) isElement() {}

// Accordion creates a collapsible section container.
func Accordion(sections []AccordionSection, state *AccordionState) Element {
	return accordionElement{Sections: sections, State: state}
}

type tooltipElement struct {
	Trigger Element
	Content Element // arbitrary widget content
	Visible bool    // controlled by hover state or explicit flag
}

func (tooltipElement) isElement() {}

// Tooltip creates an element with a hover popup. Content is arbitrary.
func Tooltip(trigger, content Element) Element {
	return tooltipElement{Trigger: trigger, Content: content}
}

// TooltipVisible creates a tooltip with explicit visibility control.
func TooltipVisible(trigger, content Element, visible bool) Element {
	return tooltipElement{Trigger: trigger, Content: content, Visible: visible}
}

type badgeElement struct {
	Content Element
}

func (badgeElement) isElement() {}

// Badge creates a small pill-shaped indicator with arbitrary Element content.
func Badge(content Element) Element {
	return badgeElement{Content: content}
}

// BadgeText is a convenience for text-only badges.
func BadgeText(label string) Element {
	return badgeElement{Content: Text(label)}
}

type chipElement struct {
	Label     Element
	Selected  bool
	OnClick   func()
	OnDismiss func() // if non-nil, shows dismiss "×" button
}

func (chipElement) isElement() {}

// Chip creates a compact selectable element with arbitrary label content.
func Chip(label Element, selected bool, onClick func()) Element {
	return chipElement{Label: label, Selected: selected, OnClick: onClick}
}

// ChipDismissible creates a dismissible chip with a "×" button.
func ChipDismissible(label Element, selected bool, onClick, onDismiss func()) Element {
	return chipElement{Label: label, Selected: selected, OnClick: onClick, OnDismiss: onDismiss}
}

// MenuItem defines an item in a MenuBar or ContextMenu.
type MenuItem struct {
	Label   Element
	OnClick func()
	Items   []MenuItem // sub-items (nested menus)
}

type menuBarElement struct {
	Items []MenuItem
}

func (menuBarElement) isElement() {}

// MenuBar creates a horizontal menu bar.
func MenuBar(items []MenuItem) Element {
	return menuBarElement{Items: items}
}

type contextMenuElement struct {
	Items   []MenuItem
	Visible bool
	PosX    float32
	PosY    float32
}

func (contextMenuElement) isElement() {}

// ContextMenu creates a floating context menu at the given position.
func ContextMenu(items []MenuItem, visible bool, x, y float32) Element {
	return contextMenuElement{Items: items, Visible: visible, PosX: x, PosY: y}
}

// ── Tier 2 element structs (continued) ──────────────────────────

type checkboxElement struct {
	Label    string
	Checked  bool
	OnToggle func(bool)
}

func (checkboxElement) isElement() {}

type radioElement struct {
	Label    string
	Selected bool
	OnSelect func()
}

func (radioElement) isElement() {}

type toggleElement struct {
	On       bool
	OnToggle func(bool)
	State    *ToggleState
}

func (toggleElement) isElement() {}

type sliderElement struct {
	Value    float32
	OnChange func(float32)
}

func (sliderElement) isElement() {}

type progressBarElement struct {
	Value         float32
	Indeterminate bool
	Phase         float32 // 0.0–1.0, drives indeterminate animation position
}

func (progressBarElement) isElement() {}

type textFieldElement struct {
	Value       string
	Placeholder string
	OnChange    func(string)
	Focus       *FocusState
	FocusID     int // assigned during layout
}

func (textFieldElement) isElement() {}

type selectElement struct {
	Value   string
	Options []string
}

func (selectElement) isElement() {}

// widgetElement wraps a Widget for embedding in element trees.
// It is expanded by the Reconciler before layout.
type widgetElement struct {
	W   Widget
	Key string
}

func (widgetElement) isElement() {}

// ScrollState tracks scroll offset for ScrollView elements.
type ScrollState struct {
	Offset   float32 // current vertical scroll offset in dp
	Velocity float32 // scroll velocity for momentum
}

// ScrollBy adjusts the scroll offset, clamping to [0, maxScroll].
func (s *ScrollState) ScrollBy(delta float32, contentHeight, viewportHeight float32) {
	s.Offset -= delta
	maxScroll := contentHeight - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.Offset < 0 {
		s.Offset = 0
	}
	if s.Offset > maxScroll {
		s.Offset = maxScroll
	}
}

// ── Focus State ──────────────────────────────────────────────────

// InputState tracks the focused TextField's value and callback so that
// the framework can handle KeyMsg/CharMsg internally without exposing
// raw input events to userland.
type InputState struct {
	Value    string
	OnChange func(string)
	FocusID  int
}

// FocusState tracks which element has keyboard focus.
// It is shared between the element tree and the app loop.
type FocusState struct {
	FocusedID int         // ID of focused element, 0 = none
	Input     *InputState // populated during layout for the focused TextField
	nextID    int         // counter for assigning IDs during layout
}

// IsFocused returns true if the element with the given ID has focus.
func (f *FocusState) IsFocused(id int) bool {
	return f != nil && f.FocusedID == id
}

// SetFocused sets the focused element.
func (f *FocusState) SetFocused(id int) {
	if f != nil {
		f.FocusedID = id
	}
}

// Blur removes focus from all elements.
func (f *FocusState) Blur() {
	if f != nil {
		f.FocusedID = 0
	}
}

// nextFocusID assigns and returns the next focus ID during layout.
func (f *FocusState) nextFocusID() int {
	if f == nil {
		return 0
	}
	f.nextID++
	return f.nextID
}

// resetCounter resets the ID counter for a new layout pass.
func (f *FocusState) resetCounter() {
	if f != nil {
		f.nextID = 0
		f.Input = nil
	}
}

// handleKeyMsg processes a key event for the focused TextField.
func handleKeyMsg(focus *FocusState, key string, value string, onChange func(string)) {
	if focus == nil || onChange == nil {
		return
	}
	switch key {
	case "Backspace":
		if len(value) > 0 {
			// Remove last rune.
			runes := []rune(value)
			onChange(string(runes[:len(runes)-1]))
		}
	case "Escape":
		focus.Blur()
	}
}

// handleCharInput appends a character to the value of a focused TextField.
func handleCharInput(ch rune, value string, onChange func(string)) {
	if onChange == nil {
		return
	}
	if ch >= 32 { // printable characters only
		onChange(value + string(ch))
	}
}

// ── Toggle State ─────────────────────────────────────────────────

// ToggleState tracks the toggle thumb animation.
type ToggleState struct {
	thumbPos anim.Anim[float32] // 0.0 = off, 1.0 = on
	lastOn   bool
	inited   bool
}

// NewToggleState creates a ready-to-use ToggleState.
func NewToggleState() *ToggleState { return &ToggleState{} }

// update returns the current animation progress [0,1] and starts a
// new transition if the on state has changed.
func (ts *ToggleState) update(on bool, dur time.Duration) float32 {
	if !ts.inited {
		if on {
			ts.thumbPos.SetImmediate(1.0)
		}
		ts.lastOn = on
		ts.inited = true
		return ts.thumbPos.Value()
	}
	if on != ts.lastOn {
		target := float32(0)
		if on {
			target = 1
		}
		ts.thumbPos.SetTarget(target, dur, anim.OutCubic)
		ts.lastOn = on
	}
	return ts.thumbPos.Value()
}

// Tick advances the toggle animation by dt.
func (ts *ToggleState) Tick(dt time.Duration) {
	if ts != nil {
		ts.thumbPos.Tick(dt)
	}
}

// ── Hover State (M4) ────────────────────────────────────────────

// HoverState tracks hover animations for interactive elements.
// It uses the previous frame's hit targets to determine hover,
// introducing at most one frame of latency (imperceptible at 60fps).
type HoverState struct {
	hoveredIdx int                  // currently hovered button index, -1 = none
	anims      []anim.Anim[float32] // per-button hover opacity [0,1]
	buttonIdx  int                  // counter during BuildScene
	inited     bool                 // tracks whether hoveredIdx has been set
}

// SetHovered updates which button (by index) is hovered and sets animation targets.
// idx == -1 means no button is hovered. dur is the animation duration.
func (h *HoverState) SetHovered(idx int, dur time.Duration) {
	if !h.inited {
		h.hoveredIdx = -1
		h.inited = true
	}
	prev := h.hoveredIdx
	h.hoveredIdx = idx

	// Animate previous button out.
	if prev >= 0 && prev < len(h.anims) && prev != idx {
		h.anims[prev].SetTarget(0.0, dur, anim.OutCubic)
	}

	// Animate new button in.
	if idx >= 0 {
		h.ensureSize(idx + 1)
		if h.anims[idx].Value() < 1.0 {
			h.anims[idx].SetTarget(1.0, dur, anim.OutCubic)
		}
	}
}

// Tick advances all hover animations by dt.
func (h *HoverState) Tick(dt time.Duration) {
	for i := range h.anims {
		h.anims[i].Tick(dt)
	}
}

// resetCounter prepares for a new BuildScene pass.
func (h *HoverState) resetCounter() { h.buttonIdx = 0 }

// nextButtonHoverOpacity returns the hover opacity for the current button
// and advances the internal counter.
func (h *HoverState) nextButtonHoverOpacity() float32 {
	idx := h.buttonIdx
	h.buttonIdx++
	h.ensureSize(h.buttonIdx)
	return h.anims[idx].Value()
}

func (h *HoverState) ensureSize(n int) {
	for len(h.anims) < n {
		h.anims = append(h.anims, anim.Anim[float32]{})
	}
}

// ── Overlay System (Tier 3) ──────────────────────────────────────

// overlayEntry is a deferred render operation drawn after the main tree.
// Used by Tooltip, ContextMenu, and MenuBar for correct Z-order.
type overlayEntry struct {
	Render func(canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map)
}

// overlayStack collects overlay entries during layout.
type overlayStack struct {
	entries []overlayEntry
}

func (s *overlayStack) push(entry overlayEntry) {
	if s != nil {
		s.entries = append(s.entries, entry)
	}
}

// ── Layout & Scene Building ──────────────────────────────────────
// BuildScene converts an Element tree into draw commands via the
// Canvas interface (RFC §6).

type bounds struct{ X, Y, W, H int }

const (
	framePadding   = 24
	columnGap      = 16
	rowGap         = 12
	buttonPadX     = 18
	buttonPadY     = 12
	buttonMinWidth = 180
	buttonBorder   = 1
)

// BuildScene lays out the element tree and paints it to the canvas.
// It returns the accumulated Scene. If hitMap is non-nil, clickable
// element bounds are registered for hit-testing (M3+).
// If hover is non-nil, hover animations are applied to buttons (M4).
// BuildScene lays out the element tree and paints it to the canvas.
// It returns the accumulated Scene. If hitMap is non-nil, clickable
// element bounds are registered for hit-testing (M3+).
// If hover is non-nil, hover animations are applied to buttons (M4).
// If focus is non-nil, text fields use it for keyboard focus tracking.
func BuildScene(root Element, canvas draw.Canvas, th theme.Theme, width, height int, hitMap *hit.Map, hover *HoverState, focusOpt ...*FocusState) draw.Scene {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}

	if hover != nil {
		hover.resetCounter()
	}

	var focus *FocusState
	if len(focusOpt) > 0 {
		focus = focusOpt[0]
	}
	if focus != nil {
		focus.resetCounter()
	}

	tokens := th.Tokens()
	area := bounds{X: framePadding, Y: framePadding, W: max(width-(framePadding*2), 0), H: max(height-(framePadding*2), 0)}
	var overlays overlayStack
	layoutElement(root, area, canvas, tokens, hitMap, hover, &overlays, focus)

	// Render overlay entries (Tooltip, ContextMenu, etc.) on top of main tree.
	for _, entry := range overlays.entries {
		entry.Render(canvas, tokens, hitMap)
	}

	// The canvas is a SceneCanvas — retrieve its scene.
	type scener interface{ Scene() draw.Scene }
	if sc, ok := canvas.(scener); ok {
		return sc.Scene()
	}
	return draw.Scene{}
}

func layoutElement(el Element, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus ...*FocusState) bounds {
	var fs *FocusState
	if len(focus) > 0 {
		fs = focus[0]
	}
	switch node := el.(type) {
	case nil, emptyElement, widgetElement:
		// widgetElement should be resolved by the Reconciler before layout.
		return bounds{X: area.X, Y: area.Y}

	case keyedElement:
		return layoutElement(node.Child, area, canvas, tokens, hitMap, hover, overlays, fs)

	case textElement:
		style := tokens.Typography.Body
		if node.Style.Size > 0 {
			style = node.Style
		}
		metrics := canvas.MeasureText(node.Content, style)
		w := int(math.Ceil(float64(metrics.Width)))
		h := int(math.Ceil(float64(metrics.Ascent)))
		canvas.DrawText(node.Content, draw.Pt(float32(area.X), float32(area.Y)), style, tokens.Colors.Text.Primary)
		return bounds{X: area.X, Y: area.Y, W: w, H: h}

	case buttonElement:
		style := tokens.Typography.Label
		metrics := canvas.MeasureText(node.Label, style)
		labelW := int(math.Ceil(float64(metrics.Width)))
		labelH := int(math.Ceil(float64(metrics.Ascent)))
		w := max(buttonMinWidth, labelW+(buttonPadX*2))
		h := labelH + (buttonPadY * 2)

		// Edge (border)
		canvas.FillRoundRect(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			tokens.Radii.Button, draw.SolidPaint(tokens.Colors.Stroke.Border))

		// Fill — blend with hover overlay (M4).
		fillColor := tokens.Colors.Accent.Primary
		var hoverOpacity float32
		if hover != nil {
			hoverOpacity = hover.nextButtonHoverOpacity()
		}
		if hoverOpacity > 0 {
			fillColor = lerpColor(fillColor, hoverHighlight(fillColor), hoverOpacity)
		}
		canvas.FillRoundRect(draw.R(float32(area.X+buttonBorder), float32(area.Y+buttonBorder),
			float32(max(w-buttonBorder*2, 0)), float32(max(h-buttonBorder*2, 0))),
			maxf(tokens.Radii.Button-float32(buttonBorder), 0), draw.SolidPaint(fillColor))

		// Label, centered
		canvas.DrawText(node.Label,
			draw.Pt(float32(area.X+(w-labelW)/2), float32(area.Y+(h-labelH)/2)),
			style, tokens.Colors.Text.OnAccent)

		// Register hit target for click handling (M3).
		if hitMap != nil {
			hitMap.Add(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)), node.OnClick)
		}

		return bounds{X: area.X, Y: area.Y, W: w, H: h}

	case dividerElement:
		h := 1
		canvas.FillRect(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(h)),
			draw.SolidPaint(tokens.Colors.Stroke.Divider))
		return bounds{X: area.X, Y: area.Y, W: area.W, H: h}

	case spacerElement:
		s := int(node.Size)
		return bounds{X: area.X, Y: area.Y, W: s, H: s}

	case iconElement:
		size := node.Size
		if size == 0 {
			size = tokens.Typography.Label.Size
		}
		// Use the Phosphor icon font for icon elements.
		// Render into a fixed square cell so all icons have uniform size
		// regardless of individual glyph bounding boxes.
		style := draw.TextStyle{
			FontFamily: "Phosphor",
			Size:       size,
			Weight:     draw.FontWeightRegular,
			LineHeight: 1.0,
		}
		cellSize := int(math.Ceil(float64(size)))
		metrics := canvas.MeasureText(node.Name, style)
		offsetX := (float32(cellSize) - metrics.Width) / 2
		offsetY := (float32(cellSize) - metrics.Ascent) / 2
		canvas.DrawText(node.Name, draw.Pt(float32(area.X)+offsetX, float32(area.Y)+offsetY), style, tokens.Colors.Text.Primary)
		return bounds{X: area.X, Y: area.Y, W: cellSize, H: cellSize}

	case stackElement:
		return layoutStack(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case scrollViewElement:
		return layoutScrollView(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case boxElement:
		return layoutBox(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case paddingElement:
		return layoutPadding(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case sizedBoxElement:
		return layoutSizedBox(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case expandedElement:
		// Outside a Flex context, Expanded passes through to its child.
		return layoutElement(node.Child, area, canvas, tokens, hitMap, hover, overlays, fs)

	case flexElement:
		return layoutFlex(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case gridElement:
		return layoutGrid(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case virtualListElement:
		return layoutVirtualList(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case treeElement:
		return layoutTree(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	case richTextElement:
		return layoutRichText(node, area, canvas, tokens)

	// Tier 2 widgets
	case checkboxElement:
		return layoutCheckbox(node, area, canvas, tokens, hitMap, hover)
	case radioElement:
		return layoutRadio(node, area, canvas, tokens, hitMap, hover)
	case toggleElement:
		return layoutToggle(node, area, canvas, tokens, hitMap, hover)
	case sliderElement:
		return layoutSlider(node, area, canvas, tokens, hitMap, hover)
	case progressBarElement:
		return layoutProgressBar(node, area, canvas, tokens)
	case textFieldElement:
		return layoutTextField(node, area, canvas, tokens, hitMap, hover, fs)
	case selectElement:
		return layoutSelect(node, area, canvas, tokens)

	// Tier 3 widgets
	case cardElement:
		return layoutCard(node, area, canvas, tokens, hitMap, hover, overlays, fs)
	case tabsElement:
		return layoutTabs(node, area, canvas, tokens, hitMap, hover, overlays, fs)
	case accordionElement:
		return layoutAccordion(node, area, canvas, tokens, hitMap, hover, overlays, fs)
	case tooltipElement:
		return layoutTooltip(node, area, canvas, tokens, hitMap, hover, overlays, fs)
	case badgeElement:
		return layoutBadge(node, area, canvas, tokens, hitMap, hover, overlays, fs)
	case chipElement:
		return layoutChip(node, area, canvas, tokens, hitMap, hover, overlays, fs)
	case menuBarElement:
		return layoutMenuBar(node, area, canvas, tokens, hitMap, hover, overlays, fs)
	case contextMenuElement:
		return layoutContextMenu(node, area, canvas, tokens, hitMap, hover, overlays, fs)

	default:
		return bounds{X: area.X, Y: area.Y}
	}
}

// hoverHighlight returns a lightened version of c for hover feedback.
func hoverHighlight(c draw.Color) draw.Color {
	return draw.Color{
		R: c.R + (1-c.R)*0.2,
		G: c.G + (1-c.G)*0.2,
		B: c.B + (1-c.B)*0.2,
		A: c.A,
	}
}

// lerpColor linearly interpolates between two colors.
func lerpColor(a, b draw.Color, t float32) draw.Color {
	return draw.Color{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A + (b.A-a.A)*t,
	}
}

func layoutBox(node boxElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus ...*FocusState) bounds {
	var fs *FocusState
	if len(focus) > 0 {
		fs = focus[0]
	}
	cursorX := area.X
	cursorY := area.Y
	maxW := 0
	maxH := 0
	count := 0

	for _, child := range node.Children {
		childW := area.W
		if node.Axis == AxisRow {
			// Give each row child only the remaining width so that
			// children like ScrollView / VirtualList clip correctly.
			childW = area.X + area.W - cursorX
			if childW < 0 {
				childW = 0
			}
		}
		childBounds := layoutElement(child, bounds{X: cursorX, Y: cursorY, W: childW, H: area.H}, canvas, tokens, hitMap, hover, overlays, fs)
		if childBounds.W == 0 && childBounds.H == 0 {
			continue
		}
		count++
		if node.Axis == AxisRow {
			cursorX += childBounds.W + rowGap
			maxW = max(maxW, cursorX-area.X-rowGap)
			maxH = max(maxH, childBounds.H)
		} else {
			cursorY += childBounds.H + columnGap
			maxW = max(maxW, childBounds.W)
			maxH = max(maxH, cursorY-area.Y-columnGap)
		}
	}

	if count == 0 {
		return bounds{X: area.X, Y: area.Y}
	}
	return bounds{X: area.X, Y: area.Y, W: maxW, H: maxH}
}

func layoutStack(node stackElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus ...*FocusState) bounds {
	var fs *FocusState
	if len(focus) > 0 {
		fs = focus[0]
	}
	maxW := 0
	maxH := 0
	for _, child := range node.Children {
		childBounds := layoutElement(child, area, canvas, tokens, hitMap, hover, overlays, fs)
		maxW = max(maxW, childBounds.W)
		maxH = max(maxH, childBounds.H)
	}
	if maxW == 0 && maxH == 0 {
		return bounds{X: area.X, Y: area.Y}
	}
	return bounds{X: area.X, Y: area.Y, W: maxW, H: maxH}
}

func layoutScrollView(node scrollViewElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus ...*FocusState) bounds {
	var fs *FocusState
	if len(focus) > 0 {
		fs = focus[0]
	}
	viewportH := int(node.MaxHeight)
	if viewportH <= 0 || viewportH > area.H {
		viewportH = area.H
	}

	// Determine scroll offset from state.
	var offset float32
	if node.State != nil {
		offset = node.State.Offset
	}

	// Clip to viewport.
	canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(viewportH)))

	// Render child offset by -offset in Y so content scrolls upward.
	childArea := bounds{X: area.X, Y: area.Y - int(offset), W: area.W, H: area.H + int(offset)}
	childBounds := layoutElement(node.Child, childArea, canvas, tokens, hitMap, hover, overlays, fs)

	canvas.PopClip()

	contentH := childBounds.H
	w := max(childBounds.W, area.W)

	// Clamp state if provided (ensures offset stays valid after content changes).
	if node.State != nil {
		maxScroll := float32(contentH) - float32(viewportH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if node.State.Offset > maxScroll {
			node.State.Offset = maxScroll
		}
		if node.State.Offset < 0 {
			node.State.Offset = 0
		}
	}

	// Register the viewport as a scroll target so the framework can
	// route mouse-wheel events directly to the ScrollState.
	if hitMap != nil && node.State != nil && contentH > viewportH {
		state := node.State
		cH := float32(contentH)
		vH := float32(viewportH)
		hitMap.AddScroll(
			draw.R(float32(area.X), float32(area.Y), float32(w), float32(viewportH)),
			cH, vH,
			func(deltaY float32) {
				state.ScrollBy(deltaY, cH, vH)
			},
		)
	}

	// Draw scrollbar if content exceeds viewport.
	if contentH > viewportH {
		w += drawScrollbar(canvas, tokens, hitMap, node.State, area.X+w, area.Y, viewportH, float32(contentH), offset)
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: viewportH}
}

// ── Tier 2 Layout Constants ──────────────────────────────────────

const (
	checkboxSize   = 16
	checkboxGap    = 8
	checkboxBorder = 1

	toggleTrackW   = 36
	toggleTrackH   = 20
	toggleThumbD   = 16
	toggleThumbPad = 2

	sliderTrackH   = 4
	sliderHeight   = 20
	sliderThumbD   = 16
	sliderMaxWidth = 200

	progressBarH    = 6
	progressBarMaxW = 200

	textFieldW    = 200
	textFieldPadX = 8
	textFieldPadY = 8
)

// ── Tier 2 Layout Functions ─────────────────────────────────────

func layoutCheckbox(node checkboxElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState) bounds {
	style := tokens.Typography.Body
	metrics := canvas.MeasureText(node.Label, style)
	labelW := int(math.Ceil(float64(metrics.Width)))
	labelH := int(math.Ceil(float64(metrics.Ascent)))
	totalH := max(checkboxSize, labelH)
	totalW := checkboxSize + checkboxGap + labelW

	// Hover
	var hoverOpacity float32
	if hover != nil {
		hoverOpacity = hover.nextButtonHoverOpacity()
	}

	boxY := area.Y + (totalH-checkboxSize)/2

	// Border
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(boxY), float32(checkboxSize), float32(checkboxSize)),
		tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Stroke.Border))

	// Fill
	fillColor := tokens.Colors.Surface.Elevated
	if hoverOpacity > 0 {
		fillColor = lerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
	}
	if node.Checked {
		fillColor = tokens.Colors.Accent.Primary
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+checkboxBorder), float32(boxY+checkboxBorder),
			float32(checkboxSize-checkboxBorder*2), float32(checkboxSize-checkboxBorder*2)),
		maxf(tokens.Radii.Input-checkboxBorder, 0), draw.SolidPaint(fillColor))

	// Checkmark
	if node.Checked {
		checkStyle := draw.TextStyle{
			Size:   float32(checkboxSize - checkboxBorder*2 - 2),
			Weight: draw.FontWeightBold,
		}
		canvas.DrawText("✓",
			draw.Pt(float32(area.X+checkboxBorder+1), float32(boxY+checkboxBorder+1)),
			checkStyle, tokens.Colors.Text.OnAccent)
	}

	// Label
	labelX := area.X + checkboxSize + checkboxGap
	labelY := area.Y + (totalH-labelH)/2
	canvas.DrawText(node.Label, draw.Pt(float32(labelX), float32(labelY)), style, tokens.Colors.Text.Primary)

	// Hit target
	if hitMap != nil && node.OnToggle != nil {
		checked := node.Checked
		onToggle := node.OnToggle
		hitMap.Add(draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(totalH)),
			func() { onToggle(!checked) })
	}

	return bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

func layoutRadio(node radioElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState) bounds {
	style := tokens.Typography.Body
	metrics := canvas.MeasureText(node.Label, style)
	labelW := int(math.Ceil(float64(metrics.Width)))
	labelH := int(math.Ceil(float64(metrics.Ascent)))
	totalH := max(checkboxSize, labelH)
	totalW := checkboxSize + checkboxGap + labelW

	// Hover
	var hoverOpacity float32
	if hover != nil {
		hoverOpacity = hover.nextButtonHoverOpacity()
	}

	boxY := area.Y + (totalH-checkboxSize)/2

	// Outer circle
	canvas.FillEllipse(
		draw.R(float32(area.X), float32(boxY), float32(checkboxSize), float32(checkboxSize)),
		draw.SolidPaint(tokens.Colors.Stroke.Border))

	// Inner fill
	fillColor := tokens.Colors.Surface.Elevated
	if hoverOpacity > 0 {
		fillColor = lerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
	}
	canvas.FillEllipse(
		draw.R(float32(area.X+checkboxBorder), float32(boxY+checkboxBorder),
			float32(checkboxSize-checkboxBorder*2), float32(checkboxSize-checkboxBorder*2)),
		draw.SolidPaint(fillColor))

	// Selected dot
	if node.Selected {
		dotSize := 8
		dotOffset := (checkboxSize - dotSize) / 2
		canvas.FillEllipse(
			draw.R(float32(area.X+dotOffset), float32(boxY+dotOffset), float32(dotSize), float32(dotSize)),
			draw.SolidPaint(tokens.Colors.Accent.Primary))
	}

	// Label
	labelX := area.X + checkboxSize + checkboxGap
	labelY := area.Y + (totalH-labelH)/2
	canvas.DrawText(node.Label, draw.Pt(float32(labelX), float32(labelY)), style, tokens.Colors.Text.Primary)

	// Hit target
	if hitMap != nil && node.OnSelect != nil {
		hitMap.Add(draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(totalH)),
			node.OnSelect)
	}

	return bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

func layoutToggle(node toggleElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState) bounds {
	// Hover
	var hoverOpacity float32
	if hover != nil {
		hoverOpacity = hover.nextButtonHoverOpacity()
	}

	// Animation progress: 0 = off, 1 = on.
	var t float32
	if node.State != nil {
		t = node.State.update(node.On, 150*time.Millisecond)
	} else {
		if node.On {
			t = 1
		}
	}

	// Track — lerp between off and on colors.
	// Use exact colors at t=0/1 to avoid float rounding artifacts.
	offTrackColor := tokens.Colors.Surface.Pressed
	onTrackColor := tokens.Colors.Accent.Primary
	var trackColor draw.Color
	switch {
	case t <= 0:
		trackColor = offTrackColor
	case t >= 1:
		trackColor = onTrackColor
	default:
		trackColor = lerpColor(offTrackColor, onTrackColor, t)
	}
	if hoverOpacity > 0 {
		trackColor = lerpColor(trackColor, hoverHighlight(trackColor), hoverOpacity)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(toggleTrackW), float32(toggleTrackH)),
		float32(toggleTrackH)/2, draw.SolidPaint(trackColor))

	// Thumb — lerp position and color.
	offX := float32(area.X + toggleThumbPad)
	onX := float32(area.X + toggleTrackW - toggleThumbD - toggleThumbPad)
	thumbX := offX + (onX-offX)*t
	thumbY := float32(area.Y + (toggleTrackH-toggleThumbD)/2)
	offThumbColor := tokens.Colors.Text.Secondary
	onThumbColor := tokens.Colors.Text.OnAccent
	var thumbColor draw.Color
	switch {
	case t <= 0:
		thumbColor = offThumbColor
	case t >= 1:
		thumbColor = onThumbColor
	default:
		thumbColor = lerpColor(offThumbColor, onThumbColor, t)
	}
	canvas.FillEllipse(
		draw.R(thumbX, thumbY, float32(toggleThumbD), float32(toggleThumbD)),
		draw.SolidPaint(thumbColor))

	// Hit target
	if hitMap != nil && node.OnToggle != nil {
		on := node.On
		onToggle := node.OnToggle
		hitMap.Add(draw.R(float32(area.X), float32(area.Y), float32(toggleTrackW), float32(toggleTrackH)),
			func() { onToggle(!on) })
	}

	return bounds{X: area.X, Y: area.Y, W: toggleTrackW, H: toggleTrackH}
}

func layoutSlider(node sliderElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState) bounds {
	trackW := sliderMaxWidth
	if area.W < trackW {
		trackW = area.W
	}

	// Hover
	var hoverOpacity float32
	if hover != nil {
		hoverOpacity = hover.nextButtonHoverOpacity()
	}

	trackY := area.Y + (sliderHeight-sliderTrackH)/2

	// Track background
	trackColor := tokens.Colors.Surface.Pressed
	if hoverOpacity > 0 {
		trackColor = lerpColor(trackColor, tokens.Colors.Surface.Hovered, hoverOpacity)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(trackY), float32(trackW), float32(sliderTrackH)),
		float32(sliderTrackH)/2, draw.SolidPaint(trackColor))

	// Filled portion
	val := node.Value
	if val < 0 {
		val = 0
	}
	if val > 1 {
		val = 1
	}
	filledW := int(float32(trackW) * val)
	if filledW > 0 {
		canvas.FillRoundRect(
			draw.R(float32(area.X), float32(trackY), float32(filledW), float32(sliderTrackH)),
			float32(sliderTrackH)/2, draw.SolidPaint(tokens.Colors.Accent.Primary))
	}

	// Thumb
	thumbX := area.X + filledW - sliderThumbD/2
	if thumbX < area.X {
		thumbX = area.X
	}
	thumbY := area.Y + (sliderHeight-sliderThumbD)/2
	canvas.FillEllipse(
		draw.R(float32(thumbX), float32(thumbY), float32(sliderThumbD), float32(sliderThumbD)),
		draw.SolidPaint(tokens.Colors.Accent.Primary))

	// Hit target (draggable positional)
	if hitMap != nil && node.OnChange != nil {
		areaX := float32(area.X)
		tw := float32(trackW)
		onChange := node.OnChange
		hitMap.AddDrag(draw.R(float32(area.X), float32(area.Y), float32(trackW), float32(sliderHeight)),
			func(x, _ float32) {
				v := (x - areaX) / tw
				if v < 0 {
					v = 0
				}
				if v > 1 {
					v = 1
				}
				onChange(v)
			})
	}

	return bounds{X: area.X, Y: area.Y, W: trackW, H: sliderHeight}
}

func layoutProgressBar(node progressBarElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet) bounds {
	trackW := progressBarMaxW
	if area.W < trackW {
		trackW = area.W
	}

	// Track
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(trackW), float32(progressBarH)),
		float32(progressBarH)/2, draw.SolidPaint(tokens.Colors.Surface.Pressed))

	if node.Indeterminate {
		// Animated 30% bar that slides across the track.
		barW := int(float32(trackW) * 0.3)
		// Phase 0→1 maps to the bar sliding from left to right.
		phase := node.Phase
		if phase < 0 {
			phase = 0
		}
		if phase > 1 {
			phase -= float32(int(phase)) // wrap
		}
		travel := trackW - barW
		barX := area.X + int(float32(travel)*phase)
		canvas.FillRoundRect(
			draw.R(float32(barX), float32(area.Y), float32(barW), float32(progressBarH)),
			float32(progressBarH)/2, draw.SolidPaint(tokens.Colors.Accent.Primary))
	} else {
		// Determinate fill
		val := node.Value
		if val < 0 {
			val = 0
		}
		if val > 1 {
			val = 1
		}
		filledW := int(float32(trackW) * val)
		if filledW > 0 {
			canvas.FillRoundRect(
				draw.R(float32(area.X), float32(area.Y), float32(filledW), float32(progressBarH)),
				float32(progressBarH)/2, draw.SolidPaint(tokens.Colors.Accent.Primary))
		}
	}

	return bounds{X: area.X, Y: area.Y, W: trackW, H: progressBarH}
}

func layoutTextField(node textFieldElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, focus *FocusState) bounds {
	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := textFieldW
	if area.W < w {
		w = area.W
	}

	// Assign a focus ID if focus state is provided.
	focusID := 0
	if focus != nil {
		focusID = focus.nextFocusID()
	}
	focused := focus.IsFocused(focusID)

	// Border — highlight when focused.
	borderColor := tokens.Colors.Stroke.Border
	if focused {
		borderColor = tokens.Colors.Accent.Primary
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
		tokens.Radii.Input, draw.SolidPaint(borderColor))

	// Fill
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Text or placeholder
	textX := area.X + textFieldPadX
	textY := area.Y + textFieldPadY
	if node.Value != "" {
		canvas.DrawText(node.Value, draw.Pt(float32(textX), float32(textY)), style, tokens.Colors.Text.Primary)
	} else if node.Placeholder != "" {
		canvas.DrawText(node.Placeholder, draw.Pt(float32(textX), float32(textY)), style, tokens.Colors.Text.Disabled)
	}

	// Cursor when focused
	if focused {
		metrics := canvas.MeasureText(node.Value, style)
		cursorX := float32(textX) + metrics.Width
		canvas.FillRect(draw.R(cursorX, float32(textY), 2, style.Size),
			draw.SolidPaint(tokens.Colors.Text.Primary))
	}

	// Store input state for the focused TextField so the framework can
	// handle KeyMsg/CharMsg internally (no userland boilerplate needed).
	if focused && node.OnChange != nil && focus != nil {
		focus.Input = &InputState{
			Value:    node.Value,
			OnChange: node.OnChange,
			FocusID:  focusID,
		}
	}

	// Hit target for focus acquisition.
	// Consume a hover slot to keep hit-target indices aligned with hover indices.
	if node.OnChange != nil && focus != nil {
		if hover != nil {
			hover.nextButtonHoverOpacity()
		}
		if hitMap != nil {
			fid := focusID
			fs := focus
			hitMap.Add(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
				func() { fs.SetFocused(fid) })
		}
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutSelect(node selectElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet) bounds {
	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := textFieldW
	if area.W < w {
		w = area.W
	}

	// Border
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
		tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Stroke.Border))

	// Fill
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Value text
	textX := area.X + textFieldPadX
	textY := area.Y + textFieldPadY
	if node.Value != "" {
		canvas.DrawText(node.Value, draw.Pt(float32(textX), float32(textY)), style, tokens.Colors.Text.Primary)
	}

	// Down arrow indicator
	arrowStyle := tokens.Typography.LabelSmall
	arrowX := area.X + w - textFieldPadX - int(arrowStyle.Size)
	canvas.DrawText("▾", draw.Pt(float32(arrowX), float32(textY)), arrowStyle, tokens.Colors.Text.Secondary)

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutPadding(node paddingElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus ...*FocusState) bounds {
	var fs *FocusState
	if len(focus) > 0 {
		fs = focus[0]
	}
	inL := int(node.Insets.Left)
	inT := int(node.Insets.Top)
	inR := int(node.Insets.Right)
	inB := int(node.Insets.Bottom)
	childArea := bounds{
		X: area.X + inL,
		Y: area.Y + inT,
		W: max(area.W-inL-inR, 0),
		H: max(area.H-inT-inB, 0),
	}
	cb := layoutElement(node.Child, childArea, canvas, tokens, hitMap, hover, overlays, fs)
	return bounds{X: area.X, Y: area.Y, W: cb.W + inL + inR, H: cb.H + inT + inB}
}

func layoutSizedBox(node sizedBoxElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus ...*FocusState) bounds {
	var fs *FocusState
	if len(focus) > 0 {
		fs = focus[0]
	}
	w := int(node.Width)
	h := int(node.Height)
	if node.Child != nil {
		childArea := bounds{X: area.X, Y: area.Y, W: w, H: h}
		layoutElement(node.Child, childArea, canvas, tokens, hitMap, hover, overlays, fs)
	}
	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── Tier 3 Layout Constants ──────────────────────────────────────

const (
	cardPadding     = 16
	cardBorder      = 1
	tabHeaderPadX   = 16
	tabHeaderPadY   = 10
	tabIndicatorH   = 2
	accordionHeaderH = 36
	tooltipPadding  = 8
	badgePadX       = 6
	badgePadY       = 2
	badgeMinSize    = 20
	chipPadX        = 12
	chipPadY        = 6
	chipDismissW    = 16
	menuBarHeight   = 32
	menuBarItemPadX = 12
	menuItemHeight  = 32
	menuItemPadX    = 12
)

// ── Tier 3 Layout Functions ──────────────────────────────────────

func layoutCard(node cardElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	// Measure child to determine card size.
	nc := nullCanvas{delegate: canvas}
	childArea := bounds{X: area.X + cardPadding, Y: area.Y + cardPadding, W: max(area.W-cardPadding*2, 0), H: max(area.H-cardPadding*2, 0)}
	cb := layoutElement(node.Child, childArea, nc, tokens, nil, nil, nil)

	w := cb.W + cardPadding*2
	h := cb.H + cardPadding*2
	if w > area.W {
		w = area.W
	}

	// Border
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
		tokens.Radii.Card, draw.SolidPaint(tokens.Colors.Stroke.Border))

	// Fill
	canvas.FillRoundRect(
		draw.R(float32(area.X+cardBorder), float32(area.Y+cardBorder), float32(max(w-cardBorder*2, 0)), float32(max(h-cardBorder*2, 0))),
		maxf(tokens.Radii.Card-cardBorder, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Child content
	layoutElement(node.Child, childArea, canvas, tokens, hitMap, hover, overlays, focus)

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutTabs(node tabsElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	if len(node.Items) == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	style := tokens.Typography.Label
	nc := nullCanvas{delegate: canvas}

	// Pass 1: measure all headers to determine tab widths.
	type tabMeasure struct{ w, h int }
	measures := make([]tabMeasure, len(node.Items))
	headerH := 0
	for i, item := range node.Items {
		cb := layoutElement(item.Header, bounds{X: 0, Y: 0, W: area.W, H: area.H}, nc, tokens, nil, nil, nil)
		w := cb.W + tabHeaderPadX*2
		h := cb.H + tabHeaderPadY*2
		measures[i] = tabMeasure{w: w, h: h}
		if h > headerH {
			headerH = h
		}
	}
	_ = style // headers use arbitrary elements; style used for reference only

	// Pass 2: draw tab header row.
	cursorX := area.X
	selected := node.Selected
	if selected < 0 || selected >= len(node.Items) {
		selected = 0
	}

	for i, item := range node.Items {
		tw := measures[i].w

		// Tab background — selected tab gets subtle highlight.
		if i == selected {
			canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH)),
				draw.SolidPaint(tokens.Colors.Surface.Hovered))
		}

		// Tab header content
		headerArea := bounds{X: cursorX + tabHeaderPadX, Y: area.Y + tabHeaderPadY, W: max(tw-tabHeaderPadX*2, 0), H: max(headerH-tabHeaderPadY*2, 0)}
		layoutElement(item.Header, headerArea, canvas, tokens, hitMap, hover, overlays, focus)

		// Selection indicator (underline)
		if i == selected {
			canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y+headerH-tabIndicatorH), float32(tw), float32(tabIndicatorH)),
				draw.SolidPaint(tokens.Colors.Accent.Primary))
		}

		// Hit target
		if hitMap != nil && node.OnSelect != nil {
			idx := i
			onSelect := node.OnSelect
			if hover != nil {
				hover.nextButtonHoverOpacity()
			}
			hitMap.Add(draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH)),
				func() { onSelect(idx) })
		}

		cursorX += tw
	}

	totalHeaderW := cursorX - area.X

	// Divider below headers
	canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y+headerH), float32(max(totalHeaderW, area.W)), 1),
		draw.SolidPaint(tokens.Colors.Stroke.Divider))

	// Selected tab content
	contentY := area.Y + headerH + 1 + columnGap
	contentArea := bounds{X: area.X, Y: contentY, W: area.W, H: max(area.H-headerH-1-columnGap, 0)}
	cb := layoutElement(node.Items[selected].Content, contentArea, canvas, tokens, hitMap, hover, overlays, focus)

	totalH := headerH + 1 + columnGap + cb.H
	totalW := max(totalHeaderW, cb.W)
	return bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

func layoutAccordion(node accordionElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	if len(node.Sections) == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	cursorY := area.Y
	maxW := 0

	chevronStyle := draw.TextStyle{
		Size:   12,
		Weight: draw.FontWeightBold,
	}

	for i, section := range node.Sections {
		expanded := node.State != nil && node.State.Expanded[i]

		// Divider between sections (not before first)
		if i > 0 {
			canvas.FillRect(
				draw.R(float32(area.X), float32(cursorY), float32(area.W), 1),
				draw.SolidPaint(tokens.Colors.Stroke.Divider))
			cursorY++
		}

		// Header background
		canvas.FillRect(
			draw.R(float32(area.X), float32(cursorY), float32(area.W), float32(accordionHeaderH)),
			draw.SolidPaint(tokens.Colors.Surface.Elevated))

		// Chevron indicator
		chevron := "▶"
		if expanded {
			chevron = "▼"
		}
		chevronX := area.X + 8
		chevronY := cursorY + (accordionHeaderH-int(chevronStyle.Size))/2
		canvas.DrawText(chevron, draw.Pt(float32(chevronX), float32(chevronY)), chevronStyle, tokens.Colors.Text.Secondary)

		// Header content
		headerX := area.X + 8 + int(chevronStyle.Size) + 8
		headerArea := bounds{X: headerX, Y: cursorY + (accordionHeaderH-16)/2, W: max(area.W-headerX+area.X, 0), H: 16}
		layoutElement(section.Header, headerArea, canvas, tokens, hitMap, hover, overlays, focus)

		// Hit target for expand/collapse
		if hitMap != nil && node.State != nil {
			idx := i
			state := node.State
			if hover != nil {
				hover.nextButtonHoverOpacity()
			}
			hitMap.Add(draw.R(float32(area.X), float32(cursorY), float32(area.W), float32(accordionHeaderH)),
				func() {
					state.Expanded[idx] = !state.Expanded[idx]
				})
		}

		if area.W > maxW {
			maxW = area.W
		}
		cursorY += accordionHeaderH

		// Content (if expanded)
		if expanded {
			contentArea := bounds{X: area.X + cardPadding, Y: cursorY + 8, W: max(area.W-cardPadding*2, 0), H: max(area.H-(cursorY-area.Y)-8, 0)}
			cb := layoutElement(section.Content, contentArea, canvas, tokens, hitMap, hover, overlays, focus)
			cursorY += cb.H + 16 // 8 top + 8 bottom padding
		}
	}

	return bounds{X: area.X, Y: area.Y, W: maxW, H: cursorY - area.Y}
}

func layoutTooltip(node tooltipElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	// Layout trigger normally.
	triggerBounds := layoutElement(node.Trigger, area, canvas, tokens, hitMap, hover, overlays, focus)

	// If visible, push overlay for the tooltip popup.
	if node.Visible {
		tB := triggerBounds
		content := node.Content
		overlays.push(overlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map) {
				// Measure content
				nc := nullCanvas{delegate: canvas}
				cb := layoutElement(content, bounds{X: 0, Y: 0, W: 300, H: 200}, nc, tokens, nil, nil, nil)

				w := cb.W + tooltipPadding*2
				h := cb.H + tooltipPadding*2
				x := tB.X
				y := tB.Y + tB.H + 4

				// Border
				canvas.FillRoundRect(
					draw.R(float32(x), float32(y), float32(w), float32(h)),
					tokens.Radii.Button, draw.SolidPaint(tokens.Colors.Stroke.Border))

				// Fill
				canvas.FillRoundRect(
					draw.R(float32(x+1), float32(y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
					maxf(tokens.Radii.Button-1, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

				// Content
				layoutElement(content, bounds{X: x + tooltipPadding, Y: y + tooltipPadding, W: max(w-tooltipPadding*2, 0), H: max(h-tooltipPadding*2, 0)}, canvas, tokens, hitMap, nil, nil)
			},
		})
	}

	return triggerBounds
}

func layoutBadge(node badgeElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	// Measure content
	nc := nullCanvas{delegate: canvas}
	cb := layoutElement(node.Content, bounds{X: 0, Y: 0, W: area.W, H: area.H}, nc, tokens, nil, nil, nil)

	w := cb.W + badgePadX*2
	h := cb.H + badgePadY*2
	// Ensure minimum size for circle shape with single characters
	if w < badgeMinSize {
		w = badgeMinSize
	}
	if h < badgeMinSize {
		h = badgeMinSize
	}

	// Pill background
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
		tokens.Radii.Pill, draw.SolidPaint(tokens.Colors.Accent.Primary))

	// Content (centered)
	contentX := area.X + (w-cb.W)/2
	contentY := area.Y + (h-cb.H)/2
	layoutElement(node.Content, bounds{X: contentX, Y: contentY, W: cb.W, H: cb.H}, canvas, tokens, hitMap, hover, overlays, focus)

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutChip(node chipElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	// Measure label
	nc := nullCanvas{delegate: canvas}
	cb := layoutElement(node.Label, bounds{X: 0, Y: 0, W: area.W, H: area.H}, nc, tokens, nil, nil, nil)

	labelW := cb.W
	dismissW := 0
	if node.OnDismiss != nil {
		dismissW = chipDismissW
	}
	w := labelW + chipPadX*2 + dismissW
	h := cb.H + chipPadY*2

	// Hover
	var hoverOpacity float32
	if hover != nil {
		hoverOpacity = hover.nextButtonHoverOpacity()
	}

	// Background
	var bgColor, borderColor draw.Color
	if node.Selected {
		bgColor = tokens.Colors.Accent.Primary
		borderColor = tokens.Colors.Accent.Primary
	} else {
		bgColor = tokens.Colors.Surface.Elevated
		borderColor = tokens.Colors.Stroke.Border
	}
	if hoverOpacity > 0 {
		bgColor = lerpColor(bgColor, hoverHighlight(bgColor), hoverOpacity)
	}

	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
		tokens.Radii.Pill, draw.SolidPaint(borderColor))
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Pill-1, 0), draw.SolidPaint(bgColor))

	// Label content
	labelArea := bounds{X: area.X + chipPadX, Y: area.Y + chipPadY, W: labelW, H: cb.H}
	layoutElement(node.Label, labelArea, canvas, tokens, hitMap, hover, overlays, focus)

	// Dismiss "×"
	if node.OnDismiss != nil {
		dismissX := area.X + chipPadX + labelW + 4
		dismissY := area.Y + chipPadY
		dismissStyle := tokens.Typography.LabelSmall
		textColor := tokens.Colors.Text.Primary
		if node.Selected {
			textColor = tokens.Colors.Text.OnAccent
		}
		canvas.DrawText("×", draw.Pt(float32(dismissX), float32(dismissY)), dismissStyle, textColor)

		if hitMap != nil {
			onDismiss := node.OnDismiss
			hitMap.Add(draw.R(float32(dismissX), float32(area.Y), float32(chipDismissW), float32(h)),
				onDismiss)
		}
	}

	// Hit target for chip click
	if hitMap != nil && node.OnClick != nil {
		onClick := node.OnClick
		clickW := w
		if node.OnDismiss != nil {
			clickW = w - dismissW // exclude dismiss area
		}
		hitMap.Add(draw.R(float32(area.X), float32(area.Y), float32(clickW), float32(h)),
			onClick)
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutMenuBar(node menuBarElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	if len(node.Items) == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	nc := nullCanvas{delegate: canvas}

	// Background strip
	canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(menuBarHeight)),
		draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Bottom border
	canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y+menuBarHeight-1), float32(area.W), 1),
		draw.SolidPaint(tokens.Colors.Stroke.Border))

	cursorX := area.X

	for _, item := range node.Items {
		// Measure label
		cb := layoutElement(item.Label, bounds{X: 0, Y: 0, W: area.W, H: menuBarHeight}, nc, tokens, nil, nil, nil)
		itemW := cb.W + menuBarItemPadX*2

		// Draw label
		labelArea := bounds{X: cursorX + menuBarItemPadX, Y: area.Y + (menuBarHeight-cb.H)/2, W: cb.W, H: cb.H}
		layoutElement(item.Label, labelArea, canvas, tokens, hitMap, hover, overlays, focus)

		// Hit target
		if hitMap != nil && item.OnClick != nil {
			onClick := item.OnClick
			if hover != nil {
				hover.nextButtonHoverOpacity()
			}
			hitMap.Add(draw.R(float32(cursorX), float32(area.Y), float32(itemW), float32(menuBarHeight)),
				onClick)
		}

		cursorX += itemW
	}

	return bounds{X: area.X, Y: area.Y, W: area.W, H: menuBarHeight}
}

func layoutContextMenu(node contextMenuElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	if !node.Visible || len(node.Items) == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	nc := nullCanvas{delegate: canvas}
	items := node.Items
	posX := int(node.PosX)
	posY := int(node.PosY)

	// Push overlay for context menu rendering.
	overlays.push(overlayEntry{
		Render: func(canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map) {
			// Measure all items.
			maxItemW := 0
			for _, item := range items {
				cb := layoutElement(item.Label, bounds{X: 0, Y: 0, W: 300, H: menuItemHeight}, nc, tokens, nil, nil, nil)
				w := cb.W + menuItemPadX*2
				if w > maxItemW {
					maxItemW = w
				}
			}
			if maxItemW < 120 {
				maxItemW = 120
			}

			totalH := len(items) * menuItemHeight
			menuW := maxItemW
			menuH := totalH

			// Border
			canvas.FillRoundRect(
				draw.R(float32(posX), float32(posY), float32(menuW), float32(menuH)),
				tokens.Radii.Card, draw.SolidPaint(tokens.Colors.Stroke.Border))

			// Fill
			canvas.FillRoundRect(
				draw.R(float32(posX+1), float32(posY+1), float32(max(menuW-2, 0)), float32(max(menuH-2, 0))),
				maxf(tokens.Radii.Card-1, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

			// Items
			cursorY := posY
			for _, item := range items {
				labelArea := bounds{X: posX + menuItemPadX, Y: cursorY + (menuItemHeight-16)/2, W: max(menuW-menuItemPadX*2, 0), H: 16}
				layoutElement(item.Label, labelArea, canvas, tokens, hitMap, nil, nil)

				if hitMap != nil && item.OnClick != nil {
					onClick := item.OnClick
					hitMap.Add(draw.R(float32(posX), float32(cursorY), float32(menuW), float32(menuItemHeight)),
						onClick)
				}

				cursorY += menuItemHeight
			}
		},
	})

	return bounds{X: area.X, Y: area.Y}
}
