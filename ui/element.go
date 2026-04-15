// Package ui defines the Widget system and Element types for the
// virtual tree (RFC §4).
package ui

import (
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/interaction"
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
	UID                UID
	Theme              theme.Theme
	Send               func(any)                       // local Send bound to this UID
	Events             []InputEvent                     // input events dispatched to this widget (RFC-002 §2.6)
	Locale             string                           // BCP 47 language tag, e.g. "de", "en-US" (RFC-003 §3.8)
	InteractionProfile *interaction.InteractionProfile  // active profile, nil = desktop default (RFC-004 §2.4)
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
	ID           SurfaceNodeID
	Parent       SurfaceNodeID // 0 = root within the surface.
	Role         a11y.AccessRole
	Label        string
	Description  string
	Value        string
	Bounds       draw.Rect // Relative to the surface in dp.
	Lang         string    // BCP 47 language tag (e.g. "de", "ar-EG"). Empty inherits from parent.
	States       a11y.AccessStates
	Actions      []a11y.AccessActionDesc
	Relations    []a11y.AccessRelationDesc
	NumericValue *a11y.AccessNumericValue // Non-nil for nodes with numeric range.
	TextState    *a11y.AccessTextState    // Non-nil for editable or selectable text nodes.
}

// ── Element Types (RFC §4.3) ─────────────────────────────────────

// Element is the base interface for all virtual-tree nodes.
type Element interface {
	isElement()
}

// BaseElement is an embeddable type that satisfies the Element interface.
// Sub-packages embed this to create Element types without accessing
// the unexported isElement() method directly.
type BaseElement struct{}

func (BaseElement) isElement() {}

// emptyElement renders nothing. It is defined in the core ui package so
// callers that cannot import ui/display (or want the short ui.Empty() form)
// have access to a zero-cost empty node.
type emptyElement struct{ BaseElement }

func (n emptyElement) LayoutSelf(ctx *LayoutContext) Bounds {
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
}

func (n emptyElement) TreeEqual(other Element) bool {
	_, ok := other.(emptyElement)
	return ok
}

func (n emptyElement) ResolveChildren(resolve func(Element, int) Element) Element { return n }

func (n emptyElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32) {}

// Empty returns an Element that renders nothing.
func Empty() Element { return emptyElement{} }

// Layouter is an optional interface on Element types. When implemented,
// layoutElement dispatches to LayoutSelf instead of the central type switch.
// Sub-packages implement this so their element types are self-laying-out.
type Layouter interface {
	Element
	LayoutSelf(ctx *LayoutContext) Bounds
}

// TreeEqualizer is an optional interface on Element types for structural
// comparison. When implemented, treeEqual dispatches to TreeEqual instead
// of the central type switch.
type TreeEqualizer interface {
	Element
	TreeEqual(other Element) bool
}

// ChildResolver is an optional interface on Element types for recursive
// widget resolution. Container elements return a copy with resolved children;
// leaf elements return themselves unchanged.
type ChildResolver interface {
	Element
	ResolveChildren(resolve func(el Element, index int) Element) Element
}

// AccessWalker is an optional interface on Element types for building
// the accessibility tree. When implemented, the access tree builder
// dispatches to WalkAccess instead of the central type switch.
type AccessWalker interface {
	Element
	WalkAccess(b *AccessTreeBuilder, parentIdx int32)
}

// LayoutAxis controls how a Box arranges its children.
type LayoutAxis int

const (
	AxisColumn LayoutAxis = iota
	AxisRow
)

// Labeler is an optional interface on Element types. Elements that hold
// a text label (TextElement and similar) implement this so code outside
// the defining package can extract the label without importing it.
type Labeler interface {
	Element
	ElementLabel() string
}

// WithKey wraps an element with an explicit key for stable UIDs
// across re-parenting (RFC §4.4).
func WithKey(key string, el Element) Element {
	return KeyedElement{Key: key, Child: el}
}

