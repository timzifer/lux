package ui

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/theme"
)

// RenderToAccessTree is a convenience helper for accessibility unit tests.
// It takes an Element tree, reconciles it with the default light theme,
// and builds the corresponding AccessTree in a single call.
//
// This replaces the manual three-step pattern:
//
//	reconciler := NewReconciler()
//	resolved, _ := reconciler.Reconcile(tree, theme.LuxLight, ...)
//	accessTree := BuildAccessTree(resolved, reconciler, bounds)
//
// The returned tree uses default window bounds of 800×600.
func RenderToAccessTree(el Element) a11y.AccessTree {
	reconciler := NewReconciler()
	resolved, _ := reconciler.Reconcile(el, theme.LuxLight, func(any) {}, nil, nil, "", nil)
	return BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})
}
