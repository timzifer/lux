package app

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/anim"
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
	return runInternal(model, func(m M, msg Msg) (M, Cmd) {
		return update(m, msg), nil
	}, view, opts...)
}

// RunWithCmd starts the application with an update function that returns commands (RFC §3.6).
// Commands are side-effect functions dispatched asynchronously after each update.
func RunWithCmd[M any](model M, update UpdateWithCmd[M], view ViewFunc[M], opts ...Option) error {
	return runInternal(model, update, view, opts...)
}

// runInternal contains the full run-loop logic, parameterized over an update
// function that returns (M, Cmd).
func runInternal[M any](model M, update func(M, Msg) (M, Cmd), view ViewFunc[M], opts ...Option) error {
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

	// Wire anim.SendFunc so AnimationEnded msgs reach the app loop (RFC-002 §1.8).
	anim.SendFunc = func(msg any) { Send(msg) }
	defer func() { anim.SendFunc = nil }()

	plat := cfg.platformFactory()
	if err := plat.Init(platform.Config{
		Title:  cfg.title,
		Width:  cfg.width,
		Height: cfg.height,
	}); err != nil {
		return fmt.Errorf("platform init: %w", err)
	}
	defer plat.Destroy()

	// Make platform accessible for package-level clipboard functions (RFC §7.1).
	activePlatform = plat
	defer func() { activePlatform = nil }()

	// A11y: initialize bridge if the platform supports it (RFC-001 §11).
	type a11yBridgeProvider interface {
		A11yBridge() a11y.A11yBridge
	}
	type a11ySendSetter interface {
		SetA11ySend(func(any))
	}
	var a11yBridge a11y.A11yBridge
	if bp, ok := plat.(a11yBridgeProvider); ok {
		a11yBridge = bp.A11yBridge()
		if ss, ok2 := plat.(a11ySendSetter); ok2 {
			ss.SetA11ySend(Send)
		}
	}
	var prevA11yFocusedID a11y.AccessNodeID

	// Apply initial fullscreen setting (RFC §7.1).
	if cfg.fullscreen {
		plat.SetFullscreen(true)
	}

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
	shaper := text.NewGoTextShaper(fonts.Fallback)
	shaper.RegisterFamily(fonts.PhosphorFamily)

	// If the renderer supports atlas-based text, wire it up.
	type atlasSetter interface{ SetAtlas(*text.GlyphAtlas) }
	if as, ok := renderer.(atlasSetter); ok {
		as.SetAtlas(atlas)
	}

	// Apply initial locale → layout direction (RFC-003 §3.8).
	if cfg.locale != "" {
		applyLocale(cfg.locale)
	}

	cachedTheme := theme.NewCachedTheme(cfg.theme)
	cachedTheme.WarmUp()
	var activeTheme theme.Theme = cachedTheme
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

	// dispatchCmd runs a Cmd asynchronously, sending its result back into the loop.
	dispatchCmd := func(cmd Cmd) {
		if cmd != nil {
			go func() {
				if result := cmd(); result != nil {
					appLoop.Send(result)
				}
			}()
		}
	}

	// State persistence: load persisted model on startup (RFC §3.4).
	var persistPath string
	if cfg.persistence != nil {
		persistPath = storagePath(cfg.title, cfg.persistence.key, cfg.storagePath)
		if restored, err := loadPersistedModel(cfg.persistence, persistPath); err == nil {
			currentModel = restored.(M)

			// Let the user's update function react to the restored model
			// (e.g., send SetDarkModeMsg to apply a persisted theme preference).
			restoredModel, cmd := update(currentModel, ModelRestoredMsg{})
			currentModel = restoredModel
			dispatchCmd(cmd)

			// Drain any framework messages queued during restore (e.g., SetDarkModeMsg)
			// so the first reconcile uses the correct theme.
			appLoop.DrainMessages(func(msg any) bool {
				switch m := msg.(type) {
				case SetThemeMsg:
					cachedTheme = theme.NewCachedTheme(m.Theme)
					cachedTheme.WarmUp()
					activeTheme = cachedTheme
					updateBgColor()
				case SetDarkModeMsg:
					if m.Dark {
						cachedTheme = theme.NewCachedTheme(darkVariant(activeTheme))
					} else {
						cachedTheme = theme.NewCachedTheme(lightVariant(activeTheme))
					}
					cachedTheme.WarmUp()
					activeTheme = cachedTheme
					updateBgColor()
				default:
					// Non-theme messages: feed through update normally.
					newModel, cmd := update(currentModel, msg)
					currentModel = newModel
					dispatchCmd(cmd)
				}
				return true
			})
		}
	}

	currentTree, _ := reconciler.Reconcile(view(currentModel), activeTheme, Send, nil, nil)

	lastFrame := time.Now()
	var hitMap hit.Map
	var hoverState ui.HoverState
	var mouseX, mouseY float32
	var dragCallback func(x, y float32) // active drag callback (non-nil while dragging)
	var dragRelease func(x, y float32)  // called once when drag ends
	var currentCursor input.CursorKind
	var dynamicHandlers []globalHandlerEntry

	// State persistence: save persisted model on shutdown (RFC §3.4).
	if cfg.persistence != nil {
		defer func() {
			_ = savePersistedModel(cfg.persistence, any(currentModel), persistPath)
		}()
	}

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
					cachedTheme = theme.NewCachedTheme(m.Theme)
					cachedTheme.WarmUp()
					activeTheme = cachedTheme
					updateBgColor()
					modelDirty = true
				case SetDarkModeMsg:
					if m.Dark {
						cachedTheme = theme.NewCachedTheme(darkVariant(activeTheme))
					} else {
						cachedTheme = theme.NewCachedTheme(lightVariant(activeTheme))
					}
					cachedTheme.WarmUp()
					activeTheme = cachedTheme
					updateBgColor()
					modelDirty = true

				case SetLocaleMsg:
					applyLocale(m.Locale)
					modelDirty = true // triggers full layout invalidation

				case SetSizeMsg:
					plat.SetSize(m.Width, m.Height)
				case SetFullscreenMsg:
					plat.SetFullscreen(m.Fullscreen)

				case OpenWindowMsg:
					handleOpenWindow(m, plat, renderer)
					return true
				case CloseWindowMsg:
					handleCloseWindow(m, plat, renderer)
					return true

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

				case RegisterHandlerMsg:
					dynamicHandlers = append(dynamicHandlers, globalHandlerEntry{id: m.ID, handler: m.Handler})
					return true
				case UnregisterHandlerMsg:
					for i, h := range dynamicHandlers {
						if h.id == m.ID {
							dynamicHandlers = append(dynamicHandlers[:i], dynamicHandlers[i+1:]...)
							break
						}
					}
					return true

				case input.KeyMsg:
					// Collect for widget-level dispatch.
					dispatcher.Collect(m)

					// Check registered shortcuts before other handling (RFC-002 §2.5).
					if m.Action == input.KeyPress || m.Action == input.KeyRepeat {
						for _, sc := range cfg.shortcuts {
							if m.Key == sc.shortcut.Key && m.Modifiers == sc.shortcut.Modifiers {
								newModel, cmd := update(currentModel, input.ShortcutMsg{Shortcut: sc.shortcut, ID: sc.id})
								if modelChanged(any(newModel), any(currentModel)) {
									modelDirty = true
								}
								currentModel = newModel
								dispatchCmd(cmd)
								return true // consumed
							}
						}
					}

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

				case input.IMEComposeMsg:
					dispatcher.Collect(m)
					// Update FocusManager's compose state for TextField rendering.
					if is := fm.Input; is != nil {
						is.ComposeText = m.Text
						is.ComposeCursorStart = m.CursorStart
						is.ComposeCursorEnd = m.CursorEnd
						modelDirty = true
					}
					return true

				case input.IMECommitMsg:
					dispatcher.Collect(m)
					// Insert committed text into focused TextField.
					if is := fm.Input; is != nil && m.Text != "" {
						is.ComposeText = "" // clear composition
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
				newModel, cmd := update(currentModel, msg)
				if modelChanged(any(newModel), any(currentModel)) {
					modelDirty = true
				}
				currentModel = newModel
				dispatchCmd(cmd)
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
			tickModel, tickCmd := update(currentModel, TickMsg{DeltaTime: dt})
			tickDirty := modelChanged(any(tickModel), any(currentModel))
			currentModel = tickModel
			dispatchCmd(tickCmd)
			modelDirty = modelDirty || tickDirty || animDirty || stateDirty

			// Re-run view and reconcile only when the model changed.
			if modelDirty {
				// Reset focus order for this frame (rebuilt during reconcile + layout).
				fm.ResetOrder()

				// Global Handler Layer: filter events before widget dispatch (RFC-002 §2.8).
				for _, h := range cfg.globalHandlers {
					dispatcher.FilterCollectedEvents(h.handler)
				}
				for _, h := range dynamicHandlers {
					dispatcher.FilterCollectedEvents(h.handler)
				}

				// Dispatch collected input events to per-UID buffers.
				dispatcher.Dispatch()

				newTree := view(currentModel)
				currentTree, _ = reconciler.Reconcile(newTree, activeTheme, Send, dispatcher, fm)

				// Sort tab order derived from layout tree (RFC-002 §2.3).
				fm.SortOrder()

				// A11y: build access tree and notify bridge (RFC-001 §11).
				if a11yBridge != nil {
					w, h := plat.WindowSize()
					accessTree := ui.BuildAccessTree(currentTree, reconciler, a11y.Rect{
						Width: float64(w), Height: float64(h),
					})
					a11yBridge.UpdateTree(accessTree)

					// Focus tracking.
					if accessTree.FocusedID != prevA11yFocusedID {
						if accessTree.FocusedID != 0 {
							a11yBridge.NotifyFocus(accessTree.FocusedID)
						}
						prevA11yFocusedID = accessTree.FocusedID
					}
				}
			}

			// 3. Update hover target from previous frame's hitMap.
			hoveredIdx := hitMap.HitTestIndex(mouseX, mouseY)
			hoverState.SetHovered(hoveredIdx, activeTheme.Tokens().Motion.Quick.Duration)

			// 4. Tick hover animations (RFC §12.2: AnimationTick before paint).
			hoverState.Tick(dt)

			// 5. Build scene with hover state.
			w, h := plat.FramebufferSize()
			canvas := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))
			hitMap.Reset()
			ix := ui.NewInteractor(&hitMap, &hoverState)
			scene := ui.BuildScene(currentTree, canvas, activeTheme, w, h, ix, fm)

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
								dragCallback = target.OnClickAt
								dragRelease = target.OnRelease
							}
						} else if target.OnClick != nil {
							target.OnClick()
						}
					}
				} else {
					// Release ends any active drag.
					if dragRelease != nil {
						dragRelease(x, y)
					}
					dragCallback = nil
					dragRelease = nil
				}
			}
		},

		OnMouseMove: func(x, y float32) {
			mouseX = x
			mouseY = y
			Send(input.MouseMsg{X: x, Y: y, Action: input.MouseMove})
			// Continue firing positional callback while dragging.
			if dragCallback != nil {
				dragCallback(x, y)
			}
			// Update cursor based on hovered hit target (RFC-002 §2.7).
			newCursor := input.CursorDefault
			if target := hitMap.HitTest(x, y); target != nil {
				newCursor = target.Cursor
			}
			if newCursor != currentCursor {
				currentCursor = newCursor
				plat.SetCursor(newCursor)
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

		OnIMECompose: func(text string, cursorStart, cursorEnd int) {
			Send(input.IMEComposeMsg{Text: text, CursorStart: cursorStart, CursorEnd: cursorEnd})
		},

		OnIMECommit: func(text string) {
			Send(input.IMECommitMsg{Text: text})
		},

		// Multi-window: when the user closes a secondary window via the X button,
		// send a WindowClosedMsg so the model can react.
		OnWindowClose: func(windowID uint32) {
			Send(WindowClosedMsg{Window: WindowID(windowID)})
		},
	})
}

