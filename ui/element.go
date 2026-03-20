// Package ui defines the Widget system and Element types for the
// virtual tree (RFC §4).
package ui

import (
	"math"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/theme"
)

// ── Widget System (RFC §4) ───────────────────────────────────────

// WidgetState is an open interface — any type qualifies (RFC §4.1).
type WidgetState interface{}

// Animator is an optional interface for WidgetState types that contain
// running animations (RFC-002 §1.3). The framework calls Tick(dt) on
// every WidgetState that implements Animator before each paint pass.
// Returning true means the animation is still running and the widget
// should be repainted; the framework marks it dirty automatically.
type Animator interface {
	Tick(dt time.Duration) (stillRunning bool)
}

// Equatable is an optional interface on Widget (RFC-001 §6.4).
// When implemented, the reconciler calls Equal() to check whether a
// widget's props have changed before calling Render(). If Equal returns
// true, the previous render output and state are reused — skipping
// Render() entirely.
//
// Widgets that do NOT implement Equatable are always re-rendered (safe
// but potentially suboptimal). "Re-render" means calling Widget.Render(),
// not repainting — the cost is a function call, not a GPU pass.
type Equatable interface {
	Widget
	// Equal returns true if this widget and other would produce identical
	// Render output. other is guaranteed to be the same concrete type.
	Equal(other Widget) bool
}

// DirtyTracker is an optional interface on WidgetState (RFC-001 §6.4).
// Widgets whose internal state can change independently of their props
// (e.g. video surfaces, external data feeds) implement DirtyTracker to
// request a repaint without waiting for a model change.
//
// The framework checks IsDirty() on every state that implements this
// interface after the animation tick pass. If any returns true, the
// widget tree is rebuilt. ClearDirty() is called after the dirty state
// has been consumed.
//
// Coupling with LayerOptions.CacheHint: a layer with CacheHint=true
// is only re-recorded when DirtyTracker.IsDirty() returns true. If the
// widget does not implement DirtyTracker, CacheHint is ignored and the
// layer is always re-recorded (safe fallback).
type DirtyTracker interface {
	// IsDirty returns true if the widget must be redrawn even though
	// its props haven't changed.
	IsDirty() bool
	// ClearDirty resets the dirty flag after the framework has consumed it.
	ClearDirty()
}

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
	Send   func(any)    // local Send bound to this UID
	Events []InputEvent // input events dispatched to this widget (RFC-002 §2.6)
}

// AdoptState is a generic helper that type-asserts the raw state or
// returns a zero-value pointer for the first render (RFC §4.2).
func AdoptState[S WidgetState](raw WidgetState) *S {
	if s, ok := raw.(*S); ok {
		return s
	}
	return new(S)
}

// Cursable is an optional interface on Widget (RFC-002 §2.7).
// Widgets that implement it declare the desired system cursor when the
// mouse pointer hovers over them. The framework calls Cursor() after
// hit-testing and sets the platform cursor accordingly.
// Default (when not implemented) is CursorDefault.
type Cursable interface {
	Widget
	Cursor(state WidgetState) input.CursorKind
}

// ── External Surfaces (RFC §8) ───────────────────────────────────

// SurfaceID identifies a surface slot (RFC §8).
type SurfaceID = draw.SurfaceID

// FrameToken is an opaque token returned by AcquireFrame, passed to ReleaseFrame.
type FrameToken uint64

// SurfaceProvider provides GPU textures for external surface rendering (RFC §8).
// External renderers (browser engines, video decoders, 3D engines) implement
// this interface to embed their content in the widget tree.
type SurfaceProvider interface {
	// AcquireFrame returns the current GPU texture for the given bounds.
	AcquireFrame(bounds draw.Rect) (draw.TextureID, FrameToken)
	// ReleaseFrame signals that the framework is done with the texture.
	ReleaseFrame(token FrameToken)
	// HandleMsg receives input events routed to this surface. Returns true if consumed.
	HandleMsg(msg any) bool
}

// SurfaceMouseMsg is sent when a mouse event occurs inside a surface area (RFC §8.3).
type SurfaceMouseMsg struct {
	SurfaceID SurfaceID
	Pos       draw.Point
	Button    input.MouseButton
	Action    input.MouseAction
}

// SurfaceKeyMsg is sent when a key event occurs while a surface has focus.
type SurfaceKeyMsg struct {
	SurfaceID SurfaceID
	Key       input.Key
	Action    input.KeyAction
	Mods      input.ModifierSet
}

// ── Surface Semantics (RFC-006 §5) ───────────────────────────────

// SemanticProvider is an optional interface for surfaces that export
// accessibility semantics. Surfaces that do not implement it remain
// black boxes with a generic fallback AccessNode.
//
// Implementation is progressive: surfaces may implement only
// SnapshotSemantics initially and add HitTest, Focus, and Action
// support incrementally.
type SemanticProvider interface {
	// SnapshotSemantics returns an immutable snapshot of the semantic
	// subtree relative to the current surface bounds.
	SnapshotSemantics(bounds draw.Rect) SurfaceSemantics

	// HitTestSemantics returns the semantic node at a position relative
	// to the surface bounds. Used for explore-by-touch and focus routing.
	HitTestSemantics(p draw.Point) (SurfaceNodeID, bool)

	// FocusSemanticNode requests focus on a specific semantic node.
	// Returns true if the surface accepted the focus change.
	FocusSemanticNode(id SurfaceNodeID) bool

	// PerformSemanticAction executes a semantic action on the given node
	// (e.g. "activate", "increment", "scrollForward").
	// Returns true if the action was handled.
	PerformSemanticAction(id SurfaceNodeID, action string) bool
}

// SurfaceNodeID identifies a node within a surface's semantic subtree.
// IDs must be stable across frames for the same logical element.
type SurfaceNodeID uint64

// SurfaceSemantics is an immutable snapshot of a surface's semantic subtree.
type SurfaceSemantics struct {
	// Roots contains the top-level semantic nodes of the surface.
	Roots []SurfaceAccessNode

	// Version is an optional monotonically increasing version number.
	// Facilitates diffing, caching, and bridge optimizations.
	Version uint64
}

