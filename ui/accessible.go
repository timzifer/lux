package ui

import "github.com/timzifer/lux/a11y"

// AccessibleWidget is an optional interface for widgets that provide
// accessibility information. Widgets that do not implement this
// interface receive a generic RoleGroup node in the access tree.
type AccessibleWidget interface {
	Widget
	// Accessibility returns the AccessNode for this widget given its
	// current state. Called during the access tree build pass.
	Accessibility(state WidgetState) a11y.AccessNode
}