// modelChanged reports whether two model values differ.
// It uses == for comparable types and conservatively assumes changed
// for non-comparable types (slices, maps, funcs) to avoid panics.
func modelChanged(a, b any) (changed bool) {
	changed = true // default: assume changed for non-comparable types
	defer func() { recover() }()
	return a != b
}

// darkVariant returns the dark theme associated with the current active theme.
// If the theme (or its underlying base) implements ThemePair, its DarkVariant
// is returned. Otherwise we fall back to the default dark theme (LuxDark).
func darkVariant(active theme.Theme) theme.Theme {
	base := unwrapBase(active)
	if tp, ok := base.(theme.ThemePair); ok {
		return tp.DarkVariant()
	}
	return theme.LuxDark
}

// lightVariant is the light-mode counterpart of darkVariant.
func lightVariant(active theme.Theme) theme.Theme {
	base := unwrapBase(active)
	if tp, ok := base.(theme.ThemePair); ok {
		return tp.LightVariant()
	}
	return theme.LuxLight
}

// unwrapBase extracts the underlying theme from a CachedTheme wrapper.
func unwrapBase(t theme.Theme) theme.Theme {
	if ct, ok := t.(*theme.CachedTheme); ok {
		return ct.Base()
	}
	return t
}

// Ensure ViewFunc constraint is satisfied at compile time.
var _ ViewFunc[struct{}] = func(_ struct{}) ui.Element { return ui.Empty() }