// SurfaceAccessNode represents a single node in a surface's semantic subtree.
// Bounds are relative to the surface origin in dp.
type SurfaceAccessNode struct {
	ID          SurfaceNodeID
	Parent      SurfaceNodeID // 0 = root within the surface.
	Role        a11y.AccessRole
	Label       string
	Description string
	Value       string
	Bounds      draw.Rect // Relative to the surface in dp.
	Lang        string    // BCP 47 language tag (e.g. "de", "ar-EG"). Empty inherits from parent.
	States      a11y.AccessStates
	Actions     []a11y.AccessActionDesc
	Relations   []a11y.AccessRelationDesc
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

// Button creates a filled button element with arbitrary Element content.
func Button(content Element, onClick func()) Element {
	return buttonElement{Content: content, OnClick: onClick, Variant: ButtonFilled}
}

// ButtonText is a convenience constructor for text-only filled buttons.
func ButtonText(label string, onClick func()) Element {
	return buttonElement{Content: textElement{Content: label}, OnClick: onClick, Variant: ButtonFilled}
}

// ButtonVariantOf creates a button with the given variant and arbitrary content.
func ButtonVariantOf(variant ButtonVariant, content Element, onClick func()) Element {
	return buttonElement{Content: content, OnClick: onClick, Variant: variant}
}

// ButtonOutlinedText creates an outlined button with a text label.
func ButtonOutlinedText(label string, onClick func()) Element {
	return buttonElement{Content: textElement{Content: label}, OnClick: onClick, Variant: ButtonOutlined}
}

// ButtonGhostText creates a text-only (chromeless) button.
func ButtonGhostText(label string, onClick func()) Element {
	return buttonElement{Content: textElement{Content: label}, OnClick: onClick, Variant: ButtonGhost}
}

// ButtonTonalText creates a tonal button with a text label.
func ButtonTonalText(label string, onClick func()) Element {
	return buttonElement{Content: textElement{Content: label}, OnClick: onClick, Variant: ButtonTonal}
}

// IconButton creates a compact icon-only button.
func IconButton(icon string, onClick func()) Element {
	return iconButtonElement{Icon: icon, OnClick: onClick, Variant: ButtonFilled}
}

// IconButtonVariant creates an icon-only button with a specific variant.
func IconButtonVariant(variant ButtonVariant, icon string, onClick func()) Element {
	return iconButtonElement{Icon: icon, OnClick: onClick, Variant: variant}
}

// SplitButton creates a button with a main action and a dropdown menu trigger.
func SplitButton(label string, onClick func(), onMenu func(), items []SplitButtonItem) Element {
	return splitButtonElement{Label: label, OnClick: onClick, MenuItems: items, OnMenu: onMenu}
}

// SegmentedButtons creates a group of connected buttons with one selected.
func SegmentedButtons(items []SegmentedItem, selected int) Element {
	return segmentedButtonsElement{Items: items, Selected: selected}
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

// GradientRect renders a gradient-filled rectangle of a fixed size (Phase E).
func GradientRect(width, height, radius float32, paint draw.Paint) Element {
	return gradientRectElement{Width: width, Height: height, Radius: radius, Paint: paint}
}

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

// CheckerRect renders a colorful checkerboard pattern of the given size.
// Useful as a complex background to demonstrate blur/frosted-glass effects.
func CheckerRect(width, height, cellSize float32) Element {
	return checkerRectElement{Width: width, Height: height, CellSize: cellSize}
}

// BlurBox wraps a child and applies a Gaussian blur backdrop effect.
// Content underneath the BlurBox bounds is blurred at the given radius.
func BlurBox(radius float32, child Element) Element {
	return blurBoxElement{Radius: radius, Child: child}
}

// ShadowBox draws a soft shadow behind a child element.
func ShadowBox(shadow draw.Shadow, radius float32, child Element) Element {
	return shadowBoxElement{Shadow: shadow, Radius: radius, Child: child}
}

// OpacityBox applies a uniform opacity to all child content.
func OpacityBox(alpha float32, child Element) Element {
	return opacityBoxElement{Alpha: alpha, Child: child}
}

// FrostedGlass renders a frosted-glass effect: backdrop blur + semi-transparent tint overlay.
func FrostedGlass(blurRadius float32, tint draw.Color, child Element) Element {
	return frostedGlassElement{BlurRadius: blurRadius, Tint: tint, Child: child}
}

// InnerShadowBox draws an inner shadow on top of its child content.
// The shadow renders inward from the edges of the child's bounds.
func InnerShadowBox(shadow draw.Shadow, radius float32, child Element) Element {
	shadow.Inset = true
	return innerShadowBoxElement{Shadow: shadow, Radius: radius, Child: child}
}

// ElevationBox renders a hover-responsive shadow behind its child.
// The shadow interpolates from rest → hover on mouse enter, and hover → rest on leave.
// If onClick is non-nil, it is invoked on click.
func ElevationBox(rest, hover, press draw.Shadow, radius float32, onClick func(), child Element) Element {
	return elevationBoxElement{Rest: rest, Hover: hover, Press: press, Radius: radius, OnClick: onClick, Child: child}
}

// ElevationCard is a convenience wrapper around ElevationBox using theme elevation presets.
// Rest = Low, Hover = High, Press = None.
func ElevationCard(onClick func(), child Element) Element {
	return elevationCardElement{OnClick: onClick, Child: child}
}

// TintedBlur is an alias for FrostedGlass with explicit naming for tinted blur effects.
func TintedBlur(blurRadius float32, tint draw.Color, child Element) Element {
	return frostedGlassElement{BlurRadius: blurRadius, Tint: tint, Child: child}
}

// Vibrancy applies a system-accent-tinted blur to its child's backdrop.
// tintAlpha controls the opacity of the accent tint overlay (0.0–1.0).
func Vibrancy(tintAlpha float32, child Element) Element {
	return vibrancyElement{TintAlpha: tintAlpha, Child: child}
}

// GlowBox renders a soft outer glow around its child using the shadow pipeline.
func GlowBox(color draw.Color, blurRadius, radius float32, child Element) Element {
	return glowBoxElement{Color: color, BlurRadius: blurRadius, Radius: radius, Child: child}
}

// Glow is a convenience GlowBox using the theme's accent color.
func Glow(blurRadius, radius float32, child Element) Element {
	return glowBoxElement{BlurRadius: blurRadius, Radius: radius, Child: child}
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

// ── External Surfaces (RFC §8) ───────────────────────────────────

// Surface creates a surface slot element that renders GPU content from a
// SurfaceProvider (RFC §8). The width and height specify the desired size
// in dp. If provider is nil, a placeholder rectangle is rendered.
func Surface(id SurfaceID, provider SurfaceProvider, width, height float32) Element {
	return surfaceElement{ID: id, Provider: provider, Width: width, Height: height}
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

// WithFocusState links the TextField to a FocusManager for keyboard input.
// Deprecated: use WithFocus instead.
func WithFocusState(fs *FocusManager) TextFieldOption {
	return func(e *textFieldElement) { e.Focus = fs }
}

// WithFocus links the TextField to a FocusManager for keyboard input.
func WithFocus(fm *FocusManager) TextFieldOption {
	return func(e *textFieldElement) { e.Focus = fm }
}

// SelectState holds the open/closed state for a Select dropdown.
type SelectState struct {
	Open bool
}

// SelectOption configures a Select element.
type SelectOption func(*selectElement)

// WithSelectState links the Select to a SelectState for dropdown behaviour.
func WithSelectState(s *SelectState) SelectOption {
	return func(e *selectElement) { e.State = s }
}

// WithOnSelect sets the callback invoked when an option is chosen.
func WithOnSelect(fn func(string)) SelectOption {
	return func(e *selectElement) { e.OnSelect = fn }
}

// Select creates a dropdown selector. When configured with
// WithSelectState and WithOnSelect, it supports interactive
// open/close and item selection via an overlay dropdown.
func Select(value string, options []string, opts ...SelectOption) Element {
	el := selectElement{Value: value, Options: options}
	for _, o := range opts {
		o(&el)
	}
	return el
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

// ButtonVariant controls the visual style of a button.
type ButtonVariant int

const (
	// ButtonFilled is the default prominent button style (accent background).
	ButtonFilled ButtonVariant = iota
	// ButtonOutlined renders with a border and transparent background.
	ButtonOutlined
	// ButtonGhost renders with no border and transparent background.
	ButtonGhost
	// ButtonTonal renders with a tinted, semi-transparent background.
	ButtonTonal
)

type buttonElement struct {
	Content Element
	OnClick func()
	Variant ButtonVariant
}

func (buttonElement) isElement() {}

// SegmentedItem describes one segment in a SegmentedButtons group.
type SegmentedItem struct {
	Label   string
	Icon    string // optional icon (from icons package)
	OnClick func()
}

type segmentedButtonsElement struct {
	Items    []SegmentedItem
	Selected int
}

func (segmentedButtonsElement) isElement() {}

// SplitButtonItem describes a dropdown menu entry for SplitButton.
type SplitButtonItem struct {
	Label   string
	OnClick func()
}

type splitButtonElement struct {
	Label     string
	OnClick   func()
	MenuItems []SplitButtonItem
	OnMenu    func() // fires when dropdown arrow is clicked
}

func (splitButtonElement) isElement() {}

type iconButtonElement struct {
	Icon    string
	OnClick func()
	Variant ButtonVariant
	Size    float32 // 0 = default
}

func (iconButtonElement) isElement() {}

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

// gradientRectElement renders a gradient-filled rectangle of a fixed size.
type gradientRectElement struct {
	Width, Height float32
	Radius        float32
	Paint         draw.Paint
}

func (gradientRectElement) isElement() {}

type spacerElement struct{ Size float32 }

func (spacerElement) isElement() {}

type iconElement struct {
	Name string
	Size float32 // 0 = use theme Label size
}

func (iconElement) isElement() {}

type stackElement struct{ Children []Element }

func (stackElement) isElement() {}

type blurBoxElement struct {
	Radius float32
	Child  Element
}

func (blurBoxElement) isElement() {}

type checkerRectElement struct {
	Width, Height, CellSize float32
}

func (checkerRectElement) isElement() {}

type shadowBoxElement struct {
	Shadow draw.Shadow
	Radius float32
	Child  Element
}

func (shadowBoxElement) isElement() {}

type opacityBoxElement struct {
	Alpha float32
	Child Element
}

func (opacityBoxElement) isElement() {}

type frostedGlassElement struct {
	BlurRadius float32
	Tint       draw.Color
	Child      Element
}

func (frostedGlassElement) isElement() {}

type innerShadowBoxElement struct {
	Shadow draw.Shadow
	Radius float32
	Child  Element
}

func (innerShadowBoxElement) isElement() {}

type elevationBoxElement struct {
	Rest    draw.Shadow
	Hover   draw.Shadow
	Press   draw.Shadow
	Radius  float32
	OnClick func()
	Child   Element
}

func (elevationBoxElement) isElement() {}

type elevationCardElement struct {
	OnClick func()
	Child   Element
}

func (elevationCardElement) isElement() {}

type vibrancyElement struct {
	TintAlpha float32
	Child     Element
}

func (vibrancyElement) isElement() {}

type scrollViewElement struct {
	Child     Element
	MaxHeight float32
	State     *ScrollState // optional; drives vertical offset
}

func (scrollViewElement) isElement() {}

type surfaceElement struct {
	ID       SurfaceID
	Provider SurfaceProvider
	Width    float32
	Height   float32
}

func (surfaceElement) isElement() {}

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

type glowBoxElement struct {
	Color      draw.Color
	BlurRadius float32
	Radius     float32
	Child      Element
}

func (glowBoxElement) isElement() {}

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
	Color   draw.Color // optional custom color; zero = Accent.Primary
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

// BadgeColor creates a badge with a custom background color.
func BadgeColor(content Element, color draw.Color) Element {
	return badgeElement{Content: content, Color: color}
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

// MenuBarState tracks which top-level menu is open (-1 = all closed).
type MenuBarState struct {
	OpenIndex int
}

// NewMenuBarState creates a MenuBarState with all menus closed.
func NewMenuBarState() *MenuBarState {
	return &MenuBarState{OpenIndex: -1}
}

type menuBarElement struct {
	Items []MenuItem
	State *MenuBarState
}

func (menuBarElement) isElement() {}

// MenuBar creates a horizontal menu bar with dropdown submenus.
func MenuBar(items []MenuItem, state *MenuBarState) Element {
	return menuBarElement{Items: items, State: state}
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
	Focus       *FocusManager
	FocusUID    UID // assigned during layout
}

func (textFieldElement) isElement() {}

type selectElement struct {
	Value    string
	Options  []string
	State    *SelectState
	OnSelect func(string)
}

func (selectElement) isElement() {}

// widgetElement wraps a Widget for embedding in element trees.
// It is expanded by the Reconciler before layout.
// themedElement overrides the theme for its child subtree (scoped theme).
// The Reconciler replaces the active theme before resolving children,
// so all descendants inherit the overridden tokens and draw functions.
type themedElement struct {
	Theme    theme.Theme
	Children []Element
}

func (themedElement) isElement() {}

// Themed creates a scoped theme override. All children inherit the given
// theme instead of the ambient one. Combine with theme.Override() to
// create partial overrides (e.g. danger-colored buttons).
func Themed(th theme.Theme, children ...Element) Element {
	return themedElement{Theme: th, Children: children}
}

type widgetElement struct {
	W   Widget
	Key string
}

func (widgetElement) isElement() {}

// widgetBoundsElement wraps a widget's resolved element subtree,
// carrying the widget UID so that layout can register screen bounds
// for event dispatching (RFC-002 §2.6).
type widgetBoundsElement struct {
	WidgetUID UID
	Child     Element
}

func (widgetBoundsElement) isElement() {}

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
	FocusUID UID

	// IME composition state (RFC-002 §2.2).
	ComposeText        string // current pre-edit text (empty when not composing)
	ComposeCursorStart int    // cursor position within compose text (rune index)
	ComposeCursorEnd   int    // selection end within compose text (rune index)
}

// FocusState is a type alias for backward compatibility.
// Deprecated: use FocusManager directly.
type FocusState = FocusManager

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
// new transition if the on state has changed. Duration and easing come
// from the theme's MotionSpec (RFC-008 §9.5).
func (ts *ToggleState) update(on bool, de theme.DurationEasing) float32 {
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
		ts.thumbPos.SetTarget(target, de.Duration, de.Easing)
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
	Render func(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor)
}

// overlayStack collects overlay entries during layout.
type overlayStack struct {
	entries []overlayEntry
	windowW int
	windowH int
}

func (s *overlayStack) push(entry overlayEntry) {
	if s != nil {
		s.entries = append(s.entries, entry)
	}
}

// ── Layout & Scene Building ──────────────────────────────────────
// BuildScene converts an Element tree into draw commands via the
// Canvas interface (RFC §6).

type bounds struct{ X, Y, W, H, Baseline int }

const (
	framePadding    = 24
	columnGap       = 16
	rowGap          = 12
	buttonPadX      = 18
	buttonPadY      = 12
	buttonBorder    = 1
	iconButtonPad   = 10
	splitArrowWidth = 36
	segmentPadX     = 16
	segmentPadY     = 10
)

// BuildScene lays out the element tree and paints it to the canvas.
// It returns the accumulated Scene. If hitMap is non-nil, clickable
// element bounds are registered for hit-testing (M3+).
// If hover is non-nil, hover animations are applied to buttons (M4).
// BuildScene lays out the element tree and paints it to the canvas.
// It returns the accumulated Scene. If ix is non-nil, clickable
// element bounds are registered for hit-testing and hover animations
// are applied to interactive elements.
// If focus is non-nil, text fields use it for keyboard focus tracking.
func BuildScene(root Element, canvas draw.Canvas, th theme.Theme, width, height int, ix *Interactor, focusOpt ...*FocusState) draw.Scene {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}

	ix.resetCounter()

	var focus *FocusManager
	if len(focusOpt) > 0 {
		focus = focusOpt[0]
	}

	tokens := th.Tokens()
	area := bounds{X: framePadding, Y: framePadding, W: max(width-(framePadding*2), 0), H: max(height-(framePadding*2), 0)}
	var overlays overlayStack
	overlays.windowW = width
	overlays.windowH = height
	layoutElement(root, area, canvas, th, tokens, ix, &overlays, focus)

	// Switch canvas to overlay mode so overlay draw commands go to
	// separate scene lists, rendered after all main content.
	type overlayModeSetter interface{ SetOverlayMode(bool) }
	if oms, ok := canvas.(overlayModeSetter); ok {
		oms.SetOverlayMode(true)
	}
	// Render overlay entries (Tooltip, ContextMenu, etc.) on top of main tree.
	for _, entry := range overlays.entries {
		entry.Render(canvas, tokens, ix)
	}
	if oms, ok := canvas.(overlayModeSetter); ok {
		oms.SetOverlayMode(false)
	}

	// The canvas is a SceneCanvas — retrieve its scene.
	type scener interface{ Scene() draw.Scene }
	if sc, ok := canvas.(scener); ok {
		return sc.Scene()
	}
	return draw.Scene{}
}

func layoutElement(el Element, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus ...*FocusManager) bounds {
	var fs *FocusManager
	if len(focus) > 0 {
		fs = focus[0]
	}
	switch node := el.(type) {
	case nil, emptyElement, widgetElement:
		// widgetElement should be resolved by the Reconciler before layout.
		return bounds{X: area.X, Y: area.Y}

	case widgetBoundsElement:
		// Layout the child subtree. The bounds are tracked so the
		// EventDispatcher can route mouse events to this widget UID.
		return layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)

	case keyedElement:
		return layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)

	case textElement:
		style := tokens.Typography.Body
		if node.Style.Size > 0 {
			style = node.Style
		}
		metrics := canvas.MeasureText(node.Content, style)
		w := int(math.Ceil(float64(metrics.Width)))
		h := int(math.Ceil(float64(metrics.Ascent)))
		canvas.DrawText(node.Content, draw.Pt(float32(area.X), float32(area.Y)), style, tokens.Colors.Text.Primary)
		return bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: h}

	case buttonElement:
		return layoutButton(node, area, canvas, th, tokens, ix, overlays, fs)

	case iconButtonElement:
		return layoutIconButton(node, area, canvas, tokens, ix)

	case splitButtonElement:
		return layoutSplitButton(node, area, canvas, tokens, ix)

	case segmentedButtonsElement:
		return layoutSegmentedButtons(node, area, canvas, tokens, ix)

	case dividerElement:
		h := 1
		canvas.FillRect(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(h)),
			draw.SolidPaint(tokens.Colors.Stroke.Divider))
		return bounds{X: area.X, Y: area.Y, W: area.W, H: h, Baseline: h}

	case spacerElement:
		s := int(node.Size)
		return bounds{X: area.X, Y: area.Y, W: s, H: s, Baseline: s}

	case gradientRectElement:
		w := int(node.Width)
		h := int(node.Height)
		if w > area.W {
			w = area.W
		}
		if h > area.H {
			h = area.H
		}
		r := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
		if node.Radius > 0 {
			canvas.FillRoundRect(r, node.Radius, node.Paint)
		} else {
			canvas.FillRect(r, node.Paint)
		}
		return bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: h}

	case iconElement:
		size := node.Size
		if size == 0 {
			size = tokens.Typography.Label.Size * 2
		}
		// Use the Phosphor icon font for icon elements.
		// Render into a fixed square cell so all icons have uniform size
		// regardless of individual glyph bounding boxes.
		style := draw.TextStyle{
			FontFamily: "Phosphor",
			Size:       size,
			Weight:     draw.FontWeightRegular,
			LineHeight: 1.0,
			Raster:     true,
		}
		cellSize := int(math.Ceil(float64(size)))
		metrics := canvas.MeasureText(node.Name, style)
		offsetX := (float32(cellSize) - metrics.Width) / 2
		offsetY := (float32(cellSize) - metrics.Ascent) / 2
		canvas.DrawText(node.Name, draw.Pt(float32(area.X)+offsetX, float32(area.Y)+offsetY), style, tokens.Colors.Text.Primary)
		return bounds{X: area.X, Y: area.Y, W: cellSize, H: cellSize, Baseline: cellSize}

	case stackElement:
		return layoutStack(node, area, canvas, th, tokens, ix, overlays, fs)

	case checkerRectElement:
		w := int(node.Width)
		h := int(node.Height)
		if w > area.W {
			w = area.W
		}
		if h > area.H {
			h = area.H
		}
		cell := node.CellSize
		if cell < 1 {
			cell = 8
		}
		colors := [6]draw.Color{
			{R: 0.93, G: 0.27, B: 0.27, A: 1}, // red
			{R: 0.96, G: 0.62, B: 0.04, A: 1}, // amber
			{R: 0.13, G: 0.77, B: 0.37, A: 1}, // green
			{R: 0.23, G: 0.51, B: 0.96, A: 1}, // blue
			{R: 0.55, G: 0.36, B: 0.96, A: 1}, // violet
			{R: 0.93, G: 0.35, B: 0.60, A: 1}, // pink
		}
		for row := float32(0); row < float32(h); row += cell {
			for col := float32(0); col < float32(w); col += cell {
				ci := (int(row/cell) + int(col/cell)) % len(colors)
				cw := cell
				ch := cell
				if col+cw > float32(w) {
					cw = float32(w) - col
				}
				if row+ch > float32(h) {
					ch = float32(h) - row
				}
				canvas.FillRect(
					draw.R(float32(area.X)+col, float32(area.Y)+row, cw, ch),
					draw.SolidPaint(colors[ci]),
				)
			}
		}
		return bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: h}

	case blurBoxElement:
		// Layout child first to determine its actual bounds.
		b := layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
		// Push a tight clip for the child's bounds, then PushBlur captures
		// exactly that region (not the full parent content area).
		canvas.PushClip(draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H)))
		canvas.PushBlur(node.Radius)
		canvas.PopBlur()
		canvas.PopClip()
		return b

	case shadowBoxElement:
		// Draw shadow first (behind content), then layout child on top.
		b := layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
		r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
		shadow := node.Shadow
		if node.Radius > 0 {
			shadow.Radius = node.Radius
		}
		canvas.DrawShadow(r, shadow)
		return b

	case opacityBoxElement:
		canvas.PushOpacity(node.Alpha)
		b := layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
		canvas.PopOpacity()
		return b

	case frostedGlassElement:
		// Frosted glass: blurred backdrop + sharp overlay content.
		// 1. Measure child with nullCanvas (no drawing) to get bounds.
		nc := nullCanvas{delegate: canvas}
		b := layoutElement(node.Child, area, nc, th, tokens, nil, nil)
		r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))

		// 2. Register blur region in main scene.
		canvas.PushClip(r)
		canvas.PushBlur(node.BlurRadius)
		canvas.PopBlur()
		canvas.PopClip()

		// 3. Draw tint + child in overlay mode (rendered after blur post-processing).
		type overlayModeSetter interface{ SetOverlayMode(bool) }
		if oms, ok := canvas.(overlayModeSetter); ok {
			oms.SetOverlayMode(true)
			canvas.FillRoundRect(r, node.BlurRadius*0.5, draw.SolidPaint(node.Tint))
			layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
			oms.SetOverlayMode(false)
		} else {
			// Fallback: draw in main scene (blur will affect tint+child too).
			canvas.FillRoundRect(r, node.BlurRadius*0.5, draw.SolidPaint(node.Tint))
			layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
		}
		return b

	case innerShadowBoxElement:
		// Layout child first, then draw inner shadow ON TOP of child content.
		// The GPU renders shadows before rects, so we must emit the inner
		// shadow into the overlay pass — otherwise the child's rect covers it.
		b := layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
		r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
		shadow := node.Shadow
		shadow.Inset = true
		if node.Radius > 0 {
			shadow.Radius = node.Radius
		}
		type overlayModeSetter interface{ SetOverlayMode(bool) }
		if oms, ok := canvas.(overlayModeSetter); ok {
			oms.SetOverlayMode(true)
			canvas.DrawShadow(r, shadow)
			oms.SetOverlayMode(false)
		} else {
			canvas.DrawShadow(r, shadow)
		}
		return b

	case elevationBoxElement:
		// Layout child, register hover, interpolate shadow.
		b := layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
		r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
		hoverOpacity := ix.RegisterHit(r, node.OnClick)
		shadow := draw.LerpShadow(node.Rest, node.Hover, hoverOpacity)
		if node.Radius > 0 {
			shadow.Radius = node.Radius
		}
		canvas.DrawShadow(r, shadow)
		return b

	case elevationCardElement:
		// Convenience: uses theme elevation presets (Low → High → None).
		b := layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
		r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
		hoverOpacity := ix.RegisterHit(r, node.OnClick)
		shadow := draw.LerpShadow(tokens.Elevation.Low, tokens.Elevation.High, hoverOpacity)
		canvas.DrawShadow(r, shadow)
		return b

	case vibrancyElement:
		// Vibrancy: accent-tinted blur using frostedGlassElement under the hood.
		tint := tokens.Colors.Accent.Primary
		tint.A = node.TintAlpha
		fg := frostedGlassElement{BlurRadius: 20, Tint: tint, Child: node.Child}
		return layoutElement(fg, area, canvas, th, tokens, ix, overlays, fs)

	case glowBoxElement:
		b := layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)
		r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
		glowColor := node.Color
		if glowColor.A == 0 {
			glowColor = tokens.Colors.Accent.Primary
			glowColor.A = 0.6
		}
		canvas.DrawShadow(r, draw.Shadow{
			Color:      glowColor,
			BlurRadius: node.BlurRadius,
			Radius:     node.Radius,
		})
		return b

	case scrollViewElement:
		return layoutScrollView(node, area, canvas, th, tokens, ix, overlays, fs)

	case surfaceElement:
		return layoutSurface(node, area, canvas, tokens, ix)

	case themedElement:
		// Switch theme and tokens for this subtree.
		subTh := node.Theme
		subTokens := subTh.Tokens()
		box := boxElement{Axis: AxisColumn, Children: node.Children}
		return layoutBox(box, area, canvas, subTh, subTokens, ix, overlays, fs)

	case boxElement:
		return layoutBox(node, area, canvas, th, tokens, ix, overlays, fs)

	case paddingElement:
		return layoutPadding(node, area, canvas, th, tokens, ix, overlays, fs)

	case sizedBoxElement:
		return layoutSizedBox(node, area, canvas, th, tokens, ix, overlays, fs)

	case expandedElement:
		// Outside a Flex context, Expanded passes through to its child.
		return layoutElement(node.Child, area, canvas, th, tokens, ix, overlays, fs)

	case flexElement:
		return layoutFlex(node, area, canvas, th, tokens, ix, overlays, fs)

	case gridElement:
		return layoutGrid(node, area, canvas, th, tokens, ix, overlays, fs)

	case virtualListElement:
		return layoutVirtualList(node, area, canvas, th, tokens, ix, overlays, fs)

	case treeElement:
		return layoutTree(node, area, canvas, th, tokens, ix, overlays, fs)

	case richTextElement:
		return layoutRichText(node, area, canvas, th, tokens)

	// Tier 2 widgets
	case checkboxElement:
		return layoutCheckbox(node, area, canvas, th, tokens, ix, fs)
	case radioElement:
		return layoutRadio(node, area, canvas, th, tokens, ix, fs)
	case toggleElement:
		return layoutToggle(node, area, canvas, th, tokens, ix, fs)
	case sliderElement:
		return layoutSlider(node, area, canvas, th, tokens, ix, fs)
	case progressBarElement:
		return layoutProgressBar(node, area, canvas, th, tokens)
	case textFieldElement:
		return layoutTextField(node, area, canvas, th, tokens, ix, fs)
	case selectElement:
		return layoutSelect(node, area, canvas, th, tokens, ix, overlays, fs)

	// Tier 3 widgets
	case cardElement:
		return layoutCard(node, area, canvas, th, tokens, ix, overlays, fs)
	case tabsElement:
		return layoutTabs(node, area, canvas, th, tokens, ix, overlays, fs)
	case accordionElement:
		return layoutAccordion(node, area, canvas, th, tokens, ix, overlays, fs)
	case tooltipElement:
		return layoutTooltip(node, area, canvas, th, tokens, ix, overlays, fs)
	case badgeElement:
		return layoutBadge(node, area, canvas, th, tokens, ix, overlays, fs)
	case chipElement:
		return layoutChip(node, area, canvas, th, tokens, ix, overlays, fs)
	case menuBarElement:
		return layoutMenuBar(node, area, canvas, th, tokens, ix, overlays, fs)
	case contextMenuElement:
		return layoutContextMenu(node, area, canvas, th, tokens, ix, overlays, fs)
	case splitViewElement:
		return layoutSplitView(node, area, canvas, th, tokens, ix, overlays, fs)
	case Overlay:
		return layoutOverlay(node, area, canvas, th, tokens, ix, overlays, fs)

	case customLayoutElement:
		return layoutCustom(node, area, canvas, th, tokens, ix, overlays, fs)

	default:
		return bounds{X: area.X, Y: area.Y}
	}
}

