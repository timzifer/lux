//go:build windows

// lux-browser is a minimal WebView2-based browser built with the Lux UI framework.
//
// It demonstrates RFC-004 §7 (WebView2 Windows integration) with a toolbar
// (back, forward, reload, address bar), a WebView2 content area, and a status bar.
//
//	go build -o lux-browser.exe ./cmd/lux-browser/
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/surface/webview"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/form"
	"github.com/timzifer/lux/ui/layout"
)

const defaultURL = "https://example.com"

// Model holds the entire application state (Elm architecture).
type Model struct {
	wv          *webview.WebView
	addressBar  string
	title       string
	url         string
	loading     bool
	canBack     bool
	canForward  bool
	initialized bool
	renderer    *gpu.WGPURenderer // set by rendererFactory
}

// Messages
type (
	NavigateMsg       struct{}
	BackMsg           struct{}
	ForwardMsg        struct{}
	ReloadMsg         struct{}
	AddressChangedMsg struct{ Text string }
)

// nativeHandleProvider is satisfied by platform backends that expose a native window handle.
type nativeHandleProvider interface {
	NativeHandle() uintptr
}

func update(m Model, msg app.Msg) Model {
	switch msg := msg.(type) {
	case app.TickMsg:
		if !m.initialized {
			m.renderer = browserRenderer
			m = initWebView(m)
		}
		if m.wv != nil {
			m.title = m.wv.Title()
			m.loading = m.wv.IsLoading()
			m.canBack = m.wv.CanGoBack()
			m.canForward = m.wv.CanGoForward()
			newURL := m.wv.CurrentURL()
			if newURL != m.url {
				m.url = newURL
				m.addressBar = newURL
			}
		}

	case NavigateMsg:
		if m.wv != nil && m.addressBar != "" {
			url := ensureScheme(m.addressBar)
			m.addressBar = url
			m.wv.Navigate(url)
		}

	case BackMsg:
		if m.wv != nil {
			m.wv.Back()
		}

	case ForwardMsg:
		if m.wv != nil {
			m.wv.Forward()
		}

	case ReloadMsg:
		if m.wv != nil {
			m.wv.Reload()
		}

	case AddressChangedMsg:
		m.addressBar = msg.Text
	}
	return m
}

func initWebView(m Model) Model {
	p := app.ActivePlatform()
	if p == nil {
		return m
	}
	nhp, ok := p.(nativeHandleProvider)
	if !ok {
		m.initialized = true
		return m
	}
	hwnd := nhp.NativeHandle()
	if hwnd == 0 {
		return m
	}

	opts := []webview.Option{webview.WithParentWindow(hwnd)}
	if m.renderer != nil {
		opts = append(opts, webview.WithRenderer(m.renderer))
	}
	m.wv = webview.New(defaultURL, opts...)
	m.addressBar = defaultURL
	m.initialized = true
	return m
}

// browserRendererFactory stores the renderer in the model so WebView can
// use it for texture capture (same pattern as kitchen-sink/pyramid).
var browserRenderer *gpu.WGPURenderer

func browserRendererFactory() gpu.Renderer {
	r := gpu.NewWGPU()
	browserRenderer = r
	return r
}

func view(m Model) ui.Element {
	// Toolbar buttons
	backBtn := button.Text("\u2190", func() { app.Send(BackMsg{}) })
	forwardBtn := button.Text("\u2192", func() { app.Send(ForwardMsg{}) })
	reloadBtn := button.Text("\u21BB", func() { app.Send(ReloadMsg{}) })

	if !m.canBack {
		backBtn = button.TextDisabled("\u2190")
	}
	if !m.canForward {
		forwardBtn = button.TextDisabled("\u2192")
	}

	addressField := form.NewTextField(m.addressBar, "Enter URL...",
		form.WithOnChange(func(text string) {
			app.Send(AddressChangedMsg{Text: text})
		}),
		form.WithFocus(app.Focus()),
	)

	goBtn := button.Text("Go", func() { app.Send(NavigateMsg{}) })

	// Toolbar: Flex row so the address bar can Expand to fill remaining space.
	toolbar := layout.NewFlex([]ui.Element{
		backBtn,
		forwardBtn,
		reloadBtn,
		layout.Expand(addressField),
		goBtn,
	}, layout.WithDirection(layout.FlexRow), layout.WithGap(4))

	// Browser content area
	var content ui.Element
	if m.wv != nil {
		content = layout.Expand(ui.Surface(1, m.wv, 4096, 4096))
	} else {
		content = layout.Expand(display.Text("Initializing WebView2..."))
	}

	// Status bar
	statusText := m.url
	if m.loading {
		statusText = fmt.Sprintf("Loading: %s", m.url)
	}
	if m.title != "" {
		statusText = fmt.Sprintf("%s \u2014 %s", m.title, statusText)
	}
	statusBar := display.Text(statusText)

	// Main layout: Flex column so the content area Expands vertically.
	return layout.NewFlex([]ui.Element{
		toolbar,
		content,
		statusBar,
	}, layout.WithDirection(layout.FlexColumn))
}

func ensureScheme(url string) string {
	if !strings.Contains(url, "://") {
		return "https://" + url
	}
	return url
}

func main() {
	if err := app.Run(
		Model{addressBar: defaultURL},
		update,
		view,
		app.WithTitle("Lux Browser"),
		app.WithSize(1280, 900),
		app.WithTheme(theme.Default),
		app.WithRenderer(browserRendererFactory),
	); err != nil {
		log.Fatal(err)
	}
}
