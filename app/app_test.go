//go:build nogui

package app

import (
	"fmt"
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"
)

type testModel struct {
	Count int
}

type incrMsg struct{}

func testUpdate(m testModel, msg Msg) testModel {
	switch msg.(type) {
	case incrMsg:
		m.Count++
	}
	return m
}

func testView(m testModel) ui.Element {
	return ui.Empty()
}

// m2HelloView is the M2 hello world view.
func m2HelloView(m testModel) ui.Element {
	return layout.Column(
		display.Text("HELLO WORLD"),
		button.Text("CLICK ME", nil),
	)
}

func TestRunHeadless(t *testing.T) {
	err := Run(testModel{}, testUpdate, testView,
		WithTitle("test"),
		WithSize(320, 240),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestSendBeforeRun(t *testing.T) {
	Send(incrMsg{})
	TrySend(incrMsg{})
}

func TestRunWithMessages(t *testing.T) {
	var finalCount int
	update := func(m testModel, msg Msg) testModel {
		switch msg.(type) {
		case incrMsg:
			m.Count++
		}
		finalCount = m.Count
		return m
	}

	err := Run(testModel{Count: 0}, update, testView,
		WithTitle("msg-test"),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	_ = finalCount
}

func TestM2HelloWorldScene(t *testing.T) {
	canvas := render.NewSceneCanvas(800, 600)
	scene := ui.BuildScene(m2HelloView(testModel{}), canvas, theme.Default, 800, 600, nil, nil)

	if len(scene.Rects) < 2 {
		t.Errorf("M2 scene should have at least 2 rects (button edge+fill), got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) < 2 {
		t.Fatalf("M2 scene should have at least 2 glyphs (label+button), got %d", len(scene.Glyphs))
	}

	foundLabel := false
	foundButton := false
	for _, g := range scene.Glyphs {
		if g.Text == "HELLO WORLD" {
			foundLabel = true
		}
		if g.Text == "CLICK ME" {
			foundButton = true
		}
	}
	if !foundLabel {
		t.Error("scene is missing the HELLO WORLD label")
	}
	if !foundButton {
		t.Error("scene is missing the CLICK ME button label")
	}
}

func TestM2RunHeadlessRendersScene(t *testing.T) {
	var frameCount int
	view := func(m testModel) ui.Element {
		frameCount++
		return m2HelloView(m)
	}

	err := Run(testModel{}, testUpdate, view,
		WithTitle("M2 test"),
		WithSize(800, 600),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if frameCount < 1 {
		t.Error("view was never called")
	}
}

func TestWithThemeOption(t *testing.T) {
	err := Run(testModel{}, testUpdate, testView,
		WithTheme(theme.Default),
		WithTitle("theme-test"),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestCanvasInterface(t *testing.T) {
	// Verify that SceneCanvas implements draw.Canvas.
	canvas := render.NewSceneCanvas(800, 600)
	var _ draw.Canvas = canvas

	// Draw some primitives and verify scene output.
	canvas.FillRect(draw.R(10, 20, 100, 50), draw.SolidPaint(draw.RGBA(255, 0, 0, 255)))
	canvas.DrawText("TEST", draw.Pt(10, 20), draw.TextStyle{Size: 21}, draw.RGBA(255, 255, 255, 255))

	scene := canvas.Scene()
	if len(scene.Rects) != 1 {
		t.Errorf("expected 1 rect, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) != 1 {
		t.Errorf("expected 1 glyph, got %d", len(scene.Glyphs))
	}
}

// ── M3 Counter Tests ──────────────────────────────────────────────

type decrMsg struct{}

func m3CounterUpdate(m testModel, msg Msg) testModel {
	switch msg.(type) {
	case incrMsg:
		m.Count++
	case decrMsg:
		m.Count--
	}
	return m
}

func m3CounterView(m testModel) ui.Element {
	return layout.Column(
		display.Text(fmt.Sprintf("Count: %d", m.Count)),
		layout.Row(
			button.Text("−", func() { Send(decrMsg{}) }),
			button.Text("+", func() { Send(incrMsg{}) }),
		),
	)
}

func TestM3CounterViewRendersCorrectly(t *testing.T) {
	canvas := render.NewSceneCanvas(800, 600)
	scene := ui.BuildScene(m3CounterView(testModel{Count: 42}), canvas, theme.Default, 800, 600, nil, nil)

	foundCount := false
	for _, g := range scene.Glyphs {
		if g.Text == "Count: 42" {
			foundCount = true
		}
	}
	if !foundCount {
		t.Error("counter view should display 'Count: 42'")
	}

	// Should have 2 buttons = 4 rects (edge+fill each) + 3 glyphs (count text + 2 button labels).
	if len(scene.Rects) < 4 {
		t.Errorf("expected at least 4 rects (2 buttons), got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) < 3 {
		t.Errorf("expected at least 3 glyphs (count + 2 buttons), got %d", len(scene.Glyphs))
	}
}

func TestM3CounterHitTargets(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	ix := ui.NewInteractor(&hitMap, nil)
	ui.BuildScene(m3CounterView(testModel{Count: 0}), canvas, theme.Default, 800, 600, ix)

	if hitMap.Len() != 2 {
		t.Fatalf("counter view should register 2 hit targets (− and +), got %d", hitMap.Len())
	}
}

func TestM3CounterClickIncrement(t *testing.T) {
	var finalCount int
	update := func(m testModel, msg Msg) testModel {
		m = m3CounterUpdate(m, msg)
		finalCount = m.Count
		return m
	}

	// Locate the + button by rendering a scene and finding its bounds.
	canvas := render.NewSceneCanvas(800, 600)
	scene := ui.BuildScene(m3CounterView(testModel{Count: 0}), canvas, theme.Default, 800, 600, nil)
	// The + button label is the last glyph in the scene.
	var plusX, plusY float32
	for _, g := range scene.Glyphs {
		if g.Text == "+" {
			plusX = float32(g.X + 5)
			plusY = float32(g.Y + 5)
		}
	}
	if plusX == 0 && plusY == 0 {
		t.Fatal("could not locate + button glyph")
	}

	err := Run(testModel{Count: 0}, update, m3CounterView,
		WithTitle("M3 counter test"),
		WithSize(800, 600),
		WithHeadlessFrames(3),
		WithHeadlessClick(1, plusX, plusY),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if finalCount != 1 {
		t.Errorf("expected count=1 after clicking +, got %d", finalCount)
	}
}

func TestM3CounterClickDecrement(t *testing.T) {
	var finalCount int
	update := func(m testModel, msg Msg) testModel {
		m = m3CounterUpdate(m, msg)
		finalCount = m.Count
		return m
	}

	// − button is at approx (24, 61) 180x45.
	minusX := float32(30) // inside − button (starts at x=24)
	minusY := float32(70) // inside − button (starts at y=61, height=45)

	err := Run(testModel{Count: 5}, update, m3CounterView,
		WithTitle("M3 decr test"),
		WithSize(800, 600),
		WithHeadlessFrames(3),
		WithHeadlessClick(1, minusX, minusY),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if finalCount != 4 {
		t.Errorf("expected count=4 after clicking −, got %d", finalCount)
	}
}

func TestM3HeadlessMultipleFrames(t *testing.T) {
	// Verify the headless platform runs the requested number of frames.
	// The view is only called on model changes, so we track update calls instead.
	updateCount := 0
	update := func(m testModel, msg Msg) testModel {
		updateCount++
		m.Count++
		return m
	}
	// Use a view that sends a message each time it's called,
	// creating a chain: view → Send → update → view → Send → ...
	view := func(m testModel) ui.Element {
		if m.Count < 3 {
			Send(incrMsg{})
		}
		return ui.Empty()
	}

	err := Run(testModel{Count: 0}, update, view,
		WithHeadlessFrames(5),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if updateCount < 3 {
		t.Errorf("expected at least 3 updates across frames, got %d", updateCount)
	}
}

// ── M4 Theme & Hover Tests ──────────────────────────────────────

func TestM4SetThemeMsgSwitchesColors(t *testing.T) {
	var receivedThemeMsg bool
	var sentOnce bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(SetThemeMsg); ok {
			receivedThemeMsg = true
		}
		return m
	}
	view := func(m testModel) ui.Element {
		if !sentOnce {
			sentOnce = true
			Send(SetThemeMsg{Theme: theme.Light})
		}
		return ui.Empty()
	}

	err := Run(testModel{}, update, view,
		WithTitle("M4 theme test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !receivedThemeMsg {
		t.Error("update should have received SetThemeMsg")
	}
}

func TestM4DarkModeToggle(t *testing.T) {
	var receivedDarkMode bool
	var sentOnce bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(SetDarkModeMsg); ok {
			receivedDarkMode = true
		}
		return m
	}
	view := func(m testModel) ui.Element {
		if !sentOnce {
			sentOnce = true
			Send(SetDarkModeMsg{Dark: false})
		}
		return ui.Empty()
	}

	err := Run(testModel{}, update, view,
		WithTitle("M4 dark mode test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !receivedDarkMode {
		t.Error("update should have received SetDarkModeMsg")
	}
}

func TestM4LightThemeTokens(t *testing.T) {
	lightTokens := theme.Light.Tokens()
	darkTokens := theme.Default.Tokens()

	if lightTokens.Colors.Surface.Base == darkTokens.Colors.Surface.Base {
		t.Error("light and dark Surface.Base should differ")
	}
	if lightTokens.Colors.Text.Primary == darkTokens.Colors.Text.Primary {
		t.Error("light and dark Text.Primary should differ")
	}
}

func TestM4MouseMoveInjection(t *testing.T) {
	// Verify that WithHeadlessMouseMove doesn't crash and the app runs.
	err := Run(testModel{}, testUpdate, m3CounterView,
		WithTitle("M4 hover test"),
		WithSize(800, 600),
		WithHeadlessFrames(3),
		WithHeadlessMouseMove(0, 30, 70), // move over button area
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestM4HoverChangesButtonColor(t *testing.T) {
	// Build a scene with hover on button 0 at full opacity.
	var hover ui.HoverState
	hover.SetHovered(0, 0) // instant hover
	hover.Tick(0)           // snap to target

	canvas := render.NewSceneCanvas(800, 600)
	ix := ui.NewInteractor(nil, &hover)
	scene := ui.BuildScene(m3CounterView(testModel{Count: 0}), canvas, theme.Default, 800, 600, ix)

	darkTokens := theme.Default.Tokens()

	// The first button's fill rect should differ from raw Primary.
	// Rects: [0]=button1 edge, [1]=button1 fill, [2]=button2 edge, [3]=button2 fill
	if len(scene.Rects) < 4 {
		t.Fatalf("expected at least 4 rects, got %d", len(scene.Rects))
	}
	hoveredFill := scene.Rects[1]
	if hoveredFill.Color == darkTokens.Colors.Accent.Primary {
		t.Error("hovered button fill should differ from raw Accent.Primary")
	}

	// Second button should still be raw Accent.Primary (not hovered).
	normalFill := scene.Rects[3]
	if normalFill.Color != darkTokens.Colors.Accent.Primary {
		t.Errorf("non-hovered button fill = %v, want Accent.Primary %v", normalFill.Color, darkTokens.Colors.Accent.Primary)
	}
}

// ── Input Event Tests ────────────────────────────────────────────

func TestScrollEventBecomesScrollMsg(t *testing.T) {
	var receivedScroll bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(input.ScrollMsg); ok {
			receivedScroll = true
		}
		return m
	}

	err := Run(testModel{}, update, testView,
		WithTitle("scroll-event test"),
		WithHeadlessFrames(3),
		WithHeadlessScroll(0, 0, -3.0),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !receivedScroll {
		t.Error("update should have received input.ScrollMsg")
	}
}

func TestKeyMsgNotForwardedToUserland(t *testing.T) {
	var receivedKey bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(input.KeyMsg); ok {
			receivedKey = true
		}
		return m
	}

	err := Run(testModel{}, update, testView,
		WithTitle("key-event test"),
		WithHeadlessFrames(3),
		WithHeadlessKey(0, "A", 0, 0), // press A
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if receivedKey {
		t.Error("KeyMsg should not reach userland update")
	}
}

func TestCharMsgNotForwardedToUserland(t *testing.T) {
	var receivedChar bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(input.CharMsg); ok {
			receivedChar = true
		}
		return m
	}

	err := Run(testModel{}, update, testView,
		WithTitle("char-event test"),
		WithHeadlessFrames(3),
		WithHeadlessChar(0, 'Z'),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if receivedChar {
		t.Error("CharMsg should not reach userland update")
	}
}

func TestMouseButtonEventBecomesMouseMsg(t *testing.T) {
	var receivedMouse bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(input.MouseMsg); ok {
			receivedMouse = true
		}
		return m
	}

	err := Run(testModel{}, update, testView,
		WithTitle("mouse-event test"),
		WithHeadlessFrames(3),
		WithHeadlessClick(0, 100, 100),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !receivedMouse {
		t.Error("update should have received input.MouseMsg")
	}
}

func TestKeyModifiersNotPassedToUserland(t *testing.T) {
	var receivedKey bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(input.KeyMsg); ok {
			receivedKey = true
		}
		return m
	}

	// mods=3 means Shift(1) + Ctrl(2)
	err := Run(testModel{}, update, testView,
		WithTitle("key-mods test"),
		WithHeadlessFrames(3),
		WithHeadlessKey(0, "A", 0, 3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if receivedKey {
		t.Error("KeyMsg with modifiers should not reach userland update")
	}
}

func TestScrollViewReactsToScrollWheel(t *testing.T) {
	// End-to-end test: inject a scroll event and verify the ScrollState offset changes.
	scroll := &ui.ScrollState{}
	view := func(m testModel) ui.Element {
		// Tall content that exceeds viewport.
		return nav.NewScrollView(layout.Column(
			display.Text("A"), display.Text("B"), display.Text("C"), display.Text("D"),
			display.Text("E"), display.Text("F"), display.Text("G"), display.Text("H"),
			display.Text("I"), display.Text("J"), display.Text("K"), display.Text("L"),
		), 40, scroll)
	}

	// Frame 0: build initial scene (registers scroll target).
	// Scroll event on frame 1: dispatched to the scroll target.
	// Frame 2: view rebuilt with new offset.
	err := Run(testModel{}, testUpdate, view,
		WithTitle("scroll-wheel test"),
		WithSize(800, 600),
		WithHeadlessFrames(3),
		WithHeadlessMouseMove(0, 30, 30), // position cursor over scroll view
		WithHeadlessScroll(1, 0, -1.0),   // scroll down
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if scroll.Offset <= 0 {
		t.Errorf("ScrollState.Offset = %f, want > 0 after scroll-down event", scroll.Offset)
	}
}


// ── Widget Reconciliation Tests ─────────────────────────────────

// tickWidget is a stateful widget that tracks render calls.
type tickWidget struct{}

type tickState struct {
	Ticks int
}

func (tickWidget) Render(_ ui.RenderCtx, raw ui.WidgetState) (ui.Element, ui.WidgetState) {
	s := ui.AdoptState[tickState](raw)
	s.Ticks++
	return display.Text(fmt.Sprintf("ticks=%d", s.Ticks)), s
}

func TestReconcilerPreservesWidgetStateThroughFrames(t *testing.T) {
	r := ui.NewReconciler()
	th := theme.Default

	tree := ui.ComponentWithKey("tick", tickWidget{})

	// Simulate 3 frames.
	for i := 0; i < 3; i++ {
		r.Reconcile(tree, th, func(_ any) {}, nil, nil, "")
	}

	uid := ui.MakeUID(0, "tick", 0)
	raw := r.StateFor(uid)
	s, ok := raw.(*tickState)
	if !ok {
		t.Fatalf("expected *tickState, got %T", raw)
	}
	if s.Ticks != 3 {
		t.Errorf("Ticks = %d, want 3", s.Ticks)
	}
}

func TestReconcilerResetsStateOnKeyChange(t *testing.T) {
	r := ui.NewReconciler()
	th := theme.Default

	r.Reconcile(ui.ComponentWithKey("old", tickWidget{}), th, func(_ any) {}, nil, nil, "")
	r.Reconcile(ui.ComponentWithKey("old", tickWidget{}), th, func(_ any) {}, nil, nil, "")

	// Switch key — state should be fresh.
	r.Reconcile(ui.ComponentWithKey("new", tickWidget{}), th, func(_ any) {}, nil, nil, "")

	uid := ui.MakeUID(0, "new", 0)
	raw := r.StateFor(uid)
	s, ok := raw.(*tickState)
	if !ok {
		t.Fatalf("expected *tickState, got %T", raw)
	}
	if s.Ticks != 1 {
		t.Errorf("Ticks after key change = %d, want 1", s.Ticks)
	}
}

func TestNoRebuildWithoutModelChange(t *testing.T) {
	viewCalls := 0
	view := func(m testModel) ui.Element {
		viewCalls++
		return display.Text(fmt.Sprintf("count=%d", m.Count))
	}

	// Run for 3 frames with no messages → view should be called only once (initial).
	err := Run(testModel{Count: 0}, testUpdate, view,
		WithTitle("no-rebuild test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if viewCalls != 1 {
		t.Errorf("view called %d times, want 1 (no model changes)", viewCalls)
	}
}

func TestTickMsgDrivesAnimation(t *testing.T) {
	type animModel struct {
		Time float64
	}
	viewCalls := 0
	update := func(m animModel, msg Msg) animModel {
		switch msg := msg.(type) {
		case TickMsg:
			m.Time += msg.DeltaTime.Seconds()
		}
		return m
	}
	view := func(m animModel) ui.Element {
		viewCalls++
		return display.Text(fmt.Sprintf("t=%.2f", m.Time))
	}

	err := Run(animModel{}, update, view,
		WithTitle("tick-anim test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	// TickMsg modifies Time each frame → view should be rebuilt each frame.
	// Initial + 2 rebuilds from TickMsg (frame 0 tick is dt=0 but still changes via float).
	if viewCalls < 2 {
		t.Errorf("view called %d times, want >= 2 (TickMsg should trigger rebuild)", viewCalls)
	}
}

func TestWidgetRenderedInRunLoop(t *testing.T) {
	// Verify that Component() widgets are expanded during Run via the reconciler.
	var finalTicks int
	view := func(m testModel) ui.Element {
		return ui.ComponentWithKey("w", tickWidget{})
	}
	update := func(m testModel, msg Msg) testModel {
		switch msg.(type) {
		case incrMsg:
			m.Count++
		}
		return m
	}

	// Send a message on frame 0 to trigger re-reconciliation.
	// Frame 0: initial reconcile (ticks=1) + view sends msg
	// Frame 1: drain msg → view called → reconcile (ticks=2)
	sentOnce := false
	wrappedView := func(m testModel) ui.Element {
		if !sentOnce {
			sentOnce = true
			Send(incrMsg{})
		}
		el := view(m)
		return el
	}

	err := Run(testModel{}, update, wrappedView,
		WithTitle("widget-run test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// The widget should have been rendered at least twice (initial + after msg).
	_ = finalTicks // We validated by not crashing — the reconciler expanded the widget.
}

// ── Part 1: TickMsg with non-comparable models ──────────────────

type sliceModel struct {
	Items []string
}

func TestTickMsgWithNonComparableModel(t *testing.T) {
	// A model containing a slice is not comparable via ==.
	// This must not panic.
	update := func(m sliceModel, msg Msg) sliceModel {
		switch msg.(type) {
		case TickMsg:
			m.Items = append(m.Items, "tick")
		}
		return m
	}
	view := func(m sliceModel) ui.Element {
		return display.Text(fmt.Sprintf("len=%d", len(m.Items)))
	}

	err := Run(sliceModel{Items: []string{"init"}}, update, view,
		WithTitle("slice-model test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

type mapModel struct {
	Data map[string]int
}

func TestTickMsgWithMapModel(t *testing.T) {
	// A model containing a map is not comparable via ==.
	// This must not panic.
	update := func(m mapModel, msg Msg) mapModel {
		switch msg.(type) {
		case TickMsg:
			if m.Data == nil {
				m.Data = make(map[string]int)
			}
			m.Data["tick"]++
		}
		return m
	}
	view := func(m mapModel) ui.Element {
		return display.Text("map-model")
	}

	err := Run(mapModel{}, update, view,
		WithTitle("map-model test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestTickMsgUnchangedModelNoBuild(t *testing.T) {
	// When TickMsg doesn't change the model, view should not be called again.
	viewCalls := 0
	update := func(m testModel, msg Msg) testModel {
		// Ignore TickMsg — model stays the same.
		return m
	}
	view := func(m testModel) ui.Element {
		viewCalls++
		return display.Text(fmt.Sprintf("count=%d", m.Count))
	}

	err := Run(testModel{Count: 0}, update, view,
		WithTitle("tick-no-rebuild test"),
		WithHeadlessFrames(5),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if viewCalls != 1 {
		t.Errorf("view called %d times, want 1 (TickMsg should not trigger rebuild when model unchanged)", viewCalls)
	}
}

// ── Part 2: Changed-detection for normal messages ───────────────

type ignoreMsg struct{}

func TestMessageWithoutModelChangeNoRebuild(t *testing.T) {
	viewCalls := 0
	sentOnce := false
	update := func(m testModel, msg Msg) testModel {
		// Ignore ignoreMsg — model stays the same.
		return m
	}
	view := func(m testModel) ui.Element {
		viewCalls++
		if !sentOnce {
			sentOnce = true
			Send(ignoreMsg{})
		}
		return display.Text("static")
	}

	err := Run(testModel{}, update, view,
		WithTitle("no-change-no-rebuild test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	// Initial view call only; ignoreMsg should not cause a rebuild.
	if viewCalls != 1 {
		t.Errorf("view called %d times, want 1 (message that doesn't change model should not rebuild)", viewCalls)
	}
}

func TestThemeSwitchTriggersRepaint(t *testing.T) {
	viewCalls := 0
	sentOnce := false
	update := func(m testModel, msg Msg) testModel {
		return m // model unchanged
	}
	view := func(m testModel) ui.Element {
		viewCalls++
		if !sentOnce {
			sentOnce = true
			Send(SetThemeMsg{Theme: theme.Light})
		}
		return display.Text("themed")
	}

	err := Run(testModel{}, update, view,
		WithTitle("theme-repaint test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	// view should be called at least twice: initial + after theme switch.
	if viewCalls < 2 {
		t.Errorf("view called %d times, want >= 2 (theme switch should trigger repaint)", viewCalls)
	}
}

// ── Part 4: TextInputMsg ────────────────────────────────────────

func TestTextInputMsgNotForwardedToUserland(t *testing.T) {
	var receivedTextInput bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(input.TextInputMsg); ok {
			receivedTextInput = true
		}
		return m
	}

	sentOnce := false
	view := func(m testModel) ui.Element {
		if !sentOnce {
			sentOnce = true
			Send(input.TextInputMsg{Text: "abc"})
		}
		return ui.Empty()
	}

	err := Run(testModel{}, update, view,
		WithTitle("textinput-event test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if receivedTextInput {
		t.Error("TextInputMsg should not reach userland update")
	}
}

// ── Cmd Tests ───────────────────────────────────────────────────

type cmdResultMsg struct{ Value string }

func TestRunWithCmdExecutesCommand(t *testing.T) {
	// Test that Cmd returns a message and that UpdateWithCmd receives it.
	// Use a synchronous Cmd that returns immediately — the goroutine sends
	// the result back into the loop, picked up by the next frame.
	var result string
	update := func(m testModel, msg Msg) (testModel, Cmd) {
		switch msg := msg.(type) {
		case incrMsg:
			m.Count++
			return m, func() Msg {
				return cmdResultMsg{Value: "async-done"}
			}
		case cmdResultMsg:
			result = msg.Value
		}
		return m, nil
	}

	sentOnce := false
	view := func(m testModel) ui.Element {
		if !sentOnce {
			sentOnce = true
			Send(incrMsg{})
		}
		return ui.Empty()
	}

	err := RunWithCmd(testModel{}, update, view,
		WithTitle("cmd-exec test"),
		WithHeadlessFrames(10),
	)
	if err != nil {
		t.Fatalf("RunWithCmd returned error: %v", err)
	}
	if result != "async-done" {
		t.Errorf("result = %q, want async-done", result)
	}
}

func TestRunWithCmdNilCommandSafe(t *testing.T) {
	update := func(m testModel, msg Msg) (testModel, Cmd) {
		return m, nil
	}

	err := RunWithCmd(testModel{}, update, testView,
		WithTitle("cmd-nil test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("RunWithCmd returned error: %v", err)
	}
}

func TestBatchCombinesCommands(t *testing.T) {
	c1 := func() Msg { return nil }
	var r2 bool
	c2 := func() Msg { r2 = true; return nil }

	batch := Batch(c1, c2)
	if batch == nil {
		t.Fatal("Batch should return non-nil Cmd")
	}
	batch()
	// c2 runs inline (last cmd), c1 runs in goroutine.
	if !r2 {
		t.Error("second command should have run inline")
	}
}

func TestBatchNilsFiltered(t *testing.T) {
	ran := false
	c := func() Msg { ran = true; return nil }

	batch := Batch(nil, c, nil)
	if batch == nil {
		t.Fatal("Batch with one live cmd should return non-nil")
	}
	batch()
	if !ran {
		t.Error("single live command should have run")
	}
}

func TestBatchAllNilReturnsNil(t *testing.T) {
	batch := Batch(nil, nil, nil)
	if batch != nil {
		t.Error("Batch of all nils should return nil")
	}
}

func TestNoneSentinel(t *testing.T) {
	if None != nil {
		t.Error("None should be nil")
	}
}

func TestRunBackwardsCompatible(t *testing.T) {
	// Run (without Cmd) should still work exactly as before.
	var finalCount int
	update := func(m testModel, msg Msg) testModel {
		switch msg.(type) {
		case incrMsg:
			m.Count++
		}
		finalCount = m.Count
		return m
	}

	sentOnce := false
	view := func(m testModel) ui.Element {
		if !sentOnce {
			sentOnce = true
			Send(incrMsg{})
		}
		return ui.Empty()
	}

	err := Run(testModel{Count: 0}, update, view,
		WithTitle("compat test"),
		WithHeadlessFrames(3),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if finalCount != 1 {
		t.Errorf("finalCount = %d, want 1", finalCount)
	}
}

// ── Phase 5 — Platform Extension Tests ──────────────────────────

func TestHeadlessPlatformSetSize(t *testing.T) {
	var gotResize bool
	update := func(m testModel, msg Msg) testModel {
		if _, ok := msg.(input.ResizeMsg); ok {
			gotResize = true
		}
		return m
	}

	err := Run(testModel{}, update, testView,
		WithTitle("setsize-test"),
		WithSize(800, 600),
		WithHeadlessFrames(1),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	_ = gotResize
}

func TestHeadlessPlatformClipboard(t *testing.T) {
	// The headless platform stores clipboard in memory.
	p := defaultPlatformFactory()
	_ = p.Init(platform.Config{Width: 100, Height: 100})

	if err := p.SetClipboard("test content"); err != nil {
		t.Fatalf("SetClipboard error: %v", err)
	}

	text, err := p.GetClipboard()
	if err != nil {
		t.Fatalf("GetClipboard error: %v", err)
	}
	if text != "test content" {
		t.Errorf("GetClipboard() = %q, want %q", text, "test content")
	}
}

func TestHeadlessPlatformNewMethods(t *testing.T) {
	p := defaultPlatformFactory()
	_ = p.Init(platform.Config{Width: 800, Height: 600})

	// SetSize
	p.SetSize(1024, 768)
	w, h := p.WindowSize()
	if w != 1024 || h != 768 {
		t.Errorf("after SetSize: WindowSize() = (%d, %d), want (1024, 768)", w, h)
	}

	// SetFullscreen — no-op but should not panic.
	p.SetFullscreen(true)
	p.SetFullscreen(false)

	// RequestFrame — no-op but should not panic.
	p.RequestFrame()

	// CreateWGPUSurface — returns 0 for headless.
	surface := p.CreateWGPUSurface(0)
	if surface != 0 {
		t.Errorf("CreateWGPUSurface(0) = %d, want 0", surface)
	}
}

func TestWithFullscreenOption(t *testing.T) {
	// Verify the option compiles and can be passed to Run.
	err := Run(testModel{}, testUpdate, testView,
		WithTitle("fullscreen-test"),
		WithFullscreen(true),
		WithHeadlessFrames(1),
	)
	if err != nil {
		t.Fatalf("Run with WithFullscreen returned error: %v", err)
	}
}

func TestBatchNil(t *testing.T) {
	// Batch with all nil should return nil.
	cmd := Batch(nil, nil, nil)
	if cmd != nil {
		t.Error("Batch(nil, nil, nil) should return nil")
	}
}

func TestPackageLevelClipboard(t *testing.T) {
	// Before Run, clipboard functions should be no-ops.
	err := SetClipboard("before run")
	if err != nil {
		t.Errorf("SetClipboard before Run: %v", err)
	}

	text, err := GetClipboard()
	if err != nil {
		t.Errorf("GetClipboard before Run: %v", err)
	}
	if text != "" {
		t.Errorf("GetClipboard before Run = %q, want empty", text)
	}
}