// layoutCustom implements the custom layout protocol (RFC-002 §4.3).
// It delegates measurement and placement to the user-provided Layout.
func layoutCustom(node customLayoutElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, fs *FocusManager) bounds {
	if node.Layout == nil || len(node.Children) == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	nc := nullCanvas{delegate: canvas}

	// Track placements: child index → offset.
	type placement struct {
		offset draw.Point
		placed bool
	}
	placements := make([]placement, len(node.Children))

	// Measure callback: layout children with nullCanvas to get their size.
	measureFn := func(child Element, c Constraints) Size {
		measureArea := bounds{X: 0, Y: 0, W: int(c.MaxWidth), H: int(c.MaxHeight)}
		cb := layoutElement(child, measureArea, nc, th, tokens, nil, nil)
		return Size{Width: float32(cb.W), Height: float32(cb.H)}
	}

	// Place callback: record offset for later painting.
	// We use treeEqual instead of == to avoid panics on uncomparable
	// element types (e.g. buttonElement contains func fields).
	placeFn := func(child Element, offset draw.Point) {
		for i, ch := range node.Children {
			if treeEqual(ch, child) {
				placements[i] = placement{offset: offset, placed: true}
				return
			}
		}
	}

	ctx := LayoutCtx{
		Constraints: Constraints{
			MaxWidth:  float32(area.W),
			MaxHeight: float32(area.H),
		},
		Measure: measureFn,
		Place:   placeFn,
		Theme:   th,
	}

	size := node.Layout.LayoutChildren(ctx, node.Children)

	// Paint pass: draw each placed child at its offset.
	maxW, maxH := 0, 0
	for i, child := range node.Children {
		p := placements[i]
		if !p.placed {
			continue
		}
		childArea := bounds{
			X: area.X + int(p.offset.X),
			Y: area.Y + int(p.offset.Y),
			W: area.W,
			H: area.H,
		}
		cb := layoutElement(child, childArea, canvas, th, tokens, ix, overlays, fs)
		endX := int(p.offset.X) + cb.W
		endY := int(p.offset.Y) + cb.H
		if endX > maxW {
			maxW = endX
		}
		if endY > maxH {
			maxH = endY
		}
	}

	w := int(size.Width)
	h := int(size.Height)
	if w == 0 {
		w = maxW
	}
	if h == 0 {
		h = maxH
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
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

// drawFocusRing renders a subtle glow aura + focus stroke around a widget (RFC-008 §9.4).
func drawFocusRing(canvas draw.Canvas, rect draw.Rect, radius float32, tokens theme.TokenSet) {
	glowColor := tokens.Colors.Stroke.Focus
	glowColor.A = 0.45
	canvas.DrawShadow(rect, draw.Shadow{
		Color:      glowColor,
		BlurRadius: 8,
		Radius:     radius,
	})
	canvas.StrokeRoundRect(rect, radius, draw.Stroke{
		Paint: draw.SolidPaint(tokens.Colors.Stroke.Focus),
		Width: 1.5,
	})
}

// ── Button layout functions ────────────────────────────────────

// buttonVariantColors returns fill, border, and text colors for a button variant.
func buttonVariantColors(variant ButtonVariant, tokens theme.TokenSet, hoverOpacity float32) (fill, border, textCol draw.Color) {
	switch variant {
	case ButtonOutlined:
		fill = draw.Color{A: 0} // transparent
		if hoverOpacity > 0 {
			fill = lerpColor(fill, tokens.Colors.Surface.Hovered, hoverOpacity)
		}
		border = tokens.Colors.Accent.Primary
		textCol = tokens.Colors.Accent.Primary
	case ButtonGhost:
		fill = draw.Color{A: 0} // transparent
		if hoverOpacity > 0 {
			fill = lerpColor(fill, tokens.Colors.Surface.Hovered, hoverOpacity)
		}
		border = draw.Color{A: 0} // no border
		textCol = tokens.Colors.Accent.Primary
	case ButtonTonal:
		// Blend accent into surface at 15% to produce an opaque tonal fill.
		accent := tokens.Colors.Accent.Primary
		base := tokens.Colors.Surface.Base
		fill = draw.Color{
			R: base.R + (accent.R-base.R)*0.15,
			G: base.G + (accent.G-base.G)*0.15,
			B: base.B + (accent.B-base.B)*0.15,
			A: 1,
		}
		if hoverOpacity > 0 {
			hoverFill := draw.Color{
				R: base.R + (accent.R-base.R)*0.25,
				G: base.G + (accent.G-base.G)*0.25,
				B: base.B + (accent.B-base.B)*0.25,
				A: 1,
			}
			fill = lerpColor(fill, hoverFill, hoverOpacity)
		}
		border = draw.Color{A: 0}
		textCol = tokens.Colors.Accent.Primary
	default: // ButtonFilled
		fill = tokens.Colors.Accent.Primary
		if hoverOpacity > 0 {
			fill = lerpColor(fill, hoverHighlight(fill), hoverOpacity)
		}
		border = draw.Color{
			R: fill.R * 0.7,
			G: fill.G * 0.7,
			B: fill.B * 0.7,
			A: 1,
		}
		textCol = tokens.Colors.Text.OnAccent
	}
	return
}

func layoutButton(node buttonElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, fs *FocusManager) bounds {
	// Pass 1: measure content via nullCanvas.
	nc := nullCanvas{delegate: canvas}
	cb := layoutElement(node.Content, bounds{X: 0, Y: 0, W: area.W, H: area.H}, nc, th, tokens, nil, nil, nil)

	contentW := cb.W
	contentH := cb.H
	w := contentW + (buttonPadX * 2)
	h := contentH + (buttonPadY * 2)

	// Register hit target and get hover opacity atomically.
	buttonRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	hoverOpacity := ix.RegisterHit(buttonRect, node.OnClick)

	// Focus management (RFC-008 §9.4).
	var focused bool
	if fs != nil {
		uid := fs.NextElementUID()
		fs.RegisterFocusable(uid, FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = fs.IsElementFocused(uid)
	}

	// Custom theme DrawFunc dispatch (RFC §5.3).
	if df := th.DrawFunc(theme.WidgetKindButton); df != nil {
		df(theme.DrawCtx{
			Canvas:  canvas,
			Bounds:  buttonRect,
			Hovered: hoverOpacity > 0,
			Focused: focused,
		}, tokens, node)
	} else {
		fillColor, borderColor, textColor := buttonVariantColors(node.Variant, tokens, hoverOpacity)

		if node.Variant == ButtonFilled {
			// Filled: border as background fill, opaque fill on top (2-rect approach).
			canvas.FillRoundRect(buttonRect,
				tokens.Radii.Button, draw.SolidPaint(borderColor))
			canvas.FillRoundRect(draw.R(float32(area.X+buttonBorder), float32(area.Y+buttonBorder),
				float32(max(w-buttonBorder*2, 0)), float32(max(h-buttonBorder*2, 0))),
				maxf(tokens.Radii.Button-float32(buttonBorder), 0), draw.SolidPaint(fillColor))
		} else {
			// Non-filled: fill first, then stroke outline on top.
			if fillColor.A > 0 {
				canvas.FillRoundRect(buttonRect, tokens.Radii.Button, draw.SolidPaint(fillColor))
			}
			if borderColor.A > 0 {
				canvas.StrokeRoundRect(buttonRect, tokens.Radii.Button, draw.Stroke{
					Paint: draw.SolidPaint(borderColor),
					Width: float32(buttonBorder),
				})
			}
		}

		// Focus glow (RFC-008 §9.4).
		if focused {
			drawFocusRing(canvas, buttonRect, tokens.Radii.Button, tokens)
		}

		// Pass 2: render content centered.
		if txt, ok := node.Content.(textElement); ok {
			style := tokens.Typography.Label
			metrics := canvas.MeasureText(txt.Content, style)
			labelW := int(math.Ceil(float64(metrics.Width)))
			labelH := int(math.Ceil(float64(metrics.Ascent)))
			canvas.DrawText(txt.Content,
				draw.Pt(float32(area.X+(w-labelW)/2), float32(area.Y+(h-labelH)/2)),
				style, textColor)
		} else {
			contentX := area.X + (w-contentW)/2
			contentY := area.Y + (h-contentH)/2
			// For non-filled variants, override theme text/icon colors so
			// child elements (Text, Icon inside a Row) use the variant color.
			contentTh := th
			contentTokens := tokens
			if node.Variant != ButtonFilled {
				contentTokens.Colors.Text.Primary = textColor
				contentTokens.Colors.Text.OnAccent = textColor
			}
			layoutElement(node.Content, bounds{X: contentX, Y: contentY, W: contentW, H: contentH}, canvas, contentTh, contentTokens, ix, overlays, fs)
		}
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: buttonPadY + cb.Baseline}
}

func layoutIconButton(node iconButtonElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) bounds {
	size := node.Size
	if size == 0 {
		size = tokens.Typography.Label.Size * 2
	}
	cellSize := int(math.Ceil(float64(size)))
	w := cellSize + iconButtonPad*2
	h := w // square

	buttonRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	hoverOpacity := ix.RegisterHit(buttonRect, node.OnClick)

	fillColor, borderColor, iconColor := buttonVariantColors(node.Variant, tokens, hoverOpacity)

	if node.Variant == ButtonFilled {
		canvas.FillRoundRect(buttonRect,
			tokens.Radii.Button, draw.SolidPaint(borderColor))
		canvas.FillRoundRect(draw.R(float32(area.X+buttonBorder), float32(area.Y+buttonBorder),
			float32(max(w-buttonBorder*2, 0)), float32(max(h-buttonBorder*2, 0))),
			maxf(tokens.Radii.Button-float32(buttonBorder), 0), draw.SolidPaint(fillColor))
	} else {
		if fillColor.A > 0 {
			canvas.FillRoundRect(buttonRect, tokens.Radii.Button, draw.SolidPaint(fillColor))
		}
		if borderColor.A > 0 {
			canvas.StrokeRoundRect(buttonRect, tokens.Radii.Button, draw.Stroke{
				Paint: draw.SolidPaint(borderColor),
				Width: float32(buttonBorder),
			})
		}
	}

	// Render icon centered.
	style := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       size,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}
	metrics := canvas.MeasureText(node.Icon, style)
	offsetX := (float32(w) - metrics.Width) / 2
	offsetY := (float32(h) - metrics.Ascent) / 2
	canvas.DrawText(node.Icon, draw.Pt(float32(area.X)+offsetX, float32(area.Y)+offsetY), style, iconColor)

	return bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: h}
}

func layoutSplitButton(node splitButtonElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) bounds {
	// Measure label.
	style := tokens.Typography.Label
	metrics := canvas.MeasureText(node.Label, style)
	labelW := int(math.Ceil(float64(metrics.Width)))
	labelH := int(math.Ceil(float64(metrics.Ascent)))

	mainW := labelW + buttonPadX*2
	arrowW := splitArrowWidth
	totalW := mainW + arrowW
	h := labelH + buttonPadY*2

	radius := tokens.Radii.Button

	// Main button hit target.
	mainRect := draw.R(float32(area.X), float32(area.Y), float32(mainW), float32(h))
	mainHover := ix.RegisterHit(mainRect, node.OnClick)

	// Arrow button hit target.
	arrowRect := draw.R(float32(area.X+mainW), float32(area.Y), float32(arrowW), float32(h))
	arrowHover := ix.RegisterHit(arrowRect, node.OnMenu)

	// Draw main button (left rounded corners).
	mainFill := tokens.Colors.Accent.Primary
	if mainHover > 0 {
		mainFill = lerpColor(mainFill, hoverHighlight(mainFill), mainHover)
	}
	// Full rounded rect, then overlay the right half to square off right corners.
	canvas.FillRoundRect(draw.R(float32(area.X), float32(area.Y), float32(mainW+1), float32(h)),
		radius, draw.SolidPaint(mainFill))
	// Square off right side.
	canvas.FillRect(draw.R(float32(area.X+mainW-int(radius)), float32(area.Y), float32(int(radius)+1), float32(h)),
		draw.SolidPaint(mainFill))

	// Draw arrow button (right rounded corners).
	arrowFill := tokens.Colors.Accent.Primary
	if arrowHover > 0 {
		arrowFill = lerpColor(arrowFill, hoverHighlight(arrowFill), arrowHover)
	}
	canvas.FillRoundRect(draw.R(float32(area.X+mainW), float32(area.Y), float32(arrowW), float32(h)),
		radius, draw.SolidPaint(arrowFill))
	// Square off left side.
	canvas.FillRect(draw.R(float32(area.X+mainW), float32(area.Y), float32(int(radius)), float32(h)),
		draw.SolidPaint(arrowFill))

	// Divider line between main and arrow.
	divX := float32(area.X + mainW)
	canvas.FillRect(draw.R(divX, float32(area.Y+4), 1, float32(h-8)),
		draw.SolidPaint(draw.Color{R: 1, G: 1, B: 1, A: 0.3}))

	// Label text centered in main area.
	canvas.DrawText(node.Label,
		draw.Pt(float32(area.X+(mainW-labelW)/2), float32(area.Y+(h-labelH)/2)),
		style, tokens.Colors.Text.OnAccent)

	// Caret icon centered in arrow area.
	iconStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       tokens.Typography.Label.Size * 1.5,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}
	caretDown := "\uE136" // icons.CaretDown
	caretMetrics := canvas.MeasureText(caretDown, iconStyle)
	caretX := float32(area.X+mainW) + (float32(arrowW)-caretMetrics.Width)/2
	caretY := float32(area.Y) + (float32(h)-caretMetrics.Ascent)/2
	canvas.DrawText(caretDown, draw.Pt(caretX, caretY), iconStyle, tokens.Colors.Text.OnAccent)

	return bounds{X: area.X, Y: area.Y, W: totalW, H: h, Baseline: buttonPadY + labelH}
}

