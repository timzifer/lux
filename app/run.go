package app

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	luximage "github.com/timzifer/lux/image"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/loop"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/internal/vellum"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// msdfReadyMsg is an internal message sent when background MSDF rasterization
// completes. It triggers a repaint so the atlas picks up the new glyphs.
type msdfReadyMsg struct{}

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

	// Wake the platform event loop when a message is enqueued from a
	// background goroutine so that idle-blocking platforms (Win32
	// WaitMessage, GLFW WaitEvents) process the message promptly.
	appLoop.SetWakeFunc(func() { plat.RequestFrame() })
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
	// DRM platforms provide a file descriptor and connector ID for VK_KHR_display.
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

	// Initialize the font rendering pipeline (RFC-003 §3).
	atlas := text.NewGlyphAtlas(512, 512)
	shaper := text.NewGoTextShaper(fonts.Fallback)
	shaper.RegisterFamily(fonts.PhosphorFamily)
	shaper.RegisterFamily(fonts.NotoEmojiFamily)

	// Wire async MSDF completion → repaint via the app loop.
	atlas.SetMSDFNotify(func() { appLoop.Send(msdfReadyMsg{}) })

	// If the renderer supports atlas-based text, wire it up.
	type atlasSetter interface{ SetAtlas(*text.GlyphAtlas) }
	if as, ok := renderer.(atlasSetter); ok {
		as.SetAtlas(atlas)
	}

	// Inspector: start Vellum server if configured (RFC-012 §5.1).
	var inspectorServer *vellum.Server
	var inspectorCollector *vellum.DebugExtensionCollector
	if cfg.inspectorAddr != "" {
		var err error
		inspectorServer, err = vellum.NewServer(cfg.inspectorAddr)
		if err != nil {
			return fmt.Errorf("vellum inspector: %w", err)
		}
		defer inspectorServer.Close()
		inspectorCollector = vellum.NewDebugExtensionCollector()
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
	currentLocale := cfg.locale // BCP 47 tag, propagated to RenderCtx (RFC-003 §3.8)
	activeProfile := cfg.profile // interaction profile, propagated to RenderCtx (RFC-004 §2.4)
	if activeProfile != nil {
		dispatcher.SetGestureConfig(ui.GestureConfigFromProfile(activeProfile))
	}

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
				case SetInteractionProfileMsg:
					p := m.Profile
					activeProfile = &p
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

	currentTree, _ := reconciler.Reconcile(view(currentModel), activeTheme, Send, nil, nil, currentLocale, activeProfile)

	lastFrame := time.Now()
	var hitMap hit.Map
	var hoverState ui.HoverState
	var mouseX, mouseY float32
	var frameCounter uint64
	var dragCallback func(x, y float32) // active drag callback (non-nil while dragging)
	var dragRelease func(x, y float32)  // called once when drag ends
	var currentCursor input.CursorKind
	var dynamicHandlers []globalHandlerEntry

	// On-Screen Keyboard state (RFC-004 §5).
	var oskState osk.OSKState
	osk.SetSendFunc(Send)
	defer osk.SetSendFunc(nil)
	// Track previous focus input to detect focus changes for auto-show/dismiss.
	var prevHadInput bool

	// State persistence: save persisted model on shutdown (RFC §3.4).
	if cfg.persistence != nil {
		defer func() {
			_ = savePersistedModel(cfg.persistence, any(currentModel), persistPath)
		}()
	}

	// needsInitialPaint ensures the first frame always paints the initial tree.
	needsInitialPaint := true

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
					currentLocale = m.Locale
					applyLocale(m.Locale)
					modelDirty = true // triggers full layout invalidation

				case SetInteractionProfileMsg:
					p := m.Profile
					activeProfile = &p
					dispatcher.SetGestureConfig(ui.GestureConfigFromProfile(activeProfile))
					modelDirty = true // triggers full layout invalidation

				case SetSizeMsg:
					plat.SetSize(m.Width, m.Height)
				case SetFullscreenMsg:
					plat.SetFullscreen(m.Fullscreen)

				case OpenWindowMsg:
					handleOpenWindow(m, plat, renderer, fm)
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
							shift := m.Modifiers.Has(input.ModShift)
							ctrl := m.Modifiers.Has(input.ModCtrl) || m.Modifiers.Has(input.ModSuper)

							// Platform shortcuts: Ctrl+C/V/X/A.
							if ctrl {
								switch m.Key {
								case input.KeyC:
									// Copy selected text.
									if is.HasSelection() {
										_ = SetClipboard(is.SelectedText())
									}
									modelDirty = true

								case input.KeyX:
									// Cut selected text.
									if is.HasSelection() {
										_ = SetClipboard(is.SelectedText())
										is.DeleteSelection()
										is.OnChange(is.Value)
										modelDirty = true
									}

								case input.KeyV:
									// Paste from clipboard.
									if clip, err := GetClipboard(); err == nil && clip != "" {
										is.DeleteSelection() // replace selection if any
										v := is.Value[:is.CursorOffset] + clip + is.Value[is.CursorOffset:]
										is.CursorOffset += len(clip)
										is.Value = v
										is.ClearSelection()
										is.OnChange(v)
										modelDirty = true
									}

								case input.KeyA:
									// Select all.
									is.SelectionStart = 0
									is.CursorOffset = len(is.Value)
									modelDirty = true
								}
							}

							// Navigation and editing keys.
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
									// Collapse selection to left edge.
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
									// Collapse selection to right edge.
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
					if is := fm.Input; is != nil {
						// Skip CR and LF -- Enter is already handled by KeyMsg(KeyEnter).
						// On Windows/GLFW both a KeyMsg and a CharMsg fire for Enter,
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
					// Collect for widget-level dispatch.
					dispatcher.Collect(m)
					// Framework-internal IME input for TextFields.
					if is := fm.Input; is != nil && m.Text != "" {
						is.DeleteSelection()
						v := is.Value[:is.CursorOffset] + m.Text + is.Value[is.CursorOffset:]
						is.CursorOffset += len(m.Text)
						is.Value = v
						is.ClearSelection()
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
						is.DeleteSelection()
						v := is.Value[:is.CursorOffset] + m.Text + is.Value[is.CursorOffset:]
						is.CursorOffset += len(m.Text)
						is.Value = v
						is.ClearSelection()
						is.OnChange(v)
						modelDirty = true
					}
					return true

				case msdfReadyMsg:
					modelDirty = true
					return true

				// OSK framework messages (RFC-004 §5).
				case ShowOSKMsg:
					oskState.Visible = true
					oskState.Layout = osk.OSKLayout(m.Layout)
					oskState.Mode = osk.ModeForLayout(osk.OSKLayout(m.Layout))
					modelDirty = true
					return true
				case DismissOSKMsg:
					oskState.Visible = false
					modelDirty = true
					return true
				case SetOSKModeMsg:
					oskState.Mode = osk.OSKMode(m.Mode)
					modelDirty = true
					return true
				case osk.OSKToggleShiftMsg:
					oskState.Shifted = !oskState.Shifted
					modelDirty = true
					return true
				case osk.OSKSwitchLayerMsg:
					// Toggle between alpha and numpad.
					if oskState.Mode == osk.ModeAlpha || oskState.Mode == osk.ModeCondensed {
						oskState.Mode = osk.ModeNumPad
					} else {
						oskState.Mode = osk.ModeAlpha
					}
					oskState.Shifted = false
					modelDirty = true
					return true
				case osk.OSKDismissMsg:
					oskState.Visible = false
					modelDirty = true
					return true
				case osk.OSKSignMsg:
					// Toggle sign on numeric input: inject +/- at start of value.
					if is := fm.Input; is != nil {
						if len(is.Value) > 0 && is.Value[0] == '-' {
							is.Value = is.Value[1:]
							if is.CursorOffset > 0 {
								is.CursorOffset--
							}
						} else {
							is.Value = "-" + is.Value
							is.CursorOffset++
						}
						is.ClearSelection()
						is.OnChange(is.Value)
						modelDirty = true
					}
					return true
				case input.ResizeMsg:
					// Window resize must force a full layout + repaint even if
					// the user's update function doesn't handle the message.
					modelDirty = true

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
			focusDirty := fm.ConsumeDirty()

			// OSK auto-show/dismiss (RFC-004 §5.1): when a text field gains
			// focus and there is no physical keyboard, show the OSK.
			hasInput := fm.Input != nil
			if hasInput != prevHadInput {
				prevHadInput = hasInput
				if activeProfile != nil && !activeProfile.HasPhysicalKeyboard {
					if hasInput && !oskState.Visible {
						oskState.Visible = true
						oskState.Mode = osk.ModeForLayout(oskState.Layout)
						oskState.Shifted = false
						modelDirty = true
					} else if !hasInput && oskState.Visible {
						oskState.Visible = false
						modelDirty = true
					}
				}
			}

			modelDirty = modelDirty || tickDirty || animDirty || stateDirty || focusDirty

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
				currentTree, _ = reconciler.Reconcile(newTree, activeTheme, Send, dispatcher, fm, currentLocale, activeProfile)

				// Sort tab order derived from layout tree (RFC-002 §2.3).
				fm.SortOrder()

				// A11y: build access tree and notify bridge (RFC-001 §11).
				if a11yBridge != nil {
					w, h := plat.WindowSize()
					accessTree := ui.BuildAccessTree(currentTree, reconciler, a11y.Rect{
						Width: float64(w), Height: float64(h),
					}, dispatcher)
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
			// When the interaction profile disables hover (touch/HMI), always
			// report no hover so widgets never show hover feedback (RFC-004 §2.6).
			hoveredIdx := hitMap.HitTestIndex(mouseX, mouseY)
			if activeProfile != nil && !activeProfile.HasHover {
				hoveredIdx = -1
			}
			hoverState.SetHovered(hoveredIdx, activeTheme.Tokens().Motion.Quick.Duration)

			// 4. Tick hover animations (RFC §12.2: AnimationTick before paint).
			hoverDirty := hoverState.Tick(dt)

			// 5. Skip scene build + GPU draw when nothing changed.
			// This is the main idle-CPU optimisation: BuildScene walks the
			// entire widget tree and renderer.Draw submits GPU commands —
			// both are expensive relative to the cheap message-drain above.
			var widgetNeedsFrame bool
			needsPaint := modelDirty || hoverDirty || needsInitialPaint
			if needsPaint {
				needsInitialPaint = false

				// Drain completed async MSDF glyphs into the atlas.
				atlas.DrainMSDFResults()

				// Reset element UID counter so BuildScene assigns the same UIDs
				// as the previous frame — hit-target callbacks capture these UIDs,
				// and they must match across frames for focus to persist.
				fm.ResetElementIDs()

				w, h := plat.FramebufferSize()
				sceneCanvas := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))

				// Inspector: wrap canvas with CanvasEncoder if active (RFC-012 §5.3).
				var encoder *vellum.CanvasEncoder
				var frameBuf *vellum.FrameBuffer
				var canvas draw.Canvas = sceneCanvas
				if inspectorServer != nil && inspectorServer.HasClient() {
					frameBuf = vellum.NewFrameBuffer()
					encoder = vellum.NewCanvasEncoder(sceneCanvas, frameBuf)
					encoder.BeginFrame(frameCounter, draw.R(0, 0, float32(w), float32(h)), sceneCanvas.DPR())
					canvas = encoder
				}
				frameCounter++

				hitMap.Reset()
				ix := ui.NewInteractor(&hitMap, &hoverState)
				ix.NeedsFrame = &widgetNeedsFrame

				// If the OSK is visible, reduce the content area by the OSK height (RFC-004 §5.2).
				contentH := h
				oskH := oskState.Height(w, h, canvas.DPR())
				if oskState.Visible && oskH > 0 {
					contentH = h - int(oskH)
					if contentH < 100 {
						contentH = 100
					}
				}

				paintStart := time.Now()

				// Build the OSK element (if visible) so BuildScene can render it (RFC-004 §5.5).
				var oskEl ui.Element
				if oskState.Visible {
					oskEl = osk.NewOSKElement(&oskState, w, h)
				}
				scene := ui.BuildSceneWithOSK(currentTree, canvas, activeTheme, w, contentH, ix, fm, oskEl, activeProfile)

				paintTime := time.Since(paintStart)

				// Inspector: finalize frame and send to client.
				if encoder != nil && inspectorServer != nil && inspectorCollector != nil {
					info := inspectorCollector.CollectFrameInfo(
						vellum.FrameTimings{PaintTime: paintTime},
						uint32(reconciler.StateCount()),
						reconciler.DirtyUIDs(),
					)
					encoder.EndFrame(info)
					inspectorServer.SendCanvas(frameBuf.Bytes())
				}

				// Sync dirty images from the image store to the renderer.
				syncImages(cfg.imageStore, renderer)

				renderer.BeginFrame()
				renderer.Draw(scene)
				renderer.EndFrame()
			}

			// Request continued rendering while animations, tick-driven
			// model changes, or dirty-tracked widgets are active, so
			// platforms that idle between frames keep ticking.
			if animDirty || hoverDirty || tickDirty || stateDirty || widgetNeedsFrame {
				plat.RequestFrame()
			}
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

// syncImages uploads dirty images from the store to the renderer.
func syncImages(store *luximage.Store, renderer gpu.Renderer) {
	if store == nil {
		return
	}
	uploader, ok := renderer.(gpu.ImageUploader)
	if !ok {
		return
	}
	for _, entry := range store.DirtyEntries() {
		uploader.UploadImage(entry.ID, entry.Width, entry.Height, entry.RGBA)
		entry.ClearDirty()
	}
}

// Ensure ViewFunc constraint is satisfied at compile time.
var _ ViewFunc[struct{}] = func(_ struct{}) ui.Element { return ui.Empty() }

