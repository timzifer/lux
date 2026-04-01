package app

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/loop"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// RunMultiView starts a multi-window application.
// The multiView function returns a map of window IDs to their element trees.
// The main window (MainWindow = 0) must always be present in the returned map.
func RunMultiView[M any](model M, update UpdateFunc[M], multiView MultiViewFunc[M], opts ...Option) error {
	return runMultiViewInternal(model, func(m M, msg Msg) (M, Cmd) {
		return update(m, msg), nil
	}, multiView, opts...)
}

// RunMultiViewWithCmd starts a multi-window application with command support.
func RunMultiViewWithCmd[M any](model M, update UpdateWithCmd[M], multiView MultiViewFunc[M], opts ...Option) error {
	return runMultiViewInternal(model, update, multiView, opts...)
}

// runMultiViewInternal is the core multi-window run loop.
func runMultiViewInternal[M any](model M, update func(M, Msg) (M, Cmd), multiView MultiViewFunc[M], opts ...Option) error {
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

	// Wake the platform event loop when a message is enqueued from a
	// background goroutine (same rationale as single-view run.go).
	appLoop.SetWakeFunc(func() { plat.RequestFrame() })

	activePlatform = plat
	defer func() { activePlatform = nil }()

	if cfg.fullscreen {
		plat.SetFullscreen(true)
	}

	renderer := cfg.rendererFactory()
	fbW, fbH := plat.FramebufferSize()

	var nativeHandle uintptr
	if nh, ok := plat.(interface{ NativeHandle() uintptr }); ok {
		nativeHandle = nh.NativeHandle()
	}
	var nativeDisplay uintptr
	if nd, ok := plat.(interface{ NativeDisplay() uintptr }); ok {
		nativeDisplay = nd.NativeDisplay()
	}
	gpuCfg := gpu.Config{
		Width:         fbW,
		Height:        fbH,
		NativeHandle:  nativeHandle,
		NativeDisplay: nativeDisplay,
		DRMfd:         -1,
	}
	if dp, ok := plat.(interface {
		DRMfd() int
		DRMConnectorID() uint32
	}); ok {
		gpuCfg.DRMfd = dp.DRMfd()
		gpuCfg.DRMConnectorID = dp.DRMConnectorID()
	}

	if err := renderer.Init(gpuCfg); err != nil {
		return fmt.Errorf("gpu init: %w", err)
	}
	defer renderer.Destroy()

	atlas := text.NewGlyphAtlas(512, 512)
	shaper := text.NewGoTextShaper(fonts.Fallback)
	shaper.RegisterFamily(fonts.PhosphorFamily)

	atlas.SetMSDFNotify(func() { appLoop.Send(msdfReadyMsg{}) })

	type atlasSetter interface{ SetAtlas(*text.GlyphAtlas) }
	if as, ok := renderer.(atlasSetter); ok {
		as.SetAtlas(atlas)
	}

	if cfg.locale != "" {
		applyLocale(cfg.locale)
	}

	cachedTheme := theme.NewCachedTheme(cfg.theme)
	cachedTheme.WarmUp()
	var activeTheme theme.Theme = cachedTheme
	bgColor := activeTheme.Tokens().Colors.Surface.Base

	updateBgColor := func() {
		bgColor = activeTheme.Tokens().Colors.Surface.Base
		if bgs, ok := renderer.(interface{ SetBackgroundColor(draw.Color) }); ok {
			bgs.SetBackgroundColor(bgColor)
		}
	}
	updateBgColor()

	fm := globalFocus
	currentModel := model
	currentLocale := cfg.locale
	activeProfile := cfg.profile // interaction profile (RFC-004 §2.4)

	dispatchCmd := func(cmd Cmd) {
		if cmd != nil {
			go func() {
				if result := cmd(); result != nil {
					appLoop.Send(result)
				}
			}()
		}
	}

	// Per-window state. MainWindow always has a context.
	windows := make(map[WindowID]*windowContext)
	mainWC := &windowContext{
		id:         MainWindow,
		reconciler: ui.NewReconciler(),
		dispatcher: ui.NewEventDispatcher(fm),
		width:      fbW,
		height:     fbH,
	}
	windows[MainWindow] = mainWC
	if activeProfile != nil {
		mainWC.dispatcher.SetGestureConfig(ui.GestureConfigFromProfile(activeProfile))
	}

	// Initial reconcile for main window.
	views := multiView(currentModel)
	if mainElem, ok := views[MainWindow]; ok {
		mainWC.currentTree, _ = mainWC.reconciler.Reconcile(mainElem, activeTheme, Send, nil, nil, currentLocale, activeProfile)
	}

	lastFrame := time.Now()
	var dynamicHandlers []globalHandlerEntry

	// State persistence.
	var persistPath string
	if cfg.persistence != nil {
		persistPath = storagePath(cfg.title, cfg.persistence.key, cfg.storagePath)
		if restored, err := loadPersistedModel(cfg.persistence, persistPath); err == nil {
			currentModel = restored.(M)
			restoredModel, cmd := update(currentModel, ModelRestoredMsg{})
			currentModel = restoredModel
			dispatchCmd(cmd)

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
					newModel, cmd := update(currentModel, msg)
					currentModel = newModel
					dispatchCmd(cmd)
				}
				return true
			})
		}
	}
	if cfg.persistence != nil {
		defer func() {
			_ = savePersistedModel(cfg.persistence, any(currentModel), persistPath)
		}()
	}

	_ = bgColor // used via updateBgColor side-effect on renderer

	// needsInitialPaint ensures the first frame always paints the initial tree.
	needsInitialPaint := true

	return plat.Run(platform.Callbacks{
		OnFrame: func() {
			// 1. Drain messages.
			modelDirty := false
			mainWC.dispatcher.ResetEvents()
			for _, wc := range windows {
				if wc.id != MainWindow {
					wc.dispatcher.ResetEvents()
				}
			}

			appLoop.DrainMessages(func(msg any) bool {
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
					currentLocale = m.Locale
					applyLocale(m.Locale)
					modelDirty = true

				case SetInteractionProfileMsg:
					p := m.Profile
					activeProfile = &p
					mainWC.dispatcher.SetGestureConfig(ui.GestureConfigFromProfile(activeProfile))
					modelDirty = true

				case SetSizeMsg:
					plat.SetSize(m.Width, m.Height)
				case SetFullscreenMsg:
					plat.SetFullscreen(m.Fullscreen)

				case OpenWindowMsg:
					wc := handleOpenWindow(m, plat, renderer, fm)
					if wc != nil {
						windows[m.ID] = wc
					}
					return true
				case CloseWindowMsg:
					delete(windows, m.ID)
					handleCloseWindow(m, plat, renderer)
					return true

				case ui.RequestFocusMsg:
					oldUID := fm.FocusedUID()
					fm.SetFocusedUID(m.Target)
					mainWC.dispatcher.QueueFocusChange(oldUID, m.Target, ui.FocusSourceProgram)
					modelDirty = true
					return true

				case ui.ReleaseFocusMsg:
					oldUID := fm.FocusedUID()
					fm.Blur()
					mainWC.dispatcher.QueueFocusChange(oldUID, 0, ui.FocusSourceProgram)
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
					mainWC.dispatcher.Collect(m)
					if m.Action == input.KeyPress || m.Action == input.KeyRepeat {
						for _, sc := range cfg.shortcuts {
							if m.Key == sc.shortcut.Key && m.Modifiers == sc.shortcut.Modifiers {
								newModel, cmd := update(currentModel, input.ShortcutMsg{Shortcut: sc.shortcut, ID: sc.id})
								if modelChanged(any(newModel), any(currentModel)) {
									modelDirty = true
								}
								currentModel = newModel
								dispatchCmd(cmd)
								return true
							}
						}
					}
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
								mainWC.dispatcher.QueueFocusChange(oldUID, newUID, ui.FocusSourceTab)
								modelDirty = true
							}
							return true
						}
						if is := fm.Input; is != nil {
							shift := m.Modifiers.Has(input.ModShift)
							ctrl := m.Modifiers.Has(input.ModCtrl) || m.Modifiers.Has(input.ModSuper)
							// Platform shortcuts: Ctrl+C/V/X/A.
							if ctrl {
								switch m.Key {
								case input.KeyC:
									if is.HasSelection() {
										_ = SetClipboard(is.SelectedText())
									}
									modelDirty = true
								case input.KeyX:
									if is.HasSelection() {
										_ = SetClipboard(is.SelectedText())
										is.DeleteSelection()
										is.OnChange(is.Value)
										modelDirty = true
									}
								case input.KeyV:
									if clip, err := GetClipboard(); err == nil && clip != "" {
										is.DeleteSelection()
										v := is.Value[:is.CursorOffset] + clip + is.Value[is.CursorOffset:]
										is.CursorOffset += len(clip)
										is.Value = v
										is.ClearSelection()
										is.OnChange(v)
										modelDirty = true
									}
								case input.KeyA:
									is.SelectionStart = 0
									is.CursorOffset = len(is.Value)
									modelDirty = true
								}
							}

							switch m.Key {
							case input.KeyEnter:
								if is.Multiline {
									is.DeleteSelection()
									v := is.Value[:is.CursorOffset] + "\n" + is.Value[is.CursorOffset:]
									is.CursorOffset++
									is.Value = v
									is.ClearSelection()
									is.OnChange(v)
									modelDirty = true
								}
							case input.KeyBackspace:
								if is.HasSelection() {
									is.DeleteSelection()
									is.OnChange(is.Value)
									modelDirty = true
								} else if is.CursorOffset > 0 {
									v, newOff := text.DeleteBackward(is.Value, is.CursorOffset)
									is.Value = v
									is.CursorOffset = newOff
									is.OnChange(v)
									modelDirty = true
								}
							case input.KeyDelete:
								if is.HasSelection() {
									is.DeleteSelection()
									is.OnChange(is.Value)
									modelDirty = true
								} else if is.CursorOffset < len(is.Value) {
									v, newOff := text.DeleteForward(is.Value, is.CursorOffset)
									is.Value = v
									is.CursorOffset = newOff
									is.OnChange(v)
									modelDirty = true
								}
							case input.KeyLeft:
								if shift {
									if is.SelectionStart < 0 {
										is.SelectionStart = is.CursorOffset
									}
								} else if is.HasSelection() {
									a, _ := is.SelectionRange()
									is.CursorOffset = a
									is.ClearSelection()
									modelDirty = true
									break
								} else {
									is.ClearSelection()
								}
								if ctrl {
									is.CursorOffset = text.PrevWordBoundary(is.Value, is.CursorOffset)
								} else {
									is.CursorOffset = text.PrevGraphemeCluster(is.Value, is.CursorOffset)
								}
								modelDirty = true
							case input.KeyRight:
								if shift {
									if is.SelectionStart < 0 {
										is.SelectionStart = is.CursorOffset
									}
								} else if is.HasSelection() {
									_, b := is.SelectionRange()
									is.CursorOffset = b
									is.ClearSelection()
									modelDirty = true
									break
								} else {
									is.ClearSelection()
								}
								if ctrl {
									is.CursorOffset = text.NextWordBoundary(is.Value, is.CursorOffset)
								} else {
									is.CursorOffset = text.NextGraphemeCluster(is.Value, is.CursorOffset)
								}
								modelDirty = true
							case input.KeyUp:
								if is.Multiline {
									if shift {
										if is.SelectionStart < 0 {
											is.SelectionStart = is.CursorOffset
										}
									} else {
										is.ClearSelection()
									}
									is.CursorOffset = text.CursorUp(is.Value, is.CursorOffset)
									modelDirty = true
								}
							case input.KeyDown:
								if is.Multiline {
									if shift {
										if is.SelectionStart < 0 {
											is.SelectionStart = is.CursorOffset
										}
									} else {
										is.ClearSelection()
									}
									is.CursorOffset = text.CursorDown(is.Value, is.CursorOffset)
									modelDirty = true
								}
							case input.KeyHome:
								if shift {
									if is.SelectionStart < 0 {
										is.SelectionStart = is.CursorOffset
									}
								} else {
									is.ClearSelection()
								}
								if is.Multiline && !ctrl {
									is.CursorOffset = text.LineStart(is.Value, is.CursorOffset)
								} else {
									is.CursorOffset = 0
								}
								modelDirty = true
							case input.KeyEnd:
								if shift {
									if is.SelectionStart < 0 {
										is.SelectionStart = is.CursorOffset
									}
								} else {
									is.ClearSelection()
								}
								if is.Multiline && !ctrl {
									is.CursorOffset = text.LineEnd(is.Value, is.CursorOffset)
								} else {
									is.CursorOffset = len(is.Value)
								}
								modelDirty = true
							case input.KeyEscape:
								oldUID := fm.FocusedUID()
								fm.Blur()
								mainWC.dispatcher.QueueFocusChange(oldUID, 0, ui.FocusSourceProgram)
								modelDirty = true
							}
						}
					}
					return true

				case input.CharMsg:
					mainWC.dispatcher.Collect(m)
					if is := fm.Input; is != nil {
						// Skip CR and LF -- Enter is already handled by KeyMsg(KeyEnter).
						// On Windows both a KeyMsg and a CharMsg fire for Enter,
						// which would insert a double newline without this guard.
						if m.Char >= 32 {
							is.DeleteSelection()
							ch := string(m.Char)
							v := is.Value[:is.CursorOffset] + ch + is.Value[is.CursorOffset:]
							is.CursorOffset += len(ch)
							is.Value = v
							is.ClearSelection()
							is.OnChange(v)
							modelDirty = true
						}
					}
					return true

				case input.TextInputMsg:
					mainWC.dispatcher.Collect(m)
					if is := fm.Input; is != nil && m.Text != "" {
						v := is.Value[:is.CursorOffset] + m.Text + is.Value[is.CursorOffset:]
						is.CursorOffset += len(m.Text)
						is.Value = v
						is.OnChange(v)
						modelDirty = true
					}
					return true

				case input.IMEComposeMsg:
					mainWC.dispatcher.Collect(m)
					if is := fm.Input; is != nil {
						is.ComposeText = m.Text
						is.ComposeCursorStart = m.CursorStart
						is.ComposeCursorEnd = m.CursorEnd
						modelDirty = true
					}
					return true

				case input.IMECommitMsg:
					mainWC.dispatcher.Collect(m)
					if is := fm.Input; is != nil && m.Text != "" {
						is.ComposeText = ""
						v := is.Value[:is.CursorOffset] + m.Text + is.Value[is.CursorOffset:]
						is.CursorOffset += len(m.Text)
						is.Value = v
						is.OnChange(v)
						modelDirty = true
					}
					return true

				case msdfReadyMsg:
					modelDirty = true
					return true

				case input.MouseMsg:
					mainWC.dispatcher.Collect(m)
				case input.ScrollMsg:
					mainWC.dispatcher.Collect(m)
				case input.TouchMsg:
					mainWC.dispatcher.Collect(m)
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

			// 2b. Animation + dirty passes for all windows.
			animDirty := false
			stateDirty := false
			for _, wc := range windows {
				if wc.reconciler.TickAnimators(dt) {
					animDirty = true
				}
				if wc.reconciler.CheckDirtyTrackers() {
					stateDirty = true
				}
			}

			tickModel, tickCmd := update(currentModel, TickMsg{DeltaTime: dt})
			tickDirty := modelChanged(any(tickModel), any(currentModel))
			currentModel = tickModel
			dispatchCmd(tickCmd)
			modelDirty = modelDirty || tickDirty || animDirty || stateDirty

			// 3. Get views for all windows and reconcile/render.
			if modelDirty {
				fm.ResetOrder()

				for _, h := range cfg.globalHandlers {
					mainWC.dispatcher.FilterCollectedEvents(h.handler)
				}
				for _, h := range dynamicHandlers {
					mainWC.dispatcher.FilterCollectedEvents(h.handler)
				}
				mainWC.dispatcher.Dispatch()
			}

			views := multiView(currentModel)

			// Render main window.
			if mainElem, ok := views[MainWindow]; ok && modelDirty {
				mainWC.currentTree, _ = mainWC.reconciler.Reconcile(mainElem, activeTheme, Send, mainWC.dispatcher, fm, currentLocale, activeProfile)
				fm.SortOrder()
			}

			// Main window: hover tick.
			hoveredIdx := mainWC.hitMap.HitTestIndex(mainWC.mouseX, mainWC.mouseY)
			if activeProfile != nil && !activeProfile.HasHover {
				hoveredIdx = -1
			}
			mainWC.hoverState.SetHovered(hoveredIdx, activeTheme.Tokens().Motion.Quick.Duration)
			hoverDirty := mainWC.hoverState.Tick(dt)

			// Tick hover animations on secondary windows.
			for winID, wc := range windows {
				if winID == MainWindow {
					continue
				}
				hovIdx := wc.hitMap.HitTestIndex(wc.mouseX, wc.mouseY)
				if activeProfile != nil && !activeProfile.HasHover {
					hovIdx = -1
				}
				wc.hoverState.SetHovered(hovIdx, activeTheme.Tokens().Motion.Quick.Duration)
				if wc.hoverState.Tick(dt) {
					hoverDirty = true
				}
			}

			// Skip scene build + GPU draw when nothing changed.
			needsPaint := modelDirty || hoverDirty || needsInitialPaint
			if needsPaint {
				needsInitialPaint = false

				// Drain completed async MSDF glyphs into the atlas.
				atlas.DrainMSDFResults()

				w, h := plat.FramebufferSize()
				canvas := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))
				mainWC.hitMap.Reset()
				ix := ui.NewInteractor(&mainWC.hitMap, &mainWC.hoverState)
				scene := ui.BuildScene(mainWC.currentTree, canvas, activeTheme, w, h, ix, fm)

				syncImages(cfg.imageStore, renderer)

				renderer.BeginFrame()
				renderer.Draw(scene)
				renderer.EndFrame()

				// Render secondary windows.
				wr, hasWR := renderer.(gpu.WindowRenderer)
				for winID, wc := range windows {
					if winID == MainWindow {
						continue
					}
					elem, ok := views[winID]
					if !ok {
						continue
					}

					// Reconcile secondary window.
					if modelDirty {
						wc.dispatcher.Dispatch()
						wc.currentTree, _ = wc.reconciler.Reconcile(elem, activeTheme, Send, wc.dispatcher, nil, currentLocale, activeProfile)
					}

					winCanvas := render.NewSceneCanvas(wc.width, wc.height, render.WithShaper(shaper), render.WithAtlas(atlas))
					wc.hitMap.Reset()
					winIx := ui.NewInteractor(&wc.hitMap, &wc.hoverState)
					winScene := ui.BuildScene(wc.currentTree, winCanvas, activeTheme, wc.width, wc.height, winIx)

					if hasWR {
						wr.BeginFrameWindow(uint32(winID))
						wr.DrawWindow(uint32(winID), winScene)
						wr.EndFrameWindow(uint32(winID))
					}
				}
			}

			// Request continued rendering while animations are active.
			if animDirty || hoverDirty {
				plat.RequestFrame()
			}
		},

		OnResize: func(width, height int) {
			renderer.Resize(width, height)
			mainWC.width = width
			mainWC.height = height
			Send(input.ResizeMsg{Width: width, Height: height})
		},

		OnMouseButton: func(x, y float32, button int, pressed bool) {
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

			if button == 0 {
				if pressed {
					oldUID := fm.FocusedUID()
					fm.Blur()
					if oldUID != 0 {
						mainWC.dispatcher.QueueFocusChange(oldUID, 0, ui.FocusSourceClick)
					}
					if target := mainWC.hitMap.HitTest(x, y); target != nil {
						if target.OnClickAt != nil {
							target.OnClickAt(x, y)
							if target.Draggable {
								mainWC.dragCB = target.OnClickAt
								mainWC.dragRelease = target.OnRelease
							}
						} else if target.OnClick != nil {
							target.OnClick()
						}
					}
				} else {
					if mainWC.dragRelease != nil {
						mainWC.dragRelease(x, y)
					}
					mainWC.dragCB = nil
					mainWC.dragRelease = nil
				}
			}
		},

		OnMouseMove: func(x, y float32) {
			mainWC.mouseX = x
			mainWC.mouseY = y
			Send(input.MouseMsg{X: x, Y: y, Action: input.MouseMove})
			if mainWC.dragCB != nil {
				mainWC.dragCB(x, y)
			}
			newCursor := input.CursorDefault
			if target := mainWC.hitMap.HitTest(x, y); target != nil {
				newCursor = target.Cursor
			}
			if newCursor != mainWC.cursor {
				mainWC.cursor = newCursor
				plat.SetCursor(newCursor)
			}
		},

		OnScroll: func(deltaX, deltaY float32) {
			Send(input.ScrollMsg{X: mainWC.mouseX, Y: mainWC.mouseY, DeltaX: deltaX, DeltaY: deltaY})
			if target := mainWC.hitMap.HitTestScroll(mainWC.mouseX, mainWC.mouseY); target != nil {
				target.OnScroll(deltaY * 30)
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

		// ── Multi-window input callbacks ──────────────────────────

		OnWindowResize: func(windowID uint32, width, height int) {
			wc := windows[WindowID(windowID)]
			if wc == nil {
				return
			}
			wc.width = width
			wc.height = height
			if wr, ok := renderer.(gpu.WindowRenderer); ok {
				wr.ResizeWindow(windowID, width, height)
			}
		},

		OnWindowClose: func(windowID uint32) {
			Send(WindowClosedMsg{Window: WindowID(windowID)})
		},

		OnWindowMouseButton: func(windowID uint32, x, y float32, button int, pressed bool) {
			wc := windows[WindowID(windowID)]
			if wc == nil {
				return
			}
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

			if button == 0 {
				if pressed {
					if target := wc.hitMap.HitTest(x, y); target != nil {
						if target.OnClickAt != nil {
							target.OnClickAt(x, y)
							if target.Draggable {
								wc.dragCB = target.OnClickAt
								wc.dragRelease = target.OnRelease
							}
						} else if target.OnClick != nil {
							target.OnClick()
						}
					}
				} else {
					if wc.dragRelease != nil {
						wc.dragRelease(x, y)
					}
					wc.dragCB = nil
					wc.dragRelease = nil
				}
			}
		},

		OnWindowMouseMove: func(windowID uint32, x, y float32) {
			wc := windows[WindowID(windowID)]
			if wc == nil {
				return
			}
			wc.mouseX = x
			wc.mouseY = y
			if wc.dragCB != nil {
				wc.dragCB(x, y)
			}
		},

		OnWindowKey: func(windowID uint32, key string, action int, mods int) {
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

		OnWindowChar: func(windowID uint32, ch rune) {
			Send(input.CharMsg{Char: ch})
		},

		OnWindowScroll: func(windowID uint32, deltaX, deltaY float32) {
			wc := windows[WindowID(windowID)]
			if wc == nil {
				return
			}
			Send(input.ScrollMsg{X: wc.mouseX, Y: wc.mouseY, DeltaX: deltaX, DeltaY: deltaY})
			if target := wc.hitMap.HitTestScroll(wc.mouseX, wc.mouseY); target != nil {
				target.OnScroll(deltaY * 30)
			}
		},
	})
}

