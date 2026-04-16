// Package apptest provides a headless test runner for interactive
// UI testing. Use TestApp to create a headless app, query elements
// via the accessibility tree, and simulate user interactions.
package apptest

import (
	"strings"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/loop"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// TestOption configures a TestApp.
type TestOption func(*testConfig)

type testConfig struct {
	width  int
	height int
	theme  theme.Theme
}

// WithSize sets the viewport size for the test app.
func WithSize(w, h int) TestOption {
	return func(c *testConfig) {
		c.width = w
		c.height = h
	}
}

// WithTheme sets the theme for the test app.
func WithTheme(th theme.Theme) TestOption {
	return func(c *testConfig) {
		c.theme = th
	}
}

// TestApp is a headless app runner for testing interactive UI behavior.
// It mirrors the frame loop from app/run.go without a platform or GPU,
// providing semantic querying and programmatic interaction.
type TestApp[M any] struct {
	model      M
	update     func(M, any) (M, app.Cmd)
	view       func(M) ui.Element

	reconciler *ui.Reconciler
	fm         *ui.FocusManager
	dispatcher *ui.EventDispatcher
	hitMap     hit.Map
	hoverState ui.HoverState

	currentTree ui.Element
	accessTree  a11y.AccessTree

	appLoop    *loop.Loop
	clipboard  memClipboard
	theme      theme.Theme

	width, height int
	frameCount    uint64
	dirty         bool
}

// memClipboard is an in-memory clipboard for testing.
type memClipboard struct {
	text string
}

func (c *memClipboard) GetClipboard() (string, error) { return c.text, nil }
func (c *memClipboard) SetClipboard(s string) error   { c.text = s; return nil }

// New creates a TestApp with a simple update function (no commands).
func New[M any](model M, update func(M, any) M, view func(M) ui.Element, opts ...TestOption) *TestApp[M] {
	return NewWithCmd(model, func(m M, msg any) (M, app.Cmd) {
		return update(m, msg), nil
	}, view, opts...)
}

// NewWithCmd creates a TestApp with an update function that returns commands.
func NewWithCmd[M any](model M, update func(M, any) (M, app.Cmd), view func(M) ui.Element, opts ...TestOption) *TestApp[M] {
	cfg := testConfig{
		width:  800,
		height: 600,
		theme:  theme.Default,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	appLoop := loop.New()
	app.SetGlobalLoopForTest(appLoop)

	fm := ui.NewFocusManager()
	app.SetGlobalFocusForTest(fm)

	dispatcher := ui.NewEventDispatcher(fm)
	reconciler := ui.NewReconciler()

	ta := &TestApp[M]{
		model:      model,
		update:     update,
		view:       view,
		reconciler: reconciler,
		fm:         fm,
		dispatcher: dispatcher,
		appLoop:    appLoop,
		theme:      cfg.theme,
		width:      cfg.width,
		height:     cfg.height,
	}

	// Initial reconcile.
	initialView := view(model)
	ta.currentTree, _ = reconciler.Reconcile(initialView, cfg.theme, app.Send, nil, nil, "", nil)

	// Double scene build: first populates hit map and registers widget bounds
	// in the dispatcher's current buffer; SwapBounds moves them to the previous
	// buffer. The second build then has accurate bounds for the access tree.
	ta.buildScene()
	ta.buildScene()

	return ta
}

// Step advances the app by one frame. It mirrors the OnFrame callback
// from app/run.go: drain messages → update → tick animations →
// reconcile → build scene → build access tree.
func (ta *TestApp[M]) Step() {
	dispatcher := ta.dispatcher
	fm := ta.fm
	modelDirty := false
	dispatcher.ResetEvents()

	// 1. Drain messages.
	ta.appLoop.DrainMessages(func(msg any) bool {
		switch m := msg.(type) {
		case ui.RequestFocusMsg:
			oldUID := fm.FocusedUID()
			fm.SetFocusedUID(m.Target)
			dispatcher.QueueFocusChange(oldUID, m.Target, ui.FocusSourceProgram)
			modelDirty = true
			return true

		case ui.ReleaseFocusMsg:
			oldUID := fm.FocusedUID()
			fm.Blur()
			dispatcher.QueueFocusChange(oldUID, 0, ui.FocusSourceProgram)
			modelDirty = true
			return true

		case input.KeyMsg:
			dispatcher.Collect(m)
			if ui.HandleTabNavigation(fm, m, dispatcher) {
				modelDirty = true
				return true
			}
			if ui.HandleTextFieldKeyMsg(fm, m, dispatcher, &ta.clipboard) {
				modelDirty = true
			}
			// Fall through to user update.

		case input.CharMsg:
			dispatcher.Collect(m)
			if ui.HandleCharMsg(fm, m) {
				modelDirty = true
			}

		case input.TextInputMsg:
			dispatcher.Collect(m)
			if ui.HandleTextInputMsg(fm, m) {
				modelDirty = true
			}

		case input.IMEComposeMsg:
			dispatcher.Collect(m)
			if ui.HandleIMEComposeMsg(fm, m) {
				modelDirty = true
			}

		case input.IMECommitMsg:
			dispatcher.Collect(m)
			if ui.HandleIMECommitMsg(fm, m) {
				modelDirty = true
			}

		case input.MouseMsg:
			dispatcher.Collect(m)
		case input.ScrollMsg:
			dispatcher.Collect(m)
		}

		newModel, cmd := ta.update(ta.model, msg)
		if modelChanged(any(newModel), any(ta.model)) {
			modelDirty = true
		}
		ta.model = newModel
		ta.dispatchCmd(cmd)
		return true
	})

	// 2. Tick animations (fixed 16ms for determinism).
	dt := 16 * time.Millisecond
	animDirty := ta.reconciler.TickAnimators(dt)
	stateDirty := ta.reconciler.CheckDirtyTrackers()

	// Deliver TickMsg.
	tickModel, tickCmd := ta.update(ta.model, app.TickMsg{DeltaTime: dt})
	tickDirty := modelChanged(any(tickModel), any(ta.model))
	ta.model = tickModel
	ta.dispatchCmd(tickCmd)

	focusDirty := fm.ConsumeDirty()
	modelDirty = modelDirty || tickDirty || animDirty || stateDirty || focusDirty

	ta.dirty = modelDirty

	// 3. Reconcile if dirty.
	if modelDirty {
		fm.ResetOrder()
		dispatcher.Dispatch()

		newTree := ta.view(ta.model)
		ta.currentTree, _ = ta.reconciler.Reconcile(newTree, ta.theme, app.Send, dispatcher, fm, "", nil)
		fm.SortOrder()
	}

	// 4. Build scene (always, to keep hit map and access tree current).
	ta.buildScene()
	ta.frameCount++
}

// buildScene performs layout + paint + hit-target registration + access tree build.
func (ta *TestApp[M]) buildScene() {
	ta.fm.ResetElementIDs()
	canvas := render.NewSceneCanvas(ta.width, ta.height)
	ta.hitMap.Reset()
	ix := ui.NewInteractor(&ta.hitMap, &ta.hoverState)
	ix.Dispatcher = ta.dispatcher
	needsFrame := false
	ix.NeedsFrame = &needsFrame

	ui.BuildScene(ta.currentTree, canvas, ta.theme, ta.width, ta.height, ix, ta.fm)
	ta.dispatcher.SwapBounds()
	ta.hoverState.Trim(ta.hitMap.Len())

	// Build access tree.
	ta.accessTree = ui.BuildAccessTree(ta.currentTree, ta.reconciler, a11y.Rect{
		Width: float64(ta.width), Height: float64(ta.height),
	}, ta.dispatcher)
}

// StepN advances the app by n frames.
func (ta *TestApp[M]) StepN(n int) {
	for i := 0; i < n; i++ {
		ta.Step()
	}
}

// StepUntilStable advances frames until no model changes occur,
// or maxFrames is reached (default 100).
func (ta *TestApp[M]) StepUntilStable(maxFrames ...int) {
	limit := 100
	if len(maxFrames) > 0 && maxFrames[0] > 0 {
		limit = maxFrames[0]
	}
	for i := 0; i < limit; i++ {
		ta.Step()
		if !ta.dirty {
			return
		}
	}
}

// Model returns the current model.
func (ta *TestApp[M]) Model() M {
	return ta.model
}

// Send injects a message into the app loop.
func (ta *TestApp[M]) Send(msg any) {
	ta.appLoop.Send(msg)
}

// Resize changes the viewport size. Call Step() after to apply.
func (ta *TestApp[M]) Resize(w, h int) {
	ta.width = w
	ta.height = h
}

// Close cleans up the test app. Always defer this.
func (ta *TestApp[M]) Close() {
	app.SetGlobalLoopForTest(nil)
	app.SetGlobalFocusForTest(ui.NewFocusManager())
}

// AccessTree returns the current accessibility tree.
func (ta *TestApp[M]) AccessTree() a11y.AccessTree {
	return ta.accessTree
}

// HitMap returns the current hit map for direct access.
func (ta *TestApp[M]) HitMap() *hit.Map {
	return &ta.hitMap
}

// FocusManager returns the focus manager.
func (ta *TestApp[M]) FocusManager() *ui.FocusManager {
	return ta.fm
}

// Dispatcher returns the event dispatcher.
func (ta *TestApp[M]) Dispatcher() *ui.EventDispatcher {
	return ta.dispatcher
}

// Clipboard returns the test clipboard contents.
func (ta *TestApp[M]) Clipboard() string {
	return ta.clipboard.text
}

// SetClipboard sets the test clipboard contents.
func (ta *TestApp[M]) SetClipboard(text string) {
	ta.clipboard.text = text
}

// ── Querying ────────────────────────────────────────────────────

// Query returns the first element matching the selector, or nil.
func (ta *TestApp[M]) Query(sel Selector) *TestElement {
	for i := range ta.accessTree.Nodes {
		n := &ta.accessTree.Nodes[i]
		if sel.matches(n) {
			return ta.elementFromNode(n)
		}
	}
	return nil
}

// QueryAll returns all elements matching the selector.
func (ta *TestApp[M]) QueryAll(sel Selector) []*TestElement {
	var result []*TestElement
	for i := range ta.accessTree.Nodes {
		n := &ta.accessTree.Nodes[i]
		if sel.matches(n) {
			result = append(result, ta.elementFromNode(n))
		}
	}
	return result
}

// elementFromNode creates a TestElement handle from an access tree node.
func (ta *TestApp[M]) elementFromNode(n *a11y.AccessTreeNode) *TestElement {
	return &TestElement{
		node:         n,
		sendFn:       func(msg any) { ta.appLoop.Send(msg) },
		stepFn:       func() { ta.Step() },
		hitMapFn:     func() *hit.Map { return &ta.hitMap },
		fmFn:         func() *ui.FocusManager { return ta.fm },
		dispatcherFn: func() *ui.EventDispatcher { return ta.dispatcher },
		accessTreeFn: func() *a11y.AccessTree { return &ta.accessTree },
		queryFn:      func(s Selector) *TestElement { return ta.Query(s) },
		queryAllFn:   func(s Selector) []*TestElement { return ta.QueryAll(s) },
	}
}

// dispatchCmd runs a Cmd asynchronously, sending its result back into the loop.
func (ta *TestApp[M]) dispatchCmd(cmd app.Cmd) {
	if cmd != nil {
		go func() {
			if result := cmd(); result != nil {
				ta.appLoop.Send(result)
			}
		}()
	}
}

// ── Selector ────────────────────────────────────────────────────

// Selector specifies criteria for matching access tree nodes.
// All non-nil fields must match (AND logic).
type Selector struct {
	role  *a11y.AccessRole
	label *string
	text  *string // substring match in label + value + description
}

// ByRole matches nodes with the given accessibility role.
func ByRole(role a11y.AccessRole) Selector {
	return Selector{role: &role}
}

// ByLabel matches nodes whose label equals the given string exactly.
func ByLabel(label string) Selector {
	return Selector{label: &label}
}

// ByText matches nodes where the given string is a substring of the
// node's label, value, or description.
func ByText(text string) Selector {
	return Selector{text: &text}
}

// ByLabelAndRole matches nodes with both the given label and role.
func ByLabelAndRole(label string, role a11y.AccessRole) Selector {
	return Selector{label: &label, role: &role}
}

// matches reports whether a node satisfies all selector criteria.
func (s Selector) matches(n *a11y.AccessTreeNode) bool {
	if s.role != nil && n.Node.Role != *s.role {
		return false
	}
	if s.label != nil && n.Node.Label != *s.label {
		return false
	}
	if s.text != nil {
		t := *s.text
		found := strings.Contains(n.Node.Label, t) ||
			strings.Contains(n.Node.Value, t) ||
			strings.Contains(n.Node.Description, t)
		if !found {
			return false
		}
	}
	return true
}

// modelChanged reports whether two model values differ.
func modelChanged(a, b any) (changed bool) {
	changed = true
	defer func() { recover() }()
	return a != b
}