func layoutSegmentedButtons(node segmentedButtonsElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) bounds {
	n := len(node.Items)
	if n == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	style := tokens.Typography.Label
	iconStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       tokens.Typography.Label.Size * 1.5,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}

	// Pass 1: measure each segment.
	type segInfo struct {
		labelW, labelH int
		iconW          int
		totalW         int
	}
	infos := make([]segInfo, n)
	maxH := 0
	for i, item := range node.Items {
		var info segInfo
		if item.Label != "" {
			m := canvas.MeasureText(item.Label, style)
			info.labelW = int(math.Ceil(float64(m.Width)))
			info.labelH = int(math.Ceil(float64(m.Ascent)))
		}
		if item.Icon != "" {
			m := canvas.MeasureText(item.Icon, iconStyle)
			info.iconW = int(math.Ceil(float64(m.Width)))
			if info.labelH == 0 {
				info.labelH = int(math.Ceil(float64(m.Ascent)))
			}
		}
		info.totalW = segmentPadX*2 + info.labelW
		if item.Icon != "" {
			info.totalW += info.iconW
			if item.Label != "" {
				info.totalW += 6 // gap between icon and label
			}
		}
		infos[i] = info
		h := info.labelH + segmentPadY*2
		if h > maxH {
			maxH = h
		}
	}

	radius := tokens.Radii.Button

	// Pass 2: render segments.
	cursorX := area.X
	for i, item := range node.Items {
		info := infos[i]
		w := info.totalW

		segRect := draw.R(float32(cursorX), float32(area.Y), float32(w), float32(maxH))
		hoverOpacity := ix.RegisterHit(segRect, item.OnClick)

		selected := i == node.Selected

		// Determine colors.
		var fillColor, textColor draw.Color
		if selected {
			fillColor = tokens.Colors.Accent.Primary
			if hoverOpacity > 0 {
				fillColor = lerpColor(fillColor, hoverHighlight(fillColor), hoverOpacity)
			}
			textColor = tokens.Colors.Text.OnAccent
		} else {
			fillColor = tokens.Colors.Surface.Elevated
			if hoverOpacity > 0 {
				fillColor = lerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
			}
			textColor = tokens.Colors.Text.Primary
		}

		// Draw segment background with appropriate corner rounding.
		var segRadius float32
		if n == 1 {
			segRadius = radius
		}
		// For first/last, draw rounded; for middle, draw square.
		if i == 0 && n > 1 {
			// Left-rounded segment.
			canvas.FillRoundRect(segRect, radius, draw.SolidPaint(fillColor))
			// Square off right side.
			canvas.FillRect(draw.R(float32(cursorX+w-int(radius)), float32(area.Y), float32(int(radius)), float32(maxH)),
				draw.SolidPaint(fillColor))
		} else if i == n-1 && n > 1 {
			// Right-rounded segment.
			canvas.FillRoundRect(segRect, radius, draw.SolidPaint(fillColor))
			// Square off left side.
			canvas.FillRect(draw.R(float32(cursorX), float32(area.Y), float32(int(radius)), float32(maxH)),
				draw.SolidPaint(fillColor))
		} else if n == 1 {
			canvas.FillRoundRect(segRect, segRadius, draw.SolidPaint(fillColor))
		} else {
			// Middle segment — no rounding.
			canvas.FillRect(segRect, draw.SolidPaint(fillColor))
		}

		// Draw border between segments (not after last).
		if i < n-1 {
			canvas.FillRect(draw.R(float32(cursorX+w), float32(area.Y+2), 1, float32(maxH-4)),
				draw.SolidPaint(tokens.Colors.Stroke.Border))
		}

		// Render content centered.
		contentX := cursorX + segmentPadX
		centerY := area.Y + (maxH-info.labelH)/2
		if item.Icon != "" {
			canvas.DrawText(item.Icon, draw.Pt(float32(contentX), float32(centerY)), iconStyle, textColor)
			contentX += info.iconW
			if item.Label != "" {
				contentX += 6
			}
		}
		if item.Label != "" {
			canvas.DrawText(item.Label, draw.Pt(float32(contentX), float32(centerY)), style, textColor)
		}

		cursorX += w
	}

	totalW := cursorX - area.X
	return bounds{X: area.X, Y: area.Y, W: totalW, H: maxH, Baseline: segmentPadY + maxH/2}
}

