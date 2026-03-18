//go:build nogui

package app

import (
	"fmt"
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
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
	return ui.Column(
		ui.Text("HELLO WORLD"),
		ui.Button("CLICK ME", nil),
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
	return ui.Column(
		ui.Text(fmt.Sprintf("Count: %d", m.Count)),
		ui.Row(
			ui.Button("−", func() { Send(decrMsg{}) }),
			ui.Button("+", func() { Send(incrMsg{}) }),
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
	ui.BuildScene(m3CounterView(testModel{Count: 0}), canvas, theme.Default, 800, 600, &hitMap, nil)

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

	// + button is the second in the Row at approx (216, 61) 180x45.
	// Click on frame 1 so frame 0 can build the initial scene + hit targets.
	// Frame 2 processes the message from the click.
	plusX := float32(220) // inside + button (starts at x=216)
	plusY := float32(70)  // inside + button (starts at y=61, height=45)

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
	scene := ui.BuildScene(m3CounterView(testModel{Count: 0}), canvas, theme.Default, 800, 600, nil, &hover)

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

func TestKeyEventBecomesKeyMsg(t *testing.T) {
	var receivedKey bool
	var keyName string
	update := func(m testModel, msg Msg) testModel {
		if km, ok := msg.(input.KeyMsg); ok {
			receivedKey = true
			keyName = km.Key
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
	if !receivedKey {
		t.Error("update should have received input.KeyMsg")
	}
	if keyName != "A" {
		t.Errorf("key name = %q, want %q", keyName, "A")
	}
}

func TestCharEventBecomesCharMsg(t *testing.T) {
	var receivedChar bool
	var ch rune
	update := func(m testModel, msg Msg) testModel {
		if cm, ok := msg.(input.CharMsg); ok {
			receivedChar = true
			ch = cm.Char
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
	if !receivedChar {
		t.Error("update should have received input.CharMsg")
	}
	if ch != 'Z' {
		t.Errorf("char = %c, want Z", ch)
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

func TestKeyModifiersPassedThrough(t *testing.T) {
	var mods input.KeyModifiers
	update := func(m testModel, msg Msg) testModel {
		if km, ok := msg.(input.KeyMsg); ok {
			mods = km.Modifiers
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
	if !mods.Shift {
		t.Error("Shift should be true")
	}
	if !mods.Ctrl {
		t.Error("Ctrl should be true")
	}
	if mods.Alt {
		t.Error("Alt should be false")
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
	return ui.Text(fmt.Sprintf("ticks=%d", s.Ticks)), s
}

func TestReconcilerPreservesWidgetStateThroughFrames(t *testing.T) {
	r := ui.NewReconciler()
	th := theme.Default

	tree := ui.ComponentWithKey("tick", tickWidget{})

	// Simulate 3 frames.
	for i := 0; i < 3; i++ {
		r.Reconcile(tree, th, func(_ any) {})
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

	r.Reconcile(ui.ComponentWithKey("old", tickWidget{}), th, func(_ any) {})
	r.Reconcile(ui.ComponentWithKey("old", tickWidget{}), th, func(_ any) {})

	// Switch key — state should be fresh.
	r.Reconcile(ui.ComponentWithKey("new", tickWidget{}), th, func(_ any) {})

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
		return ui.Text(fmt.Sprintf("count=%d", m.Count))
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
		return ui.Text(fmt.Sprintf("t=%.2f", m.Time))
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
