package app

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/loop"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// Run starts the application. It blocks until the window is closed (RFC §3.1).
//
// The model, update, and view form the Elm architecture triad:
//   - model:  initial application state
//   - update: processes a Msg and returns a new model (pure function)
//   - view:   renders the model as an Element tree (pure function)
//
// Both update and view run exclusively on the calling goroutine.
func Run[M any](model M, update UpdateFunc[M], view ViewFunc[M], opts ...Option) error {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	var loopOpts []loop.Option
	if cfg.maxFrameDelta != loop.DefaultMaxFrameDelta {
		loopOpts = append(loopOpts, loop.WithMaxFrameDelta(cfg.maxFrameDelta))
	}
	appLoop := loop.New(loopOpts...)
	globalLoop = appLoop
	defer func() { globalLoop = nil }()

	plat := cfg.platformFactory()
	if err := plat.Init(platform.Config{
		Title:  cfg.title,
		Width:  cfg.width,
		Height: cfg.height,
	}); err != nil {
		return fmt.Errorf("platform init: %w", err)
	}
	defer plat.Destroy()

	renderer := cfg.rendererFactory()
	fbW, fbH := plat.FramebufferSize()

	var nativeHandle uintptr
	if nh, ok := plat.(interface{ NativeHandle() uintptr }); ok {
		nativeHandle = nh.NativeHandle()
	}

	if err := renderer.Init(gpu.Config{
		Width:        fbW,
		Height:       fbH,
		NativeHandle: nativeHandle,
	}); err != nil {
		return fmt.Errorf("gpu init: %w", err)
	}
	defer renderer.Destroy()

	// Initialize the font rendering pipeline (RFC-003 §3).
	atlas := text.NewGlyphAtlas(512, 512)
	shaper := text.NewSfntShaper(fonts.Fallback)
	shaper.RegisterFamily(fonts.PhosphorFamily)

	// If the renderer supports atlas-based text, wire it up.
	type atlasSetter interface{ SetAtlas(*text.GlyphAtlas) }
	if as, ok := renderer.(atlasSetter); ok {
		as.SetAtlas(atlas)
	}

	activeTheme := cfg.theme
	bgColor := activeTheme.Tokens().Colors.Surface.Base

	// Tell the renderer about the background color if it supports it.
	updateBgColor := func() {
		bgColor = activeTheme.Tokens().Colors.Surface.Base
		if bgs, ok := renderer.(interface{ SetBackgroundColor(draw.Color) }); ok {
			bgs.SetBackgroundColor(bgColor)
		}
	}
	updateBgColor()

	// Focus management (RFC-002 §2.3).
	fm := globalFocus
	dispatcher := ui.NewEventDispatcher(fm)

	reconciler := ui.NewReconciler()
	currentModel := model
	currentTree, _ := reconciler.Reconcile(view(currentModel), activeTheme, Send, nil, nil)

	lastFrame := time.Now()
	var hitMap hit.Map
	var hoverState ui.HoverState
	var mouseX, mouseY float32
	var dragTarget *hit.Target // active drag target (non-nil while dragging)

	return plat.Run(platform.Callbacks{
		OnFrame: func() {
			// 1. Drain messages — intercept theme switches and focus requests
			// before user update (RFC §5.5). Collect input events for dispatch.
			modelDirty := false
			dispatcher.ResetEvents()

			appLoop.DrainMessages(func(msg any) bool {
				// Handle framework-internal messages.
				switch m := msg.(type) {
				case SetThemeMsg:
					activeTheme = m.Theme
					updateBgColor()
					modelDirty = true
				case SetDarkModeMsg:
					if m.Dark {
						activeTheme = theme.Slate
					} else {
						activeTheme = theme.SlateLight
					}
					updateBgColor()
					modelDirty = true

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
					// Collect for widget-level dispatch.
					dispatcher.Collect(m)

					// Handle Tab/Shift+Tab for focus navigation.
					if m.Action == input.KeyPress || m.Action == input.KeyRepeat {
						if m.Key == input.KeyTab {
							oldUID := fm.FocusedUID()
							var newUID ui.UID
							if m.Modifiers.Has(input.ModShift) {
								newUID = fm.FocusPrev()
							} else {
								newUID = fm.FocusNext()
							}
							if newUID != oldUID {
								dispatcher.QueueFocusChange(oldUID, newUID, ui.FocusSourceTab)
								modelDirty = true
							}
							return true
						}

						// Framework-internal keyboard handling for TextFields.
						if is := fm.Input; is != nil {
							switch m.Key {
							case input.KeyBackspace:
								if len(is.Value) > 0 {
									runes := []rune(is.Value)
									v := string(runes[:len(runes)-1])
									is.Value = v
									is.OnChange(v)
									modelDirty = true
								}
							case input.KeyEscape:
								oldUID := fm.FocusedUID()
								fm.Blur()
								dispatcher.QueueFocusChange(oldUID, 0, ui.FocusSourceProgram)
								modelDirty = true
							}
						}
					}
					return true

				case input.CharMsg:
					// Collect for widget-level dispatch.
					dispatcher.Collect(m)
					// Framework-internal character input for TextFields.
					if is := fm.Input; is != nil && m.Char >= 32 {
						v := is.Value + string(m.Char)
						is.Value = v
						is.OnChange(v)
						modelDirty = true
					}
					return true

				case input.TextInputMsg:
					// Collect for widget-level dispatch.
					dispatcher.Collect(m)
					// Framework-internal IME input for TextFields.
					if is := fm.Input; is != nil && m.Text != "" {
						v := is.Value + m.Text
						is.Value = v
						is.OnChange(v)
						modelDirty = true
					}
					return true

				case input.MouseMsg:
					dispatcher.Collect(m)
				case input.ScrollMsg:
					dispatcher.Collect(m)
				case input.TouchMsg:
					dispatcher.Collect(m)
				}
				newModel := update(currentModel, msg)
				if modelChanged(any(newModel), any(currentModel)) {
					modelDirty = true
				}
				currentModel = newModel
				return true
			})

			// 2. Compute clamped dt.
			now := time.Now()
			rawDt := now.Sub(lastFrame)
			dt := appLoop.ClampDt(rawDt)
			lastFrame = now

			// 2b. Animation pass — tick all WidgetStates that implement
			// Animator before reconcile (RFC-002 §1.3). If any animation
			// is still running, force a repaint.
			animDirty := reconciler.TickAnimators(dt)

			// 2c. DirtyTracker pass — check WidgetStates that explicitly
			// marked themselves dirty (RFC-001 §6.4), e.g. video surfaces
			// or external data feeds that change independently of the model.
			stateDirty := reconciler.CheckDirtyTrackers()

			// Deliver TickMsg directly — always call update, but only
			// force a rebuild if the model is modified.
			tickModel := update(currentModel, TickMsg{DeltaTime: dt})
			tickDirty := modelChanged(any(tickModel), any(currentModel))
			currentModel = tickModel
			modelDirty = modelDirty || tickDirty || animDirty || stateDirty

			// Re-run view and reconcile only when the model changed.
			if modelDirty {
				// Reset focus order for this frame (rebuilt during reconcile + layout).
				fm.ResetOrder()

				// Dispatch collected input events to per-UID buffers.
				dispatcher.Dispatch()

				newTree := view(currentModel)
				currentTree, _ = reconciler.Reconcile(newTree, activeTheme, Send, dispatcher, fm)

				// Sort tab order derived from layout tree (RFC-002 §2.3).
				fm.SortOrder()
			}

			// 3. Update hover target from previous frame's hitMap.
			hoveredIdx := hitMap.HitTestIndex(mouseX, mouseY)
			hoverState.SetHovered(hoveredIdx, activeTheme.Tokens().Motion.Quick)

			// 4. Tick hover animations (RFC §12.2: AnimationTick before paint).
			hoverState.Tick(dt)

			// 5. Build scene with hover state.
			w, h := plat.FramebufferSize()
			canvas := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))
			hitMap.Reset()
			scene := ui.BuildScene(currentTree, canvas, activeTheme, w, h, &hitMap, &hoverState, fm)

			renderer.BeginFrame()
			renderer.Draw(scene)
			renderer.EndFrame()
		},

		OnResize: func(width, height int) {
			renderer.Resize(width, height)
			Send(input.ResizeMsg{Width: width, Height: height})
		},

		OnMouseButton: func(x, y float32, button int, pressed bool) {
			// Send input message for all mouse button events.
			btn := input.MouseButtonLeft
			switch button {
			case 1:
				btn = input.MouseButtonRight
			case 2:
				btn = input.MouseButtonMiddle
			}
			action := input.MouseRelease
			if pressed {
				action = input.MousePress
			}
			Send(input.MouseMsg{X: x, Y: y, Button: btn, Action: action})

			// Left-click hit-test and drag tracking.
			if button == 0 {
				if pressed {
					// Blur focus first; if the click lands on a focusable element,
					// its hit target will re-focus it.
					oldUID := fm.FocusedUID()
					fm.Blur()
					if oldUID != 0 {
						dispatcher.QueueFocusChange(oldUID, 0, ui.FocusSourceClick)
					}

					if target := hitMap.HitTest(x, y); target != nil {
						if target.OnClickAt != nil {
							target.OnClickAt(x, y)
							if target.Draggable {
								dragTarget = target
							}
						} else if target.OnClick != nil {
							target.OnClick()
						}
					}
				} else {
					// Release ends any active drag.
					dragTarget = nil
				}
			}
		},

		OnMouseMove: func(x, y float32) {
			mouseX = x
			mouseY = y
			Send(input.MouseMsg{X: x, Y: y, Action: input.MouseMove})
			// Continue firing positional callback while dragging.
			if dragTarget != nil && dragTarget.OnClickAt != nil {
				dragTarget.OnClickAt(x, y)
			}
		},

		OnScroll: func(deltaX, deltaY float32) {
			Send(input.ScrollMsg{X: mouseX, Y: mouseY, DeltaX: deltaX, DeltaY: deltaY})
			// Route scroll events directly to the ScrollView under the cursor.
			if target := hitMap.HitTestScroll(mouseX, mouseY); target != nil {
				target.OnScroll(deltaY * 30) // 30dp per scroll unit
			}
		},

		OnKey: func(key string, action int, mods int) {
			a := input.KeyPress
			switch action {
			case 1:
				a = input.KeyRelease
			case 2:
				a = input.KeyRepeat
			}
			Send(input.KeyMsg{
				Key:       input.KeyNameToKey[key],
				Modifiers: input.ModsFromBits(mods),
				Action:    a,
			})
		},

		OnChar: func(ch rune) {
			Send(input.CharMsg{Char: ch})
		},
	})
}

// modelChanged reports whether two model values differ.
// It uses == for comparable types and conservatively assumes changed
// for non-comparable types (slices, maps, funcs) to avoid panics.
func modelChanged(a, b any) bool {
	defer func() { recover() }()
	return a != b
}

// Ensure ViewFunc constraint is satisfied at compile time.
var _ ViewFunc[struct{}] = func(_ struct{}) ui.Element { return ui.Empty() }