func layoutBox(node boxElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus ...*FocusManager) bounds {
	var fs *FocusManager
	if len(focus) > 0 {
		fs = focus[0]
	}

	if node.Axis == AxisRow {
		return layoutBoxRow(node, area, canvas, th, tokens, ix, overlays, fs)
	}

	// Column: single-pass layout.
	cursorY := area.Y
	maxW := 0
	maxH := 0
	count := 0
	firstBaseline := 0

	for _, child := range node.Children {
		childBounds := layoutElement(child, bounds{X: area.X, Y: cursorY, W: area.W, H: area.H}, canvas, th, tokens, ix, overlays, fs)
		if childBounds.W == 0 && childBounds.H == 0 {
			continue
		}
		if count == 0 {
			firstBaseline = childBounds.Baseline
		}
		count++
		cursorY += childBounds.H + columnGap
		maxW = max(maxW, childBounds.W)
		maxH = max(maxH, cursorY-area.Y-columnGap)
	}

	if count == 0 {
		return bounds{X: area.X, Y: area.Y}
	}
	if firstBaseline == 0 {
		firstBaseline = maxH
	}
	return bounds{X: area.X, Y: area.Y, W: maxW, H: maxH, Baseline: firstBaseline}
}

// layoutBoxRow performs a two-pass row layout with center alignment.
// Pass 1 measures all children via nullCanvas to determine maxH;
// Pass 2 renders each child vertically centered within maxH.
func layoutBoxRow(node boxElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	n := len(node.Children)
	if n == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	// Pass 1: measure with nullCanvas.
	type childInfo struct {
		w, h, baseline int
	}
	infos := make([]childInfo, n)
	nc := nullCanvas{delegate: canvas}
	cursorX := area.X
	maxH := 0
	hasContent := false

	for i, child := range node.Children {
		childW := area.X + area.W - cursorX
		if childW < 0 {
			childW = 0
		}
		cb := layoutElement(child, bounds{X: cursorX, Y: area.Y, W: childW, H: area.H}, nc, th, tokens, nil, nil, nil)
		if cb.W == 0 && cb.H == 0 {
			continue
		}
		infos[i] = childInfo{w: cb.W, h: cb.H, baseline: cb.Baseline}
		maxH = max(maxH, cb.H)
		cursorX += cb.W + rowGap
		hasContent = true
	}

	if !hasContent {
		return bounds{X: area.X, Y: area.Y}
	}

	// Pass 2: render with vertical centering.
	cursorX = area.X
	maxW := 0

	for i, child := range node.Children {
		info := infos[i]
		if info.w == 0 && info.h == 0 {
			continue
		}
		yOffset := (maxH - info.h) / 2
		childW := area.X + area.W - cursorX
		if childW < 0 {
			childW = 0
		}
		layoutElement(child, bounds{X: cursorX, Y: area.Y + yOffset, W: childW, H: area.H}, canvas, th, tokens, ix, overlays, focus)
		cursorX += info.w + rowGap
		maxW = max(maxW, cursorX-area.X-rowGap)
	}

	// Baseline: use the tallest child's baseline + its centering offset.
	baseline := maxH
	for _, info := range infos {
		if info.h > 0 && info.baseline > 0 {
			bl := (maxH-info.h)/2 + info.baseline
			if bl > 0 {
				baseline = bl
				break
			}
		}
	}
	return bounds{X: area.X, Y: area.Y, W: maxW, H: maxH, Baseline: baseline}
}

