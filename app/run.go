package app

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
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

	reconciler := ui.NewReconciler()
	currentModel := model
	currentTree, _ := reconciler.Reconcile(view(currentModel), activeTheme, Send)

	lastFrame := time.Now()
	var hitMap hit.Map
	var hoverState ui.HoverState
	var mouseX, mouseY float32

	return plat.Run(platform.Callbacks{
		OnFrame: func() {
			// 1. Drain messages — intercept theme switches before user update (RFC §5.5).
			modelDirty := false
			appLoop.DrainMessages(func(msg any) bool {
				switch m := msg.(type) {
				case SetThemeMsg:
					activeTheme = m.Theme
					updateBgColor()
				case SetDarkModeMsg:
					if m.Dark {
						activeTheme = theme.Slate
					} else {
						activeTheme = theme.SlateLight
					}
					updateBgColor()
				}
				currentModel = update(currentModel, msg)
				modelDirty = true
				return true
			})
			// Re-run view and reconcile only when the model changed.
			if modelDirty {
				newTree := view(currentModel)
				currentTree, _ = reconciler.Reconcile(newTree, activeTheme, Send)
			}

			// 2. Compute clamped dt.
			now := time.Now()
			rawDt := now.Sub(lastFrame)
			dt := appLoop.ClampDt(rawDt)
			lastFrame = now

			// 3. Update hover target from previous frame's hitMap.
			hoveredIdx := hitMap.HitTestIndex(mouseX, mouseY)
			hoverState.SetHovered(hoveredIdx, activeTheme.Tokens().Motion.Quick)

			// 4. Tick hover animations (RFC §12.2: AnimationTick before paint).
			hoverState.Tick(dt)

			// 5. Build scene with hover state.
			w, h := plat.FramebufferSize()
			canvas := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))
			hitMap.Reset()
			scene := ui.BuildScene(currentTree, canvas, activeTheme, w, h, &hitMap, &hoverState)

			renderer.BeginFrame()
			renderer.Draw(scene)
			renderer.EndFrame()
		},

		OnResize: func(width, height int) {
			renderer.Resize(width, height)
		},

		OnMouseButton: func(x, y float32, button int, pressed bool) {
			if button == 0 && pressed {
				if target := hitMap.HitTest(x, y); target != nil {
					if target.OnClickAt != nil {
						target.OnClickAt(x, y)
					} else if target.OnClick != nil {
						target.OnClick()
					}
				}
			}
		},

		OnMouseMove: func(x, y float32) {
			mouseX = x
			mouseY = y
		},
	})
}

// Ensure ViewFunc constraint is satisfied at compile time.
var _ ViewFunc[struct{}] = func(_ struct{}) ui.Element { return ui.Empty() }