// ── External Surfaces (RFC §8) ───────────────────────────────────

// Surface creates a surface slot element that renders GPU content from a
// SurfaceProvider (RFC §8). The width and height specify the desired size
// in dp. If provider is nil, a placeholder rectangle is rendered.
func Surface(id SurfaceID, provider SurfaceProvider, width, height float32) Element {
	return SurfaceElement{ID: id, Provider: provider, Width: width, Height: height}
}

// Component creates an element that wraps a Widget. The Reconciler
// expands it by calling Widget.Render with persisted state.
func Component(w Widget) Element {
	return WidgetElement{W: w}
}

// ComponentWithKey creates a keyed widget element. The key stabilises the
// widget's UID across re-ordering within the same parent.
func ComponentWithKey(key string, w Widget) Element {
	return WidgetElement{W: w, Key: key}
}

// ── Framework Element Structs ────────────────────────────────────

type KeyedElement struct {
	Key   string
	Child Element
}

func (KeyedElement) isElement() {}

type SurfaceElement struct {
	ID       SurfaceID
	Provider SurfaceProvider
	Width    float32
	Height   float32
}

func (SurfaceElement) isElement() {}

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

// WidgetElement wraps a Widget for embedding in element trees.
// It is expanded by the Reconciler before layout.
// ThemedElement overrides the theme for its child subtree (scoped theme).
// The Reconciler replaces the active theme before resolving children,
// so all descendants inherit the overridden tokens and draw functions.
type ThemedElement struct {
	Theme    theme.Theme
	Children []Element
}

func (ThemedElement) isElement() {}

// Themed creates a scoped theme override. All children inherit the given
// theme instead of the ambient one. Combine with theme.Override() to
// create partial overrides (e.g. danger-colored buttons).
func Themed(th theme.Theme, children ...Element) Element {
	return ThemedElement{Theme: th, Children: children}
}

type WidgetElement struct {
	W   Widget
	Key string
}

func (WidgetElement) isElement() {}

// WidgetBoundsElement wraps a widget's resolved element subtree,
// carrying the widget UID so that layout can register screen bounds
// for event dispatching (RFC-002 §2.6).
type WidgetBoundsElement struct {
	WidgetUID UID
	Child     Element
}

func (WidgetBoundsElement) isElement() {}