func layoutStack(node stackElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus ...*FocusManager) bounds {
	var fs *FocusManager
	if len(focus) > 0 {
		fs = focus[0]
	}
	maxW := 0
	maxH := 0
	firstBaseline := 0
	for i, child := range node.Children {
		childBounds := layoutElement(child, area, canvas, th, tokens, ix, overlays, fs)
		maxW = max(maxW, childBounds.W)
		maxH = max(maxH, childBounds.H)
		if i == 0 {
			firstBaseline = childBounds.Baseline
		}
	}
	if maxW == 0 && maxH == 0 {
		return bounds{X: area.X, Y: area.Y}
	}
	if firstBaseline == 0 {
		firstBaseline = maxH
	}
	return bounds{X: area.X, Y: area.Y, W: maxW, H: maxH, Baseline: firstBaseline}
}

// layoutSurface renders an external surface slot (RFC §8).
// If the provider is non-nil, AcquireFrame/ReleaseFrame are called to
// obtain and release the GPU texture. Otherwise a placeholder is drawn.
func layoutSurface(node surfaceElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) bounds {
	w := int(node.Width)
	h := int(node.Height)
	if w > area.W {
		w = area.W
	}
	if h > area.H {
		h = area.H
	}

	r := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	if node.Provider != nil {
		texID, token := node.Provider.AcquireFrame(r)
		canvas.DrawTexture(texID, r)
		node.Provider.ReleaseFrame(token)
	} else {
		// Fallback: render placeholder rectangle for surface slot.
		canvas.FillRect(r, draw.SolidPaint(tokens.Colors.Surface.Base))
		canvas.StrokeRect(r, draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Divider), Width: 1})
	}

	// Register draggable hit target for input routing to surface.
	// The drag callback fires on initial press and every move.
	// The release callback fires once when the mouse button is released.
	// Both forward SurfaceMouseMsg to the SurfaceProvider (RFC §8.3).
	if ix != nil && node.Provider != nil {
		provider := node.Provider
		surfID := node.ID
		origin := draw.Pt(r.X, r.Y)
		pressed := false
		ix.RegisterSurfaceDrag(r,
			func(x, y float32) {
				localPos := draw.Pt(x-origin.X, y-origin.Y)
				if !pressed {
					// First call = press.
					pressed = true
					provider.HandleMsg(SurfaceMouseMsg{
						SurfaceID: surfID,
						Pos:       localPos,
						Button:    input.MouseButtonLeft,
						Action:    input.MousePress,
					})
				} else {
					provider.HandleMsg(SurfaceMouseMsg{
						SurfaceID: surfID,
						Pos:       localPos,
						Button:    input.MouseButtonLeft,
						Action:    input.MouseMove,
					})
				}
			},
			func(x, y float32) {
				pressed = false
				provider.HandleMsg(SurfaceMouseMsg{
					SurfaceID: surfID,
					Pos:       draw.Pt(x-origin.X, y-origin.Y),
					Button:    input.MouseButtonLeft,
					Action:    input.MouseRelease,
				})
			},
		)
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: h}
}

func layoutScrollView(node scrollViewElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus ...*FocusManager) bounds {
	var fs *FocusManager
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

	// Reserve scrollbar width inside the allocated area so clipped
	// parents (e.g. SplitView) don't cut it off.
	scrollbarW := int(tokens.Scroll.TrackWidth)
	if scrollbarW <= 0 {
		scrollbarW = 8
	}
	contentW := area.W // width available for child content

	// Pre-measure to detect whether scrollbar is needed.
	nc := nullCanvas{delegate: canvas}
	measureArea := bounds{X: area.X, Y: area.Y, W: contentW, H: area.H}
	mb := layoutElement(node.Child, measureArea, nc, th, tokens, nil, nil, nil)
	needsScroll := mb.H > viewportH

	if needsScroll {
		contentW = max(area.W-scrollbarW, 0)
	}

	// Clip to viewport.
	canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(contentW), float32(viewportH)))

	// Render child offset by -offset in Y so content scrolls upward.
	childArea := bounds{X: area.X, Y: area.Y - int(offset), W: contentW, H: area.H + int(offset)}
	childBounds := layoutElement(node.Child, childArea, canvas, th, tokens, ix, overlays, fs)

	canvas.PopClip()

	contentH := childBounds.H

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
	if node.State != nil && needsScroll {
		state := node.State
		cH := float32(contentH)
		vH := float32(viewportH)
		ix.RegisterScroll(
			draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(viewportH)),
			cH, vH,
			func(deltaY float32) {
				state.ScrollBy(deltaY, cH, vH)
			},
		)
	}

	// Draw scrollbar inside allocated area.
	w := area.W
	if needsScroll {
		drawScrollbar(canvas, tokens, ix, node.State, area.X+contentW, area.Y, viewportH, float32(contentH), offset)
	} else {
		w = max(childBounds.W, area.W)
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

func layoutCheckbox(node checkboxElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, focus *FocusManager) bounds {
	style := tokens.Typography.Body
	metrics := canvas.MeasureText(node.Label, style)
	labelW := int(math.Ceil(float64(metrics.Width)))
	labelH := int(math.Ceil(float64(metrics.Ascent)))
	totalH := max(checkboxSize, labelH)
	totalW := checkboxSize + checkboxGap + labelW

	// Register hit target and get hover opacity atomically.
	checkboxRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(totalH))
	var clickFn func()
	if node.OnToggle != nil {
		checked := node.Checked
		onToggle := node.OnToggle
		clickFn = func() { onToggle(!checked) }
	}
	hoverOpacity := ix.RegisterHit(checkboxRect, clickFn)

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	boxY := area.Y + (totalH-checkboxSize)/2
	boxRect := draw.R(float32(area.X), float32(boxY), float32(checkboxSize), float32(checkboxSize))

	// Border
	canvas.FillRoundRect(boxRect,
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

	// Focus glow on the checkbox box (RFC-008 §9.4).
	if focused {
		drawFocusRing(canvas, boxRect, tokens.Radii.Input, tokens)
	}

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

	return bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

func layoutRadio(node radioElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, focus *FocusManager) bounds {
	style := tokens.Typography.Body
	metrics := canvas.MeasureText(node.Label, style)
	labelW := int(math.Ceil(float64(metrics.Width)))
	labelH := int(math.Ceil(float64(metrics.Ascent)))
	totalH := max(checkboxSize, labelH)
	totalW := checkboxSize + checkboxGap + labelW

	// Register hit target and get hover opacity atomically.
	radioRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(totalH))
	hoverOpacity := ix.RegisterHit(radioRect, node.OnSelect)

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	boxY := area.Y + (totalH-checkboxSize)/2
	circleRect := draw.R(float32(area.X), float32(boxY), float32(checkboxSize), float32(checkboxSize))

	// Outer circle
	canvas.FillEllipse(circleRect, draw.SolidPaint(tokens.Colors.Stroke.Border))

	// Inner fill
	fillColor := tokens.Colors.Surface.Elevated
	if hoverOpacity > 0 {
		fillColor = lerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
	}
	canvas.FillEllipse(
		draw.R(float32(area.X+checkboxBorder), float32(boxY+checkboxBorder),
			float32(checkboxSize-checkboxBorder*2), float32(checkboxSize-checkboxBorder*2)),
		draw.SolidPaint(fillColor))

	// Focus glow on the radio circle (RFC-008 §9.4).
	if focused {
		drawFocusRing(canvas, circleRect, float32(checkboxSize)/2, tokens)
	}

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

	return bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

func layoutToggle(node toggleElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, focus *FocusManager) bounds {
	// Register hit target and get hover opacity atomically.
	toggleRect := draw.R(float32(area.X), float32(area.Y), float32(toggleTrackW), float32(toggleTrackH))
	var toggleClickFn func()
	if node.OnToggle != nil {
		on := node.On
		onToggle := node.OnToggle
		toggleClickFn = func() { onToggle(!on) }
	}
	hoverOpacity := ix.RegisterHit(toggleRect, toggleClickFn)

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	// Animation progress: 0 = off, 1 = on.
	var t float32
	if node.State != nil {
		t = node.State.update(node.On, tokens.Motion.Quick)
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

	// Focus glow on the toggle track (RFC-008 §9.4).
	if focused {
		drawFocusRing(canvas, toggleRect, float32(toggleTrackH)/2, tokens)
	}

	return bounds{X: area.X, Y: area.Y, W: toggleTrackW, H: toggleTrackH}
}

func layoutSlider(node sliderElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, focus *FocusManager) bounds {
	trackW := sliderMaxWidth
	if area.W < trackW {
		trackW = area.W
	}

	// Register draggable hit target and get hover opacity atomically.
	sliderRect := draw.R(float32(area.X), float32(area.Y), float32(trackW), float32(sliderHeight))
	var dragFn func(x, y float32)
	if node.OnChange != nil {
		areaX := float32(area.X)
		tw := float32(trackW)
		onChange := node.OnChange
		dragFn = func(x, _ float32) {
			v := (x - areaX) / tw
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			onChange(v)
		}
	}
	hoverOpacity := ix.RegisterDrag(sliderRect, dragFn)

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
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
	thumbRect := draw.R(float32(thumbX), float32(thumbY), float32(sliderThumbD), float32(sliderThumbD))
	canvas.FillEllipse(thumbRect, draw.SolidPaint(tokens.Colors.Accent.Primary))

	// Focus glow on the slider thumb (RFC-008 §9.4).
	if focused {
		drawFocusRing(canvas, thumbRect, float32(sliderThumbD)/2, tokens)
	}

	return bounds{X: area.X, Y: area.Y, W: trackW, H: sliderHeight}
}

func layoutProgressBar(node progressBarElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet) bounds {
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

func layoutTextField(node textFieldElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, focus *FocusManager) bounds {
	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := textFieldW
	if area.W < w {
		w = area.W
	}

	// Assign a focus UID if focus manager is provided.
	var focusUID UID
	if focus != nil {
		focusUID = focus.NextElementUID()
		focus.RegisterFocusable(focusUID, FocusOpts{
			Focusable:    true,
			TabIndex:     0,
			FocusOnClick: true,
		})
	}
	focused := focus.IsElementFocused(focusUID)

	// Custom theme DrawFunc dispatch (RFC §5.3).
	if df := th.DrawFunc(theme.WidgetKindTextField); df != nil {
		df(theme.DrawCtx{
			Canvas:  canvas,
			Bounds:  draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			Focused: focused,
		}, tokens, node)
	} else {
		tfRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

		// Border
		canvas.FillRoundRect(tfRect,
			tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Stroke.Border))

		// Fill
		canvas.FillRoundRect(
			draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
			maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

		// Focus glow + ring (RFC-008 §9.4)
		if focused {
			drawFocusRing(canvas, tfRect, tokens.Radii.Input, tokens)
		}

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
	}

	// Store input state for the focused TextField so the framework can
	// handle KeyMsg/CharMsg internally (no userland boilerplate needed).
	if focused && node.OnChange != nil && focus != nil {
		focus.Input = &InputState{
			Value:    node.Value,
			OnChange: node.OnChange,
			FocusUID: focusUID,
		}
	}

	// Hit target for focus acquisition.
	if node.OnChange != nil && focus != nil {
		uid := focusUID
		fm := focus
		ix.RegisterHit(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			func() { fm.SetFocusedUID(uid) })
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutSelect(node selectElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := textFieldW
	if area.W < w {
		w = area.W
	}

	// Register hit target and get hover opacity atomically.
	selectRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	var selectClickFn func()
	if node.State != nil {
		state := node.State
		selectClickFn = func() { state.Open = !state.Open }
	}
	hoverOpacity := ix.RegisterHit(selectRect, selectClickFn)

	isOpen := node.State != nil && node.State.Open

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	// Custom theme DrawFunc dispatch (RFC §5.3).
	if df := th.DrawFunc(theme.WidgetKindSelect); df != nil {
		df(theme.DrawCtx{
			Canvas:  canvas,
			Bounds:  selectRect,
			Hovered: hoverOpacity > 0,
			Focused: focused,
		}, tokens, node)
	} else {
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

		// Focus glow (RFC-008 §9.4).
		if focused || isOpen {
			drawFocusRing(canvas, selectRect, tokens.Radii.Input, tokens)
		}
	}

	// Dropdown overlay when open.
	if isOpen && len(node.Options) > 0 {
		dropX := area.X
		dropY := area.Y + h
		dropW := w
		opts := node.Options
		onSelect := node.OnSelect
		state := node.State
		overlays.push(overlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) {
				itemH := int(tokens.Typography.Body.Size) + textFieldPadY*2
				totalH := itemH * len(opts)

				// Dropdown background.
				canvas.FillRoundRect(
					draw.R(float32(dropX), float32(dropY), float32(dropW), float32(totalH)),
					tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
				// Dropdown border.
				canvas.StrokeRoundRect(
					draw.R(float32(dropX), float32(dropY), float32(dropW), float32(totalH)),
					tokens.Radii.Input, draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

				for i, opt := range opts {
					itemY := dropY + i*itemH
					o := opt
					var itemClickFn func()
					if onSelect != nil || state != nil {
						itemClickFn = func() {
							if onSelect != nil {
								onSelect(o)
							}
							if state != nil {
								state.Open = false
							}
						}
					}
					ho := ix.RegisterHit(draw.R(float32(dropX), float32(itemY), float32(dropW), float32(itemH)), itemClickFn)
					if ho > 0 {
						canvas.FillRect(
							draw.R(float32(dropX+1), float32(itemY), float32(max(dropW-2, 0)), float32(itemH)),
							draw.SolidPaint(tokens.Colors.Surface.Hovered))
					}
					canvas.DrawText(opt,
						draw.Pt(float32(dropX+textFieldPadX), float32(itemY+textFieldPadY)),
						tokens.Typography.Body, tokens.Colors.Text.Primary)
				}
			},
		})
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutPadding(node paddingElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus ...*FocusManager) bounds {
	var fs *FocusManager
	if len(focus) > 0 {
		fs = focus[0]
	}
	// Resolve logical Start/End insets to physical Left/Right (RFC-002 §4.6).
	left, right := node.Insets.Resolve(globalDirection)
	inL := int(left)
	inT := int(node.Insets.Top)
	inR := int(right)
	inB := int(node.Insets.Bottom)
	childArea := bounds{
		X: area.X + inL,
		Y: area.Y + inT,
		W: max(area.W-inL-inR, 0),
		H: max(area.H-inT-inB, 0),
	}
	cb := layoutElement(node.Child, childArea, canvas, th, tokens, ix, overlays, fs)
	return bounds{X: area.X, Y: area.Y, W: cb.W + inL + inR, H: cb.H + inT + inB, Baseline: inT + cb.Baseline}
}

func layoutSizedBox(node sizedBoxElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus ...*FocusManager) bounds {
	var fs *FocusManager
	if len(focus) > 0 {
		fs = focus[0]
	}
	w := int(node.Width)
	h := int(node.Height)
	// Zero means "inherit from parent area" so callers can constrain
	// only one dimension (e.g. SizedBox(0, 120, child) for height-only).
	if w == 0 {
		w = area.W
	}
	if h == 0 {
		h = area.H
	}
	var baseline int
	if node.Child != nil {
		childArea := bounds{X: area.X, Y: area.Y, W: w, H: h}
		cb := layoutElement(node.Child, childArea, canvas, th, tokens, ix, overlays, fs)
		baseline = cb.Baseline
	}
	if baseline == 0 {
		baseline = h
	}
	return bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: baseline}
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float32) float32 {
	if a < b {
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
	cardPadding      = 16
	cardBorder       = 1
	tabHeaderPadX    = 16
	tabHeaderPadY    = 10
	tabIndicatorH    = 2
	accordionHeaderH = 36
	tooltipPadding   = 8
	badgePadX        = 6
	badgePadY        = 2
	badgeMinSize     = 20
	chipPadX         = 12
	chipPadY         = 6
	chipDismissW     = 16
	menuBarHeight    = 32
	menuBarItemPadX  = 12
	menuItemHeight   = 32
	menuItemPadX     = 12
)

// ── Tier 3 Layout Functions ──────────────────────────────────────

func layoutCard(node cardElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	// Measure child to determine card size.
	nc := nullCanvas{delegate: canvas}
	childArea := bounds{X: area.X + cardPadding, Y: area.Y + cardPadding, W: max(area.W-cardPadding*2, 0), H: max(area.H-cardPadding*2, 0)}
	cb := layoutElement(node.Child, childArea, nc, th, tokens, nil, nil, nil)

	w := cb.W + cardPadding*2
	h := cb.H + cardPadding*2
	if w > area.W {
		w = area.W
	}

	// Border
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
		tokens.Radii.Card, draw.SolidPaint(tokens.Colors.Surface.Pressed))

	// Fill
	canvas.FillRoundRect(
		draw.R(float32(area.X+cardBorder), float32(area.Y+cardBorder), float32(max(w-cardBorder*2, 0)), float32(max(h-cardBorder*2, 0))),
		maxf(tokens.Radii.Card-cardBorder, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Child content
	layoutElement(node.Child, childArea, canvas, th, tokens, ix, overlays, focus)

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutTabs(node tabsElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
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
		cb := layoutElement(item.Header, bounds{X: 0, Y: 0, W: area.W, H: area.H}, nc, th, tokens, nil, nil, nil)
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

		// Register tab hit target and get hover opacity.
		var hoverOpacity float32
		if node.OnSelect != nil {
			idx := i
			onSelect := node.OnSelect
			hoverOpacity = ix.RegisterHit(draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH)),
				func() { onSelect(idx) })
		}

		// Tab background — selected tab gets subtle highlight; hover blends on top.
		if i == selected {
			canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH)),
				draw.SolidPaint(tokens.Colors.Surface.Hovered))
		} else if hoverOpacity > 0 {
			hc := tokens.Colors.Surface.Hovered
			hc.A *= hoverOpacity
			canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH)),
				draw.SolidPaint(hc))
		}

		// Tab header content
		headerArea := bounds{X: cursorX + tabHeaderPadX, Y: area.Y + tabHeaderPadY, W: max(tw-tabHeaderPadX*2, 0), H: max(headerH-tabHeaderPadY*2, 0)}
		layoutElement(item.Header, headerArea, canvas, th, tokens, ix, overlays, focus)

		// Selection indicator (underline)
		if i == selected {
			canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y+headerH-tabIndicatorH), float32(tw), float32(tabIndicatorH)),
				draw.SolidPaint(tokens.Colors.Accent.Primary))
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
	cb := layoutElement(node.Items[selected].Content, contentArea, canvas, th, tokens, ix, overlays, focus)

	totalH := headerH + 1 + columnGap + cb.H
	totalW := max(totalHeaderW, cb.W)
	return bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

func layoutAccordion(node accordionElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
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

		// Register hit target and get hover opacity atomically.
		var hoverOpacity float32
		if node.State != nil {
			idx := i
			state := node.State
			hoverOpacity = ix.RegisterHit(draw.R(float32(area.X), float32(cursorY), float32(area.W), float32(accordionHeaderH)),
				func() { state.Expanded[idx] = !state.Expanded[idx] })
		}

		// Header background (with hover blend)
		hdrColor := tokens.Colors.Surface.Elevated
		if hoverOpacity > 0 {
			hdrColor = lerpColor(hdrColor, tokens.Colors.Surface.Hovered, hoverOpacity)
		}
		canvas.FillRect(
			draw.R(float32(area.X), float32(cursorY), float32(area.W), float32(accordionHeaderH)),
			draw.SolidPaint(hdrColor))

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
		layoutElement(section.Header, headerArea, canvas, th, tokens, ix, overlays, focus)

		if area.W > maxW {
			maxW = area.W
		}
		cursorY += accordionHeaderH

		// Content (if expanded)
		if expanded {
			contentArea := bounds{X: area.X + cardPadding, Y: cursorY + 8, W: max(area.W-cardPadding*2, 0), H: max(area.H-(cursorY-area.Y)-8, 0)}
			cb := layoutElement(section.Content, contentArea, canvas, th, tokens, ix, overlays, focus)
			cursorY += cb.H + 16 // 8 top + 8 bottom padding
		}
	}

	return bounds{X: area.X, Y: area.Y, W: maxW, H: cursorY - area.Y}
}

func layoutTooltip(node tooltipElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	// Layout trigger normally.
	triggerBounds := layoutElement(node.Trigger, area, canvas, th, tokens, ix, overlays, focus)

	// Determine visibility: explicit or hover-based.
	visible := node.Visible
	if !visible {
		// Register trigger as hover target so the hover system tracks it.
		hoverOpacity := ix.RegisterHit(draw.R(float32(triggerBounds.X), float32(triggerBounds.Y),
			float32(triggerBounds.W), float32(triggerBounds.H)), nil)
		visible = hoverOpacity > 0.1
	}

	if visible {
		tB := triggerBounds
		content := node.Content
		overlays.push(overlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) {
				// Measure content
				nc := nullCanvas{delegate: canvas}
				cb := layoutElement(content, bounds{X: 0, Y: 0, W: 300, H: 200}, nc, th, tokens, nil, nil, nil)

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
				layoutElement(content, bounds{X: x + tooltipPadding, Y: y + tooltipPadding, W: max(w-tooltipPadding*2, 0), H: max(h-tooltipPadding*2, 0)}, canvas, th, tokens, ix, nil, nil)
			},
		})
	}

	return triggerBounds
}