// ScrollState tracks scroll offset for ScrollView elements.
type ScrollState struct {
	Offset   float32 // current vertical scroll offset in dp
	Velocity float32 // scroll velocity for momentum

	// AutoScrollVH stores the viewport height at which auto-scroll was
	// last applied for a focused element. While focus persists at the
	// same viewport height the user can scroll freely without the
	// auto-scroll pulling the view back. Reset to -1 when focus is lost
	// so the next focus gain triggers a fresh auto-scroll.
	AutoScrollVH float32
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

// ── Safe Area Insets ────────────────────────────────────────────

// SafeAreaInsets describes regions of the viewport that are partially
// obscured by system UI (on-screen keyboard, notch, status bar, etc.).
// Layouts should avoid placing interactive content within these insets.
// Values are in dp. Zero means no inset on that edge.
type SafeAreaInsets struct {
	Top, Right, Bottom, Left float32
	NoFramePadding           bool // skip frame padding (e.g. no-compositor tab panel)
}

// IsZero reports whether all insets are zero.
func (s SafeAreaInsets) IsZero() bool {
	return s.Top == 0 && s.Right == 0 && s.Bottom == 0 && s.Left == 0
}

// ── Focus State ──────────────────────────────────────────────────

// InputState tracks the focused TextField's value and callback so that
// the framework can handle KeyMsg/CharMsg internally without exposing
// raw input events to userland.
type InputState struct {
	Value        string
	OnChange     func(string)
	FocusUID     UID
	CursorOffset int  // byte offset of cursor within Value
	Multiline    bool // true for TextArea: Enter inserts \n, Up/Down navigate lines

	// Selection state: SelectionStart is the anchor byte offset.
	// -1 means no active selection. The selected range is
	// [min(SelectionStart, CursorOffset), max(SelectionStart, CursorOffset)).
	SelectionStart int

	// SuppressOSK, when true, prevents the framework from auto-showing the
	// global on-screen keyboard for this input. Used by widgets that provide
	// their own inline keypad (e.g. NumericInput).
	SuppressOSK bool

	// IME composition state (RFC-002 §2.2).
	ComposeText        string // current pre-edit text (empty when not composing)
	ComposeCursorStart int    // cursor position within compose text (rune index)
	ComposeCursorEnd   int    // selection end within compose text (rune index)
}

// HasSelection reports whether there is an active text selection.
func (is *InputState) HasSelection() bool {
	return is.SelectionStart >= 0 && is.SelectionStart != is.CursorOffset
}

// SelectionRange returns the ordered (start, end) byte offsets of the selection.
func (is *InputState) SelectionRange() (int, int) {
	a, b := is.SelectionStart, is.CursorOffset
	if a > b {
		a, b = b, a
	}
	return a, b
}

// SelectedText returns the currently selected text.
func (is *InputState) SelectedText() string {
	if !is.HasSelection() {
		return ""
	}
	a, b := is.SelectionRange()
	return is.Value[a:b]
}

// DeleteSelection removes the selected text, updates Value and CursorOffset,
// and clears the selection. Returns true if a deletion occurred.
func (is *InputState) DeleteSelection() bool {
	if !is.HasSelection() {
		return false
	}
	a, b := is.SelectionRange()
	is.Value = is.Value[:a] + is.Value[b:]
	is.CursorOffset = a
	is.SelectionStart = -1
	return true
}

// ClearSelection clears the selection without deleting text.
func (is *InputState) ClearSelection() {
	is.SelectionStart = -1
}

// FocusState is a type alias for backward compatibility.
// Deprecated: use FocusManager directly.
type FocusState = FocusManager

// ── Hover State (M4) ────────────────────────────────────────────

// HoverState tracks hover animations for interactive elements.
// It uses the previous frame's hit targets to determine hover,
// introducing at most one frame of latency (imperceptible at 60fps).
type HoverState struct {
	hoveredIdx int                  // currently hovered button index, -1 = none
	anims      []anim.Anim[float32] // per-button hover opacity [0,1]
	ripples    []RippleState        // per-button touch ripple state
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

// Tick advances all hover and ripple animations by dt.
// Returns true if any animation is still running.
func (h *HoverState) Tick(dt time.Duration) bool {
	running := false
	for i := range h.anims {
		if h.anims[i].Tick(dt) {
			running = true
		}
	}
	for i := range h.ripples {
		if h.ripples[i].Tick(dt) {
			running = true
		}
	}
	return running
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

// currentButtonRipple returns a pointer to the ripple for the button that
// was most recently allocated via nextButtonHoverOpacity. The caller must
// have called nextButtonHoverOpacity before this.
func (h *HoverState) currentButtonRipple() *RippleState {
	idx := h.buttonIdx - 1 // buttonIdx was already incremented
	if idx < 0 {
		return nil
	}
	for len(h.ripples) <= idx {
		h.ripples = append(h.ripples, RippleState{})
	}
	return &h.ripples[idx]
}

func (h *HoverState) ensureSize(n int) {
	for len(h.anims) < n {
		h.anims = append(h.anims, anim.Anim[float32]{})
	}
}

// Trim discards hover animations beyond n entries, preventing stale
// animations from accumulating when the element tree shrinks.
func (h *HoverState) Trim(n int) {
	if len(h.anims) > n {
		h.anims = h.anims[:n]
	}
}

// ── Overlay System (Tier 3) ──────────────────────────────────────

// OverlayEntry is a deferred render operation drawn after the main tree.
// Used by Tooltip, ContextMenu, and MenuBar for correct Z-order.
type OverlayEntry struct {
	Render    func(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor)
	Animation OverlayAnimation     // enter/exit animation type
	Duration  theme.DurationEasing // animation timing (RFC-008 §9.5)
}

// OverlayStack collects overlay entries during layout.
type OverlayStack struct {
	Entries []OverlayEntry
	WindowW int
	WindowH int
}

// Push adds an overlay entry to the stack.
func (s *OverlayStack) Push(entry OverlayEntry) {
	if s != nil {
		s.Entries = append(s.Entries, entry)
	}
}

// ── Layout & Scene Building ──────────────────────────────────────
// BuildScene converts an Element tree into draw commands via the
// Canvas interface (RFC §6).

// Bounds represents the computed position and size of an element after layout.
type Bounds struct{ X, Y, W, H, Baseline int }

const (
	FramePadding    = 24
	ColumnGap       = 16
	RowGap          = 12
	ButtonPadX      = 18
	ButtonPadY      = 12
	ButtonBorder    = 1
	IconButtonPad   = 10
	SplitArrowWidth = 36
	SegmentPadX     = 16
	SegmentPadY     = 10
)

// Keep unexported aliases for backward compatibility within this package.
const (
	framePadding    = FramePadding
	columnGap       = ColumnGap
	rowGap          = RowGap
	buttonPadX      = ButtonPadX
	buttonPadY      = ButtonPadY
	buttonBorder    = ButtonBorder
	iconButtonPad   = IconButtonPad
	splitArrowWidth = SplitArrowWidth
	segmentPadX     = SegmentPadX
	segmentPadY     = SegmentPadY
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
	area := Bounds{X: framePadding, Y: framePadding, W: max(width-(framePadding*2), 0), H: max(height-(framePadding*2), 0)}
	var overlays OverlayStack
	overlays.WindowW = width
	overlays.WindowH = height
	layoutElement(root, area, canvas, th, tokens, ix, &overlays, focus)

	// Switch canvas to overlay mode so overlay draw commands go to
	// separate scene lists, rendered after all main content.
	type overlayModeSetter interface{ SetOverlayMode(bool) }
	if oms, ok := canvas.(overlayModeSetter); ok {
		oms.SetOverlayMode(true)
	}
	// Render overlay entries (Tooltip, ContextMenu, etc.) on top of main tree.
	for _, entry := range overlays.Entries {
		entry.Render(canvas, tokens, ix)
	}
	// Render drag-and-drop preview ghost on top of all overlays (RFC-005 §10).
	if ix != nil && ix.DnD != nil && ix.DnD.IsActive() {
		renderDnDPreview(ix.DnD, canvas, th, tokens, ix, width, height)
	}
	if oms, ok := canvas.(overlayModeSetter); ok {
		oms.SetOverlayMode(false)
	}

	// The canvas is a SceneCanvas — retrieve its scene.
	type scener interface{ Scene() draw.Scene }
	if sc, ok := canvas.(scener); ok {
		s := sc.Scene()
		s.Grain = tokens.Grain // RFC-008 §10.5
		return s
	}
	return draw.Scene{}
}

// BuildSceneWithOSK is like BuildScene but also renders an on-screen keyboard
// overlay element after all other overlays (RFC-004 §5.5).
// If oskOverlay is nil, it behaves identically to BuildScene.
// safeArea propagates viewport insets (e.g. OSK height) through the layout tree.
func BuildSceneWithOSK(root Element, canvas draw.Canvas, th theme.Theme, width, height int, ix *Interactor, focus *FocusManager, oskOverlay Element, profile *interaction.InteractionProfile, safeArea ...SafeAreaInsets) draw.Scene {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}

	ix.resetCounter()

	tokens := th.Tokens()
	if profile != nil && profile.ScaleTypography != 0 && profile.ScaleTypography != 1.0 {
		tokens.Typography = tokens.Typography.Scaled(profile.ScaleTypography)
	}
	var sa SafeAreaInsets
	if len(safeArea) > 0 {
		sa = safeArea[0]
	}
	pad := framePadding
	if sa.NoFramePadding {
		pad = 0
	}
	area := Bounds{X: pad, Y: pad, W: max(width-(pad*2), 0), H: max(height-(pad*2), 0)}
	var overlays OverlayStack
	overlays.WindowW = width
	overlays.WindowH = height
	layoutElementCtx(root, area, canvas, th, tokens, ix, &overlays, focus, profile, sa)

	// Switch canvas to overlay mode so overlay draw commands go to
	// separate scene lists, rendered after all main content.
	type overlayModeSetter interface{ SetOverlayMode(bool) }
	if oms, ok := canvas.(overlayModeSetter); ok {
		oms.SetOverlayMode(true)
	}
	// Render overlay entries (Tooltip, ContextMenu, etc.) on top of main tree.
	for _, entry := range overlays.Entries {
		entry.Render(canvas, tokens, ix)
	}
	// Render the OSK overlay on top of everything (RFC-004 §5.5).
	if oskOverlay != nil {
		if l, ok := oskOverlay.(Layouter); ok {
			bounds := canvas.Bounds()
			oskCtx := &LayoutContext{
				Area:   Bounds{X: 0, Y: 0, W: int(bounds.W), H: int(bounds.H)},
				Canvas: canvas,
				Theme:  th,
				Tokens: tokens,
				IX:     ix,
			}
			l.LayoutSelf(oskCtx)
		}
	}
	// Render drag-and-drop preview ghost on top of all overlays (RFC-005 §10).
	if ix != nil && ix.DnD != nil && ix.DnD.IsActive() {
		renderDnDPreview(ix.DnD, canvas, th, tokens, ix, width, height)
	}
	if oms, ok := canvas.(overlayModeSetter); ok {
		oms.SetOverlayMode(false)
	}

	// The canvas is a SceneCanvas — retrieve its scene.
	type scener interface{ Scene() draw.Scene }
	if sc, ok := canvas.(scener); ok {
		s := sc.Scene()
		s.Grain = tokens.Grain // RFC-008 §10.5
		return s
	}
	return draw.Scene{}
}

func layoutElement(el Element, area Bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *OverlayStack, focus *FocusManager, profileOpt ...*interaction.InteractionProfile) Bounds {
	var profile *interaction.InteractionProfile
	if len(profileOpt) > 0 {
		profile = profileOpt[0]
	}
	return layoutElementCtx(el, area, canvas, th, tokens, ix, overlays, focus, profile, SafeAreaInsets{})
}

// layoutElementCtx is the core layout dispatch that also propagates SafeAreaInsets.
func layoutElementCtx(el Element, area Bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *OverlayStack, focus *FocusManager, profile *interaction.InteractionProfile, safeArea SafeAreaInsets) Bounds {
	// Interface-based dispatch: sub-package element types implement Layouter
	// and bypass the type switch entirely.
	if l, ok := el.(Layouter); ok {
		dispatcher := (*EventDispatcher)(nil)
		if ix != nil {
			dispatcher = ix.Dispatcher
		}
		ctx := &LayoutContext{Area: area, Canvas: canvas, Theme: th, Tokens: tokens, IX: ix, Overlays: overlays, Focus: focus, Dispatcher: dispatcher, Profile: profile, SafeArea: safeArea}
		return l.LayoutSelf(ctx)
	}
	switch node := el.(type) {
	case nil, WidgetElement:
		// WidgetElement should be resolved by the Reconciler before layout.
		return Bounds{X: area.X, Y: area.Y}

	case WidgetBoundsElement:
		childBounds := layoutElementCtx(node.Child, area, canvas, th, tokens, ix, overlays, focus, profile, safeArea)
		if ix != nil && ix.Dispatcher != nil {
			ix.Dispatcher.RegisterWidgetBounds(node.WidgetUID, draw.R(
				float32(childBounds.X), float32(childBounds.Y),
				float32(childBounds.W), float32(childBounds.H),
			))
		}
		return childBounds

	case KeyedElement:
		return layoutElementCtx(node.Child, area, canvas, th, tokens, ix, overlays, focus, profile, safeArea)

	case ThemedElement:
		// Switch theme and tokens for this subtree, lay out children as a column.
		subTh := node.Theme
		subTokens := subTh.Tokens()
		cursorY := area.Y
		maxW := 0
		for _, child := range node.Children {
			cb := layoutElementCtx(child, Bounds{X: area.X, Y: cursorY, W: area.W, H: area.H}, canvas, subTh, subTokens, ix, overlays, focus, profile, safeArea)
			if cb.W > maxW {
				maxW = cb.W
			}
			cursorY += cb.H
		}
		return Bounds{X: area.X, Y: area.Y, W: maxW, H: cursorY - area.Y}

	case SurfaceElement:
		return layoutSurface(node, area, canvas, tokens, ix)

	case Overlay:
		return layoutOverlay(node, area, canvas, th, tokens, ix, overlays, focus, profile)

	case CustomLayoutElement:
		return layoutCustom(node, area, canvas, th, tokens, ix, overlays, focus, profile)

	default:
		return Bounds{X: area.X, Y: area.Y}
	}
}

// indexedChild wraps a child element with its index in the parent's
// children slice.  LayoutChildren implementations pass these opaque
// values to Measure / Place; the callbacks unwrap them and use the
// index for placement tracking — avoiding == or treeEqual on
// potentially uncomparable element types (structs with func fields).
type indexedChild struct {
	index int
	child Element
}

func (indexedChild) isElement() {}

// layoutCustom implements the custom layout protocol (RFC-002 §4.3).
// It delegates measurement and placement to the user-provided Layout.
func layoutCustom(node CustomLayoutElement, area Bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *OverlayStack, fs *FocusManager, profile *interaction.InteractionProfile) Bounds {
	if node.Layout == nil || len(node.Children) == 0 {
		return Bounds{X: area.X, Y: area.Y}
	}

	nc := NullCanvas{Delegate: canvas}

	// Track placements: child index → offset.
	type placement struct {
		offset draw.Point
		placed bool
	}
	placements := make([]placement, len(node.Children))

	// Wrap children so Measure/Place can identify them by index.
	wrapped := make([]Element, len(node.Children))
	for i, c := range node.Children {
		wrapped[i] = indexedChild{index: i, child: c}
	}

	// Measure callback: unwrap and layout with NullCanvas to get size.
	measureFn := func(child Element, c Constraints) Size {
		actual := child
		if ic, ok := child.(indexedChild); ok {
			actual = ic.child
		}
		measureArea := Bounds{X: 0, Y: 0, W: int(c.MaxWidth), H: int(c.MaxHeight)}
		cb := layoutElement(actual, measureArea, nc, th, tokens, nil, nil, nil, profile)
		return Size{Width: float32(cb.W), Height: float32(cb.H)}
	}

	// Place callback: record offset by index — no value comparison needed.
	placeFn := func(child Element, offset draw.Point) {
		if ic, ok := child.(indexedChild); ok {
			placements[ic.index] = placement{offset: offset, placed: true}
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

	size := node.Layout.LayoutChildren(ctx, wrapped)

	// Paint pass: draw each placed child at its offset.
	maxW, maxH := 0, 0
	for i, child := range node.Children {
		p := placements[i]
		if !p.placed {
			continue
		}
		childArea := Bounds{
			X: area.X + int(p.offset.X),
			Y: area.Y + int(p.offset.Y),
			W: area.W,
			H: area.H,
		}
		cb := layoutElement(child, childArea, canvas, th, tokens, ix, overlays, fs, profile)
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

	return Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// HoverHighlight returns a lightened version of c for hover feedback.
func HoverHighlight(c draw.Color) draw.Color {
	return draw.Color{
		R: c.R + (1-c.R)*0.2,
		G: c.G + (1-c.G)*0.2,
		B: c.B + (1-c.B)*0.2,
		A: c.A,
	}
}

// LerpColor linearly interpolates between two colors.
func LerpColor(a, b draw.Color, t float32) draw.Color {
	return draw.Color{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A + (b.A-a.A)*t,
	}
}

// DisabledColor mutes a color by blending it 50% toward the base surface (RFC-008 §9.6).
func DisabledColor(c, base draw.Color) draw.Color {
	return LerpColor(c, base, 0.5)
}

// DrawFocusRing renders a subtle glow aura + focus stroke around a widget (RFC-008 §9.4).
func DrawFocusRing(canvas draw.Canvas, rect draw.Rect, radius float32, tokens theme.TokenSet) {
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

// ButtonVariantColors returns fill, border, and text colors for a button variant.
func ButtonVariantColors(variant ButtonVariant, tokens theme.TokenSet, hoverOpacity float32) (fill, border, textCol draw.Color) {
	switch variant {
	case ButtonOutlined:
		fill = draw.Color{A: 0} // transparent
		if hoverOpacity > 0 {
			fill = LerpColor(fill, tokens.Colors.Surface.Hovered, hoverOpacity)
		}
		border = tokens.Colors.Accent.Primary
		textCol = tokens.Colors.Accent.Primary
	case ButtonGhost:
		fill = draw.Color{A: 0} // transparent
		if hoverOpacity > 0 {
			fill = LerpColor(fill, tokens.Colors.Surface.Hovered, hoverOpacity)
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
			fill = LerpColor(fill, hoverFill, hoverOpacity)
		}
		border = draw.Color{A: 0}
		textCol = tokens.Colors.Accent.Primary
	default: // ButtonFilled
		fill = tokens.Colors.Accent.Primary
		if hoverOpacity > 0 {
			fill = LerpColor(fill, HoverHighlight(fill), hoverOpacity)
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

// layoutSurface renders an external surface slot (RFC §8).
// If the provider is non-nil, AcquireFrame/ReleaseFrame are called to
// obtain and release the GPU texture. Otherwise a placeholder is drawn.
func layoutSurface(node SurfaceElement, area Bounds, canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor) Bounds {
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

	return Bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: h}
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

func layoutOverlay(node Overlay, area Bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *OverlayStack, focus *FocusManager, profile *interaction.InteractionProfile) Bounds {
	if node.Content == nil || overlays == nil {
		return Bounds{X: area.X, Y: area.Y}
	}

	content := node.Content
	anchor := node.Anchor
	placement := node.Placement
	dismissable := node.Dismissable
	onDismiss := node.OnDismiss
	backdrop := node.Backdrop
	animation := node.Animation
	winW, winH := overlays.WindowW, overlays.WindowH

	// Resolve animation duration from theme tokens (RFC-008 §9.5).
	// OverlayAnimFadeScale uses Motion.Emphasized for dialog-level transitions;
	// simpler animations (Fade) use Motion.Standard.
	overlayDuration := tokens.Motion.Standard
	if animation == OverlayAnimFadeScale {
		overlayDuration = tokens.Motion.Emphasized
	}

	overlays.Push(OverlayEntry{
		Animation: animation,
		Duration:  overlayDuration,
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
			nc := NullCanvas{Delegate: canvas}
			cb := layoutElement(content, Bounds{X: 0, Y: 0, W: 400, H: 300}, nc, th, tokens, nil, nil, nil, profile)

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
			layoutElement(content, Bounds{
				X: int(pos.X) + pad, Y: int(pos.Y) + pad,
				W: max(int(contentSize.W)-pad*2, 0), H: max(int(contentSize.H)-pad*2, 0),
			}, canvas, th, tokens, ix, nil, focus, profile)
		},
	})

	// Overlays take no space in normal layout flow.
	return Bounds{X: area.X, Y: area.Y}
}