func layoutBadge(node badgeElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	// Measure content
	nc := nullCanvas{delegate: canvas}
	cb := layoutElement(node.Content, bounds{X: 0, Y: 0, W: area.W, H: area.H}, nc, th, tokens, nil, nil, nil)

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
	bgColor := tokens.Colors.Accent.Primary
	if node.Color.A > 0 {
		bgColor = node.Color
	}
	radius := minf(tokens.Radii.Pill, float32(min(w, h))/2)
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
		radius, draw.SolidPaint(bgColor))

	// Content (centered)
	contentX := area.X + (w-cb.W)/2
	contentY := area.Y + (h-cb.H)/2
	layoutElement(node.Content, bounds{X: contentX, Y: contentY, W: cb.W, H: cb.H}, canvas, th, tokens, ix, overlays, focus)

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutChip(node chipElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	// Measure label
	nc := nullCanvas{delegate: canvas}
	cb := layoutElement(node.Label, bounds{X: 0, Y: 0, W: area.W, H: area.H}, nc, th, tokens, nil, nil, nil)

	labelW := cb.W
	dismissW := 0
	if node.OnDismiss != nil {
		dismissW = chipDismissW
	}
	w := labelW + chipPadX*2 + dismissW
	h := cb.H + chipPadY*2

	// Register chip click target and get hover opacity atomically.
	var chipClickFn func()
	chipClickW := w
	if node.OnClick != nil {
		chipClickFn = node.OnClick
		if node.OnDismiss != nil {
			chipClickW = w - dismissW // exclude dismiss area
		}
	}
	hoverOpacity := ix.RegisterHit(draw.R(float32(area.X), float32(area.Y), float32(chipClickW), float32(h)), chipClickFn)

	// Background
	var bgColor, borderColor draw.Color
	if node.Selected {
		bgColor = tokens.Colors.Accent.Primary
		borderColor = tokens.Colors.Accent.Primary
	} else {
		bgColor = tokens.Colors.Surface.Hovered
		borderColor = tokens.Colors.Surface.Pressed
	}
	if hoverOpacity > 0 {
		bgColor = lerpColor(bgColor, hoverHighlight(bgColor), hoverOpacity)
	}

	radius := minf(tokens.Radii.Pill, float32(min(w, h))/2)
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
		radius, draw.SolidPaint(borderColor))
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(radius-1, 0), draw.SolidPaint(bgColor))

	// Label content
	labelArea := bounds{X: area.X + chipPadX, Y: area.Y + chipPadY, W: labelW, H: cb.H}
	layoutElement(node.Label, labelArea, canvas, th, tokens, ix, overlays, focus)

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

		// Register dismiss hit target.
		ix.RegisterHit(draw.R(float32(dismissX), float32(area.Y), float32(chipDismissW), float32(h)),
			node.OnDismiss)
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func layoutMenuBar(node menuBarElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	if len(node.Items) == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	nc := nullCanvas{delegate: canvas}

	// Backdrop: when a dropdown is open, a full-screen hit target closes it
	// on any click outside menu bar items or dropdown items.
	if node.State != nil && node.State.OpenIndex >= 0 {
		state := node.State
		ix.RegisterHit(draw.R(0, 0, 9999, 9999), func() {
			state.OpenIndex = -1
		})
	}

	// Background strip
	canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(menuBarHeight)),
		draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Bottom border
	canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y+menuBarHeight-1), float32(area.W), 1),
		draw.SolidPaint(tokens.Colors.Stroke.Border))

	cursorX := area.X

	for i, item := range node.Items {
		// Measure label
		cb := layoutElement(item.Label, bounds{X: 0, Y: 0, W: area.W, H: menuBarHeight}, nc, th, tokens, nil, nil, nil)
		itemW := cb.W + menuBarItemPadX*2

		hasAction := item.OnClick != nil || len(item.Items) > 0

		// Register hit target and get hover opacity atomically.
		var hoverOpacity float32
		if hasAction {
			idx := i
			state := node.State
			subItems := item.Items
			onClick := item.OnClick
			hoverOpacity = ix.RegisterHit(draw.R(float32(cursorX), float32(area.Y), float32(itemW), float32(menuBarHeight)),
				func() {
					if len(subItems) > 0 && state != nil {
						if state.OpenIndex == idx {
							state.OpenIndex = -1
						} else {
							state.OpenIndex = idx
						}
					}
					if onClick != nil {
						onClick()
					}
				})
		}

		// Active highlight for open menu
		isOpen := node.State != nil && node.State.OpenIndex == i
		if isOpen || hoverOpacity > 0 {
			op := hoverOpacity
			if isOpen {
				op = 1.0
			}
			canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y), float32(itemW), float32(menuBarHeight)),
				draw.SolidPaint(lerpColor(tokens.Colors.Surface.Elevated, tokens.Colors.Surface.Hovered, op)))
		}

		// Draw label
		labelArea := bounds{X: cursorX + menuBarItemPadX, Y: area.Y + (menuBarHeight-cb.H)/2, W: cb.W, H: cb.H}
		layoutElement(item.Label, labelArea, canvas, th, tokens, ix, overlays, focus)

		// Dropdown overlay for open submenu
		if isOpen && len(item.Items) > 0 {
			dropdownX := cursorX
			dropdownY := area.Y + menuBarHeight
			subItems := item.Items
			overlays.push(overlayEntry{
				Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) {
					layoutMenuDropdown(subItems, dropdownX, dropdownY, nc, canvas, th, tokens, ix)
				},
			})
		}

		cursorX += itemW
	}

	return bounds{X: area.X, Y: area.Y, W: area.W, H: menuBarHeight}
}

// layoutMenuDropdown renders a dropdown menu at the given position.
// Shared by MenuBar dropdowns and potentially nested menus.
func layoutMenuDropdown(items []MenuItem, posX, posY int, nc nullCanvas, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor) {
	// Measure all items.
	maxItemW := 0
	for _, item := range items {
		cb := layoutElement(item.Label, bounds{X: 0, Y: 0, W: 300, H: menuItemHeight}, nc, th, tokens, nil, nil, nil)
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
	cornerR := maxf(tokens.Radii.Card-1, 0)
	for itemIdx, item := range items {
		// Register hit target and get hover opacity atomically.
		// Always register (even for nil OnClick) to keep hover indices aligned
		// and to block hover events from reaching underlying elements.
		hoverOpacity := ix.RegisterHit(draw.R(float32(posX), float32(cursorY), float32(menuW), float32(menuItemHeight)),
			item.OnClick)
		if hoverOpacity > 0 {
			hoverColor := draw.SolidPaint(lerpColor(tokens.Colors.Surface.Elevated, tokens.Colors.Surface.Hovered, hoverOpacity))
			hoverRect := draw.R(float32(posX+1), float32(cursorY), float32(max(menuW-2, 0)), float32(menuItemHeight))
			if itemIdx == 0 || itemIdx == len(items)-1 {
				canvas.FillRoundRect(hoverRect, cornerR, hoverColor)
			} else {
				canvas.FillRect(hoverRect, hoverColor)
			}
		}

		labelArea := bounds{X: posX + menuItemPadX, Y: cursorY + (menuItemHeight-16)/2, W: max(menuW-menuItemPadX*2, 0), H: 16}
		layoutElement(item.Label, labelArea, canvas, th, tokens, ix, nil, nil)

		cursorY += menuItemHeight
	}
}

func layoutContextMenu(node contextMenuElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	if !node.Visible || len(node.Items) == 0 || overlays == nil {
		return bounds{X: area.X, Y: area.Y}
	}

	nc := nullCanvas{delegate: canvas}
	items := node.Items
	// Anchor relative to the element's layout area.
	posX := area.X + int(node.PosX)
	posY := area.Y + int(node.PosY)
	winW, winH := overlays.windowW, overlays.windowH

	// Push overlay for context menu rendering.
	overlays.push(overlayEntry{
		Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) {
			// Measure dropdown size for clamping.
			maxItemW := 0
			for _, item := range items {
				cb := layoutElement(item.Label, bounds{X: 0, Y: 0, W: 300, H: menuItemHeight}, nc, th, tokens, nil, nil, nil)
				w := cb.W + menuItemPadX*2
				if w > maxItemW {
					maxItemW = w
				}
			}
			if maxItemW < 120 {
				maxItemW = 120
			}
			menuW := maxItemW
			menuH := len(items) * menuItemHeight

			// Clamp to window bounds so the menu stays fully visible.
			clampedX := posX
			clampedY := posY
			if clampedX+menuW > winW {
				clampedX = winW - menuW
			}
			if clampedX < 0 {
				clampedX = 0
			}
			if clampedY+menuH > winH {
				clampedY = winH - menuH
			}
			if clampedY < 0 {
				clampedY = 0
			}

			layoutMenuDropdown(items, clampedX, clampedY, nc, canvas, th, tokens, ix)
		},
	})

	return bounds{X: area.X, Y: area.Y}
}

func layoutOverlay(node Overlay, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	if node.Content == nil || overlays == nil {
		return bounds{X: area.X, Y: area.Y}
	}

	content := node.Content
	anchor := node.Anchor
	placement := node.Placement
	dismissable := node.Dismissable
	onDismiss := node.OnDismiss
	backdrop := node.Backdrop
	winW, winH := overlays.windowW, overlays.windowH

	overlays.push(overlayEntry{
		Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) {
			// Draw semi-transparent scrim behind the overlay for modal dialogs.
			if backdrop {
				canvas.FillRect(draw.R(0, 0, float32(winW), float32(winH)),
					draw.SolidPaint(tokens.Colors.Surface.Scrim))
			}

			// If dismissable, register a full-window backdrop hit target.
			// Added BEFORE content targets so content takes priority (hitMap is LIFO).
			if dismissable && onDismiss != nil {
				ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), onDismiss)
			}

			// Measure content with null canvas.
			nc := nullCanvas{delegate: canvas}
			cb := layoutElement(content, bounds{X: 0, Y: 0, W: 400, H: 300}, nc, th, tokens, nil, nil, nil)

			pad := 8
			contentSize := draw.Size{W: float32(cb.W + pad*2), H: float32(cb.H + pad*2)}

			// Compute position using the overlay placement logic.
			pos := ComputeOverlayPosition(anchor, placement, contentSize, winW, winH)

			// Draw border.
			overlayRect := draw.R(pos.X, pos.Y, contentSize.W, contentSize.H)
			canvas.FillRoundRect(overlayRect, tokens.Radii.Card, draw.SolidPaint(tokens.Colors.Stroke.Border))

			// Draw elevated surface fill.
			inner := draw.R(pos.X+1, pos.Y+1, contentSize.W-2, contentSize.H-2)
			canvas.FillRoundRect(inner, maxf(tokens.Radii.Card-1, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

			// Layout content inside the overlay.
			layoutElement(content, bounds{
				X: int(pos.X) + pad, Y: int(pos.Y) + pad,
				W: max(int(contentSize.W)-pad*2, 0), H: max(int(contentSize.H)-pad*2, 0),
			}, canvas, th, tokens, ix, nil, focus)
		},
	})

	// Overlays take no space in normal layout flow.
	return bounds{X: area.X, Y: area.Y}
}
